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
// the NodeIDs field collects every node transitively reachable from
// that edge up to (but excluding) the next branch node or convergence
// point. ColorIndex is assigned round-robin across the palette.
type BranchGroup struct {
	SourceNodeID string
	EdgeIndex    int
	NodeIDs      []string
	ColorIndex   int
	EdgeFromTo   [][2]string
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

	// First pass: for every branch origin+edge, compute the initial
	// reach (stopping at other branch nodes). Second pass: any node
	// that appears in 2+ reaches is a convergence point and gets
	// scrubbed from every set.
	type origin struct{ src, firstHop string }
	reach := make(map[origin]map[string]bool)
	for _, src := range sources {
		for _, target := range branchNodes[src] {
			visited := map[string]bool{src: true}
			stack := []string{target}
			for len(stack) > 0 {
				cur := stack[len(stack)-1]
				stack = stack[:len(stack)-1]
				if visited[cur] {
					continue
				}
				visited[cur] = true
				if _, isBranch := branchNodes[cur]; isBranch && cur != target {
					// Stop at downstream branch nodes so they can
					// start their own groups; exclude them from this
					// group's membership.
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

			// Collect edges fully inside this branch: both endpoints
			// in {source, members}.
			inGroup := map[string]bool{src: true}
			for _, id := range memberIDs {
				inGroup[id] = true
			}
			var fromTo [][2]string
			for _, e := range d.AllEdges() {
				if inGroup[e.From] && inGroup[e.To] {
					fromTo = append(fromTo, [2]string{e.From, e.To})
				}
			}

			groups = append(groups, BranchGroup{
				SourceNodeID: src,
				EdgeIndex:    i,
				NodeIDs:      memberIDs,
				ColorIndex:   colorIdx % len(branchColorPalette),
				EdgeFromTo:   fromTo,
			})
			colorIdx++
		}
	}
	return groups
}

// renderBranchRegions emits one shaded rounded-rect per BranchGroup,
// sized to enclose the group's member node bounding boxes with a 20px
// padding. Regions render before edges and nodes so they sit in the
// back layer.
func renderBranchRegions(groups []BranchGroup, l *layout.Result, pad float64) []any {
	if len(groups) == 0 || l == nil {
		return nil
	}
	const regionPad = 20.0
	out := make([]any, 0, len(groups))
	for _, g := range groups {
		var minX, minY, maxX, maxY float64
		minX, minY = 1e18, 1e18
		maxX, maxY = -1e18, -1e18
		any := false
		for _, id := range g.NodeIDs {
			nl, ok := l.Nodes[id]
			if !ok {
				continue
			}
			any = true
			if nl.X-nl.Width/2 < minX {
				minX = nl.X - nl.Width/2
			}
			if nl.Y-nl.Height/2 < minY {
				minY = nl.Y - nl.Height/2
			}
			if nl.X+nl.Width/2 > maxX {
				maxX = nl.X + nl.Width/2
			}
			if nl.Y+nl.Height/2 > maxY {
				maxY = nl.Y + nl.Height/2
			}
		}
		if !any {
			continue
		}
		palette := branchColorPalette[g.ColorIndex%len(branchColorPalette)]
		out = append(out, &Rect{
			X:      svgFloat(minX - regionPad + pad),
			Y:      svgFloat(minY - regionPad + pad),
			Width:  svgFloat(maxX - minX + 2*regionPad),
			Height: svgFloat(maxY - minY + 2*regionPad),
			RX:     6, RY: 6,
			Style: fmt.Sprintf("fill:%s;fill-opacity:0.35;stroke:%s;stroke-dasharray:4,3;stroke-width:1",
				palette.Fill, palette.Stroke),
		})
	}
	return out
}
