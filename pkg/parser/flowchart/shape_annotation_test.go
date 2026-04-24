package flowchart

import (
	"strings"
	"testing"

	"github.com/julianshen/mmgo/pkg/diagram"
)

// nodeShapeFromMmd parses a single-node flowchart and returns the
// resolved node so each table-driven test stays a one-liner.
func nodeShapeFromMmd(t *testing.T, body string) diagram.Node {
	t.Helper()
	d, err := Parse(strings.NewReader("flowchart TD\n    " + body))
	if err != nil {
		t.Fatalf("parse %q: %v", body, err)
	}
	if len(d.Nodes) != 1 {
		t.Fatalf("expected 1 node, got %d (%+v)", len(d.Nodes), d.Nodes)
	}
	return d.Nodes[0]
}

// Every alias in the table must resolve to its declared NodeShape.
// Failures here usually mean a typo in shapeAliases or a constant
// renumbering that desynced the map values.
func TestShapeAnnotationAliases(t *testing.T) {
	cases := []struct {
		alias string
		want  diagram.NodeShape
	}{
		{"rect", diagram.NodeShapeRectangle},
		{"rectangle", diagram.NodeShapeRectangle},
		{"proc", diagram.NodeShapeRectangle},
		{"process", diagram.NodeShapeRectangle},
		{"rounded", diagram.NodeShapeRoundedRectangle},
		{"event", diagram.NodeShapeRoundedRectangle},
		{"stadium", diagram.NodeShapeStadium},
		{"terminal", diagram.NodeShapeStadium},
		{"pill", diagram.NodeShapeStadium},
		{"fr-rect", diagram.NodeShapeSubroutine},
		{"subprocess", diagram.NodeShapeSubroutine},
		{"subroutine", diagram.NodeShapeSubroutine},
		{"cyl", diagram.NodeShapeCylinder},
		{"cylinder", diagram.NodeShapeCylinder},
		{"db", diagram.NodeShapeCylinder},
		{"database", diagram.NodeShapeCylinder},
		{"circ", diagram.NodeShapeCircle},
		{"circle", diagram.NodeShapeCircle},
		{"diam", diagram.NodeShapeDiamond},
		{"diamond", diagram.NodeShapeDiamond},
		{"decision", diagram.NodeShapeDiamond},
		{"question", diagram.NodeShapeDiamond},
		{"hex", diagram.NodeShapeHexagon},
		{"hexagon", diagram.NodeShapeHexagon},
		{"prepare", diagram.NodeShapeHexagon},
		{"lean-r", diagram.NodeShapeParallelogram},
		{"lean-right", diagram.NodeShapeParallelogram},
		{"in-out", diagram.NodeShapeParallelogram},
		{"lean-l", diagram.NodeShapeParallelogramAlt},
		{"lean-left", diagram.NodeShapeParallelogramAlt},
		{"out-in", diagram.NodeShapeParallelogramAlt},
		{"trap-b", diagram.NodeShapeTrapezoid},
		{"trapezoid", diagram.NodeShapeTrapezoid},
		{"priority", diagram.NodeShapeTrapezoid},
		{"trap-t", diagram.NodeShapeTrapezoidAlt},
		{"trapezoid-top", diagram.NodeShapeTrapezoidAlt},
		{"inv-trapezoid", diagram.NodeShapeTrapezoidAlt},
		{"manual", diagram.NodeShapeTrapezoidAlt},
		{"asym", diagram.NodeShapeAsymmetric},
		{"asymmetric", diagram.NodeShapeAsymmetric},
		{"dbl-circ", diagram.NodeShapeDoubleCircle},
		{"double-circle", diagram.NodeShapeDoubleCircle},
	}
	for _, tc := range cases {
		t.Run(tc.alias, func(t *testing.T) {
			n := nodeShapeFromMmd(t, "A@{shape:"+tc.alias+"}")
			if n.Shape != tc.want {
				t.Errorf("shape:%s ⇒ %v, want %v", tc.alias, n.Shape, tc.want)
			}
		})
	}
}

// Case-insensitive lookup: `Shape: DIAMOND` should still resolve.
func TestShapeAnnotationCaseInsensitive(t *testing.T) {
	n := nodeShapeFromMmd(t, "A@{shape: DIAMOND}")
	if n.Shape != diagram.NodeShapeDiamond {
		t.Errorf("uppercase shape value not resolved: got %v", n.Shape)
	}
}

// Label key supplies the node's label; surrounding quotes are
// stripped (the value parser unwraps the literal `"..."`).
func TestShapeAnnotationLabel(t *testing.T) {
	n := nodeShapeFromMmd(t, `A@{shape: diamond, label: "Decide"}`)
	if n.Shape != diagram.NodeShapeDiamond {
		t.Errorf("shape = %v, want Diamond", n.Shape)
	}
	if n.Label != "Decide" {
		t.Errorf("label = %q, want %q", n.Label, "Decide")
	}
}

