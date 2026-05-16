package state

import (
	"strings"
	"testing"

	"github.com/julianshen/mmgo/pkg/diagram"
)

func TestParseHeaderRequired(t *testing.T) {
	_, err := Parse(strings.NewReader("A --> B"))
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestParseEmptyDiagram(t *testing.T) {
	d, err := Parse(strings.NewReader("stateDiagram-v2"))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(d.States) != 0 || len(d.Transitions) != 0 {
		t.Errorf("empty: %+v", d)
	}
}

func TestParseV1Header(t *testing.T) {
	d, err := Parse(strings.NewReader("stateDiagram\n    A --> B"))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(d.Transitions) != 1 {
		t.Fatalf("want 1 transition, got %d", len(d.Transitions))
	}
}

func TestParseSimpleTransition(t *testing.T) {
	input := `stateDiagram-v2
    Idle --> Active
    Active --> Idle`
	d, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(d.Transitions) != 2 {
		t.Fatalf("want 2 transitions, got %d", len(d.Transitions))
	}
	if d.Transitions[0].From != "Idle" || d.Transitions[0].To != "Active" {
		t.Errorf("transition[0] = %+v", d.Transitions[0])
	}
}

func TestParseTransitionWithLabel(t *testing.T) {
	input := `stateDiagram-v2
    Idle --> Active : start`
	d, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if d.Transitions[0].Label != "start" {
		t.Errorf("label = %q, want start", d.Transitions[0].Label)
	}
}

func TestParseStartEndStates(t *testing.T) {
	input := `stateDiagram-v2
    [*] --> Idle
    Active --> [*]`
	d, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(d.Transitions) != 2 {
		t.Fatalf("want 2 transitions, got %d", len(d.Transitions))
	}
	if d.Transitions[0].From != "[*]" || d.Transitions[1].To != "[*]" {
		t.Errorf("start/end transitions: %+v", d.Transitions)
	}
}

func TestParseStateDeclaration(t *testing.T) {
	input := `stateDiagram-v2
    state "Waiting for input" as Waiting
    [*] --> Waiting`
	d, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	found := false
	for _, s := range d.States {
		if s.ID == "Waiting" && s.Label == "Waiting for input" {
			found = true
		}
	}
	if !found {
		t.Errorf("state Waiting not found with correct label: %+v", d.States)
	}
}

func TestParseCompositeState(t *testing.T) {
	input := `stateDiagram-v2
    state Active {
        Running --> Paused
        Paused --> Running
    }`
	d, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(d.States) < 1 {
		t.Fatal("expected at least 1 top-level state")
	}
	var active *diagram.StateDef
	for i := range d.States {
		if d.States[i].ID == "Active" {
			active = &d.States[i]
		}
	}
	if active == nil {
		t.Fatal("Active state not found")
	}
	if len(active.Children) < 2 {
		t.Errorf("Active should have child states, got %+v", active.Children)
	}
}

func TestParseCompositeTransitions(t *testing.T) {
	input := `stateDiagram-v2
    state Active {
        Running --> Paused
    }
    [*] --> Active`
	d, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(d.Transitions) < 2 {
		t.Errorf("want >= 2 transitions (inner + outer), got %d", len(d.Transitions))
	}
}

func TestParseSpecialStates(t *testing.T) {
	for _, tc := range []struct {
		decl string
		want diagram.StateKind
	}{
		{"state fork1 <<fork>>", diagram.StateKindFork},
		{"state join1 <<join>>", diagram.StateKindJoin},
		{"state check <<choice>>", diagram.StateKindChoice},
	} {
		t.Run(tc.want.String(), func(t *testing.T) {
			input := "stateDiagram-v2\n    " + tc.decl
			d, err := Parse(strings.NewReader(input))
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			if len(d.States) == 0 {
				t.Fatal("no states")
			}
			if d.States[0].Kind != tc.want {
				t.Errorf("kind = %v, want %v", d.States[0].Kind, tc.want)
			}
		})
	}
}

func TestParseComments(t *testing.T) {
	input := `stateDiagram-v2
    %% comment
    A --> B %% trailing`
	d, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(d.Transitions) != 1 {
		t.Errorf("want 1 transition, got %d", len(d.Transitions))
	}
}

func TestParseUnclosedComposite(t *testing.T) {
	input := `stateDiagram-v2
    state Active {
        A --> B`
	_, err := Parse(strings.NewReader(input))
	if err == nil {
		t.Fatal("expected error for unclosed composite")
	}
}

func TestParseNestedComposite(t *testing.T) {
	input := `stateDiagram-v2
    state Outer {
        state Inner {
            A --> B
        }
    }`
	d, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(d.States) != 1 || d.States[0].ID != "Outer" {
		t.Fatalf("top-level should only contain Outer: %+v", d.States)
	}
	outer := &d.States[0]
	if len(outer.Children) != 1 || outer.Children[0].ID != "Inner" {
		t.Fatalf("Outer should contain Inner: %+v", outer.Children)
	}
	inner := &outer.Children[0]
	if len(inner.Children) < 2 {
		t.Fatalf("Inner should have A and B as children: %+v", inner.Children)
	}
}

// Transition labels that contain `:::` (e.g. documenting an operator
// or quoting code) must not be misread as endpoint CSS shorthand.
// CSS shorthand only applies when `:::` appears on the endpoint
// token, before the label colon.
func TestParseTransitionLabelContainsTripleColon(t *testing.T) {
	cases := []struct {
		src                          string
		from, to, label, fromCSS, toCSS string
	}{
		{
			src: "stateDiagram-v2\nA --> B : foo:::bar",
			from: "A", to: "B", label: "foo:::bar", fromCSS: "", toCSS: "",
		},
		{
			src: "stateDiagram-v2\nA --> B : use ::: operator",
			from: "A", to: "B", label: "use ::: operator", fromCSS: "", toCSS: "",
		},
		{
			src: "stateDiagram-v2\nA --> B:::hot : go",
			from: "A", to: "B", label: "go", fromCSS: "", toCSS: "hot",
		},
		{
			src: "stateDiagram-v2\nA:::cold --> B:::hot : go",
			from: "A", to: "B", label: "go", fromCSS: "cold", toCSS: "hot",
		},
	}
	for _, c := range cases {
		d, err := Parse(strings.NewReader(c.src))
		if err != nil {
			t.Errorf("parse %q: %v", c.src, err)
			continue
		}
		if len(d.Transitions) != 1 {
			t.Errorf("%q: want 1 transition, got %d", c.src, len(d.Transitions))
			continue
		}
		got := d.Transitions[0]
		if got.From != c.from || got.To != c.to || got.Label != c.label {
			t.Errorf("%q: From=%q To=%q Label=%q; want From=%q To=%q Label=%q",
				c.src, got.From, got.To, got.Label, c.from, c.to, c.label)
		}
		// CSS shorthand should attach to the right state (not phantom).
		stateCSS := make(map[string][]string)
		for _, s := range d.States {
			stateCSS[s.ID] = s.CSSClasses
		}
		if c.fromCSS != "" {
			if !containsString(stateCSS[c.from], c.fromCSS) {
				t.Errorf("%q: state %q missing fromCSS %q (have %v)",
					c.src, c.from, c.fromCSS, stateCSS[c.from])
			}
		}
		if c.toCSS != "" {
			if !containsString(stateCSS[c.to], c.toCSS) {
				t.Errorf("%q: state %q missing toCSS %q (have %v)",
					c.src, c.to, c.toCSS, stateCSS[c.to])
			}
		}
	}
}

// containsString lives in parser.go; test file reuses it.

// Quoted state IDs must be tokenised before any `:::`/`:` splitting
// so that:
//   - a transition endpoint `"Long Name"` resolves to the unquoted
//     state ID `Long Name` declared by `state "Long Name"`
//   - a quoted ID containing `:::` (`"A:::B"`) is treated as opaque
//     and not split into endpoint + CSS class
//   - a `-->` substring inside a quoted ID doesn't trigger transition
//     parsing on the declaration line
func TestParseQuotedEndpointConsistency(t *testing.T) {
	cases := []struct {
		name           string
		src            string
		wantStateIDs   []string
		wantFrom, wantTo string
		wantToCSS      string
	}{
		{
			name: "endpoint_matches_declaration",
			src: `stateDiagram-v2
    state "Long Name"
    [*] --> "Long Name"`,
			wantStateIDs: []string{"Long Name"},
			wantFrom:     "[*]",
			wantTo:       "Long Name",
		},
		{
			name: "quoted_id_with_triple_colon_is_opaque",
			src: `stateDiagram-v2
    state "A:::B"
    "A:::B" --> C`,
			wantStateIDs: []string{"A:::B", "C"},
			wantFrom:     "A:::B",
			wantTo:       "C",
		},
		{
			name: "quoted_endpoint_then_css_shorthand",
			src: `stateDiagram-v2
    classDef hot fill:#f00
    state "Long Name"
    [*] --> "Long Name":::hot`,
			wantStateIDs: []string{"Long Name"},
			wantFrom:     "[*]",
			wantTo:       "Long Name",
			wantToCSS:    "hot",
		},
		{
			name: "quoted_id_with_label_colon_inside_label",
			src: `stateDiagram-v2
    state "Foo"
    state Bar
    "Foo" --> Bar : an "edge: label"`,
			wantStateIDs: []string{"Foo", "Bar"},
			wantFrom:     "Foo",
			wantTo:       "Bar",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			d, err := Parse(strings.NewReader(c.src))
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			if len(d.States) != len(c.wantStateIDs) {
				t.Errorf("state count = %d, want %d: %+v",
					len(d.States), len(c.wantStateIDs), d.States)
			}
			for _, want := range c.wantStateIDs {
				found := false
				for _, s := range d.States {
					if s.ID == want {
						found = true
					}
				}
				if !found {
					t.Errorf("state %q missing from %+v", want, d.States)
				}
			}
			if len(d.Transitions) != 1 {
				t.Fatalf("want 1 transition, got %d", len(d.Transitions))
			}
			tx := d.Transitions[0]
			if tx.From != c.wantFrom || tx.To != c.wantTo {
				t.Errorf("transition = (%q, %q), want (%q, %q)",
					tx.From, tx.To, c.wantFrom, c.wantTo)
			}
			if c.wantToCSS != "" {
				toState := stateByID(d, c.wantTo)
				if toState == nil || !containsString(toState.CSSClasses, c.wantToCSS) {
					t.Errorf("CSS class %q should be attached to %q; got %v",
						c.wantToCSS, c.wantTo, toState)
				}
			}
		})
	}
}

