# Flowchart Renderer Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build `pkg/renderer/flowchart/` — converts a parsed, laid-out flowchart into a complete SVG document.

**Architecture:** encoding/xml structs for SVG elements, 7-file package (svg.go, theme.go, nodes.go, edges.go, subgraphs.go, renderer.go, style.go), full API surface including subgraphs/styles/classes, straight + bezier edge rendering.

**Tech Stack:** Go 1.22 stdlib (encoding/xml, fmt, math, strings), existing pkg/diagram, pkg/layout

**Spec:** `docs/superpowers/specs/2026-04-14-flowchart-renderer-design.md`

---

## Chunk 1: Foundation (SVG structs, Theme, Stubs)

### Task 1: SVG element structs + StyleEl

**Files:**
- Create: `pkg/renderer/flowchart/svg.go`

- [ ] **Step 1: Create svg.go with all encoding/xml SVG element structs**

```go
package flowchart

import "encoding/xml"

type SVG struct {
	XMLName  xml.Name `xml:"svg"`
	XMLNS    string   `xml:"xmlns,attr"`
	ViewBox  string   `xml:"viewBox,attr"`
	Width    string   `xml:"width,attr,omitempty"`
	Height   string   `xml:"height,attr,omitempty"`
	Children []any    `xml:",any"`
}

type StyleEl struct {
	XMLName xml.Name `xml:"style"`
	Content string   `xml:",chardata"`
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

type Polyline struct {
	XMLName xml.Name `xml:"polyline"`
	Points  string   `xml:"points,attr"`
	Style   string   `xml:"style,attr,omitempty"`
}

type Path struct {
	XMLName     xml.Name `xml:"path"`
	D           string   `xml:"d,attr"`
	Style       string   `xml:"style,attr,omitempty"`
	Class       string   `xml:"class,attr,omitempty"`
	Fill        string   `xml:"fill,attr,omitempty"`
	MarkerEnd   string   `xml:"marker-end,attr,omitempty"`
	MarkerStart string   `xml:"marker-start,attr,omitempty"`
}

type Line struct {
	XMLName     xml.Name `xml:"line"`
	X1          float64  `xml:"x1,attr"`
	Y1          float64  `xml:"y1,attr"`
	X2          float64  `xml:"x2,attr"`
	Y2          float64  `xml:"y2,attr"`
	Style       string   `xml:"style,attr,omitempty"`
	Class       string   `xml:"class,attr,omitempty"`
	MarkerEnd   string   `xml:"marker-end,attr,omitempty"`
	MarkerStart string   `xml:"marker-start,attr,omitempty"`
}

type Text struct {
	XMLName    xml.Name `xml:"text"`
	X          float64  `xml:"x,attr"`
	Y          float64  `xml:"y,attr"`
	Anchor     string   `xml:"text-anchor,attr,omitempty"`
	Dominant   string   `xml:"dominant-baseline,attr,omitempty"`
	FontSize   float64  `xml:"font-size,attr,omitempty"`
	FontFamily string   `xml:"font-family,attr,omitempty"`
	Style      string   `xml:"style,attr,omitempty"`
	Class      string   `xml:"class,attr,omitempty"`
	Content    string   `xml:",chardata"`
}

type Defs struct {
	XMLName xml.Name `xml:"defs"`
	Markers []Marker `xml:"marker,omitempty"`
}

type Marker struct {
	XMLName  xml.Name `xml:"marker"`
	ID       string   `xml:"id,attr"`
	ViewBox  string   `xml:"viewBox,attr"`
	RefX     float64  `xml:"refX,attr"`
	RefY     float64  `xml:"refY,attr"`
	Width    float64  `xml:"markerWidth,attr"`
	Height   float64  `xml:"markerHeight,attr"`
	Orient   string   `xml:"orient,attr"`
	Children []any    `xml:",any"`
}
```

- [ ] **Step 2: Verify it compiles**

Run: `go build ./pkg/renderer/flowchart/`
Expected: compiles with no errors

- [ ] **Step 3: Commit**

```bash
git add pkg/renderer/flowchart/svg.go
git commit -m "Add SVG element structs for flowchart renderer"
```

---

### Task 2: Theme and Options types

**Files:**
- Create: `pkg/renderer/flowchart/theme.go`
- Create: `pkg/renderer/flowchart/theme_test.go`

- [ ] **Step 1: Write failing test for DefaultTheme**

Create `pkg/renderer/flowchart/theme_test.go`:

```go
package flowchart

import "testing"

func TestDefaultTheme(t *testing.T) {
	th := DefaultTheme()
	if th.NodeFill != "#fff" {
		t.Errorf("NodeFill = %q, want %q", th.NodeFill, "#fff")
	}
	if th.NodeStroke != "#333" {
		t.Errorf("NodeStroke = %q, want %q", th.NodeStroke, "#333")
	}
	if th.NodeText != "#333" {
		t.Errorf("NodeText = %q, want %q", th.NodeText, "#333")
	}
	if th.EdgeStroke != "#333" {
		t.Errorf("EdgeStroke = %q, want %q", th.EdgeStroke, "#333")
	}
	if th.EdgeText != "#333" {
		t.Errorf("EdgeText = %q, want %q", th.EdgeText, "#333")
	}
	if th.SubgraphFill != "#eee" {
		t.Errorf("SubgraphFill = %q, want %q", th.SubgraphFill, "#eee")
	}
	if th.SubgraphStroke != "#999" {
		t.Errorf("SubgraphStroke = %q, want %q", th.SubgraphStroke, "#999")
	}
	if th.SubgraphText != "#333" {
		t.Errorf("SubgraphText = %q, want %q", th.SubgraphText, "#333")
	}
	if th.Background != "#fff" {
		t.Errorf("Background = %q, want %q", th.Background, "#fff")
	}
}

func TestResolveBackground(t *testing.T) {
	th := DefaultTheme()
	cases := []struct {
		name string
		opts *Options
		want string
	}{
		{"nil opts", nil, "#fff"},
		{"empty opts", &Options{}, "#fff"},
		{"theme override", &Options{Theme: Theme{Background: "#000"}}, "#000"},
		{"background override", &Options{Background: "transparent"}, "transparent"},
		{"background takes precedence", &Options{Theme: Theme{Background: "#000"}, Background: "transparent"}, "transparent"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := resolveBackground(tc.opts, th)
			if got != tc.want {
				t.Errorf("resolveBackground() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestResolveFontSize(t *testing.T) {
	if resolveFontSize(nil) != 16 {
		t.Errorf("nil opts should default to 16")
	}
	if resolveFontSize(&Options{}) != 16 {
		t.Errorf("zero FontSize should default to 16")
	}
	if resolveFontSize(&Options{FontSize: 20}) != 20 {
		t.Errorf("explicit FontSize should be used")
	}
}

func TestResolvePadding(t *testing.T) {
	if resolvePadding(nil) != 20 {
		t.Errorf("nil opts should default to 20")
	}
	if resolvePadding(&Options{}) != 20 {
		t.Errorf("zero Padding should default to 20")
	}
	if resolvePadding(&Options{Padding: 40}) != 40 {
		t.Errorf("explicit Padding should be used")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/renderer/flowchart/ -run "TestDefaultTheme|TestResolve" -v`
Expected: FAIL — `DefaultTheme` undefined

- [ ] **Step 3: Write theme.go**

Create `pkg/renderer/flowchart/theme.go`:

```go
package flowchart

const (
	defaultFontSize = 16.0
	defaultPadding  = 20.0
)

type Options struct {
	FontSize   float64
	Padding    float64
	Theme      Theme
	CSSFile    string
	Background string
}

type Theme struct {
	NodeFill       string
	NodeStroke     string
	NodeText       string
	EdgeStroke     string
	EdgeText       string
	SubgraphFill   string
	SubgraphStroke string
	SubgraphText   string
	Background     string
}

func DefaultTheme() Theme {
	return Theme{
		NodeFill:       "#fff",
		NodeStroke:     "#333",
		NodeText:       "#333",
		EdgeStroke:     "#333",
		EdgeText:       "#333",
		SubgraphFill:   "#eee",
		SubgraphStroke: "#999",
		SubgraphText:   "#333",
		Background:     "#fff",
	}
}

func resolveTheme(opts *Options) Theme {
	th := DefaultTheme()
	if opts == nil {
		return th
	}
	if opts.Theme.NodeFill != "" {
		th.NodeFill = opts.Theme.NodeFill
	}
	if opts.Theme.NodeStroke != "" {
		th.NodeStroke = opts.Theme.NodeStroke
	}
	if opts.Theme.NodeText != "" {
		th.NodeText = opts.Theme.NodeText
	}
	if opts.Theme.EdgeStroke != "" {
		th.EdgeStroke = opts.Theme.EdgeStroke
	}
	if opts.Theme.EdgeText != "" {
		th.EdgeText = opts.Theme.EdgeText
	}
	if opts.Theme.SubgraphFill != "" {
		th.SubgraphFill = opts.Theme.SubgraphFill
	}
	if opts.Theme.SubgraphStroke != "" {
		th.SubgraphStroke = opts.Theme.SubgraphStroke
	}
	if opts.Theme.SubgraphText != "" {
		th.SubgraphText = opts.Theme.SubgraphText
	}
	if opts.Theme.Background != "" {
		th.Background = opts.Theme.Background
	}
	return th
}

func resolveBackground(opts *Options, th Theme) string {
	if opts != nil && opts.Background != "" {
		return opts.Background
	}
	return th.Background
}

func resolveFontSize(opts *Options) float64 {
	if opts != nil && opts.FontSize > 0 {
		return opts.FontSize
	}
	return defaultFontSize
}

func resolvePadding(opts *Options) float64 {
	if opts != nil && opts.Padding > 0 {
		return opts.Padding
	}
	return defaultPadding
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./pkg/renderer/flowchart/ -run "TestDefaultTheme|TestResolve" -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add pkg/renderer/flowchart/theme.go pkg/renderer/flowchart/theme_test.go
git commit -m "Add Theme, Options types and default resolution helpers"
```

---

### Task 3: Render stub + nodes stub + edges stub + subgraphs stub

This task creates all stub functions so every subsequent task produces a compilable state.

**Files:**
- Create: `pkg/renderer/flowchart/renderer.go`
- Create: `pkg/renderer/flowchart/nodes.go`
- Create: `pkg/renderer/flowchart/edges.go`
- Create: `pkg/renderer/flowchart/subgraphs.go`
- Create: `pkg/renderer/flowchart/style.go`
- Create: `pkg/renderer/flowchart/renderer_test.go`

- [ ] **Step 1: Write failing tests for Render**

Create `pkg/renderer/flowchart/renderer_test.go`:

```go
package flowchart

import (
	"encoding/xml"
	"strings"
	"testing"

	"github.com/julianshen/mmgo/pkg/diagram"
	"github.com/julianshen/mmgo/pkg/layout"
	"github.com/julianshen/mmgo/pkg/layout/graph"
)

func TestRenderNilInputs(t *testing.T) {
	_, err := Render(nil, &layout.Result{}, nil)
	if err == nil {
		t.Fatal("expected error for nil diagram")
	}
	if !strings.Contains(err.Error(), "diagram is nil") {
		t.Errorf("unexpected error: %v", err)
	}

	_, err = Render(&diagram.FlowchartDiagram{}, nil, nil)
	if err == nil {
		t.Fatal("expected error for nil layout")
	}
	if !strings.Contains(err.Error(), "layout is nil") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRenderEmptyDiagramProducesValidSVG(t *testing.T) {
	d := &diagram.FlowchartDiagram{}
	l := layout.Layout(graph.New(), layout.Options{})

	svgBytes, err := Render(d, l, nil)
	if err != nil {
		t.Fatalf("Render error: %v", err)
	}

	raw := string(svgBytes)
	if !strings.HasPrefix(raw, "<?xml") {
		t.Fatalf("SVG should start with XML declaration, got: %q", raw[:min(len(raw), 60)])
	}

	var svg SVG
	xmlStart := strings.Index(raw, "<svg")
	if xmlStart < 0 {
		t.Fatalf("no <svg> element in output:\n%s", raw)
	}
	if err := xml.Unmarshal([]byte(raw[xmlStart:]), &svg); err != nil {
		t.Fatalf("invalid SVG XML: %v\n%s", err, raw)
	}
	if svg.XMLNS != "http://www.w3.org/2000/svg" {
		t.Errorf("xmlns = %q, want SVG namespace", svg.XMLNS)
	}
	if svg.ViewBox == "" {
		t.Error("viewBox should be set")
	}
}

func TestRenderSingleNode(t *testing.T) {
	g := graph.New()
	g.SetNode("A", graph.NodeAttrs{Label: "Hello", Width: 100, Height: 50})
	l := layout.Layout(g, layout.Options{})
	d := &diagram.FlowchartDiagram{
		Nodes: []diagram.Node{{ID: "A", Label: "Hello", Shape: diagram.NodeShapeRectangle}},
	}

	svgBytes, err := Render(d, l, nil)
	if err != nil {
		t.Fatalf("Render error: %v", err)
	}

	raw := string(svgBytes)
	if !strings.Contains(raw, "<rect") {
		t.Errorf("SVG should contain a <rect> for a rectangle node:\n%s", raw)
	}
	if !strings.Contains(raw, ">Hello<") {
		t.Errorf("SVG should contain the label text 'Hello':\n%s", raw)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/renderer/flowchart/ -run "TestRender" -v`
Expected: FAIL — `Render` undefined

- [ ] **Step 3: Create renderer.go**

Create `pkg/renderer/flowchart/renderer.go`:

```go
package flowchart

import (
	"encoding/xml"
	"fmt"
	"strings"

	"github.com/julianshen/mmgo/pkg/diagram"
	"github.com/julianshen/mmgo/pkg/layout"
)

func Render(d *diagram.FlowchartDiagram, l *layout.Result, opts *Options) ([]byte, error) {
	if d == nil {
		return nil, fmt.Errorf("flowchart render: diagram is nil")
	}
	if l == nil {
		return nil, fmt.Errorf("flowchart render: layout is nil")
	}

	pad := resolvePadding(opts)
	th := resolveTheme(opts)
	fontSize := resolveFontSize(opts)
	bg := resolveBackground(opts, th)

	viewBoxW := l.Width + 2*pad
	viewBoxH := l.Height + 2*pad

	children := []any{
		buildDefs(d, th),
	}

	classCSS := buildClassCSS(d)
	if classCSS != "" || opts.CSSFile != "" {
		cssContent := classCSS + opts.CSSFile
		children = append(children, &StyleEl{Content: cssContent})
	}

	children = append(children, &Rect{
		X: 0, Y: 0,
		Width:  viewBoxW,
		Height: viewBoxH,
		Style:  fmt.Sprintf("fill:%s;stroke:none", bg),
	})

	children = append(children, renderSubgraphs(d, l, pad, th, fontSize)...)
	children = append(children, renderEdges(d, l, pad, th, fontSize)...)
	children = append(children, renderNodes(d, l, pad, th, fontSize)...)

	svg := SVG{
		XMLNS:   "http://www.w3.org/2000/svg",
		ViewBox: fmt.Sprintf("0 0 %.2f %.2f", viewBoxW, viewBoxH),
		Children: children,
	}

	svgBytes, err := xml.Marshal(svg)
	if err != nil {
		return nil, fmt.Errorf("flowchart render: %w", err)
	}

	xmlDecl := []byte("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n")
	return append(xmlDecl, svgBytes...), nil
}

func buildDefs(d *diagram.FlowchartDiagram, th Theme) *Defs {
	return &Defs{Markers: buildMarkers(d, th)}
}

func renderNodes(d *diagram.FlowchartDiagram, l *layout.Result, pad float64, th Theme, fontSize float64) []any {
	var elems []any
	for _, n := range d.Nodes {
		nl, ok := l.Nodes[n.ID]
		if !ok {
			continue
		}
		nodeElems := renderNode(n, nl, pad, th, fontSize)

		applyStyleOverrides(nodeElems, n, d.Styles)
		applyClassAttr(nodeElems, n)

		elems = append(elems, nodeElems...)
	}
	return elems
}

func applyStyleOverrides(elems []any, n diagram.Node, styles []diagram.StyleDef) {
	css := nodeStyleCSS(n, styles)
	if css == "" || len(elems) == 0 {
		return
	}
	switch e := elems[0].(type) {
	case *Rect:
		e.Style = css
	case *Polygon:
		e.Style = css
	case *Circle:
		e.Style = css
	case *Path:
		e.Style = css
	}
}

func applyClassAttr(elems []any, n diagram.Node) {
	if len(n.Classes) == 0 || len(elems) == 0 {
		return
	}
	classVal := strings.Join(n.Classes, " ")
	switch e := elems[0].(type) {
	case *Rect:
		e.Class = classVal
	case *Polygon:
		e.Class = classVal
	case *Circle:
		e.Class = classVal
	case *Path:
		e.Class = classVal
	}
}
```

