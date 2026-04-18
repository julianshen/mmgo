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
	"math"
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
	case parserutil.HasHeaderKeyword(line, "title"):
		rest := trimKeyword(line, "title")
		title, leftover, err := pullLeadingQuote(rest)
		if err != nil {
			return fmt.Errorf("title: %w", err)
		}
		if title == "" {
			title = leftover
		} else if leftover != "" {
			return fmt.Errorf("title: unexpected trailing text %q", leftover)
		}
		d.Title = title
	case parserutil.HasHeaderKeyword(line, "x-axis"):
		axis, err := parseAxis(trimKeyword(line, "x-axis"))
		if err != nil {
			return fmt.Errorf("x-axis: %w", err)
		}
		d.XAxis = axis
	case parserutil.HasHeaderKeyword(line, "y-axis"):
		axis, err := parseAxis(trimKeyword(line, "y-axis"))
		if err != nil {
			return fmt.Errorf("y-axis: %w", err)
		}
		d.YAxis = axis
	case parserutil.HasHeaderKeyword(line, "bar"):
		s, err := parseSeries(diagram.XYSeriesBar, trimKeyword(line, "bar"))
		if err != nil {
			return fmt.Errorf("bar: %w", err)
		}
		d.Series = append(d.Series, s)
	case parserutil.HasHeaderKeyword(line, "line"):
		s, err := parseSeries(diagram.XYSeriesLine, trimKeyword(line, "line"))
		if err != nil {
			return fmt.Errorf("line: %w", err)
		}
		d.Series = append(d.Series, s)
	}
	return nil
}

func trimKeyword(line, kw string) string {
	return strings.TrimSpace(strings.TrimPrefix(line, kw))
}

// pullLeadingQuote extracts an initial double-quoted span and returns
// the inner text plus the whitespace-trimmed remainder. If s doesn't
// start with `"`, title is empty and rest is s unchanged.
func pullLeadingQuote(s string) (title, rest string, err error) {
	if !strings.HasPrefix(s, "\"") {
		return "", s, nil
	}
	end := strings.Index(s[1:], "\"")
	if end < 0 {
		return "", s, fmt.Errorf("unterminated quoted title")
	}
	return s[1 : 1+end], strings.TrimSpace(s[2+end:]), nil
}

// parseAxis accepts:
//   - [a, b, c]                  categorical
//   - "Title" [a, b, c]          categorical with title
//   - min --> max                numeric range
//   - "Title" min --> max        numeric range with title
//   - "Title"                    title only — bounds derived from data
//
// An empty body (e.g. a bare `x-axis` line with no arguments) is an
// error: accepting it would silently lose data the author clearly
// meant to supply.
func parseAxis(s string) (diagram.XYAxis, error) {
	var a diagram.XYAxis
	s = strings.TrimSpace(s)
	if s == "" {
		return a, fmt.Errorf("axis requires a title, category list, or range")
	}

	title, rest, err := pullLeadingQuote(s)
	if err != nil {
		return a, err
	}
	a.Title = title
	s = rest

	if s == "" {
		return a, nil
	}

	if strings.HasPrefix(s, "[") {
		items, err := parseBracketList(s)
		if err != nil {
			return a, err
		}
		if len(items) == 0 {
			return a, fmt.Errorf("category list is empty")
		}
		a.Categories = items
		return a, nil
	}

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
		// NaN/Inf slip past strconv.ParseFloat and through the
		// `minV >= maxV` check (NaN comparisons are always false),
		// so guard explicitly.
		if math.IsNaN(minV) || math.IsInf(minV, 0) || math.IsNaN(maxV) || math.IsInf(maxV, 0) {
			return a, fmt.Errorf("axis bounds must be finite, got %q --> %q", lo, hi)
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

// parseSeries accepts `["title"] [v1, v2, v3]`. An empty value list
// is rejected — a `bar` line with no data is almost certainly a typo.
func parseSeries(t diagram.XYSeriesType, s string) (diagram.XYSeries, error) {
	out := diagram.XYSeries{Type: t}
	s = strings.TrimSpace(s)

	title, rest, err := pullLeadingQuote(s)
	if err != nil {
		return out, err
	}
	out.Title = title
	s = rest

	if !strings.HasPrefix(s, "[") {
		return out, fmt.Errorf("expected '['-delimited value list, got %q", s)
	}
	items, err := parseBracketList(s)
	if err != nil {
		return out, err
	}
	if len(items) == 0 {
		return out, fmt.Errorf("value list is empty")
	}
	out.Data = make([]float64, 0, len(items))
	for _, it := range items {
		v, err := strconv.ParseFloat(strings.TrimSpace(it), 64)
		if err != nil {
			return out, fmt.Errorf("invalid value %q: %w", it, err)
		}
		// strconv.ParseFloat accepts "NaN"/"Inf"; either would poison
		// yRange and produce broken SVG downstream.
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return out, fmt.Errorf("value must be finite, got %q", it)
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
