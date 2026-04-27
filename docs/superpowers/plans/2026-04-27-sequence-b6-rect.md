# B6 — Rect Block Visual Fix

**Date:** 2026-04-27
**Audit gap:** part of G4-sibling notes in [2026-04-27-sequence-parity-audit.md](2026-04-27-sequence-parity-audit.md) (the rect-specific defects under "Other minor")
**Scope:** renderer-only; no AST or parser changes
**Risk:** low (single block kind, isolated rendering path)

## Problem

`rect` blocks render with two visible defects vs mmdc:

1. The literal label badge `[rect]` appears in the top-left of the block. mmdc renders **no** label for `rect` (the rect is a styling primitive, not a labeled section like `loop` or `alt`).
2. The colored fill extends vertically past the message rows it should cover. In `examples-mmdc/sequence/rect_color.mmd` the pink and blue tints visibly bleed over the participant boxes at top and bottom; mmdc clips the fill to the message-row band only.

`rect_background.mmd` (bare `rect` with no color) renders cleanly today — actually *better* than mmdc, which paints a solid dark-gray overlay. Preserve that.

## Files

- `pkg/renderer/sequence/messages.go` — `renderBlock()` is the only rendering site; the `<text>` for the kind label and the `<rect>` for the body are both emitted there.
- `pkg/renderer/sequence/renderer_test.go` — tighten existing `TestRenderRectUsesCustomFill` and add a no-badge regression.

## Plan

### B6.1 — Suppress the "rect" label badge

In `renderBlock` the `kindLabel` text element is emitted unconditionally. Gate it: skip the label emission when `b.Kind == BlockKindRect`. Bracketed condition labels (e.g. `[Authenticated section]`) are also irrelevant for `rect` per the spec — `rect` doesn't take a condition. Drop the bracketed-label emission for rect too.

### B6.2 — Clip color fill to message-row band

Currently the rect's `y` and `height` come from `startY` and `endY - startY`, where `startY = mr.curY` *before* the half-row gap is consumed. That's fine for box-style blocks (alt/loop) where the border should encompass the kind-label row. For `rect` the fill should start at the first message row's top and end at the last message row's bottom, *excluding* the row-gap padding above and below.

Approach: when emitting the rect, use `startY + defaultRowHeight/2` and `endY - defaultRowHeight/2` so the band stops short of the leading/trailing half-row pads.

### B6.3 — Tests

- `TestRenderRectNoLabelBadge`: render a `rect` block, assert the SVG has no `>rect<` text element (the literal kind name) inside the block.
- `TestRenderRectColorClipsToMessageBand`: render `rect rgba(...) ... end` with two messages; extract the colored rect's y-extent; assert it does not overlap the participant box y-extents.

### B6.4 — Snapshot refresh

- `examples/sequence/*.svg` and `.png` — refresh after changes; verify `rect_color.mmd` and `rect_background.mmd` render correctly via `mmdc -i ... -o ...` comparison.

## Exit Criteria

- mmgo `rect_color.mmd` renders without "rect" badge text.
- Colored band does not visually overlap participant boxes (verified by reading PNG at `/tmp` or pixel-extent assertion).
- `rect_background.mmd` continues to render cleanly (no regression).
- All renderer tests green; snapshot tests pass with refreshed examples.
- No changes to `pkg/diagram` or `pkg/parser`.

## Out of Scope

- Block label badges for `loop`/`alt`/`opt`/`par`/`critical`/`break` — those are intentional and mirror mmdc.
- Nested-block rendering (V5 in audit) — separate task.
- Block-condition label collisions (V4) — separate task.
