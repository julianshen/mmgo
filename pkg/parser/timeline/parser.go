package timeline

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	"github.com/julianshen/mmgo/pkg/diagram"
	parserutil "github.com/julianshen/mmgo/pkg/parser"
)

func Parse(r io.Reader) (*diagram.TimelineDiagram, error) {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 1<<20)
	d := &diagram.TimelineDiagram{}
	lineNum := 0
	headerSeen := false
	var curSection *diagram.TimelineSection

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(parserutil.StripComment(scanner.Text()))
		if line == "" {
			continue
		}
		if !headerSeen {
			if line != "timeline" {
				return nil, fmt.Errorf("line %d: expected 'timeline' header, got %q", lineNum, line)
			}
			headerSeen = true
			continue
		}
		if v, ok := parserutil.MatchKeywordValue(line, "title"); ok {
			d.Title = v
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
		if line == "LR" || line == "TD" {
			d.Direction = line
			continue
		}
		if v, ok := parserutil.MatchKeywordValue(line, "direction"); ok {
			if v != "LR" && v != "TD" {
				return nil, fmt.Errorf("line %d: timeline direction must be LR or TD, got %q", lineNum, v)
			}
			d.Direction = v
			continue
		}
		if v, ok := parserutil.MatchKeywordValue(line, "section"); ok {
			d.Sections = append(d.Sections, diagram.TimelineSection{Name: v})
			curSection = &d.Sections[len(d.Sections)-1]
			continue
		}
		// Event line: "Time : Event1 : Event2" — split on colons.
		// A line starting with `:` (empty Time) is a continuation that
		// appends events to the most recent period in the same scope.
		event, ok := parseEvent(line)
		if !ok {
			continue
		}
		target := &d.Events
		if curSection != nil {
			target = &curSection.Events
		}
		if event.Time == "" {
			if n := len(*target); n > 0 {
				(*target)[n-1].Events = append((*target)[n-1].Events, event.Events...)
			}
			continue
		}
		*target = append(*target, event)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("reading input: %w", err)
	}
	if !headerSeen {
		return nil, fmt.Errorf("missing timeline header")
	}
	return d, nil
}

func parseEvent(line string) (diagram.TimelineEvent, bool) {
	parts := strings.Split(line, ":")
	if len(parts) < 2 {
		return diagram.TimelineEvent{}, false
	}
	event := diagram.TimelineEvent{
		Time: strings.TrimSpace(parts[0]),
	}
	for _, p := range parts[1:] {
		if t := strings.TrimSpace(p); t != "" {
			event.Events = append(event.Events, t)
		}
	}
	if len(event.Events) == 0 {
		return diagram.TimelineEvent{}, false
	}
	return event, true
}
