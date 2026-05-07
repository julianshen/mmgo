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
	d := &diagram.QuadrantChartDiagram{
		Classes: make(map[string]diagram.QuadrantPointStyle),
	}
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 64*1024), 1<<20)

	lineNum := 0
	headerSeen := false
	// inAccDescrBlock toggles when a `accDescr {` line opens a
	// multi-line description; subsequent lines accumulate until
	// the matching `}` line.
	var accDescrLines []string
	inAccDescrBlock := false
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
		if inAccDescrBlock {
			if line == "}" {
				d.AccDescr = strings.Join(accDescrLines, "\n")
				accDescrLines = accDescrLines[:0]
				inAccDescrBlock = false
				continue
			}
			accDescrLines = append(accDescrLines, line)
			continue
		}
		if line == "accDescr {" || line == "accDescr{" {
			inAccDescrBlock = true
			continue
		}
		if v, ok := parserutil.MatchKeywordValue(line, "accTitle"); ok {
			d.AccTitle = v
			continue
		}
		if v, ok := parserutil.MatchKeywordValue(line, "accDescr"); ok {
			d.AccDescr = v
			continue
		}
		if err := parseLine(line, d); err != nil {
			return nil, fmt.Errorf("line %d: %w", lineNum, err)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("reading input: %w", err)
	}
	if inAccDescrBlock {
		return nil, fmt.Errorf("unterminated accDescr { ... } block")
	}
	if !headerSeen {
		return nil, fmt.Errorf("missing %s header", headerKeyword)
	}
	return d, nil
}

