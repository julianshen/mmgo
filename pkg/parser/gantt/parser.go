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
		diagram:  &diagram.GanttDiagram{DateFormat: defaultDateFormat},
		taskByID: make(map[string]*diagram.GanttTask),
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
	p.resolveForwardRefs()
	return p.diagram, nil
}

// resolveForwardRefs runs after the line loop completes so that
// `after id` / `until id` references pointing at tasks declared
// LATER in the source pick up the right anchor. The first pass
// fell back to `lastTaskEnd` when an id wasn't yet known; here we
// recompute against the now-complete taskByID map.
//
// For `after`, we preserve the task's effective duration (parsed
// in pass 1) and re-anchor: newEnd = newStart + (oldEnd - oldStart).
// We use the small (oldEnd-oldStart) delta rather than (newStart-
// oldStart) because oldStart can be the zero time when nothing was
// resolved yet, and the gap between zero-time and real dates can
// exceed time.Duration's ~290-year range.
//
// `until` is always recomputed since the map can only grow.
func (p *parser) resolveForwardRefs() {
	for i := range p.diagram.Tasks {
		t := &p.diagram.Tasks[i]
		if len(t.After) > 0 {
			dur := t.End.Sub(t.Start)
			t.Start = p.maxEndOf(t.After)
			if len(t.Until) == 0 {
				t.End = t.Start.Add(dur)
			}
		}
		if len(t.Until) > 0 {
			t.End = p.minStartOf(t.Until)
		}
	}
}

type parser struct {
	diagram     *diagram.GanttDiagram
	taskByID    map[string]*diagram.GanttTask
	lastTaskEnd time.Time
	curSection  string
	scanner     *bufio.Scanner
	lineNum     int
}

func (p *parser) parseLine(line string) error {
	if v, ok := parserutil.MatchKeywordValue(line, "title"); ok {
		p.diagram.Title = v
		return nil
	}
	if v, ok := parserutil.MatchKeywordValue(line, "accTitle"); ok {
		p.diagram.AccTitle = v
		return nil
	}
	if v, ok := parserutil.MatchKeywordValue(line, "accDescr"); ok {
		p.diagram.AccDescr = v
		return nil
	}
	if v, ok := parserutil.MatchKeywordValue(line, "dateFormat"); ok {
		p.diagram.DateFormat = mermaidToGoFormat(v)
		return nil
	}
	if v, ok := parserutil.MatchKeywordValue(line, "axisFormat"); ok {
		p.diagram.AxisFormat = v
		return nil
	}
	if v, ok := parserutil.MatchKeywordValue(line, "tickInterval"); ok {
		p.diagram.TickInterval = v
		return nil
	}
	if v, ok := parserutil.MatchKeywordValue(line, "weekday"); ok {
		p.diagram.Weekday = v
		return nil
	}
	if v, ok := parserutil.MatchKeywordValue(line, "todayMarker"); ok {
		p.diagram.TodayMarker = v
		return nil
	}
	if v, ok := parserutil.MatchKeywordValue(line, "excludes"); ok {
		p.diagram.Excludes = append(p.diagram.Excludes, splitSpaceOrCSV(v)...)
		return nil
	}
	if v, ok := parserutil.MatchKeywordValue(line, "includes"); ok {
		p.diagram.Includes = append(p.diagram.Includes, splitSpaceOrCSV(v)...)
		return nil
	}
	if v, ok := parserutil.MatchKeywordValue(line, "section"); ok {
		p.curSection = v
		p.diagram.Sections = append(p.diagram.Sections, p.curSection)
		return nil
	}
	if strings.HasPrefix(line, "click ") {
		return p.parseClick(line)
	}
	if strings.HasPrefix(line, "vert ") {
		return p.parseVert(line)
	}
	return p.parseTask(line)
}

// parseClick handles `click TASKID href "url" ["tooltip"] ["target"]`
// and `click TASKID call func(args) ["tooltip"]`. The TASKID must
// match a task declared earlier in the source.
func (p *parser) parseClick(line string) error {
	rest := strings.TrimSpace(strings.TrimPrefix(line, "click"))
	parts, err := parserutil.SplitClickArgs(rest, 2)
	if err != nil {
		return fmt.Errorf("click: %w", err)
	}
	if len(parts) < 2 {
		return fmt.Errorf("click requires task id and target")
	}
	id := parts[0]
	if _, ok := p.taskByID[id]; !ok {
		return fmt.Errorf("click references undefined task %q", id)
	}
	afterID := strings.TrimSpace(rest[len(id):])
	cd := diagram.GanttClickDef{TaskID: id}
	switch {
	case afterID == "call" || strings.HasPrefix(afterID, "call "):
		callback := strings.TrimSpace(strings.TrimPrefix(afterID, "call"))
		if callback == "" {
			return fmt.Errorf("click %s: missing callback after `call`", id)
		}
		cd.Callback = callback
	case afterID == "href" || strings.HasPrefix(afterID, "href "):
		argSrc := strings.TrimSpace(strings.TrimPrefix(afterID, "href"))
		if err := fillClickURLArgs(&cd, argSrc); err != nil {
			return fmt.Errorf("click %s: %w", id, err)
		}
	default:
		if err := fillClickURLArgs(&cd, afterID); err != nil {
			return fmt.Errorf("click %s: %w", id, err)
		}
	}
	p.diagram.Clicks = append(p.diagram.Clicks, cd)
	return nil
}

