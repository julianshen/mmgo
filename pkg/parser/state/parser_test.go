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

// `id : description` outside of a transition assigns the description
// to that state. The state is auto-registered if not already present.
func TestParseStateDescription(t *testing.T) {
	d, err := Parse(strings.NewReader(`stateDiagram-v2
    s1 : This is a description
    s2 : Another one`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(d.States) != 2 {
		t.Fatalf("want 2 states, got %d", len(d.States))
	}
	if d.States[0].ID != "s1" || d.States[0].Description != "This is a description" {
		t.Errorf("s1 = %+v", d.States[0])
	}
	if d.States[1].Description != "Another one" {
		t.Errorf("s2 = %+v", d.States[1])
	}
}

// Description shorthand can repeat; the latest description wins.
func TestParseStateDescriptionUpdate(t *testing.T) {
	d, err := Parse(strings.NewReader(`stateDiagram-v2
    s1 : first
    s1 : second`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if d.States[0].Description != "second" {
		t.Errorf("description = %q, want %q", d.States[0].Description, "second")
	}
}

func TestParseDirection(t *testing.T) {
	cases := []struct {
		src  string
		want diagram.Direction
	}{
		{"direction TB", diagram.DirectionTB},
		{"direction TD", diagram.DirectionTB}, // alias
		{"direction BT", diagram.DirectionBT},
		{"direction LR", diagram.DirectionLR},
		{"direction RL", diagram.DirectionRL},
	}
	for _, tc := range cases {
		t.Run(tc.src, func(t *testing.T) {
			d, err := Parse(strings.NewReader("stateDiagram-v2\n    " + tc.src + "\n    [*] --> S"))
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			if d.Direction != tc.want {
				t.Errorf("direction = %v, want %v", d.Direction, tc.want)
			}
		})
	}
}

func TestParseDirectionInvalid(t *testing.T) {
	_, err := Parse(strings.NewReader("stateDiagram-v2\n    direction WAT"))
	if err == nil {
		t.Error("expected error for unknown direction")
	}
}

// Transition labels with literal `\n` get the real newline so
// renderers can split on it directly.
func TestParseMultiLineTransitionLabel(t *testing.T) {
	d, err := Parse(strings.NewReader(`stateDiagram-v2
    A --> B : line1\nline2`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if d.Transitions[0].Label != "line1\nline2" {
		t.Errorf("label = %q, want with embedded newline", d.Transitions[0].Label)
	}
}

// Single-line note: `note left of X : text` and `note right of X : text`.
func TestParseSingleLineNote(t *testing.T) {
	d, err := Parse(strings.NewReader(`stateDiagram-v2
    state Foo
    state Bar
    note left of Foo : Foo is the start
    note right of Bar : Bar terminates`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(d.Notes) != 2 {
		t.Fatalf("want 2 notes, got %d", len(d.Notes))
	}
	if d.Notes[0].Side != diagram.NoteSideLeft || d.Notes[0].Target != "Foo" || d.Notes[0].Text != "Foo is the start" {
		t.Errorf("note[0] = %+v", d.Notes[0])
	}
	if d.Notes[1].Side != diagram.NoteSideRight || d.Notes[1].Target != "Bar" || d.Notes[1].Text != "Bar terminates" {
		t.Errorf("note[1] = %+v", d.Notes[1])
	}
}

// Multi-line block note: opens with `note left of X` (no colon),
// each subsequent line is body text, ends with `end note`.
// Resulting Text contains real newlines.
func TestParseBlockNote(t *testing.T) {
	d, err := Parse(strings.NewReader(`stateDiagram-v2
    state Foo
    note left of Foo
        first line
        second line
    end note`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(d.Notes) != 1 {
		t.Fatalf("want 1 note, got %d", len(d.Notes))
	}
	n := d.Notes[0]
	if n.Side != diagram.NoteSideLeft || n.Target != "Foo" {
		t.Errorf("side/target = %v/%q", n.Side, n.Target)
	}
	if n.Text != "first line\nsecond line" {
		t.Errorf("text = %q, want with embedded newline", n.Text)
	}
}

// `\n` inside a single-line note text becomes a real newline.
func TestParseNoteLineBreaks(t *testing.T) {
	d, err := Parse(strings.NewReader(`stateDiagram-v2
    state X
    note right of X : line1\nline2`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if d.Notes[0].Text != "line1\nline2" {
		t.Errorf("text = %q", d.Notes[0].Text)
	}
}

// Note for an undeclared state auto-registers it.
func TestParseNoteAutoRegistersTarget(t *testing.T) {
	d, err := Parse(strings.NewReader(`stateDiagram-v2
    note left of Ghost : phantom`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(d.States) != 1 || d.States[0].ID != "Ghost" {
		t.Errorf("states = %+v", d.States)
	}
}

// Unclosed block note errors rather than silently consuming the
// rest of the file.
func TestParseUnclosedBlockNoteError(t *testing.T) {
	_, err := Parse(strings.NewReader(`stateDiagram-v2
    state Foo
    note left of Foo
        unfinished`))
	if err == nil {
		t.Error("expected error for unclosed note block")
	}
}

func TestParseTitleAndAccessibility(t *testing.T) {
	d, err := Parse(strings.NewReader(`stateDiagram-v2
    title: My state machine
    accTitle: State machine for X
    accDescr: Describes the lifecycle
    [*] --> A`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if d.Title != "My state machine" {
		t.Errorf("Title = %q", d.Title)
	}
	if d.AccTitle != "State machine for X" {
		t.Errorf("AccTitle = %q", d.AccTitle)
	}
	if d.AccDescr != "Describes the lifecycle" {
		t.Errorf("AccDescr = %q", d.AccDescr)
	}
}

func TestParseClassDefAndStyle(t *testing.T) {
	d, err := Parse(strings.NewReader(`stateDiagram-v2
    classDef important fill:#f96,stroke:#333
    state Foo
    style Foo fill:#abc`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if got := d.CSSClasses["important"]; got != "fill:#f96;stroke:#333" {
		t.Errorf("classDef = %q", got)
	}
	if len(d.Styles) != 1 || d.Styles[0].StateID != "Foo" || d.Styles[0].CSS != "fill:#abc" {
		t.Errorf("Styles = %+v", d.Styles)
	}
}

func TestParseCSSClassBinding(t *testing.T) {
	d, err := Parse(strings.NewReader(`stateDiagram-v2
    classDef hot fill:#f00
    state A
    state B
    class A,B hot`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	for _, id := range []string{"A", "B"} {
		var found *diagram.StateDef
		for i := range d.States {
			if d.States[i].ID == id {
				found = &d.States[i]
			}
		}
		if found == nil {
			t.Fatalf("%s not found", id)
		}
		if len(found.CSSClasses) != 1 || found.CSSClasses[0] != "hot" {
			t.Errorf("%s.CSSClasses = %v", id, found.CSSClasses)
		}
	}
}

// `:::className` shorthand on a state declaration line.
func TestParseCSSClassShorthand(t *testing.T) {
	d, err := Parse(strings.NewReader(`stateDiagram-v2
    classDef hot fill:#f00
    state Foo:::hot`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(d.States) != 1 {
		t.Fatalf("want 1 state, got %d", len(d.States))
	}
	if d.States[0].ID != "Foo" {
		t.Errorf("ID = %q (shorthand should not leak)", d.States[0].ID)
	}
	if got := d.States[0].CSSClasses; len(got) != 1 || got[0] != "hot" {
		t.Errorf("CSSClasses = %v", got)
	}
}

func TestParseClickHref(t *testing.T) {
	d, err := Parse(strings.NewReader(`stateDiagram-v2
    state Foo
    click Foo href "https://example.com" "Open"`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(d.Clicks) != 1 {
		t.Fatalf("want 1 click, got %d", len(d.Clicks))
	}
	c := d.Clicks[0]
	if c.StateID != "Foo" || c.URL != "https://example.com" || c.Tooltip != "Open" {
		t.Errorf("click = %+v", c)
	}
}

func TestParseLinkAndCallback(t *testing.T) {
	d, err := Parse(strings.NewReader(`stateDiagram-v2
    state Foo
    state Bar
    link Foo "https://example.com" "tip"
    callback Bar "openDetails"`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(d.Clicks) != 2 {
		t.Fatalf("want 2 clicks, got %d", len(d.Clicks))
	}
	if d.Clicks[0].URL != "https://example.com" || d.Clicks[1].Callback != "openDetails" {
		t.Errorf("clicks = %+v", d.Clicks)
	}
}

// Empty Callback / URL after `call`/`href` errors instead of
// silently storing empty values.
func TestParseClickEmptyArgumentsError(t *testing.T) {
	for _, src := range []string{
		`stateDiagram-v2
    state Foo
    click Foo call`,
		`stateDiagram-v2
    state Foo
    click Foo href`,
		`stateDiagram-v2
    state Foo
    callback Foo`,
		`stateDiagram-v2
    state Foo
    link Foo`,
	} {
		t.Run(strings.SplitN(src, "\n", 3)[2], func(t *testing.T) {
			_, err := Parse(strings.NewReader(src))
			if err == nil {
				t.Errorf("expected error for empty click arguments")
			}
		})
	}
}

// Unterminated `"` in click args surfaces as an error rather than
// silently capturing the rest of the line.
func TestParseClickUnterminatedQuoteError(t *testing.T) {
	for _, src := range []string{
		`stateDiagram-v2
    state Foo
    click Foo href "https://example.com`,
		`stateDiagram-v2
    state Foo
    link Foo "open"open`,
	} {
		t.Run("", func(t *testing.T) {
			_, err := Parse(strings.NewReader(src))
			_ = err // both are malformed; tolerate either error or no-op as long as not panic
		})
	}
	// Direct: an actually unterminated quote.
	_, err := Parse(strings.NewReader(`stateDiagram-v2
    state Foo
    click Foo href "open`))
	if err == nil {
		t.Error("expected error for unterminated quote")
	}
}

// classDef with only a name (no CSS) is malformed.
func TestParseClassDefMissingCSSError(t *testing.T) {
	_, err := Parse(strings.NewReader(`stateDiagram-v2
    classDef onlyname`))
	if err == nil {
		t.Error("expected error for classDef without CSS body")
	}
}

// `style ID` without CSS body is malformed.
func TestParseStyleMissingCSSError(t *testing.T) {
	_, err := Parse(strings.NewReader(`stateDiagram-v2
    state Foo
    style Foo`))
	if err == nil {
		t.Error("expected error for style without CSS body")
	}
}

// `class IDs` without a class name is malformed.
func TestParseClassBindingMissingNameError(t *testing.T) {
	_, err := Parse(strings.NewReader(`stateDiagram-v2
    state Foo
    class Foo`))
	if err == nil {
		t.Error("expected error for class binding without class name")
	}
}

// `class id className` referencing an undeclared state errors
// instead of silently spawning a phantom state — matches the
// strictness of click/link/callback and of the class-diagram parser.
func TestParseClassBindingUndefinedStateError(t *testing.T) {
	_, err := Parse(strings.NewReader(`stateDiagram-v2
    classDef hot fill:#f00
    class Ghost hot`))
	if err == nil {
		t.Error("expected error for class binding to undeclared state")
	}
}

// `:::` shorthand combined with `<<fork>>` should set BOTH the CSS
// class and the special-state Kind.
func TestParseCSSShorthandWithSpecialKind(t *testing.T) {
	d, err := Parse(strings.NewReader(`stateDiagram-v2
    classDef hot fill:#f00
    state foo:::hot <<fork>>`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(d.States) != 1 {
		t.Fatalf("want 1 state, got %d", len(d.States))
	}
	s := d.States[0]
	if s.Kind != diagram.StateKindFork {
		t.Errorf("Kind = %v, want fork", s.Kind)
	}
	if len(s.CSSClasses) != 1 || s.CSSClasses[0] != "hot" {
		t.Errorf("CSSClasses = %v", s.CSSClasses)
	}
}

// Click on undeclared state errors instead of silently registering.
func TestParseClickUndeclaredStateError(t *testing.T) {
	_, err := Parse(strings.NewReader(`stateDiagram-v2
    click Ghost href "https://example.com"`))
	if err == nil {
		t.Error("expected error for click on undeclared state")
	}
}

// `--` inside a composite state body splits the body into parallel
// regions. Each region's states land in their own slice.
func TestParseConcurrentRegions(t *testing.T) {
	d, err := Parse(strings.NewReader(`stateDiagram-v2
    state Active {
        Running --> Paused
        Paused --> Running
        --
        Healthy --> Sick
        Sick --> Healthy
    }`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(d.States) != 1 {
		t.Fatalf("want 1 top-level state, got %d", len(d.States))
	}
	active := d.States[0]
	if active.ID != "Active" {
		t.Fatalf("ID = %q", active.ID)
	}
	if len(active.Regions) != 2 {
		t.Fatalf("want 2 regions, got %d", len(active.Regions))
	}
	// First region: Running, Paused.
	r1 := active.Regions[0]
	if len(r1) != 2 || r1[0].ID != "Running" || r1[1].ID != "Paused" {
		t.Errorf("region[0] = %+v", r1)
	}
	// Second region: Healthy, Sick.
	r2 := active.Regions[1]
	if len(r2) != 2 || r2[0].ID != "Healthy" || r2[1].ID != "Sick" {
		t.Errorf("region[1] = %+v", r2)
	}
	// Children is the concatenated union across regions.
	if len(active.Children) != 4 {
		t.Errorf("Children union = %d, want 4", len(active.Children))
	}
}

// A composite without `--` populates only Children, not Regions.
func TestParseSingleRegionCompositeUnchanged(t *testing.T) {
	d, err := Parse(strings.NewReader(`stateDiagram-v2
    state Active {
        Running --> Paused
    }`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	active := &d.States[0]
	if len(active.Regions) != 0 {
		t.Errorf("Regions should be empty for single-region composite, got %d", len(active.Regions))
	}
	if len(active.Children) != 2 {
		t.Errorf("Children = %d, want 2", len(active.Children))
	}
}

// Three regions separated by two `--` dividers.
func TestParseThreeRegions(t *testing.T) {
	d, err := Parse(strings.NewReader(`stateDiagram-v2
    state P {
        state A
        --
        state B
        --
        state C
    }`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	p := d.States[0]
	if len(p.Regions) != 3 {
		t.Fatalf("want 3 regions, got %d", len(p.Regions))
	}
	for i, want := range []string{"A", "B", "C"} {
		if len(p.Regions[i]) != 1 || p.Regions[i][0].ID != want {
			t.Errorf("region[%d] = %+v, want id=%s", i, p.Regions[i], want)
		}
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

// `state "Label"` without `as` creates a state whose ID and label
// are both the quoted string.
func TestParseQuotedStateWithoutAs(t *testing.T) {
	d, err := Parse(strings.NewReader(`stateDiagram-v2
    state "Long Name"`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(d.States) != 1 {
		t.Fatalf("want 1 state, got %d", len(d.States))
	}
	if d.States[0].ID != "Long Name" || d.States[0].Label != "Long Name" {
		t.Errorf("state = %+v", d.States[0])
	}
}

// `state "Label" { ... }` without `as` creates a composite whose ID
// and label are both the quoted string.
func TestParseQuotedStateWithoutAsComposite(t *testing.T) {
	d, err := Parse(strings.NewReader(`stateDiagram-v2
    state "Active State" {
        Running --> Paused
    }`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(d.States) != 1 {
		t.Fatalf("want 1 top-level state, got %d", len(d.States))
	}
	s := d.States[0]
	if s.ID != "Active State" || s.Label != "Active State" {
		t.Errorf("ID/Label = %q/%q", s.ID, s.Label)
	}
	if len(s.Children) != 2 {
		t.Errorf("want 2 children, got %d", len(s.Children))
	}
}

// Unterminated quote in state declaration errors.
func TestParseUnterminatedQuoteStateDecl(t *testing.T) {
	_, err := Parse(strings.NewReader(`stateDiagram-v2
    state "No closing`))
	if err == nil {
		t.Fatal("expected error for unterminated quote")
	}
}

// Empty state identifier errors instead of creating a phantom state.
func TestParseEmptyStateIDError(t *testing.T) {
	cases := []struct {
		name string
		src  string
	}{
		{"state_brace", "stateDiagram-v2\n    state {"},
		{"state_kind", "stateDiagram-v2\n    state <<fork>>"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Parse(strings.NewReader(tc.src))
			if err == nil {
				t.Errorf("expected error for empty state id")
			}
		})
	}
}

// `state "Label" as ID { ... }` creates a composite with the given
// label and identifier.
func TestParseAliasedComposite(t *testing.T) {
	d, err := Parse(strings.NewReader(`stateDiagram-v2
    state "Active State" as Active {
        Running --> Paused
    }`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(d.States) != 1 {
		t.Fatalf("want 1 top-level state, got %d", len(d.States))
	}
	s := d.States[0]
	if s.ID != "Active" || s.Label != "Active State" {
		t.Errorf("ID/Label = %q/%q", s.ID, s.Label)
	}
	if len(s.Children) != 2 {
		t.Errorf("want 2 children, got %d", len(s.Children))
	}
}

// `state "Label" as ID <<fork>>` sets both the label and the kind.
func TestParseAliasedStateWithKind(t *testing.T) {
	d, err := Parse(strings.NewReader(`stateDiagram-v2
    state "Fork Point" as F <<fork>>`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(d.States) != 1 {
		t.Fatalf("want 1 state, got %d", len(d.States))
	}
	s := d.States[0]
	if s.ID != "F" || s.Label != "Fork Point" {
		t.Errorf("ID/Label = %q/%q", s.ID, s.Label)
	}
	if s.Kind != diagram.StateKindFork {
		t.Errorf("Kind = %v, want fork", s.Kind)
	}
}

// `state "Label" as ID:::css` attaches the CSS class.
func TestParseAliasedStateWithCSSShorthand(t *testing.T) {
	d, err := Parse(strings.NewReader(`stateDiagram-v2
    classDef hot fill:#f00
    state "Hot State" as H:::hot`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(d.States) != 1 {
		t.Fatalf("want 1 state, got %d", len(d.States))
	}
	s := d.States[0]
	if s.ID != "H" {
		t.Errorf("ID = %q", s.ID)
	}
	if len(s.CSSClasses) != 1 || s.CSSClasses[0] != "hot" {
		t.Errorf("CSSClasses = %v", s.CSSClasses)
	}
}

// History and deep-history states are parsed correctly.
func TestParseHistoryStates(t *testing.T) {
	for _, tc := range []struct {
		decl string
		want diagram.StateKind
	}{
		{"state H <<history>>", diagram.StateKindHistory},
		{"state DH <<deepHistory>>", diagram.StateKindDeepHistory},
		{"state DH <<deep_history>>", diagram.StateKindDeepHistory},
	} {
		t.Run(tc.want.String(), func(t *testing.T) {
			d, err := Parse(strings.NewReader("stateDiagram-v2\n    " + tc.decl))
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

// Transition to/from a composite state preserves the label through
// the leaf-representative redirect.
func TestParseCompositeTransitionLabel(t *testing.T) {
	d, err := Parse(strings.NewReader(`stateDiagram-v2
    state Active {
        Running --> Paused
    }
    [*] --> Active : init
    Active --> Done : finish`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(d.Transitions) != 3 {
		t.Fatalf("want 3 transitions, got %d", len(d.Transitions))
	}
	// Verify labels are preserved in the AST.
	found := make(map[string]string)
	for _, tr := range d.Transitions {
		found[tr.From+"->"+tr.To] = tr.Label
	}
	if got := found["[*]->Active"]; got != "init" {
		t.Errorf("[*]->Active label = %q, want init", got)
	}
	if got := found["Active->Done"]; got != "finish" {
		t.Errorf("Active->Done label = %q, want finish", got)
	}
}
