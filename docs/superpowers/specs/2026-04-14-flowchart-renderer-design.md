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

**Nil handling:** `opts` may be `nil`; it is treated as zero-value `Options` (all defaults apply). `d` and `l` must not be `nil`.

### Options

```go
type Options struct {
    FontSize   float64  // default 16
    Padding    float64  // SVG padding around diagram, default 20
    Theme      Theme    // colors; zero value uses DefaultTheme()
    CSSFile    string   // optional raw CSS to embed in <style>
    Background string   // if set, overrides Theme.Background
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

**Background precedence:** `opts.Background` (highest) > `opts.Theme.Background` > `DefaultTheme().Background`.

**Default values:** `DefaultTheme()` returns the Mermaid "default" theme colors. Future themes (dark, forest, neutral) will be added when `pkg/config/` is built in Step 15.

### Error Returns

`Render` returns errors for:
- `nil` diagram or layout result: `fmt.Errorf("flowchart render: diagram is nil")` / `fmt.Errorf("flowchart render: layout is nil")`
- Node ID in diagram not found in layout result: `fmt.Errorf("flowchart render: node %q not in layout result", id)` — this indicates a bug in the caller (mismatched diagram and layout inputs)
- `encoding/xml` marshaling failure: wrapped error (should never occur with well-formed structs)

The `Render` function never panics.

## SVG Generation with encoding/xml

Each SVG element is a Go struct with `xml:` tags, marshaled via `encoding/xml.Marshal`. This guarantees well-formed XML and provides type safety.

### SVG Element Structs (`svg.go`)

```go
type SVG struct {
    XMLName  xml.Name `xml:"svg"`
    XMLNS    string   `xml:"xmlns,attr"`
    ViewBox  string   `xml:"viewBox,attr"`
    Width    string   `xml:"width,attr,omitempty"`
    Height   string   `xml:"height,attr,omitempty"`
    Children []any    `xml:",any"` // ordered: Defs, Group(subgraphs), edges, nodes
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

### AST-to-Layout Edge Matching

The layout engine's `Result.Edges` is keyed by `graph.EdgeID{From, To, ID}` where `ID` is auto-assigned. The renderer cannot directly look up an AST edge by constructing an `EdgeID` because the integer `ID` is unknown outside the layout graph.

**Solution:** The renderer iterates `layout.Result.Edges` (which provides all edge geometry) and matches each layout edge back to the AST edge by `(From, To)` pair. For multi-edges (same From→To), the renderer matches them in order — the Nth layout edge with the same (From, To) corresponds to the Nth AST edge with the same (From, To). This is correct because `Graph.SetEdge` assigns IDs monotonically and the layout engine preserves insertion order.

The `renderEdge` function accepts a `diagram.Edge` (for style/label/arrow) and an `EdgeLayout` (for geometry), paired by the caller.

### Path Construction

- **2 points** (straight line): `<line>` element from points[0] to points[1]
- **3+ points** (polyline/curve): `<path>` with cubic bezier segments. Convert the polyline through catmull-rom to bezier conversion (tension 0.5) for smooth curves.

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

Rendered as `<text>` at `EdgeLayout.LabelPos` with a white background `<rect>` behind the text for readability. The background rect dimensions are: text width + 6px horizontal padding, text height + 4px vertical padding, positioned centered on `LabelPos`.

## Subgraph Rendering (`subgraphs.go`)

Each subgraph becomes a `<g>` group containing:
1. Background `<rect>` covering the bounding box of all child nodes (with padding)
2. Title `<text>` at the top-left of the background rect
3. Child node and edge SVG elements

Subgraphs are rendered before (behind) top-level nodes and edges to ensure proper layering.

The renderer fully implements subgraph rendering using the existing `FlowchartDiagram.Subgraphs` and `Subgraph.Children` types:

1. **Recursive walk:** Walk `FlowchartDiagram.Subgraphs` depth-first. Outer subgraphs render before inner ones.
2. **Bounding box computation:** For each subgraph, collect all descendant node IDs (including those in nested subgraphs), look up their positions in `layout.Result.Nodes`, compute the axis-aligned bounding box. Add 15px padding on all sides.
3. **Background rect:** `<rect>` with `SubgraphFill` fill, `SubgraphStroke` stroke, rounded corners (rx=5).
4. **Title:** `<text>` positioned at (bbox.x + 10, bbox.y + 18) using the subgraph's `Label` field.
5. **Nested groups:** Each subgraph's `<g>` contains its background rect, title, and nested subgraph groups.

## Style/Class Application

- `StyleDef` entries: node with matching ID gets inline `style` attribute with the raw CSS
- `Classes` entries: node with matching class name gets `class` attribute, CSS embedded in `<style>` block
- Specificity: inline `style` overrides theme defaults, class styles override theme defaults

## Rendering Pipeline (`renderer.go`)

```go
func Render(d *diagram.FlowchartDiagram, l *layout.Result, opts *Options) ([]byte, error) {
    // 1. Validate inputs, merge opts with defaults
    // 2. Compute padding offset (pad = opts.Padding); all coordinates shift by +pad
    // 3. Compute SVG viewBox: "0 0 (layout.Width + 2*pad) (layout.Height + 2*pad)"
    // 4. Build <defs> with arrow markers
    // 5. Render subgraph backgrounds (behind everything), coordinates offset by +pad
    // 6. Render edges (behind nodes, above subgraph backgrounds), coordinates offset by +pad
    // 7. Render nodes (top layer), coordinates offset by +pad
    // 8. Assemble SVG.Children in layering order: [Defs, background rect, subgraph groups, edge elements, node elements]
    // 9. Prepend XML declaration: "<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n"
    // 10. Marshal via encoding/xml and return
}
```

**Coordinate offset:** The layout engine positions nodes starting at (0,0). The renderer adds `opts.Padding` to all X and Y coordinates so content is inset from the SVG edges. This is done once at the point where layout coordinates are read, not as a transform.

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
- **Edge cases**: empty diagram, self-loops (rendered as a small circular arc path centered on the node), multi-edges, zero-dimension nodes

### Golden File Tests
- Simple flowchart (3 nodes, 2 edges) → `testdata/simple.svg`
- All shapes diagram → `testdata/all-shapes.svg`
- Styled nodes → `testdata/styled.svg`
- Subgraph grouping → `testdata/subgraph.svg`

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
