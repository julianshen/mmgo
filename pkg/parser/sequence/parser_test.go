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
		{ID: "Alice", Kind: diagram.ParticipantKindParticipant, BoxIndex: -1, CreatedAtItem: -1, DestroyedAtItem: -1},
		{ID: "B", Alias: "Bob", Kind: diagram.ParticipantKindParticipant, BoxIndex: -1, CreatedAtItem: -1, DestroyedAtItem: -1},
		{ID: "C", Alias: "Carol", Kind: diagram.ParticipantKindActor, BoxIndex: -1, CreatedAtItem: -1, DestroyedAtItem: -1},
		{ID: "D", Kind: diagram.ParticipantKindActor, BoxIndex: -1, CreatedAtItem: -1, DestroyedAtItem: -1},
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
		{"A<<->>B: x", diagram.ArrowTypeSolidBi},
		{"A<<-->>B: x", diagram.ArrowTypeDashedBi},
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
	if !d.AutoNumber.Enabled {
		t.Error("AutoNumber.Enabled should be true")
	}
	if d.AutoNumber.Start != 1 {
		t.Errorf("AutoNumber.Start = %d, want 1", d.AutoNumber.Start)
	}
	if d.AutoNumber.Step != 1 {
		t.Errorf("AutoNumber.Step = %d, want 1", d.AutoNumber.Step)
	}
}

func TestParseAutonumberStartOnly(t *testing.T) {
	input := "sequenceDiagram\n    autonumber 10\n    A->>B: hi"
	d, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if !d.AutoNumber.Enabled {
		t.Error("AutoNumber.Enabled should be true")
	}
	if d.AutoNumber.Start != 10 {
		t.Errorf("AutoNumber.Start = %d, want 10", d.AutoNumber.Start)
	}
	if d.AutoNumber.Step != 1 {
		t.Errorf("AutoNumber.Step = %d, want 1", d.AutoNumber.Step)
	}
}

func TestParseAutonumberStartAndStep(t *testing.T) {
	input := "sequenceDiagram\n    autonumber 10 5\n    A->>B: hi"
	d, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if !d.AutoNumber.Enabled {
		t.Error("AutoNumber.Enabled should be true")
	}
	if d.AutoNumber.Start != 10 {
		t.Errorf("AutoNumber.Start = %d, want 10", d.AutoNumber.Start)
	}
	if d.AutoNumber.Step != 5 {
		t.Errorf("AutoNumber.Step = %d, want 5", d.AutoNumber.Step)
	}
}

func TestParseAutonumberInvalidNegative(t *testing.T) {
	_, err := Parse(strings.NewReader("sequenceDiagram\n    autonumber -1"))
	if err == nil {
		t.Fatal("expected error for negative start")
	}
}

func TestParseAutonumberInvalidZero(t *testing.T) {
	_, err := Parse(strings.NewReader("sequenceDiagram\n    autonumber 0"))
	if err == nil {
		t.Fatal("expected error for zero start")
	}
}

