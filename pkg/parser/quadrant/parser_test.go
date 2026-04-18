package quadrant

import (
	"strings"
	"testing"

	"github.com/julianshen/mmgo/pkg/diagram"
)

func TestParseHeaderRequired(t *testing.T) {
	if _, err := Parse(strings.NewReader("title x\n")); err == nil {
		t.Fatal("expected error for missing header")
	}
}

func TestParseEmpty(t *testing.T) {
	if _, err := Parse(strings.NewReader("")); err == nil {
		t.Fatal("expected error for empty input")
	}
}

func TestParseBadHeader(t *testing.T) {
	if _, err := Parse(strings.NewReader("pie title\n")); err == nil {
		t.Fatal("expected error for non-quadrantChart header")
	}
}

func TestParseHeaderVariants(t *testing.T) {
	cases := []string{
		"quadrantChart\n",
		"quadrantChart:\n",
	}
	for _, c := range cases {
		if _, err := Parse(strings.NewReader(c)); err != nil {
			t.Errorf("header %q: %v", c, err)
		}
	}
}

func TestParseTitle(t *testing.T) {
	d, err := Parse(strings.NewReader("quadrantChart\ntitle Campaign Reach\n"))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if d.Title != "Campaign Reach" {
		t.Errorf("Title = %q", d.Title)
	}
}

func TestParseAxisBothEnds(t *testing.T) {
	d, err := Parse(strings.NewReader("quadrantChart\nx-axis Low Reach --> High Reach\n"))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if d.XAxisLow != "Low Reach" || d.XAxisHigh != "High Reach" {
		t.Errorf("x-axis = (%q, %q)", d.XAxisLow, d.XAxisHigh)
	}
}

func TestParseAxisLowOnly(t *testing.T) {
	d, err := Parse(strings.NewReader("quadrantChart\ny-axis Engagement\n"))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if d.YAxisLow != "Engagement" || d.YAxisHigh != "" {
		t.Errorf("y-axis = (%q, %q)", d.YAxisLow, d.YAxisHigh)
	}
}

func TestParseAxisEmptyRejected(t *testing.T) {
	if _, err := Parse(strings.NewReader("quadrantChart\nx-axis\n")); err == nil {
		t.Fatal("expected error for bare x-axis")
	}
}

func TestParseAxisEmptyLowRejected(t *testing.T) {
	// `--> high` with an empty low side is almost certainly a typo.
	// Reject rather than silently accept an empty label.
	if _, err := Parse(strings.NewReader("quadrantChart\nx-axis --> High\n")); err == nil {
		t.Fatal("expected error for axis with empty low label")
	}
}

// HasHeaderKeyword accepts `:` as a boundary, so `title:X` matches.
// Without the colon strip, the prior code would leave a leading `:`
// in the stored value. Pin the correct behavior.
func TestParseKeywordColonForms(t *testing.T) {
	input := `quadrantChart
title: My Chart
x-axis:Low --> High
quadrant-1:Expand
`
	d, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if d.Title != "My Chart" {
		t.Errorf("Title = %q, want %q", d.Title, "My Chart")
	}
	if d.XAxisLow != "Low" || d.XAxisHigh != "High" {
		t.Errorf("x-axis = (%q, %q)", d.XAxisLow, d.XAxisHigh)
	}
	if d.Quadrant1 != "Expand" {
		t.Errorf("Quadrant1 = %q", d.Quadrant1)
	}
}

// A data point whose label happens to match a directive keyword must
// still be captured as a point — the bracket shape is the precedence
// signal. Regression guard: without the bracket-first check in
// parseLine, `title: [0.5, 0.5]` would be parsed as a title directive
// and the point silently dropped.
func TestParseKeywordCollidingPointLabels(t *testing.T) {
	input := `quadrantChart
title: [0.5, 0.5]
x-axis: [0.1, 0.2]
quadrant-1: [0.9, 0.9]
`
	d, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(d.Points) != 3 {
		t.Fatalf("points = %d, want 3 (keyword-like labels should still parse as points)", len(d.Points))
	}
	if d.Title != "" || d.XAxisLow != "" || d.Quadrant1 != "" {
		t.Errorf("directive slots should be empty when the line is a point: title=%q x-low=%q q1=%q",
			d.Title, d.XAxisLow, d.Quadrant1)
	}
}

// `theme: dark` (unknown directive with a colon but no bracket) should
// be silently ignored — not error as a bad point and not dropped into
// any directive slot.
func TestParseUnknownDirectiveWithColonIgnored(t *testing.T) {
	input := `quadrantChart
theme: dark
Campaign A: [0.5, 0.5]
`
	d, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(d.Points) != 1 {
		t.Errorf("points = %d, want 1", len(d.Points))
	}
}

func TestParsePointsAtCorners(t *testing.T) {
	input := `quadrantChart
BottomLeft: [0, 0]
TopRight: [1, 1]
Center: [0.5, 0.5]
`
	d, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(d.Points) != 3 {
		t.Fatalf("points = %d, want 3", len(d.Points))
	}
}

