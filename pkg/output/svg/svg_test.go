package svg

import (
	"bytes"
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
	flowchartrenderer "github.com/julianshen/mmgo/pkg/renderer/flowchart"
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
	// Use a clearly bogus header so this test stays valid even after
	// sequence/pie/etc. headers are recognized.
	_, err := Render(strings.NewReader("fooDiagram\nA --> B"), nil)
	if err == nil {
		t.Fatal("expected error for unsupported diagram type")
	}
	if !strings.Contains(err.Error(), "unrecognized diagram header") {
		t.Errorf("error = %v", err)
	}
}

func TestRenderParseError(t *testing.T) {
	_, err := Render(strings.NewReader("graph LR\n    A[unclosed"), nil)
	if err == nil {
		t.Fatal("expected parse error")
	}
	if !strings.Contains(err.Error(), "parse") {
		t.Errorf("error should mention parse: %v", err)
	}
}

func TestRenderReaderError(t *testing.T) {
	_, err := Render(errReader{}, nil)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "read input") {
		t.Errorf("error should mention read input: %v", err)
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
		{"unknown header", "fooDiagram\nA --> B", kindUnknown, true},
		{"empty", "", kindUnknown, true},
		{"only comments", "%% one\n%% two\n", kindUnknown, true},
		// Word-boundary check: `grapha` must not match `graph`.
		{"grapha not matched", "grapha LR\nA --> B", kindUnknown, true},
		{"sequenceDiagram", "sequenceDiagram\n    A->>B: hi", kindSequence, false},
		{"pie", "pie title Pets\n    \"Dogs\" : 10", kindPie, false},
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
	doc := unmarshalSVG(t, out)
	if doc.ViewBox == "" {
		t.Error("viewBox attribute missing")
	}
	if !strings.Contains(raw, ">Start<") || !strings.Contains(raw, ">End<") {
		t.Errorf("node labels missing from output:\n%s", raw)
	}
}

func TestRenderHonorsDirectionHeader(t *testing.T) {
	// `graph LR` should produce a wider-than-tall layout, and `graph
	// TB` should be taller-than-wide. Asserting the relationship
	// between viewBox width and height verifies direction was honored
	// end-to-end without coupling to exact coordinates.
	lr, err := Render(strings.NewReader("graph LR\n    A --> B --> C --> D"), nil)
	if err != nil {
		t.Fatalf("LR render: %v", err)
	}
	tb, err := Render(strings.NewReader("graph TB\n    A --> B --> C --> D"), nil)
	if err != nil {
		t.Fatalf("TB render: %v", err)
	}

	lrW, lrH := viewBoxWH(t, lr)
	tbW, tbH := viewBoxWH(t, tb)

	if !(lrW > lrH) {
		t.Errorf("LR viewBox should be wider than tall: w=%v h=%v", lrW, lrH)
	}
	if !(tbH > tbW) {
		t.Errorf("TB viewBox should be taller than wide: w=%v h=%v", tbW, tbH)
	}
}

