# Flowchart Phase 1 — Parser Gaps Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement the 8 parser-gap features for flowchart where AST and renderer scaffolding already exist.

**Architecture:** All changes are in the parser (`pkg/parser/flowchart/parser.go`) and AST (`pkg/diagram/flowchart.go`). The renderer already supports these features — we just need the parser to produce the right AST output. Each feature is independent and can be implemented as a separate task.

**Tech Stack:** Go, standard library only. TDD discipline.

---

## File Structure

- **Modify:** `pkg/diagram/flowchart.go` — Add `ArrowTail`, `ID` fields to Edge; add `LineStyleInvisible`; add `Direction` to Subgraph
- **Modify:** `pkg/parser/flowchart/parser.go` — All new parsing logic
- **Modify:** `pkg/parser/flowchart/parser_test.go` — Tests for all new features
- **Modify:** `pkg/renderer/flowchart/edges.go` — MarkerStart support for bidirectional arrows
- **Modify:** `pkg/renderer/flowchart/renderer.go` — Wire up class/style/linkStyle rendering

---

## Chunk 1: AST + Edge Foundation

### Task 1: Add missing AST fields

**Files:**
- Modify: `pkg/diagram/flowchart.go`

- [ ] Add `ArrowTail ArrowHead` and `ID string` fields to `Edge` struct
- [ ] Add `LineStyleInvisible LineStyle` constant
- [ ] Add `Direction Direction` field to `Subgraph` struct
- [ ] Add `LinkStyles map[int]string` field to `FlowchartDiagram` for linkStyle by ordinal

### Task 2: Invisible edge parsing + rendering

**Files:**
- Modify: `pkg/parser/flowchart/parser.go`
- Modify: `pkg/parser/flowchart/parser_test.go`
- Modify: `pkg/renderer/flowchart/edges.go`

- [ ] Add `LineStyleInvisible` case to `edgeStyle()` returning `stroke:none` or very low opacity
- [ ] Add `~~~` pattern to `matchArrowAt` producing `LineStyleInvisible` + `ArrowHeadNone`
- [ ] Add test: `TestParseInvisibleEdge` — `A ~~~ B` produces invisible edge

### Task 3: Circle and cross arrow endpoints

**Files:**
- Modify: `pkg/parser/flowchart/parser.go`
- Modify: `pkg/parser/flowchart/parser_test.go`

- [ ] Extend `matchDashAt` to check for `o` and `x` after dashes as arrow heads
- [ ] `-->o` / `---o` → `ArrowHeadCircle`
- [ ] `-->x` / `---x` → `ArrowHeadCross`
- [ ] Same for thick (`==>x`, `==>o`) and dotted (`-.->o`, `-.->x`)
- [ ] Add tests for each combination

### Task 4: Bidirectional arrows (ArrowTail)

**Files:**
- Modify: `pkg/parser/flowchart/parser.go`
- Modify: `pkg/parser/flowchart/parser_test.go`
- Modify: `pkg/renderer/flowchart/edges.go`

- [ ] Extend `findArrow` to detect `<--` at start of line
- [ ] Parse `<-->`, `<--`, `<---` etc. setting both `ArrowHead` and `ArrowTail`
- [ ] Also `x--x`, `o--o` patterns
- [ ] In `renderEdge`, set `MarkerStart` when `ArrowTail != ArrowHeadNone`
- [ ] Add tests for each bidirectional pattern

---

## Chunk 2: Subgraphs and Styling

### Task 5: Subgraph parsing

**Files:**
- Modify: `pkg/parser/flowchart/parser.go`
- Modify: `pkg/parser/flowchart/parser_test.go`

- [ ] Add subgraph stack to `parser` struct
- [ ] Parse `subgraph title` / `subgraph id [title]` lines
- [ ] Parse `direction TB|LR|...` inside subgraphs
- [ ] Parse `end` keyword to close current subgraph
- [ ] Route nodes/edges to current subgraph when inside one
- [ ] Support edges to/from subgraph IDs
- [ ] Add tests: basic subgraph, nested subgraph, subgraph with direction, edges to subgraph

### Task 6: style / classDef / class directives

**Files:**
- Modify: `pkg/parser/flowchart/parser.go`
- Modify: `pkg/parser/flowchart/parser_test.go`

- [ ] Parse `style nodeId fill:#f9f,stroke:#333` → append to `diagram.Styles`
- [ ] Parse `classDef className fill:#f9f,stroke:#333` → add to `diagram.Classes`
- [ ] Parse `class nodeId1,nodeId2 className` → append to node's `Classes` field
- [ ] Parse `:::` inline class operator in node definitions
- [ ] Add tests for each directive type

### Task 7: linkStyle by ordinal

**Files:**
- Modify: `pkg/parser/flowchart/parser.go`
- Modify: `pkg/parser/flowchart/parser_test.go`
- Modify: `pkg/renderer/flowchart/edges.go`

- [ ] Parse `linkStyle 3 stroke:#ff3,stroke-width:4px` → store in `diagram.LinkStyles`
- [ ] Apply linkStyle overrides during edge rendering based on edge ordinal
- [ ] Support `linkStyle 1,2,7 color:blue` comma-separated indices
- [ ] Add tests

---

## Chunk 3: Remaining Parser Features

### Task 8: Ampersand branching

**Files:**
- Modify: `pkg/parser/flowchart/parser.go`
- Modify: `pkg/parser/flowchart/parser_test.go`

- [ ] Extend `parseLine` to recognize `&` as a node separator after arrows
- [ ] `A --> B & C` creates two edges: A→B and A→C
- [ ] Support chaining: `A --> B & C --> D` (B and C both point to D)
- [ ] Add tests

### Task 9: Edge IDs

**Files:**
- Modify: `pkg/parser/flowchart/parser.go`
- Modify: `pkg/parser/flowchart/parser_test.go`

- [ ] Parse `e1@-->` prefix on arrows → set `Edge.ID` field
- [ ] Support with all arrow types: `e1@-->`, `e1@-.->`, `e1@==>`, etc.
- [ ] Add tests

### Task 10: Dotted inline labels

**Files:**
- Modify: `pkg/parser/flowchart/parser.go`
- Modify: `pkg/parser/flowchart/parser_test.go`

- [ ] Extend `matchDottedAt` to handle `-. label .->` pattern
- [ ] Pattern: `-.` + whitespace + label + whitespace + `.` + dashes + optional `>`
- [ ] Add tests

### Task 11: Quote stripping for quoted labels

**Files:**
- Modify: `pkg/parser/flowchart/parser.go`
- Modify: `pkg/parser/flowchart/parser_test.go`

- [ ] When node label or edge label is surrounded by `"`, strip the quotes
- [ ] Add test: `A["label with spaces"]` → label is `label with spaces` (no quotes)
- [ ] Add test: `A -->|"pipe label"| B` → label stripped if quoted

---

## Verification

- [ ] Run `go test ./pkg/parser/flowchart/ -v -race` — all tests pass
- [ ] Run `go test ./pkg/renderer/flowchart/ -v -race` — all tests pass
- [ ] Run `go test ./... -race` — full suite passes
- [ ] Run `golangci-lint run ./...` — clean
- [ ] Regenerate examples: `go build -o /tmp/mmgo ./cmd/mmgo && for f in examples/flowchart/*.mmd; do base="${f%.mmd}"; /tmp/mmgo -i "$f" -o "${base}.svg" -q; /tmp/mmgo -i "$f" -o "${base}.png" -q; done`