func fillClickURLArgs(cd *diagram.GanttClickDef, src string) error {
	parts, err := parserutil.SplitClickArgs(src, 3)
	if err != nil {
		return err
	}
	if len(parts) == 0 || parts[0] == "" {
		return fmt.Errorf("missing URL")
	}
	cd.URL = parts[0]
	if len(parts) >= 2 {
		cd.Tooltip = parts[1]
	}
	if len(parts) >= 3 {
		cd.Target = parts[2]
	}
	return nil
}

// parseVert handles `vert <date>` and `vert <id>, <date> [, "label"]`.
// The label form attaches optional text drawn at the rule's top.
func (p *parser) parseVert(line string) error {
	rest := strings.TrimSpace(strings.TrimPrefix(line, "vert"))
	if rest == "" {
		return fmt.Errorf("vert requires a date")
	}
	parts := splitCSV(rest)
	v := diagram.GanttVert{}
	switch len(parts) {
	case 1:
		t, err := time.Parse(p.diagram.DateFormat, parts[0])
		if err != nil {
			return fmt.Errorf("vert: invalid date %q: %w", parts[0], err)
		}
		v.Date = t
	case 2, 3:
		v.ID = parts[0]
		t, err := time.Parse(p.diagram.DateFormat, parts[1])
		if err != nil {
			return fmt.Errorf("vert %s: invalid date %q: %w", v.ID, parts[1], err)
		}
		v.Date = t
		if len(parts) == 3 {
			v.Label = parserutil.Unquote(parts[2])
		}
	default:
		return fmt.Errorf("vert: expected `date` or `id, date [, label]`")
	}
	p.diagram.Verts = append(p.diagram.Verts, v)
	return nil
}

func (p *parser) parseTask(line string) error {
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

	idx := 0
	// Tag list: any leading slots that match a known status word
	// are OR'd onto Status. Mermaid allows combinations such as
	// `crit, active` and `crit, milestone`.
	for idx < len(parts) {
		flag, ok := parseStatus(parts[idx])
		if !ok {
			break
		}
		task.Status |= flag
		idx++
	}

	// ID slot: anything that isn't a date, isn't a known
	// after/until prefix, and isn't a duration.
	if idx < len(parts) && !isDate(parts[idx], p.diagram.DateFormat) &&
		!strings.HasPrefix(parts[idx], "after ") &&
		!strings.HasPrefix(parts[idx], "until ") &&
		!isDuration(parts[idx]) {
		task.ID = parts[idx]
		idx++
	}

	// Start spec: explicit date OR `after id1 id2 ...` (defaults
	// to chain-from-previous-end).
	start := p.lastTaskEnd
	if idx < len(parts) {
		switch {
		case strings.HasPrefix(parts[idx], "after "):
			task.After = splitSpace(strings.TrimPrefix(parts[idx], "after "))
			start = p.maxEndOf(task.After)
			idx++
		case isDate(parts[idx], p.diagram.DateFormat):
			t, err := time.Parse(p.diagram.DateFormat, parts[idx])
			if err == nil {
				start = t
			}
			idx++
		}
	}
	task.Start = start

	// End spec: duration OR explicit date OR `until id1 id2 ...`.
	dur := 24 * time.Hour
	endSet := false
	if idx < len(parts) {
		switch {
		case strings.HasPrefix(parts[idx], "until "):
			task.Until = splitSpace(strings.TrimPrefix(parts[idx], "until "))
			task.End = p.minStartOf(task.Until)
			endSet = true
		case isDate(parts[idx], p.diagram.DateFormat):
			t, err := time.Parse(p.diagram.DateFormat, parts[idx])
			if err == nil {
				task.End = t
				endSet = true
			}
		default:
			if d, ok := parseDuration(parts[idx]); ok {
				dur = d
			}
		}
	}
	if !endSet {
		task.End = task.Start.Add(dur)
	}
	p.lastTaskEnd = task.End
	p.diagram.Tasks = append(p.diagram.Tasks, task)
	if task.ID != "" {
		p.taskByID[task.ID] = &p.diagram.Tasks[len(p.diagram.Tasks)-1]
	}
	return nil
}

