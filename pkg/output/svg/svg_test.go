package svg

import (
	"encoding/xml"
	"errors"
	"flag"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/julianshen/mmgo/pkg/diagram"
	"github.com/julianshen/mmgo/pkg/layout"
	"github.com/julianshen/mmgo/pkg/textmeasure"
)

// --- Nil / error paths ---

func TestRenderNilReader(t *testing.T) {
	_, err := Render(nil, nil)
	if err == nil {
		t.Fatal("expected error for nil reader")
	}
	if !strings.Contains(err.Error(), "reader is nil") {
		t.Errorf("error = %v", err)
	}
}

func TestRenderEmptyInput(t *testing.T) {
	_, err := Render(strings.NewReader(""), nil)
	if err == nil {
		t.Fatal("expected error for empty input")
	}
	if !strings.Contains(err.Error(), "no diagram header") {
		t.Errorf("error = %v", err)
	}
}

func TestRenderUnknownDiagramKind(t *testing.T) {
	_, err := Render(strings.NewReader("sequenceDiagram\nA->>B: hi"), nil)
	if err == nil {
		t.Fatal("expected error for unsupported diagram type")
	}
	if !strings.Contains(err.Error(), "unrecognized diagram header") {
		t.Errorf("error = %v", err)
	}
}

// --- Diagram detection ---

func TestDetectDiagramKind(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  diagramKind
		err   bool
	}{
		{"graph", "graph LR\nA --> B", kindFlowchart, false},
		{"flowchart", "flowchart TB\nA --> B", kindFlowchart, false},
		{"with leading comment", "%% header comment\ngraph LR\nA --> B", kindFlowchart, false},
		{"with leading blank lines", "\n\n\ngraph LR\nA --> B", kindFlowchart, false},
		{"unknown header", "stateDiagram\nA --> B", kindUnknown, true},
		{"empty", "", kindUnknown, true},
		{"only comments", "%% one\n%% two\n", kindUnknown, true},
		// Word-boundary check: `grapha` must not match `graph`.
		{"grapha not matched", "grapha LR\nA --> B", kindUnknown, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := detectDiagramKind([]byte(tc.input))
			if tc.err {
				if err == nil {
					t.Errorf("expected error, got %v", got)
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Errorf("kind = %v, want %v", got, tc.want)
			}
		})
	}
}

// --- End-to-end smoke ---

func TestRenderSimpleFlowchartProducesValidSVG(t *testing.T) {
	out, err := Render(strings.NewReader("graph LR\n    A[Start] --> B[End]"), nil)
	if err != nil {
		t.Fatalf("Render err: %v", err)
	}
	raw := string(out)
	if !strings.HasPrefix(raw, "<?xml") {
		t.Errorf("output should start with XML decl, got: %q", raw[:min(60, len(raw))])
	}

	// The output must round-trip through encoding/xml as a valid
	// document — catches any malformed bytes the renderer emits.
	xmlStart := strings.Index(raw, "<svg")
	if xmlStart < 0 {
		t.Fatalf("no <svg> element:\n%s", raw)
	}
	var doc struct {
		XMLName xml.Name `xml:"svg"`
		ViewBox string   `xml:"viewBox,attr"`
	}
	if err := xml.Unmarshal([]byte(raw[xmlStart:]), &doc); err != nil {
		t.Fatalf("invalid XML: %v\n%s", err, raw)
	}
	if doc.ViewBox == "" {
		t.Error("viewBox attribute missing")
	}
	if !strings.Contains(raw, ">Start<") || !strings.Contains(raw, ">End<") {
		t.Errorf("node labels missing from output:\n%s", raw)
	}
}

func TestRenderHonorsDirectionHeader(t *testing.T) {
	// `graph LR` should produce a wider-than-tall layout (LR puts
	// nodes on the horizontal axis), and `graph TB` should be
	// taller-than-wide. We assert the relationship between viewBox
	// width and height to verify direction was honored end-to-end.
	lr, err := Render(strings.NewReader("graph LR\n    A --> B --> C --> D"), nil)
	if err != nil {
		t.Fatalf("LR render: %v", err)
	}
	tb, err := Render(strings.NewReader("graph TB\n    A --> B --> C --> D"), nil)
	if err != nil {
		t.Fatalf("TB render: %v", err)
	}

	lrW, lrH := parseViewBox(t, lr)
	tbW, tbH := parseViewBox(t, tb)

	if !(lrW > lrH) {
		t.Errorf("LR viewBox should be wider than tall: w=%v h=%v", lrW, lrH)
	}
	if !(tbH > tbW) {
		t.Errorf("TB viewBox should be taller than wide: w=%v h=%v", tbW, tbH)
	}
}

