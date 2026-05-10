package mindmap

import (
	"fmt"
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
	if len(child.CSSClasses) != 1 || child.CSSClasses[0] != "urgent" {
		t.Errorf("child classes = %v, want [urgent]", child.CSSClasses)
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
	if len(d.Root.CSSClasses) != 0 {
		t.Errorf("class before node should be ignored, got %v", d.Root.CSSClasses)
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

// accTitle / accDescr lines populate the matching AST fields and
// don't pollute the node tree.
func TestParseAccessibility(t *testing.T) {
	d, err := Parse(strings.NewReader(`mindmap
accTitle: Mind Map
accDescr: Top-level concepts
Root
    A
    B`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if d.AccTitle != "Mind Map" {
		t.Errorf("accTitle = %q", d.AccTitle)
	}
	if d.AccDescr != "Top-level concepts" {
		t.Errorf("accDescr = %q", d.AccDescr)
	}
	if d.Root == nil || d.Root.Text != "Root" || len(d.Root.Children) != 2 {
		t.Errorf("tree corrupted by accessibility lines: %+v", d.Root)
	}
}

// Backtick-wrapped labels have the backticks stripped so the
// contents (markdown) reach the renderer cleanly.
func TestParseBacktickMarkdown(t *testing.T) {
	d, err := Parse(strings.NewReader("mindmap\n    Root[`**Bold** title`]"))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if d.Root.Text != "**Bold** title" {
		t.Errorf("text = %q, want stripped backticks", d.Root.Text)
	}
}

// Literal `\n` inside a label becomes a real newline so the
// renderer can split it into multiple lines.
func TestParseLabelNewline(t *testing.T) {
	d, err := Parse(strings.NewReader(`mindmap
    Root[Line one\nLine two]`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if d.Root.Text != "Line one\nLine two" {
		t.Errorf("text = %q", d.Root.Text)
	}
}

// Historical `!text!` bang form parses to MindmapShapeBang in
// addition to the canonical `))text((`.
func TestParseBangFallback(t *testing.T) {
	d, err := Parse(strings.NewReader("mindmap\n    !Hot!"))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if d.Root.Shape != diagram.MindmapShapeBang {
		t.Errorf("shape = %v, want bang", d.Root.Shape)
	}
	if d.Root.Text != "Hot" {
		t.Errorf("text = %q", d.Root.Text)
	}
}

// A second top-level entry is rejected — Mermaid mindmaps allow
// only one root.
func TestParseMultiRootError(t *testing.T) {
	_, err := Parse(strings.NewReader(`mindmap
Root1
    A
Root2`))
	if err == nil {
		t.Error("expected error for second root")
	}
}

// `classDef name css` declarations populate the diagram's
// CSSClasses map and `:::a b c` attaches multiple class names to
// a node in document order.
func TestParseClassDefAndMultipleClasses(t *testing.T) {
	d, err := Parse(strings.NewReader(`mindmap
classDef hot fill:#f00
classDef bold stroke:#000,stroke-width:3
Root
    A
        :::hot bold`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if d.CSSClasses["hot"] != "fill:#f00" {
		t.Errorf("hot classdef = %q", d.CSSClasses["hot"])
	}
	if d.CSSClasses["bold"] != "stroke:#000;stroke-width:3" {
		t.Errorf("bold classdef = %q", d.CSSClasses["bold"])
	}
	a := d.Root.Children[0]
	if len(a.CSSClasses) != 2 || a.CSSClasses[0] != "hot" || a.CSSClasses[1] != "bold" {
		t.Errorf("A classes = %v, want [hot bold]", a.CSSClasses)
	}
}

// `style ID css` lines accumulate on the diagram's Styles slice.
func TestParseStyleRule(t *testing.T) {
	d, err := Parse(strings.NewReader(`mindmap
Root
    Body
style Body fill:#fee,stroke:#900`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(d.Styles) != 1 {
		t.Fatalf("styles = %v", d.Styles)
	}
	got := d.Styles[0]
	if got.NodeID != "Body" || got.CSS != "fill:#fee;stroke:#900" {
		t.Errorf("style = %+v", got)
	}
}

func TestParseQuotedDescription(t *testing.T) {
	d, err := Parse(strings.NewReader(`mindmap
    root["String containing []"]`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if d.Root.ID != "root" {
		t.Errorf("id = %q, want root", d.Root.ID)
	}
	if d.Root.Text != "String containing []" {
		t.Errorf("text = %q, want 'String containing []'", d.Root.Text)
	}
	if d.Root.Shape != diagram.MindmapShapeSquare {
		t.Errorf("shape = %v, want square", d.Root.Shape)
	}
}

func TestParseQuotedDescriptionInChild(t *testing.T) {
	d, err := Parse(strings.NewReader(`mindmap
    root["String containing []"]
      child1("String containing ()")`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if d.Root.Text != "String containing []" {
		t.Errorf("root text = %q", d.Root.Text)
	}
	if d.Root.Shape != diagram.MindmapShapeSquare {
		t.Errorf("root shape = %v, want square", d.Root.Shape)
	}
	if len(d.Root.Children) != 1 {
		t.Fatalf("want 1 child, got %d", len(d.Root.Children))
	}
	child := d.Root.Children[0]
	if child.Text != "String containing ()" {
		t.Errorf("child text = %q, want 'String containing ()'", child.Text)
	}
	if child.Shape != diagram.MindmapShapeRound {
		t.Errorf("child shape = %v, want round", child.Shape)
	}
}

func TestParseLeadingEmptyLines(t *testing.T) {
	cases := []string{
		"\n \nmindmap\nroot\n A\n \n\n B",
		"\n\n\nmindmap\nroot\n A\n \n\n B",
	}
	for i, input := range cases {
		t.Run(fmt.Sprintf("case_%d", i), func(t *testing.T) {
			d, err := Parse(strings.NewReader(input))
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			if d.Root == nil || d.Root.Text != "root" {
				t.Errorf("root = %+v", d.Root)
			}
			if len(d.Root.Children) != 2 {
				t.Errorf("want 2 children, got %d", len(d.Root.Children))
			}
		})
	}
}

func TestParseQuotedDescriptionAllShapes(t *testing.T) {
	cases := []struct {
		input string
		id    string
		shape diagram.MindmapNodeShape
		text  string
	}{
		{"(\"round\")", "round", diagram.MindmapShapeRound, "round"},
		{"[\"square\"]", "square", diagram.MindmapShapeSquare, "square"},
		{"((\"circle\"))", "circle", diagram.MindmapShapeCircle, "circle"},
		{"{{\"hexagon\"}}", "hexagon", diagram.MindmapShapeHexagon, "hexagon"},
		{"))\"bang\"((", "bang", diagram.MindmapShapeBang, "bang"},
		{")\"cloud\"(", "cloud", diagram.MindmapShapeCloud, "cloud"},
		{"(-\"cloud2\"-)", "cloud2", diagram.MindmapShapeCloud, "cloud2"},
		// explicit ID + quoted description
		{"id(\"quoted\")", "id", diagram.MindmapShapeRound, "quoted"},
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
			if d.Root.Shape != tc.shape {
				t.Errorf("shape = %v, want %v", d.Root.Shape, tc.shape)
			}
			if d.Root.Text != tc.text {
				t.Errorf("text = %q, want %q", d.Root.Text, tc.text)
			}
		})
	}
}

// NSTR2 form: backtick-wrapped description inside quotes.
func TestParseQuotedBacktickDescription(t *testing.T) {
	d, err := Parse(strings.NewReader("mindmap\n    root[\"`hello world`\"]"))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if d.Root.Text != "hello world" {
		t.Errorf("text = %q, want 'hello world'", d.Root.Text)
	}
	if d.Root.Shape != diagram.MindmapShapeSquare {
		t.Errorf("shape = %v, want square", d.Root.Shape)
	}
}

// A node without an id but with a quoted description uses the
// description as its id, matching Mermaid's nodeWithoutId rule.
func TestParseQuotedDescriptionNoID(t *testing.T) {
	d, err := Parse(strings.NewReader(`mindmap
    ["hello world"]`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if d.Root.ID != "hello world" {
		t.Errorf("id = %q, want 'hello world'", d.Root.ID)
	}
	if d.Root.Text != "hello world" {
		t.Errorf("text = %q, want 'hello world'", d.Root.Text)
	}
}

// Fallback behavior when quoted descriptions are malformed or
// contain unexpected content (empty, unclosed, trailing junk, etc.).
func TestParseQuotedDescriptionEdgeCases(t *testing.T) {
	cases := []struct {
		name      string
		input     string
		wantText  string
		wantShape diagram.MindmapNodeShape
	}{
		{
			name:      "empty quotes parsed as literal empty string in shape",
			input:     `mindmap
    [""]`,
			wantText:  `""`,
			wantShape: diagram.MindmapShapeSquare,
		},
		{
			name:      "unclosed quote falls back to literal text",
			input:     `mindmap
    ["unclosed]`,
			wantText:  `"unclosed`,
			wantShape: diagram.MindmapShapeSquare,
		},
		{
			name:      "trailing junk after quoted close falls back to default",
			input:     `mindmap
    ["hello"]extra`,
			wantText:  `["hello"]extra`,
			wantShape: diagram.MindmapShapeDefault,
		},
		{
			name:      "whitespace-only quotes accepted as content",
			input:     `mindmap
    ["   "]`,
			wantText:  "   ",
			wantShape: diagram.MindmapShapeSquare,
		},
		{
			name:      "quotes inside non-quoted label not intercepted",
			input:     `mindmap
    root[He said "hello"]`,
			wantText:  `He said "hello"`,
			wantShape: diagram.MindmapShapeSquare,
		},
		{
			name:      "unclosed backtick quote falls back to literal",
			input:     "mindmap\n    [\"`hello]",
			wantText:  "\"`hello",
			wantShape: diagram.MindmapShapeSquare,
		},
		{
			name:      "bare backtick inside NSTR2 falls back to literal",
			input:     "mindmap\n    [\"`hello``world`\"]",
			wantText:  "\"`hello``world`\"",
			wantShape: diagram.MindmapShapeSquare,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			d, err := Parse(strings.NewReader(tc.input))
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			if d.Root.Text != tc.wantText {
				t.Errorf("text = %q, want %q", d.Root.Text, tc.wantText)
			}
			if d.Root.Shape != tc.wantShape {
				t.Errorf("shape = %v, want %v", d.Root.Shape, tc.wantShape)
			}
		})
	}
}
