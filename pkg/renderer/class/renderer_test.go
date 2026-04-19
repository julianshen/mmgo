package class

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
	cases := []struct {
		rt           diagram.RelationType
		wantStartInline bool
		wantEndMarker bool
		wantDashed    bool
	}{
		{diagram.RelationTypeInheritance, true, false, false},
		{diagram.RelationTypeComposition, true, false, false},
		{diagram.RelationTypeAggregation, true, false, false},
		{diagram.RelationTypeRealization, false, true, true},
		{diagram.RelationTypeDependency, false, true, true},
		{diagram.RelationTypeAssociation, false, true, false},
		{diagram.RelationTypeLink, false, false, false},
		{diagram.RelationTypeDashedLink, false, false, true},
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
		})
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
	if !strings.Contains(string(out), ">uses<") {
		t.Error("edge label missing")
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
