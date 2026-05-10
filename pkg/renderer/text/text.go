// Package text provides rich-text helpers shared across diagram renderers.
// It handles math delimiter splitting ($$...$$), markdown formatting
// (**bold**, *italic*), and math-to-SVG path conversion.
package text

import (
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/julianshen/mmgo/pkg/renderer/math"
	"github.com/julianshen/mmgo/pkg/renderer/svgutil"
)

// Segment is a single piece of a parsed text label — either plain text
// (with optional bold/italic styling) or a LaTeX math expression.
type Segment struct {
	Text   string
	Bold   bool
	Italic bool
	Width  float64 // measured text width at render font size
	Math   string  // LaTeX math expression (if non-empty, Text is ignored)
}

// Parse splits s into segments, handling both math delimiters ($$...$$)
// and markdown formatting (**bold**, *italic*, ***bold+italic***).
func Parse(s string) []Segment {
	var result []Segment
	remaining := s

	for len(remaining) > 0 {
		mathStart := strings.Index(remaining, "$$")
		if mathStart < 0 {
			result = append(result, parseMarkdown(remaining)...)
			break
		}

		if mathStart > 0 {
			result = append(result, parseMarkdown(remaining[:mathStart])...)
		}

		after := remaining[mathStart+2:]
		mathEnd := strings.Index(after, "$$")
		if mathEnd < 0 {
			result = append(result, Segment{Text: remaining[mathStart:]})
			break
		}

		expr := after[:mathEnd]
		result = append(result, Segment{Math: expr})
		remaining = after[mathEnd+2:]
	}

	if len(result) == 0 {
		result = append(result, Segment{Text: s})
	}
	return result
}

func parseMarkdown(s string) []Segment {
	var segments []Segment
	remaining := s

	for len(remaining) > 0 {
		biIdx := strings.Index(remaining, "***")
		if biIdx >= 0 {
			end := strings.Index(remaining[biIdx+3:], "***")
			if end >= 0 {
				if biIdx > 0 {
					segments = append(segments, Segment{Text: remaining[:biIdx]})
				}
				segments = append(segments, Segment{
					Text:   remaining[biIdx+3 : biIdx+3+end],
					Bold:   true,
					Italic: true,
				})
				remaining = remaining[biIdx+3+end+3:]
				continue
			}
		}

		boldIdx := indexOfBold(remaining)
		if boldIdx >= 0 {
			if boldIdx > 0 {
				segments = append(segments, Segment{Text: remaining[:boldIdx]})
			}
			after := remaining[boldIdx+2:]
			end := strings.Index(after, "**")
			if end >= 0 {
				segments = append(segments, Segment{Text: after[:end], Bold: true})
				remaining = after[end+2:]
				continue
			}
			segments = append(segments, Segment{Text: remaining[boldIdx:]})
			remaining = ""
			continue
		}

		italicIdx := indexOfItalic(remaining)
		if italicIdx >= 0 {
			if italicIdx > 0 {
				segments = append(segments, Segment{Text: remaining[:italicIdx]})
			}
			after := remaining[italicIdx+1:]
			end := strings.Index(after, "*")
			if end >= 0 {
				segments = append(segments, Segment{Text: after[:end], Italic: true})
				remaining = after[end+1:]
				continue
			}
			segments = append(segments, Segment{Text: remaining[italicIdx:]})
			remaining = ""
			continue
		}

		segments = append(segments, Segment{Text: remaining})
		remaining = ""
	}

	if len(segments) == 0 {
		segments = append(segments, Segment{Text: s})
	}
	return segments
}

func indexOfBold(s string) int {
	for i := 0; i+1 < len(s); i++ {
		if s[i] == '*' && s[i+1] == '*' {
			if i+2 < len(s) && s[i+2] == '*' {
				return -1
			}
			return i
		}
	}
	return -1
}

func indexOfItalic(s string) int {
	for i := 0; i < len(s); i++ {
		if s[i] == '*' {
			if i+1 < len(s) && s[i+1] == '*' {
				return -1
			}
			if i > 0 && s[i-1] == '*' {
				continue
			}
			return i
		}
	}
	return -1
}

// ------------------------------------------------------------------
// Math size cache (thread-safe)
// ------------------------------------------------------------------

const mathSizeCacheMax = 1000

var (
	mathSizeCache   = make(map[string]struct{ w, h float64 })
	mathSizeCacheMu sync.RWMutex
)

// MathSize returns the rendered width and height of a LaTeX math
// expression, using a package-level cache to avoid re-rendering.
// On panic or error it falls back to a text-based estimate.
func MathSize(expr string) (w, h float64) {
	mathSizeCacheMu.RLock()
	if cached, ok := mathSizeCache[expr]; ok {
		mathSizeCacheMu.RUnlock()
		return cached.w, cached.h
	}
	mathSizeCacheMu.RUnlock()

	defer func() {
		if r := recover(); r != nil {
			w = float64(len(expr)) * 7
			h = 16
			setMathSizeCache(expr, w, h)
		}
	}()

	_, w, h, err := math.Render(expr)
	if err != nil {
		w = float64(len(expr)) * 7
		h = 16
	}
	setMathSizeCache(expr, w, h)
	return w, h
}

