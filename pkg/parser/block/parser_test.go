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
