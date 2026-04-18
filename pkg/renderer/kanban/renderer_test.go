package kanban

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"strings"
	"testing"

	"github.com/julianshen/mmgo/pkg/diagram"
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
	// Empty input still yields one (empty) line so cards have non-zero height.
	if got := wrapText("", 200, 13); len(got) != 1 || got[0] != "" {
		t.Errorf("wrapText(\"\") = %v, want [\"\"]", got)
	}
	// Short text fits on one line.
	if got := wrapText("Hi", 200, 13); len(got) != 1 {
		t.Errorf("short text should fit one line, got %v", got)
	}
	// Very long text wraps.
	long := strings.Repeat("word ", 30)
	if got := wrapText(long, 100, 13); len(got) < 2 {
		t.Errorf("long text should wrap, got %d line(s)", len(got))
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
