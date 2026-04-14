# Flowchart Renderer Design

Step 10 of the implementation plan: `pkg/renderer/flowchart/` — converts a parsed, laid-out flowchart into a complete SVG document.

## Context

The renderer sits between the layout engine output (`layout.Result` with positioned nodes/edges) and the final SVG output. It consumes:

- **Diagram AST** (`diagram.FlowchartDiagram`) — node shapes, labels, edge styles, subgraphs, style/class definitions
- **Layout result** (`layout.Result`) — node centers/dimensions, edge polylines, bounding box
- **Options** — theme colors, font size, padding, CSS overrides (no dependency on empty `pkg/config/`)

## Public API

```go
package flowchart

func Render(d *diagram.FlowchartDiagram, l *layout.Result, opts *Options) ([]byte, error)
```

### Options

```go
type Options struct {
    FontSize   float64  // default 16
    Padding    float64  // SVG padding around diagram, default 20
    Theme      Theme    // colors; zero value uses DefaultTheme()
    CSSFile    string   // optional raw CSS to embed in <style>
    Background string   // background color override (e.g. "transparent", "#fff")
}

type Theme struct {
    NodeFill       string  // default "#fff"
    NodeStroke     string  // default "#333"
    NodeText       string  // default "#333"
    EdgeStroke     string  // default "#333"
    EdgeText       string  // default "#333"
    SubgraphFill   string  // default "#eee"
    SubgraphStroke string  // default "#999"
    SubgraphText   string  // default "#333"
    Background     string  // default "#fff"
}
```

`DefaultTheme()` returns the Mermaid "default" theme colors. Future themes (dark, forest, neutral) will be added when `pkg/config/` is built in Step 15.

## SVG Generation with encoding/xml

Each SVG element is a Go struct with `xml:` tags, marshaled via `encoding/xml.Marshal`. This guarantees well-formed XML and provides type safety.

### SVG Element Structs (`svg.go`)

```go
type SVG struct {
    XMLName    xml.Name `xml:"svg"`
    XMLNS      string   `xml:"xmlns,attr"`
    ViewBox    string   `xml:"viewBox,attr"`
    Width      string   `xml:"width,attr,omitempty"`
    Height     string   `xml:"height,attr,omitempty"`
    Defs       *Defs    `xml:"defs,omitempty"`
    Style      string   `xml:"style,omitempty"`
    Groups     []Group  `xml:"g,omitempty"`
    Elements   []any    `xml:",any"`  // mixed content: rect, circle, polygon, path, text, line
}

type Group struct {
    XMLName   xml.Name `xml:"g"`
    ID        string   `xml:"id,attr,omitempty"`
    Class     string   `xml:"class,attr,omitempty"`
    Style     string   `xml:"style,attr,omitempty"`
    Transform string   `xml:"transform,attr,omitempty"`
    Children  []any    `xml:",any"`
}

type Rect struct {
    XMLName xml.Name `xml:"rect"`
    X       float64  `xml:"x,attr"`
    Y       float64  `xml:"y,attr"`
    Width   float64  `xml:"width,attr"`
    Height  float64  `xml:"height,attr"`
    RX      float64  `xml:"rx,attr,omitempty"`
    RY      float64  `xml:"ry,attr,omitempty"`
    Style   string   `xml:"style,attr,omitempty"`
    Class   string   `xml:"class,attr,omitempty"`
}

type Circle struct {
    XMLName xml.Name `xml:"circle"`
    CX      float64  `xml:"cx,attr"`
    CY      float64  `xml:"cy,attr"`
    R       float64  `xml:"r,attr"`
    Style   string   `xml:"style,attr,omitempty"`
    Class   string   `xml:"class,attr,omitempty"`
}

type Polygon struct {
    XMLName xml.Name `xml:"polygon"`
    Points  string   `xml:"points,attr"`
    Style   string   `xml:"style,attr,omitempty"`
    Class   string   `xml:"class,attr,omitempty"`
}

type Path struct {
    XMLName xml.Name `xml:"path"`
    D       string   `xml:"d,attr"`
    Style   string   `xml:"style,attr,omitempty"`
    Class   string   `xml:"class,attr,omitempty"`
    MarkerEnd   string `xml:"marker-end,attr,omitempty"`
    MarkerStart string `xml:"marker-start,attr,omitempty"`
}

type Line struct {
    XMLName xml.Name `xml:"line"`
    X1      float64  `xml:"x1,attr"`
    Y1      float64  `xml:"y1,attr"`
    X2      float64  `xml:"x2,attr"`
    Y2      float64  `xml:"y2,attr"`
    Style   string   `xml:"style,attr,omitempty"`
    Class   string   `xml:"class,attr,omitempty"`
    MarkerEnd   string `xml:"marker-end,attr,omitempty"`
    MarkerStart string `xml:"marker-start,attr,omitempty"`
}

type Text struct {
    XMLName   xml.Name `xml:"text"`
    X         float64  `xml:"x,attr"`
    Y         float64  `xml:"y,attr"`
    Anchor    string   `xml:"text-anchor,attr,omitempty"`
    Dominant  string   `xml:"dominant-baseline,attr,omitempty"`
    FontSize  float64  `xml:"font-size,attr,omitempty"`
    FontFamily string `xml:"font-family,attr,omitempty"`
    Style     string   `xml:"style,attr,omitempty"`
    Class     string   `xml:"class,attr,omitempty"`
    Content   string   `xml:",chardata"`
}

type Defs struct {
    XMLName  xml.Name  `xml:"defs"`
    Markers  []Marker  `xml:"marker,omitempty"`
}

type Marker struct {
    XMLName   xml.Name `xml:"marker"`
    ID        string   `xml:"id,attr"`
    ViewBox   string   `xml:"viewBox,attr"`
    RefX      float64  `xml:"refX,attr"`
    RefY      float64  `xml:"refY,attr"`
    Width     float64  `xml:"markerWidth,attr"`
    Height    float64  `xml:"markerHeight,attr"`
    Orient    string   `xml:"orient,attr"`
    Children  []any    `xml:",any"`
}
```

