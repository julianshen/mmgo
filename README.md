# mmgo

A pure-Go reimplementation of [mermaid-cli](https://github.com/mermaid-js/mermaid-cli). Renders Mermaid diagram definitions to SVG, PNG, and PDF — no Node.js, no headless Chromium, single static binary.

## Why

Upstream `mermaid-cli` requires Puppeteer + a ~300 MB Chromium download. mmgo achieves the same rendering pipeline natively in Go: hand-rolled parsers per diagram type, a Go port of [dagre](https://github.com/dagrejs/dagre) for graph layout, font-metric text measurement via `golang.org/x/image/font`, and direct SVG generation. PNG and PDF go through [tdewolff/canvas](https://github.com/tdewolff/canvas).

The result: one ~15 MB statically-linked binary that renders every Phase 6 Mermaid diagram type at full spec parity.

## Install

```bash
go install github.com/julianshen/mmgo/cmd/mmgo@latest
```

Or build from source:

```bash
git clone https://github.com/julianshen/mmgo
cd mmgo
go build -o mmgo ./cmd/mmgo
```

## CLI usage

```bash
# Markdown / .mmd file → SVG (format inferred from extension)
mmgo -i diagram.mmd -o diagram.svg

# Stdin → stdout
cat diagram.mmd | mmgo -i - -o -

# PNG with custom theme + background
mmgo -i diagram.mmd -o diagram.png -t dark -b transparent

# Markdown rewriter (replace ```mermaid blocks with image refs)
mmgo -i README.md -o README.rendered.md
```

### Flags

| Flag | Short | Default | Description |
|---|---|---|---|
| `--input` | `-i` | — | Input file (`.mmd`, `.md`, or `-` for stdin) |
| `--output` | `-o` | — | Output file (format inferred from extension; defaults to stdout) |
| `--theme` | `-t` | `default` | Mermaid theme: `default`, `dark`, `forest`, `neutral` |
| `--backgroundColor` | `-b` | (theme default) | `transparent`, `white`, `#hex` |
| `--configFile` | `-c` | — | Path to JSON config file |
| `--quiet` | `-q` | false | Suppress non-error output |

## Programmatic use

```go
import (
    "os"
    svg "github.com/julianshen/mmgo/pkg/output/svg"
)

func main() {
    f, _ := os.Open("diagram.mmd")
    defer f.Close()
    out, err := svg.Render(f, nil)
    if err != nil { panic(err) }
    os.Stdout.Write(out)
}
```

For PNG and PDF use `pkg/output/png` and `pkg/output/pdf`.

## Supported diagram types

All diagram types below are at full Mermaid spec parity for the surface Mermaid itself implements. See [docs/STATUS.md](docs/STATUS.md) for per-type detail and any minor follow-ups.

- Flowchart
- Sequence diagram
- Pie chart
- Class diagram
- State diagram
- ER diagram
- Gantt chart
- Mindmap
- Timeline
- Sankey
- Block diagram
- Quadrant chart
- C4 (Context, Container, Component, Dynamic, Deployment)
- Kanban
- GitGraph
- XY chart

## Architecture

```
cmd/mmgo/              CLI — flag parsing, I/O orchestration
pkg/parser/<type>/     Mermaid syntax → AST (one sub-package per diagram)
pkg/diagram/           Shared AST types
pkg/layout/            Go port of dagre (cycle removal → rank → order → position → routing)
pkg/textmeasure/       Font-metric text bounding-box measurement
pkg/renderer/<type>/   AST → SVG (one sub-package per diagram)
pkg/output/{svg,png,pdf,markdown}/   Format converters and file writers
pkg/config/            JSON config loading, themes, init directives
```

The public Go module surface (`pkg/...`) is designed to accept `io.Reader` / `io.Writer`, return errors as values, and never panic. Concurrency stays out of the public API; callers can add it.

## Build & development

```bash
make build         # → ./mmgo
make test          # go test ./... -race -cover
make lint          # golangci-lint run ./...
make cover         # HTML coverage report → coverage.html
```

The repo runs >2200 tests across 52 packages with golden-file snapshot testing for every example in `examples/`. Coverage target is >90% per package.

## Contributing

PRs welcome. The project follows strict TDD (red → green → refactor) and a no-shortcuts working principle: don't disable failing tests, don't silently cut scope, fix root causes rather than symptoms. See [CLAUDE.md](CLAUDE.md) for the full conventions.

## License

MIT — see [LICENSE](LICENSE). The bundled Source Sans Pro font is licensed under the SIL Open Font License.
