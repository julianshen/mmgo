# Sequence Diagram Parity Audit (mmgo vs mmdc)

**Date:** 2026-04-27 (last refreshed 2026-05-01)
**Inputs:** `examples-mmdc/sequence/*.mmd` (26 files)
**Renderers:** mmdc 11.x (Mermaid 10), mmgo HEAD (`docs/sequence-phase1-plan` branch)
**Method:** rendered each `.mmd` with both tools, visually compared PNG output for the 12 most representative cases (full set rendered to SVG).

**Status (2026-05-01):** Phase B (G1–G11 spec gaps), Phase C (V1–V7 visual polish), and the post-merge mmdc-comparison sweep (Q1–Q7) are all **complete**. Each item below is annotated with the merged PR. Items in *Other minor (deferred)* remain open as low-priority polish.

## Summary

| Category | Count |
|---|---|
| Render success (mmdc) | 25 / 26 (`comprehensive.mmd` puppeteer crash) |
| Render success (mmgo) | 25 / 26 (`activations.mmd` parser error — see G1) |
| Visually faithful to mmdc | ~3 / 25 at audit time; ~25 / 25 after Phase B+C (2026-05-01) |
| Spec-feature gaps | 7 (G1–G7) + 4 omissions (G8–G11, see below) |
| Visual-polish gaps | 7 (V1–V7; V8 retracted on review) |

The phase-A1 plan already routes most gaps to phase B (spec) and phase C (visual). This audit reconciles plan-buckets with observed reality.

## Completion table (refreshed 2026-05-01)

| Gap | Plan bucket | Merged PR |
|---|---|---|
| G1 — standalone `activate`/`deactivate` | B0 | #104 |
| G2 — bidirectional arrowheads | B5 | #105 |
| G3 — autonumber badge | B1 | #107 |
| G4 — `box` grouping visual defects | B2 | #111 |
| G5 — `create`/`destroy` rendering | B3 + B4 | #112 |
| G6 — `title:` directive | B8 | #106 |
| G7 — `<br/>` line breaks | B9 | #106 |
| G8 — YAML frontmatter | B8 | #106 |
| G9 — `accTitle`/`accDescr` | B8 | #107 |
| G11 — `autonumber off` | B1 | #107 |
| Rect label badge / fill clip | B6 | #110 |
| V1 — lifeline theme color | C/V1 | #113 |
| V2 — filled arrowheads | C/V2 | #114 |
| V3 — cross marker `×` | C/V3 | #115 |
| V4 — branch label vs message overlap | C/V4 | #116 |
| V5 — nested-block indentation | C/V5 | #117 |
| V6 — self-message loop arc | C/V6 | #118 |
| V7 — nested activation offset | C/V7 | #119 |

**Open:** G10 (`links`/`properties` participant metadata — low priority); items under *Other minor (deferred)* below.

## Post-merge mmdc visual comparison (Q1–Q7, 2026-05-01)

After Phase B+C merged, rendered 10 representative mmgo PNGs alongside fresh mmdc references and identified seven remaining gaps. All fixed in a same-day sweep.

| Gap | Symptom | Fix | Merged PR |
|---|---|---|---|
| Q1 | Block borders solid; mmdc dashed | `stroke-dasharray:5,5` for non-rect blocks | #122 |
| Q2 | Bracket labels left-anchored next to kind tab; mmdc centered | Center at `x + w/2`, `text-anchor:middle` | #122 |
| Q3 | Self-message label clipped on right of loop arc | Render left of lifeline (`text-anchor:end`); add `selfMsgLeftBleed` to layout | #123 |
| Q4 | Box grouping fills nearly invisible (opacity 0.15) | Bump `defaultBoxFillOpacity` 0.15 → 0.5 | #124 |
| Q5 | Block-kind tab square + dark stroke | Rounded corners (`RX/RY=6`) + ParticipantStroke | #126 |
| Q6 | Rect block fill drawn after content (washed out text); had dark border | Emit fill rect *before* content; drop stroke | #125 |
| Q7 | Activation bars used purple participant theme | Neutral `#F4F4F4` fill + `#666` stroke | #125 |