## Node Shape Rendering (`nodes.go`)

Each `NodeShape` maps to a specific SVG element. Node center comes from `NodeLayout.X/Y`, dimensions from `NodeLayout.Width/Height`.

| NodeShape | SVG Element | Construction |
|-----------|-------------|--------------|
| `NodeShapeRectangle` | `<rect>` | x=cx-w/2, y=cy-h/2 |
| `NodeShapeRoundedRectangle` | `<rect rx="5">` | same as rect with rx=5 |
| `NodeShapeStadium` | `<rect rx="h/2">` | fully rounded ends |
| `NodeShapeDiamond` | `<polygon>` | 4 points: top, right, bottom, left |
| `NodeShapeHexagon` | `<polygon>` | 6 points with ~15% indent |
| `NodeShapeCircle` | `<circle>` | r = min(w,h)/2 |
| `NodeShapeDoubleCircle` | `<circle>` + `<circle>` | inner r=min(w,h)/2, outer r=inner+3 |
| `NodeShapeParallelogram` | `<polygon>` | 4 points with ~15% skew |
| `NodeShapeParallelogramAlt` | `<polygon>` | reverse skew |
| `NodeShapeTrapezoid` | `<polygon>` | wider bottom |
| `NodeShapeTrapezoidAlt` | `<polygon>` | wider top |
| `NodeShapeCylinder` | `<path>` | rect body + 2 elliptical arcs |
| `NodeShapeSubroutine` | `<rect>` + 2 inner vertical lines | rect with double side bands |
| `NodeShapeAsymmetric` | `<polygon>` | one-sided skew |

Text is centered in each node using `dominant-baseline="central"` and `text-anchor="middle"` at (cx, cy).

For multi-line labels, each line gets its own `<text>` element with vertical offset = `lineIndex * lineHeight`.

## Edge Rendering (`edges.go`)

### Path Construction

- **2 points** (straight line): `<line>` element from points[0] to points[1]
- **3+ points** (polyline/curve): `<path>` with cubic bezier segments. Convert the polyline through catmull-rom to bezier conversion for smooth curves.

### Arrow Markers

Defined in `<defs>` as `<marker>` elements. Each `ArrowHead` type gets a marker:

| ArrowHead | Marker Content |
|-----------|---------------|
| `ArrowHeadArrow` | Filled triangle `<polygon>` |
| `ArrowHeadOpen` | Open `<polyline>` (two lines forming a V) |
| `ArrowHeadCross` | Cross lines `<polyline>` |
| `ArrowHeadCircle` | `<circle>` |
| `ArrowHeadNone` | No marker |

Markers are referenced via `marker-end="url(#arrow-arrow)"` and `marker-start` attributes. Marker IDs are deterministic based on arrow head type + line style combination.

### Line Styles

Applied via `style` attribute:
- `LineStyleSolid`: default (no dash)
- `LineStyleDotted`: `stroke-dasharray:2,2`
- `LineStyleThick`: `stroke-width:3`

### Edge Labels

Rendered as `<text>` at `EdgeLayout.LabelPos` with a white background `<rect>` behind the text for readability.

## Subgraph Rendering (`subgraphs.go`)

Each subgraph becomes a `<g>` group containing:
1. Background `<rect>` covering the bounding box of all child nodes (with padding)
2. Title `<text>` at the top-left of the background rect
3. Child node and edge SVG elements

Subgraphs are rendered before (behind) top-level nodes and edges to ensure proper layering.

When the parser adds subgraph support, the renderer will:
1. Walk `FlowchartDiagram.Subgraphs` recursively
2. Compute bounding boxes from child `NodeLayout` positions
3. Render nested groups for nested subgraphs

## Style/Class Application

- `StyleDef` entries: node with matching ID gets inline `style` attribute with the raw CSS
- `Classes` entries: node with matching class name gets `class` attribute, CSS embedded in `<style>` block
- Specificity: inline `style` overrides theme defaults, class styles override theme defaults

## Rendering Pipeline (`renderer.go`)

```go
func Render(d *diagram.FlowchartDiagram, l *layout.Result, opts *Options) ([]byte, error) {
    // 1. Merge opts with defaults
    // 2. Compute SVG viewBox from layout.Width/Height + padding
    // 3. Build <defs> with arrow markers
    // 4. Render subgraph backgrounds (behind everything)
    // 5. Render edges (behind nodes, above subgraph backgrounds)
    // 6. Render nodes (top layer)
    // 7. Assemble SVG root and marshal to bytes
}
```

Layering order (bottom to top): background → subgraphs → edges → nodes → labels.

## Package Structure

```
pkg/renderer/flowchart/
  renderer.go        — Render(), Options, pipeline orchestration
  nodes.go           — renderNode(), shape-specific SVG builders
  edges.go           — renderEdge(), marker definitions, curve conversion
  subgraphs.go       — renderSubgraph(), bounding box computation
  theme.go           — Theme struct, DefaultTheme(), theme color helpers
  svg.go             — encoding/xml SVG element structs
  renderer_test.go   — unit tests for each component
  testdata/          — golden SVG files
```

## Testing Strategy

### Unit Tests
- **Per-shape tests**: each `NodeShape` produces correct SVG element with expected geometry
- **Edge tests**: straight line, multi-point curve, each arrow head type, each line style
- **Subgraph tests**: background rect dimensions, label positioning
- **Theme tests**: default theme applied, custom colors override, style/class application
- **Edge cases**: empty diagram, self-loops, multi-edges, zero-dimension nodes

### Golden File Tests
- Simple flowchart (3 nodes, 2 edges) → `testdata/simple.svg`
- All shapes diagram → `testdata/all-shapes.svg`
- Styled nodes → `testdata/styled.svg`
- Subgraph grouping → `testdata/subgraph.svg` (when parser supports it)

### XML Validity
Every test case validates output parses as well-formed XML via `xml.Unmarshal`.

## Dependencies

| Dependency | Package | Status |
|------------|---------|--------|
| Flowchart AST | `pkg/diagram` | Complete (Step 3) |
| Layout result | `pkg/layout` | Complete (Step 8) |
| Text measurement | `pkg/textmeasure` | Complete (Step 2) |
| encoding/xml | stdlib | Available |
| fmt, math, strings | stdlib | Available |

No external dependencies.
