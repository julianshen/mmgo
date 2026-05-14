package state

import (
	"github.com/julianshen/mmgo/pkg/layout"
	"github.com/julianshen/mmgo/pkg/layout/graph"
)

// flatScopedLayout collapses a recursive scopedLayout into a single
// global-coordinate view that existing render primitives can consume:
//
//   - Nodes  – every leaf state and every pseudo-state from every
//     scope, keyed by its ID, with X/Y in the diagram's global frame.
//   - Composites – outer rect for each composite state (global frame),
//     including the title-bar band at top; the composite's interior
//     sub-layout is already represented by its child entries above.
//   - PseudoOwner – maps each pseudo-state ID to the ID of the composite
//     scope that owns it ("" for the root scope). Renderers can use this
//     to attach a `[*]` glyph to the right cluster when emitting SVG.
//   - Edges – every transition's polyline, with control points already
//     translated into the global frame. Keyed by graph.EdgeID using the
//     same From/To strings the scope-level dagre graph used (i.e. with
//     scope-qualified pseudo IDs for `[*]` endpoints).
//
// Width/Height are the bounding box of all flattened content.
type flatScopedLayout struct {
	Nodes        map[string]layout.NodeLayout
	Edges        map[graph.EdgeID]layout.EdgeLayout
	Composites   []flatComposite
	PseudoOwner  map[string]string
	Width        float64
	Height       float64
}

// flatComposite is a composite state's outer rect in global coords.
// X,Y is the top-left of the rect (NOT the center). Width and Height
// include the title-bar band that sits above the inner content area
// (TitleBarH). InteriorOrigin gives the top-left of the inner content
// area (i.e. X, Y+TitleBarH), kept for clarity in the renderer.
type flatComposite struct {
	ID             string
	Label          string
	X, Y           float64
	Width, Height  float64
	TitleBarH      float64
	InteriorOrigin struct{ X, Y float64 }
	Depth          int
}

// flattenScopedLayout walks the recursive layout tree and produces the
// global-frame view described above. The root scope is placed at
// (pad, pad) in the global frame, where pad is defaultPadding.
func flattenScopedLayout(root *scopedLayout) *flatScopedLayout {
	out := &flatScopedLayout{
		Nodes:       make(map[string]layout.NodeLayout),
		Edges:       make(map[graph.EdgeID]layout.EdgeLayout),
		PseudoOwner: make(map[string]string),
	}
	if root == nil {
		return out
	}
	pad := defaultPadding
	walkScope(root, pad, pad, 0, out)
	// Bounding box: take the max extent across all placed shapes.
	for _, n := range out.Nodes {
		right := n.X + n.Width/2
		bottom := n.Y + n.Height/2
		if right > out.Width {
			out.Width = right
		}
		if bottom > out.Height {
			out.Height = bottom
		}
	}
	for _, c := range out.Composites {
		if r := c.X + c.Width; r > out.Width {
			out.Width = r
		}
		if b := c.Y + c.Height; b > out.Height {
			out.Height = b
		}
	}
	// Add padding on the right/bottom edges to match the left/top inset.
	out.Width += pad
	out.Height += pad
	return out
}

// walkScope copies one scope's dagre result into the global view,
// translated by (originX, originY). originX/Y is the top-left of the
// scope's inner content area in the global frame.
func walkScope(s *scopedLayout, originX, originY float64, depth int, out *flatScopedLayout) {
	if s == nil || s.result == nil {
		return
	}
	// Each scope's dagre result places nodes with (0,0) at the
	// top-left of its own bbox. To translate into global coords we add
	// originX/originY directly to each node's center.
	for id, n := range s.result.Nodes {
		// Composite child node — emit a flatComposite for the outer
		// rect AND recurse so the child's interior lands inside it.
		if child, isComposite := s.children[id]; isComposite {
			// n.X, n.Y is the center of the composite node in the
			// parent scope. Convert to top-left.
			nodeLeft := originX + n.X - n.Width/2
			nodeTop := originY + n.Y - n.Height/2
			fc := flatComposite{
				ID:        id,
				Label:     s.labelOf(id),
				X:         nodeLeft,
				Y:         nodeTop,
				Width:     n.Width,
				Height:    n.Height,
				TitleBarH: compositeLabelH,
				Depth:     depth,
			}
			fc.InteriorOrigin.X = nodeLeft
			fc.InteriorOrigin.Y = nodeTop + compositeLabelH
			out.Composites = append(out.Composites, fc)
			// Recurse into the child's interior.
			walkScope(child, fc.InteriorOrigin.X+defaultPadding, fc.InteriorOrigin.Y+defaultPadding, depth+1, out)
			continue
		}
		// Plain node (leaf or pseudo) — translate and store.
		out.Nodes[id] = layout.NodeLayout{
			X:         originX + n.X,
			Y:         originY + n.Y,
			Width:     n.Width,
			Height:    n.Height,
			ExitPorts: shiftPoints(n.ExitPorts, originX, originY),
		}
		if info, ok := s.pseudoNodes[id]; ok {
			_ = info
			out.PseudoOwner[id] = s.scopeID
		}
	}
	// Edges: translate each control point.
	for id, e := range s.result.Edges {
		out.Edges[id] = layout.EdgeLayout{
			Points:   shiftPoints(e.Points, originX, originY),
			LabelPos: layout.Point{X: e.LabelPos.X + originX, Y: e.LabelPos.Y + originY},
			BackEdge: e.BackEdge,
		}
	}
}

func shiftPoints(pts []layout.Point, dx, dy float64) []layout.Point {
	if len(pts) == 0 {
		return nil
	}
	out := make([]layout.Point, len(pts))
	for i, p := range pts {
		out[i] = layout.Point{X: p.X + dx, Y: p.Y + dy}
	}
	return out
}

// labelOf returns the human label for a state in this scope. We don't
// have direct access to StateDef here, but the dagre graph stored it
// in NodeAttrs. Fallback to the ID when the label is empty.
func (s *scopedLayout) labelOf(id string) string {
	// scopedLayout.result.Nodes only holds geometry — the label lives
	// in the original graph.NodeAttrs which is not retained on Result.
	// For composites we can look it up by re-iterating the children
	// map: child layout's owner ID is the composite's state ID, but
	// its display label may differ. Step 2c will thread StateDef
	// references through scopedLayout; for now return the ID, which
	// matches Mermaid's default label-equals-id behavior.
	return id
}
