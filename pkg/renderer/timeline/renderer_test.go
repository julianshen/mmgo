package timeline

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
	d := &diagram.TimelineDiagram{}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	assertValidSVG(t, out)
}

func TestRenderWithTitle(t *testing.T) {
	d := &diagram.TimelineDiagram{Title: "My History"}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if !strings.Contains(string(out), ">My History<") {
		t.Error("title missing")
	}
	assertValidSVG(t, out)
}

func TestRenderTopLevelEvents(t *testing.T) {
	d := &diagram.TimelineDiagram{
		Events: []diagram.TimelineEvent{
			{Time: "1990", Events: []string{"Born"}},
			{Time: "2020", Events: []string{"Graduated", "Moved"}},
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	if !strings.Contains(raw, ">1990<") || !strings.Contains(raw, ">2020<") {
		t.Error("time labels missing")
	}
	if !strings.Contains(raw, "Born") {
		t.Error("event text missing")
	}
	assertValidSVG(t, out)
}

func TestRenderSections(t *testing.T) {
	d := &diagram.TimelineDiagram{
		Title: "Milestones",
		Sections: []diagram.TimelineSection{
			{Name: "2020s", Events: []diagram.TimelineEvent{
				{Time: "2020", Events: []string{"A"}},
			}},
			{Name: "2030s", Events: []diagram.TimelineEvent{
				{Time: "2030", Events: []string{"B"}},
			}},
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	if !strings.Contains(raw, ">2020s<") || !strings.Contains(raw, ">2030s<") {
		t.Error("section names missing")
	}
	assertValidSVG(t, out)
}

func TestRenderDeterministic(t *testing.T) {
	d := &diagram.TimelineDiagram{
		Title: "Test",
		Events: []diagram.TimelineEvent{
			{Time: "A", Events: []string{"x"}},
			{Time: "B", Events: []string{"y"}},
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

func TestRenderAppliesCustomTheme(t *testing.T) {
	d := &diagram.TimelineDiagram{
		Events: []diagram.TimelineEvent{
			{Time: "2024", Events: []string{"E"}},
		},
	}
	out, err := Render(d, &Options{Theme: Theme{
		SectionColors: []string{"#aabbcc"},
		TitleText:     "#112233",
		SectionText:   "#223344",
		EventText:     "#ffeedd",
		AxisStroke:    "#445566",
		Background:    "#000000",
	}})
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	for _, want := range []string{`fill:#000000`, `stroke:#445566`, `fill:#223344`, `fill:#aabbcc`, `fill:#ffeedd`} {
		if !strings.Contains(raw, want) {
			t.Errorf("themed output missing %q", want)
		}
	}
}

func TestDefaultThemeStable(t *testing.T) {
	got := DefaultTheme()
	want := Theme{
		SectionColors: []string{"#4e79a7", "#f28e2b", "#59a14f", "#e15759", "#76b7b2", "#edc948"},
		TitleText:     "#333",
		SectionText:   "#333",
		EventText:     "#fff",
		AxisStroke:    "#999",
		Background:    "#fff",
	}
	if len(got.SectionColors) != len(want.SectionColors) {
		t.Fatalf("SectionColors len drift")
	}
	for i := range got.SectionColors {
		if got.SectionColors[i] != want.SectionColors[i] {
			t.Errorf("SectionColors[%d] drifted: %q vs %q", i, got.SectionColors[i], want.SectionColors[i])
		}
	}
	if got.TitleText != want.TitleText || got.EventText != want.EventText || got.AxisStroke != want.AxisStroke {
		t.Errorf("DefaultTheme drifted: %+v", got)
	}
}

// AccTitle/AccDescr emit as <title>/<desc> SVG children.
func TestRenderAccessibilityFields(t *testing.T) {
	d := &diagram.TimelineDiagram{
		AccTitle: "History of Web",
		AccDescr: "Major web milestones",
		Events: []diagram.TimelineEvent{
			{Time: "2000", Events: []string{"Web 1.0"}},
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	if !strings.Contains(raw, "<title>History of Web</title>") {
		t.Errorf("expected <title> in:\n%s", raw)
	}
	if !strings.Contains(raw, "<desc>Major web milestones</desc>") {
		t.Errorf("expected <desc> in:\n%s", raw)
	}
}

// A period with multiple events renders one box per event,
// stacked under the period header.
func TestRenderMultiEventStacked(t *testing.T) {
	d := &diagram.TimelineDiagram{
		Events: []diagram.TimelineEvent{
			{Time: "2005", Events: []string{"Web 2.0 rise", "Social networks"}},
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	for _, want := range []string{">Web 2.0 rise<", ">Social networks<"} {
		if !strings.Contains(raw, want) {
			t.Errorf("expected %q stacked into separate box, got:\n%s", want, raw)
		}
	}
	// Two events → two rect elements for the period (plus
	// background, axis line; we look for the rounded rect rx="5"
	// signature).
	if got := strings.Count(raw, `rx="5.00"`); got < 2 {
		t.Errorf("expected ≥2 rounded event boxes (rx=5), got %d", got)
	}
}
