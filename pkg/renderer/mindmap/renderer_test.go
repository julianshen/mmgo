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
			Text: "Root",
			Children: []*diagram.MindmapNode{
				{Text: "Child1"},
				{Text: "Child2"},
			},
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
	for _, want := range []string{"section-root", "section-0", "section-1"} {
		if !strings.Contains(raw, `class="mindmap-node `+want) {
			t.Errorf("no <g> with class containing %q", want)
		}
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
	if !strings.Contains(raw, "font-weight:bold") {
		t.Error("bold tspan missing font-weight:bold style")
	}
	if !strings.Contains(raw, ">Bold<") {
		t.Error("bold text content missing")
	}
	if !strings.Contains(raw, "> text<") && !strings.Contains(raw, ">text<") {
		t.Error("plain text content missing")
	}
	assertValidSVG(t, out)
}

func TestRenderIconAndClass(t *testing.T) {
	d := &diagram.MindmapDiagram{
		Root: &diagram.MindmapNode{
			Text: "Root",
			Children: []*diagram.MindmapNode{
				{Text: "A", Icon: "fa fa-user", CSSClasses: []string{"urgent"}},
			},
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	if !strings.Contains(raw, `class="mindmap-node section-0 urgent"`) {
		t.Error("no <g> with class containing 'urgent'")
	}
	if !strings.Contains(raw, ">A<") {
		t.Error("child text missing")
	}
	assertValidSVG(t, out)
}

func TestRenderCyclicGraph(t *testing.T) {
	a := &diagram.MindmapNode{Text: "A"}
	b := &diagram.MindmapNode{Text: "B"}
	a.Children = []*diagram.MindmapNode{b}
	b.Children = []*diagram.MindmapNode{a}
	d := &diagram.MindmapDiagram{Root: a}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	assertValidSVG(t, out)
}

func TestRenderEmptyDiagramBounds(t *testing.T) {
	d := &diagram.MindmapDiagram{}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	if !strings.Contains(raw, `viewBox="0 0 100.00 100.00"`) {
		t.Error("empty diagram should have 100x100 viewBox")
	}
}

// AccTitle/AccDescr emit as <title>/<desc> SVG children.
func TestRenderAccessibilityFields(t *testing.T) {
	d := &diagram.MindmapDiagram{
		AccTitle: "Mind Map",
		AccDescr: "Top-level concepts",
		Root: &diagram.MindmapNode{
			ID: "root", Text: "Root", Shape: diagram.MindmapShapeDefault,
			Children: []*diagram.MindmapNode{
				{ID: "a", Text: "A"},
			},
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	if !strings.Contains(raw, "<title>Mind Map</title>") {
		t.Errorf("expected <title>Mind Map</title> in:\n%s", raw)
	}
	if !strings.Contains(raw, "<desc>Top-level concepts</desc>") {
		t.Errorf("expected <desc>Top-level concepts</desc> in:\n%s", raw)
	}
}

// Multi-line labels emit one <text> element per line, stacked
// vertically inside the node, and the node's bounding height
// expands to accommodate every line.
func TestRenderMultiLineLabel(t *testing.T) {
	d := &diagram.MindmapDiagram{
		Root: &diagram.MindmapNode{
			Text: "Line one\nLine two\nLine three",
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	for _, want := range []string{">Line one<", ">Line two<", ">Line three<"} {
		if !strings.Contains(raw, want) {
			t.Errorf("expected %q in:\n%s", want, raw)
		}
	}
	// Three lines must produce three <text> elements with the
	// same class context — count occurrences of `>Line ` to
	// confirm none collapsed into a single element.
	if got := strings.Count(raw, ">Line "); got != 3 {
		t.Errorf("expected 3 line emissions, got %d", got)
	}
}

// An icon decoration on a node renders as a muted italic caption
// inside the node, rather than being silently dropped.
func TestRenderIconCaption(t *testing.T) {
	d := &diagram.MindmapDiagram{
		Root: &diagram.MindmapNode{
			Text: "Library",
			Icon: "fa fa-book",
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	if !strings.Contains(raw, ">fa fa-book<") {
		t.Errorf("expected icon class string emitted in:\n%s", raw)
	}
	if !strings.Contains(raw, "font-style:italic") {
		t.Errorf("expected italic styling on icon caption")
	}
}

// classDef CSS is merged onto the node's shape style so authors
// see the override in the rendered SVG.
func TestRenderClassDefStyle(t *testing.T) {
	d := &diagram.MindmapDiagram{
		CSSClasses: map[string]string{"hot": "fill:#f00"},
		Root: &diagram.MindmapNode{
			Text: "Root",
			Children: []*diagram.MindmapNode{
				{Text: "A", CSSClasses: []string{"hot"}},
			},
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if !strings.Contains(string(out), "fill:#f00") {
		t.Errorf("expected classDef fill in output")
	}
}

// `style id css` overrides the theme fill on a specific node by ID.
func TestRenderStyleRule(t *testing.T) {
	d := &diagram.MindmapDiagram{
		Styles: []diagram.MindmapStyleDef{{NodeID: "body", CSS: "stroke:#900;stroke-width:3"}},
		Root: &diagram.MindmapNode{
			Text: "Root",
			Children: []*diagram.MindmapNode{
				{ID: "body", Text: "Body"},
			},
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	if !strings.Contains(raw, "stroke:#900") {
		t.Errorf("expected style override stroke in output")
	}
}