---

## Spec-feature gaps (parser/AST/data)

### G1 — Standalone `activate` / `deactivate` keywords not parsed
**File:** `activations.mmd`
**Symptom:** `mmgo: svg render: parse: line 5: unrecognized statement: "activate Server"`
**Status:** parser dispatch in `pkg/parser/sequence/parser.go` only recognises the `+`/`-` shorthand on arrows (`A->>+B`); standalone keywords fall through to the "unrecognized statement" branch.
**Severity:** **Critical** — silently encourages users to rewrite valid Mermaid input.
**Plan bucket:** new sub-task **B0** (was missing from phase-A1 plan; B1 handles autonumber render but no task covered standalone activate/deactivate). Add as B0 before B1.
**Fix sketch:** lex `activate <id>` → `Message{From=id, To=id, Lifeline=LifelineEffectActivate}` style sentinel, or a new `SequenceItem` kind. Existing renderer activation logic should accept either source.

### G2 — Bidirectional arrows render only one arrowhead
**File:** `arrows_bidirectional.mmd`
**Symptom:** `<<->>` shows arrowhead only on the *destination* end; `<<-->>` shows arrowhead only on the *source* end. mmdc shows arrowheads on both ends.
**Status:** `ArrowSolidBidir` / `ArrowDashedBidir` are present in the AST and parsed correctly; `pkg/renderer/sequence/messages.go` arrowhead emission only inspects one side of the arrow type.
**Severity:** High — feature appears to work but is wrong.
**Plan bucket:** **B5** (already in plan; downgrade scope from "implement" to "fix renderer end-cap selection").

### G3 — Autonumber rendered as plain text, not numbered badge
**File:** `autonumber.mmd`, `autonumber_custom.mmd`
**Symptom:** mmgo prints `1`, `2`, … as text above each message label. mmdc draws a filled black circle on the source side of each arrow with white numerals.
**Status:** `SequenceDiagram.AutoNumber` is read; numbering counter advances; renderer uses `<text>` only.
**Severity:** Medium — functionally present, visually wrong.
**Plan bucket:** **B1** (downgrade from "implement counter" to "render badge"). **Correction:** B7 (start/step) IS implemented — verified `autonumber 10 5` produces `10, 15, 20, …` in `autonumber_custom.mmd`. Mark B7 done; only the badge visual remains.

### G4 — `box <color> <name>` grouping has rendering defects
**File:** `box_grouping.mmd`
**Symptoms:**
- Title text is clipped to the box's left edge (`Backend` → `Bac`).
- Box border is **dashed**; mmdc uses **solid**.
- Bottom participant boxes are drawn *outside* the group box; mmdc draws them inside.
**Status:** AST has `BoxGroup`; parser handles color literal; renderer dimensions/clipping logic is wrong.
**Severity:** High — visible breakage on a flagship feature.
**Plan bucket:** **B2** (re-open; was marked complete on the feature branch).

### G5 — `create` / `destroy` participant rendering incomplete
**File:** `create_destroy.mmd`
**Symptoms:**
- ✕ destruction marker present (good), but **no bottom participant box** repeated at destruction time. mmdc draws the box at the destruction Y-coordinate.
- New participant box overlaps the spawning arrow; mmdc draws the arrow *to the box's left edge*.
- After destruction, mmgo continues drawing the lifeline as an unbroken vertical line down through to the next message ("Join" still reaches a destroyed Worker2).
**Status:** Create/destroy AST exists; renderer positions new participants but does not gate subsequent messages or add destruction box.
**Severity:** High.
**Plan bucket:** **B3 + B4** (both re-open).

