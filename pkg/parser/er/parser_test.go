package er

import (
	"strings"
	"testing"

	"github.com/julianshen/mmgo/pkg/diagram"
)

// `direction TB|BT|LR|RL` populates the diagram-level Direction.
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
			d, err := Parse(strings.NewReader("erDiagram\n    " + tc.src + "\n    A ||--o{ B"))
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			if d.Direction != tc.want {
				t.Errorf("direction = %v, want %v", d.Direction, tc.want)
			}
		})
	}
}

func TestParseDirectionInvalid(t *testing.T) {
	_, err := Parse(strings.NewReader("erDiagram\n    direction WAT"))
	if err == nil {
		t.Error("expected error for unknown direction")
	}
}

// Cardinality tokenizer covers all 4×4×2 = 32 combinations of
// (left-glyph, right-glyph, line-type). Spot-check the ones the
// hand-curated table missed.
func TestParseCardinalityFullMatrix(t *testing.T) {
	cases := []struct {
		src           string
		wantFromCard  diagram.ERCardinality
		wantToCard    diagram.ERCardinality
	}{
		// Solid-line, previously missing combos:
		{"A o|--o| B", diagram.ERCardZeroOrOne, diagram.ERCardZeroOrOne},
		{"A ||--o| B", diagram.ERCardExactlyOne, diagram.ERCardZeroOrOne},
		{"A o|--|{ B", diagram.ERCardZeroOrOne, diagram.ERCardOneOrMore},
		// Dashed-line variants the prior code missed:
		{"A ||..o| B", diagram.ERCardExactlyOne, diagram.ERCardZeroOrOne},
		{"A o|..||  B", diagram.ERCardZeroOrOne, diagram.ERCardExactlyOne},
		{"A }o..o{ B", diagram.ERCardZeroOrMore, diagram.ERCardZeroOrMore},
	}
	for _, tc := range cases {
		t.Run(tc.src, func(t *testing.T) {
			d, err := Parse(strings.NewReader("erDiagram\n    " + tc.src))
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			if len(d.Relationships) != 1 {
				t.Fatalf("want 1 relationship, got %d", len(d.Relationships))
			}
			r := d.Relationships[0]
			if r.FromCard != tc.wantFromCard || r.ToCard != tc.wantToCard {
				t.Errorf("got from=%v to=%v, want from=%v to=%v",
					r.FromCard, r.ToCard, tc.wantFromCard, tc.wantToCard)
			}
		})
	}
}

