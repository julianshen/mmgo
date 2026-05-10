package mindmap

import (
	"fmt"
	"math"
	"strings"

	"github.com/julianshen/mmgo/pkg/diagram"
	"github.com/julianshen/mmgo/pkg/renderer/svgutil"
	richtext "github.com/julianshen/mmgo/pkg/renderer/text"
	"github.com/julianshen/mmgo/pkg/textmeasure"
)

const (
	defaultFontSize = 14.0
	defaultPadding  = 20.0
	nodePadX        = 20.0
	nodePadY        = 10.0
	minNodeW        = 80.0
	minNodeH        = 35.0
	levelSpacing    = 150.0
	boldWidthFactor = 1.2
)

type Options struct {
	FontSize float64
	Theme    Theme
	Ruler    *textmeasure.Ruler
}

type layoutNode struct {
	node      *diagram.MindmapNode
	x, y      float64
	width     float64
	height    float64
	depth     int
	section   int
	leafCount int
	// segments holds every line's parsed markdown segments in
	// document order. lineHeights holds the per-line vertical
	// extent so the renderer can stack multi-line labels without
	// re-measuring. For single-line labels both have length 1
	// and the existing single-tspan fast path still applies.
	segments    [][]richtext.Segment
	lineHeights []float64
	// extraCSS holds the merged classDef + per-node `style` rule
	// CSS for this node, applied on top of the theme-based fill so
	// `classDef accent fill:#f96` and `style id stroke:#000` show
	// up on the rendered shape.
	extraCSS string
	children []*layoutNode
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
	applyNodeStyles(root, d.CSSClasses, stylesByID(d.Styles))

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
	segments := make([][]richtext.Segment, len(lines))
	lineHeights := make([]float64, len(lines))
	maxLineW := 0.0
	totalH := 0.0
	for i, line := range lines {
		segs := richtext.Parse(line)
		segments[i] = segs
		lineW := 0.0
		lineH := 0.0
		for j, seg := range segs {
			if seg.Math != "" {
				mw, mh := richtext.MathSize(seg.Math)
				segs[j].Width = mw
				if mh > lineH {
					lineH = mh
				}
				lineW += mw
				continue
			}
			sw, lh := ruler.Measure(seg.Text, fontSize)
			if seg.Bold {
				sw *= boldWidthFactor
			}
			segs[j].Width = sw
			lineW += sw
			if lh > lineH {
				lineH = lh
			}
		}
		lineHeights[i] = lineH
		if lineW > maxLineW {
			maxLineW = lineW
		}
		totalH += lineH
	}

	w := maxLineW + 2*nodePadX
	h := totalH + 2*nodePadY
	if w < minNodeW {
		w = minNodeW
	}
	if h < minNodeH {
		h = minNodeH
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

func layoutRadial(root *layoutNode, spacing float64) {
	root.x = 0
	root.y = 0
	if len(root.children) == 0 {
		return
	}
	angles := allocateAngles(root.children, 2*math.Pi, spacing)
	startAngle := -math.Pi / 2
	for i, child := range root.children {
		child.section = i
		assignSectionRecursive(child, i)
		angleSpan := angles[i]
		mid := startAngle + angleSpan/2
		child.x = root.x + spacing*math.Cos(mid)
		child.y = root.y + spacing*math.Sin(mid)
		layoutRadialSubtree(child, startAngle, angleSpan, spacing, 1)
		startAngle += angleSpan
	}
}

func assignSectionRecursive(n *layoutNode, section int) {
	n.section = section
	for _, c := range n.children {
		assignSectionRecursive(c, section)
	}
}

func layoutRadialSubtree(n *layoutNode, startAngle, angleSpan, spacing float64, depth int) {
	if len(n.children) == 0 {
		return
	}
	// Increase spacing with depth so deeper nodes have more angular
	// room (smaller asin(halfW / spacing) → smaller minimum angles).
	effectiveSpacing := spacing * (1 + 0.25*float64(depth))
	angles := allocateAngles(n.children, angleSpan, effectiveSpacing)
	childStart := startAngle
	for i, child := range n.children {
		childSpan := angles[i]
		mid := childStart + childSpan/2
		child.x = n.x + effectiveSpacing*math.Cos(mid)
		child.y = n.y + effectiveSpacing*math.Sin(mid)
		layoutRadialSubtree(child, childStart, childSpan, spacing, depth+1)
		childStart += childSpan
	}
}

// allocateAngles distributes totalAngle among children so that each
// child receives at least enough angular room for its width at the
// given radius (spacing). Minimum angles are always respected;
// remaining angle is distributed proportionally by leaf count.
func allocateAngles(children []*layoutNode, totalAngle, spacing float64) []float64 {
	if len(children) == 0 {
		return nil
	}
	totalLeaves := 0
	for _, c := range children {
		totalLeaves += c.leafCount
	}
	angles := make([]float64, len(children))
	mins := make([]float64, len(children))
	minSum := 0.0
	for i, c := range children {
		halfW := c.width / 2
		// angular half-width needed so the node edges just touch;
		// add a 30 % padding for visual breathing room.
		minHalf := math.Asin(math.Min(1, halfW/spacing))
		mins[i] = 2.6 * minHalf
		minSum += mins[i]
	}
	if minSum >= totalAngle {
		// Not enough room — scale minimums down proportionally.
		scale := totalAngle / minSum
		for i := range angles {
			angles[i] = mins[i] * scale
		}
		return angles
	}
	// Allocate minimums first, then distribute the remainder
	// proportionally by leaf count.
	remainder := totalAngle - minSum
	for i, c := range children {
		prop := remainder * float64(c.leafCount) / float64(totalLeaves)
		angles[i] = mins[i] + prop
	}
	return angles
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
	if len(n.node.CSSClasses) > 0 {
		classStr += " " + strings.Join(n.node.CSSClasses, " ")
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
	// classDef + per-node `style` overrides append after the
	// theme-derived fill so authors win when they set the same
	// property (e.g. `classDef accent fill:#f96` overrides the
	// section color).
	if n.extraCSS != "" {
		style += ";" + n.extraCSS
	}

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

	// Center the stacked label rows vertically inside the node.
	totalH := 0.0
	for _, lh := range n.lineHeights {
		totalH += lh
	}
	startY := -totalH / 2
	for i, segs := range n.segments {
		lh := n.lineHeights[i]
		ly := startY + lh/2
		if len(segs) == 1 && !segs[0].Bold && !segs[0].Italic && segs[0].Math == "" {
			// Fast path: plain text uses chardata, which the
			// tdewolff/canvas PNG rasterizer can render.
			children = append(children, &text{
				X: 0, Y: svgutil.Float(ly),
				Anchor: svgutil.AnchorMiddle, Dominant: svgutil.BaselineCentral,
				Style:   textStyle,
				Content: segs[0].Text,
			})
		} else {
			// Multi-segment text: render each segment as a separate
			// <text> element positioned horizontally.  This works
			// in both browsers and the tdewolff/canvas PNG rasterizer
			// (which drops <tspan> children).
			totalSegW := 0.0
			for _, seg := range segs {
				totalSegW += seg.Width
			}
			xOff := -totalSegW / 2
			for _, seg := range segs {
				if seg.Math != "" {
					res := richtext.RenderMath(seg.Math, lh)
					if res == nil {
						// Fallback to plain text on error.
						children = append(children, &text{
							X: svgutil.Float(xOff + seg.Width/2), Y: svgutil.Float(ly),
							Anchor: svgutil.AnchorMiddle, Dominant: svgutil.BaselineCentral,
							Style:   textStyle,
							Content: seg.Math,
						})
					} else {
						scaledW := res.Width
						scaledH := res.Height
						my := ly + (lh-scaledH)/2 - lh/2 + scaledH/2
						mx := xOff + seg.Width/2 - scaledW/2
						scale := 1.0
						if res.OrigHeight > lh {
							scale = lh / res.OrigHeight
						}
						children = append(children, &group{
							Transform: fmt.Sprintf("translate(%.2f,%.2f) scale(%.3f)", mx, my, scale),
							Children:  res.Elements,
						})
					}
				} else {
					segStyle := textStyle
					if seg.Bold {
						segStyle += ";font-weight:bold"
					}
					if seg.Italic {
						segStyle += ";font-style:italic"
					}
					segX := xOff + seg.Width/2
					children = append(children, &text{
						X: svgutil.Float(segX), Y: svgutil.Float(ly),
						Anchor: svgutil.AnchorMiddle, Dominant: svgutil.BaselineCentral,
						Style:   segStyle,
						Content: seg.Text,
					})
				}
				xOff += seg.Width
			}
		}
		startY += lh
	}
	return children
}

// applyNodeStyles walks the layout tree and stamps each node with
// the merged CSS string from its declared classes (looked up in
// classDefs) plus any per-node override (looked up by node ID in
// stylesByID). Order: classDefs in declaration order, then the
// per-node style — so authors can override a class rule with a
// targeted `style id ...`.
func applyNodeStyles(n *layoutNode, classDefs map[string]string, byID map[string]string) {
	if n == nil {
		return
	}
	var parts []string
	for _, name := range n.node.CSSClasses {
		if css := classDefs[name]; css != "" {
			parts = append(parts, css)
		}
	}
	if css, ok := byID[n.node.ID]; ok && css != "" {
		parts = append(parts, css)
	}
	if len(parts) > 0 {
		n.extraCSS = strings.Join(parts, ";")
	}
	for _, c := range n.children {
		applyNodeStyles(c, classDefs, byID)
	}
}

// stylesByID indexes per-node `style id ...` overrides for fast
// lookup during applyNodeStyles. Later entries override earlier
// ones, matching the document-order semantics other diagrams use.
func stylesByID(styles []diagram.MindmapStyleDef) map[string]string {
	if len(styles) == 0 {
		return nil
	}
	out := make(map[string]string, len(styles))
	for _, s := range styles {
		out[s.NodeID] = s.CSS
	}
	return out
}
