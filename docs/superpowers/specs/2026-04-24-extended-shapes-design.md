# Phase A: @{shape:...} Extended Shape Syntax

## Problem

Mermaid v11.3.0+ supports 40+ node shapes via the `@{ shape: short-name, label: "..." }` annotation syntax. mmgo currently supports 14 shapes via traditional delimiters (`[]`, `()`, `{}`, etc.). Users writing modern Mermaid flowcharts with extended shapes get parse errors.

## Scope

- **Parser:** Recognize `@{...}` annotations on node definitions, parse `shape` and `label` key-value pairs
- **AST:** Add ~35 new `NodeShape` constants for net-new shapes
- **Renderer:** Add SVG rendering for each new shape
- **Edge clipping:** Add shape-aware clipping for circle and diamond nodes
- **Tests:** Parser tests, renderer tests, integration tests
- **Examples:** New example `.mmd` files demonstrating extended shapes

Out of scope: icon/image shapes (require external asset loading), orthogonal edge routing, self-loop arcs, back-edge visual differentiation.

## Parser Changes

### Syntax

```
A@{ shape: rect }
A@{ shape: diamond, label: "Decide" }
A["old label"]@{ shape: diamond, label: "new label" }
```

### Implementation

In `parseNodeDef()`, after extracting the node ID and before matching traditional shape delimiters:

1. Call `stripInlineClass()` first (already done — strips trailing `:::cls`)
2. Scan the remaining text for `@{`
3. If found, extract content between `@{` and `}`, parse as comma-separated key-value pairs
4. Split on `,`, then on first `:`, trim whitespace, handle quoted values
5. Look up `shape` value in a map of short-name/alias → `NodeShape`
6. If `label` is present, it overrides any label from traditional delimiters
7. If both `@{ shape: ... }` and traditional delimiters exist, `@{}` wins for shape

The key-value parser is a simple hand-rolled splitter — no YAML dependency needed. The practical subset is just `shape: name` and `label: "text"`.

### Parsing Precedence

The processing order within `parseNodeDef()` is:
1. Extract node ID (alphanumeric + `_` + hyphen between ID chars)
2. `stripInlineClass()` — removes trailing `:::cls` tokens
3. `parseShapeAnnotation()` — consumes `@{...}` if present, returns shape + label override
4. Traditional delimiter matching (`[]`, `()`, `{}`, etc.) — only if `@{}` did not supply a shape

The `@{}` annotation must appear after `:::` class tokens: `A:::cls@{ shape: diamond }` is valid; `A@{ shape: diamond }:::cls` is not (the `:::` would be inside the node definition after `@{}` is consumed, causing a parse error for unrecognized text).

### Interaction with Edge ID Syntax

Both `@{}` and edge IDs use `@`: edge IDs use trailing `@` (e.g., `A e1@-->B` via `extractEdgeID()`), while `@{}` always has `{` after `@`. These don't conflict because:
- `extractEdgeID()` only matches `@` followed by an ID character (not `{`)
- `parseShapeAnnotation()` only matches `@{`
- `findArrow()` naturally handles `{`/`}` in its depth tracking, so `@{...}` doesn't interfere with arrow detection

### Alias Resolution

A single map `shapeAliases map[string]diagram.NodeShape` covers all 45 Mermaid short names. Entries that map to existing AST constants (e.g., `rect` → `NodeShapeRectangle`) are included alongside new ones.

## AST Changes

### New NodeShape Constants

~35 new constants added to `pkg/diagram/flowchart.go`. All existing 14 shapes are preserved. New shapes:

| Constant | Short Names | SVG Approach |
|---|---|---|
| `NodeShapeCloud` | `cloud` | Arc-based cloud `<path>` |
| `NodeShapeHourglass` | `hourglass`, `collate` | Two triangles `<polygon>` |
| `NodeShapeBolt` | `bolt`, `com-link`, `lightning-bolt` | Zigzag `<path>` |
| `NodeShapeDocument` | `doc`, `document` | Rect + wavy bottom `<path>` |
| `NodeShapeLinedDocument` | `lin-doc`, `lined-document` | Document + horizontal line |
| `NodeShapeDelay` | `delay`, `half-rounded-rectangle` | Rect rounded on left only |
| `NodeShapeHorizontalCylinder` | `h-cyl`, `das`, `horizontal-cylinder` | Rotated cylinder `<path>` |
| `NodeShapeLinedCylinder` | `lin-cyl`, `disk`, `lined-cylinder` | Cylinder + horizontal stripe |
| `NodeShapeCurvedTrapezoid` | `curv-trap`, `curved-trapezoid`, `display` | Curved-side trapezoid `<path>` |
| `NodeShapeDividedRect` | `div-rect`, `div-proc`, `divided-process`, `divided-rectangle` | Rect + horizontal divider `<line>` |
| `NodeShapeTriangle` | `tri`, `extract`, `triangle` | Triangle `<polygon>` |
| `NodeShapeFlippedTriangle` | `flip-tri`, `flipped-triangle`, `manual-file` | Upside-down triangle `<polygon>` |
| `NodeShapeWindowPane` | `win-pane`, `internal-storage`, `window-pane` | Rect + cross `<line>`s |
| `NodeShapeFilledCircle` | `f-circ`, `filled-circle`, `junction` | Filled `<circle>` |
| `NodeShapeSmallCircle` | `sm-circ`, `small-circle`, `start` | Small `<circle>` |
| `NodeShapeFramedCircle` | `fr-circ`, `framed-circle`, `stop` | Circle with thick stroke |
| `NodeShapeNotchedRect` | `notch-rect`, `card`, `notched-rectangle` | Rect path with corner notch |
| `NodeShapeLinedRect` | `lin-rect`, `lin-proc`, `lined-process`, `lined-rectangle`, `shaded-process` | Rect + vertical `<line>` |
| `NodeShapeForkJoin` | `fork`, `join` | Tall thin filled `<rect>` |
| `NodeShapeStackedRect` | `st-rect`, `procs`, `processes`, `stacked-rectangle` | Two offset `<rect>`s |
| `NodeShapeStackedDocument` | `docs`, `documents`, `st-doc`, `stacked-document` | Two offset document shapes |
| `NodeShapeBowTieRect` | `bow-rect`, `bow-tie-rectangle`, `stored-data` | Hourglass rectangle `<path>` |
| `NodeShapeCrossCircle` | `cross-circ`, `crossed-circle`, `summary` | Circle + X `<line>`s |
| `NodeShapeTaggedRect` | `tag-rect`, `tag-proc`, `tagged-process`, `tagged-rectangle` | Rect + folded corner `<path>` |
| `NodeShapeTaggedDocument` | `tag-doc`, `tagged-document` | Document + folded corner |
| `NodeShapeFlag` | `flag`, `paper-tape` | Pennant `<polygon>` |
| `NodeShapeNotchedPentagon` | `notch-pent`, `loop-limit`, `notched-pentagon` | Pentagon with notch `<polygon>` |
| `NodeShapeSlopedRect` | `sl-rect`, `manual-input`, `sloped-rectangle` | Sloped rectangle `<polygon>` |
| `NodeShapeOdd` | `odd` | Irregular polygon |
| `NodeShapeTextBlock` | `text` | No border, just `<text>` |
| `NodeShapeBang` | `bang` | Starburst `<path>` |
| `NodeShapeBrace` | `brace`, `brace-l`, `comment` | Left curly brace `<path>` |
| `NodeShapeBraceR` | `brace-r` | Right curly brace `<path>` |
| `NodeShapeBraces` | `braces` | Both curly braces `<path>` |
| `NodeShapeDataStore` | `datastore`, `data-store` | Parallel arcs `<path>` |

### Shapes Mapping to Existing Constants

These short names map to existing AST shapes — no new enum values:

- `rect`, `proc`, `process`, `rectangle` → `NodeShapeRectangle`
- `rounded`, `event` → `NodeShapeRoundedRectangle`
- `stadium`, `terminal`, `pill` → `NodeShapeStadium`
- `fr-rect`, `subprocess`, `subroutine` → `NodeShapeSubroutine`
- `cyl`, `db`, `database`, `cylinder` → `NodeShapeCylinder`
- `circle`, `circ` → `NodeShapeCircle`
- `diam`, `decision`, `diamond`, `question` → `NodeShapeDiamond`
- `hex`, `hexagon`, `prepare` → `NodeShapeHexagon`
- `lean-r`, `lean-right`, `in-out` → `NodeShapeParallelogram`
- `lean-l`, `lean-left`, `out-in` → `NodeShapeParallelogramAlt`
- `trap-b`, `priority`, `trapezoid` → `NodeShapeTrapezoid`
- `trap-t`, `manual`, `trapezoid-top`, `inv-trapezoid` → `NodeShapeTrapezoidAlt`
- `dbl-circ`, `double-circle` → `NodeShapeDoubleCircle`

