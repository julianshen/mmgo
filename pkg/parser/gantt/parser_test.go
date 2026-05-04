package gantt

import (
	"strings"
	"testing"
	"time"

	"github.com/julianshen/mmgo/pkg/diagram"
)

func TestParseHeaderRequired(t *testing.T) {
	_, err := Parse(strings.NewReader("title Test"))
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestParseEmptyDiagram(t *testing.T) {
	d, err := Parse(strings.NewReader("gantt"))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if d.Title != "" || len(d.Tasks) != 0 {
		t.Errorf("empty: %+v", d)
	}
}

func TestParseTitle(t *testing.T) {
	d, err := Parse(strings.NewReader("gantt\n    title My Project"))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if d.Title != "My Project" {
		t.Errorf("title = %q", d.Title)
	}
}

func TestParseDateFormat(t *testing.T) {
	d, err := Parse(strings.NewReader("gantt\n    dateFormat YYYY-MM-DD"))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if d.DateFormat != "2006-01-02" {
		t.Errorf("dateFormat = %q", d.DateFormat)
	}
}

func TestParseTaskWithDate(t *testing.T) {
	input := `gantt
    dateFormat YYYY-MM-DD
    Task One :a1, 2024-01-01, 30d`
	d, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(d.Tasks) != 1 {
		t.Fatalf("want 1 task, got %d", len(d.Tasks))
	}
	task := d.Tasks[0]
	if task.Name != "Task One" {
		t.Errorf("name = %q", task.Name)
	}
	if task.ID != "a1" {
		t.Errorf("id = %q", task.ID)
	}
	wantStart := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	if !task.Start.Equal(wantStart) {
		t.Errorf("start = %v, want %v", task.Start, wantStart)
	}
	wantEnd := wantStart.Add(30 * 24 * time.Hour)
	if !task.End.Equal(wantEnd) {
		t.Errorf("end = %v, want %v", task.End, wantEnd)
	}
}

func TestParseTaskWithStatus(t *testing.T) {
	input := `gantt
    dateFormat YYYY-MM-DD
    Done Task :done, a1, 2024-01-01, 10d
    Active Task :active, a2, 2024-01-11, 5d
    Critical :crit, a3, 2024-01-16, 3d`
	d, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if d.Tasks[0].Status != diagram.TaskStatusDone {
		t.Errorf("task[0] status = %v", d.Tasks[0].Status)
	}
	if d.Tasks[1].Status != diagram.TaskStatusActive {
		t.Errorf("task[1] status = %v", d.Tasks[1].Status)
	}
	if d.Tasks[2].Status != diagram.TaskStatusCrit {
		t.Errorf("task[2] status = %v", d.Tasks[2].Status)
	}
}

func TestParseTaskAfter(t *testing.T) {
	input := `gantt
    dateFormat YYYY-MM-DD
    Task A :a1, 2024-01-01, 10d
    Task B :a2, after a1, 5d`
	d, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(d.Tasks[1].After) != 1 || d.Tasks[1].After[0] != "a1" {
		t.Errorf("after = %v", d.Tasks[1].After)
	}
	wantStart := time.Date(2024, 1, 11, 0, 0, 0, 0, time.UTC)
	if !d.Tasks[1].Start.Equal(wantStart) {
		t.Errorf("start = %v, want %v", d.Tasks[1].Start, wantStart)
	}
}

func TestParseSections(t *testing.T) {
	input := `gantt
    dateFormat YYYY-MM-DD
    section Design
    Wireframe :a1, 2024-01-01, 5d
    section Development
    Code :a2, after a1, 10d`
	d, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(d.Sections) != 2 {
		t.Fatalf("want 2 sections, got %d", len(d.Sections))
	}
	if d.Tasks[0].Section != "Design" || d.Tasks[1].Section != "Development" {
		t.Errorf("sections: %q, %q", d.Tasks[0].Section, d.Tasks[1].Section)
	}
}

func TestParseComments(t *testing.T) {
	input := `gantt
    %% comment
    title X %% trailing`
	d, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if d.Title != "X" {
		t.Errorf("title = %q", d.Title)
	}
}

func TestParseWeekDuration(t *testing.T) {
	input := `gantt
    dateFormat YYYY-MM-DD
    Task :a1, 2024-01-01, 2w`
	d, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	wantEnd := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	if !d.Tasks[0].End.Equal(wantEnd) {
		t.Errorf("end = %v, want %v", d.Tasks[0].End, wantEnd)
	}
}

// `excludes weekends` and friends now record the value on the AST
// (PR1 promoted them out of no-op territory).
func TestParseExcludesRecorded(t *testing.T) {
	input := `gantt
    excludes weekends
    excludes 2024-01-01, 2024-12-25
    title X`
	d, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if d.Title != "X" {
		t.Errorf("title = %q", d.Title)
	}
	want := []string{"weekends", "2024-01-01", "2024-12-25"}
	if len(d.Excludes) != 3 {
		t.Fatalf("excludes = %v", d.Excludes)
	}
	for i, w := range want {
		if d.Excludes[i] != w {
			t.Errorf("excludes[%d] = %q, want %q", i, d.Excludes[i], w)
		}
	}
}

// A task may carry multiple status flags simultaneously
// (`crit, active`, `crit, milestone`, etc.).
func TestParseTaskTagList(t *testing.T) {
	input := `gantt
    dateFormat YYYY-MM-DD
    Hot fix :crit, active, hf, 2024-01-01, 2d
    Launch :milestone, ms, 2024-01-10, 0d`
	d, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if !d.Tasks[0].Status.Has(diagram.TaskStatusCrit) ||
		!d.Tasks[0].Status.Has(diagram.TaskStatusActive) {
		t.Errorf("hot fix status = %v", d.Tasks[0].Status)
	}
	if d.Tasks[0].ID != "hf" {
		t.Errorf("hot fix id = %q", d.Tasks[0].ID)
	}
	if !d.Tasks[1].Status.Has(diagram.TaskStatusMilestone) {
		t.Errorf("launch status = %v", d.Tasks[1].Status)
	}
}

// `after` accepts a space-separated list; the start date becomes
// the latest end among the named predecessors.
func TestParseTaskAfterMultiple(t *testing.T) {
	input := `gantt
    dateFormat YYYY-MM-DD
    A :a1, 2024-01-01, 3d
    B :b1, 2024-01-05, 2d
    C : after a1 b1, 1d`
	d, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if got := d.Tasks[2].After; len(got) != 2 || got[0] != "a1" || got[1] != "b1" {
		t.Errorf("after = %v", got)
	}
	wantStart := time.Date(2024, 1, 7, 0, 0, 0, 0, time.UTC) // b1's end
	if !d.Tasks[2].Start.Equal(wantStart) {
		t.Errorf("start = %v, want %v", d.Tasks[2].Start, wantStart)
	}
}

// Forward `after` references (predecessor declared later) are
// resolved by the post-pass so the start anchors correctly.
func TestParseTaskAfterForwardRef(t *testing.T) {
	input := `gantt
    dateFormat YYYY-MM-DD
    Pre :pre, after launch, 2d
    Launch :launch, 2024-01-10, 1d`
	d, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	wantStart := time.Date(2024, 1, 11, 0, 0, 0, 0, time.UTC) // launch end
	if !d.Tasks[0].Start.Equal(wantStart) {
		t.Errorf("pre start = %v, want %v", d.Tasks[0].Start, wantStart)
	}
	wantEnd := wantStart.AddDate(0, 0, 2)
	if !d.Tasks[0].End.Equal(wantEnd) {
		t.Errorf("pre end = %v, want %v", d.Tasks[0].End, wantEnd)
	}
}

// Negative-magnitude durations are rejected so a typo like `-1d`
// doesn't silently produce a backwards bar.
func TestParseDurationRejectsNegative(t *testing.T) {
	for _, in := range []string{"-1d", "-2.5h", "-100ms"} {
		if _, ok := parseDuration(in); ok {
			t.Errorf("parseDuration(%q): expected rejection", in)
		}
	}
}

// `until id1 id2 ...` ends the task at the earliest start of the
// named successors.
func TestParseTaskUntil(t *testing.T) {
	input := `gantt
    dateFormat YYYY-MM-DD
    Pre :pre, 2024-01-01, until launch
    Launch :launch, 2024-01-10, 1d`
	d, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(d.Tasks[0].Until) != 1 || d.Tasks[0].Until[0] != "launch" {
		t.Errorf("until = %v", d.Tasks[0].Until)
	}
	wantEnd := time.Date(2024, 1, 10, 0, 0, 0, 0, time.UTC)
	if !d.Tasks[0].End.Equal(wantEnd) {
		t.Errorf("pre end = %v, want %v", d.Tasks[0].End, wantEnd)
	}
}

// Expanded duration units: ms, s, m, h, d, w, M, y; decimals OK.
func TestParseDurationUnits(t *testing.T) {
	cases := map[string]time.Duration{
		"500ms": 500 * time.Millisecond,
		"30s":   30 * time.Second,
		"5m":    5 * time.Minute,
		"2h":    2 * time.Hour,
		"1.5d":  36 * time.Hour,
		"2w":    14 * 24 * time.Hour,
		"1M":    30 * 24 * time.Hour,
		"1y":    365 * 24 * time.Hour,
	}
	for in, want := range cases {
		got, ok := parseDuration(in)
		if !ok {
			t.Errorf("%s: not parsed", in)
			continue
		}
		if got != want {
			t.Errorf("%s: got %v, want %v", in, got, want)
		}
	}
}

// Axis-side directives now reach the AST instead of being silently
// dropped, so renderers can honour them.
func TestParseAxisDirectives(t *testing.T) {
	input := `gantt
    dateFormat YYYY-MM-DD
    axisFormat %Y/%m/%d
    tickInterval 1week
    weekday monday
    todayMarker stroke-width:3px
    includes 2024-01-06`
	d, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if d.AxisFormat != "%Y/%m/%d" {
		t.Errorf("axisFormat = %q", d.AxisFormat)
	}
	if d.TickInterval != "1week" {
		t.Errorf("tickInterval = %q", d.TickInterval)
	}
	if d.Weekday != "monday" {
		t.Errorf("weekday = %q", d.Weekday)
	}
	if d.TodayMarker != "stroke-width:3px" {
		t.Errorf("todayMarker = %q", d.TodayMarker)
	}
	if len(d.Includes) != 1 || d.Includes[0] != "2024-01-06" {
		t.Errorf("includes = %v", d.Includes)
	}
}

// mermaidToGoFormat covers the broader Moment.js token set.
func TestMermaidToGoFormatExtended(t *testing.T) {
	cases := map[string]string{
		"YYYY-MM-DD HH:mm:ss": "2006-01-02 15:04:05",
		"D MMM YYYY":          "2 Jan 2006",
		"DD MMMM YY":          "02 January 06",
		"hh:mm A":             "03:04 PM",
		"HH:mm:ss.SSS":        "15:04:05.000",
	}
	for in, want := range cases {
		if got := mermaidToGoFormat(in); got != want {
			t.Errorf("%s → %s, want %s", in, got, want)
		}
	}
}

// accTitle / accDescr lines populate the matching AST fields.
func TestParseAccessibility(t *testing.T) {
	d, err := Parse(strings.NewReader(`gantt
    accTitle Q1 plan
    accDescr Roadmap for Q1`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if d.AccTitle != "Q1 plan" {
		t.Errorf("accTitle = %q", d.AccTitle)
	}
	if d.AccDescr != "Roadmap for Q1" {
		t.Errorf("accDescr = %q", d.AccDescr)
	}
}

// `click TASKID href "url"` and `click TASKID call fn(args)` both
// register on the diagram's Clicks slice. Unknown task ids error.
func TestParseClickEvents(t *testing.T) {
	d, err := Parse(strings.NewReader(`gantt
    dateFormat YYYY-MM-DD
    Design :a1, 2024-01-01, 5d
    Build  :a2, 2024-01-06, 5d
    click a1 href "https://example.com/design" "design doc"
    click a2 call openTask("a2")`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(d.Clicks) != 2 {
		t.Fatalf("clicks = %v", d.Clicks)
	}
	if d.Clicks[0].URL != "https://example.com/design" || d.Clicks[0].Tooltip != "design doc" {
		t.Errorf("a1 click = %+v", d.Clicks[0])
	}
	if d.Clicks[1].Callback != `openTask("a2")` || d.Clicks[1].TaskID != "a2" {
		t.Errorf("a2 click = %+v", d.Clicks[1])
	}

	// Unknown id rejected.
	_, err = Parse(strings.NewReader(`gantt
    dateFormat YYYY-MM-DD
    A :a1, 2024-01-01, 1d
    click ghost href "x"`))
	if err == nil {
		t.Error("expected error for click with unknown task id")
	}
}

// `vert <date>` and `vert id, date, "label"` both register a
// vertical marker on the diagram's Verts slice.
func TestParseVert(t *testing.T) {
	d, err := Parse(strings.NewReader(`gantt
    dateFormat YYYY-MM-DD
    A :a1, 2024-01-01, 5d
    vert 2024-01-03
    vert v1, 2024-01-04, "Freeze"`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(d.Verts) != 2 {
		t.Fatalf("verts = %v", d.Verts)
	}
	if d.Verts[0].Date.Day() != 3 || d.Verts[0].Label != "" {
		t.Errorf("verts[0] = %+v", d.Verts[0])
	}
	if d.Verts[1].ID != "v1" || d.Verts[1].Label != "Freeze" {
		t.Errorf("verts[1] = %+v", d.Verts[1])
	}
}

// Click error paths surface meaningful messages: missing target,
// missing callback after `call`, and bad URL syntax.
func TestParseClickErrors(t *testing.T) {
	cases := []string{
		// missing target
		`gantt
    A :a1, 2024-01-01, 1d
    click a1`,
		// `call` keyword with no fn after it
		`gantt
    dateFormat YYYY-MM-DD
    A :a1, 2024-01-01, 1d
    click a1 call`,
		// empty href value
		`gantt
    dateFormat YYYY-MM-DD
    A :a1, 2024-01-01, 1d
    click a1 href ""`,
	}
	for _, src := range cases {
		if _, err := Parse(strings.NewReader(src)); err == nil {
			t.Errorf("expected error for:\n%s", src)
		}
	}
}

// Vert error paths: missing date, malformed date, missing label
// for too many positional args.
func TestParseVertErrors(t *testing.T) {
	cases := []string{
		`gantt
    dateFormat YYYY-MM-DD
    vert not-a-date`,
		`gantt
    dateFormat YYYY-MM-DD
    vert id, 2024-01-01, "label", extra`,
	}
	for _, src := range cases {
		if _, err := Parse(strings.NewReader(src)); err == nil {
			t.Errorf("expected error for:\n%s", src)
		}
	}
}
