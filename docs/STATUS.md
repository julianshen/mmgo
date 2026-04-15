# mmgo Implementation Status

Dashboard view of progress against [implementation-plan.md](implementation-plan.md).
Updated at each step boundary and committed to git, so any client can read
current state without relying on chat history.

**Last updated:** 2026-04-15

## Overall

- **Current milestone:** working toward v0.1.0 (Flowchart SVG rendering)
- **Current phase:** Phase 2 (Flowchart) — renderer done, SVG output next
- **Completed:** 11 of 25 steps
- **Next:** Step 11 — SVG output and end-to-end

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
| ⏳ | 11. SVG output and end-to-end | — |  |  |

## Phase 3: Sequence Diagram

| Status | Step | PR |
|--------|------|----|
| ⏳ | 12. Sequence parser (`pkg/parser/sequence/`) | — |
| ⏳ | 13. Sequence renderer (`pkg/renderer/sequence/`) | — |

## Phase 4: Pie Chart and Config

| Status | Step | PR |
|--------|------|----|
| ⏳ | 14. Pie parser and renderer | — |
| ⏳ | 15. Config and themes | — |

## Phase 5: CLI and Output Formats

| Status | Step | PR |
|--------|------|----|
| ⏳ | 16. CLI (`cmd/mmdc/`) | — |
| ⏳ | 17. PNG output | — |
| ⏳ | 18. PDF output | — |
| ⏳ | 19. Markdown processing | — |

## Phase 6: Additional Diagram Types

| Status | Step | PR |
|--------|------|----|
| ⏳ | 20. Class diagram | — |
| ⏳ | 21. State diagram | — |
| ⏳ | 22. ER diagram | — |
| ⏳ | 23. Gantt chart | — |
| ⏳ | 24. Mindmap | — |
| ⏳ | 25+. Remaining types (GitGraph, Timeline, Sankey, XY, C4, Quadrant, Kanban, Block) | — |

## Milestones

| Milestone | Steps | Status |
|-----------|-------|--------|
| v0.1.0 — Flowchart SVG + Go module API | 0–11 | 🚧 Phase 2 in progress (11/12 done) |
| v0.2.0 — + Sequence, pie, themes | 12–15 | ⏳ Not started |
| v0.3.0 — + CLI, PNG/PDF, markdown | 16–19 | ⏳ Not started |
| v0.4.0 — + Class, state, ER | 20–22 | ⏳ Not started |
| v0.5.0 — + Gantt, mindmap, others | 23–25+ | ⏳ Not started |
| v1.0.0 — Stable API, all major types, >90% coverage | — | ⏳ Not started |

## Legend

- ✅ Done and merged
- 🚧 In progress (branch exists, PR open or pre-PR)
- ⏳ Not started
