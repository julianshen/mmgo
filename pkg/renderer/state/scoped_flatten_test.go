package state

import (
	"strings"
	"testing"

	"github.com/julianshen/mmgo/pkg/diagram"
	"github.com/julianshen/mmgo/pkg/layout"
	parser "github.com/julianshen/mmgo/pkg/parser/state"
	"github.com/julianshen/mmgo/pkg/textmeasure"
)

// TestFlattenScopedLayoutNested verifies the global-coordinate view of
// the nested.mmd layout: each composite is nested inside its parent,
// each pseudo-state lives inside its declared scope's composite, and
// the end ring for `third --> [*]` sits inside Third (not floating
// outside First as in the legacy renderer).
func TestFlattenScopedLayoutNested(t *testing.T) {
	src := `stateDiagram-v2
    [*] --> First
    state First {
        [*] --> Second
        state Second {
            [*] --> second
            second --> Third
            state Third {
                [*] --> third
                third --> [*]
            }
        }
    }`
	d, err := parser.Parse(strings.NewReader(src))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	ruler, err := textmeasure.NewDefaultRuler()
	if err != nil {
		t.Fatalf("ruler: %v", err)
	}
	defer func() { _ = ruler.Close() }()

	root := layoutScope("", d.States, d.Transitions, ruler, defaultFontSize, layout.Options{})
	flat := flattenScopedLayout(root)

	// Composite rects: First, Second, Third in nesting order.
	byID := make(map[string]flatComposite, len(flat.Composites))
	for _, c := range flat.Composites {
		byID[c.ID] = c
	}
	first, ok := byID["First"]
	if !ok {
		t.Fatalf("First composite missing from flat layout: %+v", flat.Composites)
	}
	second, ok := byID["Second"]
	if !ok {
		t.Fatalf("Second composite missing: %+v", flat.Composites)
	}
	third, ok := byID["Third"]
	if !ok {
		t.Fatalf("Third composite missing: %+v", flat.Composites)
	}

	// Containment: Second fully inside First, Third fully inside Second.
	if !rectContains(first, second) {
		t.Errorf("Second not contained in First: first=%v second=%v", first, second)
	}
	if !rectContains(second, third) {
		t.Errorf("Third not contained in Second: second=%v third=%v", second, third)
	}

	// Depth grows with nesting.
	if first.Depth >= second.Depth || second.Depth >= third.Depth {
		t.Errorf("composite depths should grow inward: First=%d Second=%d Third=%d",
			first.Depth, second.Depth, third.Depth)
	}

	// The end pseudo for `third --> [*]` must sit inside Third's rect.
	endsInThird := 0
	for id, owner := range flat.PseudoOwner {
		if owner != "Third" {
			continue
		}
		if !strings.Contains(id, "_end_") {
			continue
		}
		endsInThird++
		n, ok := flat.Nodes[id]
		if !ok {
			t.Fatalf("Third end pseudo %q missing from flat nodes", id)
		}
		if !pointInRect(n.X, n.Y, third) {
			t.Errorf("end pseudo %q at (%.1f,%.1f) is outside Third rect %v",
				id, n.X, n.Y, third)
		}
	}
	if endsInThird != 1 {
		t.Errorf("expected exactly one end pseudo owned by Third, got %d", endsInThird)
	}

	// The outer `[*] --> First` start pseudo must be OUTSIDE First's rect.
	outerStartFound := false
	for id, owner := range flat.PseudoOwner {
		if owner != "" || !strings.Contains(id, "_start_") {
			continue
		}
		outerStartFound = true
		n := flat.Nodes[id]
		if pointInRect(n.X, n.Y, first) {
			t.Errorf("outer start pseudo %q at (%.1f,%.1f) should NOT be inside First %v",
				id, n.X, n.Y, first)
		}
	}
	if !outerStartFound {
		t.Errorf("outer start pseudo not found in flat layout")
	}

	// Leaf `third` must live inside Third (and therefore inside Second/First).
	if n, ok := flat.Nodes["third"]; ok {
		if !pointInRect(n.X, n.Y, third) {
			t.Errorf("third leaf at (%.1f,%.1f) outside Third %v", n.X, n.Y, third)
		}
	} else {
		t.Errorf("third leaf missing from flat nodes")
	}
}

// Each scope's dagre graph starts EdgeID.ID at 0, so two scopes can
// produce identical (From, To, ID) keys. flattenScopedLayout must
// preserve every edge by bumping the colliding ID, and each edge's
// originating scope must be recorded in EdgeScopes for renderEdges'
// label disambiguation. The natural parser path would dedup
// shared states across scopes, so this test bypasses Parse() and
// hands layoutScope a hand-built tree where the same `a --> b` edge
// genuinely appears in two sibling composites.
func TestFlattenScopedLayoutPreservesCollidingEdgeIDs(t *testing.T) {
	states := []diagram.StateDef{
		{ID: "First", Label: "First", Children: []diagram.StateDef{
			{ID: "a", Label: "a"}, {ID: "b", Label: "b"},
		}},
		{ID: "Second", Label: "Second", Children: []diagram.StateDef{
			{ID: "a", Label: "a"}, {ID: "b", Label: "b"},
		}},
	}
	transitions := []diagram.StateTransition{
		{From: "a", To: "b", Label: "first-edge", Scope: "First"},
		{From: "a", To: "b", Label: "second-edge", Scope: "Second"},
	}
	ruler, err := textmeasure.NewDefaultRuler()
	if err != nil {
		t.Fatalf("ruler: %v", err)
	}
	defer func() { _ = ruler.Close() }()

	root := layoutScope("", states, transitions, ruler, defaultFontSize, layout.Options{})
	flat := flattenScopedLayout(root)

	var firstScope, secondScope bool
	for _, sc := range flat.EdgeScopes {
		switch sc {
		case "First":
			firstScope = true
		case "Second":
			secondScope = true
		}
	}
	if !firstScope || !secondScope {
		t.Errorf("EdgeScopes missing one of First/Second; got %v", flat.EdgeScopes)
	}
	if len(flat.Edges) != len(flat.EdgeScopes) {
		t.Errorf("Edges (%d) and EdgeScopes (%d) counts should match",
			len(flat.Edges), len(flat.EdgeScopes))
	}
}

func pointInRect(x, y float64, c flatComposite) bool {
	return x >= c.X && x <= c.X+c.Width && y >= c.Y && y <= c.Y+c.Height
}

func rectContains(outer, inner flatComposite) bool {
	return inner.X >= outer.X &&
		inner.Y >= outer.Y &&
		inner.X+inner.Width <= outer.X+outer.Width &&
		inner.Y+inner.Height <= outer.Y+outer.Height
}