### G6 — `title:` directive not parsed
**Symptom:** `parse: line N: unrecognized statement: "title: My Title"`
**Status:** No example file currently exercises this; verified ad-hoc. Mermaid spec accepts both `title: Foo` and the YAML frontmatter `---\ntitle: Foo\n---` form at the diagram top.
**Severity:** Medium — common in real-world diagrams; silent break.
**Plan bucket:** new **B8** — add to phase B. Add an `examples-mmdc/sequence/title.mmd`.

### G6.1 — Severity upgrade
G6 is upgraded from Medium to **High**. Same failure mode as G1 (parser hard-error on common Mermaid syntax). Caught during review.

### G7 — `<br/>` in message labels not honored as line break
**Symptom:** `A->>B: line one<br/>line two` renders as the literal string `line one<br/>line two` (HTML-escaped). mmdc splits into two `<tspan>` lines.
**Status:** Renderer treats label as a single text run; no `<br/>` tokenisation in the label-formatting path.
**Severity:** Medium — common pattern for multi-line messages.
**Plan bucket:** new **B9** — split labels on `<br>` / `<br/>` and emit `<tspan>` per line. Affects message labels, note text, block condition text. Add `examples-mmdc/sequence/multiline_labels.mmd`.

### G8–G11 — Omissions surfaced during review

These features have no example file yet and were not exercised in the initial pass. Each is verified absent from `pkg/parser/sequence/parser.go`'s top-level dispatch.

- **G8** — YAML frontmatter (`---\ntitle: Foo\n---`). Mermaid spec accepts this on every diagram type. Pair with B8.
- **G9** — `accTitle` / `accDescr` accessibility directives. Common in Mermaid output for screen-reader compatibility. Bucket: B8 sibling.
- **G10** — `links` / `link` / `properties` directives on participants (per-participant metadata). Lower priority — rarely used in flow examples.
- **G11** — `autonumber off` to disable a previously-enabled counter mid-diagram. Parser at line 144 accepts only numeric args; "off" produces a strconv error. Pair with B1.

Add probe `.mmd` files under `examples-mmdc/sequence/` for each before fix-PRs land.

### Confirmed working (no gap)

- All **10** arrow types present and rendered with correct line style (`->`, `-->`, `->>`, `-->>`, `-x`, `--x`, `-)`, `--)`, `<<->>`, `<<-->>`). G2 only affects bi-dir arrowheads; V3 affects the `-x` cross marker visual.
- `participant Foo as Bar` aliasing (`aliases.mmd`).
- `actor` vs `participant` distinction (`actor_vs_participant.mmd` — renders both as boxes; mmdc draws stick-figure for actor — see V8).
- Notes: left/right/over single, over multiple — `note_positions.mmd` is the closest to mmdc of all examples.
- `loop`, `opt`, `alt/else`, `par/and`, `critical/option`, `break` blocks parse and render.
- `rect` block parses (color value works for `rgb()` syntax).
- Bare `rect` (no color) renders cleanly in mmgo with transparent fill — mmdc renders an opaque dark-gray box that occludes its contents (`rect_background.mmd`). mmgo's behaviour is arguably more useful, but is a parity divergence worth noting.
- `%%` line comments are skipped (verified ad-hoc).
- `autonumber 10 5` start/step (was previously listed as gap; corrected — works).

---

## Visual-polish gaps (renderer styling)

### V1 — Lifelines: thin gray vs mmdc's thick purple
Mermaid default theme uses `--lineColor: #333` and `--actorLineColor: <theme-purple>`; mmgo uses a single thin gray stroke. Affects every diagram.
**Plan bucket:** **C — theme integration** (currently a 1-line note in plan; expand).

### V2 — Arrowheads: open vs filled triangle
mmgo renders the filled-head arrows (`->>`, `-->>`) with an *open* `<polyline>` triangle. mmdc fills the triangle.
**Bucket:** C.

### V3 — Cross marker (`-x`, `--x`) renders as open arrowhead
Should be an `×` glyph at the destination end. Currently visually indistinguishable from the open-async arrow (`-)`).
**Bucket:** C.

