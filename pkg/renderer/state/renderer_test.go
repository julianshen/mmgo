package state

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"math"
	"regexp"
	"strings"
	"testing"

	"github.com/julianshen/mmgo/pkg/diagram"
	"github.com/julianshen/mmgo/pkg/layout"
)

func TestRenderNilDiagram(t *testing.T) {
	_, err := Render(nil, nil)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestRenderEmptyDiagram(t *testing.T) {
	d := &diagram.StateDiagram{}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	assertValidSVG(t, out)
}

func TestRenderSimpleStates(t *testing.T) {
	d := &diagram.StateDiagram{
		States: []diagram.StateDef{
			{ID: "Idle", Label: "Idle"},
			{ID: "Active", Label: "Active"},
		},
		Transitions: []diagram.StateTransition{
			{From: "Idle", To: "Active", Label: "start"},
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	if !strings.Contains(raw, ">Idle<") || !strings.Contains(raw, ">Active<") {
		t.Error("state labels missing")
	}
	if !strings.Contains(raw, ">start<") {
		t.Error("transition label missing")
	}
	assertValidSVG(t, out)
}

func TestRenderStartEndStates(t *testing.T) {
	d := &diagram.StateDiagram{
		States: []diagram.StateDef{
			{ID: "Running", Label: "Running"},
		},
		Transitions: []diagram.StateTransition{
			{From: "[*]", To: "Running"},
			{From: "Running", To: "[*]"},
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	if !strings.Contains(raw, "<circle") {
		t.Error("expected circle elements for [*] pseudo-states")
	}
	assertValidSVG(t, out)
}

func TestRenderSpecialStates(t *testing.T) {
	for _, tc := range []struct {
		kind diagram.StateKind
		name string
	}{
		{diagram.StateKindFork, "fork"},
		{diagram.StateKindJoin, "join"},
		{diagram.StateKindChoice, "choice"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d := &diagram.StateDiagram{
				States: []diagram.StateDef{
					{ID: "s1", Label: "s1", Kind: tc.kind},
					{ID: "A", Label: "A"},
				},
				Transitions: []diagram.StateTransition{
					{From: "s1", To: "A"},
				},
			}
			out, err := Render(d, nil)
			if err != nil {
				t.Fatalf("Render: %v", err)
			}
			assertValidSVG(t, out)
		})
	}
}

func TestRenderCompositeState(t *testing.T) {
	d := &diagram.StateDiagram{
		States: []diagram.StateDef{
			{
				ID: "Active", Label: "Active",
				Children: []diagram.StateDef{
					{ID: "Running", Label: "Running"},
					{ID: "Paused", Label: "Paused"},
				},
			},
		},
		Transitions: []diagram.StateTransition{
			{From: "Running", To: "Paused"},
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	if !strings.Contains(raw, ">Active<") {
		t.Error("composite label missing")
	}
	assertValidSVG(t, out)
}

func TestRenderDeterministic(t *testing.T) {
	d := &diagram.StateDiagram{
		States: []diagram.StateDef{
			{ID: "A", Label: "A"},
			{ID: "B", Label: "B"},
		},
		Transitions: []diagram.StateTransition{
			{From: "A", To: "B"},
		},
	}
	first, _ := Render(d, nil)
	for i := 0; i < 10; i++ {
		next, _ := Render(d, nil)
		if string(next) != string(first) {
			t.Fatalf("iter %d: output diverges", i)
		}
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

// A state with Description renders as a two-compartment box: a
// title row, a horizontal divider, and the description below.
func TestRenderStateDescription(t *testing.T) {
	d := &diagram.StateDiagram{
		States: []diagram.StateDef{
			{ID: "s1", Label: "s1", Description: "Idle phase"},
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	if !strings.Contains(raw, ">s1<") {
		t.Error("title missing")
	}
	if !strings.Contains(raw, ">Idle phase<") {
		t.Error("description missing")
	}
	// Divider line: a horizontal `<line>` with matching y1/y2 at
	// the bottom of the title band, stroked with the state stroke.
	// Geometric check is more discriminating than counting strokes
	// (rect borders use 1.5; an accidental width-1 line elsewhere
	// would otherwise pass the looser check).
	dividerStyle := fmt.Sprintf(`style="stroke:%s;stroke-width:1"`, DefaultTheme().StateStroke)
	idx := strings.Index(raw, dividerStyle)
	if idx < 0 {
		t.Fatalf("divider stroke style %q missing from output", dividerStyle)
	}
	// Walk back to find the enclosing <line ...> open tag and
	// verify y1==y2 (horizontal divider).
	lineOpen := strings.LastIndex(raw[:idx], "<line")
	if lineOpen < 0 {
		t.Fatal("no <line> element wraps the divider style")
	}
	lineTag := raw[lineOpen:idx]
	var x1, y1, x2, y2 float64
	if _, err := fmt.Sscanf(lineTag, `<line x1="%f" y1="%f" x2="%f" y2="%f"`, &x1, &y1, &x2, &y2); err != nil {
		t.Fatalf("divider geom parse %q: %v", lineTag, err)
	}
	if y1 != y2 {
		t.Errorf("divider not horizontal: y1=%f y2=%f", y1, y2)
	}
	if x1 >= x2 {
		t.Errorf("divider not left-to-right: x1=%f x2=%f", x1, x2)
	}
}

// Multi-line description lines (split on \n in the parser) emit
// one <text> element per line.
func TestRenderMultilineDescription(t *testing.T) {
	d := &diagram.StateDiagram{
		States: []diagram.StateDef{
			{ID: "s", Label: "s", Description: "alpha\nbeta\ngamma"},
		},
	}
	out, _ := Render(d, nil)
	raw := string(out)
	for _, line := range []string{">alpha<", ">beta<", ">gamma<"} {
		if !strings.Contains(raw, line) {
			t.Errorf("missing %q in output", line)
		}
	}
}

// Multi-line transition labels emit one <text> per line.
func TestRenderMultilineTransitionLabel(t *testing.T) {
	d := &diagram.StateDiagram{
		States: []diagram.StateDef{
			{ID: "A", Label: "A"},
			{ID: "B", Label: "B"},
		},
		Transitions: []diagram.StateTransition{
			{From: "A", To: "B", Label: "click\nrelease"},
		},
	}
	out, _ := Render(d, nil)
	raw := string(out)
	for _, want := range []string{">click<", ">release<"} {
		if !strings.Contains(raw, want) {
			t.Errorf("missing %q in transition label output", want)
		}
	}
}

// Direction selects layout RankDir. LR puts states side-by-side
// (wider than tall); TB stacks them (taller than wide).
func TestRenderDirection(t *testing.T) {
	cases := []struct {
		dir       diagram.Direction
		widerThan bool
	}{
		{diagram.DirectionTB, false},
		{diagram.DirectionLR, true},
	}
	for _, tc := range cases {
		t.Run(tc.dir.String(), func(t *testing.T) {
			d := &diagram.StateDiagram{
				Direction: tc.dir,
				States: []diagram.StateDef{
					{ID: "A", Label: "A"},
					{ID: "B", Label: "B"},
				},
				Transitions: []diagram.StateTransition{
					{From: "A", To: "B"},
				},
			}
			out, _ := Render(d, nil)
			body := out
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
			var minX, minY, w, h float64
			if _, err := fmt.Sscanf(doc.ViewBox, "%f %f %f %f", &minX, &minY, &w, &h); err != nil {
				t.Fatalf("viewBox parse: %v", err)
			}
			if tc.widerThan && !(w > h) {
				t.Errorf("%s viewBox should be wider than tall: %fx%f", tc.dir, w, h)
			}
			if !tc.widerThan && !(h > w) {
				t.Errorf("%s viewBox should be taller than wide: %fx%f", tc.dir, w, h)
			}
		})
	}
}

// `note left of S` produces a yellow rect to the left of S with a
// dashed connector.
func TestRenderStateNoteLeft(t *testing.T) {
	d := &diagram.StateDiagram{
		States: []diagram.StateDef{
			{ID: "S", Label: "S"},
		},
		Notes: []diagram.StateNote{
			{Target: "S", Side: diagram.NoteSideLeft, Text: "S is here"},
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	if !strings.Contains(raw, ">S is here<") {
		t.Error("note text missing")
	}
	if !strings.Contains(raw, DefaultTheme().NoteFill) {
		t.Error("note fill color missing")
	}
	if !strings.Contains(raw, "stroke-dasharray:4,3") {
		t.Error("dashed connector missing")
	}
}

// Multiline note text emits one <text> element per line.
func TestRenderStateNoteMultiline(t *testing.T) {
	d := &diagram.StateDiagram{
		States: []diagram.StateDef{{ID: "S", Label: "S"}},
		Notes: []diagram.StateNote{
			{Target: "S", Side: diagram.NoteSideRight, Text: "alpha\nbeta\ngamma"},
		},
	}
	out, _ := Render(d, nil)
	raw := string(out)
	for _, line := range []string{">alpha<", ">beta<", ">gamma<"} {
		if !strings.Contains(raw, line) {
			t.Errorf("missing %q in note output", line)
		}
	}
}

// A composite state renders as a labelled rounded rect that
// contains its child states' bbox.
func TestRenderCompositeStateBox(t *testing.T) {
	d := &diagram.StateDiagram{
		States: []diagram.StateDef{
			{ID: "Active", Label: "Active", Children: []diagram.StateDef{
				{ID: "Running", Label: "Running"},
				{ID: "Paused", Label: "Paused"},
			}},
		},
		Transitions: []diagram.StateTransition{
			{From: "Running", To: "Paused"},
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	// Composite label and fill present.
	if !strings.Contains(raw, ">Active<") {
		t.Error("composite label missing")
	}
	if !strings.Contains(raw, DefaultTheme().CompositeFill) {
		t.Error("composite fill colour missing")
	}
	// Child labels still rendered.
	for _, want := range []string{">Running<", ">Paused<"} {
		if !strings.Contains(raw, want) {
			t.Errorf("missing %q", want)
		}
	}
}

// A transition targeting a composite state must redirect to a
// representative leaf descendant in the layout graph. Otherwise
// dagre auto-creates a 0×0 phantom node and the transition's
// pseudo-state floats far from the composite.
//
// Geometric check: the start-state circle's cx should sit within
// reasonable horizontal range of the composite's center x. The
// previous (phantom-node) behavior placed the pseudo-state at
// dagre's first column while the composite landed elsewhere — a
// large delta. With the leaf-rep redirect they're aligned.
func TestRenderTransitionToCompositeUsesLeafRep(t *testing.T) {
	d := &diagram.StateDiagram{
		States: []diagram.StateDef{
			{ID: "Active", Label: "Active", Children: []diagram.StateDef{
				{ID: "Running", Label: "Running"},
				{ID: "Paused", Label: "Paused"},
			}},
		},
		Transitions: []diagram.StateTransition{
			{From: "[*]", To: "Active"},
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	if !strings.Contains(raw, ">Active<") {
		t.Fatal("composite label missing")
	}
	// Composite's <rect> is the first one with the composite stroke
	// styling. Background rect uses fill:#fff;stroke:none so won't
	// match the composite stroke.
	compStrokeMarker := `stroke:` + DefaultTheme().CompositeStroke
	idx := strings.Index(raw, compStrokeMarker)
	if idx < 0 {
		t.Fatal("composite frame not found")
	}
	rectOpen := strings.LastIndex(raw[:idx], "<rect")
	if rectOpen < 0 {
		t.Fatal("composite <rect> open not found")
	}
	var compX, compY, compW, compH float64
	if _, err := fmt.Sscanf(raw[rectOpen:],
		`<rect x="%f" y="%f" width="%f" height="%f"`,
		&compX, &compY, &compW, &compH); err != nil {
		t.Fatalf("composite rect parse: %v", err)
	}
	// Start-state filled circle: r matches startDotR (=5).
	circRe := regexp.MustCompile(`<circle cx="([\d.\-]+)" cy="([\d.\-]+)" r="7\.00"`)
	m := circRe.FindStringSubmatch(raw)
	if m == nil {
		t.Fatal("start-state circle not found")
	}
	var cx float64
	if _, err := fmt.Sscanf(m[1], "%f", &cx); err != nil {
		t.Fatalf("circle cx parse: %v", err)
	}
	compCenterX := compX + compW/2
	if delta := cx - compCenterX; delta < -100 || delta > 100 {
		t.Errorf("pseudo-state cx=%f far from composite centerX=%f; phantom-node?", cx, compCenterX)
	}
}

// Region divider is best-effort: drawn only when adjacent regions
// don't overlap vertically. Without cluster-aware layout, dagre
// often places regions side-by-side (same y range), so the divider
// is correctly suppressed in this case.
//
// This test verifies the suppression: parallel regions whose
// children dagre placed at the same y range get NO divider.
func TestRenderCompositeRegionDividerSkippedWhenInterleaved(t *testing.T) {
	d := &diagram.StateDiagram{
		States: []diagram.StateDef{
			{ID: "P", Label: "P",
				Children: []diagram.StateDef{
					{ID: "A1", Label: "A1"}, {ID: "A2", Label: "A2"},
					{ID: "B1", Label: "B1"}, {ID: "B2", Label: "B2"},
				},
				Regions: [][]diagram.StateDef{
					{{ID: "A1", Label: "A1"}, {ID: "A2", Label: "A2"}},
					{{ID: "B1", Label: "B1"}, {ID: "B2", Label: "B2"}},
				}},
		},
		Transitions: []diagram.StateTransition{
			{From: "A1", To: "A2"},
			{From: "B1", To: "B2"},
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if strings.Contains(string(out), "stroke-dasharray:5,4") {
		t.Errorf("interleaved regions should not draw a divider")
	}
}

func TestRenderAccessibilityMetadata(t *testing.T) {
	d := &diagram.StateDiagram{
		Title:    "general",
		AccTitle: "explicit acc",
		AccDescr: "long description",
		States:   []diagram.StateDef{{ID: "S", Label: "S"}},
	}
	out, _ := Render(d, nil)
	raw := string(out)
	if !strings.Contains(raw, "<title>explicit acc</title>") {
		t.Error("AccTitle should be the <title>")
	}
	if !strings.Contains(raw, "<desc>long description</desc>") {
		t.Error("AccDescr should be the <desc>")
	}
}

func TestRenderClassDefStyling(t *testing.T) {
	d := &diagram.StateDiagram{
		States: []diagram.StateDef{
			{ID: "Foo", Label: "Foo", CSSClasses: []string{"hot"}},
		},
		CSSClasses: map[string]string{
			"hot": "fill:#ff0000;stroke:#990000",
		},
	}
	out, _ := Render(d, nil)
	if !strings.Contains(string(out), "fill:#ff0000;stroke:#990000") {
		t.Errorf("classDef CSS missing from output:\n%s", out)
	}
}

func TestRenderStyleRule(t *testing.T) {
	d := &diagram.StateDiagram{
		States: []diagram.StateDef{{ID: "Foo", Label: "Foo"}},
		Styles: []diagram.StateStyleDef{{StateID: "Foo", CSS: "fill:#abcdef"}},
	}
	out, _ := Render(d, nil)
	if !strings.Contains(string(out), "fill:#abcdef") {
		t.Error("style override missing")
	}
}

func TestRenderClickHrefWrapsAnchor(t *testing.T) {
	d := &diagram.StateDiagram{
		States: []diagram.StateDef{{ID: "Foo", Label: "Foo"}},
		Clicks: []diagram.StateClickDef{{
			StateID: "Foo", URL: "https://example.com",
			Tooltip: "Open", Target: "_blank",
		}},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	if !strings.Contains(raw, `<a href="https://example.com" target="_blank">`) {
		t.Errorf("anchor element missing/malformed:\n%s", raw)
	}
	if !strings.Contains(raw, "<title>Open</title>") {
		t.Error("tooltip <title> missing inside anchor")
	}
}

// Callback-only clicks don't get wrapped — there's nothing for a
// static SVG to do with the JS reference.
func TestRenderCallbackOnlyDoesNotWrapAnchor(t *testing.T) {
	d := &diagram.StateDiagram{
		States: []diagram.StateDef{{ID: "Foo", Label: "Foo"}},
		Clicks: []diagram.StateClickDef{{StateID: "Foo", Callback: "openDetails"}},
	}
	out, _ := Render(d, nil)
	if strings.Contains(string(out), "<a ") {
		t.Error("callback-only click should not produce <a>")
	}
}

// Two notes on the same side of the same state stack vertically;
// their rects don't share a Y coordinate (the prior implementation
// stacked horizontally so connectors crossed each other's rects).
func TestRenderStateNotesStackVertically(t *testing.T) {
	d := &diagram.StateDiagram{
		States: []diagram.StateDef{{ID: "S", Label: "S"}},
		Notes: []diagram.StateNote{
			{Target: "S", Side: diagram.NoteSideLeft, Text: "first"},
			{Target: "S", Side: diagram.NoteSideLeft, Text: "second"},
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	idx1 := strings.Index(raw, ">first<")
	idx2 := strings.Index(raw, ">second<")
	if idx1 < 0 || idx2 < 0 {
		t.Fatalf("missing note text(s) in output")
	}
	// Pull the y attribute from each note's <text>.
	yOf := func(textIdx int) float64 {
		open := strings.LastIndex(raw[:textIdx], "<text")
		var x, y float64
		if _, err := fmt.Sscanf(raw[open:], `<text x="%f" y="%f"`, &x, &y); err != nil {
			t.Fatalf("text geom parse: %v", err)
		}
		return y
	}
	y1 := yOf(idx1)
	y2 := yOf(idx2)
	if y1 == y2 {
		t.Errorf("stacked same-side notes should have distinct y coords; both at y=%f", y1)
	}
	if y2 <= y1 {
		t.Errorf("second note should sit BELOW the first; y1=%f y2=%f", y1, y2)
	}
}

// A note for a state at the left edge of the diagram pushes the
// viewBox min-x negative so the note rect isn't clipped.
func TestRenderStateNoteExpandsViewBoxNegative(t *testing.T) {
	d := &diagram.StateDiagram{
		States: []diagram.StateDef{{ID: "S", Label: "S"}},
		Notes: []diagram.StateNote{
			{Target: "S", Side: diagram.NoteSideLeft, Text: "to the left"},
		},
	}
	out, _ := Render(d, nil)
	raw := string(out)
	body := raw
	if i := bytes.Index([]byte(raw), []byte("<svg")); i >= 0 {
		body = raw[i:]
	}
	var doc struct {
		XMLName xml.Name `xml:"svg"`
		ViewBox string   `xml:"viewBox,attr"`
	}
	if err := xml.Unmarshal([]byte(body), &doc); err != nil {
		t.Fatalf("svg parse: %v", err)
	}
	var minX, minY, w, h float64
	if _, err := fmt.Sscanf(doc.ViewBox, "%f %f %f %f", &minX, &minY, &w, &h); err != nil {
		t.Fatalf("viewBox parse: %v", err)
	}
	if minX >= 0 {
		t.Errorf("expected negative viewBox minX with a left-of note, got %f (full %q)", minX, doc.ViewBox)
	}
}

func TestLabelPosition(t *testing.T) {
	// A horizontal edge going +X produces perpendicular offset +Y;
	// vertical -Y edge offsets -X; and so on. Base point is the
	// midpoint supplied by the layout phase. The offset side is
	// fixed (CW in SVG Y-down), so anti-parallel edges land on
	// opposite sides — the property that separates cyclic labels.
	const off = 10.0
	cases := []struct {
		name   string
		pts    []layout.Point
		base   layout.Point
		wantX  float64
		wantY  float64
	}{
		{"too few points", []layout.Point{{X: 1, Y: 1}}, layout.Point{X: 5, Y: 5}, 5, 5},
		{"zero-length segment", []layout.Point{{X: 2, Y: 2}, {X: 2, Y: 2}}, layout.Point{X: 5, Y: 5}, 5, 5},
		{"east edge", []layout.Point{{X: 0, Y: 0}, {X: 10, Y: 0}}, layout.Point{X: 5, Y: 0}, 5, off},
		{"west edge", []layout.Point{{X: 10, Y: 0}, {X: 0, Y: 0}}, layout.Point{X: 5, Y: 0}, 5, -off},
		{"south edge", []layout.Point{{X: 0, Y: 0}, {X: 0, Y: 10}}, layout.Point{X: 0, Y: 5}, -off, 5},
		{"north edge", []layout.Point{{X: 0, Y: 10}, {X: 0, Y: 0}}, layout.Point{X: 0, Y: 5}, off, 5},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := labelPosition(tc.pts, tc.base)
			if math.Abs(got.X-tc.wantX) > 1e-9 || math.Abs(got.Y-tc.wantY) > 1e-9 {
				t.Errorf("labelPosition(%v, %v) = (%v, %v), want (%v, %v)",
					tc.pts, tc.base, got.X, got.Y, tc.wantX, tc.wantY)
			}
		})
	}
}

// Pins the cyclic-separation property: anti-parallel edges produce
// labels on opposite sides of their shared midpoint.
func TestLabelPositionAntiParallelEdgesSeparate(t *testing.T) {
	base := layout.Point{X: 50, Y: 50}
	fwd := labelPosition([]layout.Point{{X: 0, Y: 50}, {X: 100, Y: 50}}, base)
	rev := labelPosition([]layout.Point{{X: 100, Y: 50}, {X: 0, Y: 50}}, base)
	if fwd.Y == rev.Y || (fwd.Y > base.Y) == (rev.Y > base.Y) {
		t.Errorf("anti-parallel edges expected to offset to opposite sides, got fwd=%v rev=%v", fwd, rev)
	}
}

// Pins backdrop ordering: the white rect must precede the text so
// the text paints on top.
func TestRenderEdgeLabelBackdropPrecedesText(t *testing.T) {
	d := &diagram.StateDiagram{
		States: []diagram.StateDef{
			{ID: "A", Label: "A"},
			{ID: "B", Label: "B"},
		},
		Transitions: []diagram.StateTransition{
			{From: "A", To: "B", Label: "go"},
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	rectWhite := strings.Index(raw, `fill:#fff;stroke:none`)
	// Find the specific "go" label text. rectWhite must come before.
	textGo := strings.Index(raw, ">go<")
	if rectWhite < 0 || textGo < 0 {
		t.Fatalf("expected both white backdrop and label text, got rect=%d text=%d", rectWhite, textGo)
	}
	if rectWhite > textGo {
		t.Errorf("white rect at %d should precede label text at %d", rectWhite, textGo)
	}
}

func TestRenderAppliesCustomTheme(t *testing.T) {
	d := &diagram.StateDiagram{
		States: []diagram.StateDef{
			{ID: "A", Label: "A"},
			{ID: "B", Label: "B"},
		},
		Transitions: []diagram.StateTransition{
			{From: "A", To: "B", Label: "go"},
		},
	}
	out, err := Render(d, &Options{Theme: Theme{
		StateFill:     "#111111",
		StateStroke:   "#aabbcc",
		StateText:     "#ddeeff",
		EdgeStroke:    "#223344",
		EdgeText:      "#556677",
		LabelBackdrop: "#eeeeee",
		Background:    "#000000",
	}})
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	for _, want := range []string{
		`fill:#000000`,
		`fill:#111111;stroke:#aabbcc`,
		`fill:#ddeeff`,
		`stroke:#223344`,
		`fill:#556677`,
		`fill:#eeeeee;stroke:none`,
	} {
		if !strings.Contains(raw, want) {
			t.Errorf("themed output missing %q", want)
		}
	}
}

func TestDefaultThemeStable(t *testing.T) {
	got := DefaultTheme()
	want := Theme{
		StateFill:     "#ECECFF",
		StateStroke:   "#9370DB",
		StateText:     "#333",
		ChoiceFill:    "#333",
		PseudoMark:    "#333",
		EdgeStroke:    "#333",
		EdgeText:      "#333",
		LabelBackdrop: "#fff",
		Background:    "#fff",
		NoteFill:      "#fff5ad",
		NoteStroke:    "#aaaa33",
		NoteText:      "#333",
		CompositeFill:   "#f7f7ff",
		CompositeStroke: "#9370DB",
		CompositeText:   "#555",
	}
	if got != want {
		t.Errorf("DefaultTheme drifted:\n got  %+v\n want %+v", got, want)
	}
}

func TestResolveThemeNilOpts(t *testing.T) {
	if resolveTheme(nil) != DefaultTheme() {
		t.Error("resolveTheme(nil) should return DefaultTheme exactly")
	}
	if resolveTheme(&Options{}) != DefaultTheme() {
		t.Error("resolveTheme with zero Options should return DefaultTheme exactly")
	}
}