func TestRenderIgnoresCallerRankDir(t *testing.T) {
	// Direction always comes from the diagram header; an
	// opts.Layout.RankDir value must NOT override it. This pins the
	// "header is source of truth" contract.
	out, err := Render(
		strings.NewReader("graph LR\n    A --> B --> C --> D"),
		&Options{Layout: layout.Options{RankDir: layout.RankDirBT}},
	)
	if err != nil {
		t.Fatalf("Render err: %v", err)
	}
	w, h := viewBoxWH(t, out)
	// LR should still be wider than tall — caller's BT was ignored.
	if !(w > h) {
		t.Errorf("expected LR layout regardless of RankDir override: w=%v h=%v", w, h)
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
	g := buildFlowchartGraph(d, ruler, flowchartrenderer.DefaultFontSize)

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
	w, h := nodeSize("", ruler, flowchartrenderer.DefaultFontSize)
	if w < minNodeWidth || h < minNodeHeight {
		t.Errorf("empty label should clamp to minimum: w=%v h=%v", w, h)
	}
	wLong, _ := nodeSize("a very wide label that should expand the box", ruler, flowchartrenderer.DefaultFontSize)
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

// --- Font size threading ---

func TestRenderHonorsFlowchartFontSize(t *testing.T) {
	// FontSize set on the flowchart sub-options must drive both node
	// sizing AND the rendered text. A larger font produces a larger
	// viewBox because nodes grow with the measured label.
	defaultOut, err := Render(strings.NewReader("graph LR\n    A[Hello]"), nil)
	if err != nil {
		t.Fatalf("default: %v", err)
	}
	bigOut, err := Render(
		strings.NewReader("graph LR\n    A[Hello]"),
		&Options{Flowchart: &flowchartrenderer.Options{FontSize: 48}},
	)
	if err != nil {
		t.Fatalf("big: %v", err)
	}
	dw, _ := viewBoxWH(t, defaultOut)
	bw, _ := viewBoxWH(t, bigOut)
	if !(bw > dw) {
		t.Errorf("48px font viewBox %v should exceed default %v", bw, dw)
	}
	// And the rendered text must also be 48px (font-size in the
	// emitted style attribute) — this is what the previous
	// top-level FontSize bug missed.
	if !strings.Contains(string(bigOut), "font-size:48.00px") {
		t.Errorf("expected font-size:48px in output, got:\n%s", bigOut)
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

func TestRenderSequenceDiagramEndToEnd(t *testing.T) {
	input := `sequenceDiagram
    participant Alice
    participant Bob
    Alice->>Bob: Hello
    Bob-->>Alice: Hi back`
	out, err := Render(strings.NewReader(input), nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	if !strings.HasPrefix(raw, "<?xml") {
		t.Errorf("output should start with XML decl")
	}
	doc := unmarshalSVG(t, out)
	if doc.ViewBox == "" {
		t.Error("viewBox missing")
	}
	if !strings.Contains(raw, ">Alice<") || !strings.Contains(raw, ">Bob<") {
		t.Error("participant labels missing")
	}
	if !strings.Contains(raw, ">Hello<") || !strings.Contains(raw, ">Hi back<") {
		t.Error("message labels missing")
	}
}

func TestRenderPieDiagramEndToEnd(t *testing.T) {
	input := `pie title Pets
    "Dogs" : 386
    "Cats" : 85
    "Rats" : 15`
	out, err := Render(strings.NewReader(input), nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	if !strings.HasPrefix(raw, "<?xml") {
		t.Error("output should start with XML decl")
	}
	doc := unmarshalSVG(t, out)
	if doc.ViewBox == "" {
		t.Error("viewBox missing")
	}
	if !strings.Contains(raw, ">Pets<") {
		t.Error("title missing")
	}
	if !strings.Contains(raw, "<path") {
		t.Error("arc paths missing")
	}
}

// --- Init directives and config ---

func TestExtractInitDirective(t *testing.T) {
	cases := []struct {
		name    string
		input   string
		wantSrc string
		wantCfg string
	}{
		{
			"no directive",
			"graph LR\nA --> B",
			"graph LR\nA --> B",
			"",
		},
		{
			"single directive",
			"%%{init: {\"theme\": \"dark\"}}%%\ngraph LR\nA --> B",
			"graph LR\nA --> B",
			`{"theme": "dark"}`,
		},
		{
			"directive with whitespace",
			"  %%{init:   {\"theme\":\"forest\"}  }%%  \ngraph LR",
			"graph LR",
			`{"theme":"forest"}`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			src, cfg := extractInitDirective([]byte(tc.input))
			if string(src) != tc.wantSrc {
				t.Errorf("src = %q, want %q", src, tc.wantSrc)
			}
			if tc.wantCfg == "" {
				if cfg != nil {
					t.Errorf("cfg should be nil, got %+v", cfg)
				}
			} else {
				if cfg == nil {
					t.Fatal("cfg should not be nil")
				}
			}
		})
	}
}

func TestRenderWithInitDirectiveTheme(t *testing.T) {
	input := `%%{init: {"theme": "dark"}}%%
graph LR
    A[Start] --> B[End]`
	out, err := Render(strings.NewReader(input), nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	doc := unmarshalSVG(t, out)
	if doc.ViewBox == "" {
		t.Error("viewBox missing")
	}
	// Dark theme background should not be white.
	if strings.Contains(raw, "fill:#fff;stroke:none") || strings.Contains(raw, "fill:white;stroke:none") {
		t.Error("dark theme should not use white background")
	}
}

func TestRenderWithConfigTheme(t *testing.T) {
	input := `graph LR
    A --> B`
	out, err := Render(strings.NewReader(input), &Options{
		Theme: "dark",
	})
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	if strings.Contains(raw, "fill:#fff;stroke:none") || strings.Contains(raw, "fill:white;stroke:none") {
		t.Error("dark theme should not use white background")
	}
}

func TestRenderDeterministic(t *testing.T) {
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
			got, err := Render(bytes.NewReader(input), nil)
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

// unmarshalSVG parses svgBytes as XML and returns just the SVG root's
// viewBox. Round-tripping through encoding/xml validates the document
// structure as a side effect.
func unmarshalSVG(t *testing.T, svgBytes []byte) struct {
	XMLName xml.Name `xml:"svg"`
	ViewBox string   `xml:"viewBox,attr"`
} {
	t.Helper()
	var doc struct {
		XMLName xml.Name `xml:"svg"`
		ViewBox string   `xml:"viewBox,attr"`
	}
	// Skip the <?xml ?> declaration when present.
	body := svgBytes
	if i := bytes.Index(body, []byte("<svg")); i >= 0 {
		body = body[i:]
	}
	if err := xml.Unmarshal(body, &doc); err != nil {
		t.Fatalf("invalid SVG XML: %v\n%s", err, svgBytes)
	}
	return doc
}

// viewBoxWH parses the four-field viewBox ("minX minY width height")
// and returns just the width and height. Tests only care about the
// orientation, not the corner coordinates.
func viewBoxWH(t *testing.T, svgBytes []byte) (w, h float64) {
	t.Helper()
	doc := unmarshalSVG(t, svgBytes)
	parts := strings.Fields(doc.ViewBox)
	if len(parts) != 4 {
		t.Fatalf("viewBox should have 4 fields, got %d: %q", len(parts), doc.ViewBox)
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
