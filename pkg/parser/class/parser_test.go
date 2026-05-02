package class

import (
	"fmt"
	"strings"
	"testing"

	"github.com/julianshen/mmgo/pkg/diagram"
)

func TestParseHeaderRequired(t *testing.T) {
	_, err := Parse(strings.NewReader("class Animal"))
	if err == nil {
		t.Fatal("expected error for missing header")
	}
}

func TestParseEmptyDiagram(t *testing.T) {
	d, err := Parse(strings.NewReader("classDiagram"))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(d.Classes) != 0 || len(d.Relations) != 0 {
		t.Errorf("empty diagram: %+v", d)
	}
}

func TestParseClassWithMembers(t *testing.T) {
	input := `classDiagram
    class Animal {
        +String name
        +int age
        +eat(food) bool
    }`
	d, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(d.Classes) != 1 {
		t.Fatalf("want 1 class, got %d", len(d.Classes))
	}
	c := d.Classes[0]
	if c.ID != "Animal" {
		t.Errorf("ID = %q", c.ID)
	}
	if len(c.Members) != 3 {
		t.Fatalf("want 3 members, got %d", len(c.Members))
	}
	// Fields are stored verbatim (post-visibility) so the renderer can
	// preserve the source's chosen "type name" or "name: type" ordering.
	if c.Members[0].Name != "String name" || c.Members[0].Visibility != diagram.VisibilityPublic {
		t.Errorf("member[0] = %+v", c.Members[0])
	}
	if !c.Members[2].IsMethod {
		t.Error("eat should be a method")
	}
	if c.Members[2].Name != "eat" || c.Members[2].Args != "food" || c.Members[2].ReturnType != "bool" {
		t.Errorf("member[2] = %+v", c.Members[2])
	}
}

// Regression: `name: type` field syntax was previously mangled by a
// whitespace split that dropped the colon onto the wrong half — the mvc
// example rendered `-template: String` as `-String : template:`.
func TestParseFieldColonSyntax(t *testing.T) {
	input := `classDiagram
    class View {
        -template: String
    }`
	d, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	got := d.Classes[0].Members[0]
	if got.Name != "template: String" || got.IsMethod || got.ReturnType != "" {
		t.Errorf("member = %+v, want Name=\"template: String\"", got)
	}
}

// Regression: method arguments were dropped — the parser only captured
// the name before `(` and never read the text inside the parens.
func TestParseMethodArgsPreserved(t *testing.T) {
	cases := []struct {
		src      string
		wantName string
		wantArgs string
		wantRet  string
	}{
		{"+get(key) Value", "get", "key", "Value"},
		{"+set(key, value) void", "set", "key, value", "void"},
		{"+render(data): String", "render", "data", "String"},
		{"+save()", "save", "", ""},
		// Nested parens in args (lambda or grouped expression style)
		// must not be truncated at the first inner `)`.
		{"+execute(callback (x, y)) void", "execute", "callback (x, y)", "void"},
	}
	for _, tc := range cases {
		t.Run(tc.src, func(t *testing.T) {
			input := "classDiagram\n    class C {\n        " + tc.src + "\n    }"
			d, err := Parse(strings.NewReader(input))
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			m := d.Classes[0].Members[0]
			if !m.IsMethod {
				t.Fatalf("expected method, got %+v", m)
			}
			if m.Name != tc.wantName || m.Args != tc.wantArgs || m.ReturnType != tc.wantRet {
				t.Errorf("got Name=%q Args=%q ReturnType=%q; want %q/%q/%q",
					m.Name, m.Args, m.ReturnType,
					tc.wantName, tc.wantArgs, tc.wantRet)
			}
		})
	}
}

func TestParseRelationships(t *testing.T) {
	cases := []struct {
		src  string
		want diagram.RelationType
	}{
		{"Animal <|-- Dog", diagram.RelationTypeInheritance},
		{"Car *-- Engine", diagram.RelationTypeComposition},
		{"Department o-- Employee", diagram.RelationTypeAggregation},
		{"Student --> Course", diagram.RelationTypeAssociation},
		{"Class ..> Interface", diagram.RelationTypeDependency},
		{"Shape ..|> Drawable", diagram.RelationTypeRealization},
		{"A -- B", diagram.RelationTypeLink},
		{"A .. B", diagram.RelationTypeDashedLink},
	}
	for _, tc := range cases {
		t.Run(tc.want.String(), func(t *testing.T) {
			input := "classDiagram\n    " + tc.src
			d, err := Parse(strings.NewReader(input))
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			if len(d.Relations) != 1 {
				t.Fatalf("want 1 relation, got %d", len(d.Relations))
			}
			if d.Relations[0].RelationType != tc.want {
				t.Errorf("type = %v, want %v", d.Relations[0].RelationType, tc.want)
			}
		})
	}
}

