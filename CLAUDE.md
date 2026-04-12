# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**mmgo** is a pure-Go reimplementation of [mermaid-cli](https://github.com/mermaid-js/mermaid-cli) — a tool that renders Mermaid diagram definitions into SVG, PNG, and PDF output. It provides both a **public Go module** (`pkg/`) for programmatic use and a **CLI** (`cmd/mmdc/`).

Unlike the original (which requires Node.js + headless Chromium), mmgo compiles to a **single static binary with zero runtime dependencies**. It achieves this through a native Go rendering pipeline: hand-rolled Mermaid parser, a Go port of dagre for graph layout, font-based text measurement, and direct SVG generation.

## Build & Development Commands

```bash
# Build CLI
go build ./cmd/mmdc

# Run all tests with coverage
go test ./... -cover -race

# Run a single test
go test ./pkg/parser/flowchart -run TestParseSubgraph -v

# Run tests for a specific package
go test ./pkg/layout/... -v

# Generate coverage report
go test ./... -coverprofile=coverage.out && go tool cover -html=coverage.out

# Lint (requires golangci-lint installed)
golangci-lint run ./...

# Format
gofmt -w .
```

## Architecture

```text
cmd/mmdc/              CLI entry point — flag parsing, I/O orchestration

pkg/
  parser/              Mermaid syntax parsers (one sub-package per diagram type)
    flowchart/         Flowchart/graph parser → AST
    sequence/          Sequence diagram parser → AST
    pie/               Pie chart parser → AST
    ...                (additional diagram types added per phase)

  diagram/             Diagram AST types and interfaces shared across parsers
                       Defines the common Diagram interface and per-type structs

  layout/              Graph layout engine (Go port of dagre)
    graph/             Graph data structure (nodes, edges, adjacency) — public
    internal/          Algorithmic internals (not part of public API)
      acyclic/         Cycle removal — greedy feedback arc set
      rank/            Rank assignment — network simplex algorithm
      order/           Crossing minimization — barycenter heuristic
      position/        Coordinate assignment — Brandes-Kopf algorithm
      edge/            Edge routing — orthogonal polyline and spline paths

  textmeasure/         Font metrics and text bounding box measurement
                       Uses golang.org/x/image/font with bundled fonts

  renderer/            Diagram-type-specific SVG renderers
    flowchart/         Renders flowchart nodes, edges, subgraphs to SVG
    sequence/          Renders lifelines, messages, activation boxes to SVG
    pie/               Renders pie slices, labels to SVG
    ...

  output/              Output format converters and file writers
    svg/               SVG output (native)
    png/               SVG-to-PNG rasterization (pure Go)
    pdf/               SVG/diagram-to-PDF conversion
    markdown/          Markdown rewriter — replaces mermaid blocks with images

  config/              Configuration loading — JSON config, themes, CSS
```

**Key data flow:**

```text
Input (.mmd/markdown/stdin)
  → parser (text → diagram AST)
    → text measurer (computes node sizes from font metrics)
      → layout engine (sized nodes → positioned graph with coordinates)
        → renderer (positioned graph → SVG elements)
          → output (SVG/PNG/PDF/rewritten markdown)
```

### Public API Design

The `pkg/` packages are the public module surface. Follow these principles:
- Accept interfaces, return concrete types (e.g., accept `io.Reader`/`io.Writer`, not `*os.File`)
- Keep interfaces small — one or two methods
- Expose synchronous APIs; let callers add concurrency
- Return errors as values; never panic in library code

### Key Design Decisions

1. **No browser dependency.** The original mermaid-cli requires Puppeteer + headless Chromium (~300MB). mmgo achieves rendering through native Go code: parsing, layout algorithms, text measurement via font metrics, and direct SVG string generation.

2. **Hand-rolled parsers.** Mermaid's grammar is ad-hoc (JISON-based, migrating to Langium). No formal BNF exists. We implement recursive descent parsers per diagram type, tested against the Mermaid syntax docs and real-world .mmd files.

3. **Dagre port for layout.** The layout engine is a Go port of [dagrejs/dagre](https://github.com/dagrejs/dagre) (~3,500 lines of TypeScript). It implements the Sugiyama method: cycle removal → rank assignment (network simplex) → crossing minimization (barycenter) → coordinate assignment (Brandes-Kopf). Reference papers: Gansner et al. 1993, Brandes & Kopf 2002.

4. **Bundled fonts for text measurement.** We bundle a default font (Source Sans Pro, SIL Open Font License) and use `golang.org/x/image/font` to compute text bounding boxes. This replaces the browser's `getBBox()` API that Mermaid.js relies on. The OFL license text must be included alongside the font files in the repository.

5. **Phased diagram support.** Not all 26 Mermaid diagram types ship at once. Phase 1 targets flowchart + sequence + pie (~70% of real-world usage).

## Development Workflow

### Branching & PRs

- **Never commit directly to `main`.** All work happens on feature branches.
- Branch naming: `feature/<name>`, `fix/<name>`, `refactor/<name>`
- Every merge to `main` requires a PR with code review.
- PRs must pass all tests and maintain >90% coverage before merge.

### TDD Discipline (Red → Green → Refactor)

1. **Red:** Write a failing test that defines the expected behavior.
2. **Green:** Write the minimum code to make the test pass.
3. **Refactor:** Clean up while keeping tests green.

Do not write implementation code without a corresponding test first. Test files live alongside source files (`foo.go` / `foo_test.go`).

### Coverage

Target: **>90% line coverage** across all packages. Check before every PR:
```bash
go test ./... -coverprofile=coverage.out
go tool cover -func=coverage.out | grep total
```

## Go Conventions for This Project

- **Error handling:** Return errors early; let the happy path flow down. Use `fmt.Errorf("context: %w", err)` to wrap errors with context.
- **Naming:** Exported identifiers use MixedCaps. Packages are single lowercase words. No `Get` prefix on getters.
- **Dependencies:** Prefer the standard library. External deps require justification. Key allowed deps: `golang.org/x/image/font` (text measurement), `tdewolff/canvas` (PNG/PDF rendering), `spf13/pflag` (POSIX CLI flags).
- **Concurrency:** Keep it out of the public API. Use channels internally where needed; use quit channels to prevent goroutine leaks.
- **`defer`:** Use for resource cleanup (file handles, temp dirs).

## CLI Flags (target parity with mermaid-cli)

```
-i, --input         Input file (.mmd, .md, or - for stdin)
-o, --output        Output file (format inferred from extension)
-t, --theme         Mermaid theme (default, dark, forest, neutral)
-b, --backgroundColor  Background color (e.g., transparent, white, #hex)
-c, --configFile    Path to JSON config file
    --cssFile       Path to custom CSS file
-w, --width         Output width in pixels (PNG/PDF)
-H, --height        Output height in pixels (PNG/PDF)
-s, --scale         Device scale factor for PNG (default: 1)
    --pdfFit        Fit diagram to PDF page
-q, --quiet         Suppress log output
```

## Testing Patterns

- Use `testdata/` directories for fixture .mmd files and expected outputs.
- Table-driven tests for parsers and config loading.
- Golden file tests for SVG output — compare rendered SVG against known-good snapshots.
- Use `t.TempDir()` for test output files.
- For layout engine: validate node coordinates against dagre.js output on identical graphs.
- For text measurement: test against known font metric values.
