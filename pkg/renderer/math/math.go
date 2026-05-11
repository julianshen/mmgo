package math

import (
	"fmt"
	"strings"

	"github.com/go-latex/latex/drawtex"
	"github.com/go-latex/latex/mtex"
	"golang.org/x/image/font/sfnt"
	"golang.org/x/image/math/fixed"
)

// Render renders a LaTeX math expression to an SVG fragment.
// It returns the SVG content (a sequence of <path> and <rect> elements),
// the width, the height, and any error.
// fontSize controls the rendering scale; when 0 or negative, a default
// of 14 is used.
func Render(expr string, fontSize float64) (svg string, w, h float64, err error) {
	s, w, h, _, err := RenderWithBaseline(expr, fontSize)
	return s, w, h, err
}

// RenderWithBaseline is like Render but also returns the y-position of
// the expression's primary baseline within the local SVG coordinate
// system (where y=0 is the top of the rendered content). Callers that
// need to align math with surrounding text by baseline use this value.
func RenderWithBaseline(expr string, fontSize float64) (svg string, w, h, baseline float64, err error) {
	expr = normalizeMathExpr(expr)
	if fontSize <= 0 {
		fontSize = defaultFontSize
	}
	return renderRawWithBaseline(expr, fontSize*displayScale)
}

// renderRaw renders without applying displayScale; it's used internally
// so the rich-renderer can apply displayScale once at the top level
// instead of compounding it across recursive calls.
func renderRaw(expr string, fontSize float64) (svg string, w, h float64, err error) {
	s, w, h, _, err := renderRawWithBaseline(expr, fontSize)
	return s, w, h, err
}

func renderRawWithBaseline(expr string, fontSize float64) (svg string, w, h, baseline float64, err error) {
	// go-latex/mtex only resolves Greek letters and math operators
	// when the parser is in math mode.  Bare expressions like \alpha
	// are parsed in text mode and fall back to the backslash glyph.
	// Wrapping in $...$ activates math mode for the whole expression.
	if !strings.HasPrefix(expr, "$") {
		expr = "$" + expr + "$"
	}
	fonts, err := MathFonts()
	if err != nil {
		return "", 0, 0, 0, fmt.Errorf("math render: %w", err)
	}
	r := &svgRenderer{}
	err = mtex.Render(r, expr, fontSize, defaultDPI, fonts)
	if err != nil {
		return "", 0, 0, 0, fmt.Errorf("math render: %w", err)
	}
	// mtex's reported h is the bbox ascent — it understates total height
	// for expressions with descenders or fractions. Use the observed
	// y-range instead, and wrap the output in a translation so content
	// starts at y=0.
	if r.hasY {
		baseline = medianBaseline(r.glyphYs) - r.minY
		out := fmt.Sprintf(`<g transform="translate(0,%.3f)">%s</g>`, -r.minY, r.String())
		return out, r.w, r.maxY - r.minY, baseline, nil
	}
	return r.String(), r.w, r.h, r.h, nil
}

// medianBaseline returns the median value of a slice of glyph baselines.
// Median (rather than mean) keeps a stray descender or a stacked
// numerator/denominator from biasing the result away from the inline
// baseline that most glyphs share.
func medianBaseline(ys []float64) float64 {
	if len(ys) == 0 {
		return 0
	}
	sorted := make([]float64, len(ys))
	copy(sorted, ys)
	// Insertion sort — n is small.
	for i := 1; i < len(sorted); i++ {
		for j := i; j > 0 && sorted[j-1] > sorted[j]; j-- {
			sorted[j-1], sorted[j] = sorted[j], sorted[j-1]
		}
	}
	mid := len(sorted) / 2
	if len(sorted)%2 == 1 {
		return sorted[mid]
	}
	return (sorted[mid-1] + sorted[mid]) / 2
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
	// displayScale enlarges math output relative to the surrounding text
	// font size. Noto Sans Math glyphs are visually a bit smaller than
	// ordinary 16px text; a modest 1.2x bump brings the math to roughly
	// the same x-height as adjacent text without overshadowing it.
	displayScale = 1.2
	// trackingScale stretches glyph and rect x-positions horizontally.
	// mtex's stock spacing between binary operators (≈ 0.2 em) renders
	// quite tight against Noto Sans Math glyphs, and even adjacent
	// letters (dx, dt, f(x) come out touching. A 1.22x multiplier on
	// glyph anchor x positions opens up readable gaps; glyph path
	// coordinates themselves are not scaled, only placement.
	trackingScale = 1.75
	// barOverhang extends fraction bars and \sqrt vinculum rects by a
	// small amount on each side so they reach slightly past the
	// numerator/denominator/radicand — matches typesetting convention
	// and stops a fraction like 1/2 from having a bar exactly the width
	// of "1".
	barOverhang = 1.5
)

