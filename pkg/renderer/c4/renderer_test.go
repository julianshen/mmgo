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

// AccTitle/AccDescr emit as <title>/<desc> SVG children.
func TestRenderC4Accessibility(t *testing.T) {
	d := &diagram.C4Diagram{
		Variant:  diagram.C4VariantContext,
		AccTitle: "System view",
		AccDescr: "Top-level architecture",
		Elements: []diagram.C4Element{
			{ID: "u", Kind: diagram.C4ElementPerson, Label: "User"},
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	if !strings.Contains(raw, "<title>System view</title>") {
		t.Errorf("expected <title> in output:\n%s", raw)
	}
	if !strings.Contains(raw, "<desc>Top-level architecture</desc>") {
		t.Errorf("expected <desc> in output:\n%s", raw)
	}
}

// BiRel emits both marker-end and marker-start so the edge reads
// as bidirectional.
func TestRenderC4BiRelMarkers(t *testing.T) {
	d := &diagram.C4Diagram{
		Variant: diagram.C4VariantContext,
		Elements: []diagram.C4Element{
			{ID: "a", Kind: diagram.C4ElementSystem, Label: "A"},
			{ID: "b", Kind: diagram.C4ElementSystem, Label: "B"},
		},
		Relations: []diagram.C4Relation{
			{From: "a", To: "b", Label: "talks to", Direction: diagram.C4RelBi},
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	if !strings.Contains(raw, `marker-start="url(#c4-arrow)"`) {
		t.Errorf("BiRel should emit marker-start:\n%s", raw)
	}
	if !strings.Contains(raw, `marker-end="url(#c4-arrow)"`) {
		t.Errorf("BiRel should still emit marker-end:\n%s", raw)
	}
}

// Queue elements render as a stadium pill (high border-radius rect).
func TestRenderC4QueueShape(t *testing.T) {
	d := &diagram.C4Diagram{
		Variant: diagram.C4VariantContainer,
		Elements: []diagram.C4Element{
			{ID: "q", Kind: diagram.C4ElementContainerQueue, Label: "Events"},
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	// Queue uses rx=h/2 (≥ ~25 typically), much larger than the
	// default 0 — looking for a non-zero rx is enough to confirm.
	if !strings.Contains(string(out), `rx="`) {
		t.Errorf("expected rx attribute on queue rect")
	}
}

// Deployment_Node renders with a dashed border so it visually
// reads as a container of nested elements.
func TestRenderC4DeploymentNodeDashed(t *testing.T) {
	d := &diagram.C4Diagram{
		Variant: diagram.C4VariantDeployment,
		Elements: []diagram.C4Element{
			{ID: "web", Kind: diagram.C4ElementDeploymentNode, Label: "Web tier"},
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if !strings.Contains(string(out), "stroke-dasharray:6 4") {
		t.Errorf("Deployment_Node should render with dashed border")
	}
}

// A boundary block emits a dashed-frame rect plus its `«kind»`
// stereotype label behind the inner elements.
func TestRenderC4BoundaryFrame(t *testing.T) {
	d := &diagram.C4Diagram{
		Variant: diagram.C4VariantContext,
		Elements: []diagram.C4Element{
			{ID: "u", Kind: diagram.C4ElementPerson, Label: "User"},
			{ID: "s", Kind: diagram.C4ElementSystem, Label: "App"},
		},
		Boundaries: []*diagram.C4Boundary{
			{ID: "b", Label: "Bank", Kind: diagram.C4BoundarySystem, Elements: []int{0, 1}},
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	if !strings.Contains(raw, "stroke-dasharray:6 4") {
		t.Errorf("expected dashed boundary frame in:\n%s", raw)
	}
	// XML-escaped form of `Bank <<system_boundary>>` — renderer
	// emits the raw `<<…>>` and the encoder escapes the chevrons.
	if !strings.Contains(raw, "Bank &lt;&lt;system_boundary&gt;&gt;") {
		t.Errorf("expected boundary heading in:\n%s", raw)
	}
}

// A boundary whose every child id misses the layout returns nil
// — the renderer doesn't crash, just doesn't draw a frame.
func TestRenderC4BoundaryEmptyChildren(t *testing.T) {
	d := &diagram.C4Diagram{
		Variant: diagram.C4VariantContext,
		Elements: []diagram.C4Element{
			{ID: "u", Kind: diagram.C4ElementPerson, Label: "User"},
		},
		Boundaries: []*diagram.C4Boundary{
			// Index 99 is out of range; renderer must skip
			// without panicking and emit no boundary frame.
			{ID: "ghost", Label: "Ghost", Kind: diagram.C4BoundaryGeneric, Elements: []int{99}},
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if strings.Contains(string(out), "Ghost") {
		t.Errorf("ghost boundary heading should not appear")
	}
}

// Two-level nested boundaries each emit their own dashed frame
// + heading; both headings appear in the rendered SVG.
func TestRenderC4BoundaryNested(t *testing.T) {
	inner := &diagram.C4Boundary{
		ID: "in", Label: "Inner",
		Kind: diagram.C4BoundarySystem, Elements: []int{0},
	}
	outer := &diagram.C4Boundary{
		ID: "out", Label: "Outer",
		Kind: diagram.C4BoundaryEnterprise,
		Boundaries: []*diagram.C4Boundary{inner},
	}
	d := &diagram.C4Diagram{
		Variant: diagram.C4VariantContainer,
		Elements: []diagram.C4Element{
			{ID: "c", Kind: diagram.C4ElementContainer, Label: "App"},
		},
		Boundaries: []*diagram.C4Boundary{outer},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	for _, want := range []string{
		"Outer &lt;&lt;enterprise_boundary&gt;&gt;",
		"Inner &lt;&lt;system_boundary&gt;&gt;",
	} {
		if !strings.Contains(raw, want) {
			t.Errorf("expected %q in nested-frame output", want)
		}
	}
	// Two dashed frames means two `stroke-dasharray:6 4`.
	if got := strings.Count(raw, "stroke-dasharray:6 4"); got != 2 {
		t.Errorf("expected 2 dashed frames, got %d", got)
	}
}

// A boundary around top/left-most elements would clip past a
// `0 0 W H` viewBox (its frame extends boundaryPad +
// boundaryHeadingPad above/left of the layout's bbox). The
// renderer must adjust the viewBox origin so the frame stays
// visible.
func TestRenderC4BoundaryViewportNoClip(t *testing.T) {
	d := &diagram.C4Diagram{
		Variant: diagram.C4VariantContext,
		Elements: []diagram.C4Element{
			{ID: "u", Kind: diagram.C4ElementPerson, Label: "User"},
		},
		Boundaries: []*diagram.C4Boundary{
			{ID: "b", Label: "Bank", Kind: diagram.C4BoundaryGeneric, Elements: []int{0}},
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	// Extract the viewBox attribute value.
	const marker = `viewBox="`
	i := strings.Index(raw, marker)
	if i < 0 {
		t.Fatalf("viewBox missing")
	}
	rest := raw[i+len(marker):]
	end := strings.Index(rest, `"`)
	vb := rest[:end]
	parts := strings.Fields(vb)
	if len(parts) != 4 {
		t.Fatalf("viewBox shape: %q", vb)
	}
	// minX / minY should be <= 0 — top-left boundary frame
	// extends above and left of the layout's element bounds, so
	// the viewBox origin must absorb that overhead.
	if parts[0] == "0.00" && parts[1] == "0.00" {
		t.Errorf("expected viewBox origin <0 to fit boundary frame, got %q", vb)
	}
}

// `Boundary(b, "Label", "service")` — a generic Boundary with a
// TypeHint third arg — renders that hint as the stereotype
// instead of the default `<<boundary>>`. Dedicated boundary
// kinds keep their own stereotype.
func TestRenderC4BoundaryTypeHint(t *testing.T) {
	cases := []struct {
		kind diagram.C4BoundaryKind
		hint string
		want string
	}{
		{diagram.C4BoundaryGeneric, "service", "service"},
		{diagram.C4BoundaryGeneric, "", "boundary"},
		// System_Boundary keeps its stereotype even with a hint.
		{diagram.C4BoundarySystem, "service", "system_boundary"},
	}
	for _, tc := range cases {
		d := &diagram.C4Diagram{
			Variant: diagram.C4VariantContext,
			Elements: []diagram.C4Element{
				{ID: "u", Kind: diagram.C4ElementPerson, Label: "User"},
			},
			Boundaries: []*diagram.C4Boundary{
				{ID: "b", Label: "B", Kind: tc.kind, TypeHint: tc.hint, Elements: []int{0}},
			},
		}
		out, err := Render(d, nil)
		if err != nil {
			t.Fatalf("kind %v hint %q: %v", tc.kind, tc.hint, err)
		}
		want := "B &lt;&lt;" + tc.want + "&gt;&gt;"
		if !strings.Contains(string(out), want) {
			t.Errorf("kind %v hint %q: expected %q in output", tc.kind, tc.hint, want)
		}
	}
}

// $link= on an element wraps its SVG group in an <a href> so the
// rendered diagram is clickable. Untagged elements stay flat.
func TestRenderElementLinkWrapsInAnchor(t *testing.T) {
	d := &diagram.C4Diagram{
		Variant: diagram.C4VariantContext,
		Elements: []diagram.C4Element{
			{ID: "u", Kind: diagram.C4ElementPerson, Label: "User",
				Link: "https://example.com/user"},
			{ID: "s", Kind: diagram.C4ElementSystem, Label: "System"},
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	if !strings.Contains(raw, `<a href="https://example.com/user">`) {
		t.Errorf("expected <a href> wrap for linked element, got:\n%s", raw)
	}
	// The other element has no Link, so the system rect must NOT be
	// wrapped — count <a> openers.
	if strings.Count(raw, "<a ") != 1 {
		t.Errorf("expected exactly one <a> wrap (only u has Link), got %d", strings.Count(raw, "<a "))
	}
}

// All four named-arg surfaces parsed in Phase 3 round-trip through
// the AST: $tags=, $sprite=, $link= populate fields even though tags
// and sprite aren't visually rendered yet (parity with mmdc which
// also doesn't render them but accepts the input).
func TestRenderElementNamedArgsRoundTrip(t *testing.T) {
	d := &diagram.C4Diagram{
		Variant: diagram.C4VariantContext,
		Elements: []diagram.C4Element{
			{ID: "u", Kind: diagram.C4ElementPerson, Label: "User",
				Description: "Customer",
				Tags:        "external,vip",
				Sprite:      "user_icon",
			},
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	// Description renders.
	if !strings.Contains(string(out), ">Customer<") {
		t.Error("$descr=-equivalent description should render")
	}
}

// Boundary $link= wraps the boundary frame (rect + heading text) in
// <a href>; child elements/boundaries render outside the wrap.
func TestRenderBoundaryLinkWrapsFrame(t *testing.T) {
	d := &diagram.C4Diagram{
		Variant: diagram.C4VariantContainer,
		Boundaries: []*diagram.C4Boundary{
			{
				ID: "shop", Label: "Shop", Kind: diagram.C4BoundarySystem,
				Link:     "https://ops.example.com",
				Elements: []int{0},
			},
		},
		Elements: []diagram.C4Element{
			{ID: "api", Kind: diagram.C4ElementContainer, Label: "API"},
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	if !strings.Contains(raw, `<a href="https://ops.example.com">`) {
		t.Errorf("expected <a href> wrap on boundary frame, got:\n%s", raw)
	}
}

// Empty-string Link must not produce a wrap. Guards against a future
// regression that swaps `if Link != ""` for a truthier check.
func TestRenderEmptyLinkProducesNoWrap(t *testing.T) {
	d := &diagram.C4Diagram{
		Variant: diagram.C4VariantContext,
		Elements: []diagram.C4Element{
			{ID: "u", Kind: diagram.C4ElementPerson, Label: "User", Link: ""},
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if strings.Contains(string(out), "<a ") {
		t.Errorf("empty Link must not produce <a> wrap, got:\n%s", out)
	}
}

// Relation OffsetX/OffsetY shift the label position relative to its
// default mid-curve anchor. A non-zero offset must produce different
// SVG output.
func TestRenderRelationOffsetShiftsLabel(t *testing.T) {
	build := func(ox, oy float64) []byte {
		d := &diagram.C4Diagram{
			Variant: diagram.C4VariantContext,
			Elements: []diagram.C4Element{
				{ID: "u", Kind: diagram.C4ElementPerson, Label: "User"},
				{ID: "s", Kind: diagram.C4ElementSystem, Label: "System"},
			},
			Relations: []diagram.C4Relation{
				{From: "u", To: "s", Label: "uses", OffsetX: ox, OffsetY: oy},
			},
		}
		out, err := Render(d, nil)
		if err != nil {
			t.Fatalf("Render: %v", err)
		}
		return out
	}
	if string(build(0, 0)) == string(build(50, -25)) {
		t.Error("non-zero OffsetX/OffsetY must shift the rendered label")
	}
}

// UpdateElementStyle on a kind recolors every element of that kind.
func TestRenderUpdateElementStyle(t *testing.T) {
	d := &diagram.C4Diagram{
		Variant: diagram.C4VariantContext,
		Elements: []diagram.C4Element{
			{ID: "u", Kind: diagram.C4ElementPerson, Label: "User"},
		},
		ElementStyles: map[string]diagram.C4ElementStyleOverride{
			"person": {BgColor: "#ffeeaa", BorderColor: "#cc0000", FontColor: "#001122"},
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	for _, want := range []string{"#ffeeaa", "#cc0000", "#001122"} {
		if !strings.Contains(raw, want) {
			t.Errorf("UpdateElementStyle override %q missing in:\n%s", want, raw)
		}
	}
}

// UpdateRelStyle on a from->to pair recolors that edge plus its label.
func TestRenderUpdateRelStyle(t *testing.T) {
	d := &diagram.C4Diagram{
		Variant: diagram.C4VariantContext,
		Elements: []diagram.C4Element{
			{ID: "u", Kind: diagram.C4ElementPerson, Label: "User"},
			{ID: "s", Kind: diagram.C4ElementSystem, Label: "System"},
		},
		Relations: []diagram.C4Relation{
			{From: "u", To: "s", Label: "uses"},
		},
		RelStyles: map[string]diagram.C4RelStyleOverride{
			"u->s": {LineColor: "#aa0099", TextColor: "#005577"},
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	for _, want := range []string{"stroke:#aa0099", "fill:#005577"} {
		if !strings.Contains(raw, want) {
			t.Errorf("UpdateRelStyle override %q missing in:\n%s", want, raw)
		}
	}
}
