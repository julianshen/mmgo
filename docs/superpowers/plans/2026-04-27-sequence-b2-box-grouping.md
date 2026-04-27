# B2 — Box Grouping Visual Fix

**Date:** 2026-04-27
**Audit gap:** [G4](2026-04-27-sequence-parity-audit.md#g4) — `box <color> <name>` grouping has rendering defects
**Scope:** renderer-only (parser already builds `BoxGroup`/`Box` AST correctly)
**Risk:** medium — touches layout (box width and height computation)

## Problem

`examples-mmdc/sequence/box_grouping.mmd` renders three defects vs mmdc:

1. **Title text clipped.** "Backend" displays as "Bac" — the title is anchored at the box's left edge with no width budget reservation. mmdc center-aligns the title in a top label strip with the full box width available.
2. **Border is dashed.** mmgo emits `stroke-dasharray:5,5` on the box rect; mmdc uses solid stroke for `box`.
3. **Bottom participant boxes outside the group.** The bottom-row participant labels (Browser, SPA, API, Worker) are drawn at `lay.bottomY`, which is below the group box's bottom edge. mmdc draws the group box tall enough to contain both the top and bottom participant rows.

## Files

- `pkg/renderer/sequence/renderer.go` — `renderBoxes` (the only emitter for box-group rectangles)
- `pkg/renderer/sequence/renderer_test.go` — extend `TestRenderBoxEmitsRectAndLabel`

## Plan

### B2.1 — Solid border, not dashed

In `renderBoxes`, the rect's `Style` currently includes `stroke-dasharray:5,5` (or similar). Drop the dasharray so the border renders solid. Verify the rect rect stroke color and width still match mmdc's purple/light-purple palette (or come from the theme).

### B2.2 — Title in a top label strip

Currently the title `<text>` is emitted at `(boxX + smallGap, boxY + fontSize)` with `text-anchor:start`. Replace with:
- Anchor at `(boxX + boxW/2, boxY + titleStripHeight/2)` with `text-anchor:middle`, `dominant-baseline:central`.
- `titleStripHeight = fontSize + 2*titlePadY` (e.g. 18 for fontSize=12).
- Reserve `titleStripHeight` at the top of the box content (account for in B2.3 below).

Optionally draw a separator line at `boxY + titleStripHeight` to match mmdc's subtle divider — verify against reference render before committing to this.

### B2.3 — Box height contains both participant rows

`renderBoxes` computes the rect from `topY` (top participant row) to `bodyEndY` or thereabouts. It should extend to `bottomY + maxHeaderH + smallGap` so the bottom participant row is enclosed.

Concretely:
- `boxY = lay.topY - titleStripHeight - smallGap` (room for the title strip + a top margin).
- `boxH = (lay.bottomY + maxHeaderH + smallGap) - boxY` so the bottom row sits inside.

Verify this doesn't push the diagram's overall height; `computeLayout` may already reserve enough vertical space. If not, add the `titleStripHeight` to the layout's `topY` calculation when boxes are present (similar to how `Title` does it).

### B2.4 — Box width fits the title

Currently `boxW = (xs[lastMember] + widths[lastMember]/2) - (xs[firstMember] - widths[firstMember]/2) + 2*pad`. If the title text is wider than this, the title will visually overflow. Compute `titleW = textmeasure.EstimateWidth(box.Label, fontSize) + 2*titlePadX` and use `boxW = max(boxW, titleW)`.

### B2.5 — Tests

- Update `TestRenderBoxEmitsRectAndLabel` to assert solid stroke (no `stroke-dasharray` substring).
- `TestRenderBoxContainsBothParticipantRows`: render a box wrapping `A,B`; extract the box rect's y/height; assert `boxY <= lay.topY` AND `boxY + boxH >= lay.bottomY + boxHeight` (i.e. bottom row inside).
- `TestRenderBoxTitleNotClipped`: render `box LongTitleNameThatExceedsBoxWidth A` with one short-named participant; assert `titleW <= boxW`.

### B2.6 — Snapshot refresh

- `box_grouping.mmd` re-render via mmdc + mmgo, visually compare.
- Add a probe `examples/sequence/box_grouping.mmd` if it isn't already in the snapshot fixture set.

## Exit Criteria

- `box_grouping.mmd` mmgo render shows: solid borders, full titles centered above each box, bottom-row participants inside the box.
- All existing renderer tests green.
- Layout: viewBox height does not regress for non-box diagrams.

## Risks

- `renderBoxes` and `computeLayout` may not currently coordinate the topY budget for the title strip. If `topY` doesn't reserve room, the title will overlap the participant boxes. Trace the layout path before changing y values.
- Nested boxes (one box inside another) — Mermaid spec allows this. Audit the `Boxes` slice to confirm whether multiple boxes can overlap; if yes, render them in size order so larger ones don't occlude smaller ones.

## Out of Scope

- Background fill color for boxes — orthogonal concern; only adjust if mmdc parity demands it after B2.1–B2.4 land.
- Theming of box stroke color — defer to phase C.