func TestParseRelationWithLabel(t *testing.T) {
	input := `classDiagram
    Animal <|-- Dog : extends`
	d, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if d.Relations[0].Label != "extends" {
		t.Errorf("label = %q, want extends", d.Relations[0].Label)
	}
}

func TestParseRelationWithCardinality(t *testing.T) {
	input := `classDiagram
    Customer "1" --> "0..*" Order : places`
	d, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	r := d.Relations[0]
	if r.FromCardinality != "1" || r.ToCardinality != "0..*" {
		t.Errorf("cardinality = %q/%q", r.FromCardinality, r.ToCardinality)
	}
	if r.Label != "places" {
		t.Errorf("label = %q", r.Label)
	}
}

func TestParseAnnotation(t *testing.T) {
	input := `classDiagram
    class Shape {
        <<interface>>
        +draw()
    }`
	d, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if d.Classes[0].Annotation != diagram.AnnotationInterface {
		t.Errorf("annotation = %v, want interface", d.Classes[0].Annotation)
	}
}

func TestParseInlineClass(t *testing.T) {
	input := `classDiagram
    Animal <|-- Dog
    Animal <|-- Cat`
	d, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(d.Classes) < 3 {
		t.Errorf("should auto-register Animal, Dog, Cat: got %d classes", len(d.Classes))
	}
}

func TestParseComments(t *testing.T) {
	input := `classDiagram
    %% this is a comment
    class A {
        +name %% trailing
    }`
	d, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(d.Classes) != 1 {
		t.Fatalf("want 1 class, got %d", len(d.Classes))
	}
}

func TestParseAllAnnotations(t *testing.T) {
	for _, tc := range []struct {
		ann  string
		want diagram.ClassAnnotation
	}{
		{"interface", diagram.AnnotationInterface},
		{"abstract", diagram.AnnotationAbstract},
		{"service", diagram.AnnotationService},
		{"enum", diagram.AnnotationEnum},
	} {
		t.Run(tc.ann, func(t *testing.T) {
			input := fmt.Sprintf("classDiagram\n    class X {\n        <<%s>>\n    }", tc.ann)
			d, err := Parse(strings.NewReader(input))
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			if d.Classes[0].Annotation != tc.want {
				t.Errorf("got %v, want %v", d.Classes[0].Annotation, tc.want)
			}
		})
	}
}

func TestParseUnclosedClassBody(t *testing.T) {
	input := "classDiagram\n    class X {\n        +name"
	_, err := Parse(strings.NewReader(input))
	if err == nil {
		t.Fatal("expected error for unclosed class body")
	}
}

func TestParseClassNoSpaceBeforeBrace(t *testing.T) {
	input := "classDiagram\n    class Animal{\n        +name\n    }"
	d, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(d.Classes) != 1 || d.Classes[0].ID != "Animal" {
		t.Errorf("got %+v", d.Classes)
	}
	if len(d.Classes[0].Members) != 1 {
		t.Errorf("want 1 member, got %d", len(d.Classes[0].Members))
	}
}

func TestParseBareClassDeclaration(t *testing.T) {
	input := "classDiagram\n    class Animal"
	d, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(d.Classes) != 1 || d.Classes[0].ID != "Animal" {
		t.Errorf("got %+v", d.Classes)
	}
}

// Reverse-direction arrows: Mermaid accepts the relation glyph on either
// end. Internally we keep RelationType independent of source direction and
// expose the placement via Reverse so the renderer can decide which end
// gets the glyph.
func TestParseReverseRelations(t *testing.T) {
	cases := []struct {
		src     string
		want    diagram.RelationType
		reverse bool
	}{
		{"Dog --|> Animal", diagram.RelationTypeInheritance, true},
		{"Animal <|-- Dog", diagram.RelationTypeInheritance, false},
		{"Engine --* Car", diagram.RelationTypeComposition, true},
		{"Employee --o Department", diagram.RelationTypeAggregation, true},
		{"Course <-- Student", diagram.RelationTypeAssociation, true},
		{"Interface <.. Class", diagram.RelationTypeDependency, true},
		{"Drawable <|.. Shape", diagram.RelationTypeRealization, true},
	}
	for _, tc := range cases {
		t.Run(tc.src, func(t *testing.T) {
			d, err := Parse(strings.NewReader("classDiagram\n    " + tc.src))
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			if len(d.Relations) != 1 {
				t.Fatalf("want 1 relation, got %d", len(d.Relations))
			}
			r := d.Relations[0]
			if r.RelationType != tc.want {
				t.Errorf("type = %v, want %v", r.RelationType, tc.want)
			}
			if r.Reverse != tc.reverse {
				t.Errorf("reverse = %v, want %v", r.Reverse, tc.reverse)
			}
			if r.Bidirectional {
				t.Errorf("unexpected bidirectional=true on %q", tc.src)
			}
		})
	}
}

