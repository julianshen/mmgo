package flowchart

import (
	"fmt"
	"sort"

	"github.com/julianshen/mmgo/pkg/diagram"
	"github.com/julianshen/mmgo/pkg/layout"
)

// branchColorPalette is a 6-entry cycle of (fill, stroke) pastel pairs
// used to tint the regions behind each branch of a multi-outlet node.
// The tints are light enough to sit behind nodes/edges without obscuring
// them; the stroke variant is ~20% darker for the dashed region border.
var branchColorPalette = []struct {
	Fill   string
	Stroke string
}{
	{"#E3F2FD", "#90CAF9"}, // light blue
	{"#E8F5E9", "#A5D6A7"}, // light green
	{"#FFF9C4", "#FFF176"}, // light yellow
	{"#FCE4EC", "#F48FB1"}, // light pink
	{"#F3E5F5", "#CE93D8"}, // light purple
	{"#FFF3E0", "#FFB74D"}, // light orange
}

// BranchGroup represents one branch of a multi-outlet node. Each
// outgoing edge of a branch node (3+ outgoing) starts its own group;
// NodeIDs collects every node transitively reachable from that edge
// up to (but excluding) the next branch node or convergence point.
type BranchGroup struct {
	SourceNodeID string
	EdgeIndex    int
	NodeIDs      []string
	ColorIndex   int
}

// DetectBranches walks the flowchart and returns one BranchGroup per
// outgoing edge of every multi-outlet node (3+ outgoing). Nodes that
// are reachable from two or more branches are treated as convergence
// points — they stop the walk from upstream and are excluded from
// every BranchGroup's NodeIDs. Groups whose members all sit inside the
// same user-defined subgraph are filtered out so the subgraph's own
// styling provides the visual grouping instead.
func DetectBranches(d *diagram.FlowchartDiagram, l *layout.Result) []BranchGroup {
	if d == nil {
		return nil
	}

	adj := make(map[string][]string)
	for _, e := range d.AllEdges() {
		adj[e.From] = append(adj[e.From], e.To)
	}
	// branchNodes: ID → slice of direct children in AST edge order. We
	// keep stable edge order so EdgeIndex matches the per-source edge
	// ordering the renderer sees.
	branchNodes := make(map[string][]string)
	for src, targets := range adj {
		if len(targets) >= 3 {
			branchNodes[src] = targets
		}
	}
	if len(branchNodes) == 0 {
		return nil
	}

	// Sort branch sources for deterministic output.
	sources := make([]string, 0, len(branchNodes))
	for s := range branchNodes {
		sources = append(sources, s)
	}
	sort.Strings(sources)

	// A node reachable from 2+ branches is a convergence point and
	// belongs to no single group — so walk per (src, firstHop) first,
	// then subtract any node that appears in multiple reach sets.
	type origin struct{ src, firstHop string }
	reach := make(map[origin]map[string]bool)
	for _, src := range sources {
		for _, target := range branchNodes[src] {
			visited := map[string]bool{src: true}
			// Downstream branch nodes start their own groups and are
			// excluded from this upstream group — that includes the
			// target itself if it's a branch node.
			if _, isBranch := branchNodes[target]; isBranch {
				reach[origin{src, target}] = map[string]bool{}
				continue
			}
			stack := []string{target}
			for len(stack) > 0 {
				cur := stack[len(stack)-1]
				stack = stack[:len(stack)-1]
				if visited[cur] {
					continue
				}
				visited[cur] = true
				if _, isBranch := branchNodes[cur]; isBranch {
					delete(visited, cur)
					continue
				}
				for _, nxt := range adj[cur] {
					if !visited[nxt] {
						stack = append(stack, nxt)
					}
				}
			}
			delete(visited, src)
			reach[origin{src, target}] = visited
		}
	}

	convergence := map[string]bool{}
	count := map[string]int{}
	for _, set := range reach {
		for id := range set {
			count[id]++
		}
	}
	for id, c := range count {
		if c >= 2 {
			convergence[id] = true
		}
	}

	// Subgraph membership lookup: per-node, the set of subgraphs it
	// directly belongs to. Used to suppress a branch group when all
	// its members share the same subgraph.
	sgOf := map[string]string{}
	var walkSG func(sg *diagram.Subgraph)
	walkSG = func(sg *diagram.Subgraph) {
		for _, n := range sg.Nodes {
			sgOf[n.ID] = sg.ID
		}
		for _, c := range sg.Children {
			walkSG(c)
		}
	}
	for _, sg := range d.Subgraphs {
		walkSG(sg)
	}

	// Build BranchGroups in (source, edge-index) order.
	var groups []BranchGroup
	colorIdx := 0
	for _, src := range sources {
		for i, target := range branchNodes[src] {
			members := reach[origin{src, target}]
			for c := range convergence {
				delete(members, c)
			}
			if len(members) == 0 {
				colorIdx++
				continue
			}

			// Check if every member (plus the source) sits in one
			// user-defined subgraph — if so, suppress the group.
			sameSG := true
			var sgID string
			if s, ok := sgOf[src]; ok {
				sgID = s
			} else {
				sameSG = false
			}
			if sameSG {
				for id := range members {
					if sgOf[id] != sgID {
						sameSG = false
						break
					}
				}
			}
			if sameSG && sgID != "" {
				colorIdx++
				continue
			}

			memberIDs := make([]string, 0, len(members))
			for id := range members {
				memberIDs = append(memberIDs, id)
			}
			sort.Strings(memberIDs)

			groups = append(groups, BranchGroup{
				SourceNodeID: src,
				EdgeIndex:    i,
				NodeIDs:      memberIDs,
				ColorIndex:   colorIdx % len(branchColorPalette),
			})
			colorIdx++
		}
	}
	return groups
}

// renderBranchRegions emits one shaded rounded-rect per BranchGroup,
// sized to enclose the group's member nodes with a regionInset margin.
// Rendered before edges/nodes so regions sit in the back layer.
func renderBranchRegions(groups []BranchGroup, l *layout.Result, pad float64) []any {
	if len(groups) == 0 || l == nil {
		return nil
	}
	const regionInset = 20.0
	const regionCornerR = 6.0
	const regionStyle = "fill:%s;fill-opacity:0.35;stroke:%s;stroke-dasharray:4,3;stroke-width:1"
	out := make([]any, 0, len(groups))
	for _, g := range groups {
		nodes := make([]diagram.Node, len(g.NodeIDs))
		for i, id := range g.NodeIDs {
			nodes[i] = diagram.Node{ID: id}
		}
		bb, ok := subgraphBBox(nodes, l.Nodes)
		if !ok {
			continue
		}
		palette := branchColorPalette[g.ColorIndex%len(branchColorPalette)]
		out = append(out, &Rect{
			X:      svgFloat(bb.MinX - regionInset + pad),
			Y:      svgFloat(bb.MinY - regionInset + pad),
			Width:  svgFloat(bb.MaxX - bb.MinX + 2*regionInset),
			Height: svgFloat(bb.MaxY - bb.MinY + 2*regionInset),
			RX:     regionCornerR, RY: regionCornerR,
			Style: fmt.Sprintf(regionStyle, palette.Fill, palette.Stroke),
		})
	}
	return out
}
