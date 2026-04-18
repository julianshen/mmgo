// Package quadrant parses Mermaid quadrantChart diagram syntax:
//
//	quadrantChart
//	    title Campaigns
//	    x-axis Low --> High
//	    y-axis Low Engagement --> High Engagement
//	    quadrant-1 We should expand
//	    quadrant-2 Need to promote
//	    quadrant-3 Re-evaluate
//	    quadrant-4 May be improved
//	    Campaign A: [0.3, 0.6]
//
// Data-point lines use `Label: [x, y]` where x and y are in [0, 1].
// Anything else that starts with `<keyword> ` (space) is a directive.
package quadrant

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

const headerKeyword = "quadrantChart"

func Parse(r io.Reader) (*diagram.QuadrantChartDiagram, error) {
	d := &diagram.QuadrantChartDiagram{}
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

func parseLine(line string, d *diagram.QuadrantChartDiagram) error {
	// Data points have the shape `Label: [x, y]`. Check the bracket
	// form before the keyword switch so a label that collides with a
	// directive name (e.g. `title: [0.5, 0.5]`) is still captured as
	// a point. A colon without a bracket is treated as a directive
	// (or silently ignored if unknown) rather than a malformed point,
	// which tolerates forward-compat syntax like `theme: dark`.
	if idx := strings.LastIndex(line, ":"); idx >= 0 {
		if strings.HasPrefix(strings.TrimSpace(line[idx+1:]), "[") {
			p, err := parsePoint(line[:idx], line[idx+1:])
			if err != nil {
				return err
			}
			d.Points = append(d.Points, p)
			return nil
		}
	}
	switch {
	case parserutil.HasHeaderKeyword(line, "title"):
		d.Title = trimKeyword(line, "title")
	case parserutil.HasHeaderKeyword(line, "x-axis"):
		low, high, err := parseAxis(trimKeyword(line, "x-axis"))
		if err != nil {
			return fmt.Errorf("x-axis: %w", err)
		}
		d.XAxisLow, d.XAxisHigh = low, high
	case parserutil.HasHeaderKeyword(line, "y-axis"):
		low, high, err := parseAxis(trimKeyword(line, "y-axis"))
		if err != nil {
			return fmt.Errorf("y-axis: %w", err)
		}
		d.YAxisLow, d.YAxisHigh = low, high
	case parserutil.HasHeaderKeyword(line, "quadrant-1"):
		d.Quadrant1 = trimKeyword(line, "quadrant-1")
	case parserutil.HasHeaderKeyword(line, "quadrant-2"):
		d.Quadrant2 = trimKeyword(line, "quadrant-2")
	case parserutil.HasHeaderKeyword(line, "quadrant-3"):
		d.Quadrant3 = trimKeyword(line, "quadrant-3")
	case parserutil.HasHeaderKeyword(line, "quadrant-4"):
		d.Quadrant4 = trimKeyword(line, "quadrant-4")
	}
	return nil
}

// trimKeyword strips `kw` and any immediately following whitespace or
// colon from the start of `line`. HasHeaderKeyword accepts `:` as a
// word boundary so forms like `title: foo` and `x-axis:Low --> High`
// would otherwise leak the colon into the returned value.
func trimKeyword(line, kw string) string {
	return strings.TrimSpace(strings.TrimLeft(strings.TrimPrefix(line, kw), ": \t"))
}

// parseAxis handles `low --> high` or `low-only`. The separator is
// literal `-->` with optional surrounding whitespace. An empty low
// label (`--> High`) is an error — a writer who wrote the arrow
// clearly intended both ends.
func parseAxis(s string) (low, high string, err error) {
	if s == "" {
		return "", "", fmt.Errorf("axis requires a label")
	}
	if idx := strings.Index(s, "-->"); idx >= 0 {
		low = strings.TrimSpace(s[:idx])
		high = strings.TrimSpace(s[idx+3:])
		if low == "" {
			return "", "", fmt.Errorf("axis low-end label is empty")
		}
		return low, high, nil
	}
	// Low label only — Mermaid permits omitting the arrow and the high end.
	return s, "", nil
}

// parsePoint handles `Label: [x, y]`. The caller has already split on
// the last colon. Trailing text after `]` is rejected so a missed
// closing bracket surfaces cleanly rather than silently truncating.
func parsePoint(label, rest string) (diagram.QuadrantPoint, error) {
	p := diagram.QuadrantPoint{Label: strings.TrimSpace(label)}
	if p.Label == "" {
		return p, fmt.Errorf("point requires a label")
	}
	rest = strings.TrimSpace(rest)
	if !strings.HasPrefix(rest, "[") {
		return p, fmt.Errorf("expected '['-delimited [x, y] after label %q", p.Label)
	}
	end := strings.Index(rest, "]")
	if end < 0 {
		return p, fmt.Errorf("missing closing ']' in point %q", p.Label)
	}
	body := rest[1:end]
	if tail := strings.TrimSpace(rest[end+1:]); tail != "" {
		return p, fmt.Errorf("unexpected trailing text after point %q: %q", p.Label, tail)
	}
	parts := strings.SplitN(body, ",", 2)
	if len(parts) != 2 {
		return p, fmt.Errorf("point %q: expected 'x, y'", p.Label)
	}
	x, err := parseCoord(parts[0])
	if err != nil {
		return p, fmt.Errorf("point %q x: %w", p.Label, err)
	}
	y, err := parseCoord(parts[1])
	if err != nil {
		return p, fmt.Errorf("point %q y: %w", p.Label, err)
	}
	p.X, p.Y = x, y
	return p, nil
}

// parseCoord accepts a single float in [0, 1]. Finite only; values
// outside [0, 1] are rejected because they'd fall outside the plot.
func parseCoord(s string) (float64, error) {
	v, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
	if err != nil {
		return 0, fmt.Errorf("invalid coordinate %q: %w", s, err)
	}
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return 0, fmt.Errorf("coordinate must be finite, got %q", s)
	}
	if v < 0 || v > 1 {
		return 0, fmt.Errorf("coordinate %g outside [0, 1]", v)
	}
	return v, nil
}
