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
	if d.Tasks[1].After != "a1" {
		t.Errorf("after = %q", d.Tasks[1].After)
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

func TestParseIgnoresExcludes(t *testing.T) {
	input := `gantt
    excludes weekends
    title X`
	d, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if d.Title != "X" {
		t.Errorf("title = %q", d.Title)
	}
}
