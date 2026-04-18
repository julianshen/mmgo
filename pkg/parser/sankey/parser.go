// Package sankey parses Mermaid sankey-beta diagram syntax. Rows are
// comma-separated `source,target,value`. Standard CSV quoting rules
// apply: a field may be wrapped in double quotes to include commas or
// whitespace, and `""` inside a quoted field is an escaped quote.
package sankey

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"io"
	"math"
	"strconv"
	"strings"

	"github.com/julianshen/mmgo/pkg/diagram"
	parserutil "github.com/julianshen/mmgo/pkg/parser"
)

func Parse(r io.Reader) (*diagram.SankeyDiagram, error) {
	d := &diagram.SankeyDiagram{}
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 64*1024), 1<<20)
	lineNum := 0
	headerSeen := false
	// The optional `source,target,value` column-header row is only
	// meaningful as the first data row. Checking every line would
	// silently skip a legitimate flow whose source/target/value
	// happen to be literally "source", "target", "value".
	firstDataRow := true

	for scanner.Scan() {
		lineNum++
		raw := scanner.Text()
		line := strings.TrimSpace(parserutil.StripComment(raw))
		if line == "" {
			continue
		}
		if !headerSeen {
			if !isHeader(line) {
				return nil, fmt.Errorf("line %d: expected 'sankey-beta' header, got %q", lineNum, line)
			}
			headerSeen = true
			continue
		}
		if firstDataRow {
			firstDataRow = false
			if isColumnHeader(line) {
				continue
			}
		}
		flow, err := parseRow(line)
		if err != nil {
			return nil, fmt.Errorf("line %d: %w", lineNum, err)
		}
		d.Flows = append(d.Flows, flow)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("reading input: %w", err)
	}
	if !headerSeen {
		return nil, fmt.Errorf("missing sankey-beta header")
	}
	return d, nil
}

func isHeader(line string) bool {
	return parserutil.HasHeaderKeyword(line, "sankey-beta")
}

// isColumnHeader matches a literal `source,target,value` row (any
// case, any whitespace around fields). Mermaid tolerates it as an
// optional header row.
func isColumnHeader(line string) bool {
	fields, err := parseCSVLine(line)
	if err != nil || len(fields) != 3 {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(fields[0]), "source") &&
		strings.EqualFold(strings.TrimSpace(fields[1]), "target") &&
		strings.EqualFold(strings.TrimSpace(fields[2]), "value")
}

func parseRow(line string) (diagram.SankeyFlow, error) {
	fields, err := parseCSVLine(line)
	if err != nil {
		return diagram.SankeyFlow{}, err
	}
	if len(fields) != 3 {
		return diagram.SankeyFlow{}, fmt.Errorf("expected 3 columns (source,target,value), got %d", len(fields))
	}
	src := strings.TrimSpace(fields[0])
	dst := strings.TrimSpace(fields[1])
	valStr := strings.TrimSpace(fields[2])
	if src == "" || dst == "" {
		return diagram.SankeyFlow{}, fmt.Errorf("source and target must be non-empty")
	}
	val, err := strconv.ParseFloat(valStr, 64)
	if err != nil {
		return diagram.SankeyFlow{}, fmt.Errorf("invalid value %q: %w", valStr, err)
	}
	// strconv.ParseFloat accepts "NaN", "Inf", "+Inf", "-Inf" as valid
	// floats. A NaN < 0 comparison is false, so NaN would slip past the
	// non-negative check and poison the renderer's magnitude math.
	if math.IsNaN(val) || math.IsInf(val, 0) {
		return diagram.SankeyFlow{}, fmt.Errorf("value must be finite, got %q", valStr)
	}
	if val < 0 {
		return diagram.SankeyFlow{}, fmt.Errorf("value must be non-negative, got %g", val)
	}
	if src == dst {
		return diagram.SankeyFlow{}, fmt.Errorf("self-loop not allowed: %q → %q", src, dst)
	}
	return diagram.SankeyFlow{Source: src, Target: dst, Value: val}, nil
}

// parseCSVLine uses encoding/csv for a single record so quoting and
// embedded commas are handled consistently with the Mermaid JS
// implementation (which delegates to a standard CSV parser too).
func parseCSVLine(line string) ([]string, error) {
	r := csv.NewReader(strings.NewReader(line))
	r.TrimLeadingSpace = true
	return r.Read()
}
