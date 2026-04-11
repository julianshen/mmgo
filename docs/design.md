# mmgo Design Document

## 1. Goals

Build a pure-Go Mermaid diagram renderer that:

- Parses Mermaid syntax and renders to SVG, PNG, and PDF
- Ships as a single static binary with zero runtime dependencies (no Node.js, no Chrome)
- Exposes a public Go module (`pkg/`) so other Go programs can render Mermaid diagrams programmatically
- Provides a CLI (`mmdc`) with flag-compatible interface to the original mermaid-cli
- Achieves syntax compatibility with Mermaid.js (accepts the same input files)
- Accepts visual differences from the original renderer (we match syntax, not pixels)

## 2. Non-Goals

- Pixel-perfect rendering parity with Mermaid.js
- Supporting all 26 Mermaid diagram types at launch (phased rollout)
- Interactive/animated SVG output
- Client-side JavaScript rendering
- Wrapping or embedding Mermaid.js in any form

## 3. Architecture Overview

```
┌─────────────────────────────────────────────────────────┐
│                       CLI (cmd/mmdc)                    │
│         Flag parsing, I/O, markdown processing          │
└─────────────────────┬───────────────────────────────────┘
                      │
┌─────────────────────▼───────────────────────────────────┐
│                  pkg/ (public module)                    │
│                                                         │
│  ┌──────────┐  ┌──────────┐  ┌────────────┐            │
│  │  parser/  │  │ diagram/ │  │  config/   │            │
│  │          │  │          │  │            │            │
│  │flowchart │  │ AST types│  │ JSON config│            │
│  │sequence  │  │Diagram IF│  │ themes     │            │
│  │pie       │  │          │  │            │            │
│  │...       │  │          │  │            │            │
│  └────┬─────┘  └────┬─────┘  └─────┬──────┘            │
│       │              │              │                   │
│  ┌────▼──────────────▼──────────────▼──────┐            │
│  │              layout/                     │            │
│  │  ┌────────┐ ┌──────┐ ┌───────┐ ┌─────┐ │            │
│  │  │ graph/ │ │rank/ │ │order/ │ │pos/ │ │            │
│  │  └────────┘ └──────┘ └───────┘ └─────┘ │            │
│  └──────────────────┬──────────────────────┘            │
│                     │                                   │
│  ┌──────────────────▼──────────────────────┐            │
│  │           textmeasure/                   │            │
│  │    golang.org/x/image/font               │            │
│  │    Bundled font (Source Sans Pro)         │            │
│  └──────────────────┬──────────────────────┘            │
│                     │                                   │
│  ┌──────────────────▼──────────────────────┐            │
│  │            renderer/                     │            │
│  │  flowchart/ │ sequence/ │ pie/ │ ...    │            │
│  └──────────────────┬──────────────────────┘            │
│                     │                                   │
│  ┌──────────────────▼──────────────────────┐            │
│  │             output/                      │            │
│  │   svg/ │ png/ │ pdf/ │ markdown/        │            │
│  └─────────────────────────────────────────┘            │
└─────────────────────────────────────────────────────────┘
```

## 4. Component Design

### 4.1 Parser (`pkg/parser/`)

**Approach:** Hand-rolled recursive descent parsers — one sub-package per diagram type.

**Why not a parser generator?**
Mermaid's grammar is informal and ad-hoc. There is no BNF/EBNF specification. Each diagram type has its own syntax with unique lexical rules (e.g., flowchart allows `-->|label|`, sequence uses `->>` and `-->>`, pie uses simple `"label" : value`). A hand-rolled parser gives us:
- Full control over error messages
- Easy incremental addition of diagram types
- No build-time code generation step
- Simpler debugging

**Parser contract:**

```go
// pkg/parser/parser.go

// Parse detects the diagram type from the input text and delegates
// to the appropriate type-specific parser.
func Parse(input string) (diagram.Diagram, error)
```

```go
// pkg/parser/flowchart/parser.go

// Parse parses a flowchart/graph definition into a FlowchartDiagram.
func Parse(input string) (*diagram.FlowchartDiagram, error)
```

