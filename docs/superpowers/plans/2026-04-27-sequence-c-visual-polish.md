# Phase C — Sequence Diagram Visual Polish

**Date:** 2026-04-27
**Audit gaps:** [V1–V7](2026-04-27-sequence-parity-audit.md#visual-polish-gaps-renderer-styling) (V8 retracted on review)
**Scope:** renderer + theme; no AST or parser changes
**Risk:** low per item; cumulative impact is medium because every example is touched

This phase is intentionally split into seven small PRs. Each PR is independent and can land in any order after B6/B3+B4/B2 are merged. Snapshot refresh (`examples/sequence/*.svg`+`.png`) ships with every PR.

---

## V1 — Lifeline color and weight (theme integration)

**Symptom:** mmgo lifelines are thin gray (`#999`, 1.5px); mmdc renders thick purple (`stroke-width:2px`, `--actorLineColor` from theme).

**Files:**
- `pkg/renderer/sequence/theme.go` — add `LifelineStroke` field; set per-theme defaults.
- `pkg/renderer/sequence/renderer.go` — `renderLifelines` reads `th.LifelineStroke` instead of hard-coded `MessageStroke`.

**Plan:**
1. Add `LifelineStroke string` to `Theme`.
2. Default theme: `#9370DB` (mmdc default actor-line color), stroke-width 2.
3. Dark theme: lighter purple per mmdc's dark variant.
4. Update `renderLifelines` to use the new field.
5. Test: assert `<line>` for lifeline carries the new color, not `MessageStroke`.

**Exit:** `simple.mmd` renders with thick purple lifelines matching mmdc.

---

## V2 — Filled vs open arrowheads

**Symptom:** `->>` and `-->>` should render filled triangle arrowheads. mmgo's polygon (`0,0 10,5 0,10`) is correct shape but `messageLineStyle` may emit `fill:none` on the marker, producing an outline instead of fill.

**Files:**
- `pkg/renderer/sequence/messages.go` — `buildSequenceMarkers` for the solid/dashed arrow markers.

**Plan:**
1. Inspect the solid/dashed marker definitions; verify `fill:%s` (theme stroke color), not `fill:none`.
2. Make sure `bidirArrowhead` polygons (B5 inline) match the marker fill style.
3. Test: regex-grep the rendered SVG for `<marker id="seq-arrow-solid"` then assert its inner `<polygon>` style contains `fill:` followed by a color (not `none`).

**Exit:** `arrows.mmd` shows solid black arrowheads on `->>` and `-->>`, not outlined triangles.

---

## V3 — Cross marker (`-x` / `--x`) renders as ✕

**Symptom:** `-x` and `--x` arrows currently render with the open-arrow polyline. mmdc draws an actual `×` glyph at the destination end.

**Files:**
- `pkg/renderer/sequence/messages.go` — `buildSequenceMarkers` for `ArrowTypeSolidCross` / `ArrowTypeDashedCross`.

**Plan:**
1. Replace the marker's `<polyline>` with two crossed `<line>` elements, or a `<path d="M0,0 L10,10 M0,10 L10,0">`.
2. Stroke-only, no fill.
3. Test: same arrow types now contain `<line>` or `path d="M0,0 L10,10 M0,10 L10,0"` inside the marker definition.

**Exit:** `arrows.mmd` shows ✕ at the end of `-x` and `--x` rows, not an open-arrow shape.

---

## V4 — Block label collisions with messages

**Symptom:** In `alt`/`opt`/`loop`/`par`/`critical`, the bracketed condition label (e.g. `[invalid credentials]`) is drawn at the same y as the immediately following message — text overlap. mmdc reserves a label-row of height ~`fontSize + pad` above each section.

**Files:**
- `pkg/renderer/sequence/messages.go` — `renderBlock` (and the layout side: `countBlockRows` may need to account for one extra row per branch).

**Plan:**
1. In `renderBlock`, when emitting a branch's bracketed label, advance `mr.curY` by `defaultRowHeight/2` *before* emitting the label and *again* before the first message in that branch — the label gets its own y-band.
2. Update `countBlockRows` so layout reserves the matching vertical space (otherwise the diagram height truncates).
3. Test: render an `alt/else/else` block; extract `<text>` content x-positions; assert the bracketed labels and the messages are at non-overlapping y values.

**Exit:** `alt_else.mmd` shows bracket labels above their sections without overlapping the messages.

---

## V5 — Nested blocks render with offset borders

**Symptom:** `nested_blocks.mmd` shows a `loop` containing `alt` containing `par` containing `opt`. mmgo draws every block border at the same x-extents (full diagram width). mmdc inset each level by ~`blockIndent` pixels.

**Files:**
- `pkg/renderer/sequence/messages.go` — `renderBlock` recursive depth tracking.

**Plan:**
1. Pass a `depth` parameter through `renderBlock` (incremented per recursion).
2. Compute `x = blockPad + depth*blockIndent` and `w = lay.width - 2*(blockPad + depth*blockIndent)`.
3. `blockIndent = 8` is a reasonable starting value; tune from visual comparison.
4. Test: render two nested blocks; assert outer rect's x is less than inner rect's x.

**Exit:** `nested_blocks.mmd` shows visibly indented borders for each level.

---

## V6 — Self-message renders as a loop arc, not a tiny rectangle

**Symptom:** `A->>A: callback` currently emits a small 30×20 rectangular path with the label clipped on the right. mmdc draws a ~80px rounded loop arc on the right side of the lifeline with the label on the left.

**Files:**
- `pkg/renderer/sequence/messages.go` — `renderSelfMessage`.

**Plan:**
1. Replace the `M h v h` rectangular path with a quadratic-Bézier loop: `M srcX,y q loopW,−loopH/2 0,loopH` produces a right-side arc.
2. Increase `selfLoopW` to ~50, `selfLoopH` to ~40.
3. Anchor the label `(text-anchor:start, x = srcX + smallGap, y = y + selfLoopH/2)`. Account for label width via `textmeasure.EstimateWidth` and verify it fits between participants.
4. Update `messageRowGapForSelfMessage` (if exists) so layout reserves more vertical room for tall self-loops.
5. Test: render `A->>A: callback`; assert the path's `d` contains a `q` or `c` (quadratic/cubic) command, not just `h v h`.

**Exit:** `self_message.mmd` shows arcs with full label text not clipped.

---

## V7 — Activation bars offset for nested calls

**Symptom:** `activations_nested.mmd` has `Client->>Server: First` (activates), `Client->>Server: Second (nested)` (activates again on already-active Server), then `Server-->>Client: Inner reply` (deactivates inner), `Server-->>Client: Outer reply` (deactivates outer). mmdc renders two visually offset activation bars on Server (inner one shifted right by half-bar-width). mmgo collapses both onto the same bar.

**Files:**
- `pkg/renderer/sequence/messages.go` — `handleLifeline` and `activationRect`.

**Plan:**
1. Track activation depth: `actStack[id]` already holds the y-stack; the index within the stack at activation time IS the depth (0-based).
2. Pass depth to `activationRect` so it offsets the rect by `depth * (defaultActivationW/2)` to the right.
3. Inner-most bar appears most rightward; outermost stays on the lifeline.
4. Test: render two nested activations on the same participant; extract activation `<rect>` x positions; assert inner is to the right of outer.

**Exit:** `activations_nested.mmd` shows two visibly offset activation bars on Server.

---

## Sequencing within Phase C

Independent — pick whichever feels most visible. Suggested order by impact:

1. **V1** (theme) — affects every diagram.
2. **V2** (arrowhead fill) — every diagram with an arrow.
3. **V4** (block label collision) — fixes a real readability issue.
4. **V5** (nested block indent) — pairs naturally with V4.
5. **V6** (self-loop arc) — clearer than current rectangle.
6. **V3** (cross marker) — niche but cheap.
7. **V7** (nested activation offset) — niche.

After all of V1–V7, re-render every example and update the audit doc's "Visually faithful to mmdc" count from ~3/25 toward 25/25.

## Out of Scope (deferred even further)

- Theme system overhaul (true Mermaid theme parity with all CSS variables).
- `init` directive parsing (`%%{init: { theme: 'dark' } }%%`).
- Markdown-italic / -bold inside labels.
- RTL text support.
