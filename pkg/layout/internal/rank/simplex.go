package rank

import (
	"github.com/julianshen/mmgo/pkg/layout/graph"
)

func optimize(g *graph.Graph, ranks map[string]int) {
	for {
		tightTree := buildTightTree(g, ranks)
		if len(tightTree) == 0 {
			break
		}

		type candidate struct {
			eid   graph.EdgeID
			slack int
		}
		var candidates []candidate
		for _, eid := range g.Edges() {
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
			locked := make(map[string]bool)
			var lock func(node string)
			lock = func(node string) {
				if locked[node] {
					return
				}
				locked[node] = true
				for _, eid := range g.OutEdges(node) {
					if tightTree[eid] {
						lock(eid.To)
					}
				}
				for _, eid := range g.InEdges(node) {
					if tightTree[eid] {
						lock(eid.From)
					}
				}
			}
			lock(c.eid.From)

			visited := make(map[string]bool)
			var shift func(node string)
			shift = func(node string) {
				if visited[node] || locked[node] {
					return
				}
				visited[node] = true
				ranks[node] -= c.slack
				for _, eid := range g.OutEdges(node) {
					if !locked[eid.To] {
						shift(eid.To)
					}
				}
				for _, eid := range g.InEdges(node) {
					if !locked[eid.From] {
						shift(eid.From)
					}
				}
			}
			shift(c.eid.To)

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

func buildTightTree(g *graph.Graph, ranks map[string]int) map[graph.EdgeID]bool {
	tight := make(map[graph.EdgeID]bool)
	for _, eid := range g.Edges() {
		attrs, _ := g.EdgeAttrs(eid)
		if ranks[eid.To]-ranks[eid.From] == attrs.EffectiveMinLen() {
			tight[eid] = true
		}
	}
	return tight
}
