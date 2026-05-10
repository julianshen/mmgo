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
//
// ruler must implement Measure(text string, fontSize float64) (w, h float64).
func LabelElements(label string, cx, cy, fontSize float64, anchor, textStyle string, ruler interface {
	Measure(text string, fontSize float64) (width, height float64)
}, boldWidthFactor float64) []any {
	lines := strings.Split(label, "\n")
	if len(lines) == 0 {
		return nil
	}

	lineHeight := fontSize * 1.2
	startY := cy - float64(len(lines)-1)*lineHeight/2

	var elems []any
	for i, line := range lines {
		segs := Parse(line)
		ly := startY + float64(i)*lineHeight

		// Fast path: single plain-text segment.
		if len(segs) == 1 && segs[0].Math == "" && !segs[0].Bold && !segs[0].Italic {
			elems = append(elems, &svgutil.Text{
				X:       svgutil.Float(cx),
				Y:       svgutil.Float(ly),
				Anchor:  anchor,
				Dominant: svgutil.BaselineCentral,
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

		for _, seg := range segs {
			if seg.Math != "" {
				res := RenderMath(seg.Math, lineHeight)
				if res == nil {
					// Fallback to plain text.
					elems = append(elems, &svgutil.Text{
						X:       svgutil.Float(xOff + seg.Width/2),
						Y:       svgutil.Float(ly),
						Anchor:  svgutil.AnchorMiddle,
						Dominant: svgutil.BaselineCentral,
						Style:   textStyle,
						Content: seg.Math,
					})
				} else {
					_, origH := MathSize(seg.Math)
					scale := 1.0
					if origH > lineHeight {
						scale = lineHeight / origH
					}
					mx := xOff + seg.Width/2 - res.Width/2
					// Adjust my so math is vertically centred on the line.
					my := ly + (lineHeight-res.Height)/2 - lineHeight/2 + res.Height/2
					elems = append(elems, &svgutil.Group{
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
				segAnchor := svgutil.AnchorMiddle
				switch anchor {
				case svgutil.AnchorStart:
					segAnchor = svgutil.AnchorStart
				case svgutil.AnchorEnd:
					segAnchor = svgutil.AnchorEnd
				}
				elems = append(elems, &svgutil.Text{
					X:       svgutil.Float(xOff + seg.Width/2),
					Y:       svgutil.Float(ly),
					Anchor:  segAnchor,
					Dominant: svgutil.BaselineCentral,
					Style:   segStyle,
					Content: seg.Text,
				})
			}
			xOff += seg.Width
		}
	}
	return elems
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
	for _, line := range lines {
		segs := Parse(line)
		lw, _ := MeasureSegments(segs, ruler, fontSize, boldWidthFactor)
		if lw > maxW {
			maxW = lw
		}
	}
	totalH := lineHeight * float64(len(lines))
	return maxW, totalH
}
