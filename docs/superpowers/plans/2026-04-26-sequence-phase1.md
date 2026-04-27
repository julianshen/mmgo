# Sequence Diagram Phase 1 — Spec & Implementation Plan

> **For agentic workers:** Use `superpowers:subagent-driven-development`. All steps use checkbox (`- [ ]`) syntax. TDD discipline: red → green → refactor.

**Goal.** Bring mmgo's sequence diagram support to feature-complete parity with the [Mermaid sequence diagram spec](https://mermaid.js.org/syntax/sequenceDiagram.html), then close visual parity gaps against the `mmdc` reference renderer.

**Tech stack.** Go, standard library only. >90% line coverage. No new external deps for parsing/layout. Output via existing SVG/PNG/PDF pipeline.

**Source material.**
- Mermaid sequence diagram syntax docs (URL above)
- mmdc reference renders in `examples-mmdc/sequence/*.{svg,png}` (Phase A2 will produce missing ones)
- Existing AST: `pkg/diagram/sequence.go`
- Existing parser: `pkg/parser/sequence/parser.go`
- Existing renderer: `pkg/renderer/sequence/{renderer,messages,svg,theme}.go`

---

## Existing Coverage (baseline, do not regress)

| Feature | Status | Tested |
|---|---|---|
| `participant ID` declaration | ✅ | yes |
| `actor ID` (stick-figure) | ✅ | yes |
| `participant ID as Label` aliases | ✅ | yes |
| Implicit participant from message | ✅ | yes |
| 8 standard arrow types (`->`, `-->`, `->>`, `-->>`, `-x`, `--x`, `-)`, `--)`)  | ✅ | yes |
| Leftmost-longest arrow tokenization (`->>` beats `->`) | ✅ | yes |
| Message label after `:` (preserves additional colons) | ✅ | yes |
| `activate` / `deactivate` keywords | ✅ | yes |
| `+` / `-` activation suffixes on receiver | ✅ | yes |
| `note left of A`, `note right of A`, `note over A` | ✅ | yes |
| `note over A,B` multi-participant span | ✅ | yes |
| Self-messages (`A->>A: ...`) | ✅ | yes |
| `loop ... end` | ✅ | yes |
| `alt ... else ... end` | ✅ | yes |
| `opt ... end` | ✅ | yes |
| `par ... and ... end` | ✅ | yes |
| `critical ... option ... end` | ✅ | yes |
| `break ... end` | ✅ | yes |
| `rect ... end` (no color value) | ✅ | yes |
| Comments (`%% ...`) | ✅ | yes |
| `autonumber` keyword (parser sets flag; render TBD) | ⚠️ partial | parse only |

---

## Spec Gaps — exhaustive list

### Phase B targets (must implement)

| ID | Feature | Syntax example |
|---|---|---|
| **B1** | Autonumber rendering | `autonumber` produces a circled number badge per message |
| **B2** | Box grouping | `box rgb(220,240,255) Frontend\n  participant A\n  participant B\nend` |
| **B3** | Create participant | `create participant Worker` followed by message that introduces it |
| **B4** | Destroy participant | `destroy Worker` terminates the lifeline with an X glyph |
| **B5** | Bidirectional arrows | `<<->>` (solid bi), `<<-->>` (dashed bi) |
| **B6** | Rect with color value | `rect rgb(R,G,B)`, `rect rgba(R,G,B,A)`, `rect #hex` |
| **B7** | Autonumber start/step | `autonumber 10 5` (start at 10, step by 5) |

### Won't fix in Phase 1 (defer or never)

| Feature | Reason |
|---|---|
| `link Actor: label @ url` / `links Actor: {...}` | DOM popup menus — no SVG equivalent. Defer to optional phase 2. |
| `mirrorActors: false/true` config | Niche; default-on already matches mmdc. |
| Half-arrow heads (`-|\\`, `/|--`, etc., v11.12.3+) | Very recent, low usage; defer. |
| HTML entity codes `#NN;` in labels | If existing label decoder handles them (verify in A1), no-op. |
| Sequence number from CSS variable | Theming-only; covered when we expose theme tokens. |
| `mermaid` config block embedded `%%{init: ...}%%` | Requires generic config plumbing; out of scope for sequence-only PR. |

