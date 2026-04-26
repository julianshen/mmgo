package rank

import (
	"github.com/julianshen/mmgo/pkg/layout/graph"
)

func optimize(g *graph.Graph, ranks map[string]int) {
	allEdges := g.Edges()
	for {
		tightTree := buildTightTree(allEdges, g, ranks)
		if len(tightTree) == 0 {
			break
		}

		type candidate struct {
			eid   graph.EdgeID
			slack int
		}
		var candidates []candidate
		for _, eid := range allEdges {
			if tightTree[eid] {
				continue
			}
			attrs, _ := g.EdgeAttrs(eid)
			slack := ranks[eid.To] - ranks[eid.From] - attrs.EffectiveMinLen()
			if slack > 0 {
				candidates = append(candidates, candidate{eid, slack})
			}
		}
		if len(candidates) == 0 {
			break
		}

		improved := false
		for _, c := range candidates {
			locked := reachLocked(g, c.eid.From, tightTree)

			visited := make(map[string]bool)
			shiftUnlocked(g, c.eid.To, c.slack, locked, visited, ranks)

			if len(visited) > 0 {
				improved = true
				break
			}
		}
		if !improved {
			break
		}
	}
	normalize(ranks)
}

func reachLocked(g *graph.Graph, start string, tightTree map[graph.EdgeID]bool) map[string]bool {
	locked := make(map[string]bool)
	var visit func(string)
	visit = func(node string) {
		if locked[node] {
			return
		}
		locked[node] = true
		for _, eid := range g.OutEdges(node) {
			if tightTree[eid] {
				visit(eid.To)
			}
		}
		for _, eid := range g.InEdges(node) {
			if tightTree[eid] {
				visit(eid.From)
			}
		}
	}
	visit(start)
	return locked
}

func shiftUnlocked(g *graph.Graph, start string, slack int, locked, visited map[string]bool, ranks map[string]int) {
	var visit func(string)
	visit = func(node string) {
		if visited[node] || locked[node] {
			return
		}
		visited[node] = true
		ranks[node] -= slack
		for _, eid := range g.OutEdges(node) {
			if !locked[eid.To] {
				visit(eid.To)
			}
		}
		for _, eid := range g.InEdges(node) {
			if !locked[eid.From] {
				visit(eid.From)
			}
		}
	}
	visit(start)
}

func buildTightTree(allEdges []graph.EdgeID, g *graph.Graph, ranks map[string]int) map[graph.EdgeID]bool {
	tight := make(map[graph.EdgeID]bool, len(allEdges))
	for _, eid := range allEdges {
		attrs, _ := g.EdgeAttrs(eid)
		if ranks[eid.To]-ranks[eid.From] == attrs.EffectiveMinLen() {
			tight[eid] = true
		}
	}
	return tight
}