// Edge cases for the new quote-aware endpoint tokeniser: malformed
// inputs must fall through gracefully without creating phantoms or
// silently dropping data.
func TestParseQuotedEndpointEdgeCases(t *testing.T) {
	cases := []struct {
		name           string
		src            string
		wantTransition bool
	}{
		{
			name:           "unmatched_quote",
			src:            "stateDiagram-v2\n\"unclosed --> Foo",
			wantTransition: false,
		},
		{
			name:           "empty_quoted_id_rejected",
			src:            "stateDiagram-v2\n\"\" --> Foo",
			wantTransition: false,
		},
		{
			name:           "trailing_garbage_after_close_quote",
			src:            "stateDiagram-v2\n\"A\"B\"C\" --> Foo",
			wantTransition: false,
		},
		{
			name:           "lhs_quoted_endpoint_with_css_shorthand",
			src:            "stateDiagram-v2\nclassDef hot fill:#f00\n\"Long Name\":::hot --> Bar",
			wantTransition: true,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			d, err := Parse(strings.NewReader(c.src))
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			hasTransition := len(d.Transitions) > 0
			if hasTransition != c.wantTransition {
				t.Errorf("hasTransition = %v, want %v (transitions: %+v)",
					hasTransition, c.wantTransition, d.Transitions)
			}
			if c.name == "lhs_quoted_endpoint_with_css_shorthand" && hasTransition {
				tx := d.Transitions[0]
				if tx.From != "Long Name" {
					t.Errorf("LHS quoted endpoint stripped: From=%q, want %q", tx.From, "Long Name")
				}
				from := stateByID(d, "Long Name")
				if from == nil || !containsString(from.CSSClasses, "hot") {
					t.Errorf("CSS class `hot` should attach to %q; got %v", "Long Name", from)
				}
			}
		})
	}
}

func stateByID(d *diagram.StateDiagram, id string) *diagram.StateDef {
	for i := range d.States {
		if d.States[i].ID == id {
			return &d.States[i]
		}
	}
	return nil
}

