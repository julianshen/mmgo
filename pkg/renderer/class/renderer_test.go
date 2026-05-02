package class

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"regexp"
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
	d := &diagram.ClassDiagram{}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	assertValidSVG(t, out)
}

func TestRenderClassWithMembers(t *testing.T) {
	d := &diagram.ClassDiagram{
		Classes: []diagram.ClassDef{
			{
				ID: "Animal", Label: "Animal",
				Members: []diagram.ClassMember{
					{Name: "name", ReturnType: "String", Visibility: diagram.VisibilityPublic},
					{Name: "eat", IsMethod: true, Visibility: diagram.VisibilityPublic, ReturnType: "bool"},
				},
			},
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	if !strings.Contains(raw, ">Animal<") {
		t.Error("class name missing")
	}
	if !strings.Contains(raw, "name") {
		t.Error("member name missing")
	}
	assertValidSVG(t, out)
}

func TestRenderRelationship(t *testing.T) {
	d := &diagram.ClassDiagram{
		Classes: []diagram.ClassDef{
			{ID: "Animal", Label: "Animal"},
			{ID: "Dog", Label: "Dog"},
		},
		Relations: []diagram.ClassRelation{
			{From: "Animal", To: "Dog", RelationType: diagram.RelationTypeInheritance, Label: "extends"},
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	if !strings.Contains(raw, ">Animal<") || !strings.Contains(raw, ">Dog<") {
		t.Error("class labels missing")
	}
	// Inheritance: glyph sits at the parent end (From side), inlined.
	if !strings.Contains(raw, `<g transform="translate(`) {
		t.Error("expected inline start glyph for inheritance")
	}
	if strings.Contains(raw, "marker-end") {
		t.Error("inheritance should not use marker-end")
	}
	assertValidSVG(t, out)
}

func TestRenderAnnotation(t *testing.T) {
	d := &diagram.ClassDiagram{
		Classes: []diagram.ClassDef{
			{ID: "Shape", Label: "Shape", Annotation: diagram.AnnotationInterface},
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	if !strings.Contains(raw, "interface") {
		t.Error("annotation label missing")
	}
	assertValidSVG(t, out)
}

func TestRenderMultipleClasses(t *testing.T) {
	d := &diagram.ClassDiagram{
		Classes: []diagram.ClassDef{
			{ID: "A", Label: "A"},
			{ID: "B", Label: "B"},
			{ID: "C", Label: "C"},
		},
		Relations: []diagram.ClassRelation{
			{From: "A", To: "B", RelationType: diagram.RelationTypeAssociation},
			{From: "B", To: "C", RelationType: diagram.RelationTypeComposition},
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	assertValidSVG(t, out)
}

func TestRenderAllRelationTypes(t *testing.T) {
	// Start-side glyphs (parent/whole on the left of the arrow).
	// End-side glyphs (arrow/dependency target on the right).
	// Link/DashedLink carry no marker.
	// wantPolygonPoints pins the glyph identity so a fill/orientation
	// swap (e.g. composition ↔ aggregation) can't pass.
	cases := []struct {
		rt                diagram.RelationType
		wantStartInline   bool
		wantEndMarker     bool
		wantDashed        bool
		wantPolygonPoints string // unique marker-geometry substring
	}{
		{diagram.RelationTypeInheritance, true, false, false, `points="20,0 0,10 20,20"`},
		{diagram.RelationTypeComposition, true, false, false, `fill:#333;stroke:#333`},
		{diagram.RelationTypeAggregation, true, false, false, `fill:white;stroke:#333;stroke-width:1"`},
		{diagram.RelationTypeRealization, false, true, true, `fill:white;stroke:#333;stroke-width:1.5"`},
		// Dependency and Association share the same arrowhead geometry;
		// they differ only in line style (dashed vs solid). The
		// stroke-dasharray check (wantDashed) covers that distinction;
		// here we just confirm the arrowhead marker fires for both.
		{diagram.RelationTypeDependency, false, true, true, `id="cls-association"`},
		{diagram.RelationTypeAssociation, false, true, false, `id="cls-association"`},
		{diagram.RelationTypeLink, false, false, false, ""},
		{diagram.RelationTypeDashedLink, false, false, true, ""},
	}
	for _, tc := range cases {
		t.Run(tc.rt.String(), func(t *testing.T) {
			d := &diagram.ClassDiagram{
				Classes: []diagram.ClassDef{
					{ID: "A", Label: "A"},
					{ID: "B", Label: "B"},
				},
				Relations: []diagram.ClassRelation{
					{From: "A", To: "B", RelationType: tc.rt},
				},
			}
			out, err := Render(d, nil)
			if err != nil {
				t.Fatalf("Render: %v", err)
			}
			raw := string(out)
			assertValidSVG(t, out)

			hasStartInline := strings.Contains(raw, `<g transform="translate(`)
			if hasStartInline != tc.wantStartInline {
				t.Errorf("%s: start inline glyph: got %v, want %v", tc.rt, hasStartInline, tc.wantStartInline)
			}
			hasEndMarker := strings.Contains(raw, `marker-end="url(#`)
			if hasEndMarker != tc.wantEndMarker {
				t.Errorf("%s: marker-end: got %v, want %v", tc.rt, hasEndMarker, tc.wantEndMarker)
			}
			hasDashed := strings.Contains(raw, "stroke-dasharray")
			if hasDashed != tc.wantDashed {
				t.Errorf("%s: stroke-dasharray: got %v, want %v", tc.rt, hasDashed, tc.wantDashed)
			}
			if tc.wantPolygonPoints != "" && !strings.Contains(raw, tc.wantPolygonPoints) {
				t.Errorf("%s: expected %q in output", tc.rt, tc.wantPolygonPoints)
			}
		})
	}
}

// A diagram with only marker-less relations (Link) should omit <defs>
// entirely — buildDefs returns nil when nothing references end markers.
func TestRenderBuildDefsOmittedWhenUnneeded(t *testing.T) {
	d := &diagram.ClassDiagram{
		Classes: []diagram.ClassDef{
			{ID: "A", Label: "A"}, {ID: "B", Label: "B"},
		},
		Relations: []diagram.ClassRelation{
			{From: "A", To: "B", RelationType: diagram.RelationTypeInheritance},
		},
	}
	out, _ := Render(d, nil)
	if strings.Contains(string(out), "<defs") {
		t.Error("inheritance-only diagram should not emit <defs>")
	}
}

func TestRenderEdgeWithLabel(t *testing.T) {
	d := &diagram.ClassDiagram{
		Classes: []diagram.ClassDef{
			{ID: "A", Label: "A"},
			{ID: "B", Label: "B"},
		},
		Relations: []diagram.ClassRelation{
			{From: "A", To: "B", RelationType: diagram.RelationTypeAssociation, Label: "uses"},
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	if !strings.Contains(raw, ">uses<") {
		t.Error("edge label missing")
	}
	// Chip backdrop: a Rect filled with the theme background sits
	// behind the label so it stays legible against crossing lines.
	bg := DefaultTheme().Background
	if !strings.Contains(raw, fmt.Sprintf(`fill:%s;stroke:none`, bg)) {
		t.Errorf("expected edge-label chip backdrop with fill:%s", bg)
	}
}

// Method args must survive the parser→renderer round-trip. Previously
// the renderer always emitted empty parens; now it includes Args.
func TestRenderMethodArgs(t *testing.T) {
	d := &diagram.ClassDiagram{
		Classes: []diagram.ClassDef{
			{ID: "M", Label: "M", Members: []diagram.ClassMember{
				{Name: "set", Args: "key, value", ReturnType: "void", IsMethod: true, Visibility: diagram.VisibilityPublic},
			}},
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if !strings.Contains(string(out), "+set(key, value) : void") {
		t.Errorf("expected `+set(key, value) : void` in output:\n%s", string(out))
	}
}

func TestRenderMembersWithVisibility(t *testing.T) {
	d := &diagram.ClassDiagram{
		Classes: []diagram.ClassDef{
			{
				ID: "Foo", Label: "Foo",
				Members: []diagram.ClassMember{
					{Name: "pub", Visibility: diagram.VisibilityPublic},
					{Name: "priv", Visibility: diagram.VisibilityPrivate},
					{Name: "prot", Visibility: diagram.VisibilityProtected},
					{Name: "pkg", Visibility: diagram.VisibilityPackage},
					{Name: "run", IsMethod: true, ReturnType: "void"},
				},
			},
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	for _, want := range []string{"+pub", "-priv", "#prot", "~pkg", "run()"} {
		if !strings.Contains(raw, want) {
			t.Errorf("missing %q in output", want)
		}
	}
}

func TestRenderDeterministic(t *testing.T) {
	d := &diagram.ClassDiagram{
		Classes: []diagram.ClassDef{
			{ID: "A", Label: "A", Members: []diagram.ClassMember{{Name: "x"}}},
			{ID: "B", Label: "B"},
		},
		Relations: []diagram.ClassRelation{
			{From: "A", To: "B", RelationType: diagram.RelationTypeInheritance},
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

// Reverse arrows render the relation glyph at the To end (not the
// canonical-forward From end) and skip the `marker-end` SVG reference
// since the glyph is inline-placed instead. This pins the glyph's
// polygon points so a fill/orientation regression can't pass.
func TestRenderReverseRelations(t *testing.T) {
	cases := []struct {
		name      string
		rt        diagram.RelationType
		wantPoly  string // unique polygon-points substring
	}{
		{"inheritance", diagram.RelationTypeInheritance, `points="20,0 0,10 20,20"`},
		{"composition", diagram.RelationTypeComposition, `fill:#333;stroke:#333;stroke-width:1`},
		{"aggregation", diagram.RelationTypeAggregation, `fill:white;stroke:#333;stroke-width:1"`},
		{"realization", diagram.RelationTypeRealization, `fill:white;stroke:#333;stroke-width:1.5`},
		{"association", diagram.RelationTypeAssociation, `points="0,0 20,10 0,20"`},
		{"dependency", diagram.RelationTypeDependency, `points="0,0 20,10 0,20"`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			d := &diagram.ClassDiagram{
				Classes: []diagram.ClassDef{
					{ID: "A", Label: "A"}, {ID: "B", Label: "B"},
				},
				Relations: []diagram.ClassRelation{
					{From: "A", To: "B", RelationType: tc.rt, Direction: diagram.RelationReverse},
				},
			}
			out, err := Render(d, nil)
			if err != nil {
				t.Fatalf("Render: %v", err)
			}
			raw := string(out)
			if !strings.Contains(raw, tc.wantPoly) {
				t.Errorf("missing %q in output", tc.wantPoly)
			}
			if strings.Contains(raw, `marker-end="url(#`) {
				t.Error("reverse edges must not use SVG marker-end")
			}
		})
	}
}

// Bidirectional arrows draw the same glyph at BOTH ends with the
// arrowhead polygon present twice and the two transforms anchored
// at distinct coordinates (otherwise both glyphs would overlap on
// the same endpoint, which is a realistic regression to catch).
func TestRenderBidirectionalAssociation(t *testing.T) {
	d := &diagram.ClassDiagram{
		Classes: []diagram.ClassDef{
			{ID: "A", Label: "A"},
			{ID: "B", Label: "B"},
		},
		Relations: []diagram.ClassRelation{
			{From: "A", To: "B", RelationType: diagram.RelationTypeAssociation, Direction: diagram.RelationBidirectional},
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	if got := strings.Count(raw, `points="0,0 20,10 0,20"`); got != 2 {
		t.Errorf("expected arrowhead polygon twice, got %d", got)
	}
	if strings.Contains(raw, `marker-end="url(#`) {
		t.Error("bidirectional edges must not use SVG marker-end")
	}
	transforms := transformOffsets(t, raw)
	if len(transforms) != 2 {
		t.Fatalf("expected 2 inline glyph transforms, got %d", len(transforms))
	}
	if transforms[0] == transforms[1] {
		t.Errorf("both glyphs anchored at same point %v — they should sit at opposite ends", transforms[0])
	}
}

var translateRe = regexp.MustCompile(`<g transform="translate\(([-\d.]+),([-\d.]+)\)`)

func transformOffsets(t *testing.T, raw string) [][2]string {
	t.Helper()
	matches := translateRe.FindAllStringSubmatch(raw, -1)
	out := make([][2]string, 0, len(matches))
	for _, m := range matches {
		out = append(out, [2]string{m[1], m[2]})
	}
	return out
}

// Direction selects the layout RankDir. The visible signal is the
// viewBox aspect ratio: TB/BT lay classes vertically (taller than
// wide), LR/RL lay them horizontally (wider than tall).
func TestRenderDirection(t *testing.T) {
	cases := []struct {
		dir       diagram.Direction
		widerThan bool // true → expect width > height
	}{
		{diagram.DirectionTB, false},
		{diagram.DirectionBT, false},
		{diagram.DirectionLR, true},
		{diagram.DirectionRL, true},
	}
	for _, tc := range cases {
		t.Run(tc.dir.String(), func(t *testing.T) {
			d := &diagram.ClassDiagram{
				Direction: tc.dir,
				Classes: []diagram.ClassDef{
					{ID: "A", Label: "A"},
					{ID: "B", Label: "B"},
				},
				Relations: []diagram.ClassRelation{
					{From: "A", To: "B", RelationType: diagram.RelationTypeAssociation},
				},
			}
			out, err := Render(d, nil)
			if err != nil {
				t.Fatalf("Render: %v", err)
			}
			w, h := parseViewBoxWH(t, out)
			if tc.widerThan && !(w > h) {
				t.Errorf("%s viewBox should be wider than tall: %fx%f", tc.dir, w, h)
			}
			if !tc.widerThan && !(h > w) {
				t.Errorf("%s viewBox should be taller than wide: %fx%f", tc.dir, w, h)
			}
		})
	}
}

func parseViewBoxWH(t *testing.T, svgBytes []byte) (w, h float64) {
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
		t.Fatalf("invalid SVG: %v", err)
	}
	var minX, minY float64
	if _, err := fmt.Sscanf(doc.ViewBox, "%f %f %f %f", &minX, &minY, &w, &h); err != nil {
		t.Fatalf("viewBox %q unparseable: %v", doc.ViewBox, err)
	}
	return w, h
}

// Generic types render as `Label<Generic>` in the header — Mermaid's
// convention; matches the way TypeScript/Java declare parametric types.
func TestRenderGenericInHeader(t *testing.T) {
	d := &diagram.ClassDiagram{
		Classes: []diagram.ClassDef{
			{ID: "List", Label: "List", Generic: "T"},
			{ID: "Map", Label: "Map", Generic: "K, V"},
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	// XML-escaped angle brackets — what browsers render as `List<T>`.
	if !strings.Contains(raw, "List&lt;T&gt;") {
		t.Errorf("expected `List<T>` header in output")
	}
	if !strings.Contains(raw, "Map&lt;K, V&gt;") {
		t.Errorf("expected `Map<K, V>` header in output")
	}
}

// Custom labels override the ID in the header.
func TestRenderCustomLabel(t *testing.T) {
	d := &diagram.ClassDiagram{
		Classes: []diagram.ClassDef{
			{ID: "Animal", Label: "A friendly animal"},
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	if !strings.Contains(raw, ">A friendly animal<") {
		t.Errorf("expected custom label in header, got:\n%s", raw)
	}
	// The bare ID must not also appear as the header (regression
	// guard: previously the renderer always used Label which equals
	// the ID by default — here the custom label must replace the ID).
	if strings.Contains(raw, ">Animal<") {
		t.Errorf("ID should not be rendered when custom label is set")
	}
}

func TestRenderStaticUnderlined(t *testing.T) {
	d := &diagram.ClassDiagram{
		Classes: []diagram.ClassDef{
			{ID: "C", Label: "C", Members: []diagram.ClassMember{
				{Name: "pi", ReturnType: "double", Visibility: diagram.VisibilityPublic, IsStatic: true},
				{Name: "log", IsMethod: true, ReturnType: "void", Visibility: diagram.VisibilityPublic, IsStatic: true},
				{Name: "instance", Visibility: diagram.VisibilityPublic}, // not static — guard
			}},
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	if got := strings.Count(raw, "text-decoration:underline"); got != 2 {
		t.Errorf("expected 2 underlined static members, got %d", got)
	}
	// `instance` row must NOT carry the underline style.
	if strings.Contains(raw, `text-decoration:underline">+instance`) {
		t.Errorf("non-static member rendered as underlined")
	}
}

func TestRenderAbstractItalic(t *testing.T) {
	d := &diagram.ClassDiagram{
		Classes: []diagram.ClassDef{
			{ID: "Shape", Label: "Shape", Members: []diagram.ClassMember{
				{Name: "draw", IsMethod: true, ReturnType: "void", Visibility: diagram.VisibilityPublic, IsAbstract: true},
				{Name: "color", Visibility: diagram.VisibilityPublic}, // not abstract — guard
			}},
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	// 1 abstract member; the «interface» annotation is also italic
	// but this class has no annotation so any italic must come from
	// the abstract member.
	if got := strings.Count(raw, "font-style:italic"); got != 1 {
		t.Errorf("expected 1 italic abstract member, got %d", got)
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
	d := &diagram.ClassDiagram{
		Classes: []diagram.ClassDef{
			// Include an interface class so the AnnotationText path
			// is exercised.
			{ID: "A", Label: "A", Annotation: diagram.AnnotationInterface},
			{ID: "B", Label: "B"},
		},
		Relations: []diagram.ClassRelation{
			{From: "A", To: "B", RelationType: diagram.RelationTypeAssociation, Label: "uses"},
		},
	}
	out, err := Render(d, &Options{Theme: Theme{
		NodeFill:       "#111111",
		NodeStroke:     "#aabbcc",
		NodeText:       "#ddeeff",
		AnnotationText: "#778899",
		EdgeStroke:     "#223344",
		EdgeText:       "#556677",
		Background:     "#000000",
	}})
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	for _, want := range []string{
		`fill:#000000`,                // background
		`fill:#111111;stroke:#aabbcc`, // class rect
		`fill:#ddeeff`,                // class label
		`fill:#778899`,                // «interface» annotation
		`stroke:#223344`,              // edge line
		`fill:#556677`,                // edge label
	} {
		if !strings.Contains(raw, want) {
			t.Errorf("themed output missing %q", want)
		}
	}
	// Defaults must not leak through when the theme is set.
	for _, unwanted := range []string{`fill:#ECECFF`, `stroke:#9370DB`, `fill:#999`} {
		if strings.Contains(raw, unwanted) {
			t.Errorf("themed output still contains default color %q", unwanted)
		}
	}
}

// Pins the default-theme palette end-to-end so Render(d, nil) paints
// with the exact prior-to-theming colors. Without this, a drift in
// DefaultTheme, resolveTheme, or a Sprintf template could silently
// break every untuned diagram.
func TestRenderDefaultThemeColorsInSVG(t *testing.T) {
	d := &diagram.ClassDiagram{
		Classes: []diagram.ClassDef{
			{ID: "A", Label: "A", Annotation: diagram.AnnotationInterface},
			{ID: "B", Label: "B"},
		},
		Relations: []diagram.ClassRelation{
			{From: "A", To: "B", RelationType: diagram.RelationTypeAssociation, Label: "uses"},
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	for _, want := range []string{
		`fill:#fff;stroke:none`,       // background
		`fill:#ECECFF;stroke:#9370DB`, // class rect
		`fill:#333`,                   // class label + edge label
		`fill:#999`,                   // «interface» annotation
		`stroke:#9370DB`,              // member divider
	} {
		if !strings.Contains(raw, want) {
			t.Errorf("default-theme output missing %q", want)
		}
	}
}

func TestDefaultThemeStable(t *testing.T) {
	got := DefaultTheme()
	want := Theme{
		NodeFill:       "#ECECFF",
		NodeStroke:     "#9370DB",
		NodeText:       "#333",
		AnnotationText: "#999",
		EdgeStroke:     "#333",
		EdgeText:       "#333",
		Background:     "#fff",
	}
	if got != want {
		t.Errorf("DefaultTheme drifted:\n got  %+v\n want %+v", got, want)
	}
}

func TestResolveThemeNilOpts(t *testing.T) {
	if resolveTheme(nil) != DefaultTheme() {
		t.Error("resolveTheme(nil) should return DefaultTheme exactly")
	}
	if resolveTheme(&Options{}) != DefaultTheme() {
		t.Error("resolveTheme with zero-value Options should return DefaultTheme exactly")
	}
}
