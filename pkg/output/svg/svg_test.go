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
	sequencerenderer "github.com/julianshen/mmgo/pkg/renderer/sequence"
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

func TestRenderSequenceParseError(t *testing.T) {
	_, err := Render(strings.NewReader("sequenceDiagram\n    badline!!!"), nil)
	if err == nil {
		t.Fatal("expected parse error")
	}
	if !strings.Contains(err.Error(), "parse") {
		t.Errorf("error should mention parse: %v", err)
	}
}

func TestRenderPieParseError(t *testing.T) {
	_, err := Render(strings.NewReader("pie\n    unquoted : 10"), nil)
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
		Subgraphs: []*diagram.Subgraph{
			{
				ID:    "sg",
				Nodes: []diagram.Node{{ID: "Nested", Label: "Nested"}},
				Edges: []diagram.Edge{{From: "Nested", To: "Top"}},
				Children: []*diagram.Subgraph{
					{ID: "inner", Nodes: []diagram.Node{{ID: "Deep", Label: "Deep"}}},
				},
			},
		},
	}
	ruler := mustRuler(t)
	defer func() { _ = ruler.Close() }()
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

// Diamond nodes must enter the layout graph as squares (mmdc parity).
// A wide label whose text-derived width exceeds its height would
// otherwise produce a flat rhombus instead of the proper square-on-its-
// corner diamond mermaid renders.
func TestBuildFlowchartGraphSquaresDiamond(t *testing.T) {
	d := &diagram.FlowchartDiagram{
		Nodes: []diagram.Node{
			{ID: "D", Label: "Decide with a long label", Shape: diagram.NodeShapeDiamond},
		},
	}
	ruler := mustRuler(t)
	defer func() { _ = ruler.Close() }()
	g := buildFlowchartGraph(d, ruler, flowchartrenderer.DefaultFontSize)
	attrs, ok := g.NodeAttrs("D")
	if !ok {
		t.Fatal("diamond node missing from graph")
	}
	if attrs.Width != attrs.Height {
		t.Errorf("diamond not squared: w=%v h=%v", attrs.Width, attrs.Height)
	}
	wText, hText := nodeSize(d.Nodes[0].Label, ruler, flowchartrenderer.DefaultFontSize)
	want := wText
	if hText > want {
		want = hText
	}
	if attrs.Width < want {
		t.Errorf("squared side %v smaller than max(text dims)=%v", attrs.Width, want)
	}
}

// extendedShapeSize is the single source of truth for the small-glyph
// shape dimensions. Locking the exact numbers in a test guards against
// silent regressions of the mermaid-parity radius (14→7) and the
// fork-bar geometry that the layout engine depends on.
func TestExtendedShapeSizes(t *testing.T) {
	cases := []struct {
		shape diagram.NodeShape
		w, h  float64
		ok    bool
	}{
		{diagram.NodeShapeSmallCircle, 14, 14, true},
		{diagram.NodeShapeFilledCircle, 14, 14, true},
		{diagram.NodeShapeFramedCircle, 14, 14, true},
		{diagram.NodeShapeForkJoin, 60, 8, true},
		{diagram.NodeShapeRectangle, 0, 0, false},
		{diagram.NodeShapeDiamond, 0, 0, false},
	}
	for _, tc := range cases {
		w, h, ok := extendedShapeSize(tc.shape)
		if ok != tc.ok || w != tc.w || h != tc.h {
			t.Errorf("%v: got (%v, %v, %v); want (%v, %v, %v)",
				tc.shape, w, h, ok, tc.w, tc.h, tc.ok)
		}
	}
}

