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
		Items: []SequenceItem{
			NewMessageItem(Message{From: "A", To: "B", Label: "Hello", ArrowType: ArrowTypeSolid, Lifeline: LifelineEffectActivate}),
		},
		AutoNumber: AutoNumber{Enabled: true},
	}
	if len(s.Participants) != 2 {
		t.Errorf("expected 2 participants, got %d", len(s.Participants))
	}
	if !s.AutoNumber.Enabled {
		t.Error("AutoNumber should be true")
	}
	if s.Items[0].Message == nil {
		t.Fatal("expected Message in first item")
	}
	if s.Items[0].Message.Lifeline != LifelineEffectActivate {
		t.Error("Lifeline effect not preserved")
	}
}

func TestSequenceItemsPreserveSourceOrder(t *testing.T) {
	// Regression test: the bug fixed in type-design review was that
	// Messages, Blocks, and Notes were three separate slices, which
	// couldn't preserve source order like: message, loop, message.
	s := &SequenceDiagram{
		Items: []SequenceItem{
			NewMessageItem(Message{From: "A", To: "B", Label: "hi"}),
			NewBlockItem(Block{
				Kind:  BlockKindLoop,
				Label: "every minute",
				Items: []SequenceItem{
					NewMessageItem(Message{From: "B", To: "A", Label: "pong"}),
				},
			}),
			NewMessageItem(Message{From: "A", To: "B", Label: "bye"}),
		},
	}
	if len(s.Items) != 3 {
		t.Fatalf("expected 3 items, got %d", len(s.Items))
	}
	if s.Items[0].Message == nil || s.Items[0].Message.Label != "hi" {
		t.Error("first item should be 'hi' message")
	}
	if s.Items[1].Block == nil || s.Items[1].Block.Kind != BlockKindLoop {
		t.Error("second item should be loop block")
	}
	if s.Items[2].Message == nil || s.Items[2].Message.Label != "bye" {
		t.Error("third item should be 'bye' message")
	}
}

func TestSequenceItemConstructors(t *testing.T) {
	m := NewMessageItem(Message{Label: "m"})
	if m.Message == nil || m.Block != nil || m.Note != nil {
		t.Error("NewMessageItem should only populate Message field")
	}

	b := NewBlockItem(Block{Label: "b"})
	if b.Block == nil || b.Message != nil || b.Note != nil {
		t.Error("NewBlockItem should only populate Block field")
	}

	n := NewNoteItem(Note{Text: "n"})
	if n.Note == nil || n.Message != nil || n.Block != nil {
		t.Error("NewNoteItem should only populate Note field")
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
	// Pin exact string values for all 8 arrow types — not just uniqueness,
	// so name-swap regressions in arrowTypeNames are caught.
	checkStringer(t, map[ArrowType]string{
		ArrowTypeSolid:        "solid",
		ArrowTypeSolidNoHead:  "solid-no-head",
		ArrowTypeDashed:       "dashed",
		ArrowTypeDashedNoHead: "dashed-no-head",
		ArrowTypeSolidCross:   "solid-cross",
		ArrowTypeDashedCross:  "dashed-cross",
		ArrowTypeSolidOpen:    "solid-open",
		ArrowTypeDashedOpen:   "dashed-open",
	})
}

func TestArrowTypeUnknown(t *testing.T) {
	if ArrowTypeUnknown.String() != "unknown" {
		t.Errorf("expected 'unknown', got %q", ArrowTypeUnknown.String())
	}
}

func TestLifelineEffectString(t *testing.T) {
	checkStringer(t, map[LifelineEffect]string{
		LifelineEffectNone:       "none",
		LifelineEffectActivate:   "activate",
		LifelineEffectDeactivate: "deactivate",
	})
}

func TestBlockConstruction(t *testing.T) {
	b := Block{
		Kind:  BlockKindAlt,
		Label: "condition",
		Items: []SequenceItem{
			NewMessageItem(Message{From: "A", To: "B", Label: "yes"}),
		},
		Branches: []Block{
			{
				Kind:  BlockKindAlt,
				Label: "else",
				Items: []SequenceItem{
					NewMessageItem(Message{From: "A", To: "B", Label: "no"}),
				},
			},
		},
	}
	if b.Kind != BlockKindAlt {
		t.Errorf("expected BlockKindAlt, got %v", b.Kind)
	}
	if len(b.Branches) != 1 {
		t.Errorf("expected 1 branch, got %d", len(b.Branches))
	}
}

func TestBlockKindString(t *testing.T) {
	// Pin exact string values for all block kinds.
	checkStringer(t, map[BlockKind]string{
		BlockKindUnknown:  "unknown",
		BlockKindAlt:      "alt",
		BlockKindOpt:      "opt",
		BlockKindLoop:     "loop",
		BlockKindPar:      "par",
		BlockKindCritical: "critical",
		BlockKindBreak:    "break",
		BlockKindRect:     "rect",
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
