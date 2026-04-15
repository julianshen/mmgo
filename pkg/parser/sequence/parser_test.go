package sequence

import (
	"strings"
	"testing"

	"github.com/julianshen/mmgo/pkg/diagram"
)

func TestParseHeaderRequired(t *testing.T) {
	_, err := Parse(strings.NewReader("A->>B: hi"))
	if err == nil {
		t.Fatal("expected error for missing header")
	}
	if !strings.Contains(err.Error(), "sequenceDiagram") {
		t.Errorf("error should mention sequenceDiagram: %v", err)
	}
}

func TestParseEmptyInput(t *testing.T) {
	_, err := Parse(strings.NewReader(""))
	if err == nil {
		t.Fatal("expected error for empty input")
	}
}

func TestParseHeaderOnly(t *testing.T) {
	d, err := Parse(strings.NewReader("sequenceDiagram"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if d == nil {
		t.Fatal("diagram is nil")
	}
	if len(d.Participants) != 0 || len(d.Items) != 0 {
		t.Errorf("empty diagram should have no participants/items, got %+v", d)
	}
}

func TestParseHeaderWithLeadingCommentAndBlanks(t *testing.T) {
	input := `%% top comment

sequenceDiagram
    %% inline comment
`
	if _, err := Parse(strings.NewReader(input)); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestParseParticipants(t *testing.T) {
	input := `sequenceDiagram
    participant Alice
    participant B as Bob
    actor C as Carol
    actor D`
	d, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	want := []diagram.Participant{
		{ID: "Alice", Kind: diagram.ParticipantKindParticipant},
		{ID: "B", Alias: "Bob", Kind: diagram.ParticipantKindParticipant},
		{ID: "C", Alias: "Carol", Kind: diagram.ParticipantKindActor},
		{ID: "D", Kind: diagram.ParticipantKindActor},
	}
	if len(d.Participants) != len(want) {
		t.Fatalf("got %d participants, want %d: %+v", len(d.Participants), len(want), d.Participants)
	}
	for i, w := range want {
		if d.Participants[i] != w {
			t.Errorf("participant[%d] = %+v, want %+v", i, d.Participants[i], w)
		}
	}
}

func TestParseAutoImpliedParticipants(t *testing.T) {
	input := `sequenceDiagram
    A->>B: hi
    C->>A: there`
	d, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	wantIDs := []string{"A", "B", "C"}
	if len(d.Participants) != len(wantIDs) {
		t.Fatalf("got %d participants, want %d", len(d.Participants), len(wantIDs))
	}
	for i, id := range wantIDs {
		if d.Participants[i].ID != id {
			t.Errorf("participant[%d].ID = %q, want %q", i, d.Participants[i].ID, id)
		}
		if d.Participants[i].Kind != diagram.ParticipantKindParticipant {
			t.Errorf("auto-registered %q should be ParticipantKindParticipant", id)
		}
	}
}

func TestParseExplicitDeclarationWinsOverImplicit(t *testing.T) {
	input := `sequenceDiagram
    A->>B: ping
    actor B as Bob`
	d, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(d.Participants) != 2 {
		t.Fatalf("got %d participants, want 2: %+v", len(d.Participants), d.Participants)
	}
	for _, p := range d.Participants {
		if p.ID == "B" {
			if p.Kind != diagram.ParticipantKindActor {
				t.Errorf("B kind = %v, want actor", p.Kind)
			}
			if p.Alias != "Bob" {
				t.Errorf("B alias = %q, want Bob", p.Alias)
			}
			return
		}
	}
	t.Error("participant B not found")
}

func TestParseAllArrowTypes(t *testing.T) {
	cases := []struct {
		src  string
		want diagram.ArrowType
	}{
		{"A->>B: x", diagram.ArrowTypeSolid},
		{"A->B: x", diagram.ArrowTypeSolidNoHead},
		{"A-->>B: x", diagram.ArrowTypeDashed},
		{"A-->B: x", diagram.ArrowTypeDashedNoHead},
		{"A-xB: x", diagram.ArrowTypeSolidCross},
		{"A--xB: x", diagram.ArrowTypeDashedCross},
		{"A-)B: x", diagram.ArrowTypeSolidOpen},
		{"A--)B: x", diagram.ArrowTypeDashedOpen},
	}
	for _, tc := range cases {
		t.Run(tc.want.String(), func(t *testing.T) {
			input := "sequenceDiagram\n" + tc.src
			d, err := Parse(strings.NewReader(input))
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			if len(d.Items) != 1 || d.Items[0].Message == nil {
				t.Fatalf("expected one message item, got %+v", d.Items)
			}
			m := d.Items[0].Message
			if m.ArrowType != tc.want {
				t.Errorf("ArrowType = %v, want %v", m.ArrowType, tc.want)
			}
			if m.From != "A" || m.To != "B" {
				t.Errorf("From/To = %q/%q, want A/B", m.From, m.To)
			}
			if m.Label != "x" {
				t.Errorf("Label = %q, want x", m.Label)
			}
		})
	}
}

func TestParseMessageLabelTrimmingAndColons(t *testing.T) {
	input := `sequenceDiagram
    A->>B:   hello: world  `
	d, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if d.Items[0].Message.Label != "hello: world" {
		t.Errorf("Label = %q, want %q", d.Items[0].Message.Label, "hello: world")
	}
}

func TestParseMessageLabelContainingArrow(t *testing.T) {
	input := `sequenceDiagram
    A->>B: send --> response`
	d, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	m := d.Items[0].Message
	if m.From != "A" || m.To != "B" {
		t.Errorf("From/To = %q/%q, want A/B", m.From, m.To)
	}
	if m.ArrowType != diagram.ArrowTypeSolid {
		t.Errorf("ArrowType = %v, want solid", m.ArrowType)
	}
	if m.Label != "send --> response" {
		t.Errorf("Label = %q, want %q", m.Label, "send --> response")
	}
}

func TestParseMessageNoLabel(t *testing.T) {
	input := `sequenceDiagram
    A->>B`
	d, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	m := d.Items[0].Message
	if m.Label != "" {
		t.Errorf("Label = %q, want empty", m.Label)
	}
}

func TestParseAutonumber(t *testing.T) {
	input := `sequenceDiagram
    autonumber
    A->>B: hi`
	d, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if !d.AutoNumber {
		t.Error("AutoNumber should be true")
	}
}

func TestParseMessageOrderPreserved(t *testing.T) {
	input := `sequenceDiagram
    A->>B: 1
    B->>A: 2
    A->>B: 3`
	d, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(d.Items) != 3 {
		t.Fatalf("got %d items, want 3", len(d.Items))
	}
	for i, want := range []string{"1", "2", "3"} {
		if d.Items[i].Message.Label != want {
			t.Errorf("item[%d].Label = %q, want %q", i, d.Items[i].Message.Label, want)
		}
	}
}

func TestParseUnknownStatementErrors(t *testing.T) {
	input := `sequenceDiagram
    wiggle A B`
	_, err := Parse(strings.NewReader(input))
	if err == nil {
		t.Fatal("expected error for unknown statement")
	}
	if !strings.Contains(err.Error(), "line 2") {
		t.Errorf("error should include line number: %v", err)
	}
}

func TestParseActivationMarkers(t *testing.T) {
	cases := []struct {
		name string
		src  string
		want diagram.LifelineEffect
		to   string
	}{
		{"activate", "A->>+B: go", diagram.LifelineEffectActivate, "B"},
		{"deactivate", "A->>-B: done", diagram.LifelineEffectDeactivate, "B"},
		{"no marker", "A->>B: plain", diagram.LifelineEffectNone, "B"},
		{"activate dashed", "A-->>+B: go", diagram.LifelineEffectActivate, "B"},
		{"activate with spaces", "A->> +B : go", diagram.LifelineEffectActivate, "B"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			d, err := Parse(strings.NewReader("sequenceDiagram\n" + tc.src))
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			m := d.Items[0].Message
			if m == nil {
				t.Fatalf("expected message, got %+v", d.Items[0])
			}
			if m.Lifeline != tc.want {
				t.Errorf("Lifeline = %v, want %v", m.Lifeline, tc.want)
			}
			if m.To != tc.to {
				t.Errorf("To = %q, want %q", m.To, tc.to)
			}
			// The +/- must not leak into the auto-registered participant ID.
			for _, p := range d.Participants {
				if strings.ContainsAny(p.ID, "+-") {
					t.Errorf("participant ID %q must not contain +/-", p.ID)
				}
			}
		})
	}
}

func TestParseNoteLeftRight(t *testing.T) {
	input := `sequenceDiagram
    Note left of Alice: first
    Note right of Bob: second`
	d, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(d.Items) != 2 {
		t.Fatalf("want 2 items, got %d", len(d.Items))
	}
	want := []struct {
		pos  diagram.NotePosition
		who  string
		text string
	}{
		{diagram.NotePositionLeft, "Alice", "first"},
		{diagram.NotePositionRight, "Bob", "second"},
	}
	for i, w := range want {
		n := d.Items[i].Note
		if n == nil {
			t.Fatalf("item[%d] is not a note: %+v", i, d.Items[i])
		}
		if n.Position != w.pos {
			t.Errorf("item[%d] position = %v, want %v", i, n.Position, w.pos)
		}
		if len(n.Participants) != 1 || n.Participants[0] != w.who {
			t.Errorf("item[%d] participants = %v, want [%s]", i, n.Participants, w.who)
		}
		if n.Text != w.text {
			t.Errorf("item[%d] text = %q, want %q", i, n.Text, w.text)
		}
	}
	// Notes auto-register their participants too.
	ids := make([]string, 0, len(d.Participants))
	for _, p := range d.Participants {
		ids = append(ids, p.ID)
	}
	if len(ids) != 2 || ids[0] != "Alice" || ids[1] != "Bob" {
		t.Errorf("participants = %v, want [Alice Bob]", ids)
	}
}

func TestParseNoteOverOneParticipant(t *testing.T) {
	d, err := Parse(strings.NewReader("sequenceDiagram\n    Note over X: hi"))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	n := d.Items[0].Note
	if n.Position != diagram.NotePositionOver {
		t.Errorf("position = %v, want over", n.Position)
	}
	if len(n.Participants) != 1 || n.Participants[0] != "X" {
		t.Errorf("participants = %v, want [X]", n.Participants)
	}
}

func TestParseNoteOverTwoParticipants(t *testing.T) {
	d, err := Parse(strings.NewReader("sequenceDiagram\n    Note over Alice, Bob: between them"))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	n := d.Items[0].Note
	if n.Position != diagram.NotePositionOver {
		t.Errorf("position = %v, want over", n.Position)
	}
	if len(n.Participants) != 2 || n.Participants[0] != "Alice" || n.Participants[1] != "Bob" {
		t.Errorf("participants = %v, want [Alice Bob]", n.Participants)
	}
	if n.Text != "between them" {
		t.Errorf("text = %q, want %q", n.Text, "between them")
	}
}

func TestParseNoteMissingColonErrors(t *testing.T) {
	_, err := Parse(strings.NewReader("sequenceDiagram\n    Note over A"))
	if err == nil {
		t.Fatal("expected error: note missing text")
	}
}

func TestParseNoteInvalidPositionErrors(t *testing.T) {
	_, err := Parse(strings.NewReader("sequenceDiagram\n    Note under A: nope"))
	if err == nil {
		t.Fatal("expected error: unknown note position")
	}
}

func TestParseNoteOverTooManyParticipantsErrors(t *testing.T) {
	_, err := Parse(strings.NewReader("sequenceDiagram\n    Note over A, B, C: oops"))
	if err == nil {
		t.Fatal("expected error: over accepts at most 2 participants")
	}
}

func TestParseNoteLeftRightWithTwoParticipantsErrors(t *testing.T) {
	// Only `Note over` accepts a comma pair; left/right are strictly single.
	_, err := Parse(strings.NewReader("sequenceDiagram\n    Note left of A, B: oops"))
	if err == nil {
		t.Fatal("expected error: left/right notes take exactly one participant")
	}
}

func TestParseParticipantMissingID(t *testing.T) {
	_, err := Parse(strings.NewReader("sequenceDiagram\nparticipant"))
	if err == nil {
		t.Error("expected error for bare participant")
	}
}

func TestParseMessageWithoutTargetIsUnrecognized(t *testing.T) {
	_, err := Parse(strings.NewReader("sequenceDiagram\nA->> : hi"))
	if err == nil {
		t.Fatal("expected error — empty target")
	}
}

func TestParseCommentStripping(t *testing.T) {
	input := `sequenceDiagram
    %% a full-line comment
    A->>B: hi %% trailing comment
    participant C %% trailing on participant`
	d, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if d.Items[0].Message.Label != "hi" {
		t.Errorf("Label = %q, want %q", d.Items[0].Message.Label, "hi")
	}
	// C should be registered despite the trailing comment
	foundC := false
	for _, p := range d.Participants {
		if p.ID == "C" {
			foundC = true
		}
	}
	if !foundC {
		t.Error("participant C not registered")
	}
}
