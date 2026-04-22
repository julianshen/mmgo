package mindmap

import (
	"strings"
	"testing"

	"github.com/julianshen/mmgo/pkg/diagram"
)

func TestParseHeaderRequired(t *testing.T) {
	_, err := Parse(strings.NewReader("Root"))
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestParseEmptyDiagram(t *testing.T) {
	d, err := Parse(strings.NewReader("mindmap"))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if d.Root != nil {
		t.Error("empty mindmap should have nil root")
	}
}

func TestParseRootOnly(t *testing.T) {
	input := "mindmap\n    Root"
	d, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if d.Root == nil || d.Root.Text != "Root" {
		t.Errorf("root = %+v", d.Root)
	}
}

func TestParseHierarchy(t *testing.T) {
	input := `mindmap
    Root
        Child 1
            Grandchild
        Child 2`
	d, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if d.Root.Text != "Root" {
		t.Errorf("root = %q", d.Root.Text)
	}
	if len(d.Root.Children) != 2 {
		t.Fatalf("want 2 children, got %d", len(d.Root.Children))
	}
	if d.Root.Children[0].Text != "Child 1" {
		t.Errorf("child[0] = %q", d.Root.Children[0].Text)
	}
	if len(d.Root.Children[0].Children) != 1 {
		t.Fatalf("want 1 grandchild, got %d", len(d.Root.Children[0].Children))
	}
	if d.Root.Children[0].Children[0].Text != "Grandchild" {
		t.Errorf("grandchild = %q", d.Root.Children[0].Children[0].Text)
	}
	if d.Root.Children[1].Text != "Child 2" {
		t.Errorf("child[1] = %q", d.Root.Children[1].Text)
	}
}

func TestParseNodeShapes(t *testing.T) {
	cases := []struct {
		input string
		want  diagram.MindmapNodeShape
		text  string
	}{
		{"plain text", diagram.MindmapShapeDefault, "plain text"},
		{"(rounded)", diagram.MindmapShapeRound, "rounded"},
		{"[squared]", diagram.MindmapShapeSquare, "squared"},
		{"((circle))", diagram.MindmapShapeCircle, "circle"},
		{"{{hexagon}}", diagram.MindmapShapeHexagon, "hexagon"},
		{"))bang((", diagram.MindmapShapeBang, "bang"},
		{")cloud(", diagram.MindmapShapeCloud, "cloud"},
		{"(-cloud-)", diagram.MindmapShapeCloud, "cloud"},
	}
	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			input := "mindmap\n    " + tc.input
			d, err := Parse(strings.NewReader(input))
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			if d.Root.Shape != tc.want {
				t.Errorf("shape = %v, want %v", d.Root.Shape, tc.want)
			}
			if d.Root.Text != tc.text {
				t.Errorf("text = %q, want %q", d.Root.Text, tc.text)
			}
		})
	}
}

func TestParseNodeWithID(t *testing.T) {
	cases := []struct {
		input string
		id    string
		text  string
		shape diagram.MindmapNodeShape
	}{
		{"A[Hello]", "A", "Hello", diagram.MindmapShapeSquare},
		{"B(World)", "B", "World", diagram.MindmapShapeRound},
		{"C((Circle))", "C", "Circle", diagram.MindmapShapeCircle},
		{"D{{Hex}}", "D", "Hex", diagram.MindmapShapeHexagon},
	}
	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			input := "mindmap\n    " + tc.input
			d, err := Parse(strings.NewReader(input))
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			if d.Root.ID != tc.id {
				t.Errorf("id = %q, want %q", d.Root.ID, tc.id)
			}
			if d.Root.Text != tc.text {
				t.Errorf("text = %q, want %q", d.Root.Text, tc.text)
			}
			if d.Root.Shape != tc.shape {
				t.Errorf("shape = %v, want %v", d.Root.Shape, tc.shape)
			}
		})
	}
}

func TestParseIconDecoration(t *testing.T) {
	input := `mindmap
    Root
        Child
            ::icon(fa fa-user)`
	d, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(d.Root.Children) != 1 {
		t.Fatalf("want 1 child, got %d", len(d.Root.Children))
	}
	child := d.Root.Children[0]
	if child.Text != "Child" {
		t.Errorf("child text = %q", child.Text)
	}
	if child.Icon != "fa fa-user" {
		t.Errorf("child icon = %q, want %q", child.Icon, "fa fa-user")
	}
}

func TestParseClassDecoration(t *testing.T) {
	input := `mindmap
    Root
        Child
            :::urgent`
	d, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	child := d.Root.Children[0]
	if child.Class != "urgent" {
		t.Errorf("child class = %q, want %q", child.Class, "urgent")
	}
}

func TestParseComments(t *testing.T) {
	input := `mindmap
    %% comment
    Root
        Child %% trailing`
	d, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if d.Root.Text != "Root" {
		t.Errorf("root = %q", d.Root.Text)
	}
	if len(d.Root.Children) != 1 || d.Root.Children[0].Text != "Child" {
		t.Errorf("children = %+v", d.Root.Children)
	}
}

func TestParseDeepNesting(t *testing.T) {
	input := `mindmap
    A
        B
            C
                D
        E`
	d, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(d.Root.Children) != 2 {
		t.Fatalf("A should have 2 children, got %d", len(d.Root.Children))
	}
	b := d.Root.Children[0]
	if len(b.Children) != 1 || b.Children[0].Text != "C" {
		t.Errorf("B children: %+v", b.Children)
	}
	c := b.Children[0]
	if len(c.Children) != 1 || c.Children[0].Text != "D" {
		t.Errorf("C children: %+v", c.Children)
	}
}

func TestParseIconBeforeNode(t *testing.T) {
	input := "mindmap\n    ::icon(fa fa-check)\n    Root"
	d, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if d.Root.Icon != "" {
		t.Errorf("icon before node should be ignored, got %q", d.Root.Icon)
	}
}

func TestParseClassBeforeNode(t *testing.T) {
	input := "mindmap\n    :::urgent\n    Root"
	d, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if d.Root.Class != "" {
		t.Errorf("class before node should be ignored, got %q", d.Root.Class)
	}
}

func TestParseUnclosedIcon(t *testing.T) {
	input := "mindmap\n    Root\n        Child\n            ::icon(no-close"
	d, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	child := d.Root.Children[0]
	if child.Icon != "" {
		t.Errorf("unclosed icon should not be set, got %q", child.Icon)
	}
}

func TestParseEmptyShape(t *testing.T) {
	input := "mindmap\n    A[]"
	d, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if d.Root.Shape != diagram.MindmapShapeDefault {
		t.Errorf("empty brackets should be default shape, got %v", d.Root.Shape)
	}
}

func TestParseMissingHeader(t *testing.T) {
	_, err := Parse(strings.NewReader(""))
	if err == nil {
		t.Fatal("expected error for empty input")
	}
}