// Quoted label values protect commas from the top-level KV split.
func TestShapeAnnotationLabelWithComma(t *testing.T) {
	n := nodeShapeFromMmd(t, `A@{shape: rect, label: "x, y, z"}`)
	if n.Label != "x, y, z" {
		t.Errorf("label = %q, want %q", n.Label, "x, y, z")
	}
}

// `@{}` after a traditional delimiter overrides the shape; if the
// annotation has no `label:`, the delimiter-supplied label is kept.
func TestShapeAnnotationOverridesTraditionalShape(t *testing.T) {
	n := nodeShapeFromMmd(t, `A["old text"]@{shape: diamond}`)
	if n.Shape != diagram.NodeShapeDiamond {
		t.Errorf("shape = %v, want Diamond", n.Shape)
	}
	if n.Label != "old text" {
		t.Errorf("label = %q, want %q", n.Label, "old text")
	}
}

// Both shape and label override: the annotation wins on both fields.
func TestShapeAnnotationOverridesBoth(t *testing.T) {
	n := nodeShapeFromMmd(t, `A["old"]@{shape: diamond, label: "new"}`)
	if n.Shape != diagram.NodeShapeDiamond {
		t.Errorf("shape = %v, want Diamond", n.Shape)
	}
	if n.Label != "new" {
		t.Errorf("label = %q, want %q", n.Label, "new")
	}
}

// `:::cls` class tokens must come BEFORE `@{}` — the class stripper
// runs first, then the annotation. Verify the combination resolves
// both correctly.
func TestShapeAnnotationWithClass(t *testing.T) {
	n := nodeShapeFromMmd(t, `A:::cls@{shape: diamond}`)
	if n.Shape != diagram.NodeShapeDiamond {
		t.Errorf("shape = %v, want Diamond", n.Shape)
	}
	if len(n.Classes) != 1 || n.Classes[0] != "cls" {
		t.Errorf("classes = %v, want [cls]", n.Classes)
	}
}

// `@{}` on both endpoints of an edge statement — exercises the
// annotation parser through the edge-line code path, not just
// standalone node statements.
func TestShapeAnnotationOnEdgeEndpoints(t *testing.T) {
	d, err := Parse(strings.NewReader(`flowchart TD
    A@{shape: diamond} --> B@{shape: rect}`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(d.Nodes) != 2 {
		t.Fatalf("expected 2 nodes, got %d", len(d.Nodes))
	}
	byID := map[string]diagram.NodeShape{}
	for _, n := range d.Nodes {
		byID[n.ID] = n.Shape
	}
	if byID["A"] != diagram.NodeShapeDiamond {
		t.Errorf("A shape = %v, want Diamond", byID["A"])
	}
	if byID["B"] != diagram.NodeShapeRectangle {
		t.Errorf("B shape = %v, want Rectangle", byID["B"])
	}
}

// Empty `@{}` defaults to Rectangle (the canonical "shape unspecified"
// fallback), matches Mermaid behavior of treating it as a plain node.
func TestShapeAnnotationEmptyDefaultsToRectangle(t *testing.T) {
	n := nodeShapeFromMmd(t, "A@{}")
	if n.Shape != diagram.NodeShapeRectangle {
		t.Errorf("empty @{} ⇒ %v, want Rectangle", n.Shape)
	}
}

// Unknown shape names fail loudly (not silent fallback) so typos
// surface immediately at parse time.
func TestShapeAnnotationUnknownShapeErrors(t *testing.T) {
	_, err := Parse(strings.NewReader("flowchart TD\n    A@{shape: diamon}"))
	if err == nil {
		t.Fatal("expected parse error for unknown shape")
	}
	if !strings.Contains(err.Error(), "unknown shape") {
		t.Errorf("error %q should mention 'unknown shape'", err.Error())
	}
}

// Missing closing brace is a parse error too.
func TestShapeAnnotationUnclosedErrors(t *testing.T) {
	_, err := Parse(strings.NewReader("flowchart TD\n    A@{shape: rect"))
	if err == nil {
		t.Fatal("expected parse error for unclosed @{")
	}
}

// parseKV is exercised end-to-end above, but a focused unit test
// pins the value-quoting and key-lowercasing behavior independently.
func TestParseKV(t *testing.T) {
	cases := []struct {
		in   string
		want map[string]string
	}{
		{"shape:rect", map[string]string{"shape": "rect"}},
		{"shape: rect, label: hello", map[string]string{"shape": "rect", "label": "hello"}},
		{`label: "a, b"`, map[string]string{"label": "a, b"}},
		{"SHAPE: DIAMOND", map[string]string{"shape": "DIAMOND"}}, // value case preserved
		{"", map[string]string{}},
	}
	for _, tc := range cases {
		got, err := parseKV(tc.in)
		if err != nil {
			t.Fatalf("parseKV(%q): %v", tc.in, err)
		}
		if len(got) != len(tc.want) {
			t.Errorf("parseKV(%q) = %v, want %v", tc.in, got, tc.want)
			continue
		}
		for k, v := range tc.want {
			if got[k] != v {
				t.Errorf("parseKV(%q)[%q] = %q, want %q", tc.in, k, got[k], v)
			}
		}
	}
}
