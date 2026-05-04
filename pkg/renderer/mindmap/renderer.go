package mindmap

import (
	"encoding/xml"
	"fmt"
	"math"
	"strings"

	"github.com/julianshen/mmgo/pkg/diagram"
	"github.com/julianshen/mmgo/pkg/renderer/svgutil"
	"github.com/julianshen/mmgo/pkg/textmeasure"
)

const (
	defaultFontSize = 14.0
	defaultPadding  = 20.0
	nodePadX        = 20.0
	nodePadY        = 10.0
	minNodeW        = 80.0
	minNodeH        = 35.0
	levelSpacing    = 120.0
	boldWidthFactor = 1.1
	iconRowH        = 14.0
)

type Options struct {
	FontSize float64
	Theme    Theme
	Ruler    *textmeasure.Ruler
}

type layoutNode struct {
	node       *diagram.MindmapNode
	x, y       float64
	width      float64
	height     float64
	depth      int
	section    int
	leafCount  int
	// segments holds every line's parsed markdown segments in
	// document order. lineHeights holds the per-line vertical
	// extent so the renderer can stack multi-line labels without
	// re-measuring. For single-line labels both have length 1
	// and the existing single-tspan fast path still applies.
	segments    [][]textSegment
	lineHeights []float64
	children    []*layoutNode
}

type svgText struct {
	XMLName  xml.Name `xml:"text"`
	X        svgFloat `xml:"x,attr"`
	Y        svgFloat `xml:"y,attr"`
	Anchor   string   `xml:"text-anchor,attr,omitempty"`
	Dominant string   `xml:"dominant-baseline,attr,omitempty"`
	Style    string   `xml:"style,attr,omitempty"`
	Children []any    `xml:",any"`
}

type svgTspan struct {
	XMLName xml.Name `xml:"tspan"`
	Style   string   `xml:"style,attr,omitempty"`
	Content string   `xml:",chardata"`
}

func Render(d *diagram.MindmapDiagram, opts *Options) ([]byte, error) {
	if d == nil {
		return nil, fmt.Errorf("mindmap render: diagram is nil")
	}

	fontSize := defaultFontSize
	if opts != nil && opts.FontSize > 0 {
		fontSize = opts.FontSize
	}
	th := resolveTheme(opts)

	var ruler *textmeasure.Ruler
	if opts != nil && opts.Ruler != nil {
		ruler = opts.Ruler
	} else {
		var err error
		ruler, err = textmeasure.NewDefaultRuler()
		if err != nil {
			return nil, fmt.Errorf("mindmap render: text measurer: %w", err)
		}
		defer func() { _ = ruler.Close() }()
	}

	if d.Root == nil {
		return marshalDoc(svgutil.ViewBox(100, 100), th,
			&rect{X: 0, Y: 0, Width: 100, Height: 100, Style: fmt.Sprintf("fill:%s;stroke:none", th.Background)},
		)
	}

	root := buildTree(d.Root, ruler, fontSize, 0, make(map[*diagram.MindmapNode]bool))

	layoutRadial(root, levelSpacing)
	bounds := computeBounds(root)
	pad := defaultPadding

	viewW := bounds.maxX - bounds.minX + 2*pad
	viewH := bounds.maxY - bounds.minY + 2*pad
	offX := -bounds.minX + pad
	offY := -bounds.minY + pad

	var children []any
	if d.AccTitle != "" {
		children = append(children, &svgutil.Title{Content: d.AccTitle})
	}
	if d.AccDescr != "" {
		children = append(children, &svgutil.Desc{Content: d.AccDescr})
	}
	children = append(children, &rect{
		X: 0, Y: 0, Width: svgutil.Float(viewW), Height: svgutil.Float(viewH),
		Style: fmt.Sprintf("fill:%s;stroke:none", th.Background),
	})

	var edgeElems, nodeElems []any
	renderElements(root, offX, offY, fontSize, th, &edgeElems, &nodeElems)

	if len(edgeElems) > 0 {
		children = append(children, &group{Children: edgeElems})
	}
	if len(nodeElems) > 0 {
		children = append(children, &group{Children: nodeElems})
	}

	return marshalDoc(svgutil.ViewBox(viewW, viewH), th, children...)
}

func marshalDoc(viewBox string, th Theme, children ...any) ([]byte, error) {
	doc := svgDoc{
		XMLNS:               "http://www.w3.org/2000/svg",
		ViewBox:             viewBox,
		Role:                "graphics-document document",
		AriaRoleDescription: "mindmap",
		Children:            children,
	}
	return svgutil.MarshalSVG(svgutil.Doc(doc))
}

