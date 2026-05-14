package state

import (
	"github.com/julianshen/mmgo/pkg/diagram"
	"github.com/julianshen/mmgo/pkg/layout"
	"github.com/julianshen/mmgo/pkg/layout/graph"
	"github.com/julianshen/mmgo/pkg/renderer/svgutil"
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
	Nodes       map[string]layout.NodeLayout
	NodeAttrs   map[string]graph.NodeAttrs
	Edges       map[graph.EdgeID]layout.EdgeLayout
	Composites  []flatComposite
	PseudoOwner map[string]string
	Width       float64
	Height      float64
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
		NodeAttrs:   make(map[string]graph.NodeAttrs),
		Edges:       make(map[graph.EdgeID]layout.EdgeLayout),
		PseudoOwner: make(map[string]string),
	}
	if root == nil {
		return out
	}
	pad := defaultPadding
	walkScope(root, pad, pad, 0, out)
	// Bounding box: leaf/pseudo nodes contribute via their center +
	// half-extent. Composites also appear in Nodes (so edge clipping
	// can target them), but the same rect is already in Composites
	// with the top-left convention — skip them here to avoid counting
	// each composite twice.
	compositeIDs := make(map[string]struct{}, len(out.Composites))
	for _, c := range out.Composites {
		compositeIDs[c.ID] = struct{}{}
	}
	for id, n := range out.Nodes {
		if _, isComposite := compositeIDs[id]; isComposite {
			continue
		}
		if right := n.X + n.Width/2; right > out.Width {
			out.Width = right
		}
		if bottom := n.Y + n.Height/2; bottom > out.Height {
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
	// Right/bottom inset to mirror the left/top padding.
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
			// Expose the composite as a flat node too so edge clipping
			// can target it as a rectangle.
			out.Nodes[id] = layout.NodeLayout{
				X: nodeLeft + n.Width/2, Y: nodeTop + n.Height/2,
				Width: n.Width, Height: n.Height,
			}
			out.NodeAttrs[id] = graph.NodeAttrs{Label: fc.Label, Width: n.Width, Height: n.Height}
			// Recurse into the child's interior. The child's dagre
			// origin (0,0) maps to the interior top-left + padding.
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
		if attrs, ok := s.nodeAttrs[id]; ok {
			out.NodeAttrs[id] = attrs
		}
		if _, ok := s.pseudoNodes[id]; ok {
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

// buildPlacedComposites turns the flatten pass's composite list into
// the renderer's placedComposite representation, attaching the original
// StateDef so the rect emits its proper label/CSS metadata.
func buildPlacedComposites(states []diagram.StateDef, flats []flatComposite) []placedComposite {
	defByID := svgutil.IndexByID(collectAllStates(states), func(s diagram.StateDef) string { return s.ID })
	out := make([]placedComposite, 0, len(flats))
	for _, fc := range flats {
		def := defByID[fc.ID]
		// Ensure the rendered label falls back to the ID when empty.
		if def.Label == "" {
			def.Label = fc.Label
			if def.Label == "" {
				def.Label = fc.ID
			}
		}
		out = append(out, placedComposite{
			def: def,
			x:   fc.X, y: fc.Y,
			w: fc.Width, h: fc.Height,
			depth: fc.Depth,
		})
	}
	return out
}

// labelOf returns the display label for a node in this scope, falling
// back to the ID when no explicit label is recorded in nodeAttrs.
func (s *scopedLayout) labelOf(id string) string {
	if s != nil {
		if attrs, ok := s.nodeAttrs[id]; ok && attrs.Label != "" {
			return attrs.Label
		}
	}
	return id
}