### V4 — Block label collisions with messages
**Files:** `alt_else.mmd`, `nested_blocks.mmd`
Bracketed condition labels (e.g. `[invalid credentials]`) are drawn at the same Y as the immediately following message, producing overlapping text. mmdc reserves a label-row above the section.
**Bucket:** C — block layout pass needs label-height accounting.

### V5 — Nested blocks render flat, not indented
**File:** `nested_blocks.mmd`
mmdc shows nested borders inset from the parent. mmgo draws all block borders at the same x-extents.
**Bucket:** C.

### V6 — Self-message renders as tiny rectangle, not loop arc; clips text
**File:** `self_message.mmd`
mmdc: ~80px right-side loop arc, message text on the left side of the lifeline. mmgo: tiny rectangular loop, text overflows ("Recu…", "Inter…").
**Bucket:** C.

### V7 — Activation bars don't offset for nested calls
**File:** `activations_nested.mmd`
mmdc draws each level of nesting offset to the right by ~half-bar-width. mmgo collapses all levels onto the same bar.
**Bucket:** C.

### V8 — *(retracted)*
Original claim: mmgo draws a regular participant box for `actor`. **Wrong.** `renderActor()` at `pkg/renderer/sequence/renderer.go:279` draws head + body + arms + legs; `actor_vs_participant.mmd` produces correct stick-figures on both top and bottom rows. Caught during review. V-numbering preserved so downstream references stay stable.

### Other minor (deferred)

- mmgo participant boxes use thinner border + different corner radius.
- No background message-row striping (mmdc has subtle alternation under blocks).
- `rect` color block bleeds over participant boxes vertically (G4 sibling — also pin to B6).
- "rect" appears as a label badge in the rect block (mmdc has no badge for rect). Belongs to **B6**.

---

## Recommended PR sequencing (revises phase-A1 plan §"Sequencing")

1. **B0** (new) — accept standalone `activate`/`deactivate`. Smallest PR; un-breaks `activations.mmd`. **First.**
2. **B8 + B9** (new, expanded) — `title:` directive (G6), YAML frontmatter (G8), `accTitle`/`accDescr` (G9), `<br/>` line-break tokenisation (G7). All pure parser/text-layout, low blast radius. Add probe examples first.
3. **B5 fix** — bidirectional arrowhead end-cap selection.
4. **B1 fix** — autonumber numbered-circle badge + `autonumber off` parsing (G11). Start/step (B7) already done — drop B7 from queue.
5. **B6 fix** — strip "rect" label badge, clip color band to message rows only.
6. **B3 + B4** (covers G5) — create/destroy positioning + bottom box at destruction. One PR.
7. **B2 fix** — box grouping title clipping, solid border, bottom-participant containment.
8. **C / V1–V7** — visual polish, batched per-feature (theme, arrowheads, cross marker, block-label collision, nested-block indentation, self-loop arc, activation-bar nesting offset).

After each PR, re-render the corresponding `examples-mmdc/sequence/*.mmd` and update this audit's status table.

---

## Reproducing this audit

```bash
# Render reference
mkdir -p /tmp/seq-audit/{mmdc,mmgo}
for f in examples-mmdc/sequence/*.mmd; do
  name=$(basename "$f" .mmd)
  mmdc -i "$f" -o "/tmp/seq-audit/mmdc/$name.svg"
  mmdc -i "$f" -o "/tmp/seq-audit/mmdc/$name.png"
done

# Build + render mmgo
go build -o /tmp/mmgo-bin ./cmd/mmgo
for f in examples-mmdc/sequence/*.mmd; do
  name=$(basename "$f" .mmd)
  /tmp/mmgo-bin -i "$f" -o "/tmp/seq-audit/mmgo/$name.svg"
  /tmp/mmgo-bin -i "$f" -o "/tmp/seq-audit/mmgo/$name.png"
done
```

Side-by-side diff each pair manually, or `compare -metric AE` for a rough pixel delta once renderers are close enough that geometric diff is meaningful.
