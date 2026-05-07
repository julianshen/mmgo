package sankey

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"strings"
	"testing"

	"github.com/julianshen/mmgo/pkg/diagram"
	"github.com/julianshen/mmgo/pkg/textmeasure"
)

func TestRenderNilDiagram(t *testing.T) {
	if _, err := Render(nil, nil); err == nil {
		t.Fatal("expected error")
	}
}

func TestRenderEmptyDiagram(t *testing.T) {
	d := &diagram.SankeyDiagram{}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	assertValidSVG(t, out)
}

func TestRenderSingleFlow(t *testing.T) {
	d := &diagram.SankeyDiagram{
		Flows: []diagram.SankeyFlow{
			{Source: "A", Target: "B", Value: 10},
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	if !strings.Contains(raw, ">A ") || !strings.Contains(raw, ">B ") {
		t.Error("node labels missing")
	}
	if !strings.Contains(raw, "<path") {
		t.Error("ribbon path missing")
	}
	if !strings.Contains(raw, "<rect") {
		t.Error("node bar missing")
	}
	assertValidSVG(t, out)
}

func TestRenderMultiColumnChain(t *testing.T) {
	d := &diagram.SankeyDiagram{
		Flows: []diagram.SankeyFlow{
			{Source: "A", Target: "B", Value: 10},
			{Source: "B", Target: "C", Value: 5},
			{Source: "B", Target: "D", Value: 5},
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	for _, n := range []string{"A", "B", "C", "D"} {
		if !strings.Contains(raw, ">"+n+" ") {
			t.Errorf("label %q missing", n)
		}
	}
	assertValidSVG(t, out)
}

func TestRenderFanIn(t *testing.T) {
	d := &diagram.SankeyDiagram{
		Flows: []diagram.SankeyFlow{
			{Source: "A", Target: "C", Value: 3},
			{Source: "B", Target: "C", Value: 7},
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	// Two ribbons expected.
	if n := strings.Count(string(out), "<path"); n != 2 {
		t.Errorf("ribbon count = %d, want 2", n)
	}
	assertValidSVG(t, out)
}

func TestRenderDeterministic(t *testing.T) {
	d := &diagram.SankeyDiagram{
		Flows: []diagram.SankeyFlow{
			{Source: "A", Target: "B", Value: 10},
			{Source: "A", Target: "C", Value: 5},
			{Source: "B", Target: "D", Value: 8},
			{Source: "C", Target: "D", Value: 4},
		},
	}
	first, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	for i := 0; i < 10; i++ {
		next, err := Render(d, nil)
		if err != nil {
			t.Fatalf("iter %d: %v", i, err)
		}
		if string(next) != string(first) {
			t.Fatalf("iter %d: output diverges", i)
		}
	}
}

// Regression: labels on non-rightmost columns anchor leftward from
// their node bar. A long label on the leftmost column used to extend
// into negative X and clip outside the viewBox. viewW must now grow
// to include left-side label padding, and the first node's x must be
// shifted right so the label fits.
func TestRenderLongLeftLabelFitsInViewBox(t *testing.T) {
	short := &diagram.SankeyDiagram{
		Flows: []diagram.SankeyFlow{{Source: "A", Target: "B", Value: 1}},
	}
	long := &diagram.SankeyDiagram{
		Flows: []diagram.SankeyFlow{{Source: "A very long label on the left", Target: "B", Value: 1}},
	}
	sOut, _ := Render(short, nil)
	lOut, _ := Render(long, nil)
	sw := svgViewBoxWidth(t, sOut)
	lw := svgViewBoxWidth(t, lOut)
	if lw <= sw {
		t.Errorf("viewBox width did not grow for long left label (short=%.2f long=%.2f)", sw, lw)
	}
	// Long left label must fit before x=0: its anchor has to sit at
	// least one EstimateWidth past the origin or the renderer forgot
	// to shift originX right.
	firstAnchorX := firstLeftLabelX(string(lOut))
	minAnchorX := textmeasure.EstimateWidth("A very long label on the left", defaultFontSize)
	if firstAnchorX < minAnchorX {
		t.Errorf("first label anchor X = %.2f, want >= %.2f so label fits in viewBox",
			firstAnchorX, minAnchorX)
	}
}

func svgViewBoxWidth(t *testing.T, svgBytes []byte) float64 {
	t.Helper()
	body := svgBytes
	if i := bytes.Index(body, []byte("<svg")); i >= 0 {
		body = body[i:]
	}
	var doc struct {
		XMLName xml.Name `xml:"svg"`
		ViewBox string   `xml:"viewBox,attr"`
	}
	if err := xml.Unmarshal(body, &doc); err != nil {
		t.Fatalf("invalid SVG: %v", err)
	}
	var x0, y0, w, h float64
	if _, err := fmt.Sscanf(doc.ViewBox, "%f %f %f %f", &x0, &y0, &w, &h); err != nil {
		t.Fatalf("viewBox parse: %v", err)
	}
	return w
}

// firstLeftLabelX returns the x attribute of the first <text ...
// text-anchor="end"> element — a left-anchored label.
func firstLeftLabelX(raw string) float64 {
	i := strings.Index(raw, `text-anchor="end"`)
	if i < 0 {
		return 0
	}
	start := strings.LastIndex(raw[:i], "<text")
	if start < 0 {
		return 0
	}
	// Find x="..." inside this element.
	xIdx := strings.Index(raw[start:i], ` x="`)
	if xIdx < 0 {
		return 0
	}
	xIdx += start + len(` x="`)
	end := strings.Index(raw[xIdx:], `"`)
	if end < 0 {
		return 0
	}
	var v float64
	_, _ = fmt.Sscanf(raw[xIdx:xIdx+end], "%f", &v)
	return v
}

func TestRenderCustomFontSize(t *testing.T) {
	d := &diagram.SankeyDiagram{
		Flows: []diagram.SankeyFlow{{Source: "A", Target: "B", Value: 1}},
	}
	out, err := Render(d, &Options{FontSize: 20})
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if !strings.Contains(string(out), "font-size:20px") {
		t.Error("custom font size not applied")
	}
}

func TestRenderCycleBoundedIterations(t *testing.T) {
	// A -> B -> A creates a cycle. Render must terminate (iteration
	// cap) and still produce a valid SVG with both labels and both
	// ribbons visible.
	d := &diagram.SankeyDiagram{
		Flows: []diagram.SankeyFlow{
			{Source: "A", Target: "B", Value: 1},
			{Source: "B", Target: "A", Value: 1},
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	if !strings.Contains(raw, ">A ") || !strings.Contains(raw, ">B ") {
		t.Error("cycle labels missing")
	}
	if n := strings.Count(raw, "<path"); n != 2 {
		t.Errorf("ribbon count = %d, want 2", n)
	}
	assertValidSVG(t, out)
}

func TestRenderRibbonColorMatchesSourceBar(t *testing.T) {
	// Regression: assignColumns previously sorted `nodes` in place,
	// which split nodeIdx (pre-sort) from the palette index used by
	// node bars (post-sort). Guard: the palette color of node A must
	// be present in both its bar fill and in the fill of every flow
	// originating from A.
	d := &diagram.SankeyDiagram{
		Flows: []diagram.SankeyFlow{
			{Source: "A", Target: "B", Value: 1},
			{Source: "B", Target: "C", Value: 1},
			{Source: "A", Target: "C", Value: 1},
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	colorA := DefaultTheme().NodeColors[0] // A is the first-appearance node
	// The color must appear in at least 3 fills (A's bar + 2 ribbons).
	if strings.Count(raw, "fill:"+colorA) < 3 {
		t.Errorf("A's color %s should be used for A's bar and both outgoing ribbons; saw %d occurrences\n%s",
			colorA, strings.Count(raw, "fill:"+colorA), raw)
	}
}

func assertValidSVG(t *testing.T, svgBytes []byte) {
	t.Helper()
	body := svgBytes
	if i := bytes.Index(body, []byte("<svg")); i >= 0 {
		body = body[i:]
	}
	var doc struct {
		XMLName xml.Name `xml:"svg"`
		ViewBox string   `xml:"viewBox,attr"`
	}
	if err := xml.Unmarshal(body, &doc); err != nil {
		t.Fatalf("invalid SVG: %v\n%s", err, svgBytes)
	}
	if doc.ViewBox == "" {
		t.Error("viewBox missing")
	}
}

// AccTitle/AccDescr emit as <title>/<desc>; Title renders as a
// centered caption above the diagram body.
func TestRenderSankeyHeader(t *testing.T) {
	d := &diagram.SankeyDiagram{
		Title:    "Energy",
		AccTitle: "Energy budget",
		AccDescr: "Q1 power flows",
		Flows: []diagram.SankeyFlow{
			{Source: "Coal", Target: "Power", Value: 5},
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	if !strings.Contains(raw, "<title>Energy budget</title>") {
		t.Errorf("expected accTitle <title> in:\n%s", raw)
	}
	if !strings.Contains(raw, "<desc>Q1 power flows</desc>") {
		t.Errorf("expected accDescr <desc> in:\n%s", raw)
	}
	if !strings.Contains(raw, ">Energy<") {
		t.Errorf("expected diagram title rendered in:\n%s", raw)
	}
}

// LinkColor=Target paints each ribbon with the target node's
// palette color rather than the source's.
func TestRenderSankeyLinkColorTarget(t *testing.T) {
	d := &diagram.SankeyDiagram{
		Flows: []diagram.SankeyFlow{
			{Source: "A", Target: "B", Value: 5},
			{Source: "A", Target: "C", Value: 3},
		},
	}
	out, err := Render(d, &Options{LinkColor: LinkColorTarget})
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	// Source-mode would emit two ribbons with the same color
	// (both flow from A); target-mode emits the colors of B and C
	// — i.e. the second and third palette entries. Just confirm
	// at least two distinct colors appear in ribbon fills.
	raw := string(out)
	th := DefaultTheme()
	bColor := th.NodeColors[1]
	cColor := th.NodeColors[2]
	if !strings.Contains(raw, "fill:"+bColor) {
		t.Errorf("expected target-color %s in output", bColor)
	}
	if !strings.Contains(raw, "fill:"+cColor) {
		t.Errorf("expected target-color %s in output", cColor)
	}
}

// LinkColor=Hex paints every ribbon with the literal hex.
func TestRenderSankeyLinkColorHex(t *testing.T) {
	d := &diagram.SankeyDiagram{
		Flows: []diagram.SankeyFlow{
			{Source: "A", Target: "B", Value: 5},
		},
	}
	out, err := Render(d, &Options{LinkColor: LinkColorHex, LinkColorHex: "#abcdef"})
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if !strings.Contains(string(out), "fill:#abcdef") {
		t.Errorf("expected literal hex fill")
	}
}

// LinkColor=Gradient emits a <linearGradient> def per flow and
// references it via `url(#sankey-grad-N)`.
func TestRenderSankeyLinkColorGradient(t *testing.T) {
	d := &diagram.SankeyDiagram{
		Flows: []diagram.SankeyFlow{
			{Source: "A", Target: "B", Value: 5},
		},
	}
	out, err := Render(d, &Options{LinkColor: LinkColorGradient})
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	if !strings.Contains(raw, `<linearGradient id="sankey-grad-0"`) {
		t.Errorf("expected gradient def in:\n%s", raw)
	}
	if !strings.Contains(raw, "url(#sankey-grad-0)") {
		t.Errorf("expected gradient reference in:\n%s", raw)
	}
}

// ShowValues=false suppresses the magnitude in node labels.
func TestRenderSankeyShowValuesFalse(t *testing.T) {
	d := &diagram.SankeyDiagram{
		Flows: []diagram.SankeyFlow{
			{Source: "Source", Target: "Sink", Value: 7},
		},
	}
	no := false
	out, err := Render(d, &Options{ShowValues: &no})
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	if strings.Contains(raw, ">Source 7") || strings.Contains(raw, ">Sink 7") {
		t.Errorf("ShowValues=false should suppress magnitudes:\n%s", raw)
	}
	if !strings.Contains(raw, ">Source<") || !strings.Contains(raw, ">Sink<") {
		t.Errorf("expected bare labels in:\n%s", raw)
	}
}

// Prefix and Suffix wrap the formatted magnitude when ShowValues
// is on.
func TestRenderSankeyPrefixSuffix(t *testing.T) {
	d := &diagram.SankeyDiagram{
		Flows: []diagram.SankeyFlow{
			{Source: "A", Target: "B", Value: 5},
		},
	}
	out, err := Render(d, &Options{Prefix: "$", Suffix: "k"})
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if !strings.Contains(string(out), "$5k") {
		t.Errorf("expected prefix/suffix wrap in output:\n%s", out)
	}
}

// Each NodeAlignmentMode is verified directly against assignColumns:
// for the 3-node graph A→B→C and a fanout A→D, the four strategies
// should put each node in a known column.
//
//	A → B → C
//	A → D       (D is a sink; B is intermediate; C is a sink)
//
// Left:    A=0, B=1, D=1, C=2
// Right:   A=0, B=1, D=2, C=2  (D pulled to maxCol — it's a sink)
// Center:  A=0, B=1, D=1, C=2  (avg of left/right)
// Justify: A=0, B=1, D=2, C=2  (sinks pinned right; D and C are sinks)
func TestAssignColumnsByMode(t *testing.T) {
	nodes := []string{"A", "B", "C", "D"}
	flows := []diagram.SankeyFlow{
		{Source: "A", Target: "B", Value: 1},
		{Source: "B", Target: "C", Value: 1},
		{Source: "A", Target: "D", Value: 1},
	}
	cases := []struct {
		mode NodeAlignmentMode
		want map[string]int
	}{
		{NodeAlignLeft, map[string]int{"A": 0, "B": 1, "D": 1, "C": 2}},
		{NodeAlignRight, map[string]int{"A": 0, "B": 1, "D": 2, "C": 2}},
		{NodeAlignCenter, map[string]int{"A": 0, "B": 1, "D": 1, "C": 2}},
		{NodeAlignJustify, map[string]int{"A": 0, "B": 1, "D": 2, "C": 2}},
	}
	for _, tc := range cases {
		t.Run(modeName(tc.mode), func(t *testing.T) {
			got, _ := assignColumns(nodes, flows, tc.mode)
			for n, want := range tc.want {
				if got[n] != want {
					t.Errorf("mode=%v node %q col=%d, want %d (got=%v)", tc.mode, n, got[n], want, got)
				}
			}
		})
	}
}

func modeName(m NodeAlignmentMode) string {
	switch m {
	case NodeAlignLeft:
		return "Left"
	case NodeAlignRight:
		return "Right"
	case NodeAlignCenter:
		return "Center"
	case NodeAlignJustify:
		return "Justify"
	}
	return "?"
}

// End-to-end: changing NodeAlignment changes the rendered SVG.
// Graph: A→B→C→D and B→E, where E is a depth-2 sink (maxCol=3).
// Left leaves E at col 2; Justify pins it to maxCol=3. An inert
// dispatch would emit identical output for both modes.
func TestRenderNodeAlignmentDispatchesToLayout(t *testing.T) {
	d := &diagram.SankeyDiagram{
		Flows: []diagram.SankeyFlow{
			{Source: "A", Target: "B", Value: 1},
			{Source: "B", Target: "C", Value: 1},
			{Source: "C", Target: "D", Value: 1},
			{Source: "B", Target: "E", Value: 1},
		},
	}
	render := func(mode NodeAlignmentMode) string {
		out, err := Render(d, &Options{NodeAlignment: mode})
		if err != nil {
			t.Fatalf("Render(%v): %v", mode, err)
		}
		return string(out)
	}
	if render(NodeAlignLeft) == render(NodeAlignJustify) {
		t.Errorf("Left and Justify must produce different SVG when a non-max-depth sink exists")
	}
}