**Type detection:** The first non-comment line determines the diagram type:
- `graph LR`, `graph TD`, `flowchart LR`, etc. → flowchart
- `sequenceDiagram` → sequence
- `pie` → pie chart
- `classDiagram` → class diagram
- etc.

**Lexer/Scanner:** Each parser has a simple scanner that tokenizes the input line-by-line. Mermaid syntax is mostly line-oriented (except for subgraphs and blocks which nest). The scanner handles:
- Comment stripping (`%%`)
- Directive extraction (`%%{init: ...}%%`)
- String literals (quoted text in nodes/labels)
- Special character sequences (`-->`, `==>`, `->>`, `-->>`, etc.)

### 4.2 Diagram AST (`pkg/diagram/`)

Defines the shared `Diagram` interface and concrete types per diagram kind:

```go
// pkg/diagram/diagram.go

type DiagramType int

const (
    Flowchart DiagramType = iota
    Sequence
    Pie
    Class
    State
    ER
    Gantt
    // ...
)

// Diagram is implemented by all diagram AST types.
type Diagram interface {
    Type() DiagramType
}
```

**Flowchart AST:**

```go
type FlowchartDiagram struct {
    Direction  Direction  // LR, RL, TD, BT
    Nodes      []Node
    Edges      []Edge
    Subgraphs  []Subgraph
    Styles     []StyleDef
    Classes    map[string]string
}

type Node struct {
    ID    string
    Label string
    Shape NodeShape  // box, round, diamond, hexagon, etc.
    Class string
}

type Edge struct {
    From      string
    To        string
    Label     string
    LineStyle LineStyle  // solid, dotted, thick
    ArrowHead ArrowHead  // arrow, open, cross, circle
}

type Subgraph struct {
    ID       string
    Label    string
    Nodes    []Node
    Edges    []Edge
    Children []Subgraph
}
```

**Sequence AST:**

```go
type SequenceDiagram struct {
    Participants []Participant
    Messages     []Message
    Blocks       []Block  // alt, opt, loop, par, critical, break
    Notes        []Note
    AutoNumber   bool
}

type Participant struct {
    ID    string
    Alias string
    Type  ParticipantType  // participant, actor
}

type Message struct {
    From      string
    To        string
    Label     string
    ArrowType ArrowType  // solid, dashed, solid-open, dashed-open
    Activate  bool
    Deactivate bool
}
```

### 4.3 Layout Engine (`pkg/layout/`)