- [ ] **Step 4: Create nodes.go stub**

Create `pkg/renderer/flowchart/nodes.go`:

```go
package flowchart

import (
	"github.com/julianshen/mmgo/pkg/diagram"
	"github.com/julianshen/mmgo/pkg/layout"
)

func renderNode(n diagram.Node, nl layout.NodeLayout, pad float64, th Theme, fontSize float64) []any {
	return nil
}
```

- [ ] **Step 5: Create edges.go stub**

Create `pkg/renderer/flowchart/edges.go`:

```go
package flowchart

import (
	"github.com/julianshen/mmgo/pkg/diagram"
	"github.com/julianshen/mmgo/pkg/layout"
)

func buildMarkers(d *diagram.FlowchartDiagram, th Theme) []Marker {
	return nil
}

func renderEdges(d *diagram.FlowchartDiagram, l *layout.Result, pad float64, th Theme, fontSize float64) []any {
	return nil
}
```

- [ ] **Step 6: Create subgraphs.go stub**

Create `pkg/renderer/flowchart/subgraphs.go`:

```go
package flowchart

import (
	"github.com/julianshen/mmgo/pkg/diagram"
	"github.com/julianshen/mmgo/pkg/layout"
)

func renderSubgraphs(d *diagram.FlowchartDiagram, l *layout.Result, pad float64, th Theme, fontSize float64) []any {
	return nil
}
```

- [ ] **Step 7: Create style.go**

Create `pkg/renderer/flowchart/style.go`:

```go
package flowchart

import (
	"fmt"
	"strings"

	"github.com/julianshen/mmgo/pkg/diagram"
)

func buildClassCSS(d *diagram.FlowchartDiagram) string {
	if len(d.Classes) == 0 {
		return ""
	}
	var sb strings.Builder
	for name, css := range d.Classes {
		sb.WriteString(fmt.Sprintf(".%s { %s }\n", name, css))
	}
	return sb.String()
}

func nodeStyleCSS(n diagram.Node, styles []diagram.StyleDef) string {
	for _, s := range styles {
		if s.NodeID == n.ID {
			return s.CSS
		}
	}
	return ""
}
```

- [ ] **Step 8: Run tests**

Run: `go test ./pkg/renderer/flowchart/ -run "TestRender" -v`
Expected: PASS (TestRenderSingleNode will fail because renderNode returns nil, but the nil-check tests pass. This is expected — nodes get implemented next.)

Wait — TestRenderSingleNode checks for `<rect` in output which requires renderNode to return something. Let me adjust the test to only test nil inputs and empty diagram in this task. The single-node test moves to Task 4.

Revised: Remove `TestRenderSingleNode` from `renderer_test.go` for now. It will be added in Task 4.

- [ ] **Step 9: Commit**

```bash
git add pkg/renderer/flowchart/renderer.go pkg/renderer/flowchart/renderer_test.go pkg/renderer/flowchart/nodes.go pkg/renderer/flowchart/edges.go pkg/renderer/flowchart/subgraphs.go pkg/renderer/flowchart/style.go
git commit -m "Add Render function with stubs for nodes, edges, subgraphs, styles"
```

---

## Chunk 2: Node Shape Rendering

### Task 4: All 14 node shapes

**Files:**
- Modify: `pkg/renderer/flowchart/nodes.go` (replace stub)
- Modify: `pkg/renderer/flowchart/renderer_test.go` (add node tests)
- Create: `pkg/renderer/flowchart/nodes_test.go`

- [ ] **Step 1: Write failing test for rectangle node**

Add to `pkg/renderer/flowchart/renderer_test.go`:

```go
func TestRenderSingleNode(t *testing.T) {
	g := graph.New()
	g.SetNode("A", graph.NodeAttrs{Label: "Hello", Width: 100, Height: 50})
	l := layout.Layout(g, layout.Options{})
	d := &diagram.FlowchartDiagram{
		Nodes: []diagram.Node{{ID: "A", Label: "Hello", Shape: diagram.NodeShapeRectangle}},
	}

	svgBytes, err := Render(d, l, nil)
	if err != nil {
		t.Fatalf("Render error: %v", err)
	}

	raw := string(svgBytes)
	if !strings.Contains(raw, "<rect") {
		t.Errorf("SVG should contain <rect>:\n%s", raw)
	}
	if !strings.Contains(raw, ">Hello<") {
		t.Errorf("SVG should contain label text:\n%s", raw)
	}
}

func TestRenderRectangleNodeGeometry(t *testing.T) {
	n := diagram.Node{ID: "A", Label: "Hello", Shape: diagram.NodeShapeRectangle}
	nl := layout.NodeLayout{X: 100, Y: 50, Width: 80, Height: 40}
	pad := 10.0

	elems := renderNode(n, nl, pad, DefaultTheme(), 16)
	if len(elems) < 2 {
		t.Fatalf("expected at least 2 elements (rect + text), got %d", len(elems))
	}

	rect, ok := elems[0].(*Rect)
	if !ok {
		t.Fatalf("first element should be *Rect, got %T", elems[0])
	}
	wantX := nl.X - nl.Width/2 + pad
	wantY := nl.Y - nl.Height/2 + pad
	if rect.X != wantX {
		t.Errorf("rect.X = %f, want %f", rect.X, wantX)
	}
	if rect.Y != wantY {
		t.Errorf("rect.Y = %f, want %f", rect.Y, wantY)
	}

	txt, ok := elems[1].(*Text)
	if !ok {
		t.Fatalf("second element should be *Text, got %T", elems[1])
	}
	if txt.Content != "Hello" {
		t.Errorf("text.Content = %q, want %q", txt.Content, "Hello")
	}
	if txt.Anchor != "middle" {
		t.Errorf("text-anchor = %q, want middle", txt.Anchor)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/renderer/flowchart/ -run "TestRenderSingleNode|TestRenderRectangleNodeGeometry" -v`
Expected: FAIL — renderNode returns nil

- [ ] **Step 3: Replace nodes.go stub with full implementation**

Replace entire contents of `pkg/renderer/flowchart/nodes.go`:

