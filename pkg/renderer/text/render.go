package text

import (
	"fmt"
	"strings"

	"github.com/julianshen/mmgo/pkg/renderer/svgutil"
)

// LabelElements renders a text label (possibly containing $$...$$ math)
// as a slice of SVG elements. It returns plain <text> elements for
// non-math text and <g transform="..."> groups for math segments.
//
// The label is centred at (cx, cy) when anchor is "middle", left-aligned
// at (cx, cy) when "start", and right-aligned when "end".
//
// fontSize is used for text measurement and styling. textStyle should
// be a CSS style string like "fill:#000;font-size:14px".
// dominant sets the SVG dominant-baseline attribute on text elements;
// use dominant for centred labels or svgutil.BaselineAuto
// for baseline-aligned text.
//
// ruler must implement Measure(text string, fontSize float64) (w, h float64).
func LabelElements(label string, cx, cy, fontSize float64, anchor, dominant, textStyle string, ruler interface {
	Measure(text string, fontSize float64) (width, height float64)
}, boldWidthFactor float64) []any {
	lines := strings.Split(label, "\n")
	if len(lines) == 0 {
		return nil
	}

	lineHeight := fontSize * 1.2

	// Compute per-line heights so a line with tall math (e.g. \frac{a}{b})
	// reserves more vertical space than a plain text line.
	parsed := make([][]Segment, len(lines))
	lineHeights := make([]float64, len(lines))
	totalH := 0.0
	for i, line := range lines {
		segs := Parse(line)
		_, lh := MeasureSegments(segs, ruler, fontSize, boldWidthFactor)
		if lh < lineHeight {
			lh = lineHeight
		}
		parsed[i] = segs
		lineHeights[i] = lh
		totalH += lh
	}

	// Centre the block of lines vertically on cy.
	startTop := cy - totalH/2
	yCursor := startTop

	var elems []any
	for i := range lines {
		segs := parsed[i]
		lh := lineHeights[i]
		ly := yCursor + lh/2
		yCursor += lh

		// Fast path: single plain-text segment.
		if len(segs) == 1 && segs[0].Math == "" && !segs[0].Bold && !segs[0].Italic {
			elems = append(elems, &svgutil.Text{
				X:       svgutil.Float(cx),
				Y:       svgutil.Float(ly),
				Anchor:  anchor,
				Dominant: dominant,
				Style:   textStyle,
				Content: segs[0].Text,
			})
			continue
		}

		// Multi-segment line: measure everything first.
		_, _ = MeasureSegments(segs, ruler, fontSize, boldWidthFactor)
		totalW := 0.0
		for _, seg := range segs {
			totalW += seg.Width
		}

		// Determine x-offset based on anchor.
		var xOff float64
		switch anchor {
		case svgutil.AnchorStart:
			xOff = cx
		case svgutil.AnchorEnd:
			xOff = cx - totalW
		default: // middle
			xOff = cx - totalW/2
		}

		fill := extractFill(textStyle)
		for _, seg := range segs {
			if seg.Math != "" {
				res := RenderMath(seg.Math, fontSize, 0, fill)
				if res == nil {
					// Fallback to plain text.
					fx, fanchor := xOff, svgutil.AnchorStart
					switch anchor {
					case svgutil.AnchorMiddle:
						fx, fanchor = xOff+seg.Width/2, svgutil.AnchorMiddle
					case svgutil.AnchorEnd:
						fx, fanchor = xOff+seg.Width, svgutil.AnchorEnd
					}
					elems = append(elems, &svgutil.Text{
						X:        svgutil.Float(fx),
						Y:        svgutil.Float(ly),
						Anchor:   fanchor,
						Dominant: dominant,
						Style:    textStyle,
						Content:  CleanMathFallback(seg.Math),
					})
				} else {
					scale := 1.0
					var mx float64
					switch anchor {
					case svgutil.AnchorStart:
						mx = xOff
					case svgutil.AnchorEnd:
						mx = xOff + seg.Width - res.Width
					default:
						mx = xOff + seg.Width/2 - res.Width/2
					}
					// Vertical alignment depends on dominant-baseline:
					// the math's BOTTOM (its overall bbox bottom) aligns
					// with the surrounding text's letter baseline, so an
					// inline expression like "Result is α" sits on the
					// same line as the text. For dominant="central" the
					// text baseline is ~0.25 em below ly; for "auto" the
					// baseline is at ly directly.
					var textBaselineY float64
					switch dominant {
					case svgutil.BaselineAuto, "", "alphabetic":
						textBaselineY = ly
					case "hanging", "text-before-edge":
						textBaselineY = ly + fontSize
					default: // central, middle
						textBaselineY = ly + fontSize*0.25
					}
					my := textBaselineY - res.Height
					g := &svgutil.Group{
						Transform: fmt.Sprintf("translate(%.2f,%.2f) scale(%.3f)", mx, my, scale),
						Children:  res.Elements,
					}
					if fill != "" {
						g.Style = "fill:" + fill
					}
					elems = append(elems, g)
				}
			} else {
				segStyle := textStyle
				if seg.Bold {
					segStyle += ";font-weight:bold"
				}
				if seg.Italic {
					segStyle += ";font-style:italic"
				}
				var segX float64
				var segAnchor string
				switch anchor {
				case svgutil.AnchorStart:
					segX, segAnchor = xOff, svgutil.AnchorStart
				case svgutil.AnchorEnd:
					segX, segAnchor = xOff+seg.Width, svgutil.AnchorEnd
				default:
					segX, segAnchor = xOff+seg.Width/2, svgutil.AnchorMiddle
				}
				elems = append(elems, &svgutil.Text{
					X:        svgutil.Float(segX),
					Y:        svgutil.Float(ly),
					Anchor:   segAnchor,
					Dominant: dominant,
					Style:    segStyle,
					Content:  seg.Text,
				})
			}
			xOff += seg.Width
		}
	}
	return elems
}

// extractFill pulls the fill colour out of a CSS style string like
// "fill:#000;font-size:14px". Returns "" when no fill is present.
func extractFill(style string) string {
	for _, part := range strings.Split(style, ";") {
		part = strings.TrimSpace(part)
		if strings.HasPrefix(part, "fill:") {
			return strings.TrimPrefix(part, "fill:")
		}
	}
	return ""
}

// MeasureLabel returns the (width, height) of a label that may contain
// $$...$$ math segments and \n line breaks. It uses the provided ruler
// for text measurement and MathSize for math segments.
func MeasureLabel(label string, ruler interface {
	Measure(text string, fontSize float64) (width, height float64)
}, fontSize float64, boldWidthFactor float64) (w, h float64) {
	lines := strings.Split(label, "\n")
	if len(lines) == 0 {
		return 0, 0
	}
	lineHeight := fontSize * 1.2
	maxW := 0.0
	totalH := 0.0
	for _, line := range lines {
		segs := Parse(line)
		lw, lh := MeasureSegments(segs, ruler, fontSize, boldWidthFactor)
		if lw > maxW {
			maxW = lw
		}
		if lh < lineHeight {
			lh = lineHeight
		}
		totalH += lh
	}
	return maxW, totalH
}
