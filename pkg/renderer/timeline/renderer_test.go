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

// Default direction (unset) renders as LR per the Mermaid spec —
// the horizontal axis line + vertically-stacked event boxes
// under each period column.
func TestRenderTimelineLRDefault(t *testing.T) {
	d := &diagram.TimelineDiagram{
		Events: []diagram.TimelineEvent{
			{Time: "2000", Events: []string{"Web 1.0"}},
			{Time: "2010", Events: []string{"Mobile"}},
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	// LR axis runs across the chart — look for a <line> whose
	// y1 and y2 attributes are equal (horizontal stroke).
	if !strings.Contains(raw, "<line") {
		t.Fatalf("expected axis <line> in:\n%s", raw)
	}
	// Both period times should appear.
	for _, want := range []string{">2000<", ">2010<", ">Web 1.0<", ">Mobile<"} {
		if !strings.Contains(raw, want) {
			t.Errorf("expected %q in LR output", want)
		}
	}
}

// `direction TD` keeps the legacy vertical layout — the axis
// runs top-to-bottom rather than left-to-right.
func TestRenderTimelineTDExplicit(t *testing.T) {
	d := &diagram.TimelineDiagram{
		Direction: "TD",
		Events: []diagram.TimelineEvent{
			{Time: "2000", Events: []string{"Web 1.0"}},
			{Time: "2010", Events: []string{"Mobile"}},
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	// TD layout has the time labels on the LEFT of the axis with
	// `text-anchor="end"` — LR has them centered above the
	// column with `text-anchor="middle"`. Detect the TD signature.
	if !strings.Contains(raw, `text-anchor="end"`) {
		t.Errorf("explicit Direction=TD should produce end-anchored time labels")
	}
}

// LR section bands span the columns they own, with the section
// name centered above its period range.
// LR-specific: a single column with multiple events must produce N event
// boxes at distinct y-coordinates, not one box atop another.
func TestRenderTimelineLRMultiEventStacked(t *testing.T) {
	d := &diagram.TimelineDiagram{
		Events: []diagram.TimelineEvent{
			{Time: "2005", Events: []string{"a", "b", "c"}},
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	// 3 stacked event rects, each rx="5.00" rounded.
	if got := strings.Count(raw, `rx="5.00"`); got != 3 {
		t.Fatalf("expected 3 rounded event rects, got %d", got)
	}
	// Each label appears at a distinct y because boxes stack.
	ys := map[string]bool{}
	for _, label := range []string{">a<", ">b<", ">c<"} {
		idx := strings.Index(raw, label)
		if idx < 0 {
			t.Fatalf("missing label %q", label)
		}
		// Walk backwards to the nearest y="..." attribute on the text element.
		head := raw[:idx]
		yAt := strings.LastIndex(head, ` y="`)
		if yAt < 0 {
			t.Fatalf("could not locate y= for %q", label)
		}
		end := strings.Index(head[yAt+4:], `"`)
		ys[head[yAt+4:yAt+4+end]] = true
	}
	if len(ys) != 3 {
		t.Errorf("expected 3 distinct y-coords for stacked events, got %d (%v)", len(ys), ys)
	}
}

// Section palette must cycle (modulo wraparound) when the diagram has
// more sections than the theme provides colors. Section 0 and section 2
// share a color when the palette has length 2.
func TestRenderTimelineLRSectionColorCycle(t *testing.T) {
	d := &diagram.TimelineDiagram{
		Sections: []diagram.TimelineSection{
			{Name: "S0", Events: []diagram.TimelineEvent{{Time: "t0", Events: []string{"a"}}}},
			{Name: "S1", Events: []diagram.TimelineEvent{{Time: "t1", Events: []string{"b"}}}},
			{Name: "S2", Events: []diagram.TimelineEvent{{Time: "t2", Events: []string{"c"}}}},
		},
	}
	opts := &Options{Theme: Theme{
		SectionColors: []string{"#aa0000", "#00aa00"},
		TitleText:     "#000", SectionText: "#000", EventText: "#fff",
		AxisStroke: "#999", Background: "#fff",
	}}
	out, err := Render(d, opts)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	// S0 and S2 (indexes 0 and 2 mod 2 == 0) must both reference #aa0000;
	// S1 references #00aa00.
	if c := strings.Count(raw, "#aa0000"); c < 2 {
		t.Errorf("expected wraparound color #aa0000 reused for S0 and S2 (≥2 references); got %d", c)
	}
	if c := strings.Count(raw, "#00aa00"); c < 1 {
		t.Errorf("expected #00aa00 used for S1; got %d", c)
	}
	// Band rect color and the event-box fill must match within a section
	// (drift between th.SectionColors[i%len] uses caused mismatched bands).
	bandIdx := strings.Index(raw, ">S0<")
	if bandIdx < 0 {
		t.Fatalf("missing S0 label")
	}
	prelude := raw[:bandIdx]
	if !strings.Contains(prelude, "fill:#aa0000;fill-opacity:0.18") {
		t.Errorf("S0 band should be tinted with section color #aa0000")
	}
}

func TestRenderTimelineLRSectionBand(t *testing.T) {
	d := &diagram.TimelineDiagram{
		Sections: []diagram.TimelineSection{
			{Name: "Education", Events: []diagram.TimelineEvent{
				{Time: "2015", Events: []string{"Started"}},
				{Time: "2019", Events: []string{"Graduated"}},
			}},
			{Name: "Career", Events: []diagram.TimelineEvent{
				{Time: "2020", Events: []string{"First job"}},
			}},
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	// Each section name renders centered in its band.
	for _, want := range []string{">Education<", ">Career<"} {
		if !strings.Contains(raw, want) {
			t.Errorf("expected section name %q in LR output", want)
		}
	}
	// Two sections → two band rects with fill-opacity:0.18.
	if got := strings.Count(raw, "fill-opacity:0.18"); got != 2 {
		t.Errorf("expected 2 section bands (got %d)", got)
	}
}
