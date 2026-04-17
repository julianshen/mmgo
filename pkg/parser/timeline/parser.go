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
		if rest, ok := strings.CutPrefix(line, "title "); ok {
			d.Title = strings.TrimSpace(rest)
			continue
		}
		if rest, ok := strings.CutPrefix(line, "section "); ok {
			d.Sections = append(d.Sections, diagram.TimelineSection{Name: strings.TrimSpace(rest)})
			curSection = &d.Sections[len(d.Sections)-1]
			continue
		}
		// Event line: "Time : Event1 : Event2" — split on colons
		if event, ok := parseEvent(line); ok {
			if curSection != nil {
				curSection.Events = append(curSection.Events, event)
			} else {
				d.Events = append(d.Events, event)
			}
		}
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
	if event.Time == "" || len(event.Events) == 0 {
		return diagram.TimelineEvent{}, false
	}
	return event, true
}
