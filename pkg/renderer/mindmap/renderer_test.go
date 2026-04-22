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
				{Text: "Ci", Shape: diagram.MindmapShapeCircle},
				{Text: "Cl", Shape: diagram.MindmapShapeCloud},
				{Text: "B", Shape: diagram.MindmapShapeBang},
				{Text: "H", Shape: diagram.MindmapShapeHexagon},
			},
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	for _, name := range []string{"Center", "R", "S", "Ci", "Cl", "B", "H"} {
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

func TestRenderAppliesCustomTheme(t *testing.T) {
	d := &diagram.MindmapDiagram{
		Root: &diagram.MindmapNode{
			Text:     "Root",
			Children: []*diagram.MindmapNode{{Text: "Child"}},
		},
	}
	out, err := Render(d, &Options{Theme: Theme{
		SectionColors: []string{"#aabbcc", "#ccbbaa"},
		RootColor:     "#112233",
		NodeText:      "#445566",
		RootText:      "#778899",
		EdgeStroke:    "#aabbcc",
		Background:    "#000000",
	}})
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	for _, want := range []string{`#aabbcc`, `#ccbbaa`, `#112233`, `#000000`} {
		if !strings.Contains(raw, want) {
			t.Errorf("themed output missing %q", want)
		}
	}
}

func TestDefaultThemeStable(t *testing.T) {
	got := DefaultTheme()
	wantSections := []string{"#f28e2b", "#e15759", "#76b7b2", "#59a14f", "#edc948", "#b07aa1"}
	if len(got.SectionColors) != len(wantSections) {
		t.Fatalf("SectionColors len drift: %d != %d", len(got.SectionColors), len(wantSections))
	}
	for i, c := range wantSections {
		if got.SectionColors[i] != c {
			t.Errorf("SectionColors[%d] = %q, want %q", i, got.SectionColors[i], c)
		}
	}
	if got.RootColor != "#4e79a7" || got.NodeText != "#fff" || got.RootText != "#fff" || got.EdgeStroke != "#999" || got.Background != "#fff" {
		t.Errorf("DefaultTheme drifted: %+v", got)
	}
}

func TestRenderAccessibility(t *testing.T) {
	d := &diagram.MindmapDiagram{
		Root: &diagram.MindmapNode{Text: "A"},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	if !strings.Contains(raw, `role="graphics-document document"`) {
		t.Error("missing role attribute")
	}
	if !strings.Contains(raw, `aria-roledescription="mindmap"`) {
		t.Error("missing aria-roledescription attribute")
	}
}

func TestRenderSectionGrouping(t *testing.T) {
	d := &diagram.MindmapDiagram{
		Root: &diagram.MindmapNode{
			Text: "Root",
			Children: []*diagram.MindmapNode{
				{Text: "A"},
				{Text: "B"},
			},
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	if !strings.Contains(raw, "section-root") {
		t.Error("missing section-root class")
	}
	if !strings.Contains(raw, "section-0") {
		t.Error("missing section-0 class")
	}
	if !strings.Contains(raw, "section-1") {
		t.Error("missing section-1 class")
	}
}

func TestRenderMarkdownBold(t *testing.T) {
	d := &diagram.MindmapDiagram{
		Root: &diagram.MindmapNode{Text: "**Bold** text"},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	if !strings.Contains(raw, "Bold") {
		t.Error("bold text content missing")
	}
	if !strings.Contains(raw, "text") {
		t.Error("plain text content missing")
	}
	assertValidSVG(t, out)
}

func TestRenderIconAndClass(t *testing.T) {
	d := &diagram.MindmapDiagram{
		Root: &diagram.MindmapNode{
			Text: "Root",
			Children: []*diagram.MindmapNode{
				{Text: "A", Icon: "fa fa-user", Class: "urgent"},
			},
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	if !strings.Contains(raw, "urgent") {
		t.Error("custom class not in output")
	}
	if !strings.Contains(raw, ">A<") {
		t.Error("child text missing")
	}
	assertValidSVG(t, out)
}
