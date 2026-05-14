package state

import (
	"strings"
	"testing"

	"github.com/julianshen/mmgo/pkg/layout"
	parser "github.com/julianshen/mmgo/pkg/parser/state"
	"github.com/julianshen/mmgo/pkg/textmeasure"
)

// TestLayoutScopeNestsPseudoStates is the regression test for the
// nested.svg bug: every `[*]` must be laid out inside the scope where
// it was written, not flattened to the root. This drives the parser's
// new Scope field through to the layout and verifies coordinates.
func TestLayoutScopeNestsPseudoStates(t *testing.T) {
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

	if root.scopeID != "" {
		t.Errorf("root.scopeID = %q, want \"\"", root.scopeID)
	}

	// Root scope owns exactly one transition: outer `[*]` --> First.
	if len(root.pseudoNodes) != 1 {
		t.Errorf("root pseudoNodes = %d, want 1: %+v", len(root.pseudoNodes), root.pseudoNodes)
	}
	for _, info := range root.pseudoNodes {
		if info.Kind != pseudoStart {
			t.Errorf("root pseudo kind = %v, want pseudoStart", info.Kind)
		}
	}

	// Root must have First as a composite child sub-layout.
	first := root.compositeOf("First")
	if first == nil {
		t.Fatalf("root missing composite child First")
	}
	// First contains its own `[*]` --> Second transition.
	if got := countStartPseudos(first); got != 1 {
		t.Errorf("First scope start pseudos = %d, want 1", got)
	}

	second := first.compositeOf("Second")
	if second == nil {
		t.Fatalf("First missing composite child Second")
	}
	// Second contains `[*] --> second` and `second --> Third`. Only the
	// first creates a pseudo; the second goes between real nodes.
	if got := countStartPseudos(second); got != 1 {
		t.Errorf("Second scope start pseudos = %d, want 1", got)
	}

	third := second.compositeOf("Third")
	if third == nil {
		t.Fatalf("Second missing composite child Third")
	}
	// Third contains `[*] --> third` (start) and `third --> [*]` (end).
	starts, ends := 0, 0
	for _, info := range third.pseudoNodes {
		switch info.Kind {
		case pseudoStart:
			starts++
		case pseudoEnd:
			ends++
		}
	}
	if starts != 1 || ends != 1 {
		t.Errorf("Third pseudos: starts=%d ends=%d, want 1 and 1", starts, ends)
	}

	// Every scope's dagre result must include all of its own real states
	// AND its own pseudo nodes — but none belonging to a different scope.
	if _, ok := root.result.Nodes["First"]; !ok {
		t.Errorf("root.result missing First as a node")
	}
	for id := range third.pseudoNodes {
		if _, ok := root.result.Nodes[id]; ok {
			t.Errorf("root.result should not contain Third's pseudo %q", id)
		}
		if _, ok := second.result.Nodes[id]; ok {
			t.Errorf("second.result should not contain Third's pseudo %q", id)
		}
		if _, ok := third.result.Nodes[id]; !ok {
			t.Errorf("third.result missing its own pseudo %q", id)
		}
	}

	// Every leaf appears in its own scope's result, and only its own.
	leafScope := map[string]*scopedLayout{
		"second": second,
		"third":  third,
	}
	for leaf, scope := range leafScope {
		if _, ok := scope.result.Nodes[leaf]; !ok {
			t.Errorf("leaf %q missing from its scope", leaf)
		}
		// Root should not see leaves directly.
		if _, ok := root.result.Nodes[leaf]; ok {
			t.Errorf("root.result should not contain leaf %q (lives inside %s)", leaf, scope.scopeID)
		}
	}

	// Composite size must grow with descendant content: Third < Second < First.
	if !(third.height < second.height && second.height < first.height) {
		t.Errorf("composite heights should nest: Third=%v Second=%v First=%v",
			third.height, second.height, first.height)
	}
}

func countStartPseudos(s *scopedLayout) int {
	n := 0
	for _, info := range s.pseudoNodes {
		if info.Kind == pseudoStart {
			n++
		}
	}
	return n
}
