package math

import (
	"fmt"
	"strings"

	"golang.org/x/image/font"
	"golang.org/x/image/font/sfnt"
	"golang.org/x/image/math/fixed"
)

// RenderRich renders a LaTeX math expression like Render, but additionally
// supports tokens that go-latex/mtex panics on: top-level commas (`,`),
// superscripts (`^X` / `^{...}`), and subscripts (`_X` / `_{...}`).
//
// When the expression contains none of those tokens it falls back to plain
// Render. Otherwise it splits at the top level, renders each math chunk
// individually, and concatenates the SVG fragments horizontally with the
// special tokens rendered manually (comma glyph, raised/lowered groups).
func RenderRich(expr string, fontSize float64) (svg string, w, h float64, err error) {
	expr = normalizeMathExpr(expr)
	if fontSize <= 0 {
		fontSize = defaultFontSize
	}
	scaled := fontSize * displayScale
	parts, ok := splitTopLevel(expr)
	if !ok || len(parts) == 1 && parts[0].kind == partMath {
		return renderRaw(stripNestedSupSub(expr), scaled)
	}
	return renderParts(parts, scaled)
}

// stripNestedSupSub removes `^X` and `_X` patterns from the expression
// where X is either a single char/macro or a `{...}` group. mtex panics
// on superscripts and subscripts inside groups (e.g. `\frac{a}{b^2}`),
// and there is no way to handle them without a full custom typesetter,
// so the pragmatic fallback is to drop them — the rest of the expression
// then renders structurally, just without the affected sup/sub.
func stripNestedSupSub(expr string) string {
	var sb strings.Builder
	i := 0
	for i < len(expr) {
		c := expr[i]
		if c == '\\' && i+1 < len(expr) {
			sb.WriteByte(c)
			sb.WriteByte(expr[i+1])
			i += 2
			continue
		}
		if c == '^' || c == '_' {
			_, consumed := readOperand(expr[i+1:])
			i += 1 + consumed
			continue
		}
		sb.WriteByte(c)
		i++
	}
	return sb.String()
}

type partKind int

const (
	partMath partKind = iota
	partComma
	partSup
	partSub
)

type rawPart struct {
	kind partKind
	expr string // for partMath; the operand (possibly empty) for partSup/partSub; "" for partComma
}

// splitTopLevel walks expr and splits it at top-level (depth-0) commas,
// `^`, and `_`. Returns ok=false when no special token is found.
//
// Brace and bracket depth is tracked so that `\frac{a,b}{c}` or `f(x, t)`
// inside a `\sqrt{...}` aren't mistakenly split. A top-level `(` or `)`
// is treated as part of the math expression — only `{`/`}` and `[`/`]`
// adjust depth, since LaTeX grouping uses braces.
func splitTopLevel(expr string) (parts []rawPart, ok bool) {
	depth := 0
	last := 0
	flushMath := func(end int) {
		if end > last {
			parts = append(parts, rawPart{kind: partMath, expr: expr[last:end]})
		}
	}
	for i := 0; i < len(expr); i++ {
		c := expr[i]
		switch c {
		case '\\':
			// Skip the next character (escaped).
			if i+1 < len(expr) {
				i++
			}
			continue
		case '{', '[':
			depth++
		case '}', ']':
			if depth > 0 {
				depth--
			}
		case ',':
			if depth == 0 {
				flushMath(i)
				parts = append(parts, rawPart{kind: partComma})
				last = i + 1
				ok = true
			}
		case '^', '_':
			if depth == 0 {
				flushMath(i)
				operand, consumed := readOperand(expr[i+1:])
				kind := partSup
				if c == '_' {
					kind = partSub
				}
				parts = append(parts, rawPart{kind: kind, expr: operand})
				i += consumed
				last = i + 1
				ok = true
			}
		}
	}
	flushMath(len(expr))
	return parts, ok
}

// readOperand reads the operand after `^` or `_`. If s starts with `{`,
// it captures the matching brace group (unwrapped); otherwise it captures
// a single character or a single \macro token. Returns the operand and
// the number of bytes consumed from s.
func readOperand(s string) (operand string, consumed int) {
	if s == "" {
		return "", 0
	}
	if s[0] == '{' {
		depth := 1
		for i := 1; i < len(s); i++ {
			switch s[i] {
			case '\\':
				if i+1 < len(s) {
					i++
				}
			case '{':
				depth++
			case '}':
				depth--
				if depth == 0 {
					return s[1:i], i + 1
				}
			}
		}
		// Unclosed — take the rest.
		return s[1:], len(s)
	}
	if s[0] == '\\' {
		// Capture \name (alpha sequence) as one token.
		j := 1
		for j < len(s) && isAlpha(s[j]) {
			j++
		}
		if j == 1 && j < len(s) {
			j = 2 // single non-alpha character after \
		}
		return s[:j], j
	}
	// Single character.
	return s[:1], 1
}

func isAlpha(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z')
}