func TestParseAutonumberInvalidZeroStep(t *testing.T) {
	_, err := Parse(strings.NewReader("sequenceDiagram\n    autonumber 5 0"))
	if err == nil {
		t.Fatal("expected error for zero step")
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

func TestParseStandaloneActivateDeactivate(t *testing.T) {
	input := `sequenceDiagram
    participant Client
    participant Server
    Client->>Server: Request
    activate Server
    Server-->>Client: Response
    deactivate Server`
	d, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(d.Items) != 4 {
		t.Fatalf("want 4 items, got %d: %+v", len(d.Items), d.Items)
	}
	a1 := d.Items[1].Activation
	if a1 == nil || a1.Participant != "Server" || !a1.Activate {
		t.Errorf("item[1] = %+v, want activate Server", d.Items[1])
	}
	a2 := d.Items[3].Activation
	if a2 == nil || a2.Participant != "Server" || a2.Activate {
		t.Errorf("item[3] = %+v, want deactivate Server", d.Items[3])
	}
}

func TestParseActivationMissingID(t *testing.T) {
	for _, kw := range []string{"activate", "deactivate"} {
		_, err := Parse(strings.NewReader("sequenceDiagram\n" + kw))
		if err == nil {
			t.Errorf("%s without id: expected error", kw)
		}
	}
}

func TestParseActivationAutoRegistersParticipant(t *testing.T) {
	d, err := Parse(strings.NewReader("sequenceDiagram\nactivate X"))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(d.Participants) != 1 || d.Participants[0].ID != "X" {
		t.Errorf("expected participant X, got %+v", d.Participants)
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

func TestParseNoteLowercaseKeyword(t *testing.T) {
	d, err := Parse(strings.NewReader("sequenceDiagram\n    note over A: lo"))
	if err != nil {
		t.Fatalf("lowercase 'note' should be accepted: %v", err)
	}
	n := d.Items[0].Note
	if n.Position != diagram.NotePositionOver || n.Text != "lo" {
		t.Errorf("got %+v", n)
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

// --- Slice C: Block structure tests ---

func TestParseSimpleLoop(t *testing.T) {
	input := `sequenceDiagram
    loop every minute
        A->>B: ping
    end`
	d, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(d.Items) != 1 {
		t.Fatalf("want 1 top-level item, got %d", len(d.Items))
	}
	b := d.Items[0].Block
	if b == nil {
		t.Fatal("expected block item")
	}
	if b.Kind != diagram.BlockKindLoop {
		t.Errorf("Kind = %v, want loop", b.Kind)
	}
	if b.Label != "every minute" {
		t.Errorf("Label = %q, want %q", b.Label, "every minute")
	}
	if len(b.Items) != 1 || b.Items[0].Message == nil {
		t.Fatalf("want 1 message inside loop, got %+v", b.Items)
	}
	if b.Items[0].Message.Label != "ping" {
		t.Errorf("inner message label = %q, want ping", b.Items[0].Message.Label)
	}
}

func TestParseAltElse(t *testing.T) {
	input := `sequenceDiagram
    alt condition
        A->>B: yes
    else other
        A->>B: no
    end`
	d, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	b := d.Items[0].Block
	if b.Kind != diagram.BlockKindAlt {
		t.Errorf("Kind = %v, want alt", b.Kind)
	}
	if b.Label != "condition" {
		t.Errorf("Label = %q", b.Label)
	}
	if len(b.Items) != 1 {
		t.Errorf("main branch should have 1 item, got %d", len(b.Items))
	}
	if len(b.Branches) != 1 {
		t.Fatalf("want 1 else branch, got %d", len(b.Branches))
	}
	eb := b.Branches[0]
	if eb.Label != "other" {
		t.Errorf("else label = %q", eb.Label)
	}
	if len(eb.Items) != 1 {
		t.Errorf("else branch should have 1 item, got %d", len(eb.Items))
	}
}

func TestParseAltMultipleElse(t *testing.T) {
	input := `sequenceDiagram
    alt a
        A->>B: 1
    else b
        A->>B: 2
    else c
        A->>B: 3
    end`
	d, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	b := d.Items[0].Block
	if len(b.Branches) != 2 {
		t.Fatalf("want 2 else branches, got %d", len(b.Branches))
	}
	if b.Branches[0].Label != "b" || b.Branches[1].Label != "c" {
		t.Errorf("branch labels = %q, %q", b.Branches[0].Label, b.Branches[1].Label)
	}
}

func TestParseParAndCritical(t *testing.T) {
	cases := []struct {
		name     string
		openKw   string
		branchKw string
		want     diagram.BlockKind
	}{
		{"par/and", "par", "and", diagram.BlockKindPar},
		{"critical/option", "critical", "option", diagram.BlockKindCritical},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			input := "sequenceDiagram\n    " + tc.openKw + " first\n        A->>B: 1\n    " + tc.branchKw + " second\n        A->>B: 2\n    end"
			d, err := Parse(strings.NewReader(input))
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			b := d.Items[0].Block
			if b.Kind != tc.want {
				t.Errorf("Kind = %v, want %v", b.Kind, tc.want)
			}
			if len(b.Branches) != 1 {
				t.Fatalf("want 1 branch, got %d", len(b.Branches))
			}
		})
	}
}

func TestParseSingleBranchBlocks(t *testing.T) {
	for _, kw := range []string{"opt", "break", "rect"} {
		t.Run(kw, func(t *testing.T) {
			input := "sequenceDiagram\n    " + kw + " label\n        A->>B: x\n    end"
			d, err := Parse(strings.NewReader(input))
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			b := d.Items[0].Block
			if b.Label != "label" {
				t.Errorf("Label = %q", b.Label)
			}
			if len(b.Items) != 1 {
				t.Errorf("want 1 inner item, got %d", len(b.Items))
			}
			if len(b.Branches) != 0 {
				t.Errorf("single-branch block should have no branches, got %d", len(b.Branches))
			}
		})
	}
}

func TestParseNestedBlocks(t *testing.T) {
	input := `sequenceDiagram
    alt outer
        loop inner
            A->>B: deep
        end
    end`
	d, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	outer := d.Items[0].Block
	if outer.Kind != diagram.BlockKindAlt {
		t.Errorf("outer Kind = %v", outer.Kind)
	}
	if len(outer.Items) != 1 || outer.Items[0].Block == nil {
		t.Fatal("outer should contain one nested block")
	}
	inner := outer.Items[0].Block
	if inner.Kind != diagram.BlockKindLoop {
		t.Errorf("inner Kind = %v", inner.Kind)
	}
	if len(inner.Items) != 1 || inner.Items[0].Message == nil {
		t.Fatal("inner loop should contain one message")
	}
	if inner.Items[0].Message.Label != "deep" {
		t.Errorf("deep message label = %q", inner.Items[0].Message.Label)
	}
}

func TestParseBlockWithNotesAndMessages(t *testing.T) {
	input := `sequenceDiagram
    opt check
        A->>B: start
        Note over A: thinking
        B->>A: done
    end`
	d, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	b := d.Items[0].Block
	if len(b.Items) != 3 {
		t.Fatalf("want 3 items (msg, note, msg), got %d", len(b.Items))
	}
	if b.Items[0].Message == nil || b.Items[1].Note == nil || b.Items[2].Message == nil {
		t.Errorf("wrong item types: %+v", b.Items)
	}
}

func TestParseEndWithoutBlockErrors(t *testing.T) {
	_, err := Parse(strings.NewReader("sequenceDiagram\n    end"))
	if err == nil {
		t.Fatal("expected error for orphan end")
	}
}

func TestParseUnclosedBlockErrors(t *testing.T) {
	_, err := Parse(strings.NewReader("sequenceDiagram\n    loop forever\n        A->>B: go"))
	if err == nil {
		t.Fatal("expected error for unclosed block")
	}
}

func TestParseElseWithoutAltErrors(t *testing.T) {
	_, err := Parse(strings.NewReader("sequenceDiagram\n    else oops"))
	if err == nil {
		t.Fatal("expected error for else without alt")
	}
}

func TestParseBranchKeywordInWrongBlock(t *testing.T) {
	// 'and' only valid inside par, not alt.
	_, err := Parse(strings.NewReader("sequenceDiagram\n    alt x\n    and y\n    end"))
	if err == nil {
		t.Fatal("expected error: 'and' inside alt")
	}
}

func TestParseEmptyBlock(t *testing.T) {
	d, err := Parse(strings.NewReader("sequenceDiagram\n    alt x\n    end"))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	b := d.Items[0].Block
	if len(b.Items) != 0 {
		t.Errorf("empty block should have no items, got %d", len(b.Items))
	}
}

func TestParseImmediateBranch(t *testing.T) {
	input := `sequenceDiagram
    alt x
    else y
        A->>B: z
    end`
	d, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	b := d.Items[0].Block
	if len(b.Items) != 0 {
		t.Errorf("main branch should be empty, got %d items", len(b.Items))
	}
	if len(b.Branches) != 1 {
		t.Fatalf("want 1 else branch, got %d", len(b.Branches))
	}
	if len(b.Branches[0].Items) != 1 {
		t.Errorf("else branch should have 1 item, got %d", len(b.Branches[0].Items))
	}
}

func TestParseBlockNoLabel(t *testing.T) {
	d, err := Parse(strings.NewReader("sequenceDiagram\n    opt\n        A->>B: x\n    end"))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if d.Items[0].Block.Label != "" {
		t.Errorf("expected empty label, got %q", d.Items[0].Block.Label)
	}
}

func TestParseBidirectionalSolid(t *testing.T) {
	d, err := Parse(strings.NewReader("sequenceDiagram\n    A<<->>B: hi"))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(d.Items) != 1 || d.Items[0].Message == nil {
		t.Fatalf("expected one message item, got %+v", d.Items)
	}
	m := d.Items[0].Message
	if m.ArrowType != diagram.ArrowTypeSolidBi {
		t.Errorf("ArrowType = %v, want %v", m.ArrowType, diagram.ArrowTypeSolidBi)
	}
	if m.From != "A" || m.To != "B" {
		t.Errorf("From/To = %q/%q, want A/B", m.From, m.To)
	}
	if m.Label != "hi" {
		t.Errorf("Label = %q, want hi", m.Label)
	}
}

func TestParseBidirectionalDashed(t *testing.T) {
	d, err := Parse(strings.NewReader("sequenceDiagram\n    A<<-->>B: hi"))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(d.Items) != 1 || d.Items[0].Message == nil {
		t.Fatalf("expected one message item, got %+v", d.Items)
	}
	m := d.Items[0].Message
	if m.ArrowType != diagram.ArrowTypeDashedBi {
		t.Errorf("ArrowType = %v, want %v", m.ArrowType, diagram.ArrowTypeDashedBi)
	}
	if m.From != "A" || m.To != "B" {
		t.Errorf("From/To = %q/%q, want A/B", m.From, m.To)
	}
	if m.Label != "hi" {
		t.Errorf("Label = %q, want hi", m.Label)
	}
}

func TestParseRectWithRgb(t *testing.T) {
	input := "sequenceDiagram\n    rect rgb(220, 240, 255)\n        A->>B: hi\n    end"
	d, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	b := d.Items[0].Block
	if b.Fill != "rgb(220, 240, 255)" {
		t.Errorf("Fill = %q, want %q", b.Fill, "rgb(220, 240, 255)")
	}
	if b.Label != "" {
		t.Errorf("Label = %q, want empty", b.Label)
	}
}

func TestParseRectWithRgba(t *testing.T) {
	input := "sequenceDiagram\n    rect rgba(255, 220, 220, 0.6)\n        A->>B: hi\n    end"
	d, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	b := d.Items[0].Block
	if b.Fill != "rgba(255, 220, 220, 0.6)" {
		t.Errorf("Fill = %q, want %q", b.Fill, "rgba(255, 220, 220, 0.6)")
	}
}

func TestParseRectWithHex(t *testing.T) {
	input := "sequenceDiagram\n    rect #ff0000\n        A->>B: hi\n    end"
	d, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	b := d.Items[0].Block
	if b.Fill != "#ff0000" {
		t.Errorf("Fill = %q, want %q", b.Fill, "#ff0000")
	}
}

func TestParseRectWithoutColorKeepsFillEmpty(t *testing.T) {
	input := "sequenceDiagram\n    rect\n        A->>B: hi\n    end"
	d, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	b := d.Items[0].Block
	if b.Fill != "" {
		t.Errorf("Fill = %q, want empty", b.Fill)
	}
}

func TestParseRectWithColorAndLabel(t *testing.T) {
	input := "sequenceDiagram\n    rect rgb(220,240,255) my label\n        A->>B: hi\n    end"
	d, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	b := d.Items[0].Block
	if b.Fill != "rgb(220,240,255)" {
		t.Errorf("Fill = %q, want %q", b.Fill, "rgb(220,240,255)")
	}
	if b.Label != "my label" {
		t.Errorf("Label = %q, want %q", b.Label, "my label")
	}
}

func TestParseBidirectionalSolidBeatsOneSided(t *testing.T) {
	d, err := Parse(strings.NewReader("sequenceDiagram\n    A<<->>B: sync"))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	m := d.Items[0].Message
	if m.ArrowType != diagram.ArrowTypeSolidBi {
		t.Errorf("<<->> should match before ->>, got %v", m.ArrowType)
	}
}

func TestParseBoxBasic(t *testing.T) {
	input := `sequenceDiagram
    box Frontend
        participant A
    end`
	d, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(d.Boxes) != 1 {
		t.Fatalf("want 1 box, got %d", len(d.Boxes))
	}
	bx := d.Boxes[0]
	if bx.Label != "Frontend" {
		t.Errorf("Label = %q, want %q", bx.Label, "Frontend")
	}
	if bx.Fill != "" {
		t.Errorf("Fill = %q, want empty", bx.Fill)
	}
	if len(bx.Members) != 1 || bx.Members[0] != "A" {
		t.Errorf("Members = %v, want [A]", bx.Members)
	}
}

func TestParseBoxWithColor(t *testing.T) {
	input := `sequenceDiagram
    box rgb(220,240,255) Backend
        participant A
    end`
	d, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(d.Boxes) != 1 {
		t.Fatalf("want 1 box, got %d", len(d.Boxes))
	}
	bx := d.Boxes[0]
	if bx.Fill != "rgb(220,240,255)" {
		t.Errorf("Fill = %q, want %q", bx.Fill, "rgb(220,240,255)")
	}
	if bx.Label != "Backend" {
		t.Errorf("Label = %q, want %q", bx.Label, "Backend")
	}
}

func TestParseBoxNestedRejected(t *testing.T) {
	input := `sequenceDiagram
    box Outer
        box Inner
        end
    end`
	_, err := Parse(strings.NewReader(input))
	if err == nil {
		t.Fatal("expected error for nested boxes")
	}
	if !strings.Contains(err.Error(), "nested") {
		t.Errorf("error should mention nested: %v", err)
	}
}

func TestParseBoxParticipantsTagged(t *testing.T) {
	input := `sequenceDiagram
    box Frontend
        participant A
        participant B as Bob
    end
    participant C`
	d, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(d.Boxes) != 1 {
		t.Fatalf("want 1 box, got %d", len(d.Boxes))
	}
	boxIdx := 0
	for _, p := range d.Participants {
		if p.ID == "A" || p.ID == "B" {
			if p.BoxIndex != boxIdx {
				t.Errorf("%s BoxIndex = %d, want %d", p.ID, p.BoxIndex, boxIdx)
			}
		}
	}
	foundC := false
	for _, p := range d.Participants {
		if p.ID == "C" {
			foundC = true
			if p.BoxIndex != -1 {
				t.Errorf("C BoxIndex = %d, want -1 (outside box)", p.BoxIndex)
			}
		}
	}
	if !foundC {
		t.Error("participant C not found")
	}
}

func TestParseBoxParticipantsInMessages(t *testing.T) {
	input := `sequenceDiagram
    box Frontend
        A->>B: hi
    end`
	d, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(d.Boxes) != 1 {
		t.Fatalf("want 1 box, got %d", len(d.Boxes))
	}
	bx := d.Boxes[0]
	if len(bx.Members) != 2 {
		t.Fatalf("want 2 members, got %d: %v", len(bx.Members), bx.Members)
	}
	if bx.Members[0] != "A" || bx.Members[1] != "B" {
		t.Errorf("Members = %v, want [A B]", bx.Members)
	}
	for _, p := range d.Participants {
		if p.BoxIndex != 0 {
			t.Errorf("%s BoxIndex = %d, want 0", p.ID, p.BoxIndex)
		}
	}
}

func TestParseBoxNoColorNoLabel(t *testing.T) {
	input := `sequenceDiagram
    box
        participant A
    end`
	d, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(d.Boxes) != 1 {
		t.Fatalf("want 1 box, got %d", len(d.Boxes))
	}
	bx := d.Boxes[0]
	if bx.Label != "" {
		t.Errorf("Label = %q, want empty", bx.Label)
	}
	if bx.Fill != "" {
		t.Errorf("Fill = %q, want empty", bx.Fill)
	}
}

func TestParseBoxWithHexColor(t *testing.T) {
	input := `sequenceDiagram
    box #ff0000 Red
        participant A
    end`
	d, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	bx := d.Boxes[0]
	if bx.Fill != "#ff0000" {
		t.Errorf("Fill = %q, want %q", bx.Fill, "#ff0000")
	}
	if bx.Label != "Red" {
		t.Errorf("Label = %q, want %q", bx.Label, "Red")
	}
}

func TestParseCreateParticipant(t *testing.T) {
	input := `sequenceDiagram
    participant Manager
    create participant Worker
    Manager->>Worker: spawn`
	d, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	var worker *diagram.Participant
	for i := range d.Participants {
		if d.Participants[i].ID == "Worker" {
			worker = &d.Participants[i]
			break
		}
	}
	if worker == nil {
		t.Fatal("Worker participant not found")
	}
	if worker.CreatedAtItem != 0 {
		t.Errorf("CreatedAtItem = %d, want 0", worker.CreatedAtItem)
	}
	if len(d.Items) != 1 || d.Items[0].Message == nil {
		t.Fatalf("expected 1 message item, got %+v", d.Items)
	}
	if d.Items[0].Message.To != "Worker" {
		t.Errorf("message To = %q, want Worker", d.Items[0].Message.To)
	}
}

func TestParseCreateActor(t *testing.T) {
	input := `sequenceDiagram
    participant System
    create actor User
    System->>User: welcome`
	d, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	var user *diagram.Participant
	for i := range d.Participants {
		if d.Participants[i].ID == "User" {
			user = &d.Participants[i]
			break
		}
	}
	if user == nil {
		t.Fatal("User participant not found")
	}
	if user.Kind != diagram.ParticipantKindActor {
		t.Errorf("Kind = %v, want actor", user.Kind)
	}
	if user.CreatedAtItem != 0 {
		t.Errorf("CreatedAtItem = %d, want 0", user.CreatedAtItem)
	}
}

func TestParseCreateWithAlias(t *testing.T) {
	input := `sequenceDiagram
    participant Manager
    create participant W as Worker
    Manager->>W: spawn`
	d, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	var w *diagram.Participant
	for i := range d.Participants {
		if d.Participants[i].ID == "W" {
			w = &d.Participants[i]
			break
		}
	}
	if w == nil {
		t.Fatal("W participant not found")
	}
	if w.Alias != "Worker" {
		t.Errorf("Alias = %q, want Worker", w.Alias)
	}
	if w.CreatedAtItem != 0 {
		t.Errorf("CreatedAtItem = %d, want 0", w.CreatedAtItem)
	}
}

func TestParseDestroySetsItem(t *testing.T) {
	input := `sequenceDiagram
    participant A
    participant B
    A->>B: work
    destroy B`
	d, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	var b *diagram.Participant
	for i := range d.Participants {
		if d.Participants[i].ID == "B" {
			b = &d.Participants[i]
			break
		}
	}
	if b == nil {
		t.Fatal("B participant not found")
	}
	if b.DestroyedAtItem != 1 {
		t.Errorf("DestroyedAtItem = %d, want 1", b.DestroyedAtItem)
	}
	if len(d.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(d.Items))
	}
	if d.Items[1].Destroy == nil || *d.Items[1].Destroy != "B" {
		t.Errorf("second item should be destroy B, got %+v", d.Items[1])
	}
}

func TestParseDestroyTwiceErrors(t *testing.T) {
	input := `sequenceDiagram
    participant A
    destroy A
    destroy A`
	_, err := Parse(strings.NewReader(input))
	if err == nil {
		t.Fatal("expected error for double destroy")
	}
	if !strings.Contains(err.Error(), "already destroyed") {
		t.Errorf("error should mention already destroyed: %v", err)
	}
}

func TestParseDestroyUnknownErrors(t *testing.T) {
	input := `sequenceDiagram
    destroy X`
	_, err := Parse(strings.NewReader(input))
	if err == nil {
		t.Fatal("expected error for destroying unknown participant")
	}
	if !strings.Contains(err.Error(), "unknown participant") {
		t.Errorf("error should mention unknown participant: %v", err)
	}
}
