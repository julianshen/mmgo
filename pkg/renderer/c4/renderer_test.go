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
	// `<` is XML-escaped in text content, so the stereotype tag appears
	// as `&lt;&lt;person&gt;&gt;` in the SVG body.
	for _, want := range []string{">User<", ">System<", ">Uses<", "&lt;&lt;person&gt;&gt;", "&lt;&lt;system&gt;&gt;"} {
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
	raw := string(out)
	// Label and tech now render as separate <text> elements (one per
	// line), matching mmdc.
	if !strings.Contains(raw, ">Calls<") {
		t.Error("label text missing")
	}
	if !strings.Contains(raw, ">[HTTP]<") {
		t.Error("technology text missing on its own line")
	}
	assertValidSVG(t, out)
}

// DB elements (SystemDB / ContainerDB) render as a cylinder, not a
// plain rect — matches mmdc's database glyph.
func TestRenderDBCylinder(t *testing.T) {
	d := &diagram.C4Diagram{
		Elements: []diagram.C4Element{
			{ID: "db", Kind: diagram.C4ElementContainerDB, Label: "Postgres"},
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	if !strings.Contains(raw, "<path") {
		t.Error("expected <path> for cylinder body")
	}
	// Cylinder uses elliptic arcs ('A' command); a plain rect would not.
	if !strings.Contains(raw, " A") {
		t.Error("expected elliptic arc command in cylinder path")
	}
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

func TestRenderAppliesCustomTheme(t *testing.T) {
	d := &diagram.C4Diagram{
		Title: "Test",
		Elements: []diagram.C4Element{
			{ID: "U", Kind: diagram.C4ElementPerson, Label: "User"},
			{ID: "S", Kind: diagram.C4ElementSystem, Label: "Sys"},
		},
		Relations: []diagram.C4Relation{
			{From: "U", To: "S", Label: "uses"},
		},
	}
	th := DefaultTheme()
	th.Background = "#000000"
	th.TitleText = "#ffffff"
	th.EdgeStroke = "#ff00ff"
	th.EdgeText = "#112233"
	// Override just the Person role to prove partial merges work.
	th.Roles[diagram.C4ElementPerson] = RolePalette{Fill: "#aaaaaa", Stroke: "#bbbbbb", Text: "#eeeeee"}

	out, err := Render(d, &Options{Theme: th})
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	for _, want := range []string{
		`fill:#000000`,                // background
		`fill:#ffffff`,                // title text
		`stroke:#ff00ff`,              // edge
		`fill:#112233`,                // edge label
		`fill:#aaaaaa;stroke:#bbbbbb`, // Person role overridden
		`#1168BD`,                     // System role kept (default merge behavior)
	} {
		if !strings.Contains(raw, want) {
			t.Errorf("themed output missing %q", want)
		}
	}
}

func TestDefaultThemeStable(t *testing.T) {
	got := DefaultTheme()
	if got.Roles[diagram.C4ElementPerson].Fill != "#08427B" {
		t.Errorf("Person fill drifted: %q", got.Roles[diagram.C4ElementPerson].Fill)
	}
	if got.Background != "#fff" || got.EdgeStroke != "#333" || got.TitleText != "#333" {
		t.Errorf("chrome drifted: %+v", got)
	}
}