func TestNodeSizeRespectsMinimum(t *testing.T) {
	ruler := mustRuler(t)
	defer func() { _ = ruler.Close() }()
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

func TestRenderClassDiagramEndToEnd(t *testing.T) {
	input := `classDiagram
    class Animal {
        +String name
        +eat()
    }
    Animal <|-- Dog`
	out, err := Render(strings.NewReader(input), nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	doc := unmarshalSVG(t, out)
	if doc.ViewBox == "" {
		t.Error("viewBox missing")
	}
	if !strings.Contains(raw, ">Animal<") || !strings.Contains(raw, ">Dog<") {
		t.Error("class labels missing")
	}
}

func TestRenderBlockEndToEnd(t *testing.T) {
	input := `block-beta
    a[Start] b(Middle) c{End}
    a --> b
    b --> c`
	out, err := Render(strings.NewReader(input), nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	doc := unmarshalSVG(t, out)
	if doc.ViewBox == "" {
		t.Error("viewBox missing")
	}
	if !strings.Contains(raw, ">Start<") || !strings.Contains(raw, ">End<") {
		t.Error("block labels missing")
	}
}

func TestRenderC4EndToEnd(t *testing.T) {
	input := `C4Context
    title System Context
    Person(customer, "Customer", "A user")
    System(app, "App", "Core system")
    Rel(customer, app, "Uses")`
	out, err := Render(strings.NewReader(input), nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	doc := unmarshalSVG(t, out)
	if doc.ViewBox == "" {
		t.Error("viewBox missing")
	}
	if !strings.Contains(raw, ">System Context<") || !strings.Contains(raw, ">Customer<") {
		t.Error("C4 labels missing")
	}
}

func TestRenderTimelineEndToEnd(t *testing.T) {
	input := `timeline
    title My Life
    section 2020s
        2020 : Graduated
        2021 : First Job`
	out, err := Render(strings.NewReader(input), nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	doc := unmarshalSVG(t, out)
	if doc.ViewBox == "" {
		t.Error("viewBox missing")
	}
	if !strings.Contains(raw, ">My Life<") || !strings.Contains(raw, ">2020s<") {
		t.Error("timeline labels missing")
	}
}

func TestRenderKanbanEndToEnd(t *testing.T) {
	input := `kanban
  Todo
    [Write docs]
    id2[Triage]@{ priority: 'High' }
  In progress
    [Implement renderer]
  Done
    [Write tests]@{ assigned: 'alice' }`
	out, err := Render(strings.NewReader(input), nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	doc := unmarshalSVG(t, out)
	if doc.ViewBox == "" {
		t.Error("viewBox missing")
	}
	for _, s := range []string{">Todo<", ">In progress<", ">Done<",
		">Write docs<", ">Triage<", ">priority: High<"} {
		if !strings.Contains(raw, s) {
			t.Errorf("missing %s", s)
		}
	}
}

func TestRenderQuadrantEndToEnd(t *testing.T) {
	input := `quadrantChart
    title Reach and engagement
    x-axis Low Reach --> High Reach
    y-axis Low Engagement --> High Engagement
    quadrant-1 We should expand
    quadrant-2 Need to promote
    quadrant-3 Re-evaluate
    quadrant-4 May be improved
    Campaign A: [0.3, 0.6]
    Campaign B: [0.7, 0.8]`
	out, err := Render(strings.NewReader(input), nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	doc := unmarshalSVG(t, out)
	if doc.ViewBox == "" {
		t.Error("viewBox missing")
	}
	for _, s := range []string{">Reach and engagement<", ">Low Reach<",
		">High Engagement<", ">We should expand<", ">Campaign A<"} {
		if !strings.Contains(raw, s) {
			t.Errorf("missing %s", s)
		}
	}
}

func TestRenderXYChartEndToEnd(t *testing.T) {
	input := `xychart-beta
    title "Sales Revenue"
    x-axis [jan, feb, mar, apr]
    y-axis "Revenue" 0 --> 10000
    bar [5000, 6000, 7500, 8200]
    line [5000, 6000, 7500, 8200]`
	out, err := Render(strings.NewReader(input), nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	doc := unmarshalSVG(t, out)
	if doc.ViewBox == "" {
		t.Error("viewBox missing")
	}
	if !strings.Contains(raw, ">Sales Revenue<") || !strings.Contains(raw, ">Revenue<") {
		t.Error("titles missing")
	}
	if !strings.Contains(raw, ">jan<") || !strings.Contains(raw, ">apr<") {
		t.Error("x-axis labels missing")
	}
	if !strings.Contains(raw, "<polyline") {
		t.Error("line series missing")
	}
}

func TestRenderSankeyEndToEnd(t *testing.T) {
	input := `sankey-beta
source,target,value
Coal,Power,50
Gas,Power,30
Power,Industry,40
Power,Homes,40`
	out, err := Render(strings.NewReader(input), nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	doc := unmarshalSVG(t, out)
	if doc.ViewBox == "" {
		t.Error("viewBox missing")
	}
	for _, n := range []string{"Coal", "Gas", "Power", "Industry", "Homes"} {
		if !strings.Contains(raw, ">"+n+" ") {
			t.Errorf("label %q missing", n)
		}
	}
}

func TestRenderGitGraphEndToEnd(t *testing.T) {
	input := `gitGraph
    commit id: "init"
    branch develop
    commit
    checkout main
    commit
    merge develop tag: "v1"`
	out, err := Render(strings.NewReader(input), nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	doc := unmarshalSVG(t, out)
	if doc.ViewBox == "" {
		t.Error("viewBox missing")
	}
	if !strings.Contains(raw, ">main<") || !strings.Contains(raw, ">develop<") {
		t.Error("branch lane labels missing")
	}
	if !strings.Contains(raw, ">v1<") {
		t.Error("merge tag missing")
	}
}

func TestRenderMindmapEndToEnd(t *testing.T) {
	input := `mindmap
    Root
        Branch 1
            Leaf
        Branch 2`
	out, err := Render(strings.NewReader(input), nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	doc := unmarshalSVG(t, out)
	if doc.ViewBox == "" {
		t.Error("viewBox missing")
	}
	if !strings.Contains(raw, ">Root<") || !strings.Contains(raw, ">Leaf<") {
		t.Error("mindmap node labels missing")
	}
}

func TestRenderGanttDiagramEndToEnd(t *testing.T) {
	input := `gantt
    title Project
    dateFormat YYYY-MM-DD
    section Phase 1
    Design :done, a1, 2024-01-01, 10d
    section Phase 2
    Build :active, a2, after a1, 20d`
	out, err := Render(strings.NewReader(input), nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	doc := unmarshalSVG(t, out)
	if doc.ViewBox == "" {
		t.Error("viewBox missing")
	}
	if !strings.Contains(raw, ">Project<") {
		t.Error("title missing")
	}
	if !strings.Contains(raw, ">Design<") {
		t.Error("task name missing")
	}
}

func TestRenderERDiagramEndToEnd(t *testing.T) {
	input := `erDiagram
    CUSTOMER ||--o{ ORDER : places
    CUSTOMER {
        string name
        int id PK
    }`
	out, err := Render(strings.NewReader(input), nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	doc := unmarshalSVG(t, out)
	if doc.ViewBox == "" {
		t.Error("viewBox missing")
	}
	if !strings.Contains(raw, ">CUSTOMER<") {
		t.Error("entity name missing")
	}
}

func TestRenderStateDiagramEndToEnd(t *testing.T) {
	input := `stateDiagram-v2
    [*] --> Idle
    Idle --> Active : start
    Active --> [*]`
	out, err := Render(strings.NewReader(input), nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	doc := unmarshalSVG(t, out)
	if doc.ViewBox == "" {
		t.Error("viewBox missing")
	}
	if !strings.Contains(raw, ">Idle<") || !strings.Contains(raw, ">Active<") {
		t.Error("state labels missing")
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
			src, cfg, err := extractInitDirective([]byte(tc.input))
			if err != nil {
				t.Fatal(err)
			}
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

func TestRenderWithInitDirectiveOnly(t *testing.T) {
	input := `%%{init: {"theme": "forest"}}%%
sequenceDiagram
    A->>B: hello`
	out, err := Render(strings.NewReader(input), nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	unmarshalSVG(t, out)
}

func TestRenderWithSequenceTheme(t *testing.T) {
	input := `sequenceDiagram
    A->>B: hi`
	out, err := Render(strings.NewReader(input), &Options{Theme: "dark"})
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	if !strings.Contains(raw, "#1f2020") {
		t.Error("dark theme should apply to sequence diagram background")
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
	if !strings.Contains(raw, "fill:#1f2020") {
		t.Error("dark theme should use #1f2020 background")
	}
}

func TestRenderWithExistingSequenceOpts(t *testing.T) {
	input := `sequenceDiagram
    A->>B: hi`
	seqOpts := &sequencerenderer.Options{FontSize: 20}
	out, err := Render(strings.NewReader(input), &Options{
		Theme:    "forest",
		Sequence: seqOpts,
	})
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	unmarshalSVG(t, out)
	// Verify caller's opts weren't mutated.
	if seqOpts.Theme.Background != "" {
		t.Error("caller's Sequence opts should not be mutated")
	}
}

func TestDiagramKindString(t *testing.T) {
	cases := []struct {
		k    diagramKind
		want string
	}{
		{kindFlowchart, "flowchart"},
		{kindSequence, "sequence"},
		{kindPie, "pie"},
		{kindUnknown, "unknown"},
	}
	for _, tc := range cases {
		if got := tc.k.String(); got != tc.want {
			t.Errorf("kind %d String() = %q, want %q", tc.k, got, tc.want)
		}
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
