package diagram

import "testing"

func TestSequenceImplementsDiagram(t *testing.T) {
	var d Diagram = &SequenceDiagram{}
	if d.Type() != Sequence {
		t.Errorf("expected Type() = Sequence, got %v", d.Type())
	}
}

func TestSequenceConstruction(t *testing.T) {
	s := &SequenceDiagram{
		Participants: []Participant{
			{ID: "A", Alias: "Alice", Kind: ParticipantKindParticipant},
			{ID: "B", Alias: "Bob", Kind: ParticipantKindActor},
		},
		Messages: []Message{
			{From: "A", To: "B", Label: "Hello", ArrowType: ArrowTypeSolid, Activate: true},
		},
		AutoNumber: true,
	}
	if len(s.Participants) != 2 {
		t.Errorf("expected 2 participants, got %d", len(s.Participants))
	}
	if !s.AutoNumber {
		t.Error("AutoNumber should be true")
	}
}

func TestParticipantKindString(t *testing.T) {
	cases := map[ParticipantKind]string{
		ParticipantKindUnknown:     "unknown",
		ParticipantKindParticipant: "participant",
		ParticipantKindActor:       "actor",
	}
	for pk, want := range cases {
		if got := pk.String(); got != want {
			t.Errorf("ParticipantKind(%d).String() = %q, want %q", pk, got, want)
		}
	}
}

func TestArrowTypeAllEightVariants(t *testing.T) {
	// The design doc requires all 8 Mermaid sequence arrow types.
	types := []ArrowType{
		ArrowTypeSolid,
		ArrowTypeSolidNoHead,
		ArrowTypeDashed,
		ArrowTypeDashedNoHead,
		ArrowTypeSolidCross,
		ArrowTypeDashedCross,
		ArrowTypeSolidOpen,
		ArrowTypeDashedOpen,
	}
	seen := make(map[string]bool)
	for _, at := range types {
		str := at.String()
		if str == "" || str == "unknown" {
			t.Errorf("ArrowType(%d) should have a specific name", at)
		}
		if seen[str] {
			t.Errorf("duplicate ArrowType String: %q", str)
		}
		seen[str] = true
	}
	if len(seen) != 8 {
		t.Errorf("expected 8 distinct arrow types, got %d", len(seen))
	}
}

func TestArrowTypeUnknown(t *testing.T) {
	if ArrowTypeUnknown.String() != "unknown" {
		t.Errorf("expected 'unknown', got %q", ArrowTypeUnknown.String())
	}
}

func TestBlockConstruction(t *testing.T) {
	b := Block{
		Kind:  BlockKindAlt,
		Label: "condition",
		Messages: []Message{
			{From: "A", To: "B", Label: "yes"},
		},
		Else: []Block{
			{Kind: BlockKindAlt, Label: "else", Messages: []Message{{From: "A", To: "B", Label: "no"}}},
		},
	}
	if b.Kind != BlockKindAlt {
		t.Errorf("expected BlockKindAlt, got %v", b.Kind)
	}
	if len(b.Else) != 1 {
		t.Errorf("expected 1 else branch, got %d", len(b.Else))
	}
}

func TestBlockKindString(t *testing.T) {
	kinds := []BlockKind{
		BlockKindUnknown,
		BlockKindAlt,
		BlockKindOpt,
		BlockKindLoop,
		BlockKindPar,
		BlockKindCritical,
		BlockKindBreak,
		BlockKindRect,
	}
	seen := make(map[string]bool)
	for _, k := range kinds {
		s := k.String()
		if s == "" {
			t.Errorf("BlockKind(%d) has empty String()", k)
		}
		if seen[s] {
			t.Errorf("duplicate BlockKind String: %q", s)
		}
		seen[s] = true
	}
}

func TestNoteConstruction(t *testing.T) {
	n := Note{
		Position:     NotePositionRight,
		Participants: []string{"A"},
		Text:         "A note",
	}
	if n.Position != NotePositionRight {
		t.Errorf("expected NotePositionRight, got %v", n.Position)
	}
}

func TestNotePositionString(t *testing.T) {
	cases := map[NotePosition]string{
		NotePositionUnknown: "unknown",
		NotePositionLeft:    "left",
		NotePositionRight:   "right",
		NotePositionOver:    "over",
	}
	for pos, want := range cases {
		if got := pos.String(); got != want {
			t.Errorf("NotePosition(%d).String() = %q, want %q", pos, got, want)
		}
	}
}