---

## Phase A — Coverage examples & parity audit

**Goal.** Empirically catalogue every visual gap between mmgo and mmdc. Output is a parity audit doc that drives Phase C work items.

### A1 — Verify existing examples (done)

- [x] 23 new `.mmd` files added under `examples-mmdc/sequence/` covering every spec feature.
- [x] 3 pre-existing files retained (`simple.mmd`, `auth_flow.mmd`, `notes.mmd`).
- [x] Each new file isolates one or two related features so visual diffs pinpoint single issues.

### A2 — Generate mmdc reference renders

**Files:** `examples-mmdc/sequence/*.{svg,png}`

- [ ] For each new `.mmd`, run: `mmdc -i examples-mmdc/sequence/X.mmd -o examples-mmdc/sequence/X.svg` and again with `.png`.
- [ ] Verify all 26 files render successfully with mmdc (any failures expose mermaid-cli bugs or our test-input bugs — document and skip).
- [ ] Commit the regenerated artifacts.

### A3 — Generate mmgo renders

**Files:** `examples/sequence/*.{svg,png}`

- [ ] For each `.mmd`, run: `go run ./cmd/mmdc -i examples-mmdc/sequence/X.mmd -o examples/sequence/X.svg` and `.png`.
- [ ] Files using unparsed syntax (`box`, `create`, `destroy`, `<<->>`, `rect rgb(...)`, `autonumber 10 5`) will fail. Capture each failure mode (parse error vs render error vs silent-drop).
- [ ] Track which examples fail in the audit doc.

### A4 — Parity audit doc

**Files:** new `docs/superpowers/plans/2026-04-26-sequence-parity.md`

- [ ] Table per example: `name | mmdc-status | mmgo-status | gap-severity (clean|minor|major|blocked-on-Bx) | notes`.
- [ ] Inline side-by-side image links for each.
- [ ] Categorize gaps by component:
  - actor stick-figure styling
  - participant box (border, shadow, padding, font)
  - lifeline (dash pattern, color, stroke-width)
  - message line (curvature, color, label position)
  - arrow heads (filled triangle, open `>`, cross X)
  - activation bar (width, fill, border)
  - note shape (rounded corners, fold corner, spacing)
  - block frame (label tab, dashed outline)
  - self-message arc (radius, label placement)
  - autonumber badge
  - margins / padding around the whole diagram
- [ ] Each gap entry becomes a line item for Phase C.

---

## Phase B — Spec completeness

Each task is a separate PR. TDD: write failing parser test first, then implementation, then renderer test, then golden file regen.

### B1 — Autonumber rendering

**Files:**
- `pkg/renderer/sequence/messages.go`
- `pkg/renderer/sequence/renderer_test.go`
- regenerate `examples/sequence/autonumber.{svg,png}`

**Tasks:**
- [ ] Add `messageNumber int` counter to `messageRenderer`; increment per call to `renderMessage` and `renderSelfMessage` only (skip notes/blocks).
- [ ] Inside `renderMessage`/`renderSelfMessage`, when `mr.diagram.AutoNumber.Enabled`, emit a small circle (radius ~12px) at the message origin carrying the number text.
- [ ] Match mmdc badge color (theme-driven, default `#fff` fill + `#000` stroke + `#000` text).
- [ ] **Test** `TestAutoNumberRendersBadgePerMessage`: parse `autonumber\nA->>B:hi\nB-->>A:ok`, assert two `<circle>` badges with text `1` and `2`.
- [ ] **Test** `TestAutoNumberSkipsNotesAndBlocks`: insert a note + alt block between messages, assert numbers stay sequential across messages, none on the note/block label.
- [ ] **Test** `TestAutoNumberDisabledByDefault`: no `autonumber` keyword → no badges.

