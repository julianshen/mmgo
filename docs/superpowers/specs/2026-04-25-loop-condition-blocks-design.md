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

Add `EdgeFromTo [][2]string` to `BranchGroup`. `DetectBranches()` populates this during the reachability walk by collecting `(From, To)` of each edge traversed for that branch.

In `renderEdges()`, when an edge's `(From, To)` pair matches a branch group's `EdgeFromTo` entry, the branch color is applied as a **stroke overlay** using `fill-opacity:0.3` on a separate color attribute, rather than color blending. This composites correctly against any background (light or dark themes) — the base `EdgeStroke` shows through at 70% while the branch tint is visible at 30%.

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

The acyclic phase (`acyclic.Run()`) currently returns `[]graph.EdgeID` of post-reversal edge IDs. We change this:

1. `acyclic.Run()` captures the **pre-reversal** `EdgeID` (before calling `g.ReverseEdge()`) for each back-edge and returns a `map[graph.EdgeID]bool` using original IDs as keys
2. The existing `Undo()` function continues to accept `[]graph.EdgeID` — it receives the same data internally. `Run()` returns both the map and the slice for undo, or the caller extracts the slice from the map keys
3. `buildEdges()` receives the back-edge map (keyed by original EdgeIDs) and sets `EdgeLayout.BackEdge = true` for matching edges
4. Self-loops (`From == To`) are flagged in `buildEdges()` directly — they're never reversed by acyclic but need special rendering

### Self-Loop Flag

Self-loops are detected in `buildEdges()` by `eid.From == eid.To`. They get `BackEdge = false` but `SelfLoop = true` on a new field, or alternatively the renderer checks `From == To` directly. The self-loop geometry is generated inline (see Self-Loop Geometry section) and the standard polyline is not used.

## Self-Loop Geometry

Self-loops currently get `[center, center]` (zero-length). Instead, `buildEdges()` detects `From == To` and generates rank-direction-aware arc geometry:

| Direction | Exit Point | Entry Point | Control Points | Bow Direction |
|---|---|---|---|---|
| TB | `(cx + w*0.2, cy - h/2)` | `(cx - w*0.2, cy - h/2)` | `(cx + w*0.6, cy - h)` and `(cx - w*0.6, cy - h)` | Upward |
| BT | `(cx + w*0.2, cy + h/2)` | `(cx - w*0.2, cy + h/2)` | `(cx + w*0.6, cy + h)` and `(cx - w*0.6, cy + h)` | Downward |
| LR | `(cx + w/2, cy - h*0.2)` | `(cx + w/2, cy + h*0.2)` | `(cx + w, cy - h*0.6)` and `(cx + w, cy + h*0.6)` | Rightward |
| RL | `(cx - w/2, cy - h*0.2)` | `(cx - w/2, cy + h*0.2)` | `(cx - w, cy - h*0.6)` and `(cx - w, cy + h*0.6)` | Leftward |

Rendered as a cubic bezier `<path>`: `M exit C cp1 cp2 entry`. Arrowhead at the entry point, oriented along the final tangent. Label (if present) positioned at the apex of the arc.

The arc sits outside the node in the rank-progression-opposite direction, producing a visible loop that doesn't overlap node content.

## Back-Edge Rendering

When `EdgeLayout.BackEdge == true`, the renderer applies:

### Dashed Stroke

`stroke-dasharray:6,3` — distinct from `LineStyleDotted` which uses `2,2`.

### Curved Path

Instead of a straight `<line>` or Catmull-Rom spline, back-edges use a **quadratic bezier curve** that bows outward from the main flow direction:

- TB layout: bow to the **right** of the straight-line path
- BT layout: bow to the **left**
- LR layout: bow **downward** (below the straight-line path)
- RL layout: bow **upward**
- Bow magnitude: `max(30px, min(layoutWidth, layoutHeight) * 0.15)`

Rendered as SVG `<path>`: `M x1,y1 Q cx,cy x2,y2` where `(cx, cy)` is the control point offset perpendicular to the midpoint.

### Multi-Segment Back-Edges

For back-edges with 3+ dummy waypoints (spanning multiple ranks), each segment between waypoints gets its own quadratic bow. Bow direction **alternates per source node**: all back-edges originating from the same source node alternate right/left (or up/down for LR/RL) in render order. This is tracked by a counter keyed on source node ID during edge rendering.

### Arrowhead

Same marker as forward edges — the dashed+curved styling provides sufficient visual differentiation.

### Interaction with Port Assignment

Back-edges and self-loops bypass port assignment — they don't use `ExitPorts`. Back-edges start from the node center (clipped to boundary) since they typically originate from non-branch nodes. Self-loops use the dedicated arc geometry above.

## Loop & Condition Pattern Detection

### Pattern Definitions

**Loop pattern:** A diamond node with an outgoing edge that leads (directly or transitively) back to the diamond itself or to an ancestor node. The nodes in the loop body form the "loop group."

**Condition pattern:** A diamond node with 2+ outgoing forward branches that all converge at a single merge node, with no back-edges. The branches form a "condition group."

