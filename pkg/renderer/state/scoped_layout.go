package state

import (
	"fmt"

	"github.com/julianshen/mmgo/pkg/diagram"
	"github.com/julianshen/mmgo/pkg/layout"
	"github.com/julianshen/mmgo/pkg/layout/graph"
	"github.com/julianshen/mmgo/pkg/renderer/svgutil"
	"github.com/julianshen/mmgo/pkg/textmeasure"
)

// scopedLayout is the layout for one scope — the root diagram or a
// single composite state. Coordinates in `result` are local to this
// scope's origin (0,0 = top-left of the scope's inner content area,
// before adding the composite's title bar / outer padding).
//
// The hierarchy mirrors the state tree: each composite child has its
// own `scopedLayout` reachable through `children`. Rendering does a
// recursive walk that translates a child's coordinates by the parent
// node's top-left + the title bar height.
type scopedLayout struct {
	// scopeID is "" for the root scope, otherwise the composite's ID.
	scopeID string
	// result is dagre's layout of this scope's direct children.
	// Composite children appear as single nodes sized by their own
	// recursive sub-layout (width × (height + title bar)).
	result *layout.Result
	// children holds each direct child composite's sub-layout, keyed
	// by composite state ID.
	children map[string]*scopedLayout
	// pseudoNodes records the synthetic ID assigned to each `[*]`
	// endpoint in this scope, with metadata for the renderer.
	pseudoNodes map[string]pseudoNodeInfo
	// nodeAttrs preserves the graph.NodeAttrs supplied for each node
	// in this scope, keyed by ID. Used by the renderer to look up
	// labels and shapes after layout.
	nodeAttrs map[string]graph.NodeAttrs
	// labels maps the state ID to its display label for leaves and
	// composites in this scope (composites carry their state label
	// here so the flatten pass can surface it without re-walking
	// the diagram tree).
	labels map[string]string
	// width, height are this scope's content size (the dagre bbox
	// plus padding). When this scope is a composite child of another
	// scope, the parent uses these dimensions for its node attrs.
	width, height float64
}

// pseudoNodeInfo describes a synthetic pseudo-state node injected for
// a `[*]` endpoint. `Kind` is "start" when the transition's From was
// `[*]` and "end" when its To was `[*]`. `TransitionIndex` indexes
// back into the diagram's flat transitions slice so the renderer can
// resolve the edge's label and clip target.
type pseudoNodeInfo struct {
	Kind            string
	TransitionIndex int
}

// pseudoStartID and pseudoEndID produce the synthetic node IDs used
// for `[*]` endpoints within a scope. The names preserve the legacy
// `__start_…__` / `__end_…__` prefix so the existing isStartNode /
// isEndNode helpers (which prefix-match) continue to recognise them,
// while embedding the scope to keep nested `[*]` nodes distinct.
func pseudoStartID(scope string, idx int) string {
	if scope == "" {
		return fmt.Sprintf("__start_root_%d__", idx)
	}
	return fmt.Sprintf("__start_%s_%d__", scope, idx)
}

func pseudoEndID(scope string, idx int) string {
	if scope == "" {
		return fmt.Sprintf("__end_root_%d__", idx)
	}
	return fmt.Sprintf("__end_%s_%d__", scope, idx)
}

// layoutScope recursively lays out a scope. It walks composite children
// bottom-up so the parent's dagre graph can size each composite node
// from the child's already-computed sub-layout.
//
// `scope` is "" for the root and the composite state ID otherwise.
// `statesInScope` are the direct children of this scope (NOT all
// descendants). `allTransitions` is the diagram's flat transitions
// slice; this function filters by `t.Scope == scope` itself.
func layoutScope(
	scope string,
	statesInScope []diagram.StateDef,
	allTransitions []diagram.StateTransition,
	ruler *textmeasure.Ruler,
	fontSize float64,
	opts layout.Options,
) *scopedLayout {
	out := &scopedLayout{
		scopeID:     scope,
		children:    make(map[string]*scopedLayout),
		pseudoNodes: make(map[string]pseudoNodeInfo),
		nodeAttrs:   make(map[string]graph.NodeAttrs),
		labels:      make(map[string]string),
	}

	// 1. Recurse into composites first so their sizes are known.
	for _, s := range statesInScope {
		if len(s.Children) == 0 {
			continue
		}
		out.children[s.ID] = layoutScope(s.ID, s.Children, allTransitions, ruler, fontSize, opts)
	}

	// 2. Build a dagre graph for this scope.
	g := graph.New()
	for _, s := range statesInScope {
		if child, ok := out.children[s.ID]; ok {
			labelW, _ := ruler.Measure(s.Label, fontSize-1)
			minLabel := labelW + 2*compositePadX
			w := child.width
			if minLabel > w {
				w = minLabel
			}
			h := child.height + compositeLabelH
			attrs := graph.NodeAttrs{Label: s.Label, Width: w, Height: h}
			g.SetNode(s.ID, attrs)
			out.nodeAttrs[s.ID] = attrs
			out.labels[s.ID] = s.Label
			continue
		}
		w, h := stateNodeSize(s, ruler, fontSize)
		attrs := graph.NodeAttrs{Label: s.Label, Width: w, Height: h}
		switch s.Kind {
		case diagram.StateKindChoice:
			attrs.Shape = graph.ShapeDiamond
		case diagram.StateKindHistory, diagram.StateKindDeepHistory:
			attrs.Shape = graph.ShapeCircle
		}
		g.SetNode(s.ID, attrs)
		out.nodeAttrs[s.ID] = attrs
		out.labels[s.ID] = s.Label
	}

	// 3. Pseudo-state nodes + transition edges for this scope.
	startSeq, endSeq := 0, 0
	for i, t := range allTransitions {
		if t.Scope != scope {
			continue
		}
		from, to := t.From, t.To
		if from == "[*]" {
			startSeq++
			id := pseudoStartID(scope, startSeq)
			from = id
			out.pseudoNodes[id] = pseudoNodeInfo{Kind: "start", TransitionIndex: i}
			attrs := graph.NodeAttrs{Width: pseudoNodeR * 2, Height: pseudoNodeR * 2, Shape: graph.ShapeCircle}
			g.SetNode(id, attrs)
			out.nodeAttrs[id] = attrs
		}
		if to == "[*]" {
			endSeq++
			id := pseudoEndID(scope, endSeq)
			to = id
			out.pseudoNodes[id] = pseudoNodeInfo{Kind: "end", TransitionIndex: i}
			attrs := graph.NodeAttrs{Width: pseudoNodeR * 2, Height: pseudoNodeR * 2, Shape: graph.ShapeCircle}
			g.SetNode(id, attrs)
			out.nodeAttrs[id] = attrs
		}
		// `from`/`to` may still reference a composite ID — that's fine:
		// composites are real nodes in this scope's graph, sized from
		// their sub-layout. Dagre will route to the composite's edge.
		g.SetEdge(from, to, graph.EdgeAttrs{Label: t.Label})
	}

	// 4. Run dagre.
	out.result = layout.Layout(g, opts)
	pad := defaultPadding
	out.width = sanitize(out.result.Width) + 2*pad
	out.height = sanitize(out.result.Height) + 2*pad
	return out
}

// compositeOf returns the sub-layout for a composite state if any.
// nil result means `id` is a leaf in this scope (or not present).
func (s *scopedLayout) compositeOf(id string) *scopedLayout {
	if s == nil {
		return nil
	}
	return s.children[id]
}

// Ensure svgutil is referenced; some callers may want bboxes from here
// later. Keeping the import live avoids tooling noise during step 2a.
var _ = svgutil.BBox{}
