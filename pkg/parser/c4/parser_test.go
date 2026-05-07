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
	cases := []struct {
		kw  string
		dir diagram.C4RelDirection
	}{
		{"Rel", diagram.C4RelDefault},
		{"Rel_U", diagram.C4RelUp},
		{"Rel_D", diagram.C4RelDown},
		{"Rel_L", diagram.C4RelLeft},
		{"Rel_R", diagram.C4RelRight},
		{"Rel_Back", diagram.C4RelBack},
		{"BiRel", diagram.C4RelBi},
	}
	for _, tc := range cases {
		t.Run(tc.kw, func(t *testing.T) {
			input := "C4Context\n    " + tc.kw + "(a, b, \"x\")"
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
			if d.Relations[0].Direction != tc.dir {
				t.Errorf("direction = %v, want %v", d.Relations[0].Direction, tc.dir)
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

func TestParseEscapedQuoteInLabel(t *testing.T) {
	input := `C4Context
    Person(a, "Say \"Hi\"", "Greeter")`
	d, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(d.Elements) != 1 {
		t.Fatalf("want 1 element, got %d", len(d.Elements))
	}
	if d.Elements[0].Description != "Greeter" {
		t.Errorf("description = %q, escaped quote broke arg split", d.Elements[0].Description)
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

// All new element kinds (queues, *_Ext variants, Deployment_Node)
// parse to the matching C4ElementKind.
func TestParseElementKindCompleteness(t *testing.T) {
	cases := []struct {
		input string
		want  diagram.C4ElementKind
	}{
		{`SystemQueue(sq, "Q")`, diagram.C4ElementSystemQueue},
		{`SystemQueue_Ext(sqe, "Q")`, diagram.C4ElementSystemQueueExt},
		{`SystemDb_Ext(sde, "DB")`, diagram.C4ElementSystemDBExt},
		{`Container_Ext(ce, "C")`, diagram.C4ElementContainerExt},
		{`ContainerDb_Ext(cde, "CDB")`, diagram.C4ElementContainerDBExt},
		{`ContainerQueue(cq, "CQ")`, diagram.C4ElementContainerQueue},
		{`ContainerQueue_Ext(cqe, "CQ")`, diagram.C4ElementContainerQueueExt},
		{`Component_Ext(coe, "C")`, diagram.C4ElementComponentExt},
		{`ComponentDb(codb, "DB")`, diagram.C4ElementComponentDB},
		{`ComponentDb_Ext(codbe, "DB")`, diagram.C4ElementComponentDBExt},
		{`ComponentQueue(coq, "Q")`, diagram.C4ElementComponentQueue},
		{`ComponentQueue_Ext(coqe, "Q")`, diagram.C4ElementComponentQueueExt},
		{`Deployment_Node(dn, "Web tier")`, diagram.C4ElementDeploymentNode},
		{`Node(n, "Box")`, diagram.C4ElementDeploymentNode},
		{`Node_L(nl, "Left")`, diagram.C4ElementDeploymentNode},
		{`Node_R(nr, "Right")`, diagram.C4ElementDeploymentNode},
	}
	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			d, err := Parse(strings.NewReader("C4Container\n" + tc.input))
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			if len(d.Elements) != 1 {
				t.Fatalf("elements = %v", d.Elements)
			}
			if d.Elements[0].Kind != tc.want {
				t.Errorf("kind = %v, want %v", d.Elements[0].Kind, tc.want)
			}
		})
	}
}

// Long-form `Rel_Up`/`Rel_Down`/`Rel_Left`/`Rel_Right` populate
// the same Direction values the short forms produce.
func TestParseLongFormRelations(t *testing.T) {
	cases := []struct {
		input string
		want  diagram.C4RelDirection
	}{
		{`Rel_Up(a, b, "up")`, diagram.C4RelUp},
		{`Rel_Down(a, b, "down")`, diagram.C4RelDown},
		{`Rel_Left(a, b, "left")`, diagram.C4RelLeft},
		{`Rel_Right(a, b, "right")`, diagram.C4RelRight},
	}
	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			d, err := Parse(strings.NewReader("C4Context\n" + tc.input))
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			if len(d.Relations) != 1 {
				t.Fatalf("relations = %v", d.Relations)
			}
			if d.Relations[0].Direction != tc.want {
				t.Errorf("direction = %v, want %v", d.Relations[0].Direction, tc.want)
			}
		})
	}
}

// accTitle / accDescr lines populate the matching AST fields.
func TestParseC4Accessibility(t *testing.T) {
	d, err := Parse(strings.NewReader(`C4Context
accTitle: System Context
accDescr: Top-level system view
title Internet Banking`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if d.AccTitle != "System Context" {
		t.Errorf("accTitle = %q", d.AccTitle)
	}
	if d.AccDescr != "Top-level system view" {
		t.Errorf("accDescr = %q", d.AccDescr)
	}
	if d.Title != "Internet Banking" {
		t.Errorf("title = %q", d.Title)
	}
}

// Multi-line `accDescr { ... }` block accumulates body lines.
func TestParseC4AccDescrBlock(t *testing.T) {
	d, err := Parse(strings.NewReader(`C4Context
accDescr {
  Top-level system view
  with external actors
}
System(s, "Banking")`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	want := "Top-level system view\nwith external actors"
	if d.AccDescr != want {
		t.Errorf("accDescr = %q, want %q", d.AccDescr, want)
	}
}

// Unterminated `accDescr {` errors with a clear message.
func TestParseC4AccDescrUnterminated(t *testing.T) {
	_, err := Parse(strings.NewReader(`C4Context
accDescr {
  Open
System(s, "X")`))
	if err == nil {
		t.Error("expected error for unterminated accDescr block")
	}
}