func parseLine(line string, d *diagram.QuadrantChartDiagram) error {
	if rest, ok := strings.CutPrefix(line, "classDef "); ok {
		return parseClassDefLine(rest, d)
	}
	// Data points have the shape `Label: [x, y]` (optionally with a
	// `:::className` segment between the label and the colon, or a
	// trailing `color:…, radius:…` style list after `]`). Find the
	// first `: [` separator that opens the coordinate list rather
	// than splitting on the LastIndex of `:` — that approach
	// mishandles labels containing colons (`Time: 9:00 AM`) and
	// breaks the `:::class` shorthand whose colons live before the
	// coordinate list.
	if pos := findCoordSeparator(line); pos >= 0 {
		p, err := parsePoint(line[:pos], line[pos+1:])
		if err != nil {
			return err
		}
		d.Points = append(d.Points, p)
		return nil
	}
	switch {
	case parserutil.HasHeaderKeyword(line, "title"):
		d.Title = parserutil.TrimKeyword(line, "title")
	case parserutil.HasHeaderKeyword(line, "x-axis"):
		low, high, err := parseAxis(parserutil.TrimKeyword(line, "x-axis"))
		if err != nil {
			return fmt.Errorf("x-axis: %w", err)
		}
		d.XAxisLow, d.XAxisHigh = low, high
	case parserutil.HasHeaderKeyword(line, "y-axis"):
		low, high, err := parseAxis(parserutil.TrimKeyword(line, "y-axis"))
		if err != nil {
			return fmt.Errorf("y-axis: %w", err)
		}
		d.YAxisLow, d.YAxisHigh = low, high
	case parserutil.HasHeaderKeyword(line, "quadrant-1"):
		d.Quadrant1 = parserutil.TrimKeyword(line, "quadrant-1")
	case parserutil.HasHeaderKeyword(line, "quadrant-2"):
		d.Quadrant2 = parserutil.TrimKeyword(line, "quadrant-2")
	case parserutil.HasHeaderKeyword(line, "quadrant-3"):
		d.Quadrant3 = parserutil.TrimKeyword(line, "quadrant-3")
	case parserutil.HasHeaderKeyword(line, "quadrant-4"):
		d.Quadrant4 = parserutil.TrimKeyword(line, "quadrant-4")
	}
	return nil
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

// parsePoint handles `Label[:::class]: [x, y][ key:val, key:val ...]`.
// The caller passes the substring before the coordinate-introducing
// colon as `head`, and the substring after it (starting with `[`)
// as `rest`. Trailing text after the style list is rejected so a
// missed closing bracket or a malformed key surfaces cleanly.
func parsePoint(head, rest string) (diagram.QuadrantPoint, error) {
	label, class := splitLabelClass(head)
	p := diagram.QuadrantPoint{Label: label, Class: class}
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
	tail := strings.TrimSpace(rest[end+1:])
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
	if tail != "" {
		style, err := parseStyleList(tail)
		if err != nil {
			return p, fmt.Errorf("point %q: %w", p.Label, err)
		}
		p.Style = style
	}
	return p, nil
}

// splitLabelClass peels an optional trailing `:::name` shorthand
// off a point label. `Foo:::cls` → ("Foo", "cls"); `Foo` → ("Foo",
// "").
func splitLabelClass(s string) (label, class string) {
	s = strings.TrimSpace(s)
	if idx := strings.Index(s, ":::"); idx >= 0 {
		return strings.TrimSpace(s[:idx]), strings.TrimSpace(s[idx+3:])
	}
	return s, ""
}

// findCoordSeparator returns the byte index of the colon that
// opens the coordinate list `: [x, y]`, or -1 if no such colon
// is present. Skips `:::` (the class-shorthand triple-colon), so
// `Foo:::cls: [x, y]` correctly reports the colon before `[`.
func findCoordSeparator(line string) int {
	for i := 0; i < len(line); i++ {
		if line[i] != ':' {
			continue
		}
		// Skip the entire `:::` run by jumping past the third colon.
		if i+2 < len(line) && line[i+1] == ':' && line[i+2] == ':' {
			i += 2
			continue
		}
		// Accept `:` only when followed by whitespace and a `[`.
		j := i + 1
		for j < len(line) && (line[j] == ' ' || line[j] == '\t') {
			j++
		}
		if j < len(line) && line[j] == '[' {
			return i
		}
	}
	return -1
}

// parseClassDefLine handles `classDef <name> key: value, key: value …`.
// Reuses parseStyleList for the right-hand side so the class
// surface mirrors the inline-style surface exactly.
func parseClassDefLine(rest string, d *diagram.QuadrantChartDiagram) error {
	rest = strings.TrimSpace(rest)
	parts := strings.SplitN(rest, " ", 2)
	if len(parts) < 2 {
		return fmt.Errorf("classDef requires a name and at least one property")
	}
	name := strings.TrimSpace(parts[0])
	if name == "" {
		return fmt.Errorf("classDef requires a name")
	}
	style, err := parseStyleList(parts[1])
	if err != nil {
		return fmt.Errorf("classDef %s: %w", name, err)
	}
	d.Classes[name] = style
	return nil
}

// parseStyleList parses Mermaid's inline visual-override syntax for
// quadrant points: a comma-separated list of `key: value` pairs.
// Recognized keys: `color`, `stroke-color`, `radius`,
// `stroke-width`. `radius` is a bare float; `stroke-width` accepts
// a `Npx` suffix that is stripped. Unknown keys error rather than
// being silently dropped so a typo doesn't hide style intent.
func parseStyleList(s string) (diagram.QuadrantPointStyle, error) {
	var style diagram.QuadrantPointStyle
	for _, raw := range strings.Split(s, ",") {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}
		colon := strings.Index(raw, ":")
		if colon < 0 {
			return style, fmt.Errorf("style entry %q: expected `key: value`", raw)
		}
		key := strings.TrimSpace(raw[:colon])
		val := strings.TrimSpace(raw[colon+1:])
		switch key {
		case "color":
			style.Color = val
		case "stroke-color":
			style.StrokeColor = val
		case "radius":
			n, err := strconv.ParseFloat(val, 64)
			if err != nil || n < 0 {
				return style, fmt.Errorf("radius %q: expected non-negative number", val)
			}
			style.Radius = n
		case "stroke-width":
			n, err := strconv.ParseFloat(strings.TrimSuffix(val, "px"), 64)
			if err != nil || n < 0 {
				return style, fmt.Errorf("stroke-width %q: expected non-negative number (with optional `px`)", val)
			}
			style.StrokeWidth = n
		default:
			return style, fmt.Errorf("unknown style key %q", key)
		}
	}
	return style, nil
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
