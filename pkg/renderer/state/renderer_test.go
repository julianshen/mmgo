package state

import (
	"bytes"
	"encoding/xml"
	"math"
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
