package diagram

import "testing"

func TestSequenceType(t *testing.T) {
	if (&SequenceDiagram{}).Type() != Sequence {
		t.Error("SequenceDiagram.Type() != Sequence")
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
	checkStringer(t, map[ParticipantKind]string{
		ParticipantKindUnknown:     "unknown",
		ParticipantKindParticipant: "participant",
		ParticipantKindActor:       "actor",
	})
}

func TestArrowTypeAllEightVariants(t *testing.T) {
	// The design doc requires all 8 Mermaid sequence arrow types to be distinct.
	checkUniqueStringers(t, []ArrowType{
		ArrowTypeSolid,
		ArrowTypeSolidNoHead,
		ArrowTypeDashed,
		ArrowTypeDashedNoHead,
		ArrowTypeSolidCross,
		ArrowTypeDashedCross,
		ArrowTypeSolidOpen,
		ArrowTypeDashedOpen,
	})
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
	checkUniqueStringers(t, []BlockKind{
		BlockKindUnknown,
		BlockKindAlt,
		BlockKindOpt,
		BlockKindLoop,
		BlockKindPar,
		BlockKindCritical,
		BlockKindBreak,
		BlockKindRect,
	})
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
	checkStringer(t, map[NotePosition]string{
		NotePositionUnknown: "unknown",
		NotePositionLeft:    "left",
		NotePositionRight:   "right",
		NotePositionOver:    "over",
	})
}
