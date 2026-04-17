package mindmap

import (
	"bytes"
	"encoding/xml"
	"strings"
	"testing"

	"github.com/julianshen/mmgo/pkg/diagram"
)

func TestRenderNilDiagram(t *testing.T) {
	_, err := Render(nil, nil)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestRenderEmptyDiagram(t *testing.T) {
	d := &diagram.MindmapDiagram{}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	assertValidSVG(t, out)
}

func TestRenderWithRoot(t *testing.T) {
	d := &diagram.MindmapDiagram{
		Root: &diagram.MindmapNode{Text: "Central Idea"},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if !strings.Contains(string(out), ">Central Idea<") {
		t.Error("root text missing")
	}
	assertValidSVG(t, out)
}

func TestRenderWithChildren(t *testing.T) {
	d := &diagram.MindmapDiagram{
		Root: &diagram.MindmapNode{
			Text: "Root",
			Children: []*diagram.MindmapNode{
				{Text: "Branch 1"},
				{Text: "Branch 2", Children: []*diagram.MindmapNode{
					{Text: "Leaf"},
				}},
			},
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	for _, want := range []string{"Root", "Branch 1", "Branch 2", "Leaf"} {
		if !strings.Contains(raw, ">"+want+"<") {
			t.Errorf("missing %q", want)
		}
	}
	assertValidSVG(t, out)
}

func TestRenderNodeShapes(t *testing.T) {
	d := &diagram.MindmapDiagram{
		Root: &diagram.MindmapNode{
			Text:  "R",
			Shape: diagram.MindmapShapeRound,
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	assertValidSVG(t, out)
}

func TestRenderDeterministic(t *testing.T) {
	d := &diagram.MindmapDiagram{
		Root: &diagram.MindmapNode{
			Text: "A",
			Children: []*diagram.MindmapNode{
				{Text: "B"},
				{Text: "C"},
			},
		},
	}
	first, _ := Render(d, nil)
	for i := 0; i < 10; i++ {
		next, _ := Render(d, nil)
		if string(next) != string(first) {
			t.Fatalf("iter %d: output diverges", i)
		}
	}
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
