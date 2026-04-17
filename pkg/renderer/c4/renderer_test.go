package c4

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
	d := &diagram.C4Diagram{}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	assertValidSVG(t, out)
}

func TestRenderWithTitle(t *testing.T) {
	d := &diagram.C4Diagram{Title: "System Context"}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if !strings.Contains(string(out), ">System Context<") {
		t.Error("title missing")
	}
	assertValidSVG(t, out)
}

func TestRenderElements(t *testing.T) {
	d := &diagram.C4Diagram{
		Elements: []diagram.C4Element{
			{ID: "u", Kind: diagram.C4ElementPerson, Label: "User", Description: "End user"},
			{ID: "s", Kind: diagram.C4ElementSystem, Label: "System"},
		},
		Relations: []diagram.C4Relation{
			{From: "u", To: "s", Label: "Uses"},
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	for _, want := range []string{">User<", ">System<", ">Uses<", "«Person»", "«System»"} {
		if !strings.Contains(raw, want) {
			t.Errorf("missing %q", want)
		}
	}
	assertValidSVG(t, out)
}

func TestRenderAllKinds(t *testing.T) {
	d := &diagram.C4Diagram{
		Elements: []diagram.C4Element{
			{ID: "a", Kind: diagram.C4ElementPerson, Label: "A"},
			{ID: "b", Kind: diagram.C4ElementPersonExt, Label: "B"},
			{ID: "c", Kind: diagram.C4ElementSystem, Label: "C"},
			{ID: "d", Kind: diagram.C4ElementSystemExt, Label: "D"},
			{ID: "e", Kind: diagram.C4ElementSystemDB, Label: "E"},
			{ID: "f", Kind: diagram.C4ElementContainer, Label: "F", Technology: "Go"},
			{ID: "g", Kind: diagram.C4ElementContainerDB, Label: "G"},
			{ID: "h", Kind: diagram.C4ElementComponent, Label: "H"},
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	assertValidSVG(t, out)
}

func TestRenderRelationWithTechnology(t *testing.T) {
	d := &diagram.C4Diagram{
		Elements: []diagram.C4Element{
			{ID: "a", Kind: diagram.C4ElementSystem, Label: "A"},
			{ID: "b", Kind: diagram.C4ElementSystem, Label: "B"},
		},
		Relations: []diagram.C4Relation{
			{From: "a", To: "b", Label: "Calls", Technology: "HTTP"},
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if !strings.Contains(string(out), "Calls [HTTP]") {
		t.Error("label with technology missing")
	}
	assertValidSVG(t, out)
}

func TestRenderCustomFontSize(t *testing.T) {
	d := &diagram.C4Diagram{
		Elements: []diagram.C4Element{
			{ID: "a", Kind: diagram.C4ElementSystem, Label: "A"},
		},
	}
	out, err := Render(d, &Options{FontSize: 18})
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if !strings.Contains(string(out), "font-size:18px") {
		t.Error("custom font size not applied")
	}
}

func TestRenderMultiPointEdge(t *testing.T) {
	// Force enough elements to get dagre to route edges with waypoints.
	d := &diagram.C4Diagram{
		Elements: []diagram.C4Element{
			{ID: "a", Kind: diagram.C4ElementSystem, Label: "A"},
			{ID: "b", Kind: diagram.C4ElementSystem, Label: "B"},
			{ID: "c", Kind: diagram.C4ElementSystem, Label: "C"},
			{ID: "d", Kind: diagram.C4ElementSystem, Label: "D"},
		},
		Relations: []diagram.C4Relation{
			{From: "a", To: "b"},
			{From: "a", To: "c"},
			{From: "a", To: "d"},
			{From: "b", To: "d"},
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	assertValidSVG(t, out)
}

func TestRenderDeterministic(t *testing.T) {
	d := &diagram.C4Diagram{
		Title: "Test",
		Elements: []diagram.C4Element{
			{ID: "a", Kind: diagram.C4ElementPerson, Label: "A"},
			{ID: "b", Kind: diagram.C4ElementSystem, Label: "B"},
		},
		Relations: []diagram.C4Relation{
			{From: "a", To: "b", Label: "uses"},
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