func buildTree(n *diagram.MindmapNode, ruler *textmeasure.Ruler, fontSize float64, depth int, visited map[*diagram.MindmapNode]bool) *layoutNode {
	if n == nil {
		return nil
	}
	if visited[n] {
		return nil
	}
	visited[n] = true
	// Split on real newlines (the parser converted `\n` → '\n' for
	// us) and parse each line's markdown independently so a label
	// like "**Bold** title\nSubtitle" gets two stacked text rows.
	lines := strings.Split(n.Text, "\n")
	segments := make([][]textSegment, len(lines))
	lineHeights := make([]float64, len(lines))
	maxLineW := 0.0
	totalH := 0.0
	for i, line := range lines {
		segs := parseMarkdown(line)
		segments[i] = segs
		plain := stripMarkdownFromSegments(segs)
		tw, lh := ruler.Measure(plain, fontSize)
		hasBold := false
		for _, seg := range segs {
			if seg.bold {
				hasBold = true
				break
			}
		}
		if hasBold {
			tw = tw * boldWidthFactor
		}
		if tw > maxLineW {
			maxLineW = tw
		}
		lineHeights[i] = lh
		totalH += lh
	}

	w := maxLineW + 2*nodePadX
	h := totalH + 2*nodePadY
	if w < minNodeW {
		w = minNodeW
	}
	if h < minNodeH {
		h = minNodeH
	}
	// Reserve a small slot at the top of the node for the icon
	// caption when one is present. Phase B renders the icon as a
	// muted text glyph; Phase C may promote to a real font glyph.
	if n.Icon != "" {
		h += iconRowH
	}

	ln := &layoutNode{
		node:        n,
		width:       w,
		height:      h,
		depth:       depth,
		segments:    segments,
		lineHeights: lineHeights,
	}
	leafCount := 0
	for _, child := range n.Children {
		cn := buildTree(child, ruler, fontSize, depth+1, visited)
		if cn != nil {
			ln.children = append(ln.children, cn)
			leafCount += cn.leafCount
		}
	}
	if leafCount == 0 {
		leafCount = 1
	}
	ln.leafCount = leafCount
	return ln
}

func stripMarkdownFromSegments(segs []textSegment) string {
	var sb strings.Builder
	for _, seg := range segs {
		sb.WriteString(seg.text)
	}
	return sb.String()
}

func layoutRadial(root *layoutNode, spacing float64) {
	root.x = 0
	root.y = 0
	if len(root.children) == 0 {
		return
	}

	totalLeaves := root.leafCount

	startAngle := -math.Pi / 2
	for i, child := range root.children {
		child.section = i
		assignSectionRecursive(child, i)
		angleSpan := 2 * math.Pi * float64(child.leafCount) / float64(totalLeaves)
		mid := startAngle + angleSpan/2
		child.x = root.x + spacing*math.Cos(mid)
		child.y = root.y + spacing*math.Sin(mid)
		layoutRadialSubtree(child, startAngle, angleSpan, spacing)
		startAngle += angleSpan
	}
}

func assignSectionRecursive(n *layoutNode, section int) {
	n.section = section
	for _, c := range n.children {
		assignSectionRecursive(c, section)
	}
}

func layoutRadialSubtree(n *layoutNode, startAngle, angleSpan, spacing float64) {
	if len(n.children) == 0 {
		return
	}

	totalLeaves := n.leafCount

	childStart := startAngle
	for _, child := range n.children {
		childSpan := angleSpan * float64(child.leafCount) / float64(totalLeaves)
		mid := childStart + childSpan/2
		child.x = n.x + spacing*math.Cos(mid)
		child.y = n.y + spacing*math.Sin(mid)
		layoutRadialSubtree(child, childStart, childSpan, spacing)
		childStart += childSpan
	}
}

type bounds struct {
	minX, minY, maxX, maxY float64
}

func computeBounds(n *layoutNode) bounds {
	b := bounds{
		minX: n.x - n.width/2,
		minY: n.y - n.height/2,
		maxX: n.x + n.width/2,
		maxY: n.y + n.height/2,
	}
	for _, c := range n.children {
		cb := computeBounds(c)
		if cb.minX < b.minX {
			b.minX = cb.minX
		}
		if cb.minY < b.minY {
			b.minY = cb.minY
		}
		if cb.maxX > b.maxX {
			b.maxX = cb.maxX
		}
		if cb.maxY > b.maxY {
			b.maxY = cb.maxY
		}
	}
	return b
}

func renderElements(n *layoutNode, offX, offY, fontSize float64, th Theme, edges, nodes *[]any) {
	cx := n.x + offX
	cy := n.y + offY

	for _, child := range n.children {
		px := n.x + offX
		py := n.y + offY
		ccx := child.x + offX
		ccy := child.y + offY

		edgeCol := edgeColor(child.section, th)
		sw := edgeStrokeWidth(child.depth)

		*edges = append(*edges, &path{
			D:     curvedEdgePath(px, py, ccx, ccy),
			Style: fmt.Sprintf("stroke:%s;stroke-width:%.0f;fill:none", edgeCol, sw),
		})

		renderElements(child, offX, offY, fontSize, th, edges, nodes)
	}

	var classStr string
	if n.depth == 0 {
		classStr = "mindmap-node section-root"
	} else {
		classStr = fmt.Sprintf("mindmap-node section-%d", n.section)
	}
	if n.node.Class != "" {
		classStr += " " + n.node.Class
	}

	*nodes = append(*nodes, &group{
		Class:     classStr,
		Transform: fmt.Sprintf("translate(%.2f,%.2f)", cx, cy),
		Children:  renderShapeElements(n, fontSize, th),
	})
}

