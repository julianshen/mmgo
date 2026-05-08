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

// `Boundary( ... ) { ... }` opens a nested scope; elements inside
// land in the flat Elements list AND the boundary's child slice.
// The trailing `{` may be on the same line or the next.
func TestParseC4BoundaryBlock(t *testing.T) {
	d, err := Parse(strings.NewReader(`C4Context
Boundary(b1, "Bank") {
  System(s, "Internet Banking")
  Person(u, "Customer")
}`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(d.Boundaries) != 1 {
		t.Fatalf("boundaries = %v", d.Boundaries)
	}
	b := d.Boundaries[0]
	if b.ID != "b1" || b.Label != "Bank" || b.Kind != diagram.C4BoundaryGeneric {
		t.Errorf("boundary = %+v", b)
	}
	if len(b.Elements) != 2 {
		t.Errorf("boundary elements idx = %v, want 2", b.Elements)
	}
	// Flat Elements still has both entries.
	if len(d.Elements) != 2 {
		t.Errorf("flat Elements = %v", d.Elements)
	}
}

// Each documented boundary keyword maps to its kind.
func TestParseC4BoundaryKinds(t *testing.T) {
	cases := []struct {
		input string
		want  diagram.C4BoundaryKind
	}{
		{`Boundary(b, "X") {`, diagram.C4BoundaryGeneric},
		{`System_Boundary(b, "X") {`, diagram.C4BoundarySystem},
		{`Enterprise_Boundary(b, "X") {`, diagram.C4BoundaryEnterprise},
		{`Container_Boundary(b, "X") {`, diagram.C4BoundaryContainer},
	}
	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			d, err := Parse(strings.NewReader("C4Context\n" + tc.input + "\n}\n"))
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			if len(d.Boundaries) != 1 {
				t.Fatalf("boundaries = %v", d.Boundaries)
			}
			if d.Boundaries[0].Kind != tc.want {
				t.Errorf("kind = %v, want %v", d.Boundaries[0].Kind, tc.want)
			}
		})
	}
}

