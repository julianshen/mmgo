package c4

import (
	"strings"
	"testing"

	"github.com/julianshen/mmgo/pkg/diagram"
)

func TestParseHeaderRequired(t *testing.T) {
	_, err := Parse(strings.NewReader("Person(u, \"User\")"))
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestParseVariants(t *testing.T) {
	cases := []struct {
		header  string
		variant diagram.C4Variant
	}{
		{"C4Context", diagram.C4VariantContext},
		{"C4Container", diagram.C4VariantContainer},
		{"C4Component", diagram.C4VariantComponent},
		{"C4Dynamic", diagram.C4VariantDynamic},
		{"C4Deployment", diagram.C4VariantDeployment},
	}
	for _, tc := range cases {
		t.Run(tc.header, func(t *testing.T) {
			d, err := Parse(strings.NewReader(tc.header))
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			if d.Variant != tc.variant {
				t.Errorf("variant = %v, want %v", d.Variant, tc.variant)
			}
		})
	}
}

func TestParseTitle(t *testing.T) {
	d, err := Parse(strings.NewReader("C4Context\n    title System Overview"))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if d.Title != "System Overview" {
		t.Errorf("title = %q", d.Title)
	}
}

func TestParsePersonAndSystem(t *testing.T) {
	input := `C4Context
    Person(customerA, "Banking Customer", "A customer of the bank")
    System(banking, "Banking System", "Handles transactions")`
	d, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(d.Elements) != 2 {
		t.Fatalf("want 2 elements, got %d", len(d.Elements))
	}
	if d.Elements[0].Kind != diagram.C4ElementPerson {
		t.Errorf("elem[0] kind = %v", d.Elements[0].Kind)
	}
	if d.Elements[0].Label != "Banking Customer" {
		t.Errorf("elem[0] label = %q", d.Elements[0].Label)
	}
	if d.Elements[1].Kind != diagram.C4ElementSystem {
		t.Errorf("elem[1] kind = %v", d.Elements[1].Kind)
	}
}

func TestParseContainer(t *testing.T) {
	input := `C4Container
    Container(api, "API Gateway", "Go", "Routes requests")`
	d, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(d.Elements) != 1 {
		t.Fatal("want 1 element")
	}
	e := d.Elements[0]
	if e.Technology != "Go" {
		t.Errorf("technology = %q", e.Technology)
	}
	if e.Description != "Routes requests" {
		t.Errorf("description = %q", e.Description)
	}
}

func TestParseRelation(t *testing.T) {
	input := `C4Context
    Rel(a, b, "Uses")
    Rel_U(c, d, "Updates", "HTTPS")`
	d, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(d.Relations) != 2 {
		t.Fatalf("want 2 relations, got %d", len(d.Relations))
	}
	if d.Relations[0].Label != "Uses" {
		t.Errorf("rel[0] label = %q", d.Relations[0].Label)
	}
	if d.Relations[1].Technology != "HTTPS" {
		t.Errorf("rel[1] tech = %q", d.Relations[1].Technology)
	}
}

func TestParseAllRelVariants(t *testing.T) {
	for _, kw := range []string{"Rel", "Rel_U", "Rel_D", "Rel_L", "Rel_R", "Rel_Back", "BiRel"} {
		t.Run(kw, func(t *testing.T) {
			input := "C4Context\n    " + kw + "(a, b, \"x\")"
			d, err := Parse(strings.NewReader(input))
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			if len(d.Relations) != 1 {
				t.Fatalf("want 1 relation, got %d", len(d.Relations))
			}
			if d.Relations[0].Label != "x" {
				t.Errorf("label = %q", d.Relations[0].Label)
			}
		})
	}
}

func TestParseExternalAndDB(t *testing.T) {
	input := `C4Context
    Person_Ext(admin, "Admin")
    SystemDb(db, "Database")`
	d, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if d.Elements[0].Kind != diagram.C4ElementPersonExt {
		t.Errorf("elem[0] kind = %v", d.Elements[0].Kind)
	}
	if d.Elements[1].Kind != diagram.C4ElementSystemDB {
		t.Errorf("elem[1] kind = %v", d.Elements[1].Kind)
	}
}

func TestParseComments(t *testing.T) {
	input := `C4Context
    %% A comment
    Person(a, "User") %% trailing`
	d, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(d.Elements) != 1 {
		t.Errorf("want 1 element, got %d", len(d.Elements))
	}
}

func TestParseQuotedCommasInLabels(t *testing.T) {
	input := `C4Context
    Person(a, "Last, First", "Role, manager")`
	d, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if d.Elements[0].Label != "Last, First" {
		t.Errorf("label = %q", d.Elements[0].Label)
	}
}