// Forward references from inside a composite to a state declared at
// an outer scope must resolve to the outer state, not produce a
// phantom child. The transition's scope is promoted to the LCA so
// dagre lays the edge out at the right level rather than re-creating
// a phantom in the inner sub-graph.
func TestParseCrossScopeForwardReference(t *testing.T) {
	src := `stateDiagram-v2
    state Running {
        [*] --> Normal
        Normal --> DH
    }
    state DH <<deepHistory>>
    DH --> Running : restore`
	d, err := Parse(strings.NewReader(src))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	// DH must NOT appear as a child of Running — it lives at root.
	var running *diagram.StateDef
	for i := range d.States {
		if d.States[i].ID == "Running" {
			running = &d.States[i]
			break
		}
	}
	if running == nil {
		t.Fatal("Running state not found")
	}
	for _, c := range running.Children {
		if c.ID == "DH" {
			t.Errorf("DH should not be a child of Running; got phantom %+v", c)
		}
	}
	// DH at root must carry the deep-history kind from `state DH <<deepHistory>>`.
	var dh *diagram.StateDef
	for i := range d.States {
		if d.States[i].ID == "DH" {
			dh = &d.States[i]
			break
		}
	}
	if dh == nil {
		t.Fatal("DH not at root")
	}
	if dh.Kind != diagram.StateKindDeepHistory {
		t.Errorf("DH.Kind = %v, want StateKindDeepHistory", dh.Kind)
	}
	// The cross-scope edge `Normal --> DH` (written inside Running)
	// must be promoted so the edge sits at root scope, with From
	// rewritten to Running (the ancestor of Normal at root level).
	var crossEdge *diagram.StateTransition
	for i := range d.Transitions {
		t := &d.Transitions[i]
		if t.From == "Running" && t.To == "DH" {
			crossEdge = t
			break
		}
	}
	if crossEdge == nil {
		t.Errorf("cross-scope edge `Normal --> DH` should be rewritten as `Running --> DH`; got transitions: %+v", d.Transitions)
	} else if crossEdge.Scope != "" {
		t.Errorf("promoted edge should have Scope=\"\" (root), got %q", crossEdge.Scope)
	}
}

// After cross-scope promotion, the ORIGINAL inner-scope edge must
// be gone (it was rewritten in place, not appended). A regression
// that duplicated the edge instead of rewriting would slip past
// TestParseCrossScopeForwardReference because that test only checks
// the rewritten form is present.
func TestParseCrossScopePromoteIsInPlace(t *testing.T) {
	src := `stateDiagram-v2
    state Running {
        [*] --> Normal
        Normal --> DH
    }
    state DH <<deepHistory>>`
	d, err := Parse(strings.NewReader(src))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	for _, t2 := range d.Transitions {
		if t2.From == "Normal" && t2.To == "DH" {
			t.Errorf("original cross-scope edge `Normal --> DH` should have been rewritten in place, not retained: %+v", t2)
		}
	}
}

// A transition that already sits at the right scope should not be
// touched by the promotion pass.
func TestParseCrossScopeNoOpForLocalEdge(t *testing.T) {
	src := `stateDiagram-v2
    state DH <<deepHistory>>
    Running --> DH : restore
    state Running {
        [*] --> Normal
    }`
	d, err := Parse(strings.NewReader(src))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	var restore *diagram.StateTransition
	for i := range d.Transitions {
		if d.Transitions[i].Label == "restore" {
			restore = &d.Transitions[i]
		}
	}
	if restore == nil {
		t.Fatal("restore edge missing")
	}
	if restore.From != "Running" || restore.To != "DH" || restore.Scope != "" {
		t.Errorf("local root-scope edge should remain (Running, DH, scope=\"\"); got %+v", restore)
	}
}

// A transition whose endpoint isn't in the walked tree (e.g. a
// fabricated StateDiagram from a synthesizer or a future parser
// refactor that strips unreferenced states) must pass through both
// passes unchanged: no promotion, no panic on the missing path
// lookup, no spurious phantom creation. The natural parser path
// always upserts both endpoints, so this exercises the safety
// branch directly with a hand-built diagram.
func TestPromoteCrossScopeDanglingEndpoint(t *testing.T) {
	d := &diagram.StateDiagram{
		States: []diagram.StateDef{
			{ID: "A", Label: "A"},
		},
		Transitions: []diagram.StateTransition{
			{From: "A", To: "Missing"},
		},
	}
	walked := []stateWalkEntry{
		{state: &d.States[0], parent: &d.States, path: nil},
	}
	promoteCrossScopeTransitions(d, walked)
	if len(d.Transitions) != 1 {
		t.Fatalf("transitions count changed: %+v", d.Transitions)
	}
	got := d.Transitions[0]
	if got.From != "A" || got.To != "Missing" || got.Scope != "" {
		t.Errorf("dangling-endpoint edge should be unchanged; got %+v", got)
	}
}

// Ancestor → descendant edges (e.g. `Running --> Normal` where
// Normal lives inside Running) must not be collapsed into a
// self-loop on Running by promotion. The edge should layout at the
// ancestor's scope where both the composite boundary and the
// descendant are visible.
func TestParseCrossScopeAncestorToDescendant(t *testing.T) {
	src := `stateDiagram-v2
    state Running {
        state Normal
    }
    Running --> Normal : enter`
	d, err := Parse(strings.NewReader(src))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(d.Transitions) != 1 {
		t.Fatalf("want 1 transition, got %d", len(d.Transitions))
	}
	got := d.Transitions[0]
	if got.From != "Running" || got.To != "Normal" {
		t.Errorf("endpoints should be (Running, Normal), got (%q, %q)", got.From, got.To)
	}
	if got.Scope != "Running" {
		t.Errorf("Scope should be \"Running\" so the edge layouts inside Running's sub-graph; got %q", got.Scope)
	}
}

// A `state X` declaration must beat an attribute-identical phantom
// of the same ID when picking the canonical entry, regardless of
// source-order or walk-order. Both directions:
//   - phantom inside, real at root (Thread 3's repro)
//   - real inside, phantom at root (mirror case)
func TestParseDuplicateDedupPrefersExplicitDeclaration(t *testing.T) {
	t.Run("inner_phantom_outer_real", func(t *testing.T) {
		d, err := Parse(strings.NewReader(`stateDiagram-v2
    state Running {
        Normal --> DH
    }
    state DH`))
		if err != nil {
			t.Fatalf("parse: %v", err)
		}
		// DH must be at root, not inside Running.
		var dhAtRoot bool
		for _, s := range d.States {
			if s.ID == "DH" {
				dhAtRoot = true
			}
			if s.ID == "Running" {
				for _, c := range s.Children {
					if c.ID == "DH" {
						t.Errorf("DH should be at root, not Running.Children")
					}
				}
			}
		}
		if !dhAtRoot {
			t.Error("DH missing from root")
		}
	})
	t.Run("inner_real_outer_phantom", func(t *testing.T) {
		d, err := Parse(strings.NewReader(`stateDiagram-v2
    state Running {
        state Normal
    }
    Running --> Normal`))
		if err != nil {
			t.Fatalf("parse: %v", err)
		}
		// Normal must remain inside Running, not be the root phantom.
		var normalInside bool
		for _, s := range d.States {
			if s.ID == "Normal" {
				t.Errorf("Normal should not be a root state; got %+v", s)
			}
			if s.ID == "Running" {
				for _, c := range s.Children {
					if c.ID == "Normal" {
						normalInside = true
					}
				}
			}
		}
		if !normalInside {
			t.Error("Normal missing from Running.Children")
		}
	})
}

