package sankey

import (
	"bytes"
	"encoding/xml"
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
	d := &diagram.SankeyDiagram{}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	assertValidSVG(t, out)
}

func TestRenderSingleFlow(t *testing.T) {
	d := &diagram.SankeyDiagram{
		Flows: []diagram.SankeyFlow{
			{Source: "A", Target: "B", Value: 10},
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	if !strings.Contains(raw, ">A<") || !strings.Contains(raw, ">B<") {
		t.Error("node labels missing")
	}
	if !strings.Contains(raw, "<path") {
		t.Error("ribbon path missing")
	}
	if !strings.Contains(raw, "<rect") {
		t.Error("node bar missing")
	}
	assertValidSVG(t, out)
}

func TestRenderMultiColumnChain(t *testing.T) {
	d := &diagram.SankeyDiagram{
		Flows: []diagram.SankeyFlow{
			{Source: "A", Target: "B", Value: 10},
			{Source: "B", Target: "C", Value: 5},
			{Source: "B", Target: "D", Value: 5},
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	for _, n := range []string{"A", "B", "C", "D"} {
		if !strings.Contains(raw, ">"+n+"<") {
			t.Errorf("label %q missing", n)
		}
	}
	assertValidSVG(t, out)
}

func TestRenderFanIn(t *testing.T) {
	d := &diagram.SankeyDiagram{
		Flows: []diagram.SankeyFlow{
			{Source: "A", Target: "C", Value: 3},
			{Source: "B", Target: "C", Value: 7},
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	// Two ribbons expected.
	if n := strings.Count(string(out), "<path"); n != 2 {
		t.Errorf("ribbon count = %d, want 2", n)
	}
	assertValidSVG(t, out)
}

func TestRenderDeterministic(t *testing.T) {
	d := &diagram.SankeyDiagram{
		Flows: []diagram.SankeyFlow{
			{Source: "A", Target: "B", Value: 10},
			{Source: "A", Target: "C", Value: 5},
			{Source: "B", Target: "D", Value: 8},
			{Source: "C", Target: "D", Value: 4},
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

func TestRenderCustomFontSize(t *testing.T) {
	d := &diagram.SankeyDiagram{
		Flows: []diagram.SankeyFlow{{Source: "A", Target: "B", Value: 1}},
	}
	out, err := Render(d, &Options{FontSize: 20})
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if !strings.Contains(string(out), "font-size:20px") {
		t.Error("custom font size not applied")
	}
}

func TestRenderCycleBoundedIterations(t *testing.T) {
	// A -> B -> A creates a cycle. Render must terminate (iteration
	// cap) and still produce a valid SVG with both labels and both
	// ribbons visible.
	d := &diagram.SankeyDiagram{
		Flows: []diagram.SankeyFlow{
			{Source: "A", Target: "B", Value: 1},
			{Source: "B", Target: "A", Value: 1},
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	if !strings.Contains(raw, ">A<") || !strings.Contains(raw, ">B<") {
		t.Error("cycle labels missing")
	}
	if n := strings.Count(raw, "<path"); n != 2 {
		t.Errorf("ribbon count = %d, want 2", n)
	}
	assertValidSVG(t, out)
}

func TestRenderRibbonColorMatchesSourceBar(t *testing.T) {
	// Regression: assignColumns previously sorted `nodes` in place,
	// which split nodeIdx (pre-sort) from the palette index used by
	// node bars (post-sort). Guard: the palette color of node A must
	// be present in both its bar fill and in the fill of every flow
	// originating from A.
	d := &diagram.SankeyDiagram{
		Flows: []diagram.SankeyFlow{
			{Source: "A", Target: "B", Value: 1},
			{Source: "B", Target: "C", Value: 1},
			{Source: "A", Target: "C", Value: 1},
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	colorA := palette[0] // A is the first-appearance node
	// The color must appear in at least 3 fills (A's bar + 2 ribbons).
	if strings.Count(raw, "fill:"+colorA) < 3 {
		t.Errorf("A's color %s should be used for A's bar and both outgoing ribbons; saw %d occurrences\n%s",
			colorA, strings.Count(raw, "fill:"+colorA), raw)
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
