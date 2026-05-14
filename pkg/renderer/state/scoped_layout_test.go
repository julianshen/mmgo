package state

import (
	"math"
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

func TestSanitize(t *testing.T) {
	cases := []struct {
		in, want float64
	}{
		{42, 42},
		{0, 0},
		{-1, 0},
		{math.NaN(), 0},
		{math.Inf(1), 0},
		{math.Inf(-1), 0},
	}
	for _, c := range cases {
		got := sanitize(c.in)
		if got != c.want {
			t.Errorf("sanitize(%v) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestCompositeFillForDepth(t *testing.T) {
	base := "#ffffff"
	// Depth 0 returns the base colour exactly (factor=1.0).
	if got := compositeFillForDepth(base, 0); got != base {
		t.Errorf("depth 0 = %q, want %q", got, base)
	}
	// Each subsequent level darkens; values must be monotonic.
	prev := base
	for depth := 1; depth <= 4; depth++ {
		got := compositeFillForDepth(base, depth)
		if got >= prev {
			t.Errorf("depth %d (%q) not darker than depth %d (%q)", depth, got, depth-1, prev)
		}
		prev = got
	}
}

func TestPseudoIDFormat(t *testing.T) {
	cases := []struct {
		kind     pseudoKind
		scope    string
		idx      int
		want     string
		isStart  bool
		isEnd    bool
		isPseudo bool
	}{
		{pseudoStart, "", 1, "__start_root_1__", true, false, true},
		{pseudoStart, "First", 2, "__start_First_2__", true, false, true},
		{pseudoEnd, "", 1, "__end_root_1__", false, true, true},
		{pseudoEnd, "Third", 3, "__end_Third_3__", false, true, true},
	}
	for _, c := range cases {
		got := pseudoID(c.kind, c.scope, c.idx)
		if got != c.want {
			t.Errorf("pseudoID(%v,%q,%d) = %q, want %q", c.kind, c.scope, c.idx, got, c.want)
		}
		if isStartNode(got) != c.isStart {
			t.Errorf("isStartNode(%q) = %v, want %v", got, !c.isStart, c.isStart)
		}
		if isEndNode(got) != c.isEnd {
			t.Errorf("isEndNode(%q) = %v, want %v", got, !c.isEnd, c.isEnd)
		}
		if isPseudoNode(got) != c.isPseudo {
			t.Errorf("isPseudoNode(%q) = %v, want %v", got, !c.isPseudo, c.isPseudo)
		}
	}
}

func TestDarkenHex(t *testing.T) {
	cases := []struct {
		hex    string
		factor float64
		want   string
	}{
		{"#ffffff", 0.5, "#7f7f7f"},
		{"#000000", 0.9, "#000000"},
		// Malformed inputs return unchanged.
		{"not-hex", 0.5, "not-hex"},
		{"#zz1234", 0.5, "#zz1234"},
		{"#12zz34", 0.5, "#12zz34"},
		{"#1234zz", 0.5, "#1234zz"},
		// factor=0 clamps each channel through the c<0 guard.
		{"#101010", 0, "#000000"},
	}
	for _, c := range cases {
		got := darkenHex(c.hex, c.factor)
		if got != c.want {
			t.Errorf("darkenHex(%q, %v) = %q, want %q", c.hex, c.factor, got, c.want)
		}
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