```go
package flowchart

import (
	"fmt"
	"math"
	"strings"

	"github.com/julianshen/mmgo/pkg/diagram"
	"github.com/julianshen/mmgo/pkg/layout"
)

func renderNode(n diagram.Node, nl layout.NodeLayout, pad float64, th Theme, fontSize float64) []any {
	shapeStyle := fmt.Sprintf("fill:%s;stroke:%s;stroke-width:1.5", th.NodeFill, th.NodeStroke)
	textStyle := fmt.Sprintf("fill:%s;font-size:%gpx", th.NodeText, fontSize)

	cx := nl.X + pad
	cy := nl.Y + pad
	w := nl.Width
	h := nl.Height

	var elems []any

	switch n.Shape {
	case diagram.NodeShapeRectangle:
		elems = append(elems, &Rect{
			X: cx - w/2, Y: cy - h/2, Width: w, Height: h, Style: shapeStyle,
		})
	case diagram.NodeShapeRoundedRectangle:
		elems = append(elems, &Rect{
			X: cx - w/2, Y: cy - h/2, Width: w, Height: h, RX: 5, RY: 5, Style: shapeStyle,
		})
	case diagram.NodeShapeStadium:
		elems = append(elems, &Rect{
			X: cx - w/2, Y: cy - h/2, Width: w, Height: h, RX: h / 2, RY: h / 2, Style: shapeStyle,
		})
	case diagram.NodeShapeCircle:
		r := math.Min(w, h) / 2
		elems = append(elems, &Circle{CX: cx, CY: cy, R: r, Style: shapeStyle})
	case diagram.NodeShapeDoubleCircle:
		r := math.Min(w, h) / 2
		elems = append(elems, &Circle{CX: cx, CY: cy, R: r, Style: shapeStyle})
		elems = append(elems, &Circle{
			CX: cx, CY: cy, R: r + 3,
			Style: fmt.Sprintf("fill:none;stroke:%s;stroke-width:1.5", th.NodeStroke),
		})
	case diagram.NodeShapeDiamond:
		elems = append(elems, &Polygon{Points: diamondPoints(cx, cy, w, h), Style: shapeStyle})
	case diagram.NodeShapeHexagon:
		elems = append(elems, &Polygon{Points: hexagonPoints(cx, cy, w, h), Style: shapeStyle})
	case diagram.NodeShapeParallelogram:
		elems = append(elems, &Polygon{Points: parallelogramPoints(cx, cy, w, h, 0.15, false), Style: shapeStyle})
	case diagram.NodeShapeParallelogramAlt:
		elems = append(elems, &Polygon{Points: parallelogramPoints(cx, cy, w, h, 0.15, true), Style: shapeStyle})
	case diagram.NodeShapeTrapezoid:
		elems = append(elems, &Polygon{Points: trapezoidPoints(cx, cy, w, h, 0.15, false), Style: shapeStyle})
	case diagram.NodeShapeTrapezoidAlt:
		elems = append(elems, &Polygon{Points: trapezoidPoints(cx, cy, w, h, 0.15, true), Style: shapeStyle})
	case diagram.NodeShapeAsymmetric:
		elems = append(elems, &Polygon{Points: asymmetricPoints(cx, cy, w, h), Style: shapeStyle})
	case diagram.NodeShapeCylinder:
		elems = append(elems, &Path{D: cylinderPath(cx, cy, w, h), Style: shapeStyle})
	case diagram.NodeShapeSubroutine:
		elems = append(elems, &Rect{
			X: cx - w/2, Y: cy - h/2, Width: w, Height: h, Style: shapeStyle,
		})
		bandX1 := cx - w/2 + w*0.1
		bandX2 := cx + w/2 - w*0.1
		lineStyle := fmt.Sprintf("stroke:%s;stroke-width:1.5", th.NodeStroke)
		elems = append(elems, &Line{X1: bandX1, Y1: cy - h/2, X2: bandX1, Y2: cy + h/2, Style: lineStyle})
		elems = append(elems, &Line{X1: bandX2, Y1: cy - h/2, X2: bandX2, Y2: cy + h/2, Style: lineStyle})
	default:
		elems = append(elems, &Rect{
			X: cx - w/2, Y: cy - h/2, Width: w, Height: h, Style: shapeStyle,
		})
	}

	lines := strings.Split(n.Label, "\n")
	lineHeight := fontSize * 1.2
	startY := cy - float64(len(lines)-1)*lineHeight/2
	for i, line := range lines {
		elems = append(elems, &Text{
			X: cx, Y: startY + float64(i)*lineHeight,
			Anchor: "middle", Dominant: "central", FontSize: fontSize,
			Style: textStyle, Content: line,
		})
	}

	return elems
}

func diamondPoints(cx, cy, w, h float64) string {
	return fmt.Sprintf("%.2f,%.2f %.2f,%.2f %.2f,%.2f %.2f,%.2f",
		cx, cy-h/2, cx+w/2, cy, cx, cy+h/2, cx-w/2, cy)
}

func hexagonPoints(cx, cy, w, h float64) string {
	d := w * 0.15
	return fmt.Sprintf("%.2f,%.2f %.2f,%.2f %.2f,%.2f %.2f,%.2f %.2f,%.2f %.2f,%.2f",
		cx-w/2+d, cy-h/2, cx+w/2-d, cy-h/2, cx+w/2, cy,
		cx+w/2-d, cy+h/2, cx-w/2+d, cy+h/2, cx-w/2, cy)
}

func parallelogramPoints(cx, cy, w, h float64, skew float64, reverse bool) string {
	s := w * skew
	if reverse {
		return fmt.Sprintf("%.2f,%.2f %.2f,%.2f %.2f,%.2f %.2f,%.2f",
			cx-w/2+s, cy-h/2, cx+w/2+s, cy-h/2,
			cx+w/2-s, cy+h/2, cx-w/2-s, cy+h/2)
	}
	return fmt.Sprintf("%.2f,%.2f %.2f,%.2f %.2f,%.2f %.2f,%.2f",
		cx-w/2+s, cy-h/2, cx+w/2-s, cy-h/2,
		cx+w/2+s, cy+h/2, cx-w/2-s, cy+h/2)
}

func trapezoidPoints(cx, cy, w, h float64, indent float64, alt bool) string {
	d := w * indent
	if alt {
		return fmt.Sprintf("%.2f,%.2f %.2f,%.2f %.2f,%.2f %.2f,%.2f",
			cx-w/2+d, cy-h/2, cx+w/2-d, cy-h/2,
			cx+w/2, cy+h/2, cx-w/2, cy+h/2)
	}
	return fmt.Sprintf("%.2f,%.2f %.2f,%.2f %.2f,%.2f %.2f,%.2f",
		cx-w/2, cy-h/2, cx+w/2, cy-h/2,
		cx+w/2-d, cy+h/2, cx-w/2+d, cy+h/2)
}

func asymmetricPoints(cx, cy, w, h float64) string {
	s := w * 0.15
	return fmt.Sprintf("%.2f,%.2f %.2f,%.2f %.2f,%.2f %.2f,%.2f",
		cx-w/2, cy-h/2, cx+w/2-s, cy-h/2,
		cx+w/2, cy+h/2, cx-w/2, cy+h/2)
}

func cylinderPath(cx, cy, w, h float64) string {
	ry := h * 0.1
	top := cy - h/2 + ry
	bot := cy + h/2 - ry
	return fmt.Sprintf("M%.2f,%.2f L%.2f,%.2f A%.2f,%.2f 0 0,0 %.2f,%.2f "+
		"L%.2f,%.2f A%.2f,%.2f 0 0,0 %.2f,%.2f "+
		"M%.2f,%.2f A%.2f,%.2f 0 0,0 %.2f,%.2f",
		cx-w/2, top, cx-w/2, bot, w/2, ry, cx+w/2, bot,
		cx+w/2, top, w/2, ry, cx-w/2, top,
		cx-w/2, top, w/2, ry, cx+w/2, top)
}
```

- [ ] **Step 4: Run tests**

Run: `go test ./pkg/renderer/flowchart/ -run "TestRenderSingleNode|TestRenderRectangleNodeGeometry" -v`
Expected: PASS

- [ ] **Step 5: Write comprehensive shape tests**

Create `pkg/renderer/flowchart/nodes_test.go`:

