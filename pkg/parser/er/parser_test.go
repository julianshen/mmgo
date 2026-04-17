package er

import (
	"strings"
	"testing"

	"github.com/julianshen/mmgo/pkg/diagram"
)

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
