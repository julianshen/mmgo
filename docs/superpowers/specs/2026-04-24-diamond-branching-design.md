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

Layout-engine port assignment. After the Sugiyama positioning phase completes, a new pass assigns exit ports to multi-outlet nodes. Edges are routed through these ports. The renderer uses branch structure for grouping and color coding. No full orthogonal router needed.

## Branch Detection & Port Assignment

### Branch Node Definition

A node with **3+ outgoing edges** is a "branch node." Nodes with 1-2 outgoing edges are unchanged — existing center-to-boundary clipping produces acceptable results for simple forks.

### Shape Information Flow

The layout engine currently has no shape awareness — it operates on `graph.NodeAttrs` (width, height, label). Shape lives in `diagram.NodeShape` in the renderer domain. To enable shape-aware port positioning:

1. Add an optional `Shape` field to `graph.NodeAttrs` (zero value = `NodeShapeUnknown` = treat as rectangle)
2. The flowchart renderer sets this field when constructing the layout graph (in `Render()` or a new helper)
3. The layout engine reads `attrs.Shape` to choose port positioning strategy; when zero/unknown, falls back to rect-based even spacing

This is backward-compatible: all existing callers that don't set `Shape` get rect behavior. No other diagram types are affected.

### Port Assignment Algorithm

Runs after the position phase, as a new pass in `Layout()`:

1. Identify all branch nodes (3+ outgoing edges)
2. For each branch node, collect outgoing edges sorted by **target X position** (left to right for TB/BT; target Y position top-to-bottom for LR/RL)
3. Compute exit port positions distributed around the node boundary:
   - **Diamond:** For TB, use bottom vertex (center exit), left vertex, and right vertex for 3 edges. For >3 edges, interpolate along the bottom-left and bottom-right edges of the diamond. For LR, rotate accordingly (right vertex = center exit, top and bottom = sides)
   - **Hexagon:** Same approach as diamond, using the 3 vertices on the exit side
   - **Other shapes / unknown:** Distribute ports evenly along the exit side of the bounding box (bottom edge for TB/BT, right edge for LR/RL)
4. Entry ports are **deferred to a future phase.** Only `ExitPorts` is implemented in Phase B. The `EntryPorts` field is omitted from `NodeLayout` to avoid dead code; it can be added when entry-side routing is needed.

### NodeLayout Extension

```go
type NodeLayout struct {
    X, Y, Width, Height float64
    ExitPorts  []Point  // one per outgoing edge; empty for non-branch nodes
}
```

When `ExitPorts` is empty (the common case — nodes with ≤2 outgoing edges), the existing center-to-boundary clipping behavior is unchanged. Backward-compatible.

### Edge-to-Port Mapping

`ExitPorts` is ordered to match the sorted edge iteration in `buildEdges()`. The layout engine's `buildEdges` already iterates edges in `layoututil.SortEdges` order (by From, To, then ID). For a branch node with N outgoing edges, the first N entries in `ExitPorts` correspond to those edges in the same SortEdges order. If this ordering doesn't match the spatial left-to-right ordering from step 2 of the algorithm, the ports are reordered to match SortEdges order before being stored.

Concretely: during port assignment, edges are sorted spatially to compute ideal port positions, then the port array is re-sorted into SortEdges order so `buildEdges` can index by `i % len(ExitPorts)`.

### Edge Geometry Update

`buildEdges()` assigns each outgoing edge to its corresponding port by index. The polyline starts at the port point instead of the node center, then continues through any dummy waypoints to the target. The target endpoint is still clipped using the existing `ClipToRectEdge` / shape-aware clipping as today.

## Smart Edge Routing

### Segment Construction

For each edge from a branch node, the polyline becomes:

1. **Exit port** — the assigned port on the source node boundary
2. **Bend point** — computed by extending **along the outward ray from node center through the port** by `RankSep/2`. For a bottom-center port this is straight down; for a left vertex port on a diamond this is down-and-left at the diamond's edge angle, then the bend continues along rank progression direction (down for TB). The bend direction is always rank-progression-direction (down for TB, up for BT, right for LR, left for RL) regardless of which port the edge exits from — this keeps stems parallel and visually clean.
3. **Existing dummy waypoints** — threaded through as today, starting from the bend point
4. **Entry clip** — clipped to target boundary as today

The short stem segments fan out from distinct ports to their bend points, then become parallel tracks through the dummy chain. This produces the classic decision-tree look.

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

### Branch Group API

The `branch.go` module exports:

```go
type BranchGroup struct {
    SourceNodeID string   // the branch node that originates this group
    EdgeIndex    int      // which outgoing edge (0-based, in AST edge order)
    NodeIDs      []string // all nodes belonging to this branch
    ColorIndex   int      // index into the color palette
    EdgeFromTo   [][2]string // (from, to) pairs for edges in this group
}

func DetectBranches(d *diagram.FlowchartDiagram, l *layout.Result) []BranchGroup
```