// renderParts renders a sequence of split parts into a combined SVG
// fragment. Each math part is rendered via Render and translated by the
// running x-offset; commas, sup, and sub are rendered manually.
//
// All pieces are baseline-aligned at y = baseline. The math renderer
// returns coordinates in a local frame whose top is y=0; we offset each
// piece by (xCursor - localPieceMinX, 0). Since same fontSize produces
// the same baseline location, simple horizontal stacking works.
func renderParts(parts []rawPart, fontSize float64) (svg string, totalW, maxH float64, err error) {
	var sb strings.Builder
	x := 0.0
	maxH = 0.0
	for _, p := range parts {
		switch p.kind {
		case partMath:
			s, pw, ph, e := renderRaw(stripNestedSupSub(p.expr), fontSize)
			if e != nil {
				return "", 0, 0, e
			}
			fmt.Fprintf(&sb, `<g transform="translate(%.3f,0)">%s</g>`, x, s)
			x += pw
			if ph > maxH {
				maxH = ph
			}
		case partComma:
			advance, glyphSVG, err := renderCharGlyph(',', fontSize)
			if err != nil {
				// Fall back to a small gap — better than failing.
				x += fontSize * 0.3
				continue
			}
			// Position comma so its baseline aligns with math baseline.
			// Math output places its baseline near y = ascent of fonts;
			// approximate by placing the comma at y offset of fontSize * 0.85
			// (the typical ascent fraction of x-height + ascent).
			yBaseline := fontSize * 0.85
			fmt.Fprintf(&sb, `<g transform="translate(%.3f,%.3f)">%s</g>`, x, yBaseline, glyphSVG)
			x += advance
		case partSup, partSub:
			subSize := fontSize * 0.7
			// Recurse via the rich path on the inner operand, but at
			// already-scaled size; pass through renderRaw / splitTopLevel
			// directly to avoid re-applying displayScale.
			subParts, hasSplit := splitTopLevel(p.expr)
			var s string
			var pw, ph float64
			var e error
			if !hasSplit || len(subParts) == 1 && subParts[0].kind == partMath {
				s, pw, ph, e = renderRaw(p.expr, subSize)
			} else {
				s, pw, ph, e = renderParts(subParts, subSize)
			}
			if e != nil {
				continue
			}
			var dy float64
			if p.kind == partSup {
				dy = -fontSize * 0.4
			} else {
				dy = fontSize * 0.3
			}
			fmt.Fprintf(&sb, `<g transform="translate(%.3f,%.3f)">%s</g>`, x, dy, s)
			x += pw
			if ph+absFloat(dy) > maxH {
				maxH = ph + absFloat(dy)
			}
		}
	}
	return sb.String(), x, maxH, nil
}

func absFloat(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}

// renderCharGlyph renders a single character via the math font as an SVG
// path string, and returns the glyph's advance width.
func renderCharGlyph(r rune, fontSize float64) (advance float64, svg string, err error) {
	fonts, err := MathFonts()
	if err != nil {
		return 0, "", err
	}
	ft := fonts.Default
	if ft == nil {
		return 0, "", fmt.Errorf("no default font")
	}
	var buf sfnt.Buffer
	idx, err := ft.GlyphIndex(&buf, r)
	if err != nil || idx == 0 {
		return 0, "", fmt.Errorf("glyph %q not found", r)
	}
	ppem := fixed.I(int(fontSize))
	segs, err := ft.LoadGlyph(&buf, idx, ppem, nil)
	if err != nil {
		return 0, "", err
	}
	advFix, err := ft.GlyphAdvance(&buf, idx, ppem, font.HintingNone)
	if err != nil {
		return 0, "", err
	}
	advance = float64(advFix) / 64.0

	var d strings.Builder
	for i, seg := range segs {
		if seg.Op == sfnt.SegmentOpMoveTo && i > 0 {
			fmt.Fprintf(&d, "Z ")
		}
		switch seg.Op {
		case sfnt.SegmentOpMoveTo:
			p := seg.Args[0]
			fmt.Fprintf(&d, "M%.3f %.3f ", float64(p.X)/64, float64(p.Y)/64)
		case sfnt.SegmentOpLineTo:
			p := seg.Args[0]
			fmt.Fprintf(&d, "L%.3f %.3f ", float64(p.X)/64, float64(p.Y)/64)
		case sfnt.SegmentOpQuadTo:
			p1 := seg.Args[0]
			p2 := seg.Args[1]
			fmt.Fprintf(&d, "Q%.3f %.3f %.3f %.3f ",
				float64(p1.X)/64, float64(p1.Y)/64,
				float64(p2.X)/64, float64(p2.Y)/64)
		case sfnt.SegmentOpCubeTo:
			p1 := seg.Args[0]
			p2 := seg.Args[1]
			p3 := seg.Args[2]
			fmt.Fprintf(&d, "C%.3f %.3f %.3f %.3f %.3f %.3f ",
				float64(p1.X)/64, float64(p1.Y)/64,
				float64(p2.X)/64, float64(p2.Y)/64,
				float64(p3.X)/64, float64(p3.Y)/64)
		}
	}
	if d.Len() == 0 {
		return advance, "", nil
	}
	fmt.Fprintf(&d, "Z")
	return advance, fmt.Sprintf(`<path d="%s"/>`, d.String()), nil
}