// Two-way arrows: `<|--|>`, `*--*`, `o--o`, `<-->`, `<..>`, `<|..|>`.
// They represent bidirectional/symmetric relationships and the renderer
// draws a glyph on both ends.
func TestParseBidirectionalRelations(t *testing.T) {
	cases := []struct {
		src  string
		want diagram.RelationType
	}{
		{"A <|--|> B", diagram.RelationTypeInheritance},
		{"A *--* B", diagram.RelationTypeComposition},
		{"A o--o B", diagram.RelationTypeAggregation},
		{"A <--> B", diagram.RelationTypeAssociation},
		{"A <..> B", diagram.RelationTypeDependency},
		{"A <|..|> B", diagram.RelationTypeRealization},
	}
	for _, tc := range cases {
		t.Run(tc.src, func(t *testing.T) {
			d, err := Parse(strings.NewReader("classDiagram\n    " + tc.src))
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			if len(d.Relations) != 1 {
				t.Fatalf("want 1 relation, got %d", len(d.Relations))
			}
			r := d.Relations[0]
			if r.RelationType != tc.want {
				t.Errorf("type = %v, want %v", r.RelationType, tc.want)
			}
			if !r.Bidirectional {
				t.Errorf("Bidirectional should be true")
			}
			// Reverse is meaningless when Bidirectional is set.
			if r.Reverse {
				t.Errorf("Reverse must be false when Bidirectional is set")
			}
		})
	}
}

// Reverse arrows must coexist with cardinality and label syntax.
func TestParseReverseWithLabelAndCardinality(t *testing.T) {
	d, err := Parse(strings.NewReader(`classDiagram
    Order "0..*" <-- "1" Customer : placed by`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(d.Relations) != 1 {
		t.Fatalf("want 1 relation, got %d", len(d.Relations))
	}
	r := d.Relations[0]
	if r.RelationType != diagram.RelationTypeAssociation {
		t.Errorf("type = %v", r.RelationType)
	}
	if !r.Reverse {
		t.Errorf("reverse should be true")
	}
	if r.From != "Order" || r.To != "Customer" {
		t.Errorf("From/To = %q/%q", r.From, r.To)
	}
	if r.FromCardinality != "0..*" || r.ToCardinality != "1" {
		t.Errorf("cardinality = %q/%q", r.FromCardinality, r.ToCardinality)
	}
	if r.Label != "placed by" {
		t.Errorf("label = %q", r.Label)
	}
}

func TestParseDirection(t *testing.T) {
	cases := []struct {
		src  string
		want diagram.Direction
	}{
		{"direction TB", diagram.DirectionTB},
		{"direction BT", diagram.DirectionBT},
		{"direction LR", diagram.DirectionLR},
		{"direction RL", diagram.DirectionRL},
	}
	for _, tc := range cases {
		t.Run(tc.src, func(t *testing.T) {
			d, err := Parse(strings.NewReader("classDiagram\n    " + tc.src + "\n    class A"))
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			if d.Direction != tc.want {
				t.Errorf("direction = %v, want %v", d.Direction, tc.want)
			}
		})
	}
}

func TestParseDirectionDefault(t *testing.T) {
	d, err := Parse(strings.NewReader("classDiagram\n    class A"))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if d.Direction != diagram.DirectionUnknown {
		t.Errorf("direction = %v, want unknown (default)", d.Direction)
	}
}

func TestParseVisibilityMarkers(t *testing.T) {
	input := `classDiagram
    class Foo {
        +public
        -private
        #protected
        ~package
        noVisibility
    }`
	d, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	members := d.Classes[0].Members
	want := []diagram.Visibility{
		diagram.VisibilityPublic,
		diagram.VisibilityPrivate,
		diagram.VisibilityProtected,
		diagram.VisibilityPackage,
		diagram.VisibilityNone,
	}
	if len(members) != len(want) {
		t.Fatalf("got %d members, want %d", len(members), len(want))
	}
	for i, w := range want {
		if members[i].Visibility != w {
			t.Errorf("member[%d].Visibility = %v, want %v", i, members[i].Visibility, w)
		}
	}
}