func curvedEdgePath(x1, y1, x2, y2 float64) string {
	dx := x2 - x1
	dy := y2 - y1
	dist := math.Sqrt(dx*dx + dy*dy)
	if dist < 1 {
		return fmt.Sprintf("M%.2f,%.2f L%.2f,%.2f", x1, y1, x2, y2)
	}

	nx := -dy / dist
	ny := dx / dist
	offset := dist * 0.08

	mx := (x1+x2)/2 + nx*offset
	my := (y1+y2)/2 + ny*offset

	return fmt.Sprintf("M%.2f,%.2f Q%.2f,%.2f %.2f,%.2f",
		x1, y1, mx, my, x2, y2)
}

func renderShapeElements(n *layoutNode, fontSize float64, th Theme) []any {
	w := n.width
	h := n.height
	fill := shapeFillColor(n.section, n.depth, th)
	textCol := shapeTextColor(n.depth, th)
	style := fmt.Sprintf("fill:%s;stroke:none", fill)

	var children []any

	switch n.node.Shape {
	case diagram.MindmapShapeRound:
		children = append(children, &rect{
			X: svgutil.Float(-w / 2), Y: svgutil.Float(-h / 2),
			Width: svgutil.Float(w), Height: svgutil.Float(h),
			RX: svgutil.Float(h / 2), RY: svgutil.Float(h / 2),
			Style: style,
		})
	case diagram.MindmapShapeSquare:
		children = append(children, &rect{
			X: svgutil.Float(-w / 2), Y: svgutil.Float(-h / 2),
			Width: svgutil.Float(w), Height: svgutil.Float(h),
			Style: style,
		})
	case diagram.MindmapShapeCircle:
		r := w / 2
		if h/2 > r {
			r = h / 2
		}
		children = append(children, &circle{
			CX:    0,
			CY:    0,
			R:     svgutil.Float(r),
			Style: style,
		})
	case diagram.MindmapShapeCloud:
		children = append(children, &path{
			D:     cloudPath(w, h),
			Style: style,
		})
	case diagram.MindmapShapeBang:
		children = append(children, &path{
			D:     bangPath(w, h),
			Style: style,
		})
	case diagram.MindmapShapeHexagon:
		children = append(children, &polygon{
			Points: hexagonPoints(w, h),
			Style:  style,
		})
	default:
		children = append(children, &path{
			D:     defaultNodePath(w, h),
			Style: style,
		})
		children = append(children, &line{
			X1:    svgutil.Float(-w / 2),
			Y1:    svgutil.Float(h / 2),
			X2:    svgutil.Float(w / 2),
			Y2:    svgutil.Float(h / 2),
			Style: fmt.Sprintf("stroke:%s;stroke-width:2;fill:none", fill),
		})
	}

	textStyle := fmt.Sprintf("fill:%s;font-size:%.0fpx", textCol, fontSize)

	// Optional icon caption sits above the label rows. Until the
	// project ships a Font Awesome / Material Design glyph table,
	// the renderer prints the raw class string in a muted italic
	// at the top of the node. This at least surfaces what was
	// declared rather than silently dropping the icon.
	textTopOffset := 0.0
	if n.node.Icon != "" {
		iconStyle := fmt.Sprintf("fill:%s;font-size:%.0fpx;font-style:italic;opacity:0.7", textCol, fontSize-2)
		children = append(children, &text{
			X: 0, Y: svgutil.Float(-h/2 + iconRowH/2 + 2),
			Anchor: "middle", Dominant: "central",
			Style:   iconStyle,
			Content: n.node.Icon,
		})
		textTopOffset = iconRowH / 2
	}

	// Center the stacked label rows in the available space below
	// any icon caption. lineCenters holds each row's y coordinate
	// using the node's center as the origin.
	totalH := 0.0
	for _, lh := range n.lineHeights {
		totalH += lh
	}
	startY := textTopOffset - totalH/2
	for i, segs := range n.segments {
		lh := n.lineHeights[i]
		ly := startY + lh/2
		// Fast path: a single plain segment on this line gets
		// chardata so the tdewolff/canvas PNG rasterizer (which
		// drops text whose content lives in <tspan> children)
		// still draws the label.
		if len(segs) == 1 && !segs[0].bold && !segs[0].italic {
			children = append(children, &text{
				X: 0, Y: svgutil.Float(ly),
				Anchor: "middle", Dominant: "central",
				Style:   textStyle,
				Content: segs[0].text,
			})
		} else {
			var tspans []any
			for _, seg := range segs {
				segStyle := textStyle
				if seg.bold {
					segStyle += ";font-weight:bold"
				}
				if seg.italic {
					segStyle += ";font-style:italic"
				}
				tspans = append(tspans, &svgTspan{Style: segStyle, Content: seg.text})
			}
			children = append(children, &svgText{
				X: 0, Y: svgutil.Float(ly),
				Anchor: "middle", Dominant: "central",
				Style:    textStyle,
				Children: tspans,
			})
		}
		startY += lh
	}
	return children
}