## Renderer Changes

### Shape Rendering

Each new shape gets a `case` in the `renderNode()` switch in `nodes.go`. Grouped by implementation complexity:

**Simple polygons** — compute vertex coordinates, emit `<polygon>`:
Triangle, FlippedTriangle, Hourglass, NotchedPentagon, Odd, Flag, SlopedRect

**Modified rectangles** — `<rect>` plus extra SVG elements:
DividedRect, WindowPane, LinedRect, ForkJoin, NotchedRect

**Circle variants** — `<circle>` with different styling:
SmallCircle, FilledCircle, FramedCircle, CrossCircle

**Path-based shapes** — `<path>` with curves/arcs:
Cloud, Document, LinedDocument, StackedDocument, TaggedDocument, Bolt, CurvedTrapezoid, HorizontalCylinder, LinedCylinder, BowTieRect, TaggedRect, Bang, Brace, BraceR, Braces, DataStore, StackedRect

**Special:**
- TextBlock — no shape element, only `<text>`
- StackedRect — two offset `<rect>` elements
- StackedDocument — two offset document `<path>` elements

### Text Sizing

Shapes with reduced usable area (triangles, hourglass, bang) have their text constrained. For Phase A, the layout engine assigns node dimensions based on text measurement; the renderer simply centers text in the bounding box. This matches how Mermaid.js handles it — the shape draws around the text area.

### Shape-Aware Edge Clipping

Currently `ClipToRectEdge()` clips all edge endpoints to the axis-aligned bounding box. Phase A adds shape-aware clipping for:

- **Circle variants** — use existing `ClipToCircleEdge()` from `svgutil`
- **Diamond** — new `ClipToDiamondEdge()` that intersects with the rhombus edges
- **All other shapes** — keep rectangular clipping (sufficient for rect-based shapes)

The renderer's edge rendering code checks the target/source node's shape and selects the appropriate clipping function.

## Testing Strategy

### Parser Tests (table-driven)

- `@{ shape: rect }` resolves to existing `NodeShapeRectangle`
- `@{ shape: diamond, label: "Decide" }` parses both fields
- All 45 short-name/alias entries resolve to correct `NodeShape`
- `@{}` overrides traditional delimiter shape; traditional label preserved unless `@{}` specifies one
- Edge cases: empty `@{}`, unknown shape, missing closing `}`, whitespace variations
- Combined syntax: `A["label"]@{ shape: diamond }` → diamond shape, label "label"
- Combined with override: `A["old"]@{ shape: diamond, label: "new" }` → diamond shape, label "new"
- `@{}` on nodes within edge statements: `A@{ shape: diamond }-->B@{ shape: rect }` (exercises the annotation parser through the edge-line code path)
- `@{}` combined with `:::` class: `A:::cls@{ shape: diamond }` → correct shape + class

### Renderer Tests

- Spot-check key shapes produce correct SVG element types
- Shape-aware clipping: circle and diamond nodes get proper clipping
- TextBlock renders no border element

### Integration Tests

- Full `.mmd` files with `@{shape:...}` through parse→layout→render pipeline
- Compare rendered output against expected SVG structure

### New Examples

Add example `.mmd` files to `examples/flowchart/` demonstrating:
- Basic extended shapes (a selection of the most common ones)
- Combined traditional + `@{}` syntax
- A practical flowchart using loop-limit, decision, and process shapes

## Files Modified

| File | Change |
|---|---|
| `pkg/diagram/flowchart.go` | Add ~35 new `NodeShape` constants + `nodeShapeNames` entries |
| `pkg/parser/flowchart/parser.go` | Add `parseShapeAnnotation()`, `parseKV()`, `shapeAliases` map; modify `parseNodeDef()` |
| `pkg/renderer/flowchart/nodes.go` | Add `case` branches for all new shapes |
| `pkg/renderer/flowchart/edges.go` | Shape-aware clipping dispatch |
| `pkg/renderer/svgutil/svgutil.go` | Add `ClipToDiamondEdge()` |
| `pkg/parser/flowchart/parser_test.go` | New test cases |
| `pkg/renderer/flowchart/renderer_test.go` | New test cases |
| `examples/flowchart/*.mmd` | New example files |
