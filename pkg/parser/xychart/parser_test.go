package xychart

import (
	"strings"
	"testing"

	"github.com/julianshen/mmgo/pkg/diagram"
)

func TestParseHeaderRequired(t *testing.T) {
	if _, err := Parse(strings.NewReader("bar [1,2,3]\n")); err == nil {
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
		t.Fatal("expected error for non-xychart header")
	}
}

func TestParseHeaderVariants(t *testing.T) {
	cases := []string{
		"xychart-beta\n",
		"xychart-beta:\n",
		"xychart-beta horizontal\n",
		"xychart-beta\tfoo\n",
	}
	for _, c := range cases {
		if _, err := Parse(strings.NewReader(c)); err != nil {
			t.Errorf("header %q: %v", c, err)
		}
	}
}

func TestParseHorizontalFlag(t *testing.T) {
	// All three forms must set Horizontal=true. The previous
	// TrimSuffix-only trim chain only accepted the first form.
	cases := []string{
		"xychart-beta horizontal\n",
		"xychart-beta: horizontal\n",
		"xychart-beta:\thorizontal\n",
	}
	for _, c := range cases {
		d, err := Parse(strings.NewReader(c))
		if err != nil {
			t.Fatalf("Parse(%q): %v", c, err)
		}
		if !d.Horizontal {
			t.Errorf("header %q: Horizontal should be true", c)
		}
	}
}

func TestParseTitle(t *testing.T) {
	d, err := Parse(strings.NewReader(`xychart-beta
title "Sales Revenue"
`))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if d.Title != "Sales Revenue" {
		t.Errorf("Title = %q, want %q", d.Title, "Sales Revenue")
	}
}

func TestParseTitleUnquoted(t *testing.T) {
	d, err := Parse(strings.NewReader("xychart-beta\ntitle My Chart\n"))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if d.Title != "My Chart" {
		t.Errorf("Title = %q", d.Title)
	}
}

// HasHeaderKeyword accepts `:` as a word boundary, so directives may
// appear in colon form (`title:X`, `y-axis:0 --> 100`, `bar:[1,2,3]`).
// The old private trimKeyword didn't strip the colon — the stored
// values silently carried a leading ":". The migration to
// parserutil.TrimKeyword fixes that; this test pins the correct
// behavior so a future regression would break the build.
func TestParseKeywordColonForms(t *testing.T) {
	input := `xychart-beta
title:My Chart
y-axis:0 --> 100
bar:[1,2,3]
`
	d, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if d.Title != "My Chart" {
		t.Errorf("Title = %q, want %q", d.Title, "My Chart")
	}
	if !d.YAxis.HasRange || d.YAxis.Min != 0 || d.YAxis.Max != 100 {
		t.Errorf("y-axis = %+v, want HasRange min=0 max=100", d.YAxis)
	}
	if len(d.Series) != 1 || d.Series[0].Type != diagram.XYSeriesBar {
		t.Errorf("series = %+v, want one bar", d.Series)
	}
	if len(d.Series[0].Data) != 3 || d.Series[0].Data[0] != 1 {
		t.Errorf("data = %v", d.Series[0].Data)
	}
}

func TestParseCategoricalXAxis(t *testing.T) {
	d, err := Parse(strings.NewReader(`xychart-beta
x-axis [jan, feb, mar]
`))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	want := []string{"jan", "feb", "mar"}
	if got := d.XAxis.Categories; len(got) != 3 || got[0] != "jan" || got[2] != "mar" {
		t.Errorf("Categories = %v, want %v", got, want)
	}
}

func TestParseAxisWithTitleAndCategories(t *testing.T) {
	d, err := Parse(strings.NewReader(`xychart-beta
x-axis "Month" [Q1, Q2, Q3, Q4]
`))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if d.XAxis.Title != "Month" {
		t.Errorf("Title = %q, want Month", d.XAxis.Title)
	}
	if len(d.XAxis.Categories) != 4 {
		t.Errorf("Categories len = %d", len(d.XAxis.Categories))
	}
}