// Multi-constraint attributes: `id int PK, FK` records both keys.
func TestParseAttributeMultipleKeys(t *testing.T) {
	d, err := Parse(strings.NewReader(`erDiagram
    ORDER {
        int id PK, FK
    }`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	a := d.Entities[0].Attributes[0]
	if a.Type != "int" || a.Name != "id" {
		t.Errorf("attr = %+v", a)
	}
	if a.Key != diagram.ERKeyPK {
		t.Errorf("Key = %v, want PK", a.Key)
	}
	if len(a.Keys) != 2 || a.Keys[0] != diagram.ERKeyPK || a.Keys[1] != diagram.ERKeyFK {
		t.Errorf("Keys = %v, want [PK FK]", a.Keys)
	}
}

// `*name PK` (asterisk shorthand combined with explicit PK) must
// not double-record PK in Keys. Dedupe pins the contract.
func TestParseAttributeAsteriskPlusPKDedupes(t *testing.T) {
	d, err := Parse(strings.NewReader(`erDiagram
    ORDER {
        int *id PK, FK
    }`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	a := d.Entities[0].Attributes[0]
	if len(a.Keys) != 2 || a.Keys[0] != diagram.ERKeyPK || a.Keys[1] != diagram.ERKeyFK {
		t.Errorf("Keys = %v, want [PK FK]", a.Keys)
	}
}

// Direct unit test for the parseERKey helper, covering case and
// whitespace tolerance plus the unknown-token branch.
func TestParseERKey(t *testing.T) {
	cases := []struct {
		in     string
		want   diagram.ERAttributeKey
		wantOK bool
	}{
		{"PK", diagram.ERKeyPK, true},
		{"pk", diagram.ERKeyPK, true},
		{" FK ", diagram.ERKeyFK, true},
		{"UK", diagram.ERKeyUK, true},
		{"XX", diagram.ERKeyNone, false},
		{"", diagram.ERKeyNone, false},
	}
	for _, tc := range cases {
		got, ok := parseERKey(tc.in)
		if got != tc.want || ok != tc.wantOK {
			t.Errorf("parseERKey(%q) = (%v, %v), want (%v, %v)", tc.in, got, ok, tc.want, tc.wantOK)
		}
	}
}

// `*name` shorthand marks the attribute as primary key.
func TestParseAttributeAsteriskPK(t *testing.T) {
	d, err := Parse(strings.NewReader(`erDiagram
    ORDER {
        int *id
    }`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	a := d.Entities[0].Attributes[0]
	if a.Name != "id" {
		t.Errorf("Name = %q, want id (asterisk should be stripped)", a.Name)
	}
	if a.Key != diagram.ERKeyPK {
		t.Errorf("Key = %v, want PK", a.Key)
	}
}

// Quoted comments preserve embedded spaces; surrounding quotes are
// stripped so the comment text is clean for renderers.
func TestParseAttributeQuotedComment(t *testing.T) {
	d, err := Parse(strings.NewReader(`erDiagram
    ORDER {
        string name "the customer's full name"
        int age PK "primary identifier"
    }`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	attrs := d.Entities[0].Attributes
	if attrs[0].Comment != "the customer's full name" {
		t.Errorf("attr[0].Comment = %q", attrs[0].Comment)
	}
	if attrs[1].Comment != "primary identifier" {
		t.Errorf("attr[1].Comment = %q", attrs[1].Comment)
	}
}

func TestParseHeaderRequired(t *testing.T) {
	_, err := Parse(strings.NewReader("CUSTOMER ||--o{ ORDER : places"))
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestParseEmptyDiagram(t *testing.T) {
	d, err := Parse(strings.NewReader("erDiagram"))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(d.Entities) != 0 {
		t.Errorf("empty: %+v", d)
	}
}

func TestParseEntityWithAttributes(t *testing.T) {
	input := `erDiagram
    CUSTOMER {
        string name
        int age PK
        string email UK
    }`
	d, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(d.Entities) != 1 {
		t.Fatalf("want 1 entity, got %d", len(d.Entities))
	}
	e := d.Entities[0]
	if e.Name != "CUSTOMER" {
		t.Errorf("name = %q", e.Name)
	}
	if len(e.Attributes) != 3 {
		t.Fatalf("want 3 attrs, got %d", len(e.Attributes))
	}
	if e.Attributes[1].Key != diagram.ERKeyPK {
		t.Errorf("age key = %v, want PK", e.Attributes[1].Key)
	}
	if e.Attributes[2].Key != diagram.ERKeyUK {
		t.Errorf("email key = %v, want UK", e.Attributes[2].Key)
	}
}

func TestParseRelationship(t *testing.T) {
	cases := []struct {
		src      string
		fromCard diagram.ERCardinality
		toCard   diagram.ERCardinality
	}{
		{"CUSTOMER ||--o{ ORDER : places", diagram.ERCardExactlyOne, diagram.ERCardZeroOrMore},
		{"ORDER }|--|{ PRODUCT : contains", diagram.ERCardOneOrMore, diagram.ERCardOneOrMore},
		{"PERSON ||--|| PASSPORT : has", diagram.ERCardExactlyOne, diagram.ERCardExactlyOne},
	}
	for _, tc := range cases {
		t.Run(tc.src, func(t *testing.T) {
			input := "erDiagram\n    " + tc.src
			d, err := Parse(strings.NewReader(input))
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			if len(d.Relationships) != 1 {
				t.Fatalf("want 1 rel, got %d", len(d.Relationships))
			}
			r := d.Relationships[0]
			if r.FromCard != tc.fromCard {
				t.Errorf("fromCard = %v, want %v", r.FromCard, tc.fromCard)
			}
			if r.ToCard != tc.toCard {
				t.Errorf("toCard = %v, want %v", r.ToCard, tc.toCard)
			}
		})
	}
}

func TestParseRelationshipLabel(t *testing.T) {
	input := `erDiagram
    CUSTOMER ||--o{ ORDER : "places orders"`
	d, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if d.Relationships[0].Label != "places orders" {
		t.Errorf("label = %q", d.Relationships[0].Label)
	}
}

func TestParseAutoRegistersEntities(t *testing.T) {
	input := `erDiagram
    A ||--o{ B : has`
	d, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(d.Entities) != 2 {
		t.Errorf("want 2 entities, got %d", len(d.Entities))
	}
}

func TestParseComments(t *testing.T) {
	input := `erDiagram
    %% comment
    A ||--o{ B : x %% trailing`
	d, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(d.Relationships) != 1 {
		t.Errorf("want 1 rel, got %d", len(d.Relationships))
	}
}

func TestParseUnclosedEntity(t *testing.T) {
	input := `erDiagram
    CUSTOMER {
        string name`
	_, err := Parse(strings.NewReader(input))
	if err == nil {
		t.Fatal("expected error for unclosed entity")
	}
}

func TestParseFKAttribute(t *testing.T) {
	input := `erDiagram
    ORDER {
        int customer_id FK
    }`
	d, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if d.Entities[0].Attributes[0].Key != diagram.ERKeyFK {
		t.Errorf("key = %v, want FK", d.Entities[0].Attributes[0].Key)
	}
}

func TestParseAttributeWithComment(t *testing.T) {
	input := `erDiagram
    ITEM {
        string desc "the description"
    }`
	d, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if d.Entities[0].Attributes[0].Comment != "\"the description\"" && d.Entities[0].Attributes[0].Comment != "the description" {
		t.Errorf("comment = %q", d.Entities[0].Attributes[0].Comment)
	}
}

func TestParseRelationshipNoLabel(t *testing.T) {
	input := `erDiagram
    A ||--|| B`
	d, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if d.Relationships[0].Label != "" {
		t.Errorf("label should be empty, got %q", d.Relationships[0].Label)
	}
}

func TestParseMultipleEntities(t *testing.T) {
	input := `erDiagram
    CUSTOMER {
        int id PK
    }
    ORDER {
        int id PK
        int customer_id FK
    }
    CUSTOMER ||--o{ ORDER : places`
	d, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(d.Entities) != 2 {
		t.Errorf("want 2 entities, got %d", len(d.Entities))
	}
	if len(d.Relationships) != 1 {
		t.Errorf("want 1 rel, got %d", len(d.Relationships))
	}
}

func TestParseDashedRelationship(t *testing.T) {
	input := `erDiagram
    A ||..o{ B : identifies`
	d, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(d.Relationships) != 1 {
		t.Fatalf("want 1 rel, got %d", len(d.Relationships))
	}
	if d.Relationships[0].Label != "identifies" {
		t.Errorf("label = %q", d.Relationships[0].Label)
	}
}
