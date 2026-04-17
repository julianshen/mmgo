package timeline

import (
	"strings"
	"testing"
)

func TestParseHeaderRequired(t *testing.T) {
	_, err := Parse(strings.NewReader("title X"))
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestParseEmptyDiagram(t *testing.T) {
	d, err := Parse(strings.NewReader("timeline"))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(d.Sections) != 0 || len(d.Events) != 0 {
		t.Errorf("empty: %+v", d)
	}
}

func TestParseTitle(t *testing.T) {
	d, err := Parse(strings.NewReader("timeline\n    title My History"))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if d.Title != "My History" {
		t.Errorf("title = %q", d.Title)
	}
}

func TestParseTopLevelEvents(t *testing.T) {
	input := `timeline
    title Life
    1990 : Born
    2020 : Graduated : Moved`
	d, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(d.Events) != 2 {
		t.Fatalf("want 2 events, got %d", len(d.Events))
	}
	if d.Events[0].Time != "1990" || d.Events[0].Events[0] != "Born" {
		t.Errorf("event[0] = %+v", d.Events[0])
	}
	if len(d.Events[1].Events) != 2 {
		t.Errorf("event[1] should have 2 sub-events, got %d", len(d.Events[1].Events))
	}
}

func TestParseSections(t *testing.T) {
	input := `timeline
    title Milestones
    section 2020s
        2020 : Event A
        2021 : Event B
    section 2030s
        2030 : Event C`
	d, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(d.Sections) != 2 {
		t.Fatalf("want 2 sections, got %d", len(d.Sections))
	}
	if d.Sections[0].Name != "2020s" {
		t.Errorf("section[0] = %q", d.Sections[0].Name)
	}
	if len(d.Sections[0].Events) != 2 {
		t.Errorf("section[0] events = %d", len(d.Sections[0].Events))
	}
}

func TestParseComments(t *testing.T) {
	input := `timeline
    %% comment
    title X
    2020 : Event %% trailing`
	d, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if d.Title != "X" {
		t.Errorf("title = %q", d.Title)
	}
	if len(d.Events) != 1 {
		t.Errorf("want 1 event, got %d", len(d.Events))
	}
}

func TestParseInvalidEvent(t *testing.T) {
	input := `timeline
    not a valid event line`
	d, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(d.Events) != 0 {
		t.Errorf("invalid line should be ignored, got %+v", d.Events)
	}
}
