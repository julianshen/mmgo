# Phase C: Loop & Condition Blocks

## Problem

Flowchart diagrams commonly depict loops (while/for/repeat) and conditional branches (if/else), but mmgo renders these with no visual distinction from ordinary forward edges. Back-edges look identical to forward edges, self-loops are completely broken (zero-length lines), and there is no visual grouping for loop bodies or condition branches. The Phase B edge tinting feature was also left unimplemented.

## Scope

- **Phase B gap fix:** Add `EdgeFromTo` to `BranchGroup` and implement per-branch edge color tinting
- **Back-edge annotation:** Propagate back-edge identity from the acyclic phase through to the renderer via `EdgeLayout.BackEdge`
- **Back-edge rendering:** Dashed stroke + curved bezier paths that visually distinguish backward flow from forward flow
- **Self-loop rendering:** Proper arc geometry for A→A edges instead of zero-length lines
- **Loop pattern detection:** Diamond → body → back-edge to diamond/ancestor, with warm-palette shaded region and loop label
- **Condition pattern detection:** Diamond → branches that converge at a single merge node, with cool-palette shaded region

Out of scope: orthogonal edge routing, new Mermaid keywords, config-based opt-out toggle.

## Approach

Layout-level back-edge annotation. The acyclic phase already identifies back-edges for reversal — we propagate this information to the renderer. Self-loop arc geometry is generated in the layout engine's `buildEdges`. Loop/condition detection extends the Phase B `DetectBranches` framework with cycle-aware walks. No new syntax needed; patterns are auto-detected from standard Mermaid flowchart syntax.

## Phase B Gap: Edge Tinting

Add `EdgeFromTo [][2]string` to `BranchGroup`. In `renderEdges()`, when an edge's `(From, To)` pair matches a branch group's `EdgeFromTo` entry, blend the branch color into the edge's stroke style at 30% opacity. The base theme stroke color remains dominant but takes on the branch tint.

## Back-Edge Annotation

### EdgeLayout Extension

```go
type EdgeLayout struct {
    Points   []Point
    LabelPos Point
    BackEdge bool  // true if this edge was reversed during acyclic processing
}
```

### Propagation

The acyclic phase already identifies and reverses back-edges. Currently `Layout()` runs `acyclic.Run(work)` on a copy and discards the reversal info. We change this:

1. `acyclic.Run()` returns a `map[graph.EdgeID]bool` of original edges that were reversed (back-edges)
2. `buildEdges()` receives this map and sets `EdgeLayout.BackEdge = true` for matching edges
3. Self-loops (`From == To`) are also flagged — they're never reversed but need special rendering

## Self-Loop Geometry

Self-loops currently get `[center, center]` (zero-length). Instead, `buildEdges()` detects `From == To` and generates arc geometry:

1. **Exit point:** `(cx + w*0.2, cy - h/2)` — slightly right of top center
2. **Entry point:** `(cx - w*0.2, cy - h/2)` — slightly left of top center
3. **Control points:** `(cx + w*0.6, cy - h)` and `(cx - w*0.6, cy - h)` — bow upward
4. Rendered as a cubic bezier `<path>`: `M exit C cp1 cp2 entry`
5. Arrowhead at the entry point, oriented along the final tangent
6. Label (if present) positioned at `(cx, cy - h*0.8)` — top of the arc

The arc sits above the node, producing a visible loop that doesn't overlap node content.

## Back-Edge Rendering

When `EdgeLayout.BackEdge == true`, the renderer applies:

### Dashed Stroke

`stroke-dasharray:6,3` — distinct from `LineStyleDotted` which uses `2,2`.

### Curved Path

Instead of a straight `<line>` or Catmull-Rom spline, back-edges use a **quadratic bezier curve** that bows outward from the main flow direction:

- TB layout: bow to the **right** of the straight-line path
- LR layout: bow **downward** (below the straight-line path)
- Bow magnitude: `max(30px, min(layoutWidth, layoutHeight) * 0.15)`

Rendered as SVG `<path>`: `M x1,y1 Q cx,cy x2,y2` where `(cx, cy)` is the control point offset perpendicular to the midpoint.

For multi-segment back-edges (3+ dummy waypoints spanning multiple ranks), each segment gets its own quadratic bow. Bow direction alternates (right then left) to avoid all back-edges piling up on one side.

### Arrowhead

Same marker as forward edges — the dashed+curved styling provides sufficient visual differentiation.

### Interaction with Port Assignment

Back-edges and self-loops bypass port assignment — they don't use `ExitPorts`. Back-edges start from the node center (clipped to boundary) since they typically originate from non-branch nodes. Self-loops use the dedicated arc geometry above.

## Loop & Condition Pattern Detection

### Pattern Definitions

**Loop pattern:** A diamond node with an outgoing edge that leads (directly or transitively) back to the diamond itself or to an ancestor node. The nodes in the loop body form the "loop group."

**Condition pattern:** A diamond node with 2+ outgoing forward branches that all converge at a single merge node, with no back-edges. The branches form a "condition group."

### Detection Algorithm

Extends `DetectBranches()` in `branch.go`. After computing forward reachability (Phase B logic), a second pass identifies structural patterns:

1. Build a back-edge set from `layout.Result.Edges` — any edge where `BackEdge == true`
2. For each branch node (3+ outgoing edges):
   - Walk each branch forward. If any branch reaches a node that has a back-edge to the branch node or one of its ancestors → **loop pattern detected**
   - If all branches reach a common convergence node with no back-edges involved → **condition pattern detected**
   - If neither pattern matches → **generic branch** (Phase B behavior)
3. For non-branch diamonds (2 outgoing edges):
   - If either outgoing edge is a back-edge or leads to a back-edge within 1-2 hops → **simple loop** (while/do-while)
   - If both branches converge → **simple condition** (if/else)

### BranchGroup Extension

```go
type PatternType int8

const (
    PatternNone      PatternType = iota
    PatternLoop
    PatternCondition
)

type BranchGroup struct {
    SourceNodeID string
    EdgeIndex    int
    NodeIDs      []string
    ColorIndex   int
    EdgeFromTo   [][2]string
    Pattern      PatternType
    BackEdgeTo   string   // for loops: ID of node the back-edge targets
    MergeNodeID  string   // for conditions: ID of convergence node
}
```

### Visual Rendering Per Pattern

**Loop pattern:**
- Shaded region uses a **warm palette** (light orange, light yellow, light coral) — distinct from generic branch regions
- The back-edge gets the dashed+curved treatment from the back-edge rendering section
- A **loop label** is rendered at the top-right of the shaded region, derived from the diamond's label: if it contains loop keywords ("while", "for", "until", "loop", "repeat"), use the full label; otherwise use "loop"
- The back-edge arrow retains its own edge label if present

**Condition pattern:**
- Shaded region uses a **cool palette** (light blue, light green, light teal)
- No special back-edge treatment (all edges are forward)
- The merge node gets a subtle visual cue: a small filled circle or 2px wider border to indicate convergence

**Generic branch (no pattern):**
- Falls back to Phase B behavior: standard branch color palette, no loop/condition labeling

## Files Modified

| File | Change |
|---|---|
| `pkg/layout/layout.go` | Add `BackEdge` to `EdgeLayout`; propagate back-edge set from acyclic through `buildEdges`; self-loop arc geometry |
| `pkg/layout/internal/acyclic/acyclic.go` | Return `map[EdgeID]bool` of reversed edges from `Run()` |
| `pkg/renderer/flowchart/branch.go` | Add `PatternType`; extend `BranchGroup` with `Pattern`/`BackEdgeTo`/`MergeNodeID`/`EdgeFromTo`; extend `DetectBranches()` with loop/condition detection |
| `pkg/renderer/flowchart/edges.go` | Back-edge dashed+curved rendering; self-loop arc rendering; edge tinting from `EdgeFromTo` |
| `pkg/renderer/flowchart/renderer.go` | Loop/condition region rendering with warm/cool palettes; loop label annotation |
| `pkg/renderer/flowchart/theme.go` | Add loop warm palette (3 colors) and condition cool palette (3 colors) |
| `pkg/renderer/flowchart/edges_test.go` | Tests for back-edge styling, self-loop arcs, curved bezier paths |
| `pkg/renderer/flowchart/branch_test.go` | Tests for loop/condition pattern detection, cycle handling, label derivation |
| `pkg/layout/layout_test.go` | Tests for `BackEdge` flag, self-loop geometry |
| `examples/flowchart/*.mmd` | New examples: while-loop, for-loop, nested-conditions, loop-with-condition |

## Testing Strategy

- **Layout tests:** `BackEdge` flag correctness — edges reversed by acyclic are flagged, forward edges are not. Self-loop arc geometry produces a visible non-degenerate path.
- **Branch detection tests:** Unit tests for `DetectBranches()` with loop patterns, condition patterns, nested patterns, 2-node simple loops, cycles with no branch node. Verify `Pattern`, `BackEdgeTo`, `MergeNodeID` correctness.
- **Edge rendering tests:** Back-edges produce `<path>` with `stroke-dasharray`. Self-loops produce `<path>` with cubic bezier. Control point position correctness. Edge tinting: branch-group edges get blended color.
- **Integration:** Full `.mmd` files through parse→layout→render. Golden-file tests for while-loop, if/else, nested loop+condition. Backward compatibility — existing diagrams produce identical output.
- **Label derivation tests:** "while x > 0" → loop label "while x > 0". "valid?" → loop label "loop". Empty label → no annotation.

## Implementation Stages

1. **Stage 1 — Phase B gap + back-edge annotation.** Add `EdgeFromTo` to `BranchGroup`, implement edge tinting. Add `BackEdge` to `EdgeLayout`, propagate from acyclic phase.
2. **Stage 2 — Self-loop geometry.** Layout generates arc paths for self-loops. Renderer draws cubic bezier with arrowhead.
3. **Stage 3 — Back-edge rendering.** Dashed+curved SVG paths for back-edges. Quadratic bezier control point computation.
4. **Stage 4 — Pattern detection.** Extend `DetectBranches` with loop/condition patterns. Add `PatternType`, `BackEdgeTo`, `MergeNodeID`.
5. **Stage 5 — Visual rendering.** Warm/cool palettes, loop labels, merge-node cues, region rendering per pattern.
6. **Stage 6 — Examples + integration tests.** New `.mmd` files, golden-file tests, backward-compatibility verification.
