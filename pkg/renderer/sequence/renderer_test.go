package sequence

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/julianshen/mmgo/pkg/diagram"
	"github.com/julianshen/mmgo/pkg/textmeasure"
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
			{ID: "Alice", Kind: diagram.ParticipantKindParticipant, CreatedAtItem: -1, DestroyedAtItem: -1},
			{ID: "Bob", Alias: "Bob the Builder", Kind: diagram.ParticipantKindParticipant, CreatedAtItem: -1, DestroyedAtItem: -1},
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
			{ID: "A", Kind: diagram.ParticipantKindParticipant, CreatedAtItem: -1, DestroyedAtItem: -1},
			{ID: "B", Kind: diagram.ParticipantKindActor, CreatedAtItem: -1, DestroyedAtItem: -1},
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

func TestRenderLifelineUsesThemeColor(t *testing.T) {
	d := &diagram.SequenceDiagram{
		Participants: []diagram.Participant{
			{ID: "A", Kind: diagram.ParticipantKindParticipant, CreatedAtItem: -1, DestroyedAtItem: -1},
			{ID: "B", Kind: diagram.ParticipantKindParticipant, CreatedAtItem: -1, DestroyedAtItem: -1},
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

	wantStyle := fmt.Sprintf("stroke:%s;stroke-width:%.1f", DefaultTheme().LifelineStroke, defaultLifelineWidth)
	if !strings.Contains(raw, wantStyle) {
		t.Errorf("lifeline should use theme style %q", wantStyle)
	}
	assertValidSVG(t, out)
}

func TestRenderLifelineCustomTheme(t *testing.T) {
	d := &diagram.SequenceDiagram{
		Participants: []diagram.Participant{
			{ID: "A", Kind: diagram.ParticipantKindParticipant, CreatedAtItem: -1, DestroyedAtItem: -1},
			{ID: "B", Kind: diagram.ParticipantKindParticipant, CreatedAtItem: -1, DestroyedAtItem: -1},
		},
		Items: []diagram.SequenceItem{
			diagram.NewMessageItem(diagram.Message{
				From: "A", To: "B", Label: "hello",
				ArrowType: diagram.ArrowTypeSolid,
			}),
		},
	}
	out, err := Render(d, &Options{
		Theme: Theme{LifelineStroke: "#00ff00"},
	})
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	for _, m := range lifelineStyleRe.FindAllString(raw, -1) {
		if !strings.Contains(m, "stroke:#00ff00") {
			t.Errorf("lifeline should use custom color #00ff00, got: %s", m)
		}
	}
	assertValidSVG(t, out)
}

func TestRenderViewBoxScalesWithParticipants(t *testing.T) {
	d2 := &diagram.SequenceDiagram{
		Participants: []diagram.Participant{
			{ID: "A", Kind: diagram.ParticipantKindParticipant, CreatedAtItem: -1, DestroyedAtItem: -1},
			{ID: "B", Kind: diagram.ParticipantKindParticipant, CreatedAtItem: -1, DestroyedAtItem: -1},
		},
	}
	d4 := &diagram.SequenceDiagram{
		Participants: []diagram.Participant{
			{ID: "A", Kind: diagram.ParticipantKindParticipant, CreatedAtItem: -1, DestroyedAtItem: -1},
			{ID: "B", Kind: diagram.ParticipantKindParticipant, CreatedAtItem: -1, DestroyedAtItem: -1},
			{ID: "C", Kind: diagram.ParticipantKindParticipant, CreatedAtItem: -1, DestroyedAtItem: -1},
			{ID: "D", Kind: diagram.ParticipantKindParticipant, CreatedAtItem: -1, DestroyedAtItem: -1},
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
			{ID: "A", Kind: diagram.ParticipantKindParticipant, CreatedAtItem: -1, DestroyedAtItem: -1},
			{ID: "B", Kind: diagram.ParticipantKindActor, CreatedAtItem: -1, DestroyedAtItem: -1},
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
			{ID: "A", Kind: diagram.ParticipantKindParticipant, CreatedAtItem: -1, DestroyedAtItem: -1},
			{ID: "B", Kind: diagram.ParticipantKindParticipant, CreatedAtItem: -1, DestroyedAtItem: -1},
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
			{ID: "A", Kind: diagram.ParticipantKindParticipant, CreatedAtItem: -1, DestroyedAtItem: -1},
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
			{ID: "A", Kind: diagram.ParticipantKindParticipant, CreatedAtItem: -1, DestroyedAtItem: -1},
		},
	}
	out, err := Render(d, &Options{FontSize: 20, Padding: 40})
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	assertValidSVG(t, out)
}

func TestRenderAccTitleAccDescrEmitsTitleAndDesc(t *testing.T) {
	d := &diagram.SequenceDiagram{
		AccTitle: "Login flow",
		AccDescr: "User authenticates against the auth service",
		Participants: []diagram.Participant{
			{ID: "U", Kind: diagram.ParticipantKindParticipant, CreatedAtItem: -1, DestroyedAtItem: -1},
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	if !strings.Contains(raw, "<title>Login flow</title>") {
		t.Errorf("AccTitle not emitted as <title>: %s", raw)
	}
	if !strings.Contains(raw, "<desc>User authenticates against the auth service</desc>") {
		t.Errorf("AccDescr not emitted as <desc>: %s", raw)
	}
	assertValidSVG(t, out)
}

func TestRenderAutoNumberEmitsCircleBadge(t *testing.T) {
	d := &diagram.SequenceDiagram{
		AutoNumber: diagram.AutoNumber{Enabled: true, Start: 1, Step: 1},
		Participants: []diagram.Participant{
			{ID: "A", Kind: diagram.ParticipantKindParticipant, CreatedAtItem: -1, DestroyedAtItem: -1},
			{ID: "B", Kind: diagram.ParticipantKindParticipant, CreatedAtItem: -1, DestroyedAtItem: -1},
		},
		Items: []diagram.SequenceItem{
			diagram.NewMessageItem(diagram.Message{From: "A", To: "B", Label: "hi", ArrowType: diagram.ArrowTypeSolid}),
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	if !strings.Contains(raw, ">1<") {
		t.Error("autonumber should emit numeric content")
	}
	// The badge must sit on the SOURCE side of the arrow (mmdc parity),
	// not the midpoint or destination. Locate the circle's cx and the
	// two participant lifeline x-coords; assert cx matches A's lifeline
	// x, not B's, and not the midpoint.
	circleCX, ok := autoNumberCircleCX(raw)
	if !ok {
		t.Fatal("autonumber should emit a <circle> badge with cx attr")
	}
	xs := lifelineXs(t, raw)
	if len(xs) < 2 {
		t.Fatalf("expected ≥2 lifeline xs, got %v", xs)
	}
	srcX, dstX := xs[0], xs[1]
	mid := (srcX + dstX) / 2
	if math.Abs(circleCX-srcX) > 0.5 {
		t.Errorf("badge cx = %.2f, want source x = %.2f (dst=%.2f, mid=%.2f)", circleCX, srcX, dstX, mid)
	}
	assertValidSVG(t, out)
}

var (
	circleRe        = regexp.MustCompile(`<circle[^>]*cx="([^"]+)"[^>]*r="10\.00"`)
	lineRe          = regexp.MustCompile(`<line x1="([^"]+)" y1="[^"]+" x2="([^"]+)"`)
	msgLineYRe      = regexp.MustCompile(`<line x1="[\d.]+" y1="([\d.]+)" x2="[\d.]+" y2="([\d.]+)" style="stroke:#333`)
	fillRectRe      = regexp.MustCompile(`<rect[^>]*y="([\d.]+)"[^>]*height="([\d.]+)"[^>]*fill:#[0-9a-fA-F]{6}`)
	lifelineStyleRe = regexp.MustCompile(`<line[^>]*stroke-width:2\.0[^>]*>`)
	markerContentRe = regexp.MustCompile(`<marker[^>]*id="seq-arrow-([^"]+)"[^>]*>(.*?)</marker>`)
)

func autoNumberCircleCX(raw string) (float64, bool) {
	m := circleRe.FindStringSubmatch(raw)
	if len(m) != 2 {
		return 0, false
	}
	v, err := strconv.ParseFloat(m[1], 64)
	if err != nil {
		return 0, false
	}
	return v, true
}

// lifelineXs returns x positions of vertical <line> elements (x1 == x2),
// in source order. Lifelines are the only vertical lines in the SVG.
func lifelineXs(t *testing.T, raw string) []float64 {
	t.Helper()
	var xs []float64
	for _, m := range lineRe.FindAllStringSubmatch(raw, -1) {
		if m[1] != m[2] {
			continue
		}
		v, err := strconv.ParseFloat(m[1], 64)
		if err != nil {
			continue
		}
		xs = append(xs, v)
	}
	return xs
}

func TestRenderTitleAppearsAboveDiagram(t *testing.T) {
	d := &diagram.SequenceDiagram{
		Title: "My Sequence",
		Participants: []diagram.Participant{
			{ID: "A", Kind: diagram.ParticipantKindParticipant, CreatedAtItem: -1, DestroyedAtItem: -1},
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if !strings.Contains(string(out), ">My Sequence<") {
		t.Errorf("title text not found in output")
	}
	assertValidSVG(t, out)
}

func TestRenderMultilineLabelEmitsMultipleTexts(t *testing.T) {
	d := &diagram.SequenceDiagram{
		Participants: []diagram.Participant{
			{ID: "A", Kind: diagram.ParticipantKindParticipant, CreatedAtItem: -1, DestroyedAtItem: -1},
			{ID: "B", Kind: diagram.ParticipantKindParticipant, CreatedAtItem: -1, DestroyedAtItem: -1},
		},
		Items: []diagram.SequenceItem{
			diagram.NewMessageItem(diagram.Message{
				From: "A", To: "B", Label: "line one<br/>line two",
				ArrowType: diagram.ArrowTypeSolid,
			}),
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	if !strings.Contains(raw, ">line one<") {
		t.Error("first line not rendered")
	}
	if !strings.Contains(raw, ">line two<") {
		t.Error("second line not rendered")
	}
	if strings.Contains(raw, "&lt;br/&gt;") || strings.Contains(raw, "<br") {
		t.Error("<br/> token leaked into output")
	}
}

func TestRenderNoteCountsAsRow(t *testing.T) {
	d := &diagram.SequenceDiagram{
		Participants: []diagram.Participant{
			{ID: "A", Kind: diagram.ParticipantKindParticipant, CreatedAtItem: -1, DestroyedAtItem: -1},
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

func TestRenderRectUsesCustomFill(t *testing.T) {
	d := &diagram.SequenceDiagram{
		Participants: []diagram.Participant{
			{ID: "A", Kind: diagram.ParticipantKindParticipant, CreatedAtItem: -1, DestroyedAtItem: -1},
			{ID: "B", Kind: diagram.ParticipantKindParticipant, CreatedAtItem: -1, DestroyedAtItem: -1},
		},
		Items: []diagram.SequenceItem{
			diagram.NewBlockItem(diagram.Block{
				Kind: diagram.BlockKindRect,
				Fill: "rgb(220, 240, 255)",
				Items: []diagram.SequenceItem{
					diagram.NewMessageItem(diagram.Message{
						From: "A", To: "B", Label: "inside",
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
	raw := string(out)
	if !strings.Contains(raw, "rgb(220, 240, 255)") {
		t.Error("expected custom fill color in SVG output")
	}
	if !strings.Contains(raw, "fill-opacity:0.2") {
		t.Error("expected fill-opacity for non-rgba fill")
	}
	assertValidSVG(t, out)
}

func TestRenderRectUsesRgbaFillAsIs(t *testing.T) {
	d := &diagram.SequenceDiagram{
		Participants: []diagram.Participant{
			{ID: "A", Kind: diagram.ParticipantKindParticipant, CreatedAtItem: -1, DestroyedAtItem: -1},
			{ID: "B", Kind: diagram.ParticipantKindParticipant, CreatedAtItem: -1, DestroyedAtItem: -1},
		},
		Items: []diagram.SequenceItem{
			diagram.NewBlockItem(diagram.Block{
				Kind:     diagram.BlockKindRect,
				Fill:     "rgba(255, 220, 220, 0.6)",
				HasAlpha: true,
				Items: []diagram.SequenceItem{
					diagram.NewMessageItem(diagram.Message{
						From: "A", To: "B", Label: "inside",
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
	raw := string(out)
	if !strings.Contains(raw, "rgba(255, 220, 220, 0.6)") {
		t.Error("expected rgba fill color in SVG output")
	}
	if strings.Contains(raw, "fill-opacity") {
		t.Error("rgba fill should not have additional fill-opacity")
	}
	assertValidSVG(t, out)
}

func TestRenderRectNoLabelBadge(t *testing.T) {
	d := &diagram.SequenceDiagram{
		Participants: []diagram.Participant{
			{ID: "A", Kind: diagram.ParticipantKindParticipant, CreatedAtItem: -1, DestroyedAtItem: -1},
			{ID: "B", Kind: diagram.ParticipantKindParticipant, CreatedAtItem: -1, DestroyedAtItem: -1},
		},
		Items: []diagram.SequenceItem{
			diagram.NewBlockItem(diagram.Block{
				Kind:     diagram.BlockKindRect,
				Fill:     "#ffcc00",
				HasAlpha: false,
				Items: []diagram.SequenceItem{
					diagram.NewMessageItem(diagram.Message{
						From: "A", To: "B", Label: "msg",
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
	raw := string(out)
	if strings.Contains(raw, ">rect<") {
		t.Error("rect block should not render a 'rect' kind label badge")
	}
	if strings.Contains(raw, ">[") {
		t.Error("rect block should not render a bracketed label")
	}
	assertValidSVG(t, out)
}

func TestRenderRectColorClipsToMessageBand(t *testing.T) {
	d := &diagram.SequenceDiagram{
		Participants: []diagram.Participant{
			{ID: "A", Kind: diagram.ParticipantKindParticipant, CreatedAtItem: -1, DestroyedAtItem: -1},
			{ID: "B", Kind: diagram.ParticipantKindParticipant, CreatedAtItem: -1, DestroyedAtItem: -1},
		},
		Items: []diagram.SequenceItem{
			diagram.NewBlockItem(diagram.Block{
				Kind:     diagram.BlockKindRect,
				Fill:     "#ffcc00",
				HasAlpha: false,
				Items: []diagram.SequenceItem{
					diagram.NewMessageItem(diagram.Message{
						From: "A", To: "B", Label: "first",
						ArrowType: diagram.ArrowTypeSolid,
					}),
					diagram.NewMessageItem(diagram.Message{
						From: "B", To: "A", Label: "second",
						ArrowType: diagram.ArrowTypeDashed,
					}),
				},
			}),
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)

	matches := msgLineYRe.FindAllStringSubmatch(raw, -1)
	if len(matches) < 2 {
		t.Fatalf("expected at least 2 message lines in SVG, found %d", len(matches))
	}
	firstMsgY, err := strconv.ParseFloat(matches[0][1], 64)
	if err != nil {
		t.Fatalf("parse first msg y: %v", err)
	}
	secondMsgY, err := strconv.ParseFloat(matches[1][1], 64)
	if err != nil {
		t.Fatalf("parse second msg y: %v", err)
	}

	fillMatch := fillRectRe.FindStringSubmatch(raw)
	if fillMatch == nil {
		t.Fatal("expected colored fill rect in SVG output")
	}
	fillY, err := strconv.ParseFloat(fillMatch[1], 64)
	if err != nil {
		t.Fatalf("parse fill y: %v", err)
	}
	fillH, err := strconv.ParseFloat(fillMatch[2], 64)
	if err != nil {
		t.Fatalf("parse fill height: %v", err)
	}
	fillBottom := fillY + fillH

	const band = 26.0
	if fillY < firstMsgY-band {
		t.Errorf("colored rect top Y=%.2f extends above first message band (msgY=%.2f, threshold=%.2f)",
			fillY, firstMsgY, firstMsgY-band)
	}
	if fillBottom > secondMsgY+band {
		t.Errorf("colored rect bottom=%.2f extends below last message band (msgY=%.2f, threshold=%.2f)",
			fillBottom, secondMsgY, secondMsgY+band)
	}
	assertValidSVG(t, out)
}

func TestRenderRectEmptyDoesNotProduceNegativeHeight(t *testing.T) {
	d := &diagram.SequenceDiagram{
		Participants: []diagram.Participant{
			{ID: "A", Kind: diagram.ParticipantKindParticipant, CreatedAtItem: -1, DestroyedAtItem: -1},
			{ID: "B", Kind: diagram.ParticipantKindParticipant, CreatedAtItem: -1, DestroyedAtItem: -1},
		},
		Items: []diagram.SequenceItem{
			diagram.NewBlockItem(diagram.Block{
				Kind: diagram.BlockKindRect,
				Fill: "#ffcc00",
			}),
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	re := regexp.MustCompile(`height="(-?[\d.]+)"`)
	for _, m := range re.FindAllStringSubmatch(raw, -1) {
		v, err := strconv.ParseFloat(m[1], 64)
		if err != nil {
			t.Fatalf("parse height: %v (input: %q)", err, m[1])
		}
		if v < 0 {
			t.Errorf("rect height=%.2f is negative", v)
		}
	}
	assertValidSVG(t, out)
}

func TestRenderBranchLabelDoesNotOverlapMessage(t *testing.T) {
	d := &diagram.SequenceDiagram{
		Participants: []diagram.Participant{
			{ID: "A", Kind: diagram.ParticipantKindParticipant, CreatedAtItem: -1, DestroyedAtItem: -1},
			{ID: "B", Kind: diagram.ParticipantKindParticipant, CreatedAtItem: -1, DestroyedAtItem: -1},
		},
		Items: []diagram.SequenceItem{
			diagram.NewBlockItem(diagram.Block{
				Kind:  diagram.BlockKindAlt,
				Label: "main_branch",
				Items: []diagram.SequenceItem{
					diagram.NewMessageItem(diagram.Message{
						From: "A", To: "B", Label: "primary_msg",
						ArrowType: diagram.ArrowTypeSolid,
					}),
				},
				Branches: []diagram.Block{{
					Kind:  diagram.BlockKindAlt,
					Label: "ELSE_LABEL",
					Items: []diagram.SequenceItem{
						diagram.NewMessageItem(diagram.Message{
							From: "A", To: "B", Label: "ALT_MSG",
							ArrowType: diagram.ArrowTypeDashed,
						}),
					},
				}},
			}),
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)

	textYRe := regexp.MustCompile(`<text[^>]*y="([\d.]+)"[^>]*>([^<]+)</text>`)
	var bracketY, msgLabelY float64 = -1, -1
	for _, m := range textYRe.FindAllStringSubmatch(raw, -1) {
		y, err := strconv.ParseFloat(m[1], 64)
		if err != nil {
			t.Fatalf("parse text y=%q: %v", m[1], err)
		}
		switch m[2] {
		case "[ELSE_LABEL]":
			bracketY = y
		case "ALT_MSG":
			msgLabelY = y
		}
	}
	if bracketY < 0 {
		t.Fatal("did not find branch bracket label [ELSE_LABEL]")
	}
	if msgLabelY < 0 {
		t.Fatal("did not find branch message label ALT_MSG")
	}
	const minGap = 12.0
	if msgLabelY-bracketY < minGap {
		t.Errorf("branch bracket label y=%.2f overlaps message label y=%.2f (gap=%.2f, min=%.2f)",
			bracketY, msgLabelY, msgLabelY-bracketY, minGap)
	}
	assertValidSVG(t, out)
}

func TestRenderNestedBlocksHaveOffsetBorders(t *testing.T) {
	d := &diagram.SequenceDiagram{
		Participants: []diagram.Participant{
			{ID: "A", Kind: diagram.ParticipantKindParticipant, CreatedAtItem: -1, DestroyedAtItem: -1},
			{ID: "B", Kind: diagram.ParticipantKindParticipant, CreatedAtItem: -1, DestroyedAtItem: -1},
		},
		Items: []diagram.SequenceItem{
			diagram.NewBlockItem(diagram.Block{
				Kind: diagram.BlockKindLoop,
				Items: []diagram.SequenceItem{
					diagram.NewBlockItem(diagram.Block{
						Kind: diagram.BlockKindAlt,
						Items: []diagram.SequenceItem{
							diagram.NewMessageItem(diagram.Message{
								From: "A", To: "B", Label: "inner",
								ArrowType: diagram.ArrowTypeSolid,
							}),
						},
					}),
				},
			}),
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)

	rectXRe := regexp.MustCompile(`<rect x="([\d.]+)"[^>]*style="fill:none;stroke:`)
	matches := rectXRe.FindAllStringSubmatch(raw, -1)
	if len(matches) < 2 {
		t.Fatalf("expected at least 2 block-border rects, found %d", len(matches))
	}
	xs := make([]float64, 0, len(matches))
	for _, m := range matches {
		v, err := strconv.ParseFloat(m[1], 64)
		if err != nil {
			t.Fatalf("parse x=%q: %v", m[1], err)
		}
		xs = append(xs, v)
	}
	// renderItems recurses inner-first, so the inner rect is appended
	// (and emitted) before the enclosing outer rect.
	innerX, outerX := xs[0], xs[1]
	if innerX <= outerX {
		t.Errorf("inner block x=%.2f should be greater than outer block x=%.2f", innerX, outerX)
	}
	assertValidSVG(t, out)
}

func TestRenderSelfMessageUsesArcPath(t *testing.T) {
	d := &diagram.SequenceDiagram{
		Participants: []diagram.Participant{
			{ID: "A", Kind: diagram.ParticipantKindParticipant, CreatedAtItem: -1, DestroyedAtItem: -1},
		},
		Items: []diagram.SequenceItem{
			diagram.NewMessageItem(diagram.Message{
				From: "A", To: "A", Label: "callback",
				ArrowType: diagram.ArrowTypeSolid,
			}),
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)

	pathRe := regexp.MustCompile(`<path d="(M[^"]*)"`)
	matches := pathRe.FindAllStringSubmatch(raw, -1)
	var selfPath string
	for _, m := range matches {
		if strings.ContainsAny(m[1], "QqCc") {
			selfPath = m[1]
			break
		}
	}
	if selfPath == "" {
		t.Fatalf("expected self-message path with curve command (Q/C); got paths: %v", matches)
	}
	assertValidSVG(t, out)
}

func TestRenderSelfMessageLabelLeftOfLifeline(t *testing.T) {
	// mmdc renders a self-message label to the *left* of the lifeline
	// (text-anchor:end) so it doesn't collide with the right-side
	// loop arc and avoids being clipped by the rightmost layout edge.
	d := &diagram.SequenceDiagram{
		Participants: []diagram.Participant{
			{ID: "Worker", Kind: diagram.ParticipantKindParticipant, CreatedAtItem: -1, DestroyedAtItem: -1},
		},
		Items: []diagram.SequenceItem{
			diagram.NewMessageItem(diagram.Message{
				From: "Worker", To: "Worker", Label: "Recursive step",
				ArrowType: diagram.ArrowTypeSolid,
			}),
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	re := regexp.MustCompile(`<text[^>]*text-anchor="(\w+)"[^>]*>Recursive step</text>`)
	m := re.FindStringSubmatch(raw)
	if m == nil {
		t.Fatalf("did not find self-message label; raw: %s", raw)
	}
	if m[1] != "end" {
		t.Errorf("self-message label should be end-anchored (left of lifeline), got text-anchor=%q", m[1])
	}
}

func TestRenderNestedActivationsOffsetByDepth(t *testing.T) {
	d := &diagram.SequenceDiagram{
		Participants: []diagram.Participant{
			{ID: "A", Kind: diagram.ParticipantKindParticipant, CreatedAtItem: -1, DestroyedAtItem: -1},
			{ID: "B", Kind: diagram.ParticipantKindParticipant, CreatedAtItem: -1, DestroyedAtItem: -1},
		},
		Items: []diagram.SequenceItem{
			diagram.NewMessageItem(diagram.Message{From: "A", To: "B", ArrowType: diagram.ArrowTypeSolid, Lifeline: diagram.LifelineEffectActivate}),
			diagram.NewMessageItem(diagram.Message{From: "A", To: "B", ArrowType: diagram.ArrowTypeSolid, Lifeline: diagram.LifelineEffectActivate}),
			diagram.NewMessageItem(diagram.Message{From: "B", To: "A", ArrowType: diagram.ArrowTypeDashed, Lifeline: diagram.LifelineEffectDeactivate}),
			diagram.NewMessageItem(diagram.Message{From: "B", To: "A", ArrowType: diagram.ArrowTypeDashed, Lifeline: diagram.LifelineEffectDeactivate}),
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)

	// Activation bars are width-10 rects with the participant fill.
	actRe := regexp.MustCompile(`<rect x="([\d.]+)" y="[\d.]+" width="10\.00"`)
	matches := actRe.FindAllStringSubmatch(raw, -1)
	if len(matches) < 2 {
		t.Fatalf("expected 2 activation rects, found %d", len(matches))
	}
	xs := make(map[float64]bool)
	for _, m := range matches {
		v, err := strconv.ParseFloat(m[1], 64)
		if err != nil {
			t.Fatalf("parse activation x=%q: %v", m[1], err)
		}
		xs[v] = true
	}
	if len(xs) < 2 {
		t.Errorf("nested activations should render at distinct x positions, got %v", xs)
	}
	assertValidSVG(t, out)
}

func TestRenderBlockBordersAreDashed(t *testing.T) {
	d := &diagram.SequenceDiagram{
		Participants: []diagram.Participant{
			{ID: "A", Kind: diagram.ParticipantKindParticipant, CreatedAtItem: -1, DestroyedAtItem: -1},
			{ID: "B", Kind: diagram.ParticipantKindParticipant, CreatedAtItem: -1, DestroyedAtItem: -1},
		},
		Items: []diagram.SequenceItem{
			diagram.NewBlockItem(diagram.Block{
				Kind:  diagram.BlockKindLoop,
				Label: "until done",
				Items: []diagram.SequenceItem{
					diagram.NewMessageItem(diagram.Message{From: "A", To: "B", Label: "ping", ArrowType: diagram.ArrowTypeSolid}),
				},
			}),
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	// The block border rect (fill:none) must have a stroke-dasharray.
	re := regexp.MustCompile(`<rect [^>]*style="fill:none;stroke:[^"]*"`)
	m := re.FindString(raw)
	if m == "" {
		t.Fatal("did not find block-border rect")
	}
	if !strings.Contains(m, "stroke-dasharray") {
		t.Errorf("block-border rect should have stroke-dasharray for parity with mmdc; got: %s", m)
	}
}

func TestRenderBranchLabelsCentered(t *testing.T) {
	d := &diagram.SequenceDiagram{
		Participants: []diagram.Participant{
			{ID: "A", Kind: diagram.ParticipantKindParticipant, CreatedAtItem: -1, DestroyedAtItem: -1},
			{ID: "B", Kind: diagram.ParticipantKindParticipant, CreatedAtItem: -1, DestroyedAtItem: -1},
		},
		Items: []diagram.SequenceItem{
			diagram.NewBlockItem(diagram.Block{
				Kind:  diagram.BlockKindAlt,
				Label: "MAIN",
				Items: []diagram.SequenceItem{
					diagram.NewMessageItem(diagram.Message{From: "A", To: "B", Label: "m1", ArrowType: diagram.ArrowTypeSolid}),
				},
				Branches: []diagram.Block{{
					Kind:  diagram.BlockKindAlt,
					Label: "ELSE_BR",
					Items: []diagram.SequenceItem{
						diagram.NewMessageItem(diagram.Message{From: "A", To: "B", Label: "m2", ArrowType: diagram.ArrowTypeDashed}),
					},
				}},
			}),
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	re := regexp.MustCompile(`<text[^>]*text-anchor="(\w+)"[^>]*>(\[(?:MAIN|ELSE_BR)\])</text>`)
	matches := re.FindAllStringSubmatch(raw, -1)
	if len(matches) < 2 {
		t.Fatalf("expected both bracket labels in SVG, found %d: %v", len(matches), matches)
	}
	for _, m := range matches {
		if m[1] != "middle" {
			t.Errorf("bracket label %s should be center-anchored, got text-anchor=%q", m[2], m[1])
		}
	}
}

func TestRenderBlockBordersUsePurpleStroke(t *testing.T) {
	d := &diagram.SequenceDiagram{
		Participants: []diagram.Participant{
			{ID: "A", Kind: diagram.ParticipantKindParticipant, CreatedAtItem: -1, DestroyedAtItem: -1},
			{ID: "B", Kind: diagram.ParticipantKindParticipant, CreatedAtItem: -1, DestroyedAtItem: -1},
		},
		Items: []diagram.SequenceItem{
			diagram.NewBlockItem(diagram.Block{
				Kind: diagram.BlockKindLoop,
				Items: []diagram.SequenceItem{
					diagram.NewMessageItem(diagram.Message{From: "A", To: "B", ArrowType: diagram.ArrowTypeSolid}),
				},
			}),
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	re := regexp.MustCompile(`<rect [^>]*style="fill:none;stroke:([^;"]+)`)
	m := re.FindStringSubmatch(raw)
	if m == nil {
		t.Fatal("did not find block-border rect")
	}
	defaultTheme := DefaultTheme()
	if m[1] != defaultTheme.ParticipantStroke {
		t.Errorf("block border stroke=%q, want ParticipantStroke=%q", m[1], defaultTheme.ParticipantStroke)
	}
}

func TestRenderBlockEndsAboveBottomParticipantRow(t *testing.T) {
	d := &diagram.SequenceDiagram{
		Participants: []diagram.Participant{
			{ID: "A", Kind: diagram.ParticipantKindParticipant, CreatedAtItem: -1, DestroyedAtItem: -1},
			{ID: "B", Kind: diagram.ParticipantKindParticipant, CreatedAtItem: -1, DestroyedAtItem: -1},
		},
		Items: []diagram.SequenceItem{
			diagram.NewBlockItem(diagram.Block{
				Kind: diagram.BlockKindLoop,
				Items: []diagram.SequenceItem{
					diagram.NewMessageItem(diagram.Message{From: "A", To: "B", ArrowType: diagram.ArrowTypeSolid}),
				},
			}),
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	rectRe := regexp.MustCompile(`<rect x="[\d.]+" y="([\d.]+)" width="[\d.]+" height="([\d.]+)"[^>]*style="fill:none`)
	rm := rectRe.FindStringSubmatch(raw)
	if rm == nil {
		t.Fatal("did not find block-border rect")
	}
	rectY, _ := strconv.ParseFloat(rm[1], 64)
	rectH, _ := strconv.ParseFloat(rm[2], 64)
	rectBottom := rectY + rectH

	// Bottom participant boxes are width-90, fill ECECFF; pick the
	// largest y to find the bottom-row band.
	pRe := regexp.MustCompile(`<rect x="[\d.]+" y="([\d.]+)" width="90\.00" height="35\.00"[^>]*fill:#ECECFF`)
	var bottomRowY float64
	for _, m := range pRe.FindAllStringSubmatch(raw, -1) {
		v, _ := strconv.ParseFloat(m[1], 64)
		if v > bottomRowY {
			bottomRowY = v
		}
	}
	if bottomRowY == 0 {
		t.Fatal("did not find bottom participant boxes")
	}
	if rectBottom > bottomRowY {
		t.Errorf("block bottom y=%.2f bleeds into bottom participant row top y=%.2f", rectBottom, bottomRowY)
	}
}

func TestRenderBoxGroupTitleNotClippedAtTop(t *testing.T) {
	d := &diagram.SequenceDiagram{
		Participants: []diagram.Participant{
			{ID: "A", Kind: diagram.ParticipantKindParticipant, CreatedAtItem: -1, DestroyedAtItem: -1},
			{ID: "B", Kind: diagram.ParticipantKindParticipant, CreatedAtItem: -1, DestroyedAtItem: -1},
		},
		Boxes: []diagram.Box{{Label: "Frontend", Members: []string{"A", "B"}}},
		Items: []diagram.SequenceItem{
			diagram.NewMessageItem(diagram.Message{From: "A", To: "B", Label: "msg", ArrowType: diagram.ArrowTypeSolid}),
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	// The box rect is the only one with the participant fill + opacity.
	re := regexp.MustCompile(`<rect x="[\d.]+" y="(-?[\d.]+)"[^>]*fill-opacity:0\.5`)
	m := re.FindStringSubmatch(raw)
	if m == nil {
		t.Fatal("did not find box-group rect")
	}
	y, err := strconv.ParseFloat(m[1], 64)
	if err != nil {
		t.Fatalf("parse box y=%q: %v", m[1], err)
	}
	if y < 0 {
		t.Errorf("box-group rect y=%.2f is above viewBox top, title would clip", y)
	}
}

func TestRenderMultilineLabelsDoNotOverlap(t *testing.T) {
	d := &diagram.SequenceDiagram{
		Participants: []diagram.Participant{
			{ID: "A", Kind: diagram.ParticipantKindParticipant, CreatedAtItem: -1, DestroyedAtItem: -1},
			{ID: "B", Kind: diagram.ParticipantKindParticipant, CreatedAtItem: -1, DestroyedAtItem: -1},
		},
		Items: []diagram.SequenceItem{
			diagram.NewMessageItem(diagram.Message{From: "A", To: "B", Label: "alpha<br/>beta", ArrowType: diagram.ArrowTypeSolid}),
			diagram.NewMessageItem(diagram.Message{From: "A", To: "B", Label: "GAMMA<br/>DELTA<br/>EPSILON", ArrowType: diagram.ArrowTypeSolid}),
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	textRe := regexp.MustCompile(`<text[^>]*y="([\d.]+)"[^>]*>([^<]+)</text>`)
	ys := map[string]float64{}
	for _, m := range textRe.FindAllStringSubmatch(raw, -1) {
		y, _ := strconv.ParseFloat(m[1], 64)
		ys[m[2]] = y
	}
	// Bottom of label-1 (alpha is the top line, beta below; both
	// appear ABOVE the message line). Top of label-2 is the highest
	// line ("GAMMA").
	betaY, ok1 := ys["beta"]
	gammaY, ok2 := ys["GAMMA"]
	if !ok1 || !ok2 {
		t.Fatalf("missing labels in SVG: betaY=%v, gammaY=%v, found=%v", ok1, ok2, ys)
	}
	// beta's baseline must be strictly above GAMMA's baseline by at
	// least one fontSize, otherwise they overlap.
	if gammaY-betaY < 12 {
		t.Errorf("multi-line labels overlap: beta y=%.2f, GAMMA y=%.2f (gap=%.2f)", betaY, gammaY, gammaY-betaY)
	}
}

func TestRenderAutoNumberOffsetsArrowFromBadge(t *testing.T) {
	tests := []struct {
		name     string
		fromID   string
		toID     string
		wantLeft bool // arrow goes left (from B to A)
	}{
		{"left-to-right", "A", "B", false},
		{"right-to-left", "B", "A", true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			d := &diagram.SequenceDiagram{
				AutoNumber: diagram.AutoNumber{Enabled: true, Start: 1, Step: 1},
				Participants: []diagram.Participant{
					{ID: "A", Kind: diagram.ParticipantKindParticipant, CreatedAtItem: -1, DestroyedAtItem: -1},
					{ID: "B", Kind: diagram.ParticipantKindParticipant, CreatedAtItem: -1, DestroyedAtItem: -1},
				},
				Items: []diagram.SequenceItem{
					diagram.NewMessageItem(diagram.Message{From: tc.fromID, To: tc.toID, Label: "msg", ArrowType: diagram.ArrowTypeSolid}),
				},
			}
			out, err := Render(d, nil)
			if err != nil {
				t.Fatalf("Render: %v", err)
			}
			raw := string(out)
			fromX := participantXForLabel(t, raw, tc.fromID)
			lineRe := regexp.MustCompile(`<line x1="([\d.]+)"[^>]*style="stroke:#333`)
			m := lineRe.FindStringSubmatch(raw)
			if m == nil {
				t.Fatal("did not find message line")
			}
			x1, _ := strconv.ParseFloat(m[1], 64)
			expected := fromX + autoNumberRadius
			if tc.wantLeft {
				expected = fromX - autoNumberRadius
			}
			if math.Abs(x1-expected) > 0.5 {
				t.Errorf("autonumber arrow x1=%.2f, want %.2f (fromX=%.2f, radius=%v)", x1, expected, fromX, autoNumberRadius)
			}
		})
	}
}

func TestRenderRectWithLabelSuppressesBadge(t *testing.T) {
	d := &diagram.SequenceDiagram{
		Participants: []diagram.Participant{
			{ID: "A", Kind: diagram.ParticipantKindParticipant, CreatedAtItem: -1, DestroyedAtItem: -1},
			{ID: "B", Kind: diagram.ParticipantKindParticipant, CreatedAtItem: -1, DestroyedAtItem: -1},
		},
		Items: []diagram.SequenceItem{
			diagram.NewBlockItem(diagram.Block{
				Kind:     diagram.BlockKindRect,
				Fill:     "#aabbcc",
				Label:    "my section",
				HasAlpha: false,
				Items: []diagram.SequenceItem{
					diagram.NewMessageItem(diagram.Message{
						From: "A", To: "B", Label: "msg",
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
	raw := string(out)
	if strings.Contains(raw, ">rect<") {
		t.Error("rect block should not render a 'rect' kind label badge")
	}
	if strings.Contains(raw, ">[my section]<") {
		t.Error("rect block should not render a bracketed label")
	}
	assertValidSVG(t, out)
}

// --- Slice B: Messages, activation bars, auto-numbering ---

func TestRenderMessageArrow(t *testing.T) {
	d := &diagram.SequenceDiagram{
		Participants: []diagram.Participant{
			{ID: "A", Kind: diagram.ParticipantKindParticipant, CreatedAtItem: -1, DestroyedAtItem: -1},
			{ID: "B", Kind: diagram.ParticipantKindParticipant, CreatedAtItem: -1, DestroyedAtItem: -1},
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
		diagram.ArrowTypeSolidBi,
		diagram.ArrowTypeDashedBi,
	}
	for _, at := range types {
		t.Run(at.String(), func(t *testing.T) {
			d := &diagram.SequenceDiagram{
				Participants: []diagram.Participant{
					{ID: "A", Kind: diagram.ParticipantKindParticipant, CreatedAtItem: -1, DestroyedAtItem: -1},
					{ID: "B", Kind: diagram.ParticipantKindParticipant, CreatedAtItem: -1, DestroyedAtItem: -1},
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

func TestRenderArrowheadFillStyle(t *testing.T) {
	d := &diagram.SequenceDiagram{
		Participants: []diagram.Participant{
			{ID: "A", Kind: diagram.ParticipantKindParticipant, CreatedAtItem: -1, DestroyedAtItem: -1},
			{ID: "B", Kind: diagram.ParticipantKindParticipant, CreatedAtItem: -1, DestroyedAtItem: -1},
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)

	cases := []struct {
		name      string
		markers   []string
		wantShape string // "polygon" (filled), "polyline" (open arrow), or "path" (cross)
	}{
		{"filled", []string{"solid", "dashed"}, "polygon"},
		{"open", []string{"solid-open", "dashed-open"}, "polyline"},
		{"cross", []string{"solid-cross", "dashed-cross"}, "path"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			for _, id := range tc.markers {
				marker := findMarker(t, raw, id)
				if marker == "" {
					t.Fatalf("marker seq-arrow-%s not found", id)
				}
				if !strings.Contains(marker, "<"+tc.wantShape) {
					t.Errorf("marker %s should contain <%s>, got: %s", id, tc.wantShape, marker)
				}
				if tc.wantShape == "polygon" {
					if strings.Contains(marker, "fill:none") {
						t.Errorf("marker %s should not have fill:none", id)
					}
				} else {
					if !strings.Contains(marker, "fill:none") {
						t.Errorf("marker %s should have fill:none, got: %s", id, marker)
					}
				}
				if tc.wantShape == "path" {
					// Two-stroke ✕: a single-stroke regression must fail.
					if !strings.Contains(marker, "M2,2 L8,8") || !strings.Contains(marker, "M2,8 L8,2") {
						t.Errorf("cross marker %s missing crossed strokes, got: %s", id, marker)
					}
					// Cross is centered on the line endpoint, unlike the
					// open/filled markers whose refX sits at the tip.
					// Reverting refX would visibly drift the ✕ off the
					// destination lifeline.
					tag := findMarkerOpenTag(raw, id)
					if !strings.Contains(tag, `refX="5.00"`) || !strings.Contains(tag, `refY="5.00"`) {
						t.Errorf("cross marker %s should center on endpoint (refX=5, refY=5), got tag: %s", id, tag)
					}
				}
			}
		})
	}
}

func findMarker(t *testing.T, raw, id string) string {
	t.Helper()
	for _, m := range markerContentRe.FindAllStringSubmatch(raw, -1) {
		if m[1] == id {
			return m[2]
		}
	}
	return ""
}

func findMarkerOpenTag(raw, id string) string {
	for _, m := range markerContentRe.FindAllStringSubmatch(raw, -1) {
		if m[1] == id {
			if i := strings.IndexByte(m[0], '>'); i >= 0 {
				return m[0][:i+1]
			}
			return m[0]
		}
	}
	return ""
}

func TestRenderSelfMessage(t *testing.T) {
	d := &diagram.SequenceDiagram{
		Participants: []diagram.Participant{
			{ID: "A", Kind: diagram.ParticipantKindParticipant, CreatedAtItem: -1, DestroyedAtItem: -1},
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
			{ID: "A", Kind: diagram.ParticipantKindParticipant, CreatedAtItem: -1, DestroyedAtItem: -1},
			{ID: "B", Kind: diagram.ParticipantKindParticipant, CreatedAtItem: -1, DestroyedAtItem: -1},
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
			{ID: "A", Kind: diagram.ParticipantKindParticipant, CreatedAtItem: -1, DestroyedAtItem: -1},
			{ID: "B", Kind: diagram.ParticipantKindParticipant, CreatedAtItem: -1, DestroyedAtItem: -1},
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

func TestRenderStandaloneActivation(t *testing.T) {
	d := &diagram.SequenceDiagram{
		Participants: []diagram.Participant{
			{ID: "A", Kind: diagram.ParticipantKindParticipant, CreatedAtItem: -1, DestroyedAtItem: -1},
			{ID: "B", Kind: diagram.ParticipantKindParticipant, CreatedAtItem: -1, DestroyedAtItem: -1},
		},
		Items: []diagram.SequenceItem{
			diagram.NewMessageItem(diagram.Message{From: "A", To: "B", Label: "req", ArrowType: diagram.ArrowTypeSolid}),
			diagram.NewActivationItem(diagram.Activation{Participant: "B", Activate: true}),
			diagram.NewMessageItem(diagram.Message{From: "B", To: "A", Label: "resp", ArrowType: diagram.ArrowTypeDashed}),
			diagram.NewActivationItem(diagram.Activation{Participant: "B", Activate: false}),
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	// At least one activation bar should be emitted (rect filled with ParticipantFill).
	if strings.Count(string(out), DefaultTheme().ParticipantFill) < 2 {
		t.Errorf("standalone activate/deactivate did not produce an activation bar")
	}
	assertValidSVG(t, out)
}

func TestRenderAutoNumber(t *testing.T) {
	d := &diagram.SequenceDiagram{
		AutoNumber: diagram.AutoNumber{Enabled: true, Start: 1, Step: 1},
		Participants: []diagram.Participant{
			{ID: "A", Kind: diagram.ParticipantKindParticipant, CreatedAtItem: -1, DestroyedAtItem: -1},
			{ID: "B", Kind: diagram.ParticipantKindParticipant, CreatedAtItem: -1, DestroyedAtItem: -1},
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

func TestRenderAutonumberCustomStartStep(t *testing.T) {
	d := &diagram.SequenceDiagram{
		AutoNumber: diagram.AutoNumber{Enabled: true, Start: 10, Step: 5},
		Participants: []diagram.Participant{
			{ID: "A", Kind: diagram.ParticipantKindParticipant, CreatedAtItem: -1, DestroyedAtItem: -1},
			{ID: "B", Kind: diagram.ParticipantKindParticipant, CreatedAtItem: -1, DestroyedAtItem: -1},
		},
		Items: []diagram.SequenceItem{
			diagram.NewMessageItem(diagram.Message{
				From: "A", To: "B", Label: "x",
				ArrowType: diagram.ArrowTypeSolid,
			}),
			diagram.NewMessageItem(diagram.Message{
				From: "B", To: "A", Label: "y",
				ArrowType: diagram.ArrowTypeSolid,
			}),
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	if !strings.Contains(raw, ">10<") {
		t.Error("first autonumber label should be 10")
	}
	if !strings.Contains(raw, ">15<") {
		t.Error("second autonumber label should be 15")
	}
	assertValidSVG(t, out)
}

func TestRenderMessageHeightScales(t *testing.T) {
	d1 := &diagram.SequenceDiagram{
		Participants: []diagram.Participant{
			{ID: "A", Kind: diagram.ParticipantKindParticipant, CreatedAtItem: -1, DestroyedAtItem: -1},
			{ID: "B", Kind: diagram.ParticipantKindParticipant, CreatedAtItem: -1, DestroyedAtItem: -1},
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

// --- Slice C: Notes, blocks, SVG integration ---

func TestRenderNoteLeftOf(t *testing.T) {
	d := &diagram.SequenceDiagram{
		Participants: []diagram.Participant{
			{ID: "A", Kind: diagram.ParticipantKindParticipant, CreatedAtItem: -1, DestroyedAtItem: -1},
		},
		Items: []diagram.SequenceItem{
			diagram.NewNoteItem(diagram.Note{
				Participants: []string{"A"},
				Text:         "left note",
				Position:     diagram.NotePositionLeft,
			}),
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	if !strings.Contains(raw, ">left note<") {
		t.Error("note text missing")
	}
	assertValidSVG(t, out)
}

func TestRenderNoteRightOf(t *testing.T) {
	d := &diagram.SequenceDiagram{
		Participants: []diagram.Participant{
			{ID: "A", Kind: diagram.ParticipantKindParticipant, CreatedAtItem: -1, DestroyedAtItem: -1},
		},
		Items: []diagram.SequenceItem{
			diagram.NewNoteItem(diagram.Note{
				Participants: []string{"A"},
				Text:         "right note",
				Position:     diagram.NotePositionRight,
			}),
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if !strings.Contains(string(out), ">right note<") {
		t.Error("note text missing")
	}
	assertValidSVG(t, out)
}

func TestRenderNoteOverTwo(t *testing.T) {
	d := &diagram.SequenceDiagram{
		Participants: []diagram.Participant{
			{ID: "A", Kind: diagram.ParticipantKindParticipant, CreatedAtItem: -1, DestroyedAtItem: -1},
			{ID: "B", Kind: diagram.ParticipantKindParticipant, CreatedAtItem: -1, DestroyedAtItem: -1},
		},
		Items: []diagram.SequenceItem{
			diagram.NewNoteItem(diagram.Note{
				Participants: []string{"A", "B"},
				Text:         "spanning",
				Position:     diagram.NotePositionOver,
			}),
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if !strings.Contains(string(out), ">spanning<") {
		t.Error("note text missing")
	}
	assertValidSVG(t, out)
}

func TestRenderBlockRegion(t *testing.T) {
	d := &diagram.SequenceDiagram{
		Participants: []diagram.Participant{
			{ID: "A", Kind: diagram.ParticipantKindParticipant, CreatedAtItem: -1, DestroyedAtItem: -1},
			{ID: "B", Kind: diagram.ParticipantKindParticipant, CreatedAtItem: -1, DestroyedAtItem: -1},
		},
		Items: []diagram.SequenceItem{
			diagram.NewBlockItem(diagram.Block{
				Kind:  diagram.BlockKindLoop,
				Label: "retry",
				Items: []diagram.SequenceItem{
					diagram.NewMessageItem(diagram.Message{
						From: "A", To: "B", Label: "try",
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
	raw := string(out)
	if !strings.Contains(raw, ">loop<") && !strings.Contains(raw, ">Loop<") {
		t.Error("block kind label missing")
	}
	if !strings.Contains(raw, ">retry<") && !strings.Contains(raw, ">[retry]<") {
		t.Error("block label missing")
	}
	assertValidSVG(t, out)
}

func TestRenderBlockWithBranches(t *testing.T) {
	d := &diagram.SequenceDiagram{
		Participants: []diagram.Participant{
			{ID: "A", Kind: diagram.ParticipantKindParticipant, CreatedAtItem: -1, DestroyedAtItem: -1},
			{ID: "B", Kind: diagram.ParticipantKindParticipant, CreatedAtItem: -1, DestroyedAtItem: -1},
		},
		Items: []diagram.SequenceItem{
			diagram.NewBlockItem(diagram.Block{
				Kind:  diagram.BlockKindAlt,
				Label: "condition",
				Items: []diagram.SequenceItem{
					diagram.NewMessageItem(diagram.Message{
						From: "A", To: "B", Label: "yes",
						ArrowType: diagram.ArrowTypeSolid,
					}),
				},
				Branches: []diagram.Block{
					{Kind: diagram.BlockKindAlt, Label: "otherwise", Items: []diagram.SequenceItem{
						diagram.NewMessageItem(diagram.Message{
							From: "A", To: "B", Label: "no",
							ArrowType: diagram.ArrowTypeSolid,
						}),
					}},
				},
			}),
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	if !strings.Contains(raw, "stroke-dasharray") {
		t.Error("branch separator should be dashed")
	}
	assertValidSVG(t, out)
}

// --- Helpers ---

// Bidirectional arrows (<<->>, <<-->>) emit inline polygon arrowheads at
// both endpoints rather than relying on SVG marker-start/marker-end. The
// marker-based form was unreliable in PNG rasterizers, which often render
// only one of the two markers per line. See audit gap G2.
func TestRenderBidirectionalSolidHasBothArrowheads(t *testing.T) {
	d := &diagram.SequenceDiagram{
		Participants: []diagram.Participant{
			{ID: "A", Kind: diagram.ParticipantKindParticipant, CreatedAtItem: -1, DestroyedAtItem: -1},
			{ID: "B", Kind: diagram.ParticipantKindParticipant, CreatedAtItem: -1, DestroyedAtItem: -1},
		},
		Items: []diagram.SequenceItem{
			diagram.NewMessageItem(diagram.Message{
				From: "A", To: "B", Label: "sync",
				ArrowType: diagram.ArrowTypeSolidBi,
			}),
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	if strings.Contains(raw, `marker-start`) || strings.Contains(raw, `marker-end`) {
		t.Error("bidirectional should not depend on marker-start/marker-end (unreliable in rasterizers)")
	}
	tipXs := bidirArrowheadTipXs(t, raw)
	if len(tipXs) != 2 {
		t.Fatalf("expected 2 bidir arrowhead polygons, got %d", len(tipXs))
	}
	// The two tips must sit at the line endpoints (fromX != toX), not the
	// same end. Endpoints are derived from the layout — assert they differ.
	if tipXs[0] == tipXs[1] {
		t.Errorf("both arrowheads at same x=%v — expected one at each line endpoint", tipXs[0])
	}
	fillCount := strings.Count(raw, fmt.Sprintf("fill:%s", DefaultTheme().MessageStroke))
	if fillCount < 2 {
		t.Errorf("bidirectional arrowheads should have fill:%s, found %d occurrences", DefaultTheme().MessageStroke, fillCount)
	}
	assertValidSVG(t, out)
}

func TestRenderBidirectionalDashedStyle(t *testing.T) {
	d := &diagram.SequenceDiagram{
		Participants: []diagram.Participant{
			{ID: "A", Kind: diagram.ParticipantKindParticipant, CreatedAtItem: -1, DestroyedAtItem: -1},
			{ID: "B", Kind: diagram.ParticipantKindParticipant, CreatedAtItem: -1, DestroyedAtItem: -1},
		},
		Items: []diagram.SequenceItem{
			diagram.NewMessageItem(diagram.Message{
				From: "A", To: "B", Label: "async-sync",
				ArrowType: diagram.ArrowTypeDashedBi,
			}),
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	if !strings.Contains(raw, "stroke-dasharray") {
		t.Error("dashed bidirectional should have stroke-dasharray")
	}
	if n := strings.Count(raw, `<polygon`); n < 2 {
		t.Errorf("bidirectional dashed should emit two polygon arrowheads, found %d <polygon> elements", n)
	}
	assertValidSVG(t, out)
}

func TestRenderBoxEmitsRectAndLabel(t *testing.T) {
	d := &diagram.SequenceDiagram{
		Participants: []diagram.Participant{
			{ID: "A", Kind: diagram.ParticipantKindParticipant, BoxIndex: 0, CreatedAtItem: -1, DestroyedAtItem: -1},
			{ID: "B", Kind: diagram.ParticipantKindParticipant, BoxIndex: 0, CreatedAtItem: -1, DestroyedAtItem: -1},
		},
		Items: []diagram.SequenceItem{
			diagram.NewMessageItem(diagram.Message{
				From: "A", To: "B", Label: "hello",
				ArrowType: diagram.ArrowTypeSolid,
			}),
		},
		Boxes: []diagram.Box{
			{Label: "Frontend", Fill: "rgb(220,240,255)", Members: []string{"A", "B"}},
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	if !strings.Contains(raw, "rgb(220,240,255)") {
		t.Error("expected box fill color in SVG output")
	}
	if !strings.Contains(raw, ">Frontend<") {
		t.Error("expected box label text in SVG output")
	}
	if !strings.Contains(raw, "fill-opacity:0.5") {
		t.Error("expected box fill-opacity")
	}
	assertValidSVG(t, out)
}

func TestRenderBoxSolidBorder(t *testing.T) {
	d := &diagram.SequenceDiagram{
		Participants: []diagram.Participant{
			{ID: "A", Kind: diagram.ParticipantKindParticipant, BoxIndex: 0, CreatedAtItem: -1, DestroyedAtItem: -1},
			{ID: "B", Kind: diagram.ParticipantKindParticipant, BoxIndex: 0, CreatedAtItem: -1, DestroyedAtItem: -1},
		},
		Items: []diagram.SequenceItem{
			diagram.NewMessageItem(diagram.Message{From: "A", To: "B", ArrowType: diagram.ArrowTypeSolid}),
		},
		Boxes: []diagram.Box{
			{Label: "Group", Fill: "rgb(220,240,255)", Members: []string{"A", "B"}},
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	boxRectRe := regexp.MustCompile(`<rect[^>]*rgb\(220,240,255\)[^>]*>`)
	match := boxRectRe.FindString(raw)
	if match == "" {
		t.Fatal("box rect not found in SVG")
	}
	if strings.Contains(match, "stroke-dasharray") {
		t.Error("box rect should use solid border, not stroke-dasharray")
	}
	assertValidSVG(t, out)
}

func TestRenderBoxTitleCentered(t *testing.T) {
	d := &diagram.SequenceDiagram{
		Participants: []diagram.Participant{
			{ID: "A", Kind: diagram.ParticipantKindParticipant, BoxIndex: 0, CreatedAtItem: -1, DestroyedAtItem: -1},
			{ID: "B", Kind: diagram.ParticipantKindParticipant, BoxIndex: 0, CreatedAtItem: -1, DestroyedAtItem: -1},
		},
		Items: []diagram.SequenceItem{
			diagram.NewMessageItem(diagram.Message{From: "A", To: "B", ArrowType: diagram.ArrowTypeSolid}),
		},
		Boxes: []diagram.Box{
			{Label: "Backend", Fill: "rgb(220,240,255)", Members: []string{"A", "B"}},
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)

	labelRe := regexp.MustCompile(`<text[^>]*>Backend</text>`)
	labelMatch := labelRe.FindString(raw)
	if labelMatch == "" {
		t.Fatal("expected 'Backend' label text in SVG")
	}
	anchorRe := regexp.MustCompile(`text-anchor="([^"]+)"`)
	anchorMatch := anchorRe.FindStringSubmatch(labelMatch)
	if len(anchorMatch) < 2 || anchorMatch[1] != "middle" {
		t.Errorf("box label should be text-anchor=middle, got %v", anchorMatch)
	}
	assertValidSVG(t, out)
}

func TestRenderBoxTitleNotClipped(t *testing.T) {
	d := &diagram.SequenceDiagram{
		Participants: []diagram.Participant{
			{ID: "A", Kind: diagram.ParticipantKindParticipant, BoxIndex: 0, CreatedAtItem: -1, DestroyedAtItem: -1},
		},
		Items: []diagram.SequenceItem{
			diagram.NewMessageItem(diagram.Message{From: "A", To: "A", ArrowType: diagram.ArrowTypeSolid}),
		},
		Boxes: []diagram.Box{
			{Label: "VeryLongTitleNameThatExceedsBoxWidth", Fill: "rgb(220,240,255)", Members: []string{"A"}},
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)

	boxRectRe := regexp.MustCompile(`<rect[^>]*style="[^"]*rgb\(220,240,255\)[^"]*"[^>]*>`)
	boxMatch := boxRectRe.FindString(raw)
	if boxMatch == "" {
		t.Fatal("expected box rect in SVG")
	}
	widthRe := regexp.MustCompile(`width="([\d.]+)"`)
	widthMatch := widthRe.FindStringSubmatch(boxMatch)
	if len(widthMatch) < 2 {
		t.Fatal("expected width attribute on box rect")
	}
	boxW, _ := strconv.ParseFloat(widthMatch[1], 64)

	titleW := textmeasure.EstimateWidth("VeryLongTitleNameThatExceedsBoxWidth", 12)
	if titleW > boxW {
		t.Errorf("title width=%.0f exceeds box width=%.0f — title will be clipped", titleW, boxW)
	}
	assertValidSVG(t, out)
}

func TestRenderBoxWithRgbaFillNoFillOpacity(t *testing.T) {
	d := &diagram.SequenceDiagram{
		Participants: []diagram.Participant{
			{ID: "A", Kind: diagram.ParticipantKindParticipant, BoxIndex: 0, CreatedAtItem: -1, DestroyedAtItem: -1},
			{ID: "B", Kind: diagram.ParticipantKindParticipant, BoxIndex: 0, CreatedAtItem: -1, DestroyedAtItem: -1},
		},
		Items: []diagram.SequenceItem{
			diagram.NewMessageItem(diagram.Message{From: "A", To: "B", ArrowType: diagram.ArrowTypeSolid}),
		},
		Boxes: []diagram.Box{
			{Fill: "rgba(255,200,200,0.6)", HasAlpha: true, Members: []string{"A", "B"}},
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	if !strings.Contains(raw, "rgba(255,200,200,0.6)") {
		t.Error("expected rgba fill in SVG output")
	}
	if strings.Contains(raw, "fill-opacity") {
		t.Error("rgba box fill should not have additional fill-opacity")
	}
	assertValidSVG(t, out)
}

func TestRenderBoxWithNoLabel(t *testing.T) {
	d := &diagram.SequenceDiagram{
		Participants: []diagram.Participant{
			{ID: "A", Kind: diagram.ParticipantKindParticipant, BoxIndex: 0, CreatedAtItem: -1, DestroyedAtItem: -1},
			{ID: "B", Kind: diagram.ParticipantKindParticipant, BoxIndex: 0, CreatedAtItem: -1, DestroyedAtItem: -1},
		},
		Items: []diagram.SequenceItem{
			diagram.NewMessageItem(diagram.Message{From: "A", To: "B", ArrowType: diagram.ArrowTypeSolid}),
		},
		Boxes: []diagram.Box{
			{Fill: "rgb(220,240,255)", Members: []string{"A", "B"}},
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	if !strings.Contains(raw, "rgb(220,240,255)") {
		t.Error("expected box fill color in SVG output")
	}

	boxRectRe := regexp.MustCompile(`<rect[^>]*rgb\(220,240,255\)[^>]*>`)
	boxMatch := boxRectRe.FindString(raw)
	if boxMatch == "" {
		t.Fatal("expected box rect in SVG output")
	}
	assertValidSVG(t, out)
}

func participantXForLabel(t *testing.T, raw, label string) float64 {
	t.Helper()
	re := regexp.MustCompile(`<text[^>]*>` + regexp.QuoteMeta(label) + `</text>`)
	match := re.FindString(raw)
	if match == "" {
		t.Fatalf("label %q not found in SVG", label)
	}
	xRe := regexp.MustCompile(`x="([\d.]+)"`)
	xMatch := xRe.FindStringSubmatch(match)
	if len(xMatch) < 2 {
		t.Fatalf("x attribute not found for label %q", label)
	}
	x, err := strconv.ParseFloat(xMatch[1], 64)
	if err != nil {
		t.Fatalf("parse x: %v", err)
	}
	return x
}

// --- Helpers (continued) ---

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

// bidirArrowheadTipXs extracts the tip x-coordinate of every polygon
// emitted by bidirArrowhead. The helper strips the <defs> block first
// so polygons inside marker definitions (which also use <polygon>) are
// excluded. The polygon's first "points" coordinate is the tip — see
// bidirArrowhead in messages.go.
func bidirArrowheadTipXs(t *testing.T, raw string) []float64 {
	t.Helper()
	if i := strings.Index(raw, "<defs>"); i >= 0 {
		if j := strings.Index(raw, "</defs>"); j > i {
			raw = raw[:i] + raw[j+len("</defs>"):]
		}
	}
	var tips []float64
	for _, m := range polygonRe.FindAllStringSubmatch(raw, -1) {
		coords := strings.Fields(strings.ReplaceAll(m[1], ",", " "))
		if len(coords) < 2 {
			continue
		}
		x, err := strconv.ParseFloat(coords[0], 64)
		if err != nil {
			continue
		}
		tips = append(tips, x)
	}
	return tips
}

var polygonRe = regexp.MustCompile(`<polygon[^>]*points="([^"]+)"`)

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

func TestRenderDestroyTerminatesLifelineWithX(t *testing.T) {
	d := &diagram.SequenceDiagram{
		Participants: []diagram.Participant{
			{ID: "A", Kind: diagram.ParticipantKindParticipant, CreatedAtItem: -1, DestroyedAtItem: -1},
			{ID: "B", Kind: diagram.ParticipantKindParticipant, CreatedAtItem: -1, DestroyedAtItem: 1},
		},
		Items: []diagram.SequenceItem{
			diagram.NewMessageItem(diagram.Message{
				From: "A", To: "B", Label: "work",
				ArrowType: diagram.ArrowTypeSolid,
			}),
			diagram.NewDestroyItem("B"),
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	if !strings.Contains(raw, ">work<") {
		t.Error("message label missing")
	}
	crossCount := strings.Count(raw, "stroke-width:3.0")
	if crossCount < 2 {
		t.Errorf("expected at least 2 crossing lines for X glyph (got %d cross-style lines)", crossCount)
	}
	assertValidSVG(t, out)
}

func TestRenderCreatedParticipantBoxStartsMidDiagram(t *testing.T) {
	d := &diagram.SequenceDiagram{
		Participants: []diagram.Participant{
			{ID: "Manager", Kind: diagram.ParticipantKindParticipant, CreatedAtItem: -1, DestroyedAtItem: -1},
			{ID: "Worker", Kind: diagram.ParticipantKindParticipant, CreatedAtItem: 0, DestroyedAtItem: -1},
		},
		Items: []diagram.SequenceItem{
			diagram.NewMessageItem(diagram.Message{
				From: "Manager", To: "Worker", Label: "spawn",
				ArrowType: diagram.ArrowTypeSolid,
			}),
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	if !strings.Contains(raw, ">spawn<") {
		t.Error("message label missing")
	}
	if !strings.Contains(raw, ">Manager<") {
		t.Error("Manager label missing")
	}
	if !strings.Contains(raw, ">Worker<") {
		t.Error("Worker label missing")
	}
	managerCount := strings.Count(raw, ">Manager<")
	if managerCount != 2 {
		t.Errorf("Manager should appear in 2 boxes (top+bottom), got %d", managerCount)
	}
	assertValidSVG(t, out)
}

func TestRenderCreateParticipantStopsArrowAtBoxEdge(t *testing.T) {
	d := &diagram.SequenceDiagram{
		Participants: []diagram.Participant{
			{ID: "M", Kind: diagram.ParticipantKindParticipant, CreatedAtItem: -1, DestroyedAtItem: -1},
			{ID: "W", Kind: diagram.ParticipantKindParticipant, CreatedAtItem: 0, DestroyedAtItem: -1},
		},
		Items: []diagram.SequenceItem{
			diagram.NewMessageItem(diagram.Message{
				From: "M", To: "W", Label: "spawn",
				ArrowType: diagram.ArrowTypeSolid,
			}),
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)

	wX := participantXForLabel(t, raw, "W")

	msgLineRe := regexp.MustCompile(`<line x1="([\d.]+)"[^>]*y1="[\d.]+" x2="([\d.]+)"[^>]*y2="[\d.]+" style="[^"]*stroke:[^"]*"`)
	matches := msgLineRe.FindAllStringSubmatch(raw, -1)
	lifelineStroke := DefaultTheme().LifelineStroke
	for _, m := range matches {
		// Lifelines use the theme accent color; message lines use
		// MessageStroke. Filter by color so the test isn't coupled
		// to incidental width values.
		if strings.Contains(m[0], "stroke:"+lifelineStroke) {
			continue
		}
		x2, _ := strconv.ParseFloat(m[2], 64)
		if x2 > 100 {
			if x2 >= wX-1 {
				t.Errorf("spawn arrow x2=%.0f reaches W's lifeline center x=%.0f — should stop at box edge", x2, wX)
			}
			break
		}
	}
	assertValidSVG(t, out)
}

func TestRenderDestroyEmitsBottomBoxAtDestroyY(t *testing.T) {
	d := &diagram.SequenceDiagram{
		Participants: []diagram.Participant{
			{ID: "A", Kind: diagram.ParticipantKindParticipant, CreatedAtItem: -1, DestroyedAtItem: -1},
			{ID: "B", Kind: diagram.ParticipantKindParticipant, CreatedAtItem: -1, DestroyedAtItem: 2},
		},
		Items: []diagram.SequenceItem{
			diagram.NewMessageItem(diagram.Message{From: "A", To: "B", ArrowType: diagram.ArrowTypeSolid}),
			diagram.NewMessageItem(diagram.Message{From: "B", To: "A", ArrowType: diagram.ArrowTypeDashed}),
			diagram.NewDestroyItem("B"),
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)

	bCount := strings.Count(raw, ">B<")
	if bCount < 2 {
		t.Errorf("destroyed participant B should appear in at least 2 boxes (top + bottom at destroy y), got %d", bCount)
	}
	assertValidSVG(t, out)
}

func TestRenderDestroyClipsLifeline(t *testing.T) {
	d := &diagram.SequenceDiagram{
		Participants: []diagram.Participant{
			{ID: "A", Kind: diagram.ParticipantKindParticipant, CreatedAtItem: -1, DestroyedAtItem: -1},
			{ID: "B", Kind: diagram.ParticipantKindParticipant, CreatedAtItem: -1, DestroyedAtItem: 1},
		},
		Items: []diagram.SequenceItem{
			diagram.NewMessageItem(diagram.Message{From: "A", To: "B", ArrowType: diagram.ArrowTypeSolid}),
			diagram.NewDestroyItem("B"),
			diagram.NewMessageItem(diagram.Message{From: "A", To: "A", ArrowType: diagram.ArrowTypeDashed}),
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)

	bX := participantXForLabel(t, raw, "B")

	bXRe := regexp.MustCompile(`<line x1="([\d.]+)" y1="([\d.]+)" x2="([\d.]+)" y2="([\d.]+)"[^>]*style="[^"]*stroke-dasharray`)
	for _, m := range bXRe.FindAllStringSubmatch(raw, -1) {
		x1, _ := strconv.ParseFloat(m[1], 64)
		if math.Abs(x1-bX) > 1 {
			continue
		}
		y2, _ := strconv.ParseFloat(m[4], 64)
		if y2 > 150 {
			t.Errorf("B's lifeline extends to y=%.2f, should be clipped at destroy y (~140)", y2)
		}
	}
	assertValidSVG(t, out)
}

func TestRenderCreateDestroyCombined(t *testing.T) {
	d := &diagram.SequenceDiagram{
		Participants: []diagram.Participant{
			{ID: "A", Kind: diagram.ParticipantKindParticipant, CreatedAtItem: -1, DestroyedAtItem: -1},
			{ID: "W", Kind: diagram.ParticipantKindParticipant, CreatedAtItem: 0, DestroyedAtItem: 2},
		},
		Items: []diagram.SequenceItem{
			diagram.NewMessageItem(diagram.Message{From: "A", To: "W", Label: "create", ArrowType: diagram.ArrowTypeSolid}),
			diagram.NewMessageItem(diagram.Message{From: "A", To: "W", Label: "work", ArrowType: diagram.ArrowTypeSolid}),
			diagram.NewDestroyItem("W"),
			diagram.NewMessageItem(diagram.Message{From: "A", To: "A", Label: "done", ArrowType: diagram.ArrowTypeDashed}),
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)

	wCount := strings.Count(raw, ">W<")
	if wCount < 2 {
		t.Errorf("W should appear in at least 2 boxes (created mid-diagram + bottom at destroy), got %d", wCount)
	}

	wX := 0.0
	textRe := regexp.MustCompile(`<text[^>]*>W<`)
	textMatch := textRe.FindString(raw)
	if textMatch != "" {
		xRe := regexp.MustCompile(`x="([\d.]+)"`)
		xMatch := xRe.FindStringSubmatch(textMatch)
		if len(xMatch) > 1 {
			wX, _ = strconv.ParseFloat(xMatch[1], 64)
		}
	}

	bXRe := regexp.MustCompile(`<line x1="([\d.]+)" y1="([\d.]+)" x2="([\d.]+)" y2="([\d.]+)"[^>]*style="[^"]*stroke-dasharray`)
	for _, m := range bXRe.FindAllStringSubmatch(raw, -1) {
		x, _ := strconv.ParseFloat(m[1], 64)
		if math.Abs(x-wX) > 1 {
			continue
		}
		y1, _ := strconv.ParseFloat(m[2], 64)
		y2, _ := strconv.ParseFloat(m[4], 64)
		if y2-y1 > 200 {
			t.Errorf("W's lifeline span %.0f is too long — should be clipped between create and destroy", y2-y1)
		}
	}
	assertValidSVG(t, out)
}