// svgRenderer implements mtex.Renderer by converting drawtex
// operations to SVG path data.
type svgRenderer struct {
	sb         strings.Builder
	w, h       float64
	dpi        float64
	minY, maxY float64
	hasY       bool
	// glyphYs collects every glyph's op.Y (the baseline at which mtex
	// placed that glyph). The median of these is a good approximation
	// of the expression's primary baseline — for pure inline math every
	// glyph shares the same op.Y, and for stacked content (fractions,
	// roots) the median lands near the math axis.
	glyphYs []float64
}

func (r *svgRenderer) noteY(y float64) {
	if !r.hasY {
		r.minY, r.maxY = y, y
		r.hasY = true
		return
	}
	if y < r.minY {
		r.minY = y
	}
	if y > r.maxY {
		r.maxY = y
	}
}

func (r *svgRenderer) Render(w, h, dpi float64, cnv *drawtex.Canvas) error {
	// mtex.Render passes w and h in inches (box width/height divided by 72),
	// but canvas operations (GlyphOp.X/Y, RectOp coordinates) are in points.
	// Convert to points so the Y-flip in renderGlyph uses the same unit.
	r.w = w * 72 * trackingScale
	r.h = h * 72
	r.dpi = dpi
	var buf sfnt.Buffer
	for _, op := range cnv.Ops() {
		switch o := op.(type) {
		case drawtex.GlyphOp:
			r.glyphYs = append(r.glyphYs, o.Y)
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
	// Stretch the glyph's anchor x-position to add inter-glyph breathing
	// room without distorting glyph shapes themselves.
	op.X *= trackingScale
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
			// sfnt.LoadGlyph already returns segment coordinates in
			// y-down (origin at the glyph's baseline reference point);
			// add directly without flipping.
			ty := op.Y + float64(p.Y)/64
			r.noteY(ty)
			fmt.Fprintf(&d, "M%.3f %.3f ", x, ty)
		case sfnt.SegmentOpLineTo:
			p := seg.Args[0]
			x := op.X + float64(p.X)/64
			ty := op.Y + float64(p.Y)/64
			r.noteY(ty)
			fmt.Fprintf(&d, "L%.3f %.3f ", x, ty)
		case sfnt.SegmentOpQuadTo:
			p1 := seg.Args[0]
			p2 := seg.Args[1]
			x1 := op.X + float64(p1.X)/64
			y1 := op.Y + float64(p1.Y)/64
			x2 := op.X + float64(p2.X)/64
			y2 := op.Y + float64(p2.Y)/64
			r.noteY(y1)
			r.noteY(y2)
			fmt.Fprintf(&d, "Q%.3f %.3f %.3f %.3f ", x1, y1, x2, y2)
		case sfnt.SegmentOpCubeTo:
			p1 := seg.Args[0]
			p2 := seg.Args[1]
			p3 := seg.Args[2]
			x1 := op.X + float64(p1.X)/64
			y1 := op.Y + float64(p1.Y)/64
			x2 := op.X + float64(p2.X)/64
			y2 := op.Y + float64(p2.Y)/64
			x3 := op.X + float64(p3.X)/64
			y3 := op.Y + float64(p3.Y)/64
			r.noteY(y1)
			r.noteY(y2)
			r.noteY(y3)
			fmt.Fprintf(&d, "C%.3f %.3f %.3f %.3f %.3f %.3f ", x1, y1, x2, y2, x3, y3)
		}
	}
	if d.Len() > 0 {
		fmt.Fprintf(&d, "Z")
		fmt.Fprintf(&r.sb, `<path d="%s"/>`, d.String())
	}
}

func (r *svgRenderer) renderRect(op drawtex.RectOp) {
	// Bars come in two flavours that need different treatment:
	//
	//   - \frac bar: starts at op.X1≈0, sits over numerator/denominator
	//     glyphs that are themselves at op.X=0 (so they don't move under
	//     tracking). Width stays at mtex's natural extent + small overhang.
	//
	//   - \sqrt vinculum: starts at op.X1 > 0 (just after the √ glyph)
	//     and must cover the radicand, which DID move right under
	//     tracking. Scale the width by trackingScale so the bar still
	//     reaches past the now-shifted radicand.
	//
	// Width is in mtex coordinates and the glyph visual extent can
	// exceed its advance (italic math letters lean past their advance),
	// so we add a generous overhang on the right.
	x := op.X1 - barOverhang
	w := (op.X2 - op.X1)
	if op.X1 > 1 {
		// \sqrt vinculum — grow width with tracking so it still covers
		// the radicand at its new shifted position.
		w *= trackingScale
	}
	w += 2 * barOverhang
	// Canvas and SVG both use y-down: Y1 is the top edge.
	y := op.Y1
	h := op.Y2 - op.Y1
	r.noteY(y)
	r.noteY(y + h)
	fmt.Fprintf(&r.sb, `<rect x="%.3f" y="%.3f" width="%.3f" height="%.3f"/>`, x, y, w, h)
}
