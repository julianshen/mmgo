package block

import (
	"strings"
	"testing"

	"github.com/julianshen/mmgo/pkg/diagram"
)

func TestParseHeaderRequired(t *testing.T) {
	_, err := Parse(strings.NewReader("a b c"))
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestParseEmptyDiagram(t *testing.T) {
	d, err := Parse(strings.NewReader("block-beta"))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(d.Nodes) != 0 || len(d.Edges) != 0 {
		t.Errorf("empty: %+v", d)
	}
}

func TestParseColumns(t *testing.T) {
	d, err := Parse(strings.NewReader("block-beta\n    columns 3"))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if d.Columns != 3 {
		t.Errorf("columns = %d, want 3", d.Columns)
	}
}

func TestParseSimpleBlocks(t *testing.T) {
	input := `block-beta
    a b c`
	d, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(d.Nodes) != 3 {
		t.Fatalf("want 3 nodes, got %d", len(d.Nodes))
	}
	for i, id := range []string{"a", "b", "c"} {
		if d.Nodes[i].ID != id {
			t.Errorf("node[%d].ID = %q, want %q", i, d.Nodes[i].ID, id)
		}
	}
}

func TestParseShapes(t *testing.T) {
	input := `block-beta
    a[Square]
    b(Round)
    c{Diamond}
    d((Circle))
    e([Stadium])`
	d, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(d.Nodes) != 5 {
		t.Fatalf("want 5 nodes, got %d", len(d.Nodes))
	}
	want := []struct {
		id    string
		label string
		shape diagram.BlockShape
	}{
		{"a", "Square", diagram.BlockShapeRect},
		{"b", "Round", diagram.BlockShapeRound},
		{"c", "Diamond", diagram.BlockShapeDiamond},
		{"d", "Circle", diagram.BlockShapeCircle},
		{"e", "Stadium", diagram.BlockShapeStadium},
	}
	for i, w := range want {
		got := d.Nodes[i]
		if got.ID != w.id || got.Label != w.label || got.Shape != w.shape {
			t.Errorf("node[%d] = %+v, want %+v", i, got, w)
		}
	}
}

func TestParseEdges(t *testing.T) {
	input := `block-beta
    a b c
    a --> b
    b --> c: next`
	d, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(d.Edges) != 2 {
		t.Fatalf("want 2 edges, got %d", len(d.Edges))
	}
	if d.Edges[1].Label != "next" {
		t.Errorf("edge[1].Label = %q", d.Edges[1].Label)
	}
}

func TestParseEdgePipeLabel(t *testing.T) {
	input := `block-beta
    a -->|jumps| b`
	d, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if d.Edges[0].Label != "jumps" {
		t.Errorf("label = %q", d.Edges[0].Label)
	}
}

func TestParseAutoRegisterFromEdge(t *testing.T) {
	input := `block-beta
    x --> y`
	d, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(d.Nodes) != 2 {
		t.Errorf("should auto-register x and y, got %d", len(d.Nodes))
	}
}

func TestParseComments(t *testing.T) {
	input := `block-beta
    %% comment
    a b %% trailing`
	d, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(d.Nodes) != 2 {
		t.Errorf("want 2 nodes, got %d", len(d.Nodes))
	}
}

func TestParseArrowInsideLabel(t *testing.T) {
	// "a[x --> y]" should parse as a single block with label
	// containing the arrow, NOT as an edge.
	input := `block-beta
    a[x --> y]`
	d, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(d.Nodes) != 1 {
		t.Fatalf("want 1 node, got %d", len(d.Nodes))
	}
	if len(d.Edges) != 0 {
		t.Errorf("should have no edges, got %+v", d.Edges)
	}
	if d.Nodes[0].Label != "x --> y" {
		t.Errorf("label = %q, want %q", d.Nodes[0].Label, "x --> y")
	}
}

func TestParseDashedEdge(t *testing.T) {
	input := `block-beta
    a --- b`
	d, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(d.Edges) != 1 {
		t.Fatalf("want 1 edge, got %d", len(d.Edges))
	}
}

// accTitle / accDescr lines populate the matching AST fields.
func TestParseAccessibility(t *testing.T) {
	d, err := Parse(strings.NewReader(`block-beta
    accTitle: System
    accDescr: High-level layout
    A B C`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if d.AccTitle != "System" {
		t.Errorf("accTitle = %q", d.AccTitle)
	}
	if d.AccDescr != "High-level layout" {
		t.Errorf("accDescr = %q", d.AccDescr)
	}
}

// New shape lexicon: hexagon `{{...}}`, subroutine `[[...]]`,
// double-circle `(((...)))`, cylinder `[(...)]`. Order matters
// for paren-precedence (((  >  ((  >  ().
func TestParseExtendedShapes(t *testing.T) {
	cases := []struct {
		input string
		want  diagram.BlockShape
		text  string
	}{
		{"H{{Hex}}", diagram.BlockShapeHexagon, "Hex"},
		{"S[[Sub]]", diagram.BlockShapeSubroutine, "Sub"},
		{"D(((Cd)))", diagram.BlockShapeDoubleCircle, "Cd"},
		{"Y[(Cyl)]", diagram.BlockShapeCylinder, "Cyl"},
	}
	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			d, err := Parse(strings.NewReader("block-beta\n    " + tc.input))
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			if len(d.Nodes) != 1 {
				t.Fatalf("nodes = %v", d.Nodes)
			}
			n := d.Nodes[0]
			if n.Shape != tc.want {
				t.Errorf("shape = %v, want %v", n.Shape, tc.want)
			}
			if n.Label != tc.text {
				t.Errorf("label = %q, want %q", n.Label, tc.text)
			}
		})
	}
}