### B2 — Box grouping

**Files:**
- `pkg/diagram/sequence.go` — add `Box` struct + `Boxes` field
- `pkg/parser/sequence/parser.go` — `box ... end` block handling
- `pkg/parser/sequence/parser_test.go`
- `pkg/renderer/sequence/renderer.go` — render box rect spanning member lifelines
- `pkg/renderer/sequence/renderer_test.go`

**Tasks:**
- [ ] AST: `type Box struct { Label string; Fill string; Members []string }`. `SequenceDiagram.Boxes []Box`.
- [ ] AST: each `Participant` gets `BoxIndex int` (-1 if not in a box).
- [ ] Parser: detect `box [color] [label]` line, push a `boxFrame`. While inside, every `participant`/`actor` declaration appends ID to `Members` and records `BoxIndex`. On `end`, pop frame and append `Box` to `Boxes`. Disallow nested boxes (return error).
- [ ] Color value parser: shared with B6. Reuse helper `parseColorValue(s string) (fill string, rest string)` accepting `rgb(...)`, `rgba(...)`, `#hex`.
- [ ] **Test** `TestParseBoxBasic`: `box Frontend\n  participant A\nend` → one Box with Members=[A], no fill.
- [ ] **Test** `TestParseBoxWithColor`: `box rgb(220,240,255) Backend\n...end` → fill=`rgb(220,240,255)`, label=`Backend`.
- [ ] **Test** `TestParseBoxNestedRejected`: nested `box` errors at parse time.
- [ ] **Test** `TestParseBoxParticipantsTagged`: each member's `BoxIndex` matches its container.
- [ ] Renderer: in `renderParticipants`, group by `BoxIndex`, draw a labelled rect spanning the leftmost-to-rightmost member's lifeline x-range, height = full diagram height. Draw before participants so they overlay.
- [ ] **Test** `TestRenderBoxEmitsRectAndLabel`: assert one `<rect>` per box with the correct fill and a label.

### B3 — Create participant

**Files:**
- `pkg/diagram/sequence.go` — add `CreatedAtItem int` field to `Participant`
- parser + parser_test
- renderer (lifeline starts only at the `CreatedAtItem` row)

**Tasks:**
- [ ] AST: `Participant.CreatedAtItem int` (-1 = always present, ≥0 = appears starting at that item index).
- [ ] Parser: `create participant X` (and `create actor X`) declares X with `CreatedAtItem = len(Items)` at parse time. The next message MUST reference X — if not, parser error.
- [ ] **Test** `TestParseCreateParticipantSetsIndex`: `create participant Worker\nManager->>Worker: spawn` → Worker.CreatedAtItem == 0 (the spawn message is item 0).
- [ ] **Test** `TestParseCreateWithoutFollowingMessageErrors`.
- [ ] Renderer: top-of-diagram participant box renders only if `CreatedAtItem == -1`. Otherwise draw the participant box at the row's y-coordinate, and start the lifeline from there.
- [ ] **Test** `TestRenderCreatedParticipantBoxStartsMidDiagram`.

### B4 — Destroy participant

**Files:** AST, parser, renderer

**Tasks:**
- [ ] AST: `Participant.DestroyedAtItem int` (-1 = persists, ≥0 = ends at that item index).
- [ ] AST: a `SequenceItem` may carry a `Destroy *string` (participant ID) so the renderer knows where to draw the X.
- [ ] Parser: `destroy X` emits a destroy item AND sets `Participant.DestroyedAtItem`. Disallow `destroy` on an already-destroyed participant.
- [ ] **Test** `TestParseDestroySetsIndex`.
- [ ] **Test** `TestParseDestroyTwiceErrors`.
- [ ] Renderer: lifeline ends at `DestroyedAtItem`'s y-coordinate, terminated with an X glyph (two crossing lines, ~12px).
- [ ] **Test** `TestRenderDestroyTerminatesLifelineWithX`.

