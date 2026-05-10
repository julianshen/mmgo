package math

import (
	"fmt"
	"strings"

	"github.com/go-latex/latex/drawtex"
	"github.com/go-latex/latex/font/lm"
	"github.com/go-latex/latex/mtex"
	"golang.org/x/image/font/sfnt"
	"golang.org/x/image/math/fixed"
)

// Render renders a LaTeX math expression to an SVG fragment.
// It returns the SVG content (a sequence of <path> and <rect> elements),
// the width, the height, and any error.
func Render(expr string) (svg string, w, h float64, err error) {
	expr = normalizeMathExpr(expr)
	fonts := lm.Fonts()
	r := &svgRenderer{}
	err = mtex.Render(r, expr, defaultFontSize, defaultDPI, fonts)
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
	r.w = w
	r.h = h
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
	// Convert font units (26.6 fixed point) to SVG coordinates.
	em := float64(ppem) / 64.0
	upem := float64(g.Font.UnitsPerEm())
	scale := em / upem

	var d strings.Builder
	for _, seg := range segs {
		switch seg.Op {
		case sfnt.SegmentOpMoveTo:
			p := seg.Args[0]
			x := op.X + float64(p.X)/64*scale
			y := r.h - (op.Y + float64(p.Y)/64*scale)
			fmt.Fprintf(&d, "M%.3f %.3f ", x, y)
		case sfnt.SegmentOpLineTo:
			p := seg.Args[0]
			x := op.X + float64(p.X)/64*scale
			y := r.h - (op.Y + float64(p.Y)/64*scale)
			fmt.Fprintf(&d, "L%.3f %.3f ", x, y)
		case sfnt.SegmentOpQuadTo:
			p1 := seg.Args[0]
			p2 := seg.Args[1]
			x1 := op.X + float64(p1.X)/64*scale
			y1 := r.h - (op.Y + float64(p1.Y)/64*scale)
			x2 := op.X + float64(p2.X)/64*scale
			y2 := r.h - (op.Y + float64(p2.Y)/64*scale)
			fmt.Fprintf(&d, "Q%.3f %.3f %.3f %.3f ", x1, y1, x2, y2)
		case sfnt.SegmentOpCubeTo:
			p1 := seg.Args[0]
			p2 := seg.Args[1]
			p3 := seg.Args[2]
			x1 := op.X + float64(p1.X)/64*scale
			y1 := r.h - (op.Y + float64(p1.Y)/64*scale)
			x2 := op.X + float64(p2.X)/64*scale
			y2 := r.h - (op.Y + float64(p2.Y)/64*scale)
			x3 := op.X + float64(p3.X)/64*scale
			y3 := r.h - (op.Y + float64(p3.Y)/64*scale)
			fmt.Fprintf(&d, "C%.3f %.3f %.3f %.3f %.3f %.3f ", x1, y1, x2, y2, x3, y3)
		}
	}
	if d.Len() > 0 {
		fmt.Fprintf(&r.sb, `<path d="%s"/>`, d.String())
	}
}

func (r *svgRenderer) renderRect(op drawtex.RectOp) {
	x := op.X1
	y := r.h - op.Y2
	w := op.X2 - op.X1
	h := op.Y2 - op.Y1
	fmt.Fprintf(&r.sb, `<rect x="%.3f" y="%.3f" width="%.3f" height="%.3f"/>`, x, y, w, h)
}