func TestRenderOptsLayoutOverridesDirection(t *testing.T) {
	// If the caller pins a non-default RankDir, it must override the
	// header's direction. This protects callers from surprise when
	// they explicitly opt in to a layout direction.
	out, err := Render(
		strings.NewReader("graph LR\n    A --> B"),
		&Options{Layout: layout.Options{RankDir: layout.RankDirBT}},
	)
	if err != nil {
		t.Fatalf("Render err: %v", err)
	}
	w, h := parseViewBox(t, out)
	// BT is vertical (taller than wide for a 2-node chain).
	if !(h > w) {
		t.Errorf("BT viewBox should be taller than wide: w=%v h=%v", w, h)
	}
}

// (Subgraph parser support is still on the TODO list — see
// pkg/parser/flowchart/parser.go. The renderer's subgraph code is
// already covered by pkg/renderer/flowchart/subgraphs_test.go using
// in-memory ASTs. Once the parser learns `subgraph ... end`, add a
// subgraph fixture here.)

// --- buildFlowchartGraph unit tests ---

func TestBuildFlowchartGraphIncludesNestedNodes(t *testing.T) {
	d := &diagram.FlowchartDiagram{
		Nodes: []diagram.Node{{ID: "Top", Label: "Top"}},
		Subgraphs: []diagram.Subgraph{
			{
				ID:    "sg",
				Nodes: []diagram.Node{{ID: "Nested", Label: "Nested"}},
				Edges: []diagram.Edge{{From: "Nested", To: "Top"}},
				Children: []diagram.Subgraph{
					{ID: "inner", Nodes: []diagram.Node{{ID: "Deep", Label: "Deep"}}},
				},
			},
		},
	}
	ruler := mustRuler(t)
	defer ruler.Close()
	g := buildFlowchartGraph(d, ruler, defaultFontSize)

	for _, want := range []string{"Top", "Nested", "Deep"} {
		if !g.HasNode(want) {
			t.Errorf("graph missing node %q", want)
		}
	}
	if !g.HasEdge("Nested", "Top") {
		t.Error("graph missing nested edge Nested→Top")
	}
}

func TestNodeSizeRespectsMinimum(t *testing.T) {
	ruler := mustRuler(t)
	defer ruler.Close()
	w, h := nodeSize("", ruler, defaultFontSize)
	if w < minNodeWidth || h < minNodeHeight {
		t.Errorf("empty label should clamp to minimum: w=%v h=%v", w, h)
	}
	// A long label must produce a proportionally wider box.
	wLong, _ := nodeSize("a very wide label that should expand the box", ruler, defaultFontSize)
	if wLong <= minNodeWidth {
		t.Errorf("long label width %v should exceed min %v", wLong, minNodeWidth)
	}
}

func TestDirectionToRankDir(t *testing.T) {
	cases := []struct {
		d    diagram.Direction
		want layout.RankDir
	}{
		{diagram.DirectionTB, layout.RankDirTB},
		{diagram.DirectionBT, layout.RankDirBT},
		{diagram.DirectionLR, layout.RankDirLR},
		{diagram.DirectionRL, layout.RankDirRL},
	}
	for _, tc := range cases {
		if got := directionToRankDir(tc.d); got != tc.want {
			t.Errorf("directionToRankDir(%v) = %v, want %v", tc.d, got, tc.want)
		}
	}
}

// --- Error paths and option forwarding ---

func TestRenderParseError(t *testing.T) {
	// Unclosed bracket — parser should error and Render should
	// surface that with a "parse:" prefix.
	_, err := Render(strings.NewReader("graph LR\n    A[unclosed"), nil)
	if err == nil {
		t.Fatal("expected parse error")
	}
	if !strings.Contains(err.Error(), "parse") {
		t.Errorf("error should mention parse: %v", err)
	}
}

func TestRenderReaderError(t *testing.T) {
	// io.ReadAll error path: a reader that always errors.
	_, err := Render(errReader{}, nil)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "read input") {
		t.Errorf("error should mention read input: %v", err)
	}
}

