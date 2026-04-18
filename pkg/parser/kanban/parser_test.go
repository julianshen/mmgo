package kanban

import (
	"strings"
	"testing"

	"github.com/julianshen/mmgo/pkg/diagram"
)

func TestParseHeaderRequired(t *testing.T) {
	if _, err := Parse(strings.NewReader("Todo\n")); err == nil {
		t.Fatal("expected error for missing header")
	}
}

func TestParseEmpty(t *testing.T) {
	if _, err := Parse(strings.NewReader("")); err == nil {
		t.Fatal("expected error for empty input")
	}
}

func TestParseBadHeader(t *testing.T) {
	if _, err := Parse(strings.NewReader("pie\n")); err == nil {
		t.Fatal("expected error for non-kanban header")
	}
}

func TestParseSingleSection(t *testing.T) {
	d, err := Parse(strings.NewReader("kanban\nTodo\n"))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(d.Sections) != 1 || d.Sections[0].Title != "Todo" {
		t.Errorf("sections = %+v", d.Sections)
	}
}

func TestParseSectionWithBrackets(t *testing.T) {
	d, err := Parse(strings.NewReader("kanban\nid1[Todo]\n"))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	s := d.Sections[0]
	if s.ID != "id1" || s.Title != "Todo" {
		t.Errorf("section = %+v", s)
	}
}

// The first body line after the header sets the section indent level.
// A diagram with only one indented line is a chart with a single empty
// section, not an orphan-task error.
func TestParseFirstBodyLineIsSection(t *testing.T) {
	d, err := Parse(strings.NewReader("kanban\n    [Solo]\n"))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(d.Sections) != 1 || d.Sections[0].Title != "Solo" {
		t.Errorf("sections = %+v", d.Sections)
	}
}

func TestParseSectionWithTasks(t *testing.T) {
	input := `kanban
Todo
    [Write docs]
    [Write tests]
In progress
    [Implement renderer]
`
	d, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(d.Sections) != 2 {
		t.Fatalf("sections = %d", len(d.Sections))
	}
	if len(d.Sections[0].Tasks) != 2 {
		t.Errorf("Todo tasks = %d", len(d.Sections[0].Tasks))
	}
	if d.Sections[0].Tasks[0].Text != "Write docs" {
		t.Errorf("first task text = %q", d.Sections[0].Tasks[0].Text)
	}
	if d.Sections[1].Tasks[0].Text != "Implement renderer" {
		t.Errorf("second section first task = %q", d.Sections[1].Tasks[0].Text)
	}
}

func TestParseTaskWithID(t *testing.T) {
	input := `kanban
Todo
    t1[Write docs]
    t2[Write tests]
`
	d, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	tasks := d.Sections[0].Tasks
	if tasks[0].ID != "t1" || tasks[1].ID != "t2" {
		t.Errorf("task IDs = %q, %q", tasks[0].ID, tasks[1].ID)
	}
}

func TestParseAutoTaskID(t *testing.T) {
	// Tasks without explicit IDs get auto-generated t1, t2, ...
	input := `kanban
Todo
    [A]
    [B]
`
	d, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	tasks := d.Sections[0].Tasks
	if tasks[0].ID == "" || tasks[1].ID == "" {
		t.Errorf("auto IDs not assigned: %+v", tasks)
	}
	if tasks[0].ID == tasks[1].ID {
		t.Errorf("auto IDs collided: %q == %q", tasks[0].ID, tasks[1].ID)
	}
}

func TestParseTaskWithMetadata(t *testing.T) {
	input := `kanban
Todo
    [Design]@{ priority: 'High', assigned: 'alice' }
`
	d, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	task := d.Sections[0].Tasks[0]
	if task.Metadata["priority"] != "High" {
		t.Errorf("priority = %q", task.Metadata["priority"])
	}
	if task.Metadata["assigned"] != "alice" {
		t.Errorf("assigned = %q", task.Metadata["assigned"])
	}
}

func TestParseMetadataWithCommaInQuotes(t *testing.T) {
	input := `kanban
Todo
    [T]@{ note: 'hello, world', priority: 'Low' }
`
	d, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	task := d.Sections[0].Tasks[0]
	if task.Metadata["note"] != "hello, world" {
		t.Errorf("note = %q", task.Metadata["note"])
	}
	if task.Metadata["priority"] != "Low" {
		t.Errorf("priority = %q", task.Metadata["priority"])
	}
}

func TestParseMetadataUnquoted(t *testing.T) {
	input := `kanban
Todo
    [T]@{ ticket: MC-123, priority: High }
`
	d, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	task := d.Sections[0].Tasks[0]
	if task.Metadata["ticket"] != "MC-123" {
		t.Errorf("ticket = %q", task.Metadata["ticket"])
	}
}

func TestParseBareTextTask(t *testing.T) {
	input := `kanban
Todo
    Just text
`
	d, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if d.Sections[0].Tasks[0].Text != "Just text" {
		t.Errorf("text = %q", d.Sections[0].Tasks[0].Text)
	}
}

func TestParseBadBrackets(t *testing.T) {
	cases := []string{
		"kanban\nTodo\n    [unterminated\n",
		"kanban\nTodo\n    []\n",        // empty brackets
		"kanban\nTodo\n    [x] extra\n", // trailing text after ']'
	}
	for _, c := range cases {
		if _, err := Parse(strings.NewReader(c)); err == nil {
			t.Errorf("expected error for:\n%s", c)
		}
	}
}

func TestParseBadMetadata(t *testing.T) {
	cases := []string{
		"kanban\nTodo\n    [x]@{ priority \n", // unterminated @{
		"kanban\nTodo\n    [x]@{ bad }\n",     // no colon
		"kanban\nTodo\n    [x]@{ :v }\n",      // empty key
	}
	for _, c := range cases {
		if _, err := Parse(strings.NewReader(c)); err == nil {
			t.Errorf("expected error for:\n%s", c)
		}
	}
}

func TestParseFullExample(t *testing.T) {
	input := `kanban
  Todo
    [Create tickets]
    id2[Triage]@{ priority: 'High' }
  id4[In progress]
    [Design]
  Done
    [Write tests]@{ assigned: 'alice' }
`
	d, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(d.Sections) != 3 {
		t.Fatalf("sections = %d", len(d.Sections))
	}
	if d.Sections[1].ID != "id4" || d.Sections[1].Title != "In progress" {
		t.Errorf("section 1 = %+v", d.Sections[1])
	}
}

// Mermaid permits per-section metadata (icons, colors). The AST
// preserves it on KanbanSection.Metadata even though the renderer
// doesn't yet use it — silently dropping would lose fidelity.
func TestParseSectionMetadataPreserved(t *testing.T) {
	input := `kanban
Todo@{ icon: 'clock', color: '#f00' }
    [x]
`
	d, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	s := d.Sections[0]
	if s.Metadata["icon"] != "clock" {
		t.Errorf("icon = %q", s.Metadata["icon"])
	}
	if s.Metadata["color"] != "#f00" {
		t.Errorf("color = %q", s.Metadata["color"])
	}
}

func TestParseCommentsIgnored(t *testing.T) {
	input := `kanban
%% comment
Todo
    [X] %% inline
`
	d, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(d.Sections) != 1 || len(d.Sections[0].Tasks) != 1 {
		t.Errorf("parsed = %+v", d)
	}
}

func TestKanbanDiagramType(t *testing.T) {
	var d diagram.Diagram = &diagram.KanbanDiagram{}
	if d.Type() != diagram.Kanban {
		t.Errorf("Type() = %v", d.Type())
	}
}