// maxEndOf returns the latest End among the named tasks, falling
// back to the previous task's end when no id resolves. Used for
// `after` spec where a task waits on multiple predecessors.
func (p *parser) maxEndOf(ids []string) time.Time {
	out := p.lastTaskEnd
	any := false
	for _, id := range ids {
		ref, ok := p.taskByID[id]
		if !ok {
			continue
		}
		if !any || ref.End.After(out) {
			out = ref.End
			any = true
		}
	}
	return out
}

// minStartOf returns the earliest Start among the named tasks, used
// for `until` spec where a task ends as soon as the first listed
// successor begins. Falls back to the current accumulator end if
// none resolve.
func (p *parser) minStartOf(ids []string) time.Time {
	out := p.lastTaskEnd
	any := false
	for _, id := range ids {
		ref, ok := p.taskByID[id]
		if !ok {
			continue
		}
		if !any || ref.Start.Before(out) {
			out = ref.Start
			any = true
		}
	}
	return out
}

// splitCSV defers to the canonical SplitUnquotedCommas helper so
// quoted commas inside task names are preserved, then trims each
// item. Empty input → nil.
func splitCSV(s string) []string {
	parts := parserutil.SplitUnquotedCommas(s)
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return parts
}

// splitSpace splits on runs of whitespace, dropping empty tokens.
func splitSpace(s string) []string {
	return strings.Fields(s)
}

// splitSpaceOrCSV accepts either a comma-separated list or a
// whitespace-separated list, used for `excludes`/`includes`
// directives where Mermaid accepts both styles.
func splitSpaceOrCSV(s string) []string {
	if strings.Contains(s, ",") {
		return splitCSV(s)
	}
	return splitSpace(s)
}

func parseStatus(s string) (diagram.TaskStatus, bool) {
	switch s {
	case "done":
		return diagram.TaskStatusDone, true
	case "active":
		return diagram.TaskStatusActive, true
	case "crit":
		return diagram.TaskStatusCrit, true
	case "milestone":
		return diagram.TaskStatusMilestone, true
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

// parseDuration accepts the Mermaid-supported set of suffixes:
//
//	ms (millisecond), s (second), m (minute), h (hour),
//	d (day), w (week), M (month, 30d approximation),
//	y (year, 365d approximation).
//
// Decimal magnitudes are accepted (`1.5d`, `0.25w`).
func parseDuration(s string) (time.Duration, bool) {
	type unit struct {
		suffix string
		dur    time.Duration
	}
	// Order matters: longer suffixes first so `ms` is tested
	// before `s`, and lowercase `m` (minute) doesn't shadow `M`
	// (month) — they're distinguished by case.
	units := []unit{
		{"ms", time.Millisecond},
		{"s", time.Second},
		{"m", time.Minute},
		{"M", 30 * 24 * time.Hour},
		{"h", time.Hour},
		{"d", 24 * time.Hour},
		{"w", 7 * 24 * time.Hour},
		{"y", 365 * 24 * time.Hour},
	}
	for _, u := range units {
		if !strings.HasSuffix(s, u.suffix) {
			continue
		}
		n := strings.TrimSuffix(s, u.suffix)
		if n == "" {
			continue
		}
		v, err := strconv.ParseFloat(n, 64)
		if err != nil || v < 0 {
			continue
		}
		return time.Duration(v * float64(u.dur)), true
	}
	return 0, false
}

// mermaidToGoFormat translates a Moment.js-style date format into
// the Go reference layout. Covers the token set Mermaid documents
// for `dateFormat`: year (`YYYY`/`YY`), month (`MM`/`M`/`MMM`/
// `MMMM`), day (`DD`/`D`), 24-h (`HH`/`H`), 12-h (`hh`/`h`),
// AM/PM (`A`/`a`), minute (`mm`/`m`), second (`ss`/`s`), fractional
// second (`SSS`/`SS`/`S`), and timezone (`ZZ`/`Z`).
func mermaidToGoFormat(f string) string {
	// Replacer applies replacements in the order pairs are given,
	// so longer tokens must come first to avoid `MMMM` being eaten
	// as `MM` + `MM`.
	r := strings.NewReplacer(
		"YYYY", "2006", "YY", "06",
		"MMMM", "January", "MMM", "Jan", "MM", "01", "M", "1",
		"DD", "02", "D", "2",
		"HH", "15", "H", "15",
		"hh", "03", "h", "3",
		"mm", "04", "m", "4",
		"ss", "05", "s", "5",
		"SSS", "000", "SS", "00", "S", "0",
		"ZZ", "-0700", "Z", "-07:00",
		"A", "PM", "a", "pm",
	)
	return r.Replace(f)
}
