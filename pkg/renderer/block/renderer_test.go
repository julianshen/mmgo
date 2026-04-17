package block

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
	d := &diagram.BlockDiagram{}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	assertValidSVG(t, out)
}

func TestRenderSimpleBlocks(t *testing.T) {
	d := &diagram.BlockDiagram{
		Nodes: []diagram.BlockNode{
			{ID: "a", Label: "A"},
			{ID: "b", Label: "B"},
		},
		Edges: []diagram.BlockEdge{
			{From: "a", To: "b", Label: "flows"},
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	if !strings.Contains(raw, ">A<") || !strings.Contains(raw, ">B<") {
		t.Error("block labels missing")
	}
	if !strings.Contains(raw, ">flows<") {
		t.Error("edge label missing")
	}
	assertValidSVG(t, out)
}

func TestRenderAllShapes(t *testing.T) {
	d := &diagram.BlockDiagram{
		Nodes: []diagram.BlockNode{
			{ID: "r", Label: "R", Shape: diagram.BlockShapeRect},
			{ID: "o", Label: "O", Shape: diagram.BlockShapeRound},
			{ID: "d", Label: "D", Shape: diagram.BlockShapeDiamond},
			{ID: "s", Label: "S", Shape: diagram.BlockShapeStadium},
			{ID: "c", Label: "C", Shape: diagram.BlockShapeCircle},
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	if !strings.Contains(raw, "<polygon") {
		t.Error("diamond should produce polygon")
	}
	if !strings.Contains(raw, "<circle") {
		t.Error("circle shape should produce circle element")
	}
	assertValidSVG(t, out)
}

func TestRenderColumnsHint(t *testing.T) {
	d := &diagram.BlockDiagram{
		Columns: 3,
		Nodes: []diagram.BlockNode{
			{ID: "a", Label: "A"},
			{ID: "b", Label: "B"},
			{ID: "c", Label: "C"},
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	assertValidSVG(t, out)
}

func TestRenderEdgeLabels(t *testing.T) {
	d := &diagram.BlockDiagram{
		Nodes: []diagram.BlockNode{
			{ID: "a", Label: "A"},
			{ID: "b", Label: "B"},
			{ID: "c", Label: "C"},
			{ID: "d", Label: "D"},
		},
		Edges: []diagram.BlockEdge{
			{From: "a", To: "b", Label: "step1"},
			{From: "a", To: "c", Label: "step2"},
			{From: "b", To: "d"},
			{From: "c", To: "d"},
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	if !strings.Contains(raw, ">step1<") || !strings.Contains(raw, ">step2<") {
		t.Error("edge labels missing")
	}
	assertValidSVG(t, out)
}

func TestRenderCustomFontSize(t *testing.T) {
	d := &diagram.BlockDiagram{
		Nodes: []diagram.BlockNode{{ID: "a", Label: "A"}},
	}
	out, err := Render(d, &Options{FontSize: 20})
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if !strings.Contains(string(out), "font-size:20px") {
		t.Error("custom font size not applied")
	}
}

func TestRenderDeterministic(t *testing.T) {
	d := &diagram.BlockDiagram{
		Nodes: []diagram.BlockNode{
			{ID: "a", Label: "A"},
			{ID: "b", Label: "B"},
			{ID: "c", Label: "C"},
		},
		Edges: []diagram.BlockEdge{
			{From: "a", To: "b"},
			{From: "b", To: "c"},
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
