package pie

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
		t.Fatal("expected error for nil diagram")
	}
}

func TestRenderEmptyDiagram(t *testing.T) {
	d := &diagram.PieDiagram{}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	assertValidSVG(t, out)
}

func TestRenderWithTitle(t *testing.T) {
	d := &diagram.PieDiagram{
		Title:  "Pets",
		Slices: []diagram.Slice{{Label: "Dogs", Value: 70}, {Label: "Cats", Value: 30}},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	if !strings.Contains(raw, ">Pets<") {
		t.Error("title missing")
	}
	assertValidSVG(t, out)
}

func TestRenderSliceLabels(t *testing.T) {
	d := &diagram.PieDiagram{
		Slices: []diagram.Slice{
			{Label: "Alpha", Value: 60},
			{Label: "Beta", Value: 40},
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	if !strings.Contains(raw, "Alpha") || !strings.Contains(raw, "Beta") {
		t.Error("slice labels missing")
	}
	assertValidSVG(t, out)
}

func TestRenderShowData(t *testing.T) {
	d := &diagram.PieDiagram{
		ShowData: true,
		Slices:   []diagram.Slice{{Label: "X", Value: 42}},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	if !strings.Contains(raw, "42") {
		t.Error("data value should appear when ShowData is true")
	}
	assertValidSVG(t, out)
}

func TestRenderSingleSlice(t *testing.T) {
	d := &diagram.PieDiagram{
		Slices: []diagram.Slice{{Label: "All", Value: 100}},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	assertValidSVG(t, out)
}

func TestRenderAllZeroValues(t *testing.T) {
	d := &diagram.PieDiagram{
		Slices: []diagram.Slice{{Label: "A", Value: 0}, {Label: "B", Value: 0}},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	assertValidSVG(t, out)
}

func TestRenderDeterministic(t *testing.T) {
	d := &diagram.PieDiagram{
		Title:  "Test",
		Slices: []diagram.Slice{{Label: "A", Value: 50}, {Label: "B", Value: 30}, {Label: "C", Value: 20}},
	}
	first, _ := Render(d, nil)
	for i := 0; i < 10; i++ {
		next, _ := Render(d, nil)
		if string(next) != string(first) {
			t.Fatalf("iter %d: output diverges", i)
		}
	}
}

func TestRenderProducesArcPaths(t *testing.T) {
	d := &diagram.PieDiagram{
		Slices: []diagram.Slice{{Label: "A", Value: 60}, {Label: "B", Value: 40}},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if !strings.Contains(string(out), "<path") {
		t.Error("expected arc <path> elements for pie slices")
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
		t.Fatalf("invalid SVG XML: %v\n%s", err, svgBytes)
	}
	if doc.ViewBox == "" {
		t.Error("viewBox attribute missing")
	}
}