// Non-conflicting metadata (CSSClasses, explicit Label) from the
// loser of a dedup must be merged into the canonical entry — Mermaid
// users routinely tack `:::class` on transition endpoints, and the
// `state X` declaration is canonical for Kind but doesn't carry the
// transition-side CSS hint.
func TestParseDuplicateDedupMergesCSSAndLabel(t *testing.T) {
	d, err := Parse(strings.NewReader(`stateDiagram-v2
    state Running {
        Normal --> DH:::hot
    }
    state DH <<deepHistory>>`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	var dh *diagram.StateDef
	for i := range d.States {
		if d.States[i].ID == "DH" {
			dh = &d.States[i]
		}
	}
	if dh == nil {
		t.Fatal("DH missing")
	}
	if dh.Kind != diagram.StateKindDeepHistory {
		t.Errorf("DH.Kind = %v, want StateKindDeepHistory", dh.Kind)
	}
	if !containsString(dh.CSSClasses, "hot") {
		t.Errorf("CSS class %q should be merged into canonical DH; got %v", "hot", dh.CSSClasses)
	}
}

// Regions slices are independent copies of region members, so a
// phantom dropped from Children would otherwise still appear in
// Regions. The post-prune sync must filter Regions against the
// post-prune Children of the same composite.
func TestParseDuplicateDedupSyncsRegions(t *testing.T) {
	d, err := Parse(strings.NewReader(`stateDiagram-v2
    state Active {
        A --> DH
        --
        B --> DH
    }
    state DH <<deepHistory>>`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	var active *diagram.StateDef
	for i := range d.States {
		if d.States[i].ID == "Active" {
			active = &d.States[i]
		}
	}
	if active == nil {
		t.Fatal("Active missing")
	}
	for ri, region := range active.Regions {
		for _, r := range region {
			if r.ID == "DH" {
				t.Errorf("region[%d] should not contain DH (deduped to root); got %v", ri, region)
			}
		}
	}
}

// When a transition's endpoints both live in different sibling
// composites, the edge must promote up to the LCA (root) and both
// endpoints get rewritten to their respective top-level composite
// ancestors. This is the "cross-cluster wire" case.
func TestParseCrossScopeBothEndpointsDeep(t *testing.T) {
	src := `stateDiagram-v2
    state A {
        state inA
    }
    state B {
        state inB
    }
    inA --> inB`
	d, err := Parse(strings.NewReader(src))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	var cross *diagram.StateTransition
	for i := range d.Transitions {
		t := &d.Transitions[i]
		if t.From == "A" && t.To == "B" {
			cross = t
		}
	}
	if cross == nil {
		t.Fatalf("expected `A --> B` after promotion of `inA --> inB`; got transitions: %+v", d.Transitions)
	}
	if cross.Scope != "" {
		t.Errorf("Scope should be root after LCA promotion to sibling composites; got %q", cross.Scope)
	}
}

// Direct unit test for moreCanonicalState's preference ordering.
func TestMoreCanonicalState(t *testing.T) {
	plain := diagram.StateDef{ID: "X", Label: "X"}
	withKind := diagram.StateDef{ID: "X", Label: "X", Kind: diagram.StateKindChoice}
	withChildren := diagram.StateDef{ID: "X", Label: "X", Children: []diagram.StateDef{{ID: "c"}}}
	withLabel := diagram.StateDef{ID: "X", Label: "Long Label"}
	withCSS := diagram.StateDef{ID: "X", Label: "X", CSSClasses: []string{"hot"}}

	if !moreCanonicalState(withKind, plain) {
		t.Error("a state with explicit Kind should beat a plain phantom")
	}
	if !moreCanonicalState(withChildren, plain) {
		t.Error("a composite (Children) should beat a leaf")
	}
	if !moreCanonicalState(withKind, withChildren) {
		t.Error("Kind weighs more than Children")
	}
	if !moreCanonicalState(withLabel, plain) {
		t.Error("an explicit description Label should beat a default")
	}
	if !moreCanonicalState(withCSS, plain) {
		t.Error("CSSClasses should beat a plain phantom")
	}
	// Ties resolve to current (returns false).
	if moreCanonicalState(plain, plain) {
		t.Error("equal candidates should not promote")
	}
}

// commonPrefix returns the longest shared head of two paths.
func TestCommonPrefix(t *testing.T) {
	cases := []struct {
		a, b []string
		want []string
	}{
		{nil, nil, []string{}},
		{[]string{"A"}, []string{"A"}, []string{"A"}},
		{[]string{"A", "B"}, []string{"A", "C"}, []string{"A"}},
		{[]string{"A", "B", "C"}, []string{"A", "B", "C"}, []string{"A", "B", "C"}},
		{[]string{"A"}, []string{"B"}, []string{}},
		{[]string{"A", "B"}, []string{"A"}, []string{"A"}},
	}
	for _, c := range cases {
		got := commonPrefix(c.a, c.b)
		if len(got) != len(c.want) {
			t.Errorf("commonPrefix(%v, %v) = %v, want %v", c.a, c.b, got, c.want)
			continue
		}
		for i := range got {
			if got[i] != c.want[i] {
				t.Errorf("commonPrefix(%v, %v)[%d] = %q, want %q", c.a, c.b, i, got[i], c.want[i])
			}
		}
	}
}

func TestParseTransitionScope(t *testing.T) {
	input := `stateDiagram-v2
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
	d, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	want := []struct {
		from, to, scope string
	}{
		{"[*]", "First", ""},
		{"[*]", "Second", "First"},
		{"[*]", "second", "Second"},
		{"second", "Third", "Second"},
		{"[*]", "third", "Third"},
		{"third", "[*]", "Third"},
	}
	if len(d.Transitions) != len(want) {
		t.Fatalf("transitions count = %d, want %d: %+v", len(d.Transitions), len(want), d.Transitions)
	}
	for i, w := range want {
		got := d.Transitions[i]
		if got.From != w.from || got.To != w.to || got.Scope != w.scope {
			t.Errorf("transition[%d] = %+v, want From=%q To=%q Scope=%q", i, got, w.from, w.to, w.scope)
		}
	}
}

func TestParseCompositeWithAliasedChild(t *testing.T) {
	input := `stateDiagram-v2
    state Active {
        state "Long Name" as L
        L --> L2
    }`
	d, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	active := &d.States[0]
	var lChild *diagram.StateDef
	for i := range active.Children {
		if active.Children[i].ID == "L" {
			lChild = &active.Children[i]
		}
	}
	if lChild == nil {
		t.Fatal("L not found in Active.Children")
	}
	if lChild.Label != "Long Name" {
		t.Errorf("L.Label = %q, want %q", lChild.Label, "Long Name")
	}
}

func TestParseCompositeWithStateDecl(t *testing.T) {
	input := `stateDiagram-v2
    state Active {
        state Running
        Running --> Paused
    }`
	d, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	var active *diagram.StateDef
	for i := range d.States {
		if d.States[i].ID == "Active" {
			active = &d.States[i]
		}
	}
	if active == nil {
		t.Fatal("Active not found")
	}
	foundRunning := false
	for _, c := range active.Children {
		if c.ID == "Running" {
			foundRunning = true
		}
	}
	if !foundRunning {
		t.Error("Running should be a child of Active")
	}
}

func TestParseInvalidTransition(t *testing.T) {
	input := "stateDiagram-v2\n    --> B"
	d, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(d.Transitions) != 0 {
		t.Errorf("empty-from transition should be ignored, got %+v", d.Transitions)
	}
}

func TestParseStartToEnd(t *testing.T) {
	input := "stateDiagram-v2\n    [*] --> [*]"
	d, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(d.Transitions) != 1 {
		t.Fatalf("want 1 transition, got %d", len(d.Transitions))
	}
	if d.Transitions[0].From != "[*]" || d.Transitions[0].To != "[*]" {
		t.Errorf("got %+v", d.Transitions[0])
	}
	if len(d.States) != 0 {
		t.Errorf("[*] should not create real states, got %+v", d.States)
	}
}

func TestParseUnknownStateKind(t *testing.T) {
	input := "stateDiagram-v2\n    state foo <<unknown>>"
	d, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if d.States[0].Kind != diagram.StateKindNormal {
		t.Errorf("unknown kind should default to normal, got %v", d.States[0].Kind)
	}
}

func TestParseBareStateDecl(t *testing.T) {
	input := `stateDiagram-v2
    state Idle`
	d, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(d.States) != 1 || d.States[0].ID != "Idle" {
		t.Errorf("got %+v", d.States)
	}
}

// `id : label` outside of a transition assigns the label to that state.
// The state is auto-registered if not already present.
func TestParseStateLabel(t *testing.T) {
	d, err := Parse(strings.NewReader(`stateDiagram-v2
    s1 : This is a label
    s2 : Another one`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(d.States) != 2 {
		t.Fatalf("want 2 states, got %d", len(d.States))
	}
	if d.States[0].ID != "s1" || d.States[0].Label != "This is a label" {
		t.Errorf("s1 = %+v", d.States[0])
	}
	if d.States[1].Label != "Another one" {
		t.Errorf("s2 = %+v", d.States[1])
	}
}

// Label shorthand can repeat; the latest label wins.
func TestParseStateLabelUpdate(t *testing.T) {
	d, err := Parse(strings.NewReader(`stateDiagram-v2
    s1 : first
    s1 : second`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if d.States[0].Label != "second" {
		t.Errorf("label = %q, want %q", d.States[0].Label, "second")
	}
}

func TestParseDirection(t *testing.T) {
	cases := []struct {
		src  string
		want diagram.Direction
	}{
		{"direction TB", diagram.DirectionTB},
		{"direction TD", diagram.DirectionTB}, // alias
		{"direction BT", diagram.DirectionBT},
		{"direction LR", diagram.DirectionLR},
		{"direction RL", diagram.DirectionRL},
	}
	for _, tc := range cases {
		t.Run(tc.src, func(t *testing.T) {
			d, err := Parse(strings.NewReader("stateDiagram-v2\n    " + tc.src + "\n    [*] --> S"))
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			if d.Direction != tc.want {
				t.Errorf("direction = %v, want %v", d.Direction, tc.want)
			}
		})
	}
}

func TestParseDirectionInvalid(t *testing.T) {
	_, err := Parse(strings.NewReader("stateDiagram-v2\n    direction WAT"))
	if err == nil {
		t.Error("expected error for unknown direction")
	}
}

// Transition labels with literal `\n` get the real newline so
// renderers can split on it directly.
func TestParseMultiLineTransitionLabel(t *testing.T) {
	d, err := Parse(strings.NewReader(`stateDiagram-v2
    A --> B : line1\nline2`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if d.Transitions[0].Label != "line1\nline2" {
		t.Errorf("label = %q, want with embedded newline", d.Transitions[0].Label)
	}
}

// Single-line note: `note left of X : text` and `note right of X : text`.
func TestParseSingleLineNote(t *testing.T) {
	d, err := Parse(strings.NewReader(`stateDiagram-v2
    state Foo
    state Bar
    note left of Foo : Foo is the start
    note right of Bar : Bar terminates`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(d.Notes) != 2 {
		t.Fatalf("want 2 notes, got %d", len(d.Notes))
	}
	if d.Notes[0].Side != diagram.NoteSideLeft || d.Notes[0].Target != "Foo" || d.Notes[0].Text != "Foo is the start" {
		t.Errorf("note[0] = %+v", d.Notes[0])
	}
	if d.Notes[1].Side != diagram.NoteSideRight || d.Notes[1].Target != "Bar" || d.Notes[1].Text != "Bar terminates" {
		t.Errorf("note[1] = %+v", d.Notes[1])
	}
}

// Multi-line block note: opens with `note left of X` (no colon),
// each subsequent line is body text, ends with `end note`.
// Resulting Text contains real newlines.
func TestParseBlockNote(t *testing.T) {
	d, err := Parse(strings.NewReader(`stateDiagram-v2
    state Foo
    note left of Foo
        first line
        second line
    end note`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(d.Notes) != 1 {
		t.Fatalf("want 1 note, got %d", len(d.Notes))
	}
	n := d.Notes[0]
	if n.Side != diagram.NoteSideLeft || n.Target != "Foo" {
		t.Errorf("side/target = %v/%q", n.Side, n.Target)
	}
	if n.Text != "first line\nsecond line" {
		t.Errorf("text = %q, want with embedded newline", n.Text)
	}
}

// `\n` inside a single-line note text becomes a real newline.
func TestParseNoteLineBreaks(t *testing.T) {
	d, err := Parse(strings.NewReader(`stateDiagram-v2
    state X
    note right of X : line1\nline2`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if d.Notes[0].Text != "line1\nline2" {
		t.Errorf("text = %q", d.Notes[0].Text)
	}
}

// Note for an undeclared state auto-registers it.
func TestParseNoteAutoRegistersTarget(t *testing.T) {
	d, err := Parse(strings.NewReader(`stateDiagram-v2
    note left of Ghost : phantom`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(d.States) != 1 || d.States[0].ID != "Ghost" {
		t.Errorf("states = %+v", d.States)
	}
}

// Unclosed block note errors rather than silently consuming the
// rest of the file.
func TestParseUnclosedBlockNoteError(t *testing.T) {
	_, err := Parse(strings.NewReader(`stateDiagram-v2
    state Foo
    note left of Foo
        unfinished`))
	if err == nil {
		t.Error("expected error for unclosed note block")
	}
}

func TestParseTitleAndAccessibility(t *testing.T) {
	d, err := Parse(strings.NewReader(`stateDiagram-v2
    title: My state machine
    accTitle: State machine for X
    accDescr: Describes the lifecycle
    [*] --> A`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if d.Title != "My state machine" {
		t.Errorf("Title = %q", d.Title)
	}
	if d.AccTitle != "State machine for X" {
		t.Errorf("AccTitle = %q", d.AccTitle)
	}
	if d.AccDescr != "Describes the lifecycle" {
		t.Errorf("AccDescr = %q", d.AccDescr)
	}
}

func TestParseClassDefAndStyle(t *testing.T) {
	d, err := Parse(strings.NewReader(`stateDiagram-v2
    classDef important fill:#f96,stroke:#333
    state Foo
    style Foo fill:#abc`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if got := d.CSSClasses["important"]; got != "fill:#f96;stroke:#333" {
		t.Errorf("classDef = %q", got)
	}
	if len(d.Styles) != 1 || d.Styles[0].StateID != "Foo" || d.Styles[0].CSS != "fill:#abc" {
		t.Errorf("Styles = %+v", d.Styles)
	}
}

func TestParseCSSClassBinding(t *testing.T) {
	d, err := Parse(strings.NewReader(`stateDiagram-v2
    classDef hot fill:#f00
    state A
    state B
    class A,B hot`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	for _, id := range []string{"A", "B"} {
		var found *diagram.StateDef
		for i := range d.States {
			if d.States[i].ID == id {
				found = &d.States[i]
			}
		}
		if found == nil {
			t.Fatalf("%s not found", id)
		}
		if len(found.CSSClasses) != 1 || found.CSSClasses[0] != "hot" {
			t.Errorf("%s.CSSClasses = %v", id, found.CSSClasses)
		}
	}
}

// `:::className` shorthand on a state declaration line.
func TestParseCSSClassShorthand(t *testing.T) {
	d, err := Parse(strings.NewReader(`stateDiagram-v2
    classDef hot fill:#f00
    state Foo:::hot`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(d.States) != 1 {
		t.Fatalf("want 1 state, got %d", len(d.States))
	}
	if d.States[0].ID != "Foo" {
		t.Errorf("ID = %q (shorthand should not leak)", d.States[0].ID)
	}
	if got := d.States[0].CSSClasses; len(got) != 1 || got[0] != "hot" {
		t.Errorf("CSSClasses = %v", got)
	}
}

func TestParseClickHref(t *testing.T) {
	d, err := Parse(strings.NewReader(`stateDiagram-v2
    state Foo
    click Foo href "https://example.com" "Open"`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(d.Clicks) != 1 {
		t.Fatalf("want 1 click, got %d", len(d.Clicks))
	}
	c := d.Clicks[0]
	if c.StateID != "Foo" || c.URL != "https://example.com" || c.Tooltip != "Open" {
		t.Errorf("click = %+v", c)
	}
}

func TestParseLinkAndCallback(t *testing.T) {
	d, err := Parse(strings.NewReader(`stateDiagram-v2
    state Foo
    state Bar
    link Foo "https://example.com" "tip"
    callback Bar "openDetails"`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(d.Clicks) != 2 {
		t.Fatalf("want 2 clicks, got %d", len(d.Clicks))
	}
	if d.Clicks[0].URL != "https://example.com" || d.Clicks[1].Callback != "openDetails" {
		t.Errorf("clicks = %+v", d.Clicks)
	}
}

// Empty Callback / URL after `call`/`href` errors instead of
// silently storing empty values.
func TestParseClickEmptyArgumentsError(t *testing.T) {
	for _, src := range []string{
		`stateDiagram-v2
    state Foo
    click Foo call`,
		`stateDiagram-v2
    state Foo
    click Foo href`,
		`stateDiagram-v2
    state Foo
    callback Foo`,
		`stateDiagram-v2
    state Foo
    link Foo`,
	} {
		t.Run(strings.SplitN(src, "\n", 3)[2], func(t *testing.T) {
			_, err := Parse(strings.NewReader(src))
			if err == nil {
				t.Errorf("expected error for empty click arguments")
			}
		})
	}
}

// Unterminated `"` in click args surfaces as an error rather than
// silently capturing the rest of the line.
func TestParseClickUnterminatedQuoteError(t *testing.T) {
	for _, src := range []string{
		`stateDiagram-v2
    state Foo
    click Foo href "https://example.com`,
		`stateDiagram-v2
    state Foo
    link Foo "open"open`,
	} {
		t.Run("", func(t *testing.T) {
			_, err := Parse(strings.NewReader(src))
			_ = err // both are malformed; tolerate either error or no-op as long as not panic
		})
	}
	// Direct: an actually unterminated quote.
	_, err := Parse(strings.NewReader(`stateDiagram-v2
    state Foo
    click Foo href "open`))
	if err == nil {
		t.Error("expected error for unterminated quote")
	}
}

// classDef with only a name (no CSS) is malformed.
func TestParseClassDefMissingCSSError(t *testing.T) {
	_, err := Parse(strings.NewReader(`stateDiagram-v2
    classDef onlyname`))
	if err == nil {
		t.Error("expected error for classDef without CSS body")
	}
}

// `style ID` without CSS body is malformed.
func TestParseStyleMissingCSSError(t *testing.T) {
	_, err := Parse(strings.NewReader(`stateDiagram-v2
    state Foo
    style Foo`))
	if err == nil {
		t.Error("expected error for style without CSS body")
	}
}

// `class IDs` without a class name is malformed.
func TestParseClassBindingMissingNameError(t *testing.T) {
	_, err := Parse(strings.NewReader(`stateDiagram-v2
    state Foo
    class Foo`))
	if err == nil {
		t.Error("expected error for class binding without class name")
	}
}

// `---\ntitle: ...\n---` frontmatter is stripped before parsing,
// with `title:` populating diagram.Title — Mermaid's universal
// frontmatter convention.
func TestParseFrontmatterTitle(t *testing.T) {
	d, err := Parse(strings.NewReader(`---
title: Simple sample
---
stateDiagram-v2
    [*] --> Still
    Still --> [*]`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if d.Title != "Simple sample" {
		t.Errorf("Title = %q, want %q", d.Title, "Simple sample")
	}
	if len(d.Transitions) != 2 {
		t.Errorf("Transitions count = %d, want 2", len(d.Transitions))
	}
}

// `class id className` referencing an undeclared state is silently
// skipped (Mermaid's behaviour — the syntax-docs styling example
// binds `class end badBadEvent` where `end` is never declared).
// Parsing succeeds and the surrounding diagram is unaffected.
func TestParseClassBindingUndefinedStateSkipped(t *testing.T) {
	d, err := Parse(strings.NewReader(`stateDiagram-v2
    classDef hot fill:#f00
    Real --> Other
    class Ghost hot`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	for _, s := range d.States {
		if s.ID == "Ghost" {
			t.Errorf("undeclared state %q was created", s.ID)
		}
	}
}

// `:::` shorthand combined with `<<fork>>` should set BOTH the CSS
// class and the special-state Kind.
func TestParseCSSShorthandWithSpecialKind(t *testing.T) {
	d, err := Parse(strings.NewReader(`stateDiagram-v2
    classDef hot fill:#f00
    state foo:::hot <<fork>>`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(d.States) != 1 {
		t.Fatalf("want 1 state, got %d", len(d.States))
	}
	s := d.States[0]
	if s.Kind != diagram.StateKindFork {
		t.Errorf("Kind = %v, want fork", s.Kind)
	}
	if len(s.CSSClasses) != 1 || s.CSSClasses[0] != "hot" {
		t.Errorf("CSSClasses = %v", s.CSSClasses)
	}
}

// Click on undeclared state errors instead of silently registering.
func TestParseClickUndeclaredStateError(t *testing.T) {
	_, err := Parse(strings.NewReader(`stateDiagram-v2
    click Ghost href "https://example.com"`))
	if err == nil {
		t.Error("expected error for click on undeclared state")
	}
}

// `--` inside a composite state body splits the body into parallel
// regions. Each region's states land in their own slice.
func TestParseConcurrentRegions(t *testing.T) {
	d, err := Parse(strings.NewReader(`stateDiagram-v2
    state Active {
        Running --> Paused
        Paused --> Running
        --
        Healthy --> Sick
        Sick --> Healthy
    }`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(d.States) != 1 {
		t.Fatalf("want 1 top-level state, got %d", len(d.States))
	}
	active := d.States[0]
	if active.ID != "Active" {
		t.Fatalf("ID = %q", active.ID)
	}
	if len(active.Regions) != 2 {
		t.Fatalf("want 2 regions, got %d", len(active.Regions))
	}
	// First region: Running, Paused.
	r1 := active.Regions[0]
	if len(r1) != 2 || r1[0].ID != "Running" || r1[1].ID != "Paused" {
		t.Errorf("region[0] = %+v", r1)
	}
	// Second region: Healthy, Sick.
	r2 := active.Regions[1]
	if len(r2) != 2 || r2[0].ID != "Healthy" || r2[1].ID != "Sick" {
		t.Errorf("region[1] = %+v", r2)
	}
	// Children is the concatenated union across regions.
	if len(active.Children) != 4 {
		t.Errorf("Children union = %d, want 4", len(active.Children))
	}
}

// A composite without `--` populates only Children, not Regions.
func TestParseSingleRegionCompositeUnchanged(t *testing.T) {
	d, err := Parse(strings.NewReader(`stateDiagram-v2
    state Active {
        Running --> Paused
    }`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	active := &d.States[0]
	if len(active.Regions) != 0 {
		t.Errorf("Regions should be empty for single-region composite, got %d", len(active.Regions))
	}
	if len(active.Children) != 2 {
		t.Errorf("Children = %d, want 2", len(active.Children))
	}
}

// Three regions separated by two `--` dividers.
func TestParseThreeRegions(t *testing.T) {
	d, err := Parse(strings.NewReader(`stateDiagram-v2
    state P {
        state A
        --
        state B
        --
        state C
    }`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	p := d.States[0]
	if len(p.Regions) != 3 {
		t.Fatalf("want 3 regions, got %d", len(p.Regions))
	}
	for i, want := range []string{"A", "B", "C"} {
		if len(p.Regions[i]) != 1 || p.Regions[i][0].ID != want {
			t.Errorf("region[%d] = %+v, want id=%s", i, p.Regions[i], want)
		}
	}
}

func TestParseAutoRegistersStates(t *testing.T) {
	input := `stateDiagram-v2
    A --> B`
	d, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(d.States) < 2 {
		t.Errorf("should auto-register A and B, got %d states", len(d.States))
	}
}

// `state "Label"` without `as` creates a state whose ID and label
// are both the quoted string.
func TestParseQuotedStateWithoutAs(t *testing.T) {
	d, err := Parse(strings.NewReader(`stateDiagram-v2
    state "Long Name"`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(d.States) != 1 {
		t.Fatalf("want 1 state, got %d", len(d.States))
	}
	if d.States[0].ID != "Long Name" || d.States[0].Label != "Long Name" {
		t.Errorf("state = %+v", d.States[0])
	}
}

// `state "Label" { ... }` without `as` creates a composite whose ID
// and label are both the quoted string.
func TestParseQuotedStateWithoutAsComposite(t *testing.T) {
	d, err := Parse(strings.NewReader(`stateDiagram-v2
    state "Active State" {
        Running --> Paused
    }`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(d.States) != 1 {
		t.Fatalf("want 1 top-level state, got %d", len(d.States))
	}
	s := d.States[0]
	if s.ID != "Active State" || s.Label != "Active State" {
		t.Errorf("ID/Label = %q/%q", s.ID, s.Label)
	}
	if len(s.Children) != 2 {
		t.Errorf("want 2 children, got %d", len(s.Children))
	}
}

// Unterminated quote in state declaration errors.
func TestParseUnterminatedQuoteStateDecl(t *testing.T) {
	_, err := Parse(strings.NewReader(`stateDiagram-v2
    state "No closing`))
	if err == nil {
		t.Fatal("expected error for unterminated quote")
	}
}

// Empty state identifier errors instead of creating a phantom state.
func TestParseEmptyStateIDError(t *testing.T) {
	cases := []struct {
		name string
		src  string
	}{
		{"state_brace", "stateDiagram-v2\n    state {"},
		{"state_kind", "stateDiagram-v2\n    state <<fork>>"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Parse(strings.NewReader(tc.src))
			if err == nil {
				t.Errorf("expected error for empty state id")
			}
		})
	}
}

// `state "Label" as ID { ... }` creates a composite with the given
// label and identifier.
func TestParseAliasedComposite(t *testing.T) {
	d, err := Parse(strings.NewReader(`stateDiagram-v2
    state "Active State" as Active {
        Running --> Paused
    }`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(d.States) != 1 {
		t.Fatalf("want 1 top-level state, got %d", len(d.States))
	}
	s := d.States[0]
	if s.ID != "Active" || s.Label != "Active State" {
		t.Errorf("ID/Label = %q/%q", s.ID, s.Label)
	}
	if len(s.Children) != 2 {
		t.Errorf("want 2 children, got %d", len(s.Children))
	}
}

// `state "Label" as ID <<fork>>` sets both the label and the kind.
func TestParseAliasedStateWithKind(t *testing.T) {
	d, err := Parse(strings.NewReader(`stateDiagram-v2
    state "Fork Point" as F <<fork>>`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(d.States) != 1 {
		t.Fatalf("want 1 state, got %d", len(d.States))
	}
	s := d.States[0]
	if s.ID != "F" || s.Label != "Fork Point" {
		t.Errorf("ID/Label = %q/%q", s.ID, s.Label)
	}
	if s.Kind != diagram.StateKindFork {
		t.Errorf("Kind = %v, want fork", s.Kind)
	}
}

// `state "Label" as ID:::css` attaches the CSS class.
func TestParseAliasedStateWithCSSShorthand(t *testing.T) {
	d, err := Parse(strings.NewReader(`stateDiagram-v2
    classDef hot fill:#f00
    state "Hot State" as H:::hot`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(d.States) != 1 {
		t.Fatalf("want 1 state, got %d", len(d.States))
	}
	s := d.States[0]
	if s.ID != "H" {
		t.Errorf("ID = %q", s.ID)
	}
	if len(s.CSSClasses) != 1 || s.CSSClasses[0] != "hot" {
		t.Errorf("CSSClasses = %v", s.CSSClasses)
	}
}

// History and deep-history states are parsed correctly.
func TestParseHistoryStates(t *testing.T) {
	for _, tc := range []struct {
		decl string
		want diagram.StateKind
	}{
		{"state H <<history>>", diagram.StateKindHistory},
		{"state DH <<deepHistory>>", diagram.StateKindDeepHistory},
		{"state DH <<deep_history>>", diagram.StateKindDeepHistory},
	} {
		t.Run(tc.want.String(), func(t *testing.T) {
			d, err := Parse(strings.NewReader("stateDiagram-v2\n    " + tc.decl))
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			if len(d.States) == 0 {
				t.Fatal("no states")
			}
			if d.States[0].Kind != tc.want {
				t.Errorf("kind = %v, want %v", d.States[0].Kind, tc.want)
			}
		})
	}
}

// Transition to/from a composite state preserves the label through
// the leaf-representative redirect.
func TestParseCompositeTransitionLabel(t *testing.T) {
	d, err := Parse(strings.NewReader(`stateDiagram-v2
    state Active {
        Running --> Paused
    }
    [*] --> Active : init
    Active --> Done : finish`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(d.Transitions) != 3 {
		t.Fatalf("want 3 transitions, got %d", len(d.Transitions))
	}
	// Verify labels are preserved in the AST.
	found := make(map[string]string)
	for _, tr := range d.Transitions {
		found[tr.From+"->"+tr.To] = tr.Label
	}
	if got := found["[*]->Active"]; got != "init" {
		t.Errorf("[*]->Active label = %q, want init", got)
	}
	if got := found["Active->Done"]; got != "finish" {
		t.Errorf("Active->Done label = %q, want finish", got)
	}
}

// A bare state identifier on its own line registers the state.
func TestParseBareStateID(t *testing.T) {
	d, err := Parse(strings.NewReader(`stateDiagram-v2
    stateId`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(d.States) != 1 {
		t.Fatalf("want 1 state, got %d", len(d.States))
	}
	if d.States[0].ID != "stateId" || d.States[0].Label != "stateId" {
		t.Errorf("state = %+v", d.States[0])
	}
}

// CSS shorthand on transition endpoints is stripped before state
// registration so the state ID isn't polluted.
func TestParseTransitionCSSShorthand(t *testing.T) {
	d, err := Parse(strings.NewReader(`stateDiagram-v2
    classDef hot fill:#f00
    A:::hot --> B:::cold
    B --> C`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	ids := make(map[string]bool)
	for _, s := range d.States {
		ids[s.ID] = true
	}
	for _, id := range []string{"A", "B", "C"} {
		if !ids[id] {
			t.Errorf("state %q not found", id)
		}
	}
	if ids["A:::hot"] || ids["B:::cold"] {
		t.Error("CSS shorthand leaked into state IDs")
	}
	if len(d.Transitions) != 2 {
		t.Fatalf("want 2 transitions, got %d", len(d.Transitions))
	}
	if d.Transitions[0].From != "A" || d.Transitions[0].To != "B" {
		t.Errorf("transition[0] = %+v", d.Transitions[0])
	}
}

// `id : text` on a composite state sets its label (display name).
func TestParseCompositeLabelViaColon(t *testing.T) {
	d, err := Parse(strings.NewReader(`stateDiagram-v2
    NamedComposite : Another Composite
    state NamedComposite {
        [*] --> namedSimple
    }`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(d.States) != 1 {
		t.Fatalf("want 1 state, got %d", len(d.States))
	}
	s := d.States[0]
	if s.ID != "NamedComposite" {
		t.Errorf("ID = %q", s.ID)
	}
	if s.Label != "Another Composite" {
		t.Errorf("Label = %q, want %q", s.Label, "Another Composite")
	}
}
