package flowchart

import (
	"encoding/xml"
	"strings"
	"testing"

	"github.com/julianshen/mmgo/pkg/diagram"
	"github.com/julianshen/mmgo/pkg/layout"
	"github.com/julianshen/mmgo/pkg/layout/graph"
)

func TestRenderNilInputs(t *testing.T) {
	_, err := Render(nil, &layout.Result{}, nil)
	if err == nil {
		t.Fatal("expected error for nil diagram")
	}
	if !strings.Contains(err.Error(), "diagram is nil") {
		t.Errorf("unexpected error: %v", err)
	}

	_, err = Render(&diagram.FlowchartDiagram{}, nil, nil)
	if err == nil {
		t.Fatal("expected error for nil layout")
	}
	if !strings.Contains(err.Error(), "layout is nil") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRenderSingleNode(t *testing.T) {
	g := graph.New()
	g.SetNode("A", graph.NodeAttrs{Label: "Hello", Width: 100, Height: 50})
	l := layout.Layout(g, layout.Options{})
	d := &diagram.FlowchartDiagram{
		Nodes: []diagram.Node{{ID: "A", Label: "Hello", Shape: diagram.NodeShapeRectangle}},
	}

	svgBytes, err := Render(d, l, nil)
	if err != nil {
		t.Fatalf("Render error: %v", err)
	}

	raw := string(svgBytes)
	if !strings.Contains(raw, "<rect") {
		t.Errorf("SVG should contain <rect>:\n%s", raw)
	}
	if !strings.Contains(raw, ">Hello<") {
		t.Errorf("SVG should contain label text:\n%s", raw)
	}
}

func TestRenderRectangleNodeGeometry(t *testing.T) {
	n := diagram.Node{ID: "A", Label: "Hello", Shape: diagram.NodeShapeRectangle}
	nl := layout.NodeLayout{X: 100, Y: 50, Width: 80, Height: 40}
	pad := 10.0

	elems := renderNode(n, nl, pad, DefaultTheme(), 16)
	if len(elems) < 2 {
		t.Fatalf("expected at least 2 elements (rect + text), got %d", len(elems))
	}

	rect, ok := elems[0].(*Rect)
	if !ok {
		t.Fatalf("first element should be *Rect, got %T", elems[0])
	}
	wantX := nl.X - nl.Width/2 + pad
	wantY := nl.Y - nl.Height/2 + pad
	if rect.X != wantX {
		t.Errorf("rect.X = %f, want %f", rect.X, wantX)
	}
	if rect.Y != wantY {
		t.Errorf("rect.Y = %f, want %f", rect.Y, wantY)
	}

	txt, ok := elems[1].(*Text)
	if !ok {
		t.Fatalf("second element should be *Text, got %T", elems[1])
	}
	if txt.Content != "Hello" {
		t.Errorf("text.Content = %q, want %q", txt.Content, "Hello")
	}
	if txt.Anchor != "middle" {
		t.Errorf("text-anchor = %q, want middle", txt.Anchor)
	}
}

func TestRenderEmptyDiagramProducesValidSVG(t *testing.T) {
	d := &diagram.FlowchartDiagram{}
	l := layout.Layout(graph.New(), layout.Options{})

	svgBytes, err := Render(d, l, nil)
	if err != nil {
		t.Fatalf("Render error: %v", err)
	}

	raw := string(svgBytes)
	if !strings.HasPrefix(raw, "<?xml") {
		t.Fatalf("SVG should start with XML declaration, got: %q", raw[:min(len(raw), 60)])
	}

	var svg SVG
	xmlStart := strings.Index(raw, "<svg")
	if xmlStart < 0 {
		t.Fatalf("no <svg> element in output:\n%s", raw)
	}
	if err := xml.Unmarshal([]byte(raw[xmlStart:]), &svg); err != nil {
		t.Fatalf("invalid SVG XML: %v\n%s", err, raw)
	}
	if svg.XMLNS != "http://www.w3.org/2000/svg" {
		t.Errorf("xmlns = %q, want SVG namespace", svg.XMLNS)
	}
	if svg.ViewBox == "" {
		t.Error("viewBox should be set")
	}
}
