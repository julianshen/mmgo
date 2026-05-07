# mmgo Implementation Status

Dashboard view of progress against [implementation-plan.md](implementation-plan.md).
Updated at each step boundary and committed to git, so any client can read
current state without relying on chat history.

**Last updated:** 2026-05-08 (post issue-cleanup #190–#194 + Kanban Phase 3 + GitGraph Phase 3 + Sankey NodeAlignment — PRs #197–#203)

## Overall

- **Current milestone:** v0.4.0 complete; working toward v0.5.0 (Gantt, mindmap, others)
- **Current phase:** Phase 6 in progress
- **At full spec parity:** Block (Step 27); Class / State / ER / Gantt / Mindmap; Timeline (Step 25); Quadrant (Step 28); **Sankey (Step 26)**, **Kanban (Step 30)**, and **XY (Step 32)** newly promoted after the issue-cleanup pass
- **Phase 2 / B landed but follow-ups still queued:** **C4** (phases 3–4: named-arg `$descr=`, `UpdateElementStyle`, `LAYOUT_*`, legend); **GitGraph** (Phase 4: TB / BT renderer wiring, `parallelCommits` algorithmic pass, named themes, validation)
- **Next:** Sankey width / height config and `%%{init}%%` block; GitGraph Phase 4 layout direction; new diagram types or v1.0 API stabilization

## Phase 0: Project Scaffold

| Status | Step | PR | Notes |
|--------|------|----|----|
| ✅ | Project scaffold (Go module, CI, Makefile, LICENSE, directory structure) | #2 | |

## Phase 1: Core Infrastructure

| Status | Step | PR | Coverage | Notes |
|--------|------|----|----|---|
| ✅ | 1. Graph data structure (`pkg/layout/graph/`) | #3 | 98.8% | Directed graph with multi-edges, compound (parent/child), topo sort |
| ✅ | 2. Text measurement (`pkg/textmeasure/`) | #4 | 95.2% | Font-based bbox via `golang.org/x/image/font`, bundled Go Regular |
| ✅ | 3. Diagram AST types (`pkg/diagram/`) | #5 | 100% | Flowchart, Sequence, Pie ASTs with typed int8 enums |
| ✅ | 4. Layout — cycle removal (`pkg/layout/internal/acyclic/`) | #6 | 96.1% | Greedy feedback arc set (Eades-Lin-Smyth) |
| ✅ | 5. Layout — rank assignment (`pkg/layout/internal/rank/`) | #7 | 100% | Longest-path ranking; TODO(perf) for network simplex |
| ✅ | 6. Layout — crossing minimization (`pkg/layout/internal/order/`) | #8 | 100% | Barycenter heuristic with 24-iter up/down sweep |
| ✅ | 7. Layout — coordinate assignment (`pkg/layout/internal/position/`) | #10 | 96.7% | Median-based alignment + centered compact; TODO(perf) for Brandes-Kopf |
| ✅ | 8. Layout engine integration (`pkg/layout/`) | #11 | 96.9% | Top-level Layout() with TB/BT/LR/RL directions; straight-line edge routing |

## Phase 2: Flowchart (First Diagram Type)

| Status | Step | PR | Coverage | Notes |
|--------|------|----|----|---|
| ✅ | 9. Flowchart parser (`pkg/parser/flowchart/`) | #12 | 95.9% | 14 shapes, 6 edge ops (+ long-dash variants), inline + pipe edge labels, chained edges, bracket-aware arrow scanning, hyphen IDs |
| ✅ | 10. Flowchart renderer (`pkg/renderer/flowchart/`) | #14 | 92.1% | All 14 shapes, bezier curves, 4 arrow markers, subgraph contents (recursive), style merging, deterministic markers/CSS, encoding/xml SVG structs |
| ✅ | 11. SVG output and end-to-end (`pkg/output/svg/`) | #16 | 88.6% | `svg.Render(r, opts)` wires parser→measure→layout→renderer; header-as-truth for direction; shared ruler; golden fixtures |

## Phase 3: Sequence Diagram

| Status | Step | PR | Coverage | Notes |
|--------|------|----|----|---|
| ✅ | 12. Sequence parser (`pkg/parser/sequence/`) | #18, #19, #20 | 98.8% | 3 slices: A (header, participants, 8 arrows), B (activation markers, notes), C (7 block kinds with nesting) |
| ✅ | 13. Sequence renderer (`pkg/renderer/sequence/`) | #21, #22, #23 | 94.6% | 3 slices: A (participants, lifelines, layout), B (messages, activation bars, auto-number), C (notes, blocks, SVG integration) |

## Phase 4: Pie Chart and Config

| Status | Step | PR | Coverage | Notes |
|--------|------|----|----|---|
| ✅ | 14. Pie parser and renderer | #25 | 91.8% / 94.5% | Parser + renderer + SVG integration in single PR; arc paths, legend, single-slice special case |
| ✅ | 15. Config and themes (`pkg/config/`) | #27, #28 | 100% / 91.2% | JSON config loading, 4 built-in themes, init directive extraction, theme wiring into svg.Render |

## Phase 5: CLI and Output Formats

| Status | Step | PR | Coverage | Notes |
|--------|------|----|----|---|
| ✅ | 16. CLI (`cmd/mmgo/`) | #30 | — | pflag, stdin/stdout, theme/config, SVG/PNG/PDF output |
| ✅ | 17. PNG output (`pkg/output/png/`) | #31 | 88.0% | tdewolff/canvas rasterizer, scale/fixed dims, NearestNeighbor |
| ✅ | 18. PDF output (`pkg/output/pdf/`) | #32 | 88.9% | Vector PDF via canvas renderers.PDF() |
| ✅ | 19. Markdown processing (`pkg/output/markdown/`) | #33 | 92.5% | Rewrite ```mermaid blocks to image refs, shared ConvertSVG |

## Phase 6: Additional Diagram Types

| Status | Step | PR | Coverage | Notes |
|--------|------|----|----|---|
| ✅ | 20. Class diagram | #35, #36, #139–#145 | ~95% / ~95% | Full Mermaid surface: reverse / two-way arrows, `direction`, generics `~T~`, custom labels `["…"]`, static `$` / abstract `*` modifiers, single-line members, inline + bare-line annotations, notes, classDef / style / cssClass / `:::`, accTitle / accDescr / title, lollipop interfaces `()--`, click / link / callback events with SVG `<a>` wrap, `namespace { … }` blocks |
| ✅ | 21. State diagram | #38, #39, #148–#151 | ~95% / ~95% | Full Mermaid surface: `id : description` shorthand, `direction` keyword, multi-line transition labels, notes (`note left/right of`, `end note` block), classDef / style / cssClass / `:::` styling, accTitle / accDescr / title, click / link / callback events with SVG `<a>` wrap, concurrent regions (`--` separator), composite-state bounding boxes |
| ✅ | 22. ER diagram | #40, #153–#155 | ~95% / ~95% | Full Mermaid surface: entity attributes with PK/FK/UK + multi-key (`PK, FK`) and `*name` markers, full cardinality matrix (4×4×2), `direction` keyword, quoted attribute comments, title / accTitle / accDescr, classDef / style / cssClass / `:::` shorthand, click / link / callback events with SVG `<a>` wrap, aliased entity names (`EntityID["Display Label"]`) |
| ✅ | 23. Gantt chart | #157, #158, #159 | ~94% / ~89% | Full Mermaid surface: tag-list status flags (`done` / `active` / `crit` / `milestone` combinable), `after id1 id2 ...` and `until id1 id2 ...` dependencies (forward + backward), full duration unit set (ms / s / m / h / d / w / M / y) with decimals, complete Moment.js `dateFormat` token set, d3-strftime `axisFormat`, calendar-aware `tickInterval`, `weekday` / `excludes` / `includes` / `todayMarker` directives, milestone diamond glyphs, crit stroke emphasis, today-marker rule, `vert` marker lines, `accTitle` / `accDescr`, click / href / call events with SVG `<a>` wrap |
| ✅ | 24. Mindmap | #161, #162, #163 | ~93% / ~92% | Full Mermaid surface: 7 node shapes (default / round / square / circle / cloud / bang / hexagon) including the historical `!text!` bang form, indentation hierarchy with multi-root rejection, `::icon(...)` decoration with rendered caption, multi-class `:::a b c` shorthand, `classDef` / `style id` styling pipeline, accTitle / accDescr (`<title>`/`<desc>` emission), backtick-wrapped markdown labels, `**bold**` / `*italic*` segments, `\n` multi-line labels with per-line measurement and stacked `<text>` rendering |
| ✅ | 25. Timeline | #165, #188 | ~92% / ~92% | Phase 1: accTitle / accDescr, `direction LR\|TD` (parsed), multi-event periods rendered as stacked boxes. Phase 2: LR layout is now the spec default with section bands spanning their column ranges, period time labels on a single row, horizontal axis dot-line, vertical event-box stacks; `direction TD` flips back to the legacy vertical layout. Remaining Phase 3+ follow-ups (`cScale*` / `disableMulticolor` theme variables) tracked separately. |
| ✅ | 26. Sankey | #166, #183 | ~92% / ~94% | PR1: `sankey` alias, frontmatter `title:`, accTitle / accDescr inline with CSV rows. PR2: `LinkColor` (Source / Target / Gradient / Hex) with per-flow `<linearGradient>` defs in Gradient mode, `ShowValues *bool` toggle, `Prefix` / `Suffix` value wrapping, `NodeAlignment` enum stub. PR3: every NodeAlignment mode wired (Justify pins sinks to maxCol; Left = longest-path-from-sources; Right = longest-path-to-sinks i.e. column = maxCol − height; Center averages Left and Right per node). Remaining (width / height config, %%{init}%% for sankey block) tracked as follow-ups. |
| ✅ | 27. Block | #167–#171 | ~93% / ~89% | Full Mermaid surface (modulo `<["..."]>(dir)` block-arrow shape): 13 node shapes including parallelograms / trapezoids / asymmetric / cylinder / subroutine / double-circle / hexagon, `block:ID[:N]["label"] ... end` group nesting with per-group `columns`, `space` / `space:N` spacers, `id:N` width spans, full edge lexicon (`-->`, `---`, `<-->`, `==>`, `-.->`, `~~~`, `--x`, `--o`) with inline `-- text -->` labels, `style` / `classDef` / `class id name` / `id:::name` styling, frontmatter `title:`, `accTitle` / `accDescr` (single + multi-line block), `columns auto` |
| ✅ | 28. Quadrant | #173, #174, #187 | ~92% / ~92% | Phase A + C: per-point `Style`, `classDef` + `:::class`, `accTitle` / `accDescr`, `<title>` / `<desc>`, label-with-colon edge case fix. Phase B: every documented `themeVariables.quadrantChart.*` key plumbed through Theme; full Config covering all 17 layout / typography knobs (chart dims, title font, quadrant border strokes, axis label/title fonts/padding, point radius / label font / text padding); per-quadrant fill + text colors via `Theme.Quadrants[4]`; named `XAxisPosition` / `YAxisPosition` enums with auto-flip when only the bottom or right half carries data; internal vs external border separation. Inner-quadrant padding wiring (#191) tracked as follow-up. |
| 🚧 | 29. C4 | #175, #186 | ~92% / ~91% | Phase 1: 14 new element kinds covering every queue / `_Ext` / `Deployment_Node` / `Node_*` keyword, long-form `Rel_Up`/`Down`/`Left`/`Right`, `BiRel` with `marker-start` + `marker-end`, queue stadium + DB cylinder + dashed `Deployment_Node` shapes, `accTitle` / `accDescr`. Phase 2: `Boundary(...) { ... }` block scoping with stack-based parser (Generic / System / Enterprise / Container kinds), nested boundaries, viewport adjustment for boundary frames extending past layout bounds, `TypeHint` honoured on generic Boundary kind. Phases 3–4 deferred (named-arg `$descr=` / `$tags=` / `$link=` / `$sprite=` / `RelIndex`, `UpdateElementStyle` / `UpdateRelStyle` / `UpdateLayoutConfig`, `LAYOUT_*` directives, legend). |
| ✅ | 30. Kanban | #176, #181 | ~92% / ~92% | Phase 1: frontmatter `title:` + `ticketBaseUrl:`, `accTitle` / `accDescr` (single + block), `<title>` / `<desc>` SVG emission + title caption above columns. Phase 2: priority-driven 4-px left-edge stripe (Very High / High / Low / Very Low palette), `<a href>` ticket-link wrap with `#TICKET#` substitution. Phase 3: card text wrapping switched to `textmeasure.Ruler` so proportional-font glyph widths drive the wrap point. The originally-deferred `classDef` / `:::class` / `%%{init}%%` / `<br/>` / markdown items are not in the Mermaid Kanban spec (https://mermaid.js.org/syntax/kanban.html documents only `kanban` header, columns, `taskId[Text]@{ key: value }` metadata, and `ticketBaseUrl` config) — dropped from the queue. |
| 🚧 | 31. GitGraph | #178, #182 | ~92% / ~91% | Phase 1: `cherry-pick id: "..." [tag] [parent]` with `GitCommitCherryPick` glyph, `switch` alias, `commit msg: "..."`, `branch <name> order: N`, quoted names. Phase 2: frontmatter `title:` + `mainBranchName:`, `accTitle` / `accDescr` (single + block), `<title>` / `<desc>` SVG + title caption. Phase 3: full Config covering every documented `%%{init: gitGraph: …}%%` toggle (`ShowBranches` / `ShowCommitLabel` / `RotateCommitLabel` / `ParallelCommits`, all `*bool` tri-state) plus `MainBranchOrder` lane shifting; commit labels now rotate -45° by default (matching Mermaid's spec default), with `RotateCommitLabel=false` for the legacy horizontal layout; lanes sort by `(BranchOrder asc, declaration index asc)` with `MainBranchOrder` shifting the implicit main branch downward; `Direction` parsed from header (LR/TB/BT) and captured on the AST. Phase 4 deferred (TB / BT renderer wiring — only LR is currently rendered; `parallelCommits` algorithmic pass; named theme resolution, validation). |
| ✅ | 32. XY chart | #179, #189, #197, #200 | ~95% / ~93% | Phase A: stable `xychart` keyword alongside legacy `xychart-beta`, frontmatter `title:`, `accTitle` / `accDescr`, `<title>` / `<desc>` SVG emission. Phase B: every documented `themeVariables.xyChart.*` key (12 surfaces + aggregate fallbacks that rebroadcast); full Config covering width/height/title font + every per-axis surface (showLabel / labelFontSize / labelPadding / showTitle / titleFontSize / titlePadding / showTick / tickLength / tickWidth / showAxisLine / axisLineWidth + showDataLabel / showDataLabelOutsideBar / chartOrientation); horizontal layout with correct AST-aligned axis swap; continuous numeric X-axis (`x-axis title min --> max`); data labels (inside vs outside-bar); `BoolPtr` helper for tri-state Show* flags; tickRow unification across all three axis directions. Phase 4 follow-ups: negative-bar baseline (#190 → #197) makes ranges crossing zero render with valid SVG dimensions; renderSeries vertical/horizontal unified into one orientation-parameterised function (#192 → #200). |

## Milestones

| Milestone | Steps | Status |
|-----------|-------|--------|
| v0.1.0 — Flowchart SVG + Go module API | 0–11 | ✅ Shipped (12/12 done) |
| v0.2.0 — + Sequence, pie, themes | 12–15 | ✅ Shipped (16/16 done) |
| v0.3.0 — + CLI, PNG/PDF, markdown | 16–19 | ✅ Shipped (20/20 done) |
| v0.4.0 — + Class, state, ER | 20–22 | ✅ Shipped (23/23 done) |
| v0.5.0 — + Gantt, mindmap, others | 23–25+ | 🚧 In progress |
| v1.0.0 — Stable API, all major types, >90% coverage | — | ⏳ Not started |

## Legend

- ✅ Done and merged
- 🚧 In progress (branch exists, PR open or pre-PR)
- ⏳ Not started