### B5 — Bidirectional arrows

**Files:** AST, parser, renderer

**Tasks:**
- [ ] AST: extend `ArrowType` with `ArrowTypeSolidBi` (`<<->>`) and `ArrowTypeDashedBi` (`<<-->>`). Update `arrowTypeNames`.
- [ ] Parser: extend `arrowTokens` table. Order matters: `<<->>` and `<<-->>` must precede their right-half (`->>`, `-->>`) so leftmost-longest match selects the bi-arrow.
- [ ] **Test** `TestParseBidirectionalSolid`: `A<<->>B: hi` → Message with ArrowTypeSolidBi.
- [ ] **Test** `TestParseBidirectionalDashedBeatsDashed`: ensure `<<-->>` matches before `-->>`.
- [ ] Renderer: in `messageLineStyle` and arrow rendering, emit BOTH `marker-start` and `marker-end`. Reuse the existing arrow marker; mirror it via `orient="auto-start-reverse"` if not already.
- [ ] **Test** `TestRenderBidirectionalEmitsBothArrowheads`.

### B6 — Rect with color

**Files:** AST, parser, renderer

**Tasks:**
- [ ] AST: add `Block.Fill string` (empty = use default theme color).
- [ ] Parser: when block kind is `rect`, accept an optional color expression after `rect`. Reuse `parseColorValue` from B2.
- [ ] **Test** `TestParseRectWithRgb`, `TestParseRectWithRgba`, `TestParseRectWithHex`.
- [ ] **Test** `TestParseRectWithoutColorKeepsFillEmpty`.
- [ ] Renderer: `renderBlock` for `BlockKindRect` uses `Block.Fill` if non-empty, else default theme.
- [ ] **Test** `TestRenderRectUsesCustomFill`.

### B7 — Autonumber start/step

**Files:** AST, parser, renderer

**Tasks:**
- [ ] AST: replace `AutoNumber bool` with `type AutoNumber struct { Enabled bool; Start, Step int }`. Update `SequenceDiagram.AutoNumber`.
- [ ] Parser: `autonumber` → Enabled=true, Start=1, Step=1. `autonumber 10` → Start=10, Step=1. `autonumber 10 5` → Start=10, Step=5. Reject negative or non-integer values.
- [ ] **Test** `TestParseAutonumberDefault`, `TestParseAutonumberStartOnly`, `TestParseAutonumberStartAndStep`, `TestParseAutonumberInvalid`.
- [ ] Renderer: B1's counter starts from `AutoNumber.Start` and increments by `AutoNumber.Step`.
- [ ] **Test** `TestRenderAutonumberCustomStartStep`: 3 messages with `autonumber 10 5` produce badges `10`, `15`, `20`.

### Phase B exit criteria

- [ ] All 7 task groups merged.
- [ ] All 23 new examples now render without parse errors via mmgo.
- [ ] Coverage ≥90%.
- [ ] No regressions in pre-existing 3 examples.

---

## Phase C — Visual parity polish

Driven entirely by Phase A's parity audit doc. Each gap is a line item; related items group into PRs.

### Process

For each Phase C PR:
1. Pick 2–4 related gaps from the audit.
2. Render the affected examples with mmgo, capture before screenshots.
3. Adjust renderer (theme constants, geometry, paths, etc.).
4. Re-render, capture after screenshots.
5. Update the audit doc: mark each gap as `fixed` or `partial` with notes.
6. Regenerate any affected golden files in `pkg/renderer/sequence/testdata/`.

### Likely gap categories (to be confirmed by A4)

These are the structural areas where mmdc and mmgo are most likely to differ; concrete fixes will be defined per gap once measured.

