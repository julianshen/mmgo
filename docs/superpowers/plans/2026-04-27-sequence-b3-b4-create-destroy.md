# B3 + B4 — Create / Destroy Participant Polish

**Date:** 2026-04-27
**Audit gap:** [G5](2026-04-27-sequence-parity-audit.md#g5) — `create` / `destroy` participant rendering incomplete
**Scope:** renderer (positioning, lifeline gating, terminal box). Parser already accepts the keywords.
**Risk:** medium — touches participant positioning which affects other features.

## Problem

`examples-mmdc/sequence/create_destroy.mmd` exposes three defects vs mmdc:

1. **Spawn arrow overlap.** When a `create participant Worker1` is followed by `Manager->>Worker1: Spawn`, the new Worker1 box is drawn centered on its lifeline x — and the Spawn arrow draws *into* the box (overlapping). mmdc draws the arrow to the *left edge* of the box, treating the box as the message destination's visual terminator at that y.
2. **No bottom box at destruction.** mmdc draws a participant box at the `destroy <id>` y-coordinate (mirror of the top box). mmgo renders a small ✕ marker (good — keep) but no bottom box. Without the box, the destroyed lifeline's bottom is ambiguous.
3. **Lifeline continues past destroy.** After `destroy Worker2`, mmgo continues drawing Worker2's vertical lifeline to the diagram bottom row. The "Join" message later in the example still reaches a destroyed-but-still-rendered Worker2. mmdc clips the lifeline at the destroy y AND positions the bottom-row Worker2 box *at* that y rather than at the diagram's `bottomY`.

## Files

- `pkg/renderer/sequence/renderer.go` — `renderParticipants` and `renderLifelines`
  - `renderParticipants` already gates top-row creation on `createY[id]` and bottom-row creation on `destroyY[id]`. Verify the gating logic is right when participant has BOTH non-zero (created mid-diagram, destroyed mid-diagram).
  - `renderLifelines` produces the dashed verticals — needs to clip to `[createY[id], destroyY[id]]`.
- `pkg/renderer/sequence/messages.go` — `renderItems` create-participant emission, `renderMessage` arrow termination, `renderDestroy` already places the ✕.
- `pkg/parser/sequence/parser.go` — verify `parseCreate`/`parseDestroy` set `CreatedAtItem`/`DestroyedAtItem` correctly (likely already done).

## Plan

### B3.1 — Spawn arrow stops at box edge (covers G5 defect 1)

When a message's destination is a `create`d participant whose creation y matches the message y, shorten the arrow's `toX` by `participantW[toIdx]/2 + small_gap` (approx 4px). Equivalently: terminate the arrow on the box's left edge instead of the lifeline midline.

Detection: `mr.created[m.To] && mr.createY[m.To] == y`.

For arrows going right-to-left (`m.From > m.To` in x), terminate on the box's right edge instead.

### B3.2 — Bottom box at destroy y (covers G5 defect 2)

In `renderParticipants`, when `destroyY[id] > 0`, emit the bottom box at `destroyY[id]` instead of `lay.bottomY`. The ✕ marker stays at `destroyY[id]` (already there); the bottom box should appear *just below* the marker, not at the diagram bottom. Use `destroyY[id] + smallGap` for the box top.

Edge case: if the participant is BOTH created and destroyed mid-diagram, we get top box at `createY[id]`, ✕ at `destroyY[id]`, bottom box at `destroyY[id] + smallGap`, and the lifeline runs only between createY and destroyY.

### B3.3 — Clip lifeline to [createY, destroyY] (covers G5 defect 3)

In `renderLifelines`, current logic emits a single line spanning `lay.bodyStartY` → `lay.bodyEndY` per participant. Modify to use:
- top = `max(lay.bodyStartY, createY[id])`
- bottom = `min(lay.bodyEndY, destroyY[id])` if destroyY[id] > 0 else `lay.bodyEndY`

Skip lifeline entirely if `top >= bottom` (degenerate case).

### B3.4 — Tests

- `TestRenderCreateParticipantStopsArrow`: render `create participant W` followed by `M->>W: spawn`; assert the message's `<line>` x2 is less than `participantX[W]` (arrow stops at box edge, not lifeline center).
- `TestRenderDestroyEmitsBottomBox`: render `destroy X`; assert a `<rect>` participant box exists at `destroyY[X] + smallGap`.
- `TestRenderDestroyClipsLifeline`: render `destroy X` mid-diagram; assert no lifeline `<line>` for X extends past `destroyY[X]`.
- `TestRenderCreateDestroyCombined`: render a participant created at item 2, destroyed at item 5; verify lifeline only spans those y values.

### B3.5 — Snapshot refresh

- Re-render `examples-mmdc/sequence/create_destroy.mmd` and visually compare against mmdc output.
- `examples/sequence/*` snapshots — none currently exercise create/destroy at the example level; consider adding a `create_destroy.mmd` to `examples/sequence/` so the change is locked into the SVG snapshot test.

## Exit Criteria

- `create_destroy.mmd` renders with: spawn arrows ending at the new box edge; ✕ + bottom box at each destroy y; lifelines clipped to alive intervals.
- No "Join" arrow draws to a destroyed participant beyond its destroy y (test it: the existing `comprehensive.mmd` will catch this if create/destroy is exercised there).
- All existing tests still pass; snapshot diff intentional and reviewed.

## Risks

- Participant `pIndex` lookups in `renderParticipants`/`renderLifelines` may share state with `messageRenderer`'s `created`/`destroyed` maps. Make sure both consult the same source-of-truth.
- Self-messages on a participant whose lifeline is clipped — currently self-loops draw a 30×20 path; verify it still fits when the lifeline window is narrow.

## Out of Scope

- mmdc's special "destroy emits a top-box at the destroy y mirroring the original top label" — only verify after B3.2 lands; defer if mmdc semantics turn out to be more elaborate.
- Activation bars on destroyed lifelines — the existing `actStack` flush at end-of-render assumes lifelines extend to `bodyEndY`. May need to flush at `destroyY[id]` instead. Address as a follow-up if test fails.
