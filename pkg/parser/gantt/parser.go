package gantt

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/julianshen/mmgo/pkg/diagram"
	parserutil "github.com/julianshen/mmgo/pkg/parser"
)

const defaultDateFormat = "2006-01-02"

func Parse(r io.Reader) (*diagram.GanttDiagram, error) {
	p := &parser{
		diagram:    &diagram.GanttDiagram{DateFormat: defaultDateFormat},
		taskByID:   make(map[string]*diagram.GanttTask),
		lastTaskEnd: time.Now(),
	}
	p.scanner = bufio.NewScanner(r)
	p.scanner.Buffer(make([]byte, 0, 64*1024), 1<<20)
	headerSeen := false
	for p.scanner.Scan() {
		p.lineNum++
		line := strings.TrimSpace(parserutil.StripComment(p.scanner.Text()))
		if line == "" {
			continue
		}
		if !headerSeen {
			if line != "gantt" {
				return nil, fmt.Errorf("line %d: expected 'gantt' header, got %q", p.lineNum, line)
			}
			headerSeen = true
			continue
		}
		if err := p.parseLine(line); err != nil {
			return nil, fmt.Errorf("line %d: %w", p.lineNum, err)
		}
	}
	if err := p.scanner.Err(); err != nil {
		return nil, fmt.Errorf("reading input: %w", err)
	}
	if !headerSeen {
		return nil, fmt.Errorf("missing gantt header")
	}
	return p.diagram, nil
}

type parser struct {
	diagram       *diagram.GanttDiagram
	taskByID      map[string]*diagram.GanttTask
	lastTaskEnd   time.Time
	curSection    string
	scanner       *bufio.Scanner
	lineNum       int
}

func (p *parser) parseLine(line string) error {
	if rest, ok := strings.CutPrefix(line, "title "); ok {
		p.diagram.Title = strings.TrimSpace(rest)
		return nil
	}
	if rest, ok := strings.CutPrefix(line, "dateFormat "); ok {
		p.diagram.DateFormat = mermaidToGoFormat(strings.TrimSpace(rest))
		return nil
	}
	if rest, ok := strings.CutPrefix(line, "section "); ok {
		p.curSection = strings.TrimSpace(rest)
		p.diagram.Sections = append(p.diagram.Sections, p.curSection)
		return nil
	}
	if strings.HasPrefix(line, "excludes ") || strings.HasPrefix(line, "todayMarker ") ||
		strings.HasPrefix(line, "axisFormat ") || strings.HasPrefix(line, "tickInterval ") {
		return nil
	}
	return p.parseTask(line)
}

func (p *parser) parseTask(line string) error {
	// Format: "Task Name :status, id, startDate, duration"
	// or: "Task Name :status, id, after otherId, duration"
	colonIdx := strings.Index(line, ":")
	if colonIdx < 0 {
		return nil
	}
	name := strings.TrimSpace(line[:colonIdx])
	meta := strings.TrimSpace(line[colonIdx+1:])
	parts := splitCSV(meta)

	task := diagram.GanttTask{
		Name:    name,
		Section: p.curSection,
	}

	var status diagram.TaskStatus
	idx := 0
	if idx < len(parts) {
		s, consumed := parseStatus(parts[idx])
		if consumed {
			status = s
			idx++
		}
	}
	task.Status = status

	if idx < len(parts) && !isDate(parts[idx], p.diagram.DateFormat) && !strings.HasPrefix(parts[idx], "after ") && !isDuration(parts[idx]) {
		task.ID = parts[idx]
		idx++
	}

	start := p.lastTaskEnd
	if idx < len(parts) {
		if strings.HasPrefix(parts[idx], "after ") {
			afterID := strings.TrimSpace(strings.TrimPrefix(parts[idx], "after "))
			task.After = afterID
			if ref, ok := p.taskByID[afterID]; ok {
				start = ref.End
			}
			idx++
		} else if isDate(parts[idx], p.diagram.DateFormat) {
			t, err := time.Parse(p.diagram.DateFormat, parts[idx])
			if err == nil {
				start = t
			}
			idx++
		}
	}
	task.Start = start

	dur := 24 * time.Hour
	if idx < len(parts) {
		if d, ok := parseDuration(parts[idx]); ok {
			dur = d
		} else if isDate(parts[idx], p.diagram.DateFormat) {
			t, err := time.Parse(p.diagram.DateFormat, parts[idx])
			if err == nil {
				task.End = t
				p.lastTaskEnd = task.End
				if task.ID != "" {
					p.taskByID[task.ID] = &task
				}
				p.diagram.Tasks = append(p.diagram.Tasks, task)
				return nil
			}
		}
	}
	task.End = task.Start.Add(dur)
	p.lastTaskEnd = task.End

	if task.ID != "" {
		p.taskByID[task.ID] = &task
	}
	p.diagram.Tasks = append(p.diagram.Tasks, task)
	return nil
}

func splitCSV(s string) []string {
	parts := strings.Split(s, ",")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return parts
}

func parseStatus(s string) (diagram.TaskStatus, bool) {
	switch s {
	case "done":
		return diagram.TaskStatusDone, true
	case "active":
		return diagram.TaskStatusActive, true
	case "crit":
		return diagram.TaskStatusCrit, true
	default:
		return diagram.TaskStatusNone, false
	}
}

func isDate(s, format string) bool {
	_, err := time.Parse(format, s)
	return err == nil
}

func isDuration(s string) bool {
	_, ok := parseDuration(s)
	return ok
}

func parseDuration(s string) (time.Duration, bool) {
	if strings.HasSuffix(s, "d") {
		n, err := strconv.Atoi(strings.TrimSuffix(s, "d"))
		if err == nil {
			return time.Duration(n) * 24 * time.Hour, true
		}
	}
	if strings.HasSuffix(s, "w") {
		n, err := strconv.Atoi(strings.TrimSuffix(s, "w"))
		if err == nil {
			return time.Duration(n) * 7 * 24 * time.Hour, true
		}
	}
	if strings.HasSuffix(s, "h") {
		n, err := strconv.Atoi(strings.TrimSuffix(s, "h"))
		if err == nil {
			return time.Duration(n) * time.Hour, true
		}
	}
	return 0, false
}

func mermaidToGoFormat(f string) string {
	r := strings.NewReplacer(
		"YYYY", "2006", "YY", "06",
		"MM", "01", "DD", "02",
		"HH", "15", "mm", "04", "ss", "05",
	)
	return r.Replace(f)
}