```go
package flowchart

import (
	"strings"
	"testing"

	"github.com/julianshen/mmgo/pkg/diagram"
	"github.com/julianshen/mmgo/pkg/layout"
)

func TestRenderAllShapes(t *testing.T) {
	shapes := []diagram.NodeShape{
		diagram.NodeShapeRectangle,
		diagram.NodeShapeRoundedRectangle,
		diagram.NodeShapeStadium,
		diagram.NodeShapeDiamond,
		diagram.NodeShapeHexagon,
		diagram.NodeShapeCircle,
		diagram.NodeShapeDoubleCircle,
		diagram.NodeShapeParallelogram,
		diagram.NodeShapeParallelogramAlt,
		diagram.NodeShapeTrapezoid,
		diagram.NodeShapeTrapezoidAlt,
		diagram.NodeShapeCylinder,
		diagram.NodeShapeSubroutine,
		diagram.NodeShapeAsymmetric,
		diagram.NodeShapeUnknown,
	}
	for _, shape := range shapes {
		t.Run(shape.String(), func(t *testing.T) {
			n := diagram.Node{ID: "A", Label: "Test", Shape: shape}
			nl := layout.NodeLayout{X: 100, Y: 50, Width: 80, Height: 40}
			elems := renderNode(n, nl, 10, DefaultTheme(), 16)
			if len(elems) < 2 {
				t.Fatalf("shape %s: expected at least 2 elements, got %d", shape, len(elems))
			}
			txt, ok := elems[len(elems)-1].(*Text)
			if !ok {
				t.Fatalf("shape %s: last element should be *Text, got %T", shape, elems[len(elems)-1])
			}
			if txt.Content != "Test" {
				t.Errorf("shape %s: text = %q, want %q", shape, txt.Content, "Test")
			}
		})
	}
}

func TestRenderRoundedRectHasRX(t *testing.T) {
	elems := renderNode(
		diagram.Node{ID: "A", Label: "R", Shape: diagram.NodeShapeRoundedRectangle},
		layout.NodeLayout{X: 50, Y: 50, Width: 80, Height: 40}, 0, DefaultTheme(), 16)
	rect, ok := elems[0].(*Rect)
	if !ok {
		t.Fatalf("expected *Rect, got %T", elems[0])
	}
	if rect.RX != 5 {
		t.Errorf("RX = %f, want 5", rect.RX)
	}
}

func TestRenderStadiumHasFullRX(t *testing.T) {
	elems := renderNode(
		diagram.Node{ID: "A", Label: "S", Shape: diagram.NodeShapeStadium},
		layout.NodeLayout{X: 50, Y: 50, Width: 80, Height: 40}, 0, DefaultTheme(), 16)
	rect, ok := elems[0].(*Rect)
	if !ok {
		t.Fatalf("expected *Rect, got %T", elems[0])
	}
	if rect.RX != 20 {
		t.Errorf("RX = %f, want 20 (h/2)", rect.RX)
	}
}

func TestRenderDoubleCircleHasTwoCircles(t *testing.T) {
	elems := renderNode(
		diagram.Node{ID: "A", Label: "D", Shape: diagram.NodeShapeDoubleCircle},
		layout.NodeLayout{X: 50, Y: 50, Width: 80, Height: 40}, 0, DefaultTheme(), 16)
	circles := 0
	for _, e := range elems {
		if _, ok := e.(*Circle); ok {
			circles++
		}
	}
	if circles != 2 {
		t.Errorf("expected 2 circles, got %d", circles)
	}
}

func TestRenderSubroutineHasLines(t *testing.T) {
	elems := renderNode(
		diagram.Node{ID: "A", Label: "Sub", Shape: diagram.NodeShapeSubroutine},
		layout.NodeLayout{X: 50, Y: 50, Width: 80, Height: 40}, 0, DefaultTheme(), 16)
	lines := 0
	for _, e := range elems {
		if _, ok := e.(*Line); ok {
			lines++
		}
	}
	if lines != 2 {
		t.Errorf("subroutine should have 2 vertical lines, got %d", lines)
	}
}

func TestRenderMultiLineLabel(t *testing.T) {
	elems := renderNode(
		diagram.Node{ID: "A", Label: "Line1\nLine2\nLine3", Shape: diagram.NodeShapeRectangle},
		layout.NodeLayout{X: 50, Y: 50, Width: 80, Height: 40}, 0, DefaultTheme(), 16)
	texts := 0
	for _, e := range elems {
		if txt, ok := e.(*Text); ok {
			texts++
			if !strings.HasPrefix(txt.Content, "Line") {
				t.Errorf("unexpected text: %q", txt.Content)
			}
		}
	}
	if texts != 3 {
		t.Errorf("expected 3 text elements, got %d", texts)
	}
}
```

- [ ] **Step 6: Run all tests**

Run: `go test ./pkg/renderer/flowchart/ -v`
Expected: PASS

- [ ] **Step 7: Commit**

```bash
git add pkg/renderer/flowchart/nodes.go pkg/renderer/flowchart/nodes_test.go pkg/renderer/flowchart/renderer_test.go
git commit -m "Add all 14 node shape renderers with comprehensive tests"
```

---

## Chunk 3: Edge Rendering

### Task 5: Arrow markers and edge rendering

**Files:**
- Replace: `pkg/renderer/flowchart/edges.go` (replace stub)
- Create: `pkg/renderer/flowchart/edges_test.go`

- [ ] **Step 1: Write failing tests for markers and edges**

Create `pkg/renderer/flowchart/edges_test.go`:

```go
package flowchart

import (
	"strings"
	"testing"

	"github.com/julianshen/mmgo/pkg/diagram"
	"github.com/julianshen/mmgo/pkg/layout"
)

func TestBuildMarkers(t *testing.T) {
	d := &diagram.FlowchartDiagram{
		Edges: []diagram.Edge{
			{From: "A", To: "B", ArrowHead: diagram.ArrowHeadArrow},
			{From: "B", To: "C", ArrowHead: diagram.ArrowHeadOpen},
			{From: "C", To: "D", ArrowHead: diagram.ArrowHeadCross},
			{From: "D", To: "E", ArrowHead: diagram.ArrowHeadCircle},
			{From: "E", To: "F", ArrowHead: diagram.ArrowHeadNone},
		},
	}
	defs := buildDefs(d, DefaultTheme())
	ids := map[string]bool{}
	for _, m := range defs.Markers {
		ids[m.ID] = true
	}
	if !ids["arrow-arrow-solid"] {
		t.Error("expected marker arrow-arrow-solid")
	}
	if !ids["arrow-open-solid"] {
		t.Error("expected marker arrow-open-solid")
	}
	if !ids["arrow-cross-solid"] {
		t.Error("expected marker arrow-cross-solid")
	}
	if !ids["arrow-circle-solid"] {
		t.Error("expected marker arrow-circle-solid")
	}
	if ids["arrow-none-solid"] {
		t.Error("should not have marker for ArrowHeadNone")
	}
}

func TestBuildMarkersDotted(t *testing.T) {
	d := &diagram.FlowchartDiagram{
		Edges: []diagram.Edge{
			{From: "A", To: "B", ArrowHead: diagram.ArrowHeadArrow, LineStyle: diagram.LineStyleDotted},
		},
	}
	defs := buildDefs(d, DefaultTheme())
	if len(defs.Markers) != 1 {
		t.Fatalf("expected 1 marker, got %d", len(defs.Markers))
	}
	if !strings.Contains(defs.Markers[0].ID, "dotted") {
		t.Errorf("marker ID should contain 'dotted': %s", defs.Markers[0].ID)
	}
}

func TestRenderEdgeStraightLine(t *testing.T) {
	e := diagram.Edge{From: "A", To: "B", ArrowHead: diagram.ArrowHeadArrow}
	el := layout.EdgeLayout{
		Points:   []layout.Point{{X: 0, Y: 0}, {X: 100, Y: 0}},
		LabelPos: layout.Point{X: 50, Y: 0},
	}
	elems := renderEdge(e, el, 10, DefaultTheme(), 16)
	if len(elems) < 1 {
		t.Fatal("expected at least 1 element")
	}
	line, ok := elems[0].(*Line)
	if !ok {
		t.Fatalf("expected *Line, got %T", elems[0])
	}
	if line.X1 != 10 || line.Y1 != 10 {
		t.Errorf("start = (%.2f,%.2f), want (10,10)", line.X1, line.Y1)
	}
	if line.X2 != 110 || line.Y2 != 10 {
		t.Errorf("end = (%.2f,%.2f), want (110,10)", line.X2, line.Y2)
	}
	if !strings.Contains(line.MarkerEnd, "arrow-arrow") {
		t.Errorf("expected arrow marker, got %s", line.MarkerEnd)
	}
}

func TestRenderEdgeWithLabel(t *testing.T) {
	e := diagram.Edge{From: "A", To: "B", Label: "yes", ArrowHead: diagram.ArrowHeadArrow}
	el := layout.EdgeLayout{
		Points:   []layout.Point{{X: 0, Y: 0}, {X: 100, Y: 0}},
		LabelPos: layout.Point{X: 50, Y: 0},
	}
	elems := renderEdge(e, el, 0, DefaultTheme(), 16)
	hasLabel := false
	for _, elem := range elems {
		if txt, ok := elem.(*Text); ok && txt.Content == "yes" {
			hasLabel = true
		}
	}
	if !hasLabel {
		t.Error("expected text element with label 'yes'")
	}
}

func TestRenderEdgeLabelBackgroundRect(t *testing.T) {
	e := diagram.Edge{From: "A", To: "B", Label: "bg", ArrowHead: diagram.ArrowHeadArrow}
	el := layout.EdgeLayout{
		Points:   []layout.Point{{X: 0, Y: 0}, {X: 100, Y: 0}},
		LabelPos: layout.Point{X: 50, Y: 0},
	}
	elems := renderEdge(e, el, 0, DefaultTheme(), 16)
	hasBgRect := false
	for _, elem := range elems {
		if r, ok := elem.(*Rect); ok && r.Style != "" && strings.Contains(r.Style, "fill:white") {
			hasBgRect = true
		}
	}
	if !hasBgRect {
		t.Error("edge label should have a white background rect")
	}
}

func TestRenderEdgeDotted(t *testing.T) {
	e := diagram.Edge{From: "A", To: "B", LineStyle: diagram.LineStyleDotted, ArrowHead: diagram.ArrowHeadArrow}
	el := layout.EdgeLayout{
		Points:   []layout.Point{{X: 0, Y: 0}, {X: 100, Y: 0}},
		LabelPos: layout.Point{X: 50, Y: 0},
	}
	elems := renderEdge(e, el, 0, DefaultTheme(), 16)
	line, ok := elems[0].(*Line)
	if !ok {
		t.Fatalf("expected *Line, got %T", elems[0])
	}
	if !strings.Contains(line.Style, "stroke-dasharray:2,2") {
		t.Errorf("dotted edge should have dasharray, got: %s", line.Style)
	}
}

func TestRenderEdgeNoMarker(t *testing.T) {
	e := diagram.Edge{From: "A", To: "B", ArrowHead: diagram.ArrowHeadNone}
	el := layout.EdgeLayout{
		Points:   []layout.Point{{X: 0, Y: 0}, {X: 100, Y: 0}},
		LabelPos: layout.Point{X: 50, Y: 0},
	}
	elems := renderEdge(e, el, 0, DefaultTheme(), 16)
	line, ok := elems[0].(*Line)
	if !ok {
		t.Fatalf("expected *Line, got %T", elems[0])
	}
	if line.MarkerEnd != "" {
		t.Errorf("no-arrow edge should have empty marker-end, got %s", line.MarkerEnd)
	}
}

func TestRenderEdgeCurve(t *testing.T) {
	e := diagram.Edge{From: "A", To: "B", ArrowHead: diagram.ArrowHeadArrow}
	el := layout.EdgeLayout{
		Points:   []layout.Point{{X: 0, Y: 0}, {X: 50, Y: 50}, {X: 100, Y: 0}},
		LabelPos: layout.Point{X: 50, Y: 25},
	}
	elems := renderEdge(e, el, 0, DefaultTheme(), 16)
	path, ok := elems[0].(*Path)
	if !ok {
		t.Fatalf("expected *Path for 3-point edge, got %T", elems[0])
	}
	if !strings.Contains(path.D, " C") {
		t.Errorf("curve path should contain cubic bezier, got: %s", path.D)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/renderer/flowchart/ -run "TestBuildMarkers|TestRenderEdge" -v`
Expected: FAIL — buildMarkers returns nil

- [ ] **Step 3: Replace edges.go stub with full implementation**

Replace entire contents of `pkg/renderer/flowchart/edges.go`:

```go
package flowchart

import (
	"fmt"
	"strings"

	"github.com/julianshen/mmgo/pkg/diagram"
	"github.com/julianshen/mmgo/pkg/layout"
)

func markerID(ah diagram.ArrowHead, ls diagram.LineStyle) string {
	return fmt.Sprintf("arrow-%s-%s", ah, ls)
}

func buildMarkers(d *diagram.FlowchartDiagram, th Theme) []Marker {
	needed := map[string]diagram.ArrowHead{}
	for _, e := range d.Edges {
		if e.ArrowHead == diagram.ArrowHeadNone || e.ArrowHead == diagram.ArrowHeadUnknown {
			continue
		}
		id := markerID(e.ArrowHead, e.LineStyle)
		needed[id] = e.ArrowHead
	}

	var markers []Marker
	for id, ah := range needed {
		markers = append(markers, buildMarker(id, ah, th))
	}
	return markers
}

func buildMarker(id string, ah diagram.ArrowHead, th Theme) Marker {
	m := Marker{
		ID: id, ViewBox: "0 0 10 10",
		RefX: 9, RefY: 5, Width: 8, Height: 8,
		Orient: "auto",
	}

	switch ah {
	case diagram.ArrowHeadArrow:
		m.Children = []any{
			&Polygon{Points: "0,0 10,5 0,10", Style: fmt.Sprintf("fill:%s", th.EdgeStroke)},
		}
	case diagram.ArrowHeadOpen:
		m.RefX = 10
		m.Children = []any{
			&Polyline{
				Points: "0,1 10,5 0,9",
				Style:  fmt.Sprintf("stroke:%s;stroke-width:1.5;fill:none", th.EdgeStroke),
			},
		}
	case diagram.ArrowHeadCross:
		m.Children = []any{
			&Polyline{
				Points: "0,0 10,5 0,10 10,5",
				Style:  fmt.Sprintf("stroke:%s;stroke-width:1.5;fill:none", th.EdgeStroke),
			},
		}
	case diagram.ArrowHeadCircle:
		m.RefX = 5
		m.Children = []any{
			&Circle{
				CX: 5, CY: 5, R: 4,
				Style: fmt.Sprintf("stroke:%s;stroke-width:1.5;fill:none", th.EdgeStroke),
			},
		}
	}
	return m
}

func renderEdges(d *diagram.FlowchartDiagram, l *layout.Result, pad float64, th Theme, fontSize float64) []any {
	type indexed struct {
		diagram.Edge
		idx int
	}
	fromTo := map[string][]indexed{}
	for i, e := range d.Edges {
		key := e.From + "->" + e.To
		fromTo[key] = append(fromTo[key], indexed{Edge: e, idx: i})
	}

	var elems []any
	for eid, elayout := range l.Edges {
		key := eid.From + "->" + eid.To
		candidates := fromTo[key]
		if len(candidates) == 0 {
			continue
		}
		ae := candidates[0]
		fromTo[key] = candidates[1:]

		elems = append(elems, renderEdge(ae.Edge, elayout, pad, th, fontSize)...)
	}
	return elems
}

func renderEdge(e diagram.Edge, el layout.EdgeLayout, pad float64, th Theme, fontSize float64) []any {
	pts := el.Points
	if len(pts) == 0 {
		return nil
	}

	for i := range pts {
		pts[i].X += pad
		pts[i].Y += pad
	}

	style := edgeStyle(th, e.LineStyle)
	var elems []any

	if len(pts) == 2 {
		line := &Line{
			X1: pts[0].X, Y1: pts[0].Y,
			X2: pts[1].X, Y2: pts[1].Y,
			Style: style,
		}
		if e.ArrowHead != diagram.ArrowHeadNone && e.ArrowHead != diagram.ArrowHeadUnknown {
			line.MarkerEnd = fmt.Sprintf("url(#%s)", markerID(e.ArrowHead, e.LineStyle))
		}
		elems = append(elems, line)
	} else if len(pts) >= 3 {
		p := &Path{D: buildCurvePath(pts), Style: style}
		if e.ArrowHead != diagram.ArrowHeadNone && e.ArrowHead != diagram.ArrowHeadUnknown {
			p.MarkerEnd = fmt.Sprintf("url(#%s)", markerID(e.ArrowHead, e.LineStyle))
		}
		elems = append(elems, p)
	}

	if e.Label != "" {
		lx := el.LabelPos.X + pad
		ly := el.LabelPos.Y + pad
		textStyle := fmt.Sprintf("fill:%s;font-size:%gpx", th.EdgeText, fontSize)

		elems = append(elems, &Rect{
			X: lx - 20, Y: ly - 10,
			Width: 40, Height: 20,
			Style: "fill:white;stroke:none",
		})
		elems = append(elems, &Text{
			X: lx, Y: ly,
			Anchor: "middle", Dominant: "central",
			FontSize: fontSize, Style: textStyle, Content: e.Label,
		})
	}

	return elems
}

func edgeStyle(th Theme, ls diagram.LineStyle) string {
	extra := ""
	switch ls {
	case diagram.LineStyleDotted:
		extra = "stroke-dasharray:2,2;"
	case diagram.LineStyleThick:
		extra = "stroke-width:3;"
	}
	return fmt.Sprintf("stroke:%s;stroke-width:1.5;fill:none;%s", th.EdgeStroke, extra)
}

func buildCurvePath(pts []layout.Point) string {
	if len(pts) < 3 {
		return ""
	}
	d := fmt.Sprintf("M%.2f,%.2f", pts[0].X, pts[0].Y)

	tension := 0.5
	for i := 0; i < len(pts)-1; i++ {
		p0 := pts[max(i-1, 0)]
		p1 := pts[i]
		p2 := pts[i+1]
		p3 := pts[min(i+2, len(pts)-1)]

		cp1x := p1.X + (p2.X-p0.X)*tension/3
		cp1y := p1.Y + (p2.Y-p0.Y)*tension/3
		cp2x := p2.X - (p3.X-p1.X)*tension/3
		cp2y := p2.Y - (p3.Y-p1.Y)*tension/3

		d += fmt.Sprintf(" C%.2f,%.2f %.2f,%.2f %.2f,%.2f",
			cp1x, cp1y, cp2x, cp2y, p2.X, p2.Y)
	}
	return d
}
```