func TestParseAllQuadrants(t *testing.T) {
	input := `quadrantChart
quadrant-1 Expand
quadrant-2 Promote
quadrant-3 Re-evaluate
quadrant-4 Improve
`
	d, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if d.Quadrant1 != "Expand" || d.Quadrant2 != "Promote" ||
		d.Quadrant3 != "Re-evaluate" || d.Quadrant4 != "Improve" {
		t.Errorf("quadrants = (%q, %q, %q, %q)",
			d.Quadrant1, d.Quadrant2, d.Quadrant3, d.Quadrant4)
	}
}

func TestParsePoints(t *testing.T) {
	input := `quadrantChart
Campaign A: [0.3, 0.6]
Campaign B: [0.45, 0.23]
`
	d, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(d.Points) != 2 {
		t.Fatalf("points = %d", len(d.Points))
	}
	a := d.Points[0]
	if a.Label != "Campaign A" || a.X != 0.3 || a.Y != 0.6 {
		t.Errorf("point 0 = %+v", a)
	}
}

func TestParsePointOutOfRange(t *testing.T) {
	if _, err := Parse(strings.NewReader("quadrantChart\nA: [1.5, 0.5]\n")); err == nil {
		t.Fatal("expected error for x > 1")
	}
	if _, err := Parse(strings.NewReader("quadrantChart\nA: [-0.1, 0.5]\n")); err == nil {
		t.Fatal("expected error for x < 0")
	}
}

func TestParsePointNonFinite(t *testing.T) {
	if _, err := Parse(strings.NewReader("quadrantChart\nA: [NaN, 0.5]\n")); err == nil {
		t.Fatal("expected error for NaN")
	}
	if _, err := Parse(strings.NewReader("quadrantChart\nA: [0.5, Inf]\n")); err == nil {
		t.Fatal("expected error for Inf")
	}
}

// Malformed *point* lines (bracket-shaped but invalid contents) must
// error so users see their typo. A missing-bracket line like
// `A: 0.3, 0.6` is treated as an unknown directive (silently ignored
// for forward-compat) rather than a bad point.
func TestParsePointBadFormat(t *testing.T) {
	cases := []string{
		"quadrantChart\nA: [0.3\n",             // missing close
		"quadrantChart\nA: [0.3, 0.5] extra\n", // trailing garbage
		"quadrantChart\nA: [0.3]\n",            // one coord
		"quadrantChart\n: [0.3, 0.5]\n",        // empty label
		"quadrantChart\nA: [abc, 0.5]\n",       // non-numeric
	}
	for _, c := range cases {
		if _, err := Parse(strings.NewReader(c)); err == nil {
			t.Errorf("expected error for:\n%s", c)
		}
	}
}

// A colon line without brackets is NOT a point — it's treated as an
// unknown directive and silently ignored for forward-compat. This
// keeps `theme: dark` tolerable but shifts `A: 0.3, 0.6` (missing
// brackets) from "error" to "no point created."
func TestParseColonLineWithoutBracketIgnored(t *testing.T) {
	d, err := Parse(strings.NewReader("quadrantChart\nA: 0.3, 0.6\n"))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(d.Points) != 0 {
		t.Errorf("points = %d, want 0 (non-bracket colon line is a directive, not a point)", len(d.Points))
	}
}

func TestParseFullExample(t *testing.T) {
	input := `quadrantChart
    title Reach and engagement of campaigns
    x-axis Low Reach --> High Reach
    y-axis Low Engagement --> High Engagement
    quadrant-1 We should expand
    quadrant-2 Need to promote
    quadrant-3 Re-evaluate
    quadrant-4 May be improved
    Campaign A: [0.3, 0.6]
    Campaign B: [0.45, 0.23]
    Campaign C: [0.57, 0.69]
    Campaign D: [0.78, 0.34]
    Campaign E: [0.40, 0.34]
    Campaign F: [0.35, 0.78]
`
	d, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if d.Title != "Reach and engagement of campaigns" {
		t.Error("title mismatch")
	}
	if d.XAxisLow != "Low Reach" || d.XAxisHigh != "High Reach" {
		t.Error("x-axis mismatch")
	}
	if d.Quadrant1 != "We should expand" {
		t.Error("Q1 mismatch")
	}
	if len(d.Points) != 6 {
		t.Errorf("points = %d, want 6", len(d.Points))
	}
}

func TestParseCommentsIgnored(t *testing.T) {
	input := `quadrantChart
%% comment
title T %% trailing
A: [0.5, 0.5]
`
	d, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if d.Title != "T" || len(d.Points) != 1 {
		t.Errorf("parsed = %+v", d)
	}
}

func TestParseUnknownLineIgnored(t *testing.T) {
	// A line with no colon and no known keyword is silently ignored
	// (forward-compat tolerance for directives like `themeVariables`).
	input := `quadrantChart
themeVariables.primaryColor #fff
A: [0.5, 0.5]
`
	d, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(d.Points) != 1 {
		t.Errorf("points = %d", len(d.Points))
	}
}

func TestQuadrantChartDiagramType(t *testing.T) {
	var d diagram.Diagram = &diagram.QuadrantChartDiagram{}
	if d.Type() != diagram.Quadrant {
		t.Errorf("Type() = %v", d.Type())
	}
}