func TestRenderHonorsCustomFontSize(t *testing.T) {
	// A larger font should produce larger node boxes (since width is
	// driven by the measured label width).
	defaultOut, err := Render(strings.NewReader("graph LR\n    A[Hello]"), nil)
	if err != nil {
		t.Fatalf("default: %v", err)
	}
	bigOut, err := Render(
		strings.NewReader("graph LR\n    A[Hello]"),
		&Options{FontSize: 48},
	)
	if err != nil {
		t.Fatalf("big: %v", err)
	}
	dw, _ := parseViewBox(t, defaultOut)
	bw, _ := parseViewBox(t, bigOut)
	if !(bw > dw) {
		t.Errorf("48px font viewBox %v should exceed default %v", bw, dw)
	}
}

// errReader returns an error on the first Read so we can exercise the
// io.ReadAll error path in Render.
type errReader struct{}

func (errReader) Read(p []byte) (int, error) {
	return 0, errIOFailure
}

var errIOFailure = errors.New("forced read failure")

// --- Determinism ---

func TestRenderDeterministic(t *testing.T) {
	// Same input must produce byte-identical output across many
	// invocations. Catches any new map-iteration leaks.
	input := `graph LR
    A[Start] --> B{Check}
    B -->|yes| C[Ok]
    B -->|no| D[Fail]`
	first, err := Render(strings.NewReader(input), nil)
	if err != nil {
		t.Fatalf("Render err: %v", err)
	}
	for i := 0; i < 20; i++ {
		next, err := Render(strings.NewReader(input), nil)
		if err != nil {
			t.Fatalf("iter %d: %v", i, err)
		}
		if string(next) != string(first) {
			t.Fatalf("iter %d: output diverges", i)
		}
	}
}

// --- Golden fixtures ---

var updateGolden = flag.Bool("update", false, "update golden files in testdata/")

func TestGoldenFiles(t *testing.T) {
	entries, err := os.ReadDir("testdata")
	if err != nil {
		t.Skipf("testdata directory missing: %v", err)
	}
	for _, ent := range entries {
		name := ent.Name()
		if !strings.HasSuffix(name, ".mmd") {
			continue
		}
		t.Run(strings.TrimSuffix(name, ".mmd"), func(t *testing.T) {
			input, err := os.ReadFile(filepath.Join("testdata", name))
			if err != nil {
				t.Fatalf("read input: %v", err)
			}
			got, err := Render(strings.NewReader(string(input)), nil)
			if err != nil {
				t.Fatalf("Render err: %v", err)
			}
			goldenPath := filepath.Join("testdata", strings.TrimSuffix(name, ".mmd")+".golden.svg")
			if *updateGolden {
				if err := os.WriteFile(goldenPath, got, 0o644); err != nil {
					t.Fatalf("write golden: %v", err)
				}
				t.Logf("updated %s", goldenPath)
				return
			}
			golden, err := os.ReadFile(goldenPath)
			if err != nil {
				t.Fatalf("read golden: %v (run with -update to create)", err)
			}
			if string(got) != string(golden) {
				t.Errorf("output differs from %s\n--- got ---\n%s\n--- want ---\n%s",
					goldenPath, got, golden)
			}
		})
	}
}

// --- Helpers ---

func mustRuler(t *testing.T) *textmeasure.Ruler {
	t.Helper()
	ruler, err := textmeasure.NewDefaultRuler()
	if err != nil {
		t.Fatalf("ruler init: %v", err)
	}
	return ruler
}

// parseViewBox extracts the width and height from the SVG viewBox
// attribute ("0 0 W H"). Used by direction-honoring tests to assert
// orientation without comparing exact coordinates.
func parseViewBox(t *testing.T, svgBytes []byte) (w, h float64) {
	t.Helper()
	raw := string(svgBytes)
	i := strings.Index(raw, `viewBox="`)
	if i < 0 {
		t.Fatalf("no viewBox attribute in:\n%s", raw)
	}
	i += len(`viewBox="`)
	end := strings.Index(raw[i:], `"`)
	if end < 0 {
		t.Fatalf("unterminated viewBox attribute")
	}
	parts := strings.Fields(raw[i : i+end])
	if len(parts) != 4 {
		t.Fatalf("viewBox should have 4 fields, got %d: %q", len(parts), parts)
	}
	pw, err := strconv.ParseFloat(parts[2], 64)
	if err != nil {
		t.Fatalf("parse viewBox width: %v", err)
	}
	ph, err := strconv.ParseFloat(parts[3], 64)
	if err != nil {
		t.Fatalf("parse viewBox height: %v", err)
	}
	return pw, ph
}

