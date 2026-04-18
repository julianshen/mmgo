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
	switch {
	case parserutil.HasHeaderKeyword(line, "title"):
		d.Title = strings.TrimSpace(strings.TrimPrefix(line, "title"))
	case parserutil.HasHeaderKeyword(line, "x-axis"):
		low, high, err := parseAxis(strings.TrimSpace(strings.TrimPrefix(line, "x-axis")))
		if err != nil {
			return fmt.Errorf("x-axis: %w", err)
		}
		d.XAxisLow, d.XAxisHigh = low, high
	case parserutil.HasHeaderKeyword(line, "y-axis"):
		low, high, err := parseAxis(strings.TrimSpace(strings.TrimPrefix(line, "y-axis")))
		if err != nil {
			return fmt.Errorf("y-axis: %w", err)
		}
		d.YAxisLow, d.YAxisHigh = low, high
	case parserutil.HasHeaderKeyword(line, "quadrant-1"):
		d.Quadrant1 = strings.TrimSpace(strings.TrimPrefix(line, "quadrant-1"))
	case parserutil.HasHeaderKeyword(line, "quadrant-2"):
		d.Quadrant2 = strings.TrimSpace(strings.TrimPrefix(line, "quadrant-2"))
	case parserutil.HasHeaderKeyword(line, "quadrant-3"):
		d.Quadrant3 = strings.TrimSpace(strings.TrimPrefix(line, "quadrant-3"))
	case parserutil.HasHeaderKeyword(line, "quadrant-4"):
		d.Quadrant4 = strings.TrimSpace(strings.TrimPrefix(line, "quadrant-4"))
	default:
		// Anything else with a colon and a `[x, y]` is a data point.
		if idx := strings.LastIndex(line, ":"); idx >= 0 {
			p, err := parsePoint(line[:idx], line[idx+1:])
			if err != nil {
				return err
			}
			d.Points = append(d.Points, p)
		}
	}
	return nil
}

// parseAxis handles `low --> high` or `low-only`. The separator is
// literal `-->` with optional surrounding whitespace.
func parseAxis(s string) (low, high string, err error) {
	if s == "" {
		return "", "", fmt.Errorf("axis requires a label")
	}
	if idx := strings.Index(s, "-->"); idx >= 0 {
		return strings.TrimSpace(s[:idx]), strings.TrimSpace(s[idx+3:]), nil
	}
	// Only a low label (Mermaid permits omitting --> high).
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
