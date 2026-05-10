package mindmap

import (
	"encoding/xml"
	"strconv"
	"strings"

	"github.com/julianshen/mmgo/pkg/renderer/svgutil"
)

type (
	svgDoc  = svgutil.Doc
	rect    = svgutil.Rect
	line    = svgutil.Line
	path    = svgutil.Path
	circle  = svgutil.Circle
	polygon = svgutil.Polygon
	group   = svgutil.Group
	text    = svgutil.Text
)

// parseMathSVG parses the SVG fragment produced by the math renderer
// (a sequence of <path> and <rect> elements) into proper svgutil
// types so they can be marshaled without escaping.
func parseMathSVG(svg string) []any {
	// The math renderer produces self-closing elements with no
	// namespace or nested content, so a lightweight line-based scan
	// is sufficient and avoids pulling in a full XML parser.
	var elems []any
	for {
		svg = strings.TrimSpace(svg)
		if svg == "" {
			break
		}
		// Find the next self-closing element.
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

		// Parse attributes from the tag.
		attrs := parseAttrs(tag)
		switch {
		case strings.HasPrefix(tag, "<path "):
			if d := attrs["d"]; d != "" {
				elems = append(elems, &path{D: d})
			}
		case strings.HasPrefix(tag, "<rect "):
			x, _ := strconv.ParseFloat(attrs["x"], 64)
			y, _ := strconv.ParseFloat(attrs["y"], 64)
			w, _ := strconv.ParseFloat(attrs["width"], 64)
			h, _ := strconv.ParseFloat(attrs["height"], 64)
			elems = append(elems, &rect{
				X:      svgutil.Float(x),
				Y:      svgutil.Float(y),
				Width:  svgutil.Float(w),
				Height: svgutil.Float(h),
			})
		}
	}
	return elems
}

// parseAttrs extracts key="value" pairs from a simple XML start tag.
// It assumes well-formed input with double-quoted values and no
// nested quotes.
func parseAttrs(tag string) map[string]string {
	m := make(map[string]string)
	for {
		// Find next key="value" pair.
		eq := strings.IndexByte(tag, '=')
		if eq < 0 {
			break
		}
		key := strings.TrimSpace(tag[:eq])
		// Skip the '<tagname ' prefix for the first key.
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

// rawSVG holds pre-formatted SVG markup that should be inserted verbatim.
// Deprecated: use parseMathSVG instead to avoid XML escaping issues.
type rawSVG struct {
	Content string
}

func (r rawSVG) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	return e.EncodeToken(xml.CharData(r.Content))
}