func setMathSizeCache(expr string, w, h float64) {
	mathSizeCacheMu.Lock()
	defer mathSizeCacheMu.Unlock()
	if len(mathSizeCache) >= mathSizeCacheMax {
		mathSizeCache = make(map[string]struct{ w, h float64 })
	}
	mathSizeCache[expr] = struct{ w, h float64 }{w, h}
}

// ------------------------------------------------------------------
// SVG path helpers
// ------------------------------------------------------------------

// ParseMathSVG parses the SVG fragment produced by the math renderer
// (a sequence of <path> and <rect> elements) into proper svgutil
// types so they can be marshaled without escaping.
func ParseMathSVG(svg string) []any {
	var elems []any
	for {
		svg = strings.TrimSpace(svg)
		if svg == "" {
			break
		}
		start := strings.IndexByte(svg, '<')
		if start < 0 {
			break
		}
		end := strings.Index(svg[start:], "/>")
		if end < 0 {
			break
		}
		end += start + 2
		tag := svg[start:end]
		svg = svg[end:]

		attrs := parseAttrs(tag)
		switch {
		case strings.HasPrefix(tag, "<path "):
			if d := attrs["d"]; d != "" {
				elems = append(elems, &svgutil.Path{D: d})
			}
		case strings.HasPrefix(tag, "<rect "):
			x, _ := strconv.ParseFloat(attrs["x"], 64)
			y, _ := strconv.ParseFloat(attrs["y"], 64)
			w, _ := strconv.ParseFloat(attrs["width"], 64)
			h, _ := strconv.ParseFloat(attrs["height"], 64)
			elems = append(elems, &svgutil.Rect{
				X:      svgutil.Float(x),
				Y:      svgutil.Float(y),
				Width:  svgutil.Float(w),
				Height: svgutil.Float(h),
			})
		}
	}
	return elems
}

func parseAttrs(tag string) map[string]string {
	m := make(map[string]string)
	for {
		eq := strings.IndexByte(tag, '=')
		if eq < 0 {
			break
		}
		key := strings.TrimSpace(tag[:eq])
		space := strings.LastIndexByte(key, ' ')
		if space >= 0 {
			key = key[space+1:]
		}
		rest := tag[eq+1:]
		rest = strings.TrimLeft(rest, " \t")
		if len(rest) == 0 || rest[0] != '"' {
			break
		}
		rest = rest[1:]
		closeQuote := strings.IndexByte(rest, '"')
		if closeQuote < 0 {
			break
		}
		m[key] = rest[:closeQuote]
		tag = rest[closeQuote+1:]
	}
	return m
}

// ------------------------------------------------------------------
// Render helpers
// ------------------------------------------------------------------

// MeasureSegments measures each segment using the provided ruler and
// fontSize, returning the total width and maximum height. Math
// segments are measured via MathSize.
func MeasureSegments(segs []Segment, ruler interface {
	Measure(text string, fontSize float64) (width, height float64)
}, fontSize float64, boldWidthFactor float64) (totalW, maxH float64) {
	for i, seg := range segs {
		if seg.Math != "" {
			mw, mh := MathSize(seg.Math)
			segs[i].Width = mw
			totalW += mw
			if mh > maxH {
				maxH = mh
			}
			continue
		}
		sw, lh := ruler.Measure(seg.Text, fontSize)
		if seg.Bold {
			sw *= boldWidthFactor
		}
		segs[i].Width = sw
		totalW += sw
		if lh > maxH {
			maxH = lh
		}
	}
	return totalW, maxH
}

// MathRenderResult holds the output of a successful math render call.
type MathRenderResult struct {
	Elements    []any   // svgutil.Path / svgutil.Rect elements
	Width       float64 // scaled width
	Height      float64 // scaled height
	OrigWidth   float64 // original unscaled width
	OrigHeight  float64 // original unscaled height
}

// RenderMath renders a math expression to SVG elements with optional
// scaling to fit a target height. Returns nil elements on error.
func RenderMath(expr string, targetH float64) *MathRenderResult {
	var svgMath string
	var mw, mh float64
	var err error
	func() {
		defer func() {
			if r := recover(); r != nil {
				err = fmt.Errorf("math render panic: %v", r)
			}
		}()
		svgMath, mw, mh, err = math.Render(expr)
	}()
	if err != nil {
		return nil
	}

	elems := ParseMathSVG(svgMath)
	if len(elems) == 0 {
		return nil
	}

	scale := 1.0
	if targetH > 0 && mh > targetH {
		scale = targetH / mh
	}

	return &MathRenderResult{
		Elements:   elems,
		Width:      mw * scale,
		Height:     mh * scale,
		OrigWidth:  mw,
		OrigHeight: mh,
	}
}
