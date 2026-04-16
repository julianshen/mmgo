package sequence

import (
	"bytes"
	"encoding/xml"
	"strconv"
	"strings"
	"testing"

	"github.com/julianshen/mmgo/pkg/diagram"
)

func TestRenderNilDiagram(t *testing.T) {
	_, err := Render(nil, nil)
	if err == nil {
		t.Fatal("expected error for nil diagram")
	}
}

func TestRenderEmptyDiagram(t *testing.T) {
	d := &diagram.SequenceDiagram{}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.HasPrefix(string(out), "<?xml") {
		t.Errorf("output should start with XML decl, got: %q", string(out)[:min(60, len(out))])
	}
	assertValidSVG(t, out)
}

func TestRenderParticipantBoxes(t *testing.T) {
	d := &diagram.SequenceDiagram{
		Participants: []diagram.Participant{
			{ID: "Alice", Kind: diagram.ParticipantKindParticipant},
			{ID: "Bob", Alias: "Bob the Builder", Kind: diagram.ParticipantKindParticipant},
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	if !strings.Contains(raw, ">Alice<") {
		t.Error("Alice label missing")
	}
	if !strings.Contains(raw, ">Bob the Builder<") {
		t.Error("Bob alias label missing")
	}
	assertValidSVG(t, out)
}

func TestRenderActorDrawsDifferently(t *testing.T) {
	d := &diagram.SequenceDiagram{
		Participants: []diagram.Participant{
			{ID: "A", Kind: diagram.ParticipantKindParticipant},
			{ID: "B", Kind: diagram.ParticipantKindActor},
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	// Actor should produce a stick figure (circle head + lines),
	// participant a rectangle. Just check both labels are present.
	raw := string(out)
	if !strings.Contains(raw, ">A<") || !strings.Contains(raw, ">B<") {
		t.Errorf("participant labels missing: %s", raw)
	}
	assertValidSVG(t, out)
}

func TestRenderLifelines(t *testing.T) {
	d := &diagram.SequenceDiagram{
		Participants: []diagram.Participant{
			{ID: "A", Kind: diagram.ParticipantKindParticipant},
			{ID: "B", Kind: diagram.ParticipantKindParticipant},
		},
		Items: []diagram.SequenceItem{
			diagram.NewMessageItem(diagram.Message{
				From: "A", To: "B", Label: "hello",
				ArrowType: diagram.ArrowTypeSolid,
			}),
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	// Lifelines are dashed vertical lines — check for stroke-dasharray.
	raw := string(out)
	if !strings.Contains(raw, "stroke-dasharray") {
		t.Error("expected dashed lifeline (stroke-dasharray)")
	}
	assertValidSVG(t, out)
}

func TestRenderViewBoxScalesWithParticipants(t *testing.T) {
	d2 := &diagram.SequenceDiagram{
		Participants: []diagram.Participant{
			{ID: "A", Kind: diagram.ParticipantKindParticipant},
			{ID: "B", Kind: diagram.ParticipantKindParticipant},
		},
	}
	d4 := &diagram.SequenceDiagram{
		Participants: []diagram.Participant{
			{ID: "A", Kind: diagram.ParticipantKindParticipant},
			{ID: "B", Kind: diagram.ParticipantKindParticipant},
			{ID: "C", Kind: diagram.ParticipantKindParticipant},
			{ID: "D", Kind: diagram.ParticipantKindParticipant},
		},
	}
	out2, err := Render(d2, nil)
	if err != nil {
		t.Fatalf("Render d2: %v", err)
	}
	out4, err := Render(d4, nil)
	if err != nil {
		t.Fatalf("Render d4: %v", err)
	}
	w2 := viewBoxWidth(t, out2)
	w4 := viewBoxWidth(t, out4)
	if !(w4 > w2) {
		t.Errorf("4 participants should produce wider viewBox than 2: %v vs %v", w4, w2)
	}
}

func TestRenderDeterministic(t *testing.T) {
	d := &diagram.SequenceDiagram{
		Participants: []diagram.Participant{
			{ID: "A", Kind: diagram.ParticipantKindParticipant},
			{ID: "B", Kind: diagram.ParticipantKindActor},
		},
		Items: []diagram.SequenceItem{
			diagram.NewMessageItem(diagram.Message{
				From: "A", To: "B", Label: "hello",
				ArrowType: diagram.ArrowTypeSolid,
			}),
		},
	}
	first, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	for i := 0; i < 10; i++ {
		next, err := Render(d, nil)
		if err != nil {
			t.Fatalf("iter %d: %v", i, err)
		}
		if string(next) != string(first) {
			t.Fatalf("iter %d: output diverges", i)
		}
	}
}

func TestRenderWithBlocks(t *testing.T) {
	d := &diagram.SequenceDiagram{
		Participants: []diagram.Participant{
			{ID: "A", Kind: diagram.ParticipantKindParticipant},
			{ID: "B", Kind: diagram.ParticipantKindParticipant},
		},
		Items: []diagram.SequenceItem{
			diagram.NewBlockItem(diagram.Block{
				Kind:  diagram.BlockKindLoop,
				Label: "repeat",
				Items: []diagram.SequenceItem{
					diagram.NewMessageItem(diagram.Message{
						From: "A", To: "B", Label: "ping",
						ArrowType: diagram.ArrowTypeSolid,
					}),
				},
			}),
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	assertValidSVG(t, out)
}

func TestRenderWithBlockBranches(t *testing.T) {
	d := &diagram.SequenceDiagram{
		Participants: []diagram.Participant{
			{ID: "A", Kind: diagram.ParticipantKindParticipant},
		},
		Items: []diagram.SequenceItem{
			diagram.NewBlockItem(diagram.Block{
				Kind:  diagram.BlockKindAlt,
				Label: "x",
				Items: []diagram.SequenceItem{
					diagram.NewMessageItem(diagram.Message{From: "A", To: "A", ArrowType: diagram.ArrowTypeSolid}),
				},
				Branches: []diagram.Block{
					{Kind: diagram.BlockKindAlt, Label: "y", Items: []diagram.SequenceItem{
						diagram.NewMessageItem(diagram.Message{From: "A", To: "A", ArrowType: diagram.ArrowTypeSolid}),
					}},
				},
			}),
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	assertValidSVG(t, out)
}

func TestRenderCustomOptions(t *testing.T) {
	d := &diagram.SequenceDiagram{
		Participants: []diagram.Participant{
			{ID: "A", Kind: diagram.ParticipantKindParticipant},
		},
	}
	out, err := Render(d, &Options{FontSize: 20, Padding: 40})
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	assertValidSVG(t, out)
}

func TestRenderNoteCountsAsRow(t *testing.T) {
	d := &diagram.SequenceDiagram{
		Participants: []diagram.Participant{
			{ID: "A", Kind: diagram.ParticipantKindParticipant},
		},
		Items: []diagram.SequenceItem{
			diagram.NewNoteItem(diagram.Note{
				Participants: []string{"A"},
				Text:         "hi",
				Position:     diagram.NotePositionOver,
			}),
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	assertValidSVG(t, out)
}

// --- Slice B: Messages, activation bars, auto-numbering ---

func TestRenderMessageArrow(t *testing.T) {
	d := &diagram.SequenceDiagram{
		Participants: []diagram.Participant{
			{ID: "A", Kind: diagram.ParticipantKindParticipant},
			{ID: "B", Kind: diagram.ParticipantKindParticipant},
		},
		Items: []diagram.SequenceItem{
			diagram.NewMessageItem(diagram.Message{
				From: "A", To: "B", Label: "hello",
				ArrowType: diagram.ArrowTypeSolid,
			}),
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	if !strings.Contains(raw, ">hello<") {
		t.Error("message label missing")
	}
	if !strings.Contains(raw, "marker-end") {
		t.Error("expected arrow marker on solid message")
	}
	assertValidSVG(t, out)
}

func TestRenderAllArrowTypes(t *testing.T) {
	types := []diagram.ArrowType{
		diagram.ArrowTypeSolid,
		diagram.ArrowTypeSolidNoHead,
		diagram.ArrowTypeDashed,
		diagram.ArrowTypeDashedNoHead,
		diagram.ArrowTypeSolidCross,
		diagram.ArrowTypeDashedCross,
		diagram.ArrowTypeSolidOpen,
		diagram.ArrowTypeDashedOpen,
	}
	for _, at := range types {
		t.Run(at.String(), func(t *testing.T) {
			d := &diagram.SequenceDiagram{
				Participants: []diagram.Participant{
					{ID: "A", Kind: diagram.ParticipantKindParticipant},
					{ID: "B", Kind: diagram.ParticipantKindParticipant},
				},
				Items: []diagram.SequenceItem{
					diagram.NewMessageItem(diagram.Message{
						From: "A", To: "B", Label: "x",
						ArrowType: at,
					}),
				},
			}
			out, err := Render(d, nil)
			if err != nil {
				t.Fatalf("Render: %v", err)
			}
			assertValidSVG(t, out)
		})
	}
}

func TestRenderSelfMessage(t *testing.T) {
	d := &diagram.SequenceDiagram{
		Participants: []diagram.Participant{
			{ID: "A", Kind: diagram.ParticipantKindParticipant},
		},
		Items: []diagram.SequenceItem{
			diagram.NewMessageItem(diagram.Message{
				From: "A", To: "A", Label: "self",
				ArrowType: diagram.ArrowTypeSolid,
			}),
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	if !strings.Contains(raw, ">self<") {
		t.Error("self-message label missing")
	}
	assertValidSVG(t, out)
}

func TestRenderDashedMessage(t *testing.T) {
	d := &diagram.SequenceDiagram{
		Participants: []diagram.Participant{
			{ID: "A", Kind: diagram.ParticipantKindParticipant},
			{ID: "B", Kind: diagram.ParticipantKindParticipant},
		},
		Items: []diagram.SequenceItem{
			diagram.NewMessageItem(diagram.Message{
				From: "A", To: "B", Label: "reply",
				ArrowType: diagram.ArrowTypeDashed,
			}),
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	if !strings.Contains(raw, "stroke-dasharray") {
		t.Error("dashed message should have stroke-dasharray")
	}
	assertValidSVG(t, out)
}

func TestRenderActivationBars(t *testing.T) {
	d := &diagram.SequenceDiagram{
		Participants: []diagram.Participant{
			{ID: "A", Kind: diagram.ParticipantKindParticipant},
			{ID: "B", Kind: diagram.ParticipantKindParticipant},
		},
		Items: []diagram.SequenceItem{
			diagram.NewMessageItem(diagram.Message{
				From: "A", To: "B", Label: "go",
				ArrowType: diagram.ArrowTypeSolid,
				Lifeline:  diagram.LifelineEffectActivate,
			}),
			diagram.NewMessageItem(diagram.Message{
				From: "B", To: "A", Label: "done",
				ArrowType: diagram.ArrowTypeDashed,
				Lifeline:  diagram.LifelineEffectDeactivate,
			}),
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	// Activation bar is a rect with the participant fill — count rects
	// with that fill to verify at least one activation bar was emitted
	// beyond the background and participant boxes.
	fillCount := strings.Count(raw, DefaultTheme().ParticipantFill)
	if fillCount < 2 {
		t.Errorf("expected activation bar rect with ParticipantFill, got %d occurrences", fillCount)
	}
	assertValidSVG(t, out)
}

func TestRenderAutoNumber(t *testing.T) {
	d := &diagram.SequenceDiagram{
		AutoNumber: true,
		Participants: []diagram.Participant{
			{ID: "A", Kind: diagram.ParticipantKindParticipant},
			{ID: "B", Kind: diagram.ParticipantKindParticipant},
		},
		Items: []diagram.SequenceItem{
			diagram.NewMessageItem(diagram.Message{
				From: "A", To: "B", Label: "first",
				ArrowType: diagram.ArrowTypeSolid,
			}),
			diagram.NewMessageItem(diagram.Message{
				From: "B", To: "A", Label: "second",
				ArrowType: diagram.ArrowTypeSolid,
			}),
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	if !strings.Contains(raw, ">1<") || !strings.Contains(raw, ">2<") {
		t.Error("auto-number labels missing")
	}
	assertValidSVG(t, out)
}

func TestRenderMessageHeightScales(t *testing.T) {
	d1 := &diagram.SequenceDiagram{
		Participants: []diagram.Participant{
			{ID: "A", Kind: diagram.ParticipantKindParticipant},
			{ID: "B", Kind: diagram.ParticipantKindParticipant},
		},
		Items: []diagram.SequenceItem{
			diagram.NewMessageItem(diagram.Message{From: "A", To: "B", ArrowType: diagram.ArrowTypeSolid}),
		},
	}
	d3 := &diagram.SequenceDiagram{
		Participants: d1.Participants,
		Items: []diagram.SequenceItem{
			diagram.NewMessageItem(diagram.Message{From: "A", To: "B", ArrowType: diagram.ArrowTypeSolid}),
			diagram.NewMessageItem(diagram.Message{From: "B", To: "A", ArrowType: diagram.ArrowTypeSolid}),
			diagram.NewMessageItem(diagram.Message{From: "A", To: "B", ArrowType: diagram.ArrowTypeSolid}),
		},
	}
	out1, err := Render(d1, nil)
	if err != nil {
		t.Fatalf("Render d1: %v", err)
	}
	out3, err := Render(d3, nil)
	if err != nil {
		t.Fatalf("Render d3: %v", err)
	}
	h1 := viewBoxHeight(t, out1)
	h3 := viewBoxHeight(t, out3)
	if !(h3 > h1) {
		t.Errorf("3 messages should be taller than 1: %v vs %v", h3, h1)
	}
}

// --- Helpers ---

func viewBoxHeight(t *testing.T, svgBytes []byte) float64 {
	t.Helper()
	body := svgBytes
	if i := bytes.Index(body, []byte("<svg")); i >= 0 {
		body = body[i:]
	}
	var doc struct {
		XMLName xml.Name `xml:"svg"`
		ViewBox string   `xml:"viewBox,attr"`
	}
	if err := xml.Unmarshal(body, &doc); err != nil {
		t.Fatalf("invalid SVG: %v", err)
	}
	parts := strings.Fields(doc.ViewBox)
	if len(parts) != 4 {
		t.Fatalf("viewBox should have 4 fields, got %d", len(parts))
	}
	h, err := strconv.ParseFloat(parts[3], 64)
	if err != nil {
		t.Fatalf("parse viewBox height: %v", err)
	}
	return h
}

func assertValidSVG(t *testing.T, svgBytes []byte) {
	t.Helper()
	body := svgBytes
	if i := bytes.Index(body, []byte("<svg")); i >= 0 {
		body = body[i:]
	}
	var doc struct {
		XMLName xml.Name `xml:"svg"`
		ViewBox string   `xml:"viewBox,attr"`
	}
	if err := xml.Unmarshal(body, &doc); err != nil {
		t.Fatalf("invalid SVG XML: %v\n%s", err, svgBytes)
	}
	if doc.ViewBox == "" {
		t.Error("viewBox attribute missing")
	}
}

func viewBoxWidth(t *testing.T, svgBytes []byte) float64 {
	t.Helper()
	body := svgBytes
	if i := bytes.Index(body, []byte("<svg")); i >= 0 {
		body = body[i:]
	}
	var doc struct {
		XMLName xml.Name `xml:"svg"`
		ViewBox string   `xml:"viewBox,attr"`
	}
	if err := xml.Unmarshal(body, &doc); err != nil {
		t.Fatalf("invalid SVG: %v", err)
	}
	parts := strings.Fields(doc.ViewBox)
	if len(parts) != 4 {
		t.Fatalf("viewBox should have 4 fields, got %d", len(parts))
	}
	w, err := strconv.ParseFloat(parts[2], 64)
	if err != nil {
		t.Fatalf("parse viewBox width: %v", err)
	}
	return w
}
