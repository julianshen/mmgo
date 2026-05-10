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

		expr := normalizeMathExpr(after[:mathEnd])
		result = append(result, Segment{Math: expr})
		remaining = after[mathEnd+2:]
	}

	if len(result) == 0 {
		result = append(result, Segment{Text: s})
	}
	return result
}

// normalizeMathExpr cleans up common escaping issues in math expressions
// copied from Mermaid sources. Mermaid users often write \\frac when
// they mean \frac because of Markdown/JSON escaping layers.
func normalizeMathExpr(expr string) string {
	return strings.ReplaceAll(expr, `\\`, `\`)
}

func parseMarkdown(s string) []Segment {
	var segments []Segment
	remaining := s

	for len(remaining) > 0 {
		// Find the earliest opening marker among ***, **, and *.
		biIdx := strings.Index(remaining, "***")
		boldIdx := indexOfBold(remaining)
		italicIdx := indexOfItalic(remaining)

		firstIdx := -1
		var kind int // 0=***, 1=**, 2=*
		if biIdx >= 0 {
			firstIdx = biIdx
			kind = 0
		}
		if boldIdx >= 0 && (firstIdx < 0 || boldIdx < firstIdx) {
			firstIdx = boldIdx
			kind = 1
		}
		if italicIdx >= 0 && (firstIdx < 0 || italicIdx < firstIdx) {
			firstIdx = italicIdx
			kind = 2
		}

		if firstIdx < 0 {
			// No more markdown markers.
			segments = append(segments, Segment{Text: remaining})
			break
		}

		if firstIdx > 0 {
			segments = append(segments, Segment{Text: remaining[:firstIdx]})
		}

		switch kind {
		case 0: // ***
			end := strings.Index(remaining[firstIdx+3:], "***")
			if end >= 0 {
				segments = append(segments, Segment{
					Text:   remaining[firstIdx+3 : firstIdx+3+end],
					Bold:   true,
					Italic: true,
				})
				remaining = remaining[firstIdx+3+end+3:]
			} else {
				segments = append(segments, Segment{Text: remaining[firstIdx:]})
				remaining = ""
			}
		case 1: // **
			after := remaining[firstIdx+2:]
			end := strings.Index(after, "**")
			if end >= 0 {
				segments = append(segments, Segment{Text: after[:end], Bold: true})
				remaining = after[end+2:]
			} else {
				segments = append(segments, Segment{Text: remaining[firstIdx:]})
				remaining = ""
			}
		case 2: // *
			after := remaining[firstIdx+1:]
			end := strings.Index(after, "*")
			if end >= 0 {
				segments = append(segments, Segment{Text: after[:end], Italic: true})
				remaining = after[end+1:]
			} else {
				segments = append(segments, Segment{Text: remaining[firstIdx:]})
				remaining = ""
			}
		}
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

func mathCacheKey(expr string, fontSize float64) string {
	return fmt.Sprintf("%s|%.2f", expr, fontSize)
}

// MathSize returns the rendered width and height of a LaTeX math
// expression at the given font size, using a package-level cache to
// avoid re-rendering. On panic or error it falls back to a text-based
// estimate.
func MathSize(expr string, fontSize float64) (w, h float64) {
	key := mathCacheKey(expr, fontSize)
	mathSizeCacheMu.RLock()
	if cached, ok := mathSizeCache[key]; ok {
		mathSizeCacheMu.RUnlock()
		return cached.w, cached.h
	}
	mathSizeCacheMu.RUnlock()

	// Fallback dimensions scale with fontSize so layout stays proportional
	// when the renderer is unavailable or the expression is unsupported.
	fallbackW := float64(len(expr)) * fontSize * 0.5
	fallbackH := fontSize * 1.2

	defer func() {
		if r := recover(); r != nil {
			w = fallbackW
			h = fallbackH
			setMathSizeCache(key, w, h)
		}
	}()

	_, w, h, err := math.Render(expr, fontSize)
	if err != nil {
		w = fallbackW
		h = fallbackH
	}
	setMathSizeCache(key, w, h)
	return w, h
}

func setMathSizeCache(key string, w, h float64) {
	mathSizeCacheMu.Lock()
	defer mathSizeCacheMu.Unlock()
	if len(mathSizeCache) >= mathSizeCacheMax {
		mathSizeCache = make(map[string]struct{ w, h float64 })
	}
	mathSizeCache[key] = struct{ w, h float64 }{w, h}
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
			mw, mh := MathSize(seg.Math, fontSize)
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
// When fill is non-empty it is applied directly to every path and
// rect element so the output works in renderers (e.g. tdewolff/canvas)
// that do not support CSS style inheritance from parent <g> nodes.
func RenderMath(expr string, fontSize, targetH float64, fill string) *MathRenderResult {
	var svgMath string
	var mw, mh float64
	var err error
	func() {
		defer func() {
			if r := recover(); r != nil {
				err = fmt.Errorf("math render panic: %v", r)
			}
		}()
		svgMath, mw, mh, err = math.Render(expr, fontSize)
	}()
	if err != nil {
		return nil
	}

	elems := ParseMathSVG(svgMath)
	if len(elems) == 0 {
		return nil
	}

	if fill != "" {
		style := "fill:" + fill
		for i, el := range elems {
			switch e := el.(type) {
			case *svgutil.Path:
				e.Style = style
			case *svgutil.Rect:
				e.Style = style
			}
			elems[i] = el
		}
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

// CleanMathFallback strips backslashes from a math expression so it
// reads as plain text when math rendering fails.  E.g. \alpha + \beta
// becomes "alpha + beta".
func CleanMathFallback(expr string) string {
	var sb strings.Builder
	for i := 0; i < len(expr); i++ {
		if expr[i] == '\\' && i+1 < len(expr) && isAlpha(expr[i+1]) {
			i++ // skip backslash, keep the letter
		}
		sb.WriteByte(expr[i])
	}
	return sb.String()
}

func isAlpha(b byte) bool { return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') }
