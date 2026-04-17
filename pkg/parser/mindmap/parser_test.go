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
		{"((cloud))", diagram.MindmapShapeCloud, "cloud"},
		{"{{bang}}", diagram.MindmapShapeBang, "bang"},
	}
	for _, tc := range cases {
		t.Run(tc.want.String(), func(t *testing.T) {
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