| Category | What to measure | Likely fix surface |
|---|---|---|
| Diagram margins | top/bottom/left/right padding around the whole canvas | renderer constants: `diagramPadX`, `diagramPadY` |
| Participant box | width=max(label,minWidth), height, font, border-radius, shadow | `renderParticipantBox` + `theme.go` |
| Actor stick-figure | head radius, body proportions, stroke-width | `renderActor` |
| Lifeline | dash pattern (`5,5` vs `2,2`), stroke color, top/bottom margins | `renderLifelines` |
| Message line stroke | width (1.5 vs 2), dash style for dashed arrows | `messageLineStyle` |
| Arrow head triangles | size, fill, refX/refY positioning | `buildSequenceMarkers` |
| Async open arrow `>` | open polyline vs filled triangle | `buildSequenceMarkers` for `*Open` types |
| Cross-arrow X | stroke-width, length | `buildSequenceMarkers` for `*Cross` |
| Activation bar | width (~10px), fill color, border, vertical bounds | `activationRect` |
| Self-message arc | radius (~25px?), label position above arc | `renderSelfMessage` |
| Note rectangle | rounded corners, optional fold corner glyph, padding, fill, font | `renderNote` |
| Multi-participant note | width spans accurately, centred label | `renderNote` |
| Block frame | dashed outline color, label tab background, label font | `renderBlock` |
| `alt`/`else` separators | dashed line per branch, branch label position | `renderBlock` |
| Autonumber badge | circle vs square, fill, font size, position offset | B1 implementation |
| `box` group rect | fill alpha, border, label position above box | B2 implementation |
| Vertical message spacing | row height per message | `computeLayout`, `actorHeight`, etc. |
| Note bleed beyond participant | left/right note overhang into margins | `noteBleed` |

### Phase C exit criteria

- [ ] Every audit gap marked `fixed` (or explicitly `won't-fix` with rationale).
- [ ] All 26 example renders visually match mmdc within tolerance (font glyph variance acceptable).
- [ ] Coverage stays ≥90%.
- [ ] No regressions in the existing flowchart goldens.

---

## Cross-cutting concerns

### Theme tokens

All visual constants tunable in Phase C should land in `pkg/renderer/sequence/theme.go` with a `DefaultTheme()` that matches mmdc. This makes future theme work (dark, forest, neutral) a single struct change, not a renderer rewrite.

### Test fixtures

Per-feature parser tests use inline strings. Renderer tests for visual structure assert on emitted SVG element types and counts (not byte equality) — this avoids brittleness from formatting tweaks. Golden-file tests under `pkg/renderer/sequence/testdata/` cover full diagrams and run via the existing golden compare helper.

### Backwards compatibility

The `AutoNumber` field type change in B7 (`bool` → `struct`) is a breaking AST change. Mitigation: do B7 first or alongside B1 so external consumers see one breaking commit, not two.

### Acceptance criteria (Phase 1 overall)

- [ ] All 7 Phase B feature gaps closed with parser + renderer tests.
- [ ] Phase A parity audit doc exists and is referenced by every Phase C PR.
- [ ] All Phase C audit items either `fixed` or `won't-fix` with rationale.
- [ ] `go test ./... -race -cover` passes with ≥90% line coverage on `pkg/parser/sequence/...` and `pkg/renderer/sequence/...`.
- [ ] `golangci-lint run ./...` clean.
- [ ] All 26 example `.mmd` files render via mmgo and produce visually-equivalent output to mmdc.
- [ ] No new external dependencies.

---

## Sequencing

```
Phase A (audit)
  A1 done
  A2 → A3 → A4   (single PR; produces audit doc)

Phase B (spec completeness) — each its own PR
  B7 + B1   (combined: AutoNumber struct breakage + render in one breaking commit)
  B5        (small, independent)
  B6        (small, independent)
  B2        (medium)
  B3 + B4   (combined: lifecycle pair, share AST plumbing)

Phase C (visual parity)
  Iterative PRs driven by the A4 audit doc; group 2–4 related gaps per PR.
```

This sequencing keeps each PR small, independent where possible, and gets the breaking AST change (B7) out of the way first.
