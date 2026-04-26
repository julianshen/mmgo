# Flowchart Visual Parity Fixes

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Close the remaining visual gaps between mmgo and mmdc for flowchart rendering — node sizing, edge curves, and rank optimization.

**Architecture:** Three independent fixes: (1) tune node padding/line-height constants to match mmdc, (2) replace straight `<line>` edges with cubic bezier `<path>` edges that include padding segments, (3) add a network simplex rank optimizer to tighten cross-subgraph edge lengths. Branch mirroring is noted as won't-fix (cosmetic, inherent to crossing minimization).

**Tech Stack:** Go, existing layout engine (dagre port), SVG generation via `encoding/xml`

---

## Chunk 1: Node Sizing

### Task 1: Adjust node padding and line height constants

**Files:**
- Modify: `pkg/output/svg/svg.go:86-91` (constants)
- Modify: `pkg/output/svg/testdata/*.golden.svg` (golden files)

- [ ] **Step 1: Update constants**

In `pkg/output/svg/svg.go`, change the constants block:

```go
const (
	nodePaddingX     = 60.0
	nodePaddingY     = 30.0
	minNodeWidth     = 60.0
	minNodeHeight    = 40.0
	lineHeightFactor = 1.3
)
```

- [ ] **Step 2: Build and regenerate all flowchart examples**

```bash
go build -o /tmp/mmgo ./cmd/mmgo
for f in examples/flowchart/*.mmd; do
  base="${f%.mmd}"
  /tmp/mmgo -i "$f" -o "${base}.svg" -q
  /tmp/mmgo -i "$f" -o "${base}.png" -q
done
```

- [ ] **Step 3: Update golden files**

```bash
go test ./pkg/output/svg/ -run TestGoldenFiles -update
```

- [ ] **Step 4: Run all tests**

```bash
go test ./... -race
```

Expected: all tests pass.

- [ ] **Step 5: Commit**

```bash
git add -A
git commit -m "fix: increase node padding and line height to match mmdc node sizing"
```

---

## Chunk 2: Edge Curves

### Task 2: Replace straight-line edges with padded cubic bezier paths

In mmdc, every edge uses a `<path>` with short straight "padding" segments at the start/end near the nodes, and a cubic bezier curve in between. mmgo currently uses `<line>` for 2-point edges.

**Files:**
- Modify: `pkg/renderer/flowchart/edges.go:227-236` (the `len(pts) == 2` branch)

- [ ] **Step 1: Write failing test**

In `pkg/renderer/flowchart/edges_test.go`, add a test that renders a 2-point edge and verifies it produces a `<path>`, not a `<line>`:

```go
func TestStraightEdgeRendersAsPath(t *testing.T) {
	d := &diagram.FlowchartDiagram{
		Nodes: []diagram.Node{
			{ID: "A", Label: "From", Shape: diagram.NodeShapeRectangle},
			{ID: "B", Label: "To", Shape: diagram.NodeShapeRectangle},
		},
		Edges: []diagram.Edge{
			{From: "A", To: "B", ArrowHead: diagram.ArrowHeadArrow},
		},
	}
	g := graph.New()
	g.SetNode("A", graph.NodeAttrs{Width: 80, Height: 40})
	g.SetNode("B", graph.NodeAttrs{Width: 80, Height: 40})
	g.SetEdge("A", "B", graph.EdgeAttrs{})
	l := layout.Layout(g, layout.Options{})
	out, err := Render(d, l, nil)
	if err != nil {
		t.Fatal(err)
	}
	raw := string(out)
	if strings.Contains(raw, "<line ") {
		t.Error("2-point edge should render as <path>, not <line>")
	}
	if !strings.Contains(raw, "<path ") {
		t.Error("expected <path> element in output")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./pkg/renderer/flowchart/ -run TestStraightEdgeRendersAsPath -v
```

Expected: FAIL ("2-point edge should render as <path>, not <line>")

- [ ] **Step 3: Implement — replace the `len(pts) == 2` branch**

In `pkg/renderer/flowchart/edges.go`, replace the `case len(pts) == 2:` block (lines 227-236) with:

```go
	case len(pts) == 2:
		d := paddedEdgePath(pts[0], pts[1])
		p := &Path{D: d, Style: style}
		if isVisibleArrow(e.ArrowHead) {
			p.MarkerEnd = fmt.Sprintf("url(#%s)", markerID(e.ArrowHead, e.LineStyle))
		}
		elems = append(elems, p)
		elems = append(elems, startMarkerElems(e.ArrowTail, pts[0], pts[1], th)...)
```

Add the `paddedEdgePath` helper function near the bottom of `edges.go`:

