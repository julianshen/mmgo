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
// end. Internally we keep RelationType independent of source direction
// and expose the placement via Direction so the renderer can decide
// which end gets the glyph.
func TestParseReverseRelations(t *testing.T) {
	cases := []struct {
		src     string
		want    diagram.RelationType
		wantDir diagram.RelationDirection
	}{
		{"Dog --|> Animal", diagram.RelationTypeInheritance, diagram.RelationReverse},
		{"Animal <|-- Dog", diagram.RelationTypeInheritance, diagram.RelationForward},
		{"Engine --* Car", diagram.RelationTypeComposition, diagram.RelationReverse},
		{"Employee --o Department", diagram.RelationTypeAggregation, diagram.RelationReverse},
		{"Course <-- Student", diagram.RelationTypeAssociation, diagram.RelationReverse},
		{"Interface <.. Class", diagram.RelationTypeDependency, diagram.RelationReverse},
		{"Drawable <|.. Shape", diagram.RelationTypeRealization, diagram.RelationReverse},
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
			if r.Direction != tc.wantDir {
				t.Errorf("direction = %v, want %v", r.Direction, tc.wantDir)
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
			if r.Direction != diagram.RelationBidirectional {
				t.Errorf("direction = %v, want bidirectional", r.Direction)
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
	if r.Direction != diagram.RelationReverse {
		t.Errorf("direction = %v, want reverse", r.Direction)
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

// Mixed glyphs (e.g. `<|--*`, `*--o`) can't be resolved to a single
// relation kind. Once an arrow is recognised, Mermaid's grammar
// requires the glyph kinds to agree; mismatches must surface as a
// parse error rather than silently dropping the line.
func TestParseMixedGlyphsError(t *testing.T) {
	for _, src := range []string{
		"A <|--* B",
		"A *--o B",
		"A <|--> B",
	} {
		t.Run(src, func(t *testing.T) {
			_, err := Parse(strings.NewReader("classDiagram\n    " + src))
			if err == nil {
				t.Errorf("expected parse error for mismatched arrow %q", src)
			}
		})
	}
}

// Empty endpoints (`--> Dog`, `Animal <|--`) are ambiguous; surface
// the error rather than silently dropping.
func TestParseRelationMissingEndpointError(t *testing.T) {
	for _, src := range []string{
		"--> Dog",
		"Animal <|--",
	} {
		t.Run(src, func(t *testing.T) {
			_, err := Parse(strings.NewReader("classDiagram\n    " + src))
			if err == nil {
				t.Errorf("expected parse error for %q", src)
			}
		})
	}
}

// Whitespace is not part of the arrow grammar; Mermaid accepts
// `Animal<|--Dog` exactly the same as `Animal <|-- Dog`.
func TestParseArrowNoWhitespace(t *testing.T) {
	d, err := Parse(strings.NewReader("classDiagram\n    Animal<|--Dog"))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(d.Relations) != 1 {
		t.Fatalf("want 1 relation, got %d", len(d.Relations))
	}
	r := d.Relations[0]
	if r.From != "Animal" || r.To != "Dog" {
		t.Errorf("From/To = %q/%q", r.From, r.To)
	}
	if r.RelationType != diagram.RelationTypeInheritance || r.Direction != diagram.RelationForward {
		t.Errorf("type=%v dir=%v", r.RelationType, r.Direction)
	}
}

// Multi-dash / multi-dot runs (`A --- B`, `A .... B`) are accepted
// by mermaid-cli for style-only emphasis; they still classify as
// the same Link / DashedLink relations.
func TestParseMultiDashRuns(t *testing.T) {
	cases := []struct {
		src  string
		want diagram.RelationType
	}{
		{"A --- B", diagram.RelationTypeLink},
		{"A ---- B", diagram.RelationTypeLink},
		{"A ... B", diagram.RelationTypeDashedLink},
		{"A .... B", diagram.RelationTypeDashedLink},
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
			if d.Relations[0].RelationType != tc.want {
				t.Errorf("type = %v, want %v", d.Relations[0].RelationType, tc.want)
			}
		})
	}
}

// Bidirectional arrows must coexist with cardinality + label syntax.
func TestParseBidirectionalWithCardinality(t *testing.T) {
	d, err := Parse(strings.NewReader(`classDiagram
    A "1" <--> "*" B : knows`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(d.Relations) != 1 {
		t.Fatalf("want 1 relation, got %d", len(d.Relations))
	}
	r := d.Relations[0]
	if r.Direction != diagram.RelationBidirectional {
		t.Errorf("direction = %v, want bidirectional", r.Direction)
	}
	if r.FromCardinality != "1" || r.ToCardinality != "*" {
		t.Errorf("cardinality = %q/%q", r.FromCardinality, r.ToCardinality)
	}
	if r.Label != "knows" {
		t.Errorf("label = %q", r.Label)
	}
}

func TestParseGenericClass(t *testing.T) {
	cases := []struct {
		src         string
		wantID      string
		wantGeneric string
	}{
		{"class List~T~", "List", "T"},
		{"class Map~K, V~", "Map", "K, V"},
		{"class Wrapper~List~int~~", "Wrapper", "List~int~"},
		{"class List~T~ {\n        +get(i) T\n    }", "List", "T"},
	}
	for _, tc := range cases {
		t.Run(tc.wantID+"_"+tc.wantGeneric, func(t *testing.T) {
			d, err := Parse(strings.NewReader("classDiagram\n    " + tc.src))
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			if len(d.Classes) != 1 {
				t.Fatalf("want 1 class, got %d", len(d.Classes))
			}
			c := d.Classes[0]
			if c.ID != tc.wantID {
				t.Errorf("ID = %q, want %q", c.ID, tc.wantID)
			}
			if c.Generic != tc.wantGeneric {
				t.Errorf("Generic = %q, want %q", c.Generic, tc.wantGeneric)
			}
		})
	}
}

// Generics on the class header must not leak into relation lookups —
// `List~T~ <|-- ArrayList` references the bare ID "List".
func TestParseGenericInRelation(t *testing.T) {
	d, err := Parse(strings.NewReader(`classDiagram
    class List~T~
    class ArrayList~T~
    List <|-- ArrayList`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(d.Classes) != 2 {
		t.Fatalf("want 2 classes, got %d", len(d.Classes))
	}
	if len(d.Relations) != 1 {
		t.Fatalf("want 1 relation, got %d", len(d.Relations))
	}
	r := d.Relations[0]
	if r.From != "List" || r.To != "ArrayList" {
		t.Errorf("From/To = %q/%q", r.From, r.To)
	}
}

func TestParseStaticMember(t *testing.T) {
	d, err := Parse(strings.NewReader(`classDiagram
    class C {
        +pi$ double
        +log()$ void
    }`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	members := d.Classes[0].Members
	if len(members) != 2 {
		t.Fatalf("want 2 members, got %d", len(members))
	}
	for i, m := range members {
		if !m.IsStatic {
			t.Errorf("member[%d] IsStatic = false", i)
		}
		if m.IsAbstract {
			t.Errorf("member[%d] IsAbstract = true (unexpected)", i)
		}
	}
	// The `$` marker must not leak into the rendered name/return type.
	if strings.Contains(members[0].Name, "$") {
		t.Errorf("member[0].Name still contains $: %q", members[0].Name)
	}
	if members[1].ReturnType == "" || strings.Contains(members[1].ReturnType, "$") {
		t.Errorf("member[1] ReturnType=%q, expected `void`", members[1].ReturnType)
	}
}

func TestParseAbstractMember(t *testing.T) {
	d, err := Parse(strings.NewReader(`classDiagram
    class Shape {
        +draw()* void
        +area()*
    }`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	members := d.Classes[0].Members
	if len(members) != 2 {
		t.Fatalf("want 2 members, got %d", len(members))
	}
	for i, m := range members {
		if !m.IsAbstract {
			t.Errorf("member[%d] IsAbstract = false", i)
		}
		if m.IsStatic {
			t.Errorf("member[%d] IsStatic = true (unexpected)", i)
		}
		if strings.Contains(m.Name, "*") {
			t.Errorf("member[%d].Name still contains *: %q", i, m.Name)
		}
	}
}

func TestParseCustomLabel(t *testing.T) {
	d, err := Parse(strings.NewReader(`classDiagram
    class Animal["A friendly animal"]
    class Dog["A man's best friend"] {
        +bark()
    }`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(d.Classes) != 2 {
		t.Fatalf("want 2 classes, got %d", len(d.Classes))
	}
	if d.Classes[0].ID != "Animal" || d.Classes[0].Label != "A friendly animal" {
		t.Errorf("class[0]: ID=%q Label=%q", d.Classes[0].ID, d.Classes[0].Label)
	}
	if d.Classes[1].ID != "Dog" || d.Classes[1].Label != "A man's best friend" {
		t.Errorf("class[1]: ID=%q Label=%q", d.Classes[1].ID, d.Classes[1].Label)
	}
	if len(d.Classes[1].Members) != 1 {
		t.Errorf("class[1] members lost: %v", d.Classes[1].Members)
	}
}

// Custom label on a class also-used in a relation: relation must
// resolve to the bare ID, not the labeled form.
func TestParseCustomLabelWithRelation(t *testing.T) {
	d, err := Parse(strings.NewReader(`classDiagram
    class Animal["Animal label"]
    class Dog
    Animal <|-- Dog`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if d.Relations[0].From != "Animal" || d.Relations[0].To != "Dog" {
		t.Errorf("relation = %+v", d.Relations[0])
	}
}

// Malformed class headers should error rather than swallow the brackets/
// tildes into the parsed ID and trigger mysterious lookup failures.
func TestParseClassHeaderErrors(t *testing.T) {
	for _, src := range []string{
		"class Foo[no quotes]",         // bracketed but unquoted
		"class Foo[unclosed",           // no `]`
		"class Foo~T",                  // unmatched `~`
	} {
		t.Run(src, func(t *testing.T) {
			_, err := Parse(strings.NewReader("classDiagram\n    " + src))
			if err == nil {
				t.Errorf("expected parse error for %q", src)
			}
		})
	}
}

// Conflicting redeclarations of the same class with different explicit
// metadata are an error, not a silent overwrite.
func TestParseClassDuplicateLabelConflict(t *testing.T) {
	_, err := Parse(strings.NewReader(`classDiagram
    class Foo["A"]
    class Foo["B"]`))
	if err == nil {
		t.Error("expected error for conflicting labels")
	}
}

func TestParseClassDuplicateGenericConflict(t *testing.T) {
	_, err := Parse(strings.NewReader(`classDiagram
    class Foo~T~
    class Foo~U~`))
	if err == nil {
		t.Error("expected error for conflicting generics")
	}
}

// Re-declaring a class with the *same* label should be a no-op, not
// an error — common in real diagrams that mention a class in a
// relation before its full declaration.
func TestParseClassDuplicateLabelSameValueOK(t *testing.T) {
	d, err := Parse(strings.NewReader(`classDiagram
    Foo --> Bar
    class Foo["My Label"]
    class Foo["My Label"]`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if d.Classes[0].Label != "My Label" {
		t.Errorf("Label = %q", d.Classes[0].Label)
	}
}

// Token-boundary modifier stripping: `$`/`*` inside a token (e.g. an
// identifier or generic-typed return) must NOT be stripped. Only
// trailing markers on whitespace-separated tokens count.
func TestParseMemberModifierTokenBoundary(t *testing.T) {
	// A type literal containing `*` (e.g. C-style pointer in a comment)
	// preserved verbatim: not a real Mermaid type, but a stress test.
	d, err := Parse(strings.NewReader(`classDiagram
    class C {
        +ptr Foo*Bar
    }`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	m := d.Classes[0].Members[0]
	if m.IsAbstract {
		t.Errorf("embedded `*` in `Foo*Bar` should not flag abstract")
	}
	if !strings.Contains(m.Name, "Foo*Bar") {
		t.Errorf("embedded `*` was stripped: Name=%q", m.Name)
	}
}

// Single-line member shorthand: `ClassName : memberText` adds a
// member to ClassName without a `{ … }` block.
func TestParseSingleLineMember(t *testing.T) {
	d, err := Parse(strings.NewReader(`classDiagram
    class Vehicle
    Vehicle : +tires int
    Vehicle : +start() void
    BareClass : +autoRegistered`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(d.Classes) != 2 {
		t.Fatalf("want 2 classes (Vehicle + auto BareClass), got %d", len(d.Classes))
	}
	veh := d.Classes[0]
	if veh.ID != "Vehicle" || len(veh.Members) != 2 {
		t.Errorf("Vehicle = %+v", veh)
	}
	if veh.Members[0].Name != "tires int" || veh.Members[0].Visibility != diagram.VisibilityPublic {
		t.Errorf("Vehicle field = %+v", veh.Members[0])
	}
	if !veh.Members[1].IsMethod || veh.Members[1].Name != "start" || veh.Members[1].ReturnType != "void" {
		t.Errorf("Vehicle method = %+v", veh.Members[1])
	}
	bare := d.Classes[1]
	if bare.ID != "BareClass" || len(bare.Members) != 1 {
		t.Errorf("BareClass = %+v", bare)
	}
}

// Inline annotation on the class declaration line — `class Foo <<Interface>>`
// — and the bare-line form `Foo <<Service>>` both attach to class Foo.
func TestParseInlineAnnotation(t *testing.T) {
	d, err := Parse(strings.NewReader(`classDiagram
    class Drawable <<Interface>>
    class Repository
    Repository <<Service>>`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(d.Classes) != 2 {
		t.Fatalf("want 2 classes, got %d", len(d.Classes))
	}
	if d.Classes[0].Annotation != diagram.AnnotationInterface {
		t.Errorf("Drawable annotation = %v", d.Classes[0].Annotation)
	}
	if d.Classes[1].Annotation != diagram.AnnotationService {
		t.Errorf("Repository annotation = %v", d.Classes[1].Annotation)
	}
}

// Inline annotation must work even when the class also has a body.
func TestParseInlineAnnotationWithBody(t *testing.T) {
	d, err := Parse(strings.NewReader(`classDiagram
    class Shape <<Abstract>> {
        +draw() void
    }`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	c := d.Classes[0]
	if c.Annotation != diagram.AnnotationAbstract {
		t.Errorf("annotation = %v", c.Annotation)
	}
	if len(c.Members) != 1 {
		t.Errorf("members = %v", c.Members)
	}
}

func TestParseGeneralNote(t *testing.T) {
	d, err := Parse(strings.NewReader(`classDiagram
    class Foo
    note "This diagram is a stub."`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(d.Notes) != 1 {
		t.Fatalf("want 1 note, got %d", len(d.Notes))
	}
	n := d.Notes[0]
	if n.Text != "This diagram is a stub." || n.For != "" {
		t.Errorf("note = %+v", n)
	}
}

func TestParseNoteForClass(t *testing.T) {
	d, err := Parse(strings.NewReader(`classDiagram
    class Foo
    note for Foo "Annotation describing Foo"`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(d.Notes) != 1 {
		t.Fatalf("want 1 note, got %d", len(d.Notes))
	}
	n := d.Notes[0]
	if n.For != "Foo" || n.Text != "Annotation describing Foo" {
		t.Errorf("note = %+v", n)
	}
}

// Mermaid uses the literal `\n` sequence to mean a line break inside
// a note's text. The parser converts it to a real newline so
// renderers can split on '\n' directly.
func TestParseNoteLineBreaks(t *testing.T) {
	d, err := Parse(strings.NewReader(`classDiagram
    note "line one\nline two"`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if d.Notes[0].Text != "line one\nline two" {
		t.Errorf("note text = %q, want with embedded newline", d.Notes[0].Text)
	}
}

// `note for UnknownClass "..."` should auto-register the target
// rather than error — mirrors how relations create classes on the
// fly. Otherwise users who write notes before declaring classes get
// surprise errors.
func TestParseNoteAutoRegistersTarget(t *testing.T) {
	d, err := Parse(strings.NewReader(`classDiagram
    note for Foo "hello"`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(d.Classes) != 1 || d.Classes[0].ID != "Foo" {
		t.Errorf("Foo not auto-registered: %v", d.Classes)
	}
}

func TestParseDirectionInvalid(t *testing.T) {
	_, err := Parse(strings.NewReader("classDiagram\n    direction WAT"))
	if err == nil {
		t.Fatal("expected error for unknown direction")
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
