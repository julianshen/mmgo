# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**mmgo** is a Go reimplementation of [mermaid-cli](https://github.com/mermaid-js/mermaid-cli) — a tool that renders Mermaid diagram definitions into SVG, PNG, and PDF output. It provides both a **public Go module** (`pkg/`) for programmatic use and a **CLI** (`cmd/mmdc/`).

The rendering pipeline uses headless Chromium (via chromedp) to execute the Mermaid JavaScript library and capture output — mirroring the original's Puppeteer-based approach.

## Build & Development Commands

```bash
# Build CLI
go build ./cmd/mmdc

# Run all tests with coverage
go test ./... -cover -race

# Run a single test
go test ./pkg/renderer -run TestRenderSVG -v

# Run tests for a specific package
go test ./pkg/renderer/... -v

# Generate coverage report
go test ./... -coverprofile=coverage.out && go tool cover -html=coverage.out

# Lint (requires golangci-lint installed)
golangci-lint run ./...

# Format
gofmt -w .
```

## Architecture

```
cmd/mmdc/          CLI entry point — flag parsing, I/O orchestration
pkg/
  renderer/        Core rendering engine — manages headless Chromium, executes
                   Mermaid JS, captures SVG/PNG/PDF output
  parser/          Input parsing — reads .mmd files, extracts mermaid blocks
                   from markdown, handles stdin
  config/          Configuration loading — JSON config files, theme settings,
                   CSS injection, Puppeteer-equivalent browser options
  output/          Output writers — file writing, format conversion,
                   markdown rewriting with image references
```

**Key data flow:** Input (.mmd/markdown/stdin) → Parser → Renderer (chromedp + mermaid.js) → Output (SVG/PNG/PDF/rewritten markdown)

### Public API Design

The `pkg/` packages are the public module surface. Follow these principles:
- Accept interfaces, return concrete types (e.g., accept `io.Reader`/`io.Writer`, not `*os.File`)
- Keep interfaces small — one or two methods
- Expose synchronous APIs; let callers add concurrency
- Return errors as values; never panic in library code

### Renderer Architecture

The renderer embeds the Mermaid JS library and uses chromedp to:
1. Launch a headless Chromium context
2. Load an HTML page with the Mermaid library
3. Inject the diagram definition and configuration
4. Wait for Mermaid to render the SVG in the DOM
5. Capture output (SVG text, PNG screenshot, or PDF print)

The Chromium context should be reusable across multiple renders (batch mode).

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
- **Dependencies:** Prefer the standard library. External deps require justification (chromedp is the notable exception).
- **Concurrency:** Keep it out of the public API. Use channels internally where needed; use quit channels to prevent goroutine leaks.
- **`defer`:** Use for resource cleanup (Chromium contexts, file handles, temp dirs).

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
- Table-driven tests for flag parsing and config loading.
- Integration tests for the renderer require Chrome/Chromium installed — guard with `testing.Short()` to skip in CI-light environments.
- Use `t.TempDir()` for test output files.
- Mock the browser interface for unit-testing renderer logic without Chromium.
