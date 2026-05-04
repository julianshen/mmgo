package gantt

import (
	"bytes"
	"encoding/xml"
	"strings"
	"testing"
	"time"

	"github.com/julianshen/mmgo/pkg/diagram"
)

func TestRenderNilDiagram(t *testing.T) {
	_, err := Render(nil, nil)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestRenderEmptyDiagram(t *testing.T) {
	d := &diagram.GanttDiagram{}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	assertValidSVG(t, out)
}

func TestRenderWithTitle(t *testing.T) {
	d := &diagram.GanttDiagram{Title: "Project Plan"}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if !strings.Contains(string(out), ">Project Plan<") {
		t.Error("title missing")
	}
	assertValidSVG(t, out)
}

func TestRenderTasks(t *testing.T) {
	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	d := &diagram.GanttDiagram{
		Tasks: []diagram.GanttTask{
			{ID: "a1", Name: "Design", Start: start, End: start.Add(10 * 24 * time.Hour), Status: diagram.TaskStatusDone},
			{ID: "a2", Name: "Build", Start: start.Add(10 * 24 * time.Hour), End: start.Add(30 * 24 * time.Hour), Status: diagram.TaskStatusActive},
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	if !strings.Contains(raw, ">Design<") || !strings.Contains(raw, ">Build<") {
		t.Error("task names missing")
	}
	assertValidSVG(t, out)
}

func TestRenderSections(t *testing.T) {
	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	d := &diagram.GanttDiagram{
		Sections: []string{"Phase 1", "Phase 2"},
		Tasks: []diagram.GanttTask{
			{Name: "A", Section: "Phase 1", Start: start, End: start.Add(5 * 24 * time.Hour)},
			{Name: "B", Section: "Phase 2", Start: start.Add(5 * 24 * time.Hour), End: start.Add(10 * 24 * time.Hour)},
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	if !strings.Contains(raw, ">Phase 1<") || !strings.Contains(raw, ">Phase 2<") {
		t.Error("section labels missing")
	}
	assertValidSVG(t, out)
}

func TestRenderCriticalTask(t *testing.T) {
	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	d := &diagram.GanttDiagram{
		Tasks: []diagram.GanttTask{
			{Name: "Urgent", Start: start, End: start.Add(3 * 24 * time.Hour), Status: diagram.TaskStatusCrit},
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if !strings.Contains(string(out), "#e15759") {
		t.Error("critical task should use red color")
	}
	assertValidSVG(t, out)
}

func TestRenderDeterministic(t *testing.T) {
	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	d := &diagram.GanttDiagram{
		Title: "Test",
		Tasks: []diagram.GanttTask{
			{Name: "A", Start: start, End: start.Add(5 * 24 * time.Hour)},
			{Name: "B", Start: start.Add(5 * 24 * time.Hour), End: start.Add(10 * 24 * time.Hour)},
		},
	}
	first, _ := Render(d, nil)
	for i := 0; i < 10; i++ {
		next, _ := Render(d, nil)
		if string(next) != string(first) {
			t.Fatalf("iter %d: output diverges", i)
		}
	}
}

// A task whose name fits inside its bar gets the label centered in
// the bar (text-anchor=middle, white fill). Mirrors the narrow-bar
// outside-label test so a sign flip on the inside/outside decision
// fails one of them.
func TestRenderWideBarInsideLabel(t *testing.T) {
	now := time.Now()
	d := &diagram.GanttDiagram{
		Tasks: []diagram.GanttTask{
			{Name: "X", Start: now, End: now.Add(60 * 24 * time.Hour)},
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	if !strings.Contains(raw, `text-anchor="middle"`) {
		t.Error(`expected text-anchor="middle" for inside-bar label`)
	}
	if !strings.Contains(raw, "fill:white") {
		t.Error("expected white fill for inside-bar label")
	}
}

// A task whose name doesn't fit inside its bar gets the label
// rendered to the right of the bar (text-anchor=start, dark fill)
// instead of centered (white).
func TestRenderNarrowBarOutsideLabel(t *testing.T) {
	now := time.Now()
	d := &diagram.GanttDiagram{
		Tasks: []diagram.GanttTask{
			{Name: "An extremely long task name that won't fit",
				Start: now, End: now.Add(24 * time.Hour)},
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	if !strings.Contains(raw, `text-anchor="start"`) {
		t.Error("expected text-anchor=\"start\" for outside-bar label")
	}
	if !strings.Contains(raw, "fill:#333") {
		t.Error("expected dark fill for outside-bar label")
	}
}

// Section bands emit one theme-colored backdrop rect per contiguous
// named section, cycling through the palette in document order.
func TestRenderSectionBands(t *testing.T) {
	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	d := &diagram.GanttDiagram{
		Tasks: []diagram.GanttTask{
			{Name: "A", Section: "Design", Start: start, End: start.Add(3 * 24 * time.Hour)},
			{Name: "B", Section: "Design", Start: start.Add(3 * 24 * time.Hour), End: start.Add(6 * 24 * time.Hour)},
			{Name: "C", Section: "Build", Start: start.Add(6 * 24 * time.Hour), End: start.Add(9 * 24 * time.Hour)},
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	// Both palette tints (slots 0 and 1) must appear behind the two
	// named sections.
	bands := DefaultTheme().SectionBands
	for _, want := range []string{"fill:" + bands[0], "fill:" + bands[1]} {
		if !strings.Contains(raw, want) {
			t.Errorf("expected band %q in output", want)
		}
	}
}

// Leading unnamed tasks (Section="") must not consume palette[0] —
// the first *named* section should still get the first tint.
func TestRenderSectionBandsSkipsUnnamed(t *testing.T) {
	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	d := &diagram.GanttDiagram{
		Tasks: []diagram.GanttTask{
			{Name: "unsectioned", Start: start, End: start.Add(3 * 24 * time.Hour)},
			{Name: "first", Section: "S1", Start: start.Add(3 * 24 * time.Hour), End: start.Add(6 * 24 * time.Hour)},
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	bands := DefaultTheme().SectionBands
	// Palette[0] must land on the named section — not the unnamed one.
	if !strings.Contains(raw, "fill:"+bands[0]) {
		t.Errorf("expected palette[0] %q behind first named section", bands[0])
	}
	// palette[1] must NOT appear: only one named span exists.
	if strings.Contains(raw, "fill:"+bands[1]) {
		t.Errorf("unexpected palette[1] %q — only one named section", bands[1])
	}
}

// Axis labels use ISO-8601 (`2024-01-01`), not the old `Jan 01`.
func TestRenderAxisISOFormat(t *testing.T) {
	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	d := &diagram.GanttDiagram{
		Tasks: []diagram.GanttTask{
			{Name: "X", Start: start, End: start.Add(3 * 24 * time.Hour)},
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	if !strings.Contains(raw, ">2024-01-01<") {
		t.Error("expected ISO-8601 axis label 2024-01-01")
	}
	if strings.Contains(raw, ">Jan 01<") {
		t.Error("old `Jan 01` axis label should be gone")
	}
}

// The new muted palette: done is gray, active is a lighter accent,
// neither falls back to the `none` color.
func TestRenderDoneActiveColors(t *testing.T) {
	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	d := &diagram.GanttDiagram{
		Tasks: []diagram.GanttTask{
			{Name: "D", Start: start, End: start.Add(3 * 24 * time.Hour), Status: diagram.TaskStatusDone},
			{Name: "A", Start: start.Add(3 * 24 * time.Hour), End: start.Add(6 * 24 * time.Hour), Status: diagram.TaskStatusActive},
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	th := DefaultTheme()
	if !strings.Contains(raw, "fill:"+th.TaskColors[diagram.TaskStatusDone]) {
		t.Errorf("done task missing color %q", th.TaskColors[diagram.TaskStatusDone])
	}
	if !strings.Contains(raw, "fill:"+th.TaskColors[diagram.TaskStatusActive]) {
		t.Errorf("active task missing color %q", th.TaskColors[diagram.TaskStatusActive])
	}
}

// Full-height vertical grid lines at every axis tick. Previously a
// tick was a ±3px mark; now it spans the full body height.
func TestRenderVerticalGrid(t *testing.T) {
	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	d := &diagram.GanttDiagram{
		Tasks: []diagram.GanttTask{
			{Name: "A", Start: start, End: start.Add(3 * 24 * time.Hour)},
			{Name: "B", Start: start.Add(3 * 24 * time.Hour), End: start.Add(6 * 24 * time.Hour)},
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	grid := DefaultTheme().GridStroke
	if !strings.Contains(raw, "stroke:"+grid) {
		t.Errorf("expected grid lines with stroke %q", grid)
	}
}

// `axisFormat` overrides the default ISO label, using d3-strftime
// translated to a Go layout.
func TestRenderAxisFormat(t *testing.T) {
	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	d := &diagram.GanttDiagram{
		AxisFormat: "%b %d",
		Tasks: []diagram.GanttTask{
			{Name: "A", Start: start, End: start.Add(2 * 24 * time.Hour)},
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	if !strings.Contains(raw, ">Jan 01<") {
		t.Errorf("expected axis label `Jan 01`, got:\n%s", raw)
	}
	if strings.Contains(raw, ">2024-01-01<") {
		t.Errorf("default ISO label should be replaced by axisFormat")
	}
}

// `tickInterval 1week` advances ticks by 7 calendar days from
// minDate, regardless of the auto-interval that the chart span
// would otherwise pick.
func TestRenderTickIntervalWeek(t *testing.T) {
	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC) // Mon
	d := &diagram.GanttDiagram{
		TickInterval: "1week",
		Tasks: []diagram.GanttTask{
			{Name: "A", Start: start, End: start.Add(21 * 24 * time.Hour)},
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	for _, want := range []string{">2024-01-01<", ">2024-01-08<", ">2024-01-15<", ">2024-01-22<"} {
		if !strings.Contains(raw, want) {
			t.Errorf("expected weekly tick %q, missing from output", want)
		}
	}
}

// Milestone tasks render as a diamond (polygon) at the start
// position rather than a rectangle, with the label outside.
func TestRenderMilestone(t *testing.T) {
	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	d := &diagram.GanttDiagram{
		Tasks: []diagram.GanttTask{
			{Name: "Anchor", Start: start, End: start.Add(5 * 24 * time.Hour)},
			{Name: "Launch", Start: start.Add(5 * 24 * time.Hour), End: start.Add(5 * 24 * time.Hour),
				Status: diagram.TaskStatusMilestone},
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	if !strings.Contains(raw, "<polygon") {
		t.Errorf("expected milestone polygon glyph, got:\n%s", raw)
	}
	if !strings.Contains(raw, ">Launch<") {
		t.Errorf("milestone label missing")
	}
}

// A `crit, milestone` task gets the crit stroke applied to the
// diamond glyph, not just the rectangle path.
func TestRenderCritMilestoneStroke(t *testing.T) {
	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	d := &diagram.GanttDiagram{
		Tasks: []diagram.GanttTask{
			{Name: "Anchor", Start: start, End: start.Add(2 * 24 * time.Hour)},
			{Name: "GA", Start: start.Add(2 * 24 * time.Hour), End: start.Add(2 * 24 * time.Hour),
				Status: diagram.TaskStatusCrit | diagram.TaskStatusMilestone},
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	if !strings.Contains(raw, "<polygon") {
		t.Fatalf("expected milestone polygon")
	}
	if !strings.Contains(raw, "stroke:"+DefaultTheme().CritStroke) {
		t.Errorf("expected crit stroke on milestone polygon, got:\n%s", raw)
	}
}

// Crit tasks get a stroke outline on top of their fill.
func TestRenderCritStroke(t *testing.T) {
	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	d := &diagram.GanttDiagram{
		Tasks: []diagram.GanttTask{
			{Name: "Hot", Start: start, End: start.Add(2 * 24 * time.Hour),
				Status: diagram.TaskStatusCrit},
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	if !strings.Contains(raw, "stroke:"+DefaultTheme().CritStroke) {
		t.Errorf("expected crit stroke %q in:\n%s", DefaultTheme().CritStroke, raw)
	}
}

// `todayMarker` draws a vertical rule when today's date sits in
// the chart range; off / out-of-range diagrams omit it.
func TestRenderTodayMarker(t *testing.T) {
	now := time.Now().UTC().Truncate(24 * time.Hour)
	in := &diagram.GanttDiagram{
		TodayMarker: "stroke-width:2px",
		Tasks: []diagram.GanttTask{
			{Name: "Span", Start: now.AddDate(0, 0, -3), End: now.AddDate(0, 0, 3)},
		},
	}
	rawIn, err := Render(in, nil)
	if err != nil {
		t.Fatalf("Render in-range: %v", err)
	}
	if !strings.Contains(string(rawIn), "stroke-width:2px") {
		t.Errorf("expected today marker style in:\n%s", rawIn)
	}

	off := &diagram.GanttDiagram{
		TodayMarker: "off",
		Tasks: []diagram.GanttTask{
			{Name: "Span", Start: now.AddDate(0, 0, -3), End: now.AddDate(0, 0, 3)},
		},
	}
	rawOff, err := Render(off, nil)
	if err != nil {
		t.Fatalf("Render off: %v", err)
	}
	if strings.Contains(string(rawOff), "stroke-dasharray:4 2") {
		t.Errorf("today marker should be suppressed when set to off")
	}
}

// Axis-format helper round-trips the documented Mermaid token set.
func TestD3StrftimeToGoLayout(t *testing.T) {
	cases := map[string]string{
		"%Y-%m-%d":  "2006-01-02",
		"%b %d, %Y": "Jan 02, 2006",
		"%H:%M:%S":  "15:04:05",
		"%A %B":     "Monday January",
		"%I:%M %p":  "03:04 PM",
		"%j":        "002",
		"100%%":     "100%",
		"%Q":        "%Q", // unknown token survives verbatim
	}
	for in, want := range cases {
		if got := d3StrftimeToGoLayout(in); got != want {
			t.Errorf("%s → %q, want %q", in, got, want)
		}
	}
}

// `tickInterval 6hour` exercises the duration-based branch of
// tickStep.advance (millisecond/second/minute/hour) rather than
// the calendar AddDate branch.
func TestRenderTickIntervalHourly(t *testing.T) {
	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	d := &diagram.GanttDiagram{
		AxisFormat:   "%H:%M",
		TickInterval: "6hour",
		Tasks: []diagram.GanttTask{
			{Name: "Run", Start: start, End: start.Add(24 * time.Hour)},
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	for _, want := range []string{">00:00<", ">06:00<", ">12:00<", ">18:00<"} {
		if !strings.Contains(raw, want) {
			t.Errorf("expected hourly tick %q in output", want)
		}
	}
}

// Tick-interval parser covers every documented unit and rejects
// malformed input.
func TestParseTickInterval(t *testing.T) {
	for _, ok := range []string{"1day", "2 weeks", "15minute", "1month", "1year"} {
		if _, valid := parseTickInterval(ok); !valid {
			t.Errorf("%q should parse", ok)
		}
	}
	for _, bad := range []string{"", "1", "day", "0day", "-1d", "1fortnight"} {
		if _, valid := parseTickInterval(bad); valid {
			t.Errorf("%q should fail to parse", bad)
		}
	}
}

func assertValidSVG(t *testing.T, svgBytes []byte) {
	t.Helper()
	body := svgBytes
	if i := bytes.Index(body, []byte("<svg")); i >= 0 {
		body = body[i:]
	}
	var doc struct {
		XMLName xml.Name `xml:"svg"`
		ViewBox string   `xml:"viewBox,attr"`
	}
	if err := xml.Unmarshal(body, &doc); err != nil {
		t.Fatalf("invalid SVG: %v\n%s", err, svgBytes)
	}
	if doc.ViewBox == "" {
		t.Error("viewBox missing")
	}
}