**Mixed pattern:** A branch node where some branches loop back and others converge. Each branch is classified independently — looping branches form loop groups, converging branches form condition groups. A single branch node can produce both loop and condition `BranchGroup` entries.

### Detection Algorithm

Extends `DetectBranches()` in `branch.go`. After computing forward reachability (Phase B logic), a second pass identifies structural patterns:

1. Build a back-edge set from `layout.Result.Edges` — any edge where `BackEdge == true`
2. For each branch node (3+ outgoing edges), evaluate **each branch independently**:
   - If the branch reaches a node that has a back-edge to the branch node or one of its ancestors → **loop branch** (emits a `BranchGroup` with `Pattern = PatternLoop`)
   - If the branch converges with other branches at a common merge node with no back-edges involved → **condition branch** (emits a `BranchGroup` with `Pattern = PatternCondition`)
   - If neither → **generic branch** (Phase B behavior)
3. For non-branch diamonds (2 outgoing edges):
   - If either outgoing edge is a back-edge or leads to a back-edge within 1-2 hops → **simple loop**
   - If both branches converge → **simple condition**

A single branch node can produce multiple `BranchGroup` entries of different pattern types (mixed pattern support).

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
    EdgeFromTo   [][2]string   // populated during reachability walk
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
| `pkg/layout/layout.go` | Add `BackEdge` to `EdgeLayout`; propagate back-edge map from acyclic through `buildEdges`; self-loop arc geometry parameterized by `RankDir` |
| `pkg/layout/internal/acyclic/acyclic.go` | Return `map[EdgeID]bool` of pre-reversal edges from `Run()`; maintain backward compatibility with `Undo()` |
| `pkg/renderer/flowchart/branch.go` | Add `PatternType`; extend `BranchGroup` with `Pattern`/`BackEdgeTo`/`MergeNodeID`/`EdgeFromTo`; extend `DetectBranches()` with loop/condition detection and mixed-pattern support |
| `pkg/renderer/flowchart/edges.go` | Back-edge dashed+curved rendering; self-loop arc rendering; edge tinting from `EdgeFromTo` via opacity overlay; per-source-node bow alternation |
| `pkg/renderer/flowchart/renderer.go` | Loop/condition region rendering with warm/cool palettes; loop label annotation |
| `pkg/renderer/flowchart/theme.go` | Add loop warm palette (3 colors) and condition cool palette (3 colors) |
| `pkg/renderer/flowchart/edges_test.go` | Tests for back-edge styling, self-loop arcs, curved bezier paths, edge tinting |
| `pkg/renderer/flowchart/branch_test.go` | Tests for loop/condition pattern detection, mixed patterns, cycle handling, label derivation |
| `pkg/layout/layout_test.go` | Tests for `BackEdge` flag, self-loop geometry for all rank directions |
| `examples/flowchart/*.mmd` | New examples: while-loop, for-loop, nested-conditions, loop-with-condition |

## Testing Strategy

- **Layout tests:** `BackEdge` flag correctness — edges reversed by acyclic are flagged (using pre-reversal IDs), forward edges are not. Self-loop arc geometry produces visible non-degenerate paths for all four rank directions.
- **Branch detection tests:** Unit tests for `DetectBranches()` with loop patterns, condition patterns, mixed patterns (some branches loop, some converge), nested patterns, 2-node simple loops, cycles with no branch node. Verify `Pattern`, `BackEdgeTo`, `MergeNodeID`, `EdgeFromTo` correctness.
- **Edge rendering tests:** Back-edges produce `<path>` with `stroke-dasharray`. Self-loops produce `<path>` with cubic bezier. Bow alternation is per-source-node. Edge tinting via opacity overlay works on both light and dark backgrounds.
- **Integration:** Full `.mmd` files through parse→layout→render. Golden-file tests for while-loop, if/else, nested loop+condition. Backward compatibility — existing diagrams produce identical output.
- **Label derivation tests:** "while x > 0" → loop label "while x > 0". "valid?" → loop label "loop". Empty label → no annotation.

## Implementation Stages

1. **Stage 1 — Phase B gap + back-edge annotation.** Add `EdgeFromTo` to `BranchGroup`, populate in `DetectBranches()`, implement edge tinting via opacity overlay. Add `BackEdge` to `EdgeLayout`, propagate pre-reversal IDs from acyclic phase.
2. **Stage 2 — Self-loop geometry.** Layout generates rank-direction-aware arc paths for self-loops. Renderer draws cubic bezier with arrowhead.
3. **Stage 3 — Back-edge rendering.** Dashed+curved SVG paths for back-edges. Quadratic bezier control point computation. Per-source-node bow alternation.
4. **Stage 4 — Pattern detection.** Extend `DetectBranches` with loop/condition/mixed patterns. Add `PatternType`, `BackEdgeTo`, `MergeNodeID`.
5. **Stage 5 — Visual rendering.** Warm/cool palettes, loop labels, merge-node cues, region rendering per pattern.
6. **Stage 6 — Examples + integration tests.** New `.mmd` files, golden-file tests, backward-compatibility verification.