// `block:ID ... end` opens a nested group. Items inside the group
// flow into the group's Items slice, while flat Nodes still
// receives every node for renderer-side ID lookup.
func TestParseGroupBlock(t *testing.T) {
	d, err := Parse(strings.NewReader(`block-beta
columns 3
A
block:G
  X Y
end
B`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if d.Columns != 3 {
		t.Errorf("columns = %d", d.Columns)
	}
	if len(d.Items) != 3 {
		t.Fatalf("top-level items = %d, want 3", len(d.Items))
	}
	if d.Items[0].Kind != diagram.BlockItemNodeRef || d.Items[0].NodeID != "A" {
		t.Errorf("items[0] = %+v", d.Items[0])
	}
	if d.Items[1].Kind != diagram.BlockItemGroup {
		t.Fatalf("items[1] = %+v, want group", d.Items[1])
	}
	g := d.Items[1].Group
	if g.ID != "G" || len(g.Items) != 2 {
		t.Errorf("group = %+v", g)
	}
	if g.Items[0].NodeID != "X" || g.Items[1].NodeID != "Y" {
		t.Errorf("group items = %+v", g.Items)
	}
	if d.Items[2].Kind != diagram.BlockItemNodeRef || d.Items[2].NodeID != "B" {
		t.Errorf("items[2] = %+v", d.Items[2])
	}
	// Flat Nodes must include every id including those inside the
	// group, so renderer/ensureNode lookups stay consistent.
	gotIDs := []string{}
	for _, n := range d.Nodes {
		gotIDs = append(gotIDs, n.ID)
	}
	wantIDs := []string{"A", "X", "Y", "B"}
	if len(gotIDs) != len(wantIDs) {
		t.Errorf("Nodes ids = %v, want %v", gotIDs, wantIDs)
	}
	for i, w := range wantIDs {
		if i < len(gotIDs) && gotIDs[i] != w {
			t.Errorf("Nodes[%d].ID = %q, want %q", i, gotIDs[i], w)
		}
	}
}

// `block:ID:N` and `block:ID["label"]:N` populate the group's
// Width and Label fields.
func TestParseGroupWidthAndLabel(t *testing.T) {
	d, err := Parse(strings.NewReader(`block-beta
block:wide:3
  A
end
block:nice["Display"]:2
  B
end`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(d.Items) != 2 {
		t.Fatalf("items = %v", d.Items)
	}
	g0 := d.Items[0].Group
	if g0.ID != "wide" || g0.Width != 3 || g0.Label != "" {
		t.Errorf("g0 = %+v", g0)
	}
	g1 := d.Items[1].Group
	if g1.ID != "nice" || g1.Width != 2 || g1.Label != "Display" {
		t.Errorf("g1 = %+v", g1)
	}
}

// Nested groups push and pop their own scope. A `columns N` line
// inside an inner group sets the inner group's Columns rather
// than the outer's.
func TestParseNestedGroups(t *testing.T) {
	d, err := Parse(strings.NewReader(`block-beta
columns 4
block:outer
  columns 2
  block:inner
    A B
  end
end`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if d.Columns != 4 {
		t.Errorf("top columns = %d, want 4", d.Columns)
	}
	outer := d.Items[0].Group
	if outer.ID != "outer" || outer.Columns != 2 {
		t.Errorf("outer = %+v", outer)
	}
	if len(outer.Items) != 1 || outer.Items[0].Kind != diagram.BlockItemGroup {
		t.Fatalf("outer.Items = %+v", outer.Items)
	}
	inner := outer.Items[0].Group
	if inner.ID != "inner" || len(inner.Items) != 2 {
		t.Errorf("inner = %+v", inner)
	}
}

// `space` and `space:N` emit a Space item with the requested
// column count (default 1). Bare `space` may also appear inline
// among other tokens on a row.
func TestParseSpaceTokens(t *testing.T) {
	d, err := Parse(strings.NewReader(`block-beta
A space B
space:3
C`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	wantKinds := []diagram.BlockItemKind{
		diagram.BlockItemNodeRef, // A
		diagram.BlockItemSpace,   // inline space
		diagram.BlockItemNodeRef, // B
		diagram.BlockItemSpace,   // space:3
		diagram.BlockItemNodeRef, // C
	}
	if len(d.Items) != len(wantKinds) {
		t.Fatalf("items = %v", d.Items)
	}
	for i, want := range wantKinds {
		if d.Items[i].Kind != want {
			t.Errorf("items[%d].Kind = %v, want %v", i, d.Items[i].Kind, want)
		}
	}
	if d.Items[1].Cols != 1 {
		t.Errorf("inline space cols = %d", d.Items[1].Cols)
	}
	if d.Items[3].Cols != 3 {
		t.Errorf("space:3 cols = %d", d.Items[3].Cols)
	}
}

// `id:N` width suffix on a node persists onto BlockNode.Width.
func TestParseNodeWidthSuffix(t *testing.T) {
	d, err := Parse(strings.NewReader(`block-beta
columns 4
A:2 B[Wide]:3 C`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	want := map[string]int{"A": 2, "B": 3, "C": 0}
	for _, n := range d.Nodes {
		if got := n.Width; got != want[n.ID] {
			t.Errorf("%s width = %d, want %d", n.ID, got, want[n.ID])
		}
	}
	if d.Nodes[1].Label != "Wide" {
		t.Errorf("B label = %q", d.Nodes[1].Label)
	}
}

// Unbalanced 'end' (no open group) and missing 'end' (open group
// at EOF) both surface as parse errors with line context.
func TestParseGroupBalanceErrors(t *testing.T) {
	for _, src := range []string{
		"block-beta\nend",
		"block-beta\nblock:G\n  A",
	} {
		if _, err := Parse(strings.NewReader(src)); err == nil {
			t.Errorf("expected error for:\n%s", src)
		}
	}
}
