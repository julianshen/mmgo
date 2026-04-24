# Phase B: Diamond Branching

## Problem

When a flowchart node (especially a diamond/decision) has 3+ outgoing edges, all edges exit from the geometric intersection point on the node boundary along the center-to-target ray. This produces overlapping edges near the node, labels far from the decision point, and no visual indication of which paths belong to which branch. Real-world decision flowcharts become unreadable.

## Scope

- **Port assignment:** Assign distinct exit points on multi-outlet nodes so branch edges fan out cleanly
- **Smart edge routing:** Edges route through assigned ports with short stem segments, producing a classic decision-tree look
- **Improved label placement:** Branch labels ("Yes", "No", condition text) positioned near the exit port on the first segment, not at edge midpoints
- **Automatic branch grouping:** Shaded background regions + edge color coding for each branch path, auto-detected from graph structure
- **All shapes:** Applies to any node shape with 3+ outgoing edges, with shape-aware port positioning (diamond vertices, rect edges, hexagon vertices, etc.)

Out of scope: self-loop arcs, full orthogonal edge routing, config-based opt-out toggle, back-edge visual differentiation.

## Approach

Layout-engine port assignment. After the Sugiyama positioning phase completes, a new pass assigns exit/entry ports to multi-outlet nodes. Edges are routed through these ports. The renderer uses branch structure for grouping and color coding. No full orthogonal router needed.

## Branch Detection & Port Assignment

### Branch Node Definition

A node with **3+ outgoing edges** is a "branch node." Nodes with 1-2 outgoing edges are unchanged — existing center-to-boundary clipping produces acceptable results for simple forks.

### Port Assignment Algorithm

Runs after the position phase, as a new pass in `Layout()`:

1. Identify all branch nodes (3+ outgoing edges)
2. For each branch node, collect outgoing edges sorted by target X position (left to right)
3. Compute exit port positions distributed around the node boundary:
   - **Diamond:** Use the 3 available exit vertices (left, bottom, right) for 3 edges. For >3 edges, interpolate between these vertices
   - **Other shapes:** Distribute ports evenly along the exit side of the bounding box (bottom edge for TB/BT, right edge for LR/RL)
4. Compute entry ports similarly for nodes with 3+ incoming edges (convergence nodes)

### NodeLayout Extension

```go
type NodeLayout struct {
    X, Y, Width, Height float64
    ExitPorts  []Point  // one per outgoing edge, in edge order
    EntryPorts []Point  // one per incoming edge, in edge order
}
```

When `ExitPorts` is empty (the common case — nodes with ≤2 outgoing edges), the existing center-to-boundary clipping behavior is unchanged. Backward-compatible.

### Edge Geometry Update

`buildEdges()` assigns each outgoing edge to its corresponding port. The polyline starts at the port point instead of the node center, then continues through any dummy waypoints to the target.

## Smart Edge Routing

### Segment Construction

For each edge from a branch node, the polyline becomes:

1. **Exit port** — the assigned port on the source node boundary
2. **Bend point** — a point one `RankSep/2` below (TB) / above (BT) / right (LR) / left (RL) of the exit port, creating a short stem from the node
3. **Existing dummy waypoints** — threaded through as today, starting from the bend point
4. **Entry clip** — clipped to target boundary as today

The short stem segments fan out from distinct ports, then converge into parallel tracks through the dummy chain. This produces the classic decision-tree look.

### 2-Edge Case

Unchanged. Straight line from center, clipped to boundary. No ports, no bend points.

### Edge Label Placement

Currently labels sit at the polyline midpoint, far from the decision node for short edges. For branch edges, place the label on the first segment:

- Position the label at **40% along the first segment** from the exit port
- Offset **perpendicular** to the segment direction by `(textHeight/2 + 4px)`
- Offset direction: **away from the node center** (outward)

For non-branch edges (≤2 outgoing), label placement is unchanged (polyline midpoint).

### Spline vs. Straight