func TestParseAxisWithQuotedCategory(t *testing.T) {
	d, err := Parse(strings.NewReader(`xychart-beta
x-axis [one, "two, three", four]
`))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(d.XAxis.Categories) != 3 || d.XAxis.Categories[1] != "two, three" {
		t.Errorf("Categories = %v", d.XAxis.Categories)
	}
}

func TestParseNumericYAxis(t *testing.T) {
	d, err := Parse(strings.NewReader(`xychart-beta
y-axis "Revenue" 0 --> 1000
`))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if !d.YAxis.HasRange {
		t.Fatal("expected HasRange")
	}
	if d.YAxis.Min != 0 || d.YAxis.Max != 1000 {
		t.Errorf("range = [%g, %g]", d.YAxis.Min, d.YAxis.Max)
	}
	if d.YAxis.Title != "Revenue" {
		t.Errorf("Title = %q", d.YAxis.Title)
	}
}

func TestParseYAxisRangeOnly(t *testing.T) {
	d, err := Parse(strings.NewReader("xychart-beta\ny-axis 10 --> 100\n"))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if !d.YAxis.HasRange || d.YAxis.Min != 10 || d.YAxis.Max != 100 {
		t.Errorf("axis = %+v", d.YAxis)
	}
}

func TestParseYAxisBadRange(t *testing.T) {
	if _, err := Parse(strings.NewReader("xychart-beta\ny-axis 100 --> 10\n")); err == nil {
		t.Fatal("expected error when min >= max")
	}
	if _, err := Parse(strings.NewReader("xychart-beta\ny-axis abc --> 10\n")); err == nil {
		t.Fatal("expected error for non-numeric min")
	}
}

// NaN/Inf in axis bounds slip past strconv.ParseFloat and through the
// `minV >= maxV` check (NaN comparisons are false). The explicit
// finiteness guard must catch them.
func TestParseAxisNonFiniteRejected(t *testing.T) {
	cases := []string{
		"xychart-beta\ny-axis NaN --> 10\n",
		"xychart-beta\ny-axis 0 --> Inf\n",
		"xychart-beta\ny-axis -Inf --> 10\n",
	}
	for _, c := range cases {
		if _, err := Parse(strings.NewReader(c)); err == nil {
			t.Errorf("expected error for:\n%s", c)
		}
	}
}

// NaN/Inf in series data must be rejected — they would poison yRange
// and produce invalid SVG with zero-height bars.
func TestParseSeriesNonFiniteRejected(t *testing.T) {
	cases := []string{
		"xychart-beta\nbar [1, NaN, 3]\n",
		"xychart-beta\nline [Inf, 2]\n",
		"xychart-beta\nbar [-Inf]\n",
	}
	for _, c := range cases {
		if _, err := Parse(strings.NewReader(c)); err == nil {
			t.Errorf("expected error for:\n%s", c)
		}
	}
}

func TestParseAxisEmptyBracketsRejected(t *testing.T) {
	if _, err := Parse(strings.NewReader("xychart-beta\nx-axis []\n")); err == nil {
		t.Fatal("expected error for empty category list")
	}
}

func TestParseAxisUnterminatedQuote(t *testing.T) {
	if _, err := Parse(strings.NewReader("xychart-beta\nx-axis \"Month\n")); err == nil {
		t.Fatal("expected error for unterminated quoted title")
	}
}

func TestParseSeriesUnterminatedQuote(t *testing.T) {
	if _, err := Parse(strings.NewReader("xychart-beta\nbar \"Revenue [1,2,3]\n")); err == nil {
		t.Fatal("expected error for unterminated quoted series title")
	}
}

// title now routes through pullLeadingQuote, so an unterminated
// title quote errors symmetrically with the axis/series paths.
func TestParseTitleUnterminatedQuote(t *testing.T) {
	if _, err := Parse(strings.NewReader("xychart-beta\ntitle \"My Chart\n")); err == nil {
		t.Fatal("expected error for unterminated title quote")
	}
}