`DetectBranches` takes the AST and layout result and returns branch groups. `renderer.go` calls this once and uses the result for region rendering and edge tinting. This makes the branch detection independently testable without the full render pipeline.

### Detection Algorithm

After layout, for each branch node (3+ outgoing edges):

1. Build a forward adjacency map from the AST edges
2. Walk the graph from each outgoing edge's target, collecting all reachable nodes
3. Stop walking at other branch nodes (they start their own groups) or convergence points
4. A **convergence point** is a node reachable from 2+ branches — it's the merge point, not part of any single branch group
5. When a node is both a convergence point and a branch node (merges paths then forks again), it is treated as a convergence point only — it stops the walk from incoming branches but starts its own branch groups from its own outgoing edges

### Visual Rendering — Shaded Regions

For each branch group, compute the bounding box of all member nodes (using their layout positions), add padding (20px). Render a rounded `<rect>` behind the branch with:

- **Fill:** pastel tint derived from branch index, rotated through a 6-color palette (light blue, light green, light yellow, light pink, light purple, light orange)
- **Stroke:** matching color but darker, dashed
- **Render order:** before edges and nodes (back layer in the SVG)

### Visual Rendering — Edge Color Coding

All edges within a branch group (identified by `EdgeFromTo` pairs) get the branch's color as a subtle tint on top of the theme's edge stroke color. This is not a full color replacement — it's a blend that preserves the theme's visual identity while making branches distinguishable.

### Subgraph Interaction

A branch group is suppressed (no shaded region, no edge tinting) when **all of its member nodes belong to the same user-defined subgraph** (checked via node membership in the AST, not bounding-box overlap). The subgraph's own styling provides visual grouping; double-shading is avoided.

### Threshold

Grouping only triggers for nodes with 3+ outgoing edges. For 2 edges (simple if/else), the visual noise of shading isn't justified. Only the routing improvements from port assignment apply.

### No New Syntax

The grouping is entirely automatic, detected from graph structure. Users don't need to write anything different. A future config-based opt-out is possible but out of scope for Phase B.

## Files Modified

| File | Change |
|---|---|
| `pkg/layout/graph/graph.go` | Add optional `Shape int8` field to `NodeAttrs` |
| `pkg/layout/layout.go` | Add `ExitPorts` to `NodeLayout`; add port assignment pass after position phase; update `buildEdges` to start polylines from ports |
| `pkg/layout/internal/position/position.go` | Expose rank metadata needed for port calculation |
| `pkg/renderer/flowchart/edges.go` | Branch-aware label placement (first-segment positioning); edge color tinting for branch groups |
| `pkg/renderer/flowchart/renderer.go` | Render branch group shaded regions (back layer); set Shape on graph nodes before calling Layout |
| `pkg/renderer/flowchart/branch.go` | **New file** — `BranchGroup` type, `DetectBranches()`, color palette, region bounding box |
| `pkg/renderer/flowchart/theme.go` | Add branch color palette (6 pastel tints + darker stroke variants) |
| `pkg/renderer/flowchart/renderer_test.go` | Tests for branch grouping, edge coloring, region rendering |
| `pkg/layout/layout_test.go` | Tests for port assignment, edge geometry with ports |
| `examples/flowchart/*.mmd` | New example files showing branching patterns |

## Testing Strategy

- **Layout tests:** Port assignment correctness — verify port positions for 3, 4, 5+ edges on diamond, rect, hexagon nodes. Verify edge polylines start from assigned ports. Test with all rank directions (TB, BT, LR, RL).
- **Branch detection tests:** Unit tests for `DetectBranches()` — simple branching, nested branching, convergence points, convergence+branch nodes, subgraph suppression.
- **Renderer tests:** Spot-check branch group region renders as `<rect>`. Verify edge label position is on first segment near exit port. Verify color coding.
- **Integration:** Full `.mmd` files through parse→layout→render. Compare SVG structure (golden files) for diamond branching patterns.
- **Backward compatibility:** Existing tests must pass unchanged. Nodes with ≤2 outgoing edges must produce identical output. All existing `NodeLayout` construction must use field names (not positional) to avoid breakage from the new `ExitPorts` field.

## Implementation Stages

1. **Stage 1 — Layout: Port assignment + edge geometry.** Add `Shape` to `NodeAttrs`; add `ExitPorts` to `NodeLayout`; implement port assignment pass; update `buildEdges`; wire Shape from renderer.
2. **Stage 2 — Renderer: Improved label placement.** First-segment label positioning for branch edges.
3. **Stage 3 — Renderer: Branch grouping.** `branch.go` with `DetectBranches()`, shaded regions, edge color coding, subgraph suppression.
4. **Stage 4 — Examples + integration tests.** New `.mmd` files, golden-file tests, backward-compatibility verification.
