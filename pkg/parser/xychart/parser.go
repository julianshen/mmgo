// Package xychart parses Mermaid xychart-beta diagram syntax:
//
//	xychart-beta [horizontal]
//	    title "chart title"
//	    x-axis ["Title"] [a, b, c, ...]
//	    x-axis ["Title"] min --> max
//	    y-axis ["Title"] [min --> max]
//	    bar   ["series title"] [v1, v2, v3, ...]
//	    line  ["series title"] [v1, v2, v3, ...]
//
// Quoted strings are double-quoted. The bracket list is comma-separated;
// items may themselves be quoted to include commas/spaces.
package xychart

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/julianshen/mmgo/pkg/diagram"
	parserutil "github.com/julianshen/mmgo/pkg/parser"
)

const headerKeyword = "xychart-beta"

func Parse(r io.Reader) (*diagram.XYChartDiagram, error) {
	d := &diagram.XYChartDiagram{}
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 64*1024), 1<<20)

	lineNum := 0
	headerSeen := false
	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(parserutil.StripComment(scanner.Text()))
		if line == "" {
			continue
		}
		if !headerSeen {
			if !parserutil.HasHeaderKeyword(line, headerKeyword) {
				return nil, fmt.Errorf("line %d: expected '%s' header, got %q", lineNum, headerKeyword, line)
			}
			rest := strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(strings.TrimPrefix(line, headerKeyword)), ":"))
			if rest == "horizontal" {
				d.Horizontal = true
			}
			headerSeen = true
			continue
		}
		if err := parseLine(line, d); err != nil {
			return nil, fmt.Errorf("line %d: %w", lineNum, err)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("reading input: %w", err)
	}
	if !headerSeen {
		return nil, fmt.Errorf("missing %s header", headerKeyword)
	}
	return d, nil
}

func parseLine(line string, d *diagram.XYChartDiagram) error {
	switch {
	case hasKeyword(line, "title"):
		d.Title = unquote(strings.TrimSpace(trimKeyword(line, "title")))
	case hasKeyword(line, "x-axis"):
		axis, err := parseAxis(trimKeyword(line, "x-axis"))
		if err != nil {
			return fmt.Errorf("x-axis: %w", err)
		}
		d.XAxis = axis
	case hasKeyword(line, "y-axis"):
		axis, err := parseAxis(trimKeyword(line, "y-axis"))
		if err != nil {
			return fmt.Errorf("y-axis: %w", err)
		}
		d.YAxis = axis
	case hasKeyword(line, "bar"):
		s, err := parseSeries(diagram.XYSeriesBar, trimKeyword(line, "bar"))
		if err != nil {
			return fmt.Errorf("bar: %w", err)
		}
		d.Series = append(d.Series, s)
	case hasKeyword(line, "line"):
		s, err := parseSeries(diagram.XYSeriesLine, trimKeyword(line, "line"))
		if err != nil {
			return fmt.Errorf("line: %w", err)
		}
		d.Series = append(d.Series, s)
	}
	return nil
}

// hasKeyword reports whether line starts with kw followed by space,
// tab, or end-of-string (a word boundary).
func hasKeyword(line, kw string) bool {
	return parserutil.HasHeaderKeyword(line, kw)
}

func trimKeyword(line, kw string) string {
	return strings.TrimSpace(strings.TrimPrefix(line, kw))
}

// parseAxis handles three forms:
//   - [a, b, c]                  → categorical
//   - "Title" [a, b, c]          → categorical with title
//   - min --> max                → numeric range
//   - "Title" min --> max        → numeric range with title
//   - "Title"                    → title only (bounds derived from data)
func parseAxis(s string) (diagram.XYAxis, error) {
	var a diagram.XYAxis
	s = strings.TrimSpace(s)

	// Pull a leading quoted title if present.
	if strings.HasPrefix(s, "\"") {
		end := strings.Index(s[1:], "\"")
		if end < 0 {
			return a, fmt.Errorf("unterminated quoted title")
		}
		a.Title = s[1 : 1+end]
		s = strings.TrimSpace(s[2+end:])
	}

	if s == "" {
		return a, nil
	}

	// Bracket list → categories.
	if strings.HasPrefix(s, "[") {
		items, err := parseBracketList(s)
		if err != nil {
			return a, err
		}
		a.Categories = items
		return a, nil
	}

	// min --> max
	if idx := strings.Index(s, "-->"); idx >= 0 {
		lo := strings.TrimSpace(s[:idx])
		hi := strings.TrimSpace(s[idx+3:])
		minV, err := strconv.ParseFloat(lo, 64)
		if err != nil {
			return a, fmt.Errorf("invalid min %q: %w", lo, err)
		}
		maxV, err := strconv.ParseFloat(hi, 64)
		if err != nil {
			return a, fmt.Errorf("invalid max %q: %w", hi, err)
		}
		if minV >= maxV {
			return a, fmt.Errorf("min (%g) must be less than max (%g)", minV, maxV)
		}
		a.Min = minV
		a.Max = maxV
		a.HasRange = true
		return a, nil
	}

	return a, fmt.Errorf("unrecognized axis form: %q", s)
}

// parseSeries extracts an optional quoted title then a bracket list of
// floats: `["title"] [v1, v2, v3]`.
func parseSeries(t diagram.XYSeriesType, s string) (diagram.XYSeries, error) {
	s = strings.TrimSpace(s)
	out := diagram.XYSeries{Type: t}

	if strings.HasPrefix(s, "\"") {
		end := strings.Index(s[1:], "\"")
		if end < 0 {
			return out, fmt.Errorf("unterminated quoted title")
		}
		out.Title = s[1 : 1+end]
		s = strings.TrimSpace(s[2+end:])
	}

	if !strings.HasPrefix(s, "[") {
		return out, fmt.Errorf("expected '['-delimited value list, got %q", s)
	}
	items, err := parseBracketList(s)
	if err != nil {
		return out, err
	}
	out.Data = make([]float64, 0, len(items))
	for _, it := range items {
		v, err := strconv.ParseFloat(strings.TrimSpace(it), 64)
		if err != nil {
			return out, fmt.Errorf("invalid value %q: %w", it, err)
		}
		out.Data = append(out.Data, v)
	}
	return out, nil
}

// parseBracketList takes `[a, b, "c, d", e]` and returns
// ["a", "b", "c, d", "e"]. The opening bracket must be the first
// character; the closing bracket must be present before end-of-string.
// Commas inside double-quoted items are preserved.
func parseBracketList(s string) ([]string, error) {
	if !strings.HasPrefix(s, "[") {
		return nil, fmt.Errorf("expected '['")
	}
	end := strings.LastIndex(s, "]")
	if end < 0 {
		return nil, fmt.Errorf("missing closing ']'")
	}
	body := s[1:end]
	var out []string
	var cur strings.Builder
	inQuote := false
	for i := 0; i < len(body); i++ {
		c := body[i]
		switch {
		case c == '"':
			inQuote = !inQuote
		case c == ',' && !inQuote:
			out = append(out, strings.TrimSpace(cur.String()))
			cur.Reset()
		default:
			cur.WriteByte(c)
		}
	}
	if cur.Len() > 0 || len(out) > 0 {
		out = append(out, strings.TrimSpace(cur.String()))
	}
	return out, nil
}

// unquote strips surrounding double quotes if both are present.
func unquote(s string) string {
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		return s[1 : len(s)-1]
	}
	return s
}
