package state

import (
	"strings"
	"testing"

	"github.com/julianshen/mmgo/pkg/diagram"
)

func TestParseHeaderRequired(t *testing.T) {
	_, err := Parse(strings.NewReader("A --> B"))
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestParseEmptyDiagram(t *testing.T) {
	d, err := Parse(strings.NewReader("stateDiagram-v2"))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(d.States) != 0 || len(d.Transitions) != 0 {
		t.Errorf("empty: %+v", d)
	}
}

func TestParseV1Header(t *testing.T) {
	d, err := Parse(strings.NewReader("stateDiagram\n    A --> B"))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(d.Transitions) != 1 {
		t.Fatalf("want 1 transition, got %d", len(d.Transitions))
	}
}

func TestParseSimpleTransition(t *testing.T) {
	input := `stateDiagram-v2
    Idle --> Active
    Active --> Idle`
	d, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(d.Transitions) != 2 {
		t.Fatalf("want 2 transitions, got %d", len(d.Transitions))
	}
	if d.Transitions[0].From != "Idle" || d.Transitions[0].To != "Active" {
		t.Errorf("transition[0] = %+v", d.Transitions[0])
	}
}

func TestParseTransitionWithLabel(t *testing.T) {
	input := `stateDiagram-v2
    Idle --> Active : start`
	d, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if d.Transitions[0].Label != "start" {
		t.Errorf("label = %q, want start", d.Transitions[0].Label)
	}
}

func TestParseStartEndStates(t *testing.T) {
	input := `stateDiagram-v2
    [*] --> Idle
    Active --> [*]`
	d, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(d.Transitions) != 2 {
		t.Fatalf("want 2 transitions, got %d", len(d.Transitions))
	}
	if d.Transitions[0].From != "[*]" || d.Transitions[1].To != "[*]" {
		t.Errorf("start/end transitions: %+v", d.Transitions)
	}
}

func TestParseStateDeclaration(t *testing.T) {
	input := `stateDiagram-v2
    state "Waiting for input" as Waiting
    [*] --> Waiting`
	d, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	found := false
	for _, s := range d.States {
		if s.ID == "Waiting" && s.Label == "Waiting for input" {
			found = true
		}
	}
	if !found {
		t.Errorf("state Waiting not found with correct label: %+v", d.States)
	}
}

func TestParseCompositeState(t *testing.T) {
	input := `stateDiagram-v2
    state Active {
        Running --> Paused
        Paused --> Running
    }`
	d, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(d.States) < 1 {
		t.Fatal("expected at least 1 top-level state")
	}
	var active *diagram.StateDef
	for i := range d.States {
		if d.States[i].ID == "Active" {
			active = &d.States[i]
		}
	}
	if active == nil {
		t.Fatal("Active state not found")
	}
	if len(active.Children) < 2 {
		t.Errorf("Active should have child states, got %+v", active.Children)
	}
}

func TestParseCompositeTransitions(t *testing.T) {
	input := `stateDiagram-v2
    state Active {
        Running --> Paused
    }
    [*] --> Active`
	d, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(d.Transitions) < 2 {
		t.Errorf("want >= 2 transitions (inner + outer), got %d", len(d.Transitions))
	}
}

func TestParseSpecialStates(t *testing.T) {
	for _, tc := range []struct {
		decl string
		want diagram.StateKind
	}{
		{"state fork1 <<fork>>", diagram.StateKindFork},
		{"state join1 <<join>>", diagram.StateKindJoin},
		{"state check <<choice>>", diagram.StateKindChoice},
	} {
		t.Run(tc.want.String(), func(t *testing.T) {
			input := "stateDiagram-v2\n    " + tc.decl
			d, err := Parse(strings.NewReader(input))
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			if len(d.States) == 0 {
				t.Fatal("no states")
			}
			if d.States[0].Kind != tc.want {
				t.Errorf("kind = %v, want %v", d.States[0].Kind, tc.want)
			}
		})
	}
}

func TestParseComments(t *testing.T) {
	input := `stateDiagram-v2
    %% comment
    A --> B %% trailing`
	d, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(d.Transitions) != 1 {
		t.Errorf("want 1 transition, got %d", len(d.Transitions))
	}
}

func TestParseUnclosedComposite(t *testing.T) {
	input := `stateDiagram-v2
    state Active {
        A --> B`
	_, err := Parse(strings.NewReader(input))
	if err == nil {
		t.Fatal("expected error for unclosed composite")
	}
}

func TestParseNestedComposite(t *testing.T) {
	input := `stateDiagram-v2
    state Outer {
        state Inner {
            A --> B
        }
    }`
	d, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(d.States) != 1 || d.States[0].ID != "Outer" {
		t.Fatalf("top-level should only contain Outer: %+v", d.States)
	}
	outer := &d.States[0]
	if len(outer.Children) != 1 || outer.Children[0].ID != "Inner" {
		t.Fatalf("Outer should contain Inner: %+v", outer.Children)
	}
	inner := &outer.Children[0]
	if len(inner.Children) < 2 {
		t.Fatalf("Inner should have A and B as children: %+v", inner.Children)
	}
}

func TestParseCompositeWithAliasedChild(t *testing.T) {
	input := `stateDiagram-v2
    state Active {
        state "Long Name" as L
        L --> L2
    }`
	d, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	active := &d.States[0]
	var lChild *diagram.StateDef
	for i := range active.Children {
		if active.Children[i].ID == "L" {
			lChild = &active.Children[i]
		}
	}
	if lChild == nil {
		t.Fatal("L not found in Active.Children")
	}
	if lChild.Label != "Long Name" {
		t.Errorf("L.Label = %q, want %q", lChild.Label, "Long Name")
	}
}

func TestParseCompositeWithStateDecl(t *testing.T) {
	input := `stateDiagram-v2
    state Active {
        state Running
        Running --> Paused
    }`
	d, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	var active *diagram.StateDef
	for i := range d.States {
		if d.States[i].ID == "Active" {
			active = &d.States[i]
		}
	}
	if active == nil {
		t.Fatal("Active not found")
	}
	foundRunning := false
	for _, c := range active.Children {
		if c.ID == "Running" {
			foundRunning = true
		}
	}
	if !foundRunning {
		t.Error("Running should be a child of Active")
	}
}

func TestParseInvalidTransition(t *testing.T) {
	input := "stateDiagram-v2\n    --> B"
	d, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(d.Transitions) != 0 {
		t.Errorf("empty-from transition should be ignored, got %+v", d.Transitions)
	}
}

func TestParseStartToEnd(t *testing.T) {
	input := "stateDiagram-v2\n    [*] --> [*]"
	d, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(d.Transitions) != 1 {
		t.Fatalf("want 1 transition, got %d", len(d.Transitions))
	}
	if d.Transitions[0].From != "[*]" || d.Transitions[0].To != "[*]" {
		t.Errorf("got %+v", d.Transitions[0])
	}
	if len(d.States) != 0 {
		t.Errorf("[*] should not create real states, got %+v", d.States)
	}
}

func TestParseUnknownStateKind(t *testing.T) {
	input := "stateDiagram-v2\n    state foo <<unknown>>"
	d, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if d.States[0].Kind != diagram.StateKindNormal {
		t.Errorf("unknown kind should default to normal, got %v", d.States[0].Kind)
	}
}

func TestParseBareStateDecl(t *testing.T) {
	input := `stateDiagram-v2
    state Idle`
	d, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(d.States) != 1 || d.States[0].ID != "Idle" {
		t.Errorf("got %+v", d.States)
	}
}

func TestParseAutoRegistersStates(t *testing.T) {
	input := `stateDiagram-v2
    A --> B`
	d, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(d.States) < 2 {
		t.Errorf("should auto-register A and B, got %d states", len(d.States))
	}
}