```go
func paddedEdgePath(src, dst layout.Point) string {
	const pad = 5.0
	dx := dst.X - src.X
	dy := dst.Y - src.Y
	length := math.Hypot(dx, dy)
	if length <= 2*pad {
		return fmt.Sprintf("M%.2f,%.2f L%.2f,%.2f", src.X, src.Y, dst.X, dst.Y)
	}
	ux, uy := dx/length, dy/length
	ps := layout.Point{X: src.X + ux*pad, Y: src.Y + uy*pad}
	pe := layout.Point{X: dst.X - ux*pad, Y: dst.Y - uy*pad}
	return fmt.Sprintf("M%.2f,%.2f L%.2f,%.2f C%.2f,%.2f %.2f,%.2f %.2f,%.2f L%.2f,%.2f",
		src.X, src.Y, ps.X, ps.Y,
		ps.X, ps.Y, pe.X, pe.Y, pe.X, pe.Y,
		dst.X, dst.Y)
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
go test ./pkg/renderer/flowchart/ -run TestStraightEdgeRendersAsPath -v
```

Expected: PASS

- [ ] **Step 5: Run all tests**

```bash
go test ./... -race
```

Expected: all pass (golden files may need updating — if so, regenerate examples and update golden files as in Task 1 Step 2-3)

- [ ] **Step 6: Commit**

```bash
git add -A
git commit -m "fix: render 2-point edges as padded bezier paths instead of straight lines"
```

---

## Chunk 3: Network Simplex Rank Optimization

### Task 3: Implement network simplex rank optimizer

The current rank assignment uses longest-path only, which places leaf nodes at the maximum rank. Network simplex tightens ranks to minimize total edge length, fixing the ci_cd Notify node placement.

This is the most complex task. The algorithm is from Gansner et al. 1993, §4.

**Files:**
- Create: `pkg/layout/internal/rank/simplex.go` (network simplex optimizer)
- Modify: `pkg/layout/internal/rank/rank.go` (call simplex after longest-path)
- Test: `pkg/layout/internal/rank/rank_test.go`

- [ ] **Step 1: Write failing test for rank tightness**

In `pkg/layout/internal/rank/rank_test.go`, add:

```go
func TestNetworkSimplexTightensRanks(t *testing.T) {
	// Diamond with two sinks at different depths:
	//   A -> B -> C -> D -> E
	//                  \\-> F
	// Without simplex, F gets rank 4 (same as E).
	// With simplex, F gets rank 3 (tight: rank(C)+1).
	g := graph.New()
	g.SetNode("A", graph.NodeAttrs{})
	g.SetNode("B", graph.NodeAttrs{})
	g.SetNode("C", graph.NodeAttrs{})
	g.SetNode("D", graph.NodeAttrs{})
	g.SetNode("E", graph.NodeAttrs{})
	g.SetNode("F", graph.NodeAttrs{})
	g.SetEdge("A", "B", graph.EdgeAttrs{})
	g.SetEdge("B", "C", graph.EdgeAttrs{})
	g.SetEdge("C", "D", graph.EdgeAttrs{})
	g.SetEdge("D", "E", graph.EdgeAttrs{})
	g.SetEdge("C", "F", graph.EdgeAttrs{})
	ranks := rank.Run(g)
	if ranks["F"] != ranks["C"]+1 {
		t.Errorf("rank(F) = %d, want %d (rank(C)+1); got ranks: %v",
			ranks["F"], ranks["C"]+1, ranks)
	}
	if ranks["E"] != ranks["D"]+1 {
		t.Errorf("rank(E) = %d, want %d (rank(D)+1); got ranks: %v",
			ranks["E"], ranks["D"]+1, ranks)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./pkg/layout/internal/rank/ -run TestNetworkSimplexTightensRanks -v
```

Expected: FAIL (F gets rank 4 instead of 3)

- [ ] **Step 3: Implement network simplex**

Create `pkg/layout/internal/rank/simplex.go` with the network simplex optimizer. The algorithm:

1. Build a spanning tree (tight tree) from the current feasible ranking
2. For each non-tree edge, compute its slack (amount by which the constraint is non-tight)
3. If any edge has slack > 0, perform a tree rotation: find the cut, swap an edge, update ranks
4. Repeat until no non-tree edge has positive slack (optimal)

The implementation should follow dagre.js's approach in `lib/rank/util.js` (`tightenRank`). Key data structures:
- `cutValues`: for each tree edge, the sum of weights of non-tree edges crossing the cut
- `slack`: for each non-tree edge, `rank(to) - rank(from) - minLen`

Here is the skeleton:

