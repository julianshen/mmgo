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
	if c.Members[0].Name != "name" || c.Members[0].Visibility != diagram.VisibilityPublic {
		t.Errorf("member[0] = %+v", c.Members[0])
	}
	if !c.Members[2].IsMethod {
		t.Error("eat should be a method")
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
