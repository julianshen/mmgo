package math

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/go-latex/latex/drawtex"
	"github.com/go-latex/latex/font/lm"
	"github.com/go-latex/latex/mtex"
	"golang.org/x/image/font/sfnt"
	"golang.org/x/image/math/fixed"
)

var mathCmdRe = regexp.MustCompile(`\\([a-zA-Z]+)`)

var supportedCmds = map[string]bool{
	"frac": true,
	"sqrt": true,
}

// CanRender reports whether expr contains only commands that go-latex/mtex
// supports.  When false, math.Render will fall back to plain text.
func CanRender(expr string) bool {
	for _, m := range mathCmdRe.FindAllStringSubmatch(expr, -1) {
		if !supportedCmds[m[1]] {
			return false
		}
	}
	return true
}

// Render renders a LaTeX math expression to an SVG fragment.
// It returns the SVG content (a sequence of <path> and <rect> elements),
// the width, the height, and any error.
// fontSize controls the rendering scale; when 0 or negative, a default
// of 14 is used.
func Render(expr string, fontSize float64) (svg string, w, h float64, err error) {
	expr = normalizeMathExpr(expr)
	if fontSize <= 0 {
		fontSize = defaultFontSize
	}
	if !CanRender(expr) {
		return "", 0, 0, fmt.Errorf("math render: unsupported command")
	}
	fonts := lm.Fonts()
	r := &svgRenderer{}
	err = mtex.Render(r, expr, fontSize, defaultDPI, fonts)
	if err != nil {
		return "", 0, 0, fmt.Errorf("math render: %w", err)
	}
	return r.String(), r.w, r.h, nil
}

// normalizeMathExpr cleans up common escaping issues in math expressions
// copied from Mermaid sources. Mermaid users often write \\frac when
// they mean \frac because of Markdown/JSON escaping layers.
func normalizeMathExpr(expr string) string {
	return strings.ReplaceAll(expr, `\\`, `\`)
}

const (
	defaultFontSize = 14.0
	defaultDPI      = 72.0
)

// svgRenderer implements mtex.Renderer by converting drawtex
// operations to SVG path data.
type svgRenderer struct {
	sb   strings.Builder
	w, h float64
	dpi  float64
}

func (r *svgRenderer) Render(w, h, dpi float64, cnv *drawtex.Canvas) error {
	// mtex.Render passes w and h in inches (box width/height divided by 72),
	// but canvas operations (GlyphOp.X/Y, RectOp coordinates) are in points.
	// Convert to points so the Y-flip in renderGlyph uses the same unit.
	r.w = w * 72
	r.h = h * 72
	r.dpi = dpi
	var buf sfnt.Buffer
	for _, op := range cnv.Ops() {
		switch o := op.(type) {
		case drawtex.GlyphOp:
			r.renderGlyph(&buf, o)
		case drawtex.RectOp:
			r.renderRect(o)
		}
	}
	return nil
}

func (r *svgRenderer) String() string {
	return r.sb.String()
}

func (r *svgRenderer) renderGlyph(buf *sfnt.Buffer, op drawtex.GlyphOp) {
	g := op.Glyph
	if g.Font == nil {
		return
	}
	ppem := fixed.I(int(g.Size * r.dpi / 72))
	segs, err := g.Font.LoadGlyph(buf, g.Num, ppem, nil)
	if err != nil {
		return
	}

	var d strings.Builder
	for i, seg := range segs {
		if seg.Op == sfnt.SegmentOpMoveTo && i > 0 {
			// Close the previous contour before starting a new one.
			fmt.Fprintf(&d, "Z ")
		}
		switch seg.Op {
		case sfnt.SegmentOpMoveTo:
			p := seg.Args[0]
			x := op.X + float64(p.X)/64
			// TrueType y-positive is UP; SVG is DOWN.
			ty := op.Y - float64(p.Y)/64
			fmt.Fprintf(&d, "M%.3f %.3f ", x, ty)
		case sfnt.SegmentOpLineTo:
			p := seg.Args[0]
			x := op.X + float64(p.X)/64
			ty := op.Y - float64(p.Y)/64
			fmt.Fprintf(&d, "L%.3f %.3f ", x, ty)
		case sfnt.SegmentOpQuadTo:
			p1 := seg.Args[0]
			p2 := seg.Args[1]
			x1 := op.X + float64(p1.X)/64
			y1 := op.Y - float64(p1.Y)/64
			x2 := op.X + float64(p2.X)/64
			y2 := op.Y - float64(p2.Y)/64
			fmt.Fprintf(&d, "Q%.3f %.3f %.3f %.3f ", x1, y1, x2, y2)
		case sfnt.SegmentOpCubeTo:
			p1 := seg.Args[0]
			p2 := seg.Args[1]
			p3 := seg.Args[2]
			x1 := op.X + float64(p1.X)/64
			y1 := op.Y - float64(p1.Y)/64
			x2 := op.X + float64(p2.X)/64
			y2 := op.Y - float64(p2.Y)/64
			x3 := op.X + float64(p3.X)/64
			y3 := op.Y - float64(p3.Y)/64
			fmt.Fprintf(&d, "C%.3f %.3f %.3f %.3f %.3f %.3f ", x1, y1, x2, y2, x3, y3)
		}
	}
	if d.Len() > 0 {
		fmt.Fprintf(&d, "Z")
		fmt.Fprintf(&r.sb, `<path d="%s"/>`, d.String())
	}
}

func (r *svgRenderer) renderRect(op drawtex.RectOp) {
	x := op.X1
	// Canvas and SVG both use y-down: Y1 is the top edge.
	y := op.Y1
	w := op.X2 - op.X1
	h := op.Y2 - op.Y1
	fmt.Fprintf(&r.sb, `<rect x="%.3f" y="%.3f" width="%.3f" height="%.3f"/>`, x, y, w, h)
}