Note: `min` and `max` are Go 1.22 builtins — no manual implementation needed.

- [ ] **Step 4: Run edge tests**

Run: `go test ./pkg/renderer/flowchart/ -run "TestBuildMarkers|TestRenderEdge" -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add pkg/renderer/flowchart/edges.go pkg/renderer/flowchart/edges_test.go
git commit -m "Add edge rendering with markers, curves, labels, and line styles"
```

---

## Chunk 4: Subgraphs, Styles, Golden Files

### Task 6: Subgraph rendering

**Files:**
- Replace: `pkg/renderer/flowchart/subgraphs.go` (replace stub)
- Create: `pkg/renderer/flowchart/subgraphs_test.go`

- [ ] **Step 1: Write failing tests**

Create `pkg/renderer/flowchart/subgraphs_test.go`:

```go
package flowchart

import (
	"testing"

	"github.com/julianshen/mmgo/pkg/diagram"
	"github.com/julianshen/mmgo/pkg/layout"
)

func TestRenderSubgraph(t *testing.T) {
	d := &diagram.FlowchartDiagram{
		Subgraphs: []diagram.Subgraph{
			{
				ID: "sg1", Label: "My Group",
				Nodes: []diagram.Node{
					{ID: "A", Label: "Node A"},
					{ID: "B", Label: "Node B"},
				},
			},
		},
	}
	l := &layout.Result{
		Nodes: map[string]layout.NodeLayout{
			"A": {X: 50, Y: 50, Width: 80, Height: 40},
			"B": {X: 200, Y: 50, Width: 80, Height: 40},
		},
		Width: 280, Height: 100,
	}

	elems := renderSubgraphs(d, l, 10, DefaultTheme(), 16)
	if len(elems) == 0 {
		t.Fatal("expected subgraph elements")
	}
	group, ok := elems[0].(*Group)
	if !ok {
		t.Fatalf("expected *Group, got %T", elems[0])
	}
	if group.ID != "sg1" {
		t.Errorf("group ID = %q, want %q", group.ID, "sg1")
	}

	hasRect, hasText := false, false
	for _, child := range group.Children {
		if _, ok := child.(*Rect); ok {
			hasRect = true
		}
		if txt, ok := child.(*Text); ok && txt.Content == "My Group" {
			hasText = true
		}
	}
	if !hasRect {
		t.Error("subgraph should contain a background rect")
	}
	if !hasText {
		t.Error("subgraph should contain title 'My Group'")
	}
}

func TestSubgraphBBox(t *testing.T) {
	nodes := []diagram.Node{{ID: "A"}, {ID: "B"}}
	layoutNodes := map[string]layout.NodeLayout{
		"A": {X: 100, Y: 100, Width: 60, Height: 40},
		"B": {X: 300, Y: 200, Width: 60, Height: 40},
	}
	bb := subgraphBBox(nodes, layoutNodes)
	wantMinX := 100.0 - 60/2
	if bb.MinX != wantMinX {
		t.Errorf("MinX = %f, want %f", bb.MinX, wantMinX)
	}
}

func TestNestedSubgraph(t *testing.T) {
	d := &diagram.FlowchartDiagram{
		Subgraphs: []diagram.Subgraph{
			{
				ID: "outer", Label: "Outer",
				Nodes: []diagram.Node{{ID: "A"}},
				Children: []diagram.Subgraph{
					{ID: "inner", Label: "Inner", Nodes: []diagram.Node{{ID: "B"}}},
				},
			},
		},
	}
	l := &layout.Result{
		Nodes: map[string]layout.NodeLayout{
			"A": {X: 100, Y: 100, Width: 60, Height: 40},
			"B": {X: 100, Y: 200, Width: 60, Height: 40},
		},
		Width: 200, Height: 300,
	}
	elems := renderSubgraphs(d, l, 10, DefaultTheme(), 16)
	outer, ok := elems[0].(*Group)
	if !ok {
		t.Fatalf("expected *Group, got %T", elems[0])
	}
	hasInner := false
	for _, child := range outer.Children {
		if g, ok := child.(*Group); ok && g.ID == "inner" {
			hasInner = true
		}
	}
	if !hasInner {
		t.Error("outer should contain inner group")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/renderer/flowchart/ -run "TestRenderSubgraph|TestSubgraphBBox|TestNestedSubgraph" -v`
Expected: FAIL — renderSubgraphs returns nil

- [ ] **Step 3: Replace subgraphs.go stub**

Replace entire contents of `pkg/renderer/flowchart/subgraphs.go`:

```go
package flowchart

import (
	"fmt"
	"math"

	"github.com/julianshen/mmgo/pkg/diagram"
	"github.com/julianshen/mmgo/pkg/layout"
)

type bbox struct {
	MinX, MinY, MaxX, MaxY float64
}

func subgraphBBox(nodes []diagram.Node, layoutNodes map[string]layout.NodeLayout) bbox {
	b := bbox{MinX: math.Inf(1), MinY: math.Inf(1), MaxX: math.Inf(-1), MaxY: math.Inf(-1)}
	for _, n := range nodes {
		nl, ok := layoutNodes[n.ID]
		if !ok {
			continue
		}
		left := nl.X - nl.Width/2
		right := nl.X + nl.Width/2
		top := nl.Y - nl.Height/2
		bottom := nl.Y + nl.Height/2
		if left < b.MinX {
			b.MinX = left
		}
		if right > b.MaxX {
			b.MaxX = right
		}
		if top < b.MinY {
			b.MinY = top
		}
		if bottom > b.MaxY {
			b.MaxY = bottom
		}
	}
	return b
}

func allDescendantNodes(sg *diagram.Subgraph) []diagram.Node {
	var nodes []diagram.Node
	nodes = append(nodes, sg.Nodes...)
	for i := range sg.Children {
		nodes = append(nodes, allDescendantNodes(&sg.Children[i])...)
	}
	return nodes
}

func renderSubgraphGroup(sg diagram.Subgraph, l *layout.Result, pad float64, th Theme, fontSize float64) *Group {
	allNodes := allDescendantNodes(&sg)
	bb := subgraphBBox(allNodes, l.Nodes)

	const sgPad = 15.0
	rx := bb.MinX - sgPad + pad
	ry := bb.MinY - sgPad + pad
	rw := bb.MaxX - bb.MinX + 2*sgPad
	rh := bb.MaxY - bb.MinY + 2*sgPad

	g := &Group{
		ID: sg.ID,
		Children: []any{
			&Rect{
				X: rx, Y: ry, Width: rw, Height: rh,
				RX: 5, RY: 5,
				Style: fmt.Sprintf("fill:%s;stroke:%s;stroke-width:1.5", th.SubgraphFill, th.SubgraphStroke),
			},
			&Text{
				X: rx + 10, Y: ry + 18,
				FontSize: fontSize,
				Style:    fmt.Sprintf("fill:%s;font-size:%gpx", th.SubgraphText, fontSize),
				Content:  sg.Label,
			},
		},
	}

	for i := range sg.Children {
		g.Children = append(g.Children, renderSubgraphGroup(sg.Children[i], l, pad, th, fontSize))
	}
	return g
}

func renderSubgraphs(d *diagram.FlowchartDiagram, l *layout.Result, pad float64, th Theme, fontSize float64) []any {
	var elems []any
	for _, sg := range d.Subgraphs {
		elems = append(elems, renderSubgraphGroup(sg, l, pad, th, fontSize))
	}
	return elems
}
```

- [ ] **Step 4: Run subgraph tests**

Run: `go test ./pkg/renderer/flowchart/ -run "TestRenderSubgraph|TestSubgraphBBox|TestNestedSubgraph" -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add pkg/renderer/flowchart/subgraphs.go pkg/renderer/flowchart/subgraphs_test.go
git commit -m "Add subgraph rendering with bounding box and nested groups"
```

---

### Task 7: Style/Class tests and golden file

**Files:**
- Modify: `pkg/renderer/flowchart/renderer_test.go`
- Create: `pkg/renderer/flowchart/testdata/simple.mmd`

- [ ] **Step 1: Add style/class tests**

Add to `pkg/renderer/flowchart/renderer_test.go`:

```go
func TestRenderStyledNode(t *testing.T) {
	d := &diagram.FlowchartDiagram{
		Nodes: []diagram.Node{
			{ID: "A", Label: "Styled", Shape: diagram.NodeShapeRectangle},
		},
		Styles: []diagram.StyleDef{
			{NodeID: "A", CSS: "fill:#ff0000;stroke:#00ff00"},
		},
	}
	g := graph.New()
	g.SetNode("A", graph.NodeAttrs{Label: "Styled", Width: 80, Height: 40})
	l := layout.Layout(g, layout.Options{})

	svgBytes, err := Render(d, l, nil)
	if err != nil {
		t.Fatalf("Render error: %v", err)
	}
	raw := string(svgBytes)
	if !strings.Contains(raw, "fill:#ff0000") {
		t.Errorf("styled node should have custom fill:\n%s", raw)
	}
}

func TestRenderClassNode(t *testing.T) {
	d := &diagram.FlowchartDiagram{
		Nodes: []diagram.Node{
			{ID: "A", Label: "Classy", Shape: diagram.NodeShapeRectangle, Classes: []string{"highlight"}},
		},
		Classes: map[string]string{"highlight": "fill:#ffff00"},
	}
	g := graph.New()
	g.SetNode("A", graph.NodeAttrs{Label: "Classy", Width: 80, Height: 40})
	l := layout.Layout(g, layout.Options{})

	svgBytes, err := Render(d, l, nil)
	if err != nil {
		t.Fatalf("Render error: %v", err)
	}
	raw := string(svgBytes)
	if !strings.Contains(raw, `class="highlight"`) {
		t.Errorf("class node should have class attr:\n%s", raw)
	}
	if !strings.Contains(raw, ".highlight") {
		t.Errorf("CSS should include class rule:\n%s", raw)
	}
}
```

- [ ] **Step 2: Run style tests**

Run: `go test ./pkg/renderer/flowchart/ -run "TestRenderStyledNode|TestRenderClassNode" -v`
Expected: PASS

- [ ] **Step 3: Create golden file testdata**

Create `pkg/renderer/flowchart/testdata/simple.mmd`:
```
graph TD
    A[Hello] --> B[World]
    B --> C[Test]
```

Add golden file test to `pkg/renderer/flowchart/renderer_test.go`:

```go
import (
	"flag"
	"os"
	"path/filepath"

	fcparser "github.com/julianshen/mmgo/pkg/parser/flowchart"
)

var updateGolden = flag.Bool("update", false, "update golden files")

func TestGoldenSimple(t *testing.T) {
	input, err := os.ReadFile(filepath.Join("testdata", "simple.mmd"))
	if err != nil {
		t.Skip("testdata/simple.mmd not found")
	}

	d, err := fcparser.Parse(strings.NewReader(string(input)))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	g := graph.New()
	for _, n := range d.Nodes {
		g.SetNode(n.ID, graph.NodeAttrs{Label: n.Label, Width: 100, Height: 50})
	}
	for _, e := range d.Edges {
		g.SetEdge(e.From, e.To, graph.EdgeAttrs{Label: e.Label})
	}
	l := layout.Layout(g, layout.Options{})

	svgBytes, err := Render(d, l, nil)
	if err != nil {
		t.Fatalf("Render error: %v", err)
	}

	goldenPath := filepath.Join("testdata", "simple.golden.svg")
	if *updateGolden {
		os.WriteFile(goldenPath, svgBytes, 0644)
		t.Logf("updated golden file: %s", goldenPath)
		return
	}

	golden, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("read golden: %v (run with -update to create)", err)
	}
	if string(svgBytes) != string(golden) {
		t.Errorf("output does not match golden file")
	}
}
```

- [ ] **Step 4: Generate golden file**

Run: `go test ./pkg/renderer/flowchart/ -run TestGoldenSimple -update -v`
Expected: creates `testdata/simple.golden.svg`

- [ ] **Step 5: Verify golden file test passes**

Run: `go test ./pkg/renderer/flowchart/ -run TestGoldenSimple -v`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add pkg/renderer/flowchart/renderer_test.go pkg/renderer/flowchart/testdata/
git commit -m "Add style/class tests and golden file test"
```

---

### Task 8: Full test suite and coverage

- [ ] **Step 1: Run full test suite with coverage**

Run: `go test ./pkg/renderer/flowchart/ -v -cover`
Expected: All tests PASS, coverage >80%

- [ ] **Step 2: Run full project tests**

Run: `go test ./... -race -cover`
Expected: All tests PASS

- [ ] **Step 3: Run linter**

Run: `golangci-lint run ./pkg/renderer/flowchart/...`
Expected: No issues

- [ ] **Step 4: Update STATUS.md**

Update `docs/STATUS.md` to mark Step 10 complete:

Change:
```
| ⏳ | 10. Flowchart renderer (`pkg/renderer/flowchart/`) | — |  |  |
```
To:
```
| ✅ | 10. Flowchart renderer (`pkg/renderer/flowchart/`) | #14 | 95%+ | All 14 shapes, bezier curves, markers, subgraphs, style/class |
```

Update overall counts and next step.

- [ ] **Step 5: Final commit**

```bash
git add docs/STATUS.md
git commit -m "Mark Step 10 (flowchart renderer) complete in STATUS"
```
