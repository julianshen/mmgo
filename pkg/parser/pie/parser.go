// Package pie parses Mermaid pie chart syntax into a PieDiagram AST.
package pie

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/julianshen/mmgo/pkg/diagram"
	parserutil "github.com/julianshen/mmgo/pkg/parser"
)

func Parse(r io.Reader) (*diagram.PieDiagram, error) {
	d := &diagram.PieDiagram{}
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 1<<20)
	lineNum := 0
	headerSeen := false
	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(parserutil.StripComment(scanner.Text()))
		if line == "" {
			continue
		}
		if !headerSeen {
			if !strings.HasPrefix(line, "pie") {
				return nil, fmt.Errorf("line %d: expected 'pie' header, got %q", lineNum, line)
			}
			rest := strings.TrimSpace(line[len("pie"):])
			parseHeaderFlags(rest, d)
			headerSeen = true
			continue
		}
		if rest, ok := strings.CutPrefix(line, "title "); ok {
			d.Title = strings.TrimSpace(rest)
			continue
		}
		if strings.HasPrefix(line, "showData") {
			d.ShowData = true
			continue
		}
		if err := parseSlice(line, d); err != nil {
			return nil, fmt.Errorf("line %d: %w", lineNum, err)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("reading input: %w", err)
	}
	if !headerSeen {
		return nil, fmt.Errorf("missing pie header")
	}
	return d, nil
}

func parseHeaderFlags(rest string, d *diagram.PieDiagram) {
	if rest, ok := strings.CutPrefix(rest, "title "); ok {
		d.Title = strings.TrimSpace(rest)
		return
	}
	if strings.HasPrefix(rest, "showData") {
		d.ShowData = true
		rest = strings.TrimSpace(rest[len("showData"):])
		if rest, ok := strings.CutPrefix(rest, "title "); ok {
			d.Title = strings.TrimSpace(rest)
		}
	}
}

func parseSlice(line string, d *diagram.PieDiagram) error {
	if !strings.HasPrefix(line, "\"") {
		return fmt.Errorf("slice label must be quoted: %q", line)
	}
	end := strings.Index(line[1:], "\"")
	if end < 0 {
		return fmt.Errorf("unterminated quote in slice: %q", line)
	}
	label := line[1 : end+1]
	rest := strings.TrimSpace(line[end+2:])
	if !strings.HasPrefix(rest, ":") {
		return fmt.Errorf("expected ':' after label in slice: %q", line)
	}
	valStr := strings.TrimSpace(rest[1:])
	val, err := strconv.ParseFloat(valStr, 64)
	if err != nil {
		return fmt.Errorf("invalid slice value %q: %w", valStr, err)
	}
	if val < 0 {
		return fmt.Errorf("slice value must be non-negative, got %v", val)
	}
	d.Slices = append(d.Slices, diagram.Slice{Label: label, Value: val})
	return nil
}

