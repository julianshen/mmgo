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

func TestRenderAllShapes(t *testing.T) {
	d := &diagram.MindmapDiagram{
		Root: &diagram.MindmapNode{
			Text:  "Center",
			Shape: diagram.MindmapShapeDefault,
			Children: []*diagram.MindmapNode{
				{Text: "R", Shape: diagram.MindmapShapeRound},
				{Text: "S", Shape: diagram.MindmapShapeSquare},
				{Text: "C", Shape: diagram.MindmapShapeCloud},
				{Text: "B", Shape: diagram.MindmapShapeBang},
			},
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	for _, name := range []string{"Center", "R", "S", "C", "B"} {
		if !strings.Contains(raw, ">"+name+"<") {
			t.Errorf("missing %q", name)
		}
	}
	assertValidSVG(t, out)
}

func TestRenderDeepTree(t *testing.T) {
	d := &diagram.MindmapDiagram{
		Root: &diagram.MindmapNode{
			Text: "L0",
			Children: []*diagram.MindmapNode{
				{Text: "L1", Children: []*diagram.MindmapNode{
					{Text: "L2", Children: []*diagram.MindmapNode{
						{Text: "L3", Children: []*diagram.MindmapNode{
							{Text: "L4", Children: []*diagram.MindmapNode{
								{Text: "L5"},
								{Text: "L6"},
							}},
						}},
					}},
				}},
			},
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	assertValidSVG(t, out)
}

func TestRenderCustomFontSize(t *testing.T) {
	d := &diagram.MindmapDiagram{
		Root: &diagram.MindmapNode{Text: "Big"},
	}
	out, err := Render(d, &Options{FontSize: 24})
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if !strings.Contains(string(out), "font-size:24px") {
		t.Error("custom font size not applied")
	}
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