```go
package rank

import (
	"math"
	"github.com/julianshen/mmgo/pkg/layout/graph"
)

func optimize(g *graph.Graph, ranks map[string]int) {
	// Build initial tight tree
	tightEdges := findTightEdges(g, ranks)
	if len(tightEdges) == 0 {
		return
	}
	
	for {
		// Find a non-tree edge with minimum slack
		e, slack := minSlackEdge(g, ranks, tightEdges)
		if e == "" || slack == math.MaxInt {
			break
		}
		// Tighten: shift ranks on one side of the cut
		tighten(g, ranks, e, slack, tightEdges)
		tightEdges[e] = true
	}
}

func findTightEdges(g *graph.Graph, ranks map[string]int) map[string]bool {
	tight := make(map[string]bool)
	for _, eid := range g.Edges() {
		from, to := parseEdgeID(eid)
		if ranks[to]-ranks[from] == g.EdgeAttrs(eid).EffectiveMinLen() {
			tight[eid] = true
		}
	}
	return tight
}

func minSlackEdge(g *graph.Graph, ranks map[string]int, tight map[string]bool) (string, int) {
	minSlack := math.MaxInt
	var minEdge string
	for _, eid := range g.Edges() {
		if tight[eid] {
			continue
		}
		from, to := parseEdgeID(eid)
		slack := ranks[to] - ranks[from] - g.EdgeAttrs(eid).EffectiveMinLen()
		if slack < minSlack {
			minSlack = slack
			minEdge = eid
		}
	}
	return minEdge, minSlack
}

func tighten(g *graph.Graph, ranks map[string]int, edgeID string, slack int, tight map[string]bool) {
	from, to := parseEdgeID(edgeID)
	if slack > 0 {
		// Shift all nodes on the 'to' side down by slack
		// ... DFS from 'to' without crossing tight tree edges
	}
}
```

Note: This is a simplified version. The full network simplex from the paper uses cut values and tree rotations. A pragmatic approach that matches dagre.js's behavior is to iterate: find the non-tight edge with smallest slack, and shift one partition. This converges quickly for flowchart-sized graphs.

- [ ] **Step 4: Call `optimize` from `rank.Run()`**

In `rank.go`, after `normalize(s.ranks)`, call the optimizer:

```go
func Run(g *graph.Graph) map[string]int {
	// ... existing longest-path code ...
	normalize(s.ranks)
	optimize(g, s.ranks)
	return s.ranks
}
```

- [ ] **Step 5: Run test to verify it passes**

```bash
go test ./pkg/layout/internal/rank/ -run TestNetworkSimplexTightensRanks -v
```

Expected: PASS

- [ ] **Step 6: Run all tests**

```bash
go test ./... -race
```

Expected: all pass (some golden files may need updating due to changed layout positions)

- [ ] **Step 7: Regenerate examples and update golden files**

```bash
go build -o /tmp/mmgo ./cmd/mmgo
for f in examples/flowchart/*.mmd; do
  base="${f%.mmd}"
  /tmp/mmgo -i "$f" -o "${base}.svg" -q
  /tmp/mmgo -i "$f" -o "${base}.png" -q
done
go test ./pkg/output/svg/ -run TestGoldenFiles -update
```

- [ ] **Step 8: Commit**

```bash
git add -A
git commit -m "feat: add network simplex rank optimizer for tighter edge lengths"
```

---

## Chunk 4: Final Verification

### Task 4: Re-audit all 14 flowchart examples

- [ ] **Step 1: Regenerate all examples**

```bash
go build -o /tmp/mmgo ./cmd/mmgo
for f in examples/flowchart/*.mmd; do
  base="${f%.mmd}"
  /tmp/mmgo -i "$f" -o "${base}.svg" -q
  /tmp/mmgo -i "$f" -o "${base}.png" -q
done
```

- [ ] **Step 2: Full test suite with coverage**

```bash
go test ./... -race -coverprofile=coverage.out
go tool cover -func=coverage.out | grep total
```

Expected: all pass, coverage >= 90%

- [ ] **Step 3: Lint**

```bash
golangci-lint run ./...
```

Expected: clean

- [ ] **Step 4: Final commit if any example regeneration needed**

```bash
git add -A
git commit -m "chore: regenerate examples after visual parity fixes"
```

---

## Won't Fix: Branch Mirroring

The "flowchart_practical" example shows "yes"/"no" branches swapped left↔right vs mmdc. This is caused by crossing minimization producing a different valid ordering. dagre.js itself does not guarantee left/right branch order — it depends on initial node order and tie-breaking heuristics. Matching mmdc exactly would require replicating dagre.js's specific tie-breaking logic, which is fragile and may change between mmdc versions. This is cosmetic, not a correctness issue.