Branch edges with 3+ waypoints still use Catmull-Rom splines (tension 0.3) as today. The port + bend geometry provides better starting points; the smoothing algorithm is unchanged.

## Automatic Branch Grouping

### Detection Algorithm

After layout, for each branch node (3+ outgoing edges):

1. Build a forward adjacency map from the AST edges
2. Walk the graph from each outgoing edge's target, collecting all reachable nodes
3. Stop walking at other branch nodes (they start their own groups) or convergence points
4. A **convergence point** is a node reachable from 2+ branches — it's the merge point, not part of any single branch group

### Visual Rendering — Shaded Regions

For each branch group, compute the bounding box of all member nodes (using their layout positions), add padding (20px). Render a rounded `<rect>` behind the branch with:

- **Fill:** pastel tint derived from branch index, rotated through a 6-color palette (light blue, light green, light yellow, light pink, light purple, light orange)
- **Stroke:** matching color but darker, dashed
- **Render order:** before edges and nodes (back layer in the SVG)

### Visual Rendering — Edge Color Coding

All edges within a branch group get the branch's color as a subtle tint on top of the theme's edge stroke color. This is not a full color replacement — it's a blend that preserves the theme's visual identity while making branches distinguishable.

### Subgraph Interaction

If a branch group is entirely contained within a user-defined subgraph, the subgraph's styling takes precedence. The branch shaded region is suppressed for that group to avoid double-shading.

### Threshold

Grouping only triggers for nodes with 3+ outgoing edges. For 2 edges (simple if/else), the visual noise of shading isn't justified. Only the routing improvements from port assignment apply.

### No New Syntax

The grouping is entirely automatic, detected from graph structure. Users don't need to write anything different. A future config-based opt-out is possible but out of scope for Phase B.

## Files Modified

| File | Change |
|---|---|
| `pkg/layout/layout.go` | Add `ExitPorts`/`EntryPorts` to `NodeLayout`; add port assignment pass after position phase; update `buildEdges` to start polylines from ports |
| `pkg/layout/internal/position/position.go` | Expose rank metadata needed for port calculation |
| `pkg/renderer/flowchart/edges.go` | Branch-aware label placement (first-segment positioning); edge color tinting for branch groups |
| `pkg/renderer/flowchart/renderer.go` | Render branch group shaded regions (back layer); branch detection orchestration |
| `pkg/renderer/flowchart/branch.go` | **New file** — branch detection, group computation, color palette, region bounding box |
| `pkg/renderer/flowchart/theme.go` | Add branch color palette (6 pastel tints + darker stroke variants) |
| `pkg/renderer/flowchart/renderer_test.go` | Tests for branch grouping, edge coloring, region rendering |
| `pkg/layout/layout_test.go` | Tests for port assignment, edge geometry with ports |
| `examples/flowchart/*.mmd` | New example files showing branching patterns |

## Testing Strategy

- **Layout tests:** Port assignment correctness — verify port positions for 3, 4, 5+ edges on diamond, rect, hexagon nodes. Verify edge polylines start from assigned ports.
- **Renderer tests:** Spot-check branch group region renders as `<rect>`. Verify edge label position is on first segment near exit port. Verify color coding. Verify subgraph suppression.
- **Integration:** Full `.mmd` files through parse→layout→render. Compare SVG structure (golden files) for diamond branching patterns.
- **Backward compatibility:** Existing tests must pass unchanged. Nodes with ≤2 outgoing edges must produce identical output.

## Implementation Stages

1. **Stage 1 — Layout: Port assignment + edge geometry.** Add `ExitPorts`/`EntryPorts` to `NodeLayout`, implement port assignment pass, update `buildEdges`.
2. **Stage 2 — Renderer: Improved label placement.** First-segment label positioning for branch edges.
3. **Stage 3 — Renderer: Branch grouping.** Detection, shaded regions, edge color coding, subgraph suppression.
4. **Stage 4 — Examples + integration tests.** New `.mmd` files, golden-file tests, backward-compatibility verification.
