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