// Nested boundaries form a tree; inner element idx lands in the
// inner boundary, NOT the outer.
func TestParseC4BoundaryNested(t *testing.T) {
	d, err := Parse(strings.NewReader(`C4Container
Enterprise_Boundary(ent, "Enterprise") {
  System_Boundary(sys, "System") {
    Container(c, "App", "Go")
  }
}`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(d.Boundaries) != 1 {
		t.Fatalf("boundaries = %v", d.Boundaries)
	}
	ent := d.Boundaries[0]
	if len(ent.Boundaries) != 1 || len(ent.Elements) != 0 {
		t.Errorf("ent = %+v", ent)
	}
	sys := ent.Boundaries[0]
	if sys.Kind != diagram.C4BoundarySystem || len(sys.Elements) != 1 {
		t.Errorf("sys = %+v", sys)
	}
	if d.Elements[sys.Elements[0]].ID != "c" {
		t.Errorf("inner element id mismatch")
	}
}

// A stray top-level `}` is silently skipped — it may belong to a
// brace-delimited construct mmgo doesn't yet recognise (e.g. a
// Deployment_Node block), and aborting the whole parse on a
// trailing brace would regress accepted inputs.
func TestParseC4BoundaryStrayCloseSkipped(t *testing.T) {
	d, err := Parse(strings.NewReader(`C4Context
Person(u, "X")
}`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(d.Elements) != 1 || d.Elements[0].ID != "u" {
		t.Errorf("element survived stray '}': %+v", d.Elements)
	}
}

// A `Boundary(` without a closing `}` errors at EOF.
func TestParseC4BoundaryUnterminated(t *testing.T) {
	_, err := Parse(strings.NewReader(`C4Context
Boundary(b, "X") {
  Person(u, "Y")`))
	if err == nil {
		t.Error("expected error for unterminated boundary")
	}
}

// The trailing `{` may live on its own line.
func TestParseC4BoundaryBraceOnNextLine(t *testing.T) {
	d, err := Parse(strings.NewReader(`C4Context
Boundary(b, "X")
{
  Person(u, "Y")
}`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(d.Boundaries) != 1 || len(d.Boundaries[0].Elements) != 1 {
		t.Errorf("boundary = %+v", d.Boundaries[0])
	}
}

// `Boundary(...)` without an eventual `{` (e.g., immediately
// followed by an element line) is rejected — the silent-scope-
// swallow path the simplify review flagged as critical.
func TestParseC4BoundaryMissingBraceRejected(t *testing.T) {
	_, err := Parse(strings.NewReader(`C4Context
Boundary(b, "X")
Person(p, "Y")`))
	if err == nil {
		t.Error("expected error for Boundary(...) without { before next line")
	}
}

// `parseBoundary` accepts the alias-only / 2-arg / 3-arg
// positional forms and rejects an empty argument list.
func TestParseC4BoundaryArities(t *testing.T) {
	cases := []struct {
		input    string
		wantID   string
		wantLabel string
		wantHint string
	}{
		{`Boundary(b) {`, "b", "", ""},
		{`Boundary(b, "Bank") {`, "b", "Bank", ""},
		{`Boundary(b, "Bank", "system") {`, "b", "Bank", "system"},
	}
	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			d, err := Parse(strings.NewReader("C4Context\n" + tc.input + "\n}\n"))
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			b := d.Boundaries[0]
			if b.ID != tc.wantID || b.Label != tc.wantLabel || b.TypeHint != tc.wantHint {
				t.Errorf("got %+v", b)
			}
		})
	}

	// Empty argument list rejected.
	if _, err := Parse(strings.NewReader("C4Context\nBoundary() {\n}\n")); err == nil {
		t.Error("expected error for empty Boundary arg list")
	}
	// Empty first arg also rejected so a typo like `Boundary(, "X")`
	// doesn't ship a boundary with ID="" (which would render as a
	// bare ` <<boundary>>` heading).
	if _, err := Parse(strings.NewReader("C4Context\nBoundary(, \"Bank\") {\n}\n")); err == nil {
		t.Error("expected error for empty Boundary alias")
	}
}

// `splitBoundaryHead` rejects malformed headers — missing
// closing paren and unexpected trailing content after `)`.
func TestParseC4BoundaryHeaderErrors(t *testing.T) {
	cases := []string{
		`Boundary(b, "X"` + "\n", // missing `)`
		`Boundary(b, "X") garbage`,
	}
	for _, head := range cases {
		t.Run(head, func(t *testing.T) {
			if _, err := Parse(strings.NewReader("C4Context\n" + head + "\n")); err == nil {
				t.Error("expected error")
			}
		})
	}
}

// Elements OUTSIDE every boundary block stay on the flat
// Elements list but DON'T appear in any boundary's child idx.
func TestParseC4ElementsOutsideBoundaries(t *testing.T) {
	d, err := Parse(strings.NewReader(`C4Context
Person(outside, "Out")
Boundary(b, "Inner") {
  System(inside, "In")
}`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(d.Elements) != 2 {
		t.Fatalf("flat elements = %v", d.Elements)
	}
	b := d.Boundaries[0]
	if len(b.Elements) != 1 {
		t.Fatalf("boundary children = %v", b.Elements)
	}
	if d.Elements[b.Elements[0]].ID != "inside" {
		t.Errorf("boundary should only own 'inside'")
	}
	// `outside` must be flat-indexed but NOT in b.Elements.
	for _, idx := range b.Elements {
		if d.Elements[idx].ID == "outside" {
			t.Error("'outside' should not appear in boundary children")
		}
	}
}

func TestParseElementNamedArgs(t *testing.T) {
	input := `C4Context
Person(u, "User", $descr="A user with very long description", $link="https://example.com", $tags="external,vip", $sprite="users")
`
	d, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(d.Elements) != 1 {
		t.Fatalf("got %d elements, want 1", len(d.Elements))
	}
	e := d.Elements[0]
	if e.Description != "A user with very long description" {
		t.Errorf("Description = %q", e.Description)
	}
	if e.Link != "https://example.com" {
		t.Errorf("Link = %q", e.Link)
	}
	if e.Tags != "external,vip" {
		t.Errorf("Tags = %q", e.Tags)
	}
	if e.Sprite != "users" {
		t.Errorf("Sprite = %q", e.Sprite)
	}
}

// Named `$descr=` overrides the positional description when both are
// present — Mermaid precedence rule.
func TestParseElementNamedDescrOverridesPositional(t *testing.T) {
	input := `C4Context
Person(u, "User", "positional descr", $descr="named descr wins")
`
	d, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if d.Elements[0].Description != "named descr wins" {
		t.Errorf("named $descr= should win, got %q", d.Elements[0].Description)
	}
}

// Container's positional 3rd-arg is technology, but a `?techn=` named
// arg should also work and override it.
func TestParseContainerTechnologyNamedArg(t *testing.T) {
	input := `C4Container
Container(api, "API", "Go", "REST surface", ?techn="Rust")
`
	d, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if d.Elements[0].Technology != "Rust" {
		t.Errorf("?techn= should override positional, got %q", d.Elements[0].Technology)
	}
}

func TestParseRelationNamedArgs(t *testing.T) {
	input := `C4Context
Person(u, "User")
System(s, "System")
Rel(u, s, "uses", $tags="async", $offsetX="10", $offsetY="-5", $link="https://docs.example.com")
`
	d, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(d.Relations) != 1 {
		t.Fatalf("got %d relations, want 1", len(d.Relations))
	}
	r := d.Relations[0]
	if r.Tags != "async" {
		t.Errorf("Tags = %q", r.Tags)
	}
	if r.Link != "https://docs.example.com" {
		t.Errorf("Link = %q", r.Link)
	}
	if r.OffsetX != 10 {
		t.Errorf("OffsetX = %v, want 10", r.OffsetX)
	}
	if r.OffsetY != -5 {
		t.Errorf("OffsetY = %v, want -5", r.OffsetY)
	}
}

// Boundary inherits the same named-arg surface; $link= wraps the
// boundary frame in a clickable anchor.
func TestParseBoundaryNamedArgs(t *testing.T) {
	input := `C4Container
System_Boundary(b, "Backend", $link="https://ops.example.com", $tags="prod") {
  Container(api, "API", "Go")
}
`
	d, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(d.Boundaries) != 1 {
		t.Fatalf("got %d boundaries, want 1", len(d.Boundaries))
	}
	b := d.Boundaries[0]
	if b.Link != "https://ops.example.com" {
		t.Errorf("Boundary Link = %q", b.Link)
	}
	if b.Tags != "prod" {
		t.Errorf("Boundary Tags = %q", b.Tags)
	}
}

// Malformed numeric offsets should silently no-op (return 0) rather
// than reject the whole line.
func TestParseRelationOffsetMalformed(t *testing.T) {
	input := `C4Context
Person(u, "User")
System(s, "S")
Rel(u, s, "x", $offsetX="not-a-number")
`
	d, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if d.Relations[0].OffsetX != 0 {
		t.Errorf("malformed offset should be 0, got %v", d.Relations[0].OffsetX)
	}
}

// Interleaved positional + named args parse correctly: the named
// entry doesn't consume a positional slot, so positional indices
// still address the surrounding fields.
func TestParseElementInterleavedNamedAndPositional(t *testing.T) {
	input := `C4Context
Person(u, $tags="external", "User", $descr="From outside")
`
	d, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	e := d.Elements[0]
	if e.ID != "u" || e.Label != "User" {
		t.Errorf("positional fields wrong: ID=%q Label=%q", e.ID, e.Label)
	}
	if e.Tags != "external" {
		t.Errorf("Tags = %q", e.Tags)
	}
	if e.Description != "From outside" {
		t.Errorf("Description = %q", e.Description)
	}
}

// Empty-value named args (`$descr=`) must not silently clobber a
// positional value with "" — splitNamed rejects them so the
// override loop's comma-ok branch never fires.
func TestParseElementEmptyNamedArgPreservesPositional(t *testing.T) {
	input := `C4Context
Person(u, "User", "positional descr", $descr="")
`
	d, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if d.Elements[0].Description != "positional descr" {
		t.Errorf("empty $descr= must not clobber positional, got %q", d.Elements[0].Description)
	}
}

// Duplicate named keys: last wins via map assignment. Pin so a
// future refactor to multi-value slices doesn't silently change it.
func TestParseElementDuplicateNamedKeyLastWins(t *testing.T) {
	input := `C4Context
Person(u, "User", $tags="first", $tags="second")
`
	d, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if d.Elements[0].Tags != "second" {
		t.Errorf("duplicate $tags=: last must win, got %q", d.Elements[0].Tags)
	}
}

// `?techn=` consumes the technology slot, so the next positional arg
// is description (not silently treated as technology and shadowed by
// the named override).
func TestParseContainerNamedTechnShiftsPositional(t *testing.T) {
	input := `C4Container
Container(api, "API", ?techn="Rust", "REST surface")
`
	d, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	e := d.Elements[0]
	if e.Technology != "Rust" {
		t.Errorf("Technology = %q, want Rust", e.Technology)
	}
	if e.Description != "REST surface" {
		t.Errorf("Description = %q, want REST surface", e.Description)
	}
}

// Every Container/Component variant — including _Ext / DB / Queue —
// must accept the technology positional slot. A coverage gap would
// silently route technology into description on those kinds.
func TestParseAllContainerComponentVariantsTakeTechnology(t *testing.T) {
	cases := []struct {
		header string
		line   string
	}{
		{"C4Container", `ContainerDb(db, "DB", "Postgres", "stores X")`},
		{"C4Container", `ContainerQueue(q, "Q", "Kafka", "events")`},
		{"C4Container", `Container_Ext(api, "API", "Go", "external")`},
		{"C4Container", `ContainerDb_Ext(db, "DB", "Postgres", "external")`},
		{"C4Container", `ContainerQueue_Ext(q, "Q", "Kafka", "external")`},
		{"C4Component", `Component(c, "Comp", "Go", "logic")`},
		{"C4Component", `Component_Ext(c, "Comp", "Go", "external")`},
		{"C4Component", `ComponentDb(c, "Comp", "SQLite", "local")`},
		{"C4Component", `ComponentDb_Ext(c, "Comp", "SQLite", "external")`},
		{"C4Component", `ComponentQueue(c, "Comp", "RabbitMQ", "queue")`},
		{"C4Component", `ComponentQueue_Ext(c, "Comp", "RabbitMQ", "external")`},
	}
	for _, c := range cases {
		t.Run(c.line, func(t *testing.T) {
			d, err := Parse(strings.NewReader(c.header + "\n" + c.line + "\n"))
			if err != nil {
				t.Fatalf("Parse: %v", err)
			}
			if len(d.Elements) != 1 {
				t.Fatalf("got %d elements", len(d.Elements))
			}
			e := d.Elements[0]
			if e.Technology == "" {
				t.Errorf("expected non-empty Technology, got Description=%q (slot likely misrouted)", e.Description)
			}
			if e.Description == "" {
				t.Errorf("expected non-empty Description")
			}
		})
	}
}

// `$descr=""` is recognised as a named arg with empty value (not
// reclassified as positional input), so the literal token doesn't
// leak into the Label or Description field. Codex regression pin.
func TestParseElementEmptyNamedArgDoesNotLeakAsPositional(t *testing.T) {
	input := `C4Context
Person(u, "User", $descr="")
`
	d, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	e := d.Elements[0]
	if e.ID != "u" || e.Label != "User" {
		t.Errorf("positional fields wrong: ID=%q Label=%q", e.ID, e.Label)
	}
	if e.Description != "" {
		t.Errorf("Description must be empty (no positional fallback, empty named): got %q", e.Description)
	}
	// Critically: the literal token must not appear anywhere.
	for _, field := range []string{e.ID, e.Label, e.Description, e.Technology, e.Tags, e.Sprite, e.Link} {
		if strings.Contains(field, "$descr=") {
			t.Errorf("named-arg sigil leaked into a semantic field: %q", field)
		}
	}
}