func TestParseBarSeries(t *testing.T) {
	d, err := Parse(strings.NewReader("xychart-beta\nbar [1, 2, 3, 4]\n"))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(d.Series) != 1 {
		t.Fatalf("series = %d", len(d.Series))
	}
	s := d.Series[0]
	if s.Type != diagram.XYSeriesBar {
		t.Errorf("Type = %v", s.Type)
	}
	if len(s.Data) != 4 || s.Data[0] != 1 || s.Data[3] != 4 {
		t.Errorf("Data = %v", s.Data)
	}
}

func TestParseLineSeries(t *testing.T) {
	d, err := Parse(strings.NewReader("xychart-beta\nline [0.5, 1e3, -2]\n"))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	s := d.Series[0]
	if s.Type != diagram.XYSeriesLine {
		t.Errorf("Type = %v", s.Type)
	}
	want := []float64{0.5, 1000, -2}
	for i, v := range want {
		if s.Data[i] != v {
			t.Errorf("Data[%d] = %g, want %g", i, s.Data[i], v)
		}
	}
}

func TestParseSeriesWithTitle(t *testing.T) {
	d, err := Parse(strings.NewReader(`xychart-beta
bar "Revenue" [10, 20, 30]
line "Target" [15, 25, 35]
`))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(d.Series) != 2 {
		t.Fatalf("series = %d", len(d.Series))
	}
	if d.Series[0].Title != "Revenue" || d.Series[1].Title != "Target" {
		t.Errorf("titles = %q, %q", d.Series[0].Title, d.Series[1].Title)
	}
}

func TestParseSeriesBadValue(t *testing.T) {
	if _, err := Parse(strings.NewReader("xychart-beta\nbar [1, abc, 3]\n")); err == nil {
		t.Fatal("expected error for non-numeric series value")
	}
}

func TestParseSeriesMissingBracket(t *testing.T) {
	if _, err := Parse(strings.NewReader("xychart-beta\nbar 1 2 3\n")); err == nil {
		t.Fatal("expected error for missing brackets")
	}
}

func TestParseFullExample(t *testing.T) {
	input := `xychart-beta
    title "Sales Revenue"
    x-axis [jan, feb, mar, apr, may, jun]
    y-axis "Revenue (in $)" 4000 --> 11000
    bar [5000, 6000, 7500, 8200, 9500, 10500]
    line [5000, 6000, 7500, 8200, 9500, 10500]
`
	d, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if d.Title != "Sales Revenue" {
		t.Error("title mismatch")
	}
	if len(d.XAxis.Categories) != 6 {
		t.Error("x categories mismatch")
	}
	if !d.YAxis.HasRange || d.YAxis.Min != 4000 || d.YAxis.Max != 11000 {
		t.Error("y range mismatch")
	}
	if len(d.Series) != 2 {
		t.Fatalf("series = %d, want 2", len(d.Series))
	}
	if d.Series[0].Type != diagram.XYSeriesBar || d.Series[1].Type != diagram.XYSeriesLine {
		t.Error("series types wrong")
	}
}

func TestParseCommentsIgnored(t *testing.T) {
	input := `xychart-beta
%% comment
title "T" %% trailing
bar [1,2,3]
`
	d, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if d.Title != "T" || len(d.Series) != 1 {
		t.Errorf("parsed = %+v", d)
	}
}

func TestParseUnknownKeywordIgnored(t *testing.T) {
	input := `xychart-beta
showLegend true
bar [1,2,3]
`
	d, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(d.Series) != 1 {
		t.Errorf("series = %d", len(d.Series))
	}
}

func TestXYSeriesTypeString(t *testing.T) {
	if diagram.XYSeriesBar.String() != "bar" {
		t.Error("bar.String()")
	}
	if diagram.XYSeriesLine.String() != "line" {
		t.Error("line.String()")
	}
}

func TestXYChartDiagramType(t *testing.T) {
	var d diagram.Diagram = &diagram.XYChartDiagram{}
	if d.Type() != diagram.XYChart {
		t.Errorf("Type() = %v", d.Type())
	}
}
