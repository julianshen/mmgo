package kanban

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"strings"
	"testing"

	"github.com/julianshen/mmgo/pkg/diagram"
	"github.com/julianshen/mmgo/pkg/textmeasure"
)

func TestRenderNilDiagram(t *testing.T) {
	if _, err := Render(nil, nil); err == nil {
		t.Fatal("expected error")
	}
}

func TestRenderEmptyDiagram(t *testing.T) {
	out, err := Render(&diagram.KanbanDiagram{}, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	assertValidSVG(t, out)
}

func TestRenderSingleSection(t *testing.T) {
	d := &diagram.KanbanDiagram{
		Sections: []diagram.KanbanSection{
			{Title: "Todo", Tasks: []diagram.KanbanTask{
				{Text: "Write docs"},
			}},
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	if !strings.Contains(raw, ">Todo<") {
		t.Error("section title missing")
	}
	if !strings.Contains(raw, ">Write docs<") {
		t.Error("task text missing")
	}
	assertValidSVG(t, out)
}

func TestRenderMultipleSections(t *testing.T) {
	d := &diagram.KanbanDiagram{
		Sections: []diagram.KanbanSection{
			{Title: "Todo", Tasks: []diagram.KanbanTask{{Text: "A"}}},
			{Title: "Doing", Tasks: []diagram.KanbanTask{{Text: "B"}}},
			{Title: "Done", Tasks: []diagram.KanbanTask{{Text: "C"}}},
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	for _, s := range []string{">Todo<", ">Doing<", ">Done<", ">A<", ">B<", ">C<"} {
		if !strings.Contains(raw, s) {
			t.Errorf("missing %s", s)
		}
	}
}

func TestRenderColumnPositionsLeftToRight(t *testing.T) {
	// Sections render side-by-side in declaration order. The X of
	// section N must be greater than section N-1.
	d := &diagram.KanbanDiagram{
		Sections: []diagram.KanbanSection{
			{Title: "First"}, {Title: "Second"}, {Title: "Third"},
		},
	}
	out, _ := Render(d, nil)
	raw := string(out)
	x1 := textX(t, raw, "First")
	x2 := textX(t, raw, "Second")
	x3 := textX(t, raw, "Third")
	if !(x1 < x2 && x2 < x3) {
		t.Errorf("column X order broken: %.2f %.2f %.2f", x1, x2, x3)
	}
}

func TestRenderTaskMetadata(t *testing.T) {
	d := &diagram.KanbanDiagram{
		Sections: []diagram.KanbanSection{
			{Title: "Todo", Tasks: []diagram.KanbanTask{
				{Text: "Design", Metadata: map[string]string{
					"priority": "High",
					"assigned": "alice",
				}},
			}},
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	if !strings.Contains(raw, "priority: High") {
		t.Error("priority metadata missing")
	}
	if !strings.Contains(raw, "assigned: alice") {
		t.Error("assigned metadata missing")
	}
}

func TestRenderMetadataOrderingDeterministic(t *testing.T) {
	// Map iteration is random in Go. The renderer's formatMetadata
	// must produce a stable order so SVG output is deterministic.
	d := &diagram.KanbanDiagram{
		Sections: []diagram.KanbanSection{
			{Title: "S", Tasks: []diagram.KanbanTask{
				{Text: "T", Metadata: map[string]string{
					"priority": "High",
					"assigned": "alice",
					"ticket":   "MC-1",
					"zlast":    "z",
					"middle":   "m",
				}},
			}},
		},
	}
	first, _ := Render(d, nil)
	for i := 0; i < 10; i++ {
		next, _ := Render(d, nil)
		if string(next) != string(first) {
			t.Fatalf("iter %d: output diverges (metadata order non-deterministic)", i)
		}
	}
	// The fixed-priority keys come first in the configured order.
	raw := string(first)
	priIdx := strings.Index(raw, "priority:")
	asgIdx := strings.Index(raw, "assigned:")
	tktIdx := strings.Index(raw, "ticket:")
	if priIdx >= asgIdx || asgIdx >= tktIdx {
		t.Errorf("priority/assigned/ticket order broken: %d %d %d", priIdx, asgIdx, tktIdx)
	}
}

func TestRenderDeterministic(t *testing.T) {
	d := &diagram.KanbanDiagram{
		Sections: []diagram.KanbanSection{
			{Title: "Todo", Tasks: []diagram.KanbanTask{
				{ID: "t1", Text: "First"},
				{ID: "t2", Text: "Second"},
			}},
			{Title: "Done", Tasks: []diagram.KanbanTask{
				{ID: "t3", Text: "Third", Metadata: map[string]string{"priority": "Low"}},
			}},
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
			t.Fatalf("iter %d: diverges", i)
		}
	}
}

func TestRenderLongTaskWraps(t *testing.T) {
	d := &diagram.KanbanDiagram{
		Sections: []diagram.KanbanSection{
			{Title: "Todo", Tasks: []diagram.KanbanTask{
				{Text: "A very long task description that should wrap across multiple lines because the column is narrow"},
			}},
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	// At least 2 text elements for the wrapped task body.
	raw := string(out)
	textCount := strings.Count(raw, "<text")
	if textCount < 3 {
		// 1 column title + ≥2 wrapped lines = 3+
		t.Errorf("text count = %d, want ≥3 (wrapping should produce multiple lines)", textCount)
	}
}

func TestRenderCustomFontSize(t *testing.T) {
	d := &diagram.KanbanDiagram{
		Sections: []diagram.KanbanSection{
			{Title: "T", Tasks: []diagram.KanbanTask{{Text: "x"}}},
		},
	}
	out, err := Render(d, &Options{FontSize: 20})
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if !strings.Contains(string(out), "font-size:20px") {
		t.Error("custom font size not applied")
	}
}

func TestRenderEmptySection(t *testing.T) {
	d := &diagram.KanbanDiagram{
		Sections: []diagram.KanbanSection{
			{Title: "Empty"},
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if !strings.Contains(string(out), ">Empty<") {
		t.Error("empty section title missing")
	}
}

func TestWrapText(t *testing.T) {
	ruler, err := textmeasure.NewDefaultRuler()
	if err != nil {
		t.Fatalf("NewDefaultRuler: %v", err)
	}
	defer ruler.Close()

	// Empty input still yields one (empty) line so cards have non-zero height.
	if got := wrapText(ruler, "", 200, 13); len(got) != 1 || got[0] != "" {
		t.Errorf("wrapText(\"\") = %v, want [\"\"]", got)
	}
	// Short text fits on one line.
	if got := wrapText(ruler, "Hi", 200, 13); len(got) != 1 {
		t.Errorf("short text should fit one line, got %v", got)
	}
	// Very long text wraps.
	long := strings.Repeat("word ", 30)
	if got := wrapText(ruler, long, 100, 13); len(got) < 2 {
		t.Errorf("long text should wrap, got %d line(s)", len(got))
	}
}

// Proportional-font-aware wrapping: at the same character count and
// font size, a string of narrow glyphs (i) must wrap on fewer lines
// than a string of wide glyphs (M). The pre-Phase-3 char-count
// heuristic returned identical line counts for both (over-wrapping
// narrow text and under-wrapping wide text); the Ruler-based version
// reflects the actual rendered widths.
func TestWrapTextRespectsGlyphWidth(t *testing.T) {
	ruler, err := textmeasure.NewDefaultRuler()
	if err != nil {
		t.Fatalf("NewDefaultRuler: %v", err)
	}
	defer ruler.Close()

	const fontSize = 13.0
	const width = 80.0
	narrow := strings.Repeat("i ", 30)
	wide := strings.Repeat("M ", 30)

	narrowLines := wrapText(ruler, narrow, width, fontSize)
	wideLines := wrapText(ruler, wide, width, fontSize)
	if len(narrowLines) >= len(wideLines) {
		t.Errorf("narrow-glyph text (%d lines) must wrap on fewer lines than wide-glyph text (%d lines) for accurate measurement",
			len(narrowLines), len(wideLines))
	}
}

func textX(t *testing.T, raw, content string) float64 {
	t.Helper()
	needle := ">" + content + "<"
	idx := strings.Index(raw, needle)
	if idx < 0 {
		t.Fatalf("text %q not found", content)
	}
	start := strings.LastIndex(raw[:idx], "<text")
	if start < 0 {
		t.Fatalf("<text opening for %q not found", content)
	}
	xIdx := strings.Index(raw[start:idx], ` x="`)
	if xIdx < 0 {
		t.Fatalf("x attr missing for %q", content)
	}
	xIdx += start + len(` x="`)
	end := strings.Index(raw[xIdx:], `"`)
	var v float64
	_, _ = fmt.Sscanf(raw[xIdx:xIdx+end], "%f", &v)
	return v
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
		t.Fatalf("invalid SVG: %v\n%s", err, svgBytes)
	}
	if doc.ViewBox == "" {
		t.Error("viewBox missing")
	}
}

// AccTitle/AccDescr emit as <title>/<desc> SVG children; Title
// renders as a centered caption above the columns.
func TestRenderKanbanHeader(t *testing.T) {
	d := &diagram.KanbanDiagram{
		Title:    "Sprint 12",
		AccTitle: "Sprint board",
		AccDescr: "Tasks across the sprint",
		Sections: []diagram.KanbanSection{
			{Title: "Todo", Tasks: []diagram.KanbanTask{{ID: "t1", Text: "Write tests"}}},
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	if !strings.Contains(raw, "<title>Sprint board</title>") {
		t.Errorf("expected accTitle <title> in:\n%s", raw)
	}
	if !strings.Contains(raw, "<desc>Tasks across the sprint</desc>") {
		t.Errorf("expected accDescr <desc> in:\n%s", raw)
	}
	if !strings.Contains(raw, ">Sprint 12<") {
		t.Errorf("expected diagram title rendered in:\n%s", raw)
	}
}

// Each documented priority level paints a colored left-edge
// stripe on the card.
func TestRenderKanbanPriorityStripe(t *testing.T) {
	cases := []struct {
		level string
		want  string
	}{
		{"Very High", DefaultTheme().PriorityVeryHigh},
		{"High", DefaultTheme().PriorityHigh},
		{"Low", DefaultTheme().PriorityLow},
		{"Very Low", DefaultTheme().PriorityVeryLow},
	}
	for _, tc := range cases {
		d := &diagram.KanbanDiagram{
			Sections: []diagram.KanbanSection{
				{Title: "Todo", Tasks: []diagram.KanbanTask{
					{ID: "t1", Text: "Hot fix", Metadata: map[string]string{"priority": tc.level}},
				}},
			},
		}
		out, err := Render(d, nil)
		if err != nil {
			t.Fatalf("%s: %v", tc.level, err)
		}
		if !strings.Contains(string(out), "fill:"+tc.want) {
			t.Errorf("priority %q: expected stripe fill %q in output", tc.level, tc.want)
		}
	}
}

// An unrecognised priority value silently falls through; the card
// has no stripe but otherwise renders cleanly.
func TestRenderKanbanPriorityUnknown(t *testing.T) {
	d := &diagram.KanbanDiagram{
		Sections: []diagram.KanbanSection{
			{Title: "Todo", Tasks: []diagram.KanbanTask{
				{ID: "t1", Text: "Maybe", Metadata: map[string]string{"priority": "Medium"}},
			}},
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	// None of the four priority colors should appear.
	th := DefaultTheme()
	for _, color := range []string{th.PriorityVeryHigh, th.PriorityHigh, th.PriorityLow, th.PriorityVeryLow} {
		if strings.Contains(string(out), "fill:"+color) {
			t.Errorf("unknown priority should not paint %q", color)
		}
	}
}

// When a diagram has TicketBaseURL set and a card carries a
// `ticket:` metadata value, the card is wrapped in an <a href> with
// `#TICKET#` substituted.
func TestRenderKanbanTicketLink(t *testing.T) {
	d := &diagram.KanbanDiagram{
		TicketBaseURL: "https://example.com/issues/#TICKET#",
		Sections: []diagram.KanbanSection{
			{Title: "Todo", Tasks: []diagram.KanbanTask{
				{ID: "t1", Text: "Linked", Metadata: map[string]string{"ticket": "MC-42"}},
			}},
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if !strings.Contains(string(out), `href="https://example.com/issues/MC-42"`) {
		t.Errorf("expected substituted ticket href in output:\n%s", out)
	}
}

// Cards without a `ticket:` value or without TicketBaseURL stay
// unwrapped — no <a> for that card.
func TestRenderKanbanNoTicketLink(t *testing.T) {
	for _, base := range []string{"", "https://x.example/#TICKET#"} {
		d := &diagram.KanbanDiagram{
			TicketBaseURL: base,
			Sections: []diagram.KanbanSection{
				{Title: "Todo", Tasks: []diagram.KanbanTask{
					{ID: "t1", Text: "Plain"},
				}},
			},
		}
		out, err := Render(d, nil)
		if err != nil {
			t.Fatalf("Render: %v", err)
		}
		if strings.Contains(string(out), "<a ") {
			t.Errorf("no ticket = no <a> wrap, got:\n%s", out)
		}
	}
}