A Go port of [dagrejs/dagre](https://github.com/dagrejs/dagre), implementing the Sugiyama framework for layered graph drawing.

**Sub-packages mirror dagre's algorithmic phases:**

#### 4.3.1 `layout/graph/` — Graph Data Structure

Replaces dagre's `graphlib` dependency. A simple directed graph with:
- Node/edge storage with arbitrary attributes (label, width, height)
- Adjacency list representation
- Predecessor/successor queries
- Subgraph (compound graph) support

```go
type Graph struct { ... }

func (g *Graph) SetNode(id string, attrs NodeAttrs)
func (g *Graph) SetEdge(from, to string, attrs EdgeAttrs)
func (g *Graph) Nodes() []string
func (g *Graph) Edges() []EdgeID
func (g *Graph) Successors(id string) []string
func (g *Graph) Predecessors(id string) []string
func (g *Graph) NodeAttrs(id string) NodeAttrs
```

#### 4.3.2 `layout/acyclic/` — Cycle Removal

**Algorithm:** Greedy Feedback Arc Set (Eades, Lin, Smyth).

Identifies and reverses edges to make the graph acyclic. Reversed edges are marked so they can be un-reversed after layout.

~150 lines in dagre. Straightforward port.

#### 4.3.3 `layout/rank/` — Rank Assignment

**Algorithm:** Network Simplex (Gansner et al., 1993).

Assigns each node to a rank (layer/row) to minimize total edge length. Three sub-phases:
1. Longest-path initialization — assigns initial ranks
2. Feasible tight tree — builds a spanning tree
3. Iterative optimization — swaps tree/non-tree edges to reduce cost

~255 lines in dagre. Medium complexity — tree data structures and cut value computation.

#### 4.3.4 `layout/order/` — Crossing Minimization

**Algorithm:** Barycenter heuristic with iterative up/down sweeping.

Determines node order within each rank to minimize edge crossings. Runs multiple passes (typically 4), alternating sweep direction. Uses cross-counting to evaluate quality.

~400 lines across 9 files in dagre. Medium complexity — the core logic is straightforward, supporting functions handle conflict resolution.

#### 4.3.5 `layout/position/` — Coordinate Assignment

**Algorithm:** Brandes-Kopf (2002).

Computes final x/y coordinates in O(N) time. The most complex phase:
1. Type-1 conflict detection
2. Vertical alignment (assigns each node to a block)
3. Horizontal compaction (two-pass coordinate assignment)
4. Four-orientation balancing (UL, UR, LL, LR) — takes median

~526 lines in dagre. Hard — requires careful implementation of the paper's algorithm.

#### 4.3.6 `layout/edge/` — Edge Routing

Computes control points for edge paths (straight lines, orthogonal segments, or spline curves). Handles:
- Self-loops
- Multi-edges between same nodes
- Edge labels (positioned at midpoint)

**Top-level layout API:**

```go
// pkg/layout/layout.go

type LayoutOptions struct {
    RankDir    string  // TB, BT, LR, RL
    NodeSep    float64 // horizontal separation between nodes
    RankSep    float64 // vertical separation between ranks
    EdgeSep    float64 // separation between edges
    MarginX    float64
    MarginY    float64
}

// Layout computes positions for all nodes and edges in the graph.
func Layout(g *graph.Graph, opts LayoutOptions) *LayoutResult

type LayoutResult struct {
    Nodes map[string]NodeLayout  // id → {x, y, width, height}
    Edges map[EdgeID]EdgeLayout  // edge → {points []Point, labelPos Point}
    Width  float64
    Height float64
}
```

### 4.4 Text Measurement (`pkg/textmeasure/`)

Provides text bounding box computation using `golang.org/x/image/font`. This replaces the browser's `SVGTextElement.getBBox()` that Mermaid.js depends on.

```go
type Ruler struct { ... }

func NewRuler(fontData []byte, defaultSize float64) (*Ruler, error)

// Measure returns the width and height of the given text in pixels
// at the specified font size.
func (r *Ruler) Measure(text string, fontSize float64) (width, height float64)
```

**Font bundling:** A default font (e.g., Source Sans Pro, licensed under OFL) is embedded via `//go:embed`. Users can override with `--cssFile` to specify custom fonts.

**Why not just estimate?** Layout quality depends on accurate text measurement. A 10% error in text width cascades through the layout engine, causing overlapping labels or excessive whitespace. Actual font metrics are necessary.

### 4.5 Renderer (`pkg/renderer/`)

Each diagram type has a dedicated renderer that transforms a positioned layout into SVG elements.

```go
// pkg/renderer/renderer.go

// Renderer converts a diagram with computed layout into SVG.
type Renderer interface {
    Render(d diagram.Diagram, layout *layout.LayoutResult, opts RenderOptions) (string, error)
}
```

**Flowchart renderer** generates:
- `<rect>`, `<polygon>`, `<circle>` for node shapes
- `<path>` with marker-end for edges (arrows)
- `<text>` for labels
- `<g>` groups for subgraphs with background rectangles
- CSS classes for styling/theming

**Sequence renderer** generates:
- Vertical `<line>` for lifelines
- `<rect>` for participant boxes and activation bars
- `<line>`/`<path>` for messages with arrow markers
- `<rect>` with `<text>` for block labels (alt, loop, etc.)
- `<rect>` for notes

SVG is generated via string building (not a DOM library). Each renderer produces a complete `<svg>` document with viewBox, styles, and defs.

### 4.6 Output (`pkg/output/`)

**SVG:** Native output format — the renderer already produces SVG strings.

**PNG:** Rasterize SVG to PNG using a pure-Go approach:
- Option A: Parse SVG and rasterize with `golang.org/x/image` + vector rasterizer
- Option B: Use a Go SVG rasterizer library (e.g., `srwiley/oksvg` + `srwiley/rasterx`)
- The `--scale` flag controls device pixel ratio (DPR)

**PDF:** Embed SVG or rasterized image in a PDF page:
- Use `go-pdf/fpdf` or `jung-kurt/gofpdf`
- `--pdfFit` scales the diagram to fit the page

**Markdown rewriter:** Processes .md files, finds mermaid code blocks, renders each to a separate image file, and replaces the code block with an image reference (`![](diagram-1.svg)`).

### 4.7 Config (`pkg/config/`)

Loads JSON configuration matching mermaid-cli's format:

```json
{
  "theme": "default",
  "themeVariables": {
    "primaryColor": "#326ce5"
  },
  "flowchart": {
    "curve": "basis",
    "padding": 15
  },
  "sequence": {
    "actorMargin": 50,
    "messageMargin": 35
  }
}
```

Theme determines default colors for node fills, strokes, text, and backgrounds. Four built-in themes: `default`, `dark`, `forest`, `neutral`.

### 4.8 CLI (`cmd/mmdc/`)

Thin orchestrator that wires together the packages:

1. Parse CLI flags
2. Load config file (if specified)
3. Read input (file, stdin, or markdown)
4. For each diagram:
   a. Parse → AST
   b. Measure text (for node sizing)
   c. Layout → positioned graph
   d. Render → SVG
   e. Convert to output format (SVG/PNG/PDF)
5. Write output (file or markdown rewrite)

## 5. Dependencies

| Dependency | Purpose | Justification |
|-----------|---------|---------------|
| `golang.org/x/image/font` | Text measurement (font metrics) | No stdlib alternative for font metric computation |
| `golang.org/x/image/math/fixed` | Fixed-point math for font metrics | Required by x/image/font |
| `srwiley/oksvg` + `srwiley/rasterx` | SVG → PNG rasterization | Pure Go SVG rasterizer; avoids CGO |
| Standard library only | Everything else | Minimize dependency tree |

CLI flag parsing: Use stdlib `flag` package or evaluate `spf13/pflag` for POSIX-style `--long-flag` support (mermaid-cli uses POSIX flags).

## 6. Error Handling Strategy

- **Parser errors:** Return line number, column, and descriptive message. Support multiple errors per parse (collect and report all, not just first).
- **Layout errors:** Should not fail on valid ASTs. Panic indicates a bug in the layout engine.
- **Render errors:** Unlikely on valid layouts. Return error for unsupported features.
- **I/O errors:** Wrap with `fmt.Errorf("reading input %s: %w", path, err)` for context.

## 7. Testing Strategy

- **Parsers:** Table-driven tests with input strings and expected AST structures. Golden files for complex diagrams. Fuzz testing for robustness.
- **Layout engine:** Validate coordinates against dagre.js output on identical graph inputs. Property-based tests (e.g., no node overlaps, edges connect correct nodes, ranks are monotonic).
- **Text measurement:** Compare measurements against known font metric values from font inspection tools.
- **Renderers:** Golden file SVG comparisons. Visual regression tests (render → rasterize → compare images).
- **CLI:** Integration tests that run the binary end-to-end on fixture .mmd files and verify output.
- **Coverage target:** >90% across all packages.

## 8. Risks and Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| Mermaid syntax edge cases not documented | Parser rejects valid input | Test against real-world .mmd files from public repos; fuzz testing |
| Text measurement inaccuracy | Layout has overlapping labels | Bundle a specific font and test against known metrics; allow user font override |
| Dagre port introduces layout bugs | Incorrect diagram rendering | Golden-file tests comparing Go output vs dagre.js output on same inputs |
| SVG rasterization quality for PNG | Blurry or broken PNG output | Evaluate multiple Go rasterizers; fall back to external tool if needed |
| Scope creep across 26 diagram types | Never ships | Strict phased rollout; ship Phase 1 (3 types) as v0.1.0 |
