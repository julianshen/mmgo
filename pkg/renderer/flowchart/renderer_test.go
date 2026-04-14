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
