package xychart

import (
	"bytes"
	"encoding/xml"
	"math"
	"strconv"
	"strings"
	"testing"

	"github.com/julianshen/mmgo/pkg/diagram"
)

func TestRenderNilDiagram(t *testing.T) {
	if _, err := Render(nil, nil); err == nil {
		t.Fatal("expected error")
	}
}

func TestRenderEmptyDiagram(t *testing.T) {
	d := &diagram.XYChartDiagram{}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	assertValidSVG(t, out)
}

func TestRenderSimpleBar(t *testing.T) {
	d := &diagram.XYChartDiagram{
		Title: "Sales",
		XAxis: diagram.XYAxis{Categories: []string{"Q1", "Q2", "Q3"}},
		Series: []diagram.XYSeries{
			{Type: diagram.XYSeriesBar, Data: []float64{10, 20, 30}},
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	if !strings.Contains(raw, ">Sales<") {
		t.Error("title missing")
	}
	if !strings.Contains(raw, ">Q1<") || !strings.Contains(raw, ">Q3<") {
		t.Error("category labels missing")
	}
	// Expect 3 bars (as rects in addition to the background rect).
	if n := strings.Count(raw, "<rect"); n < 4 {
		t.Errorf("rect count = %d, want at least 4 (bg + 3 bars)", n)
	}
	assertValidSVG(t, out)
}

func TestRenderLine(t *testing.T) {
	d := &diagram.XYChartDiagram{
		XAxis: diagram.XYAxis{Categories: []string{"a", "b", "c"}},
		Series: []diagram.XYSeries{
			{Type: diagram.XYSeriesLine, Data: []float64{1, 4, 2}},
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	if !strings.Contains(raw, "<polyline") {
		t.Error("polyline missing")
	}
	// Three data-point dots (circles).
	if n := strings.Count(raw, "<circle"); n != 3 {
		t.Errorf("circle count = %d, want 3", n)
	}
	assertValidSVG(t, out)
}

func TestRenderBarAndLineMixed(t *testing.T) {
	d := &diagram.XYChartDiagram{
		XAxis: diagram.XYAxis{Categories: []string{"a", "b", "c"}},
		YAxis: diagram.XYAxis{HasRange: true, Min: 0, Max: 100},
		Series: []diagram.XYSeries{
			{Type: diagram.XYSeriesBar, Data: []float64{30, 60, 90}},
			{Type: diagram.XYSeriesLine, Data: []float64{20, 50, 80}},
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	if !strings.Contains(raw, "<polyline") {
		t.Error("line series missing")
	}
	if n := strings.Count(raw, "<rect"); n < 4 {
		t.Error("bar series missing")
	}
	assertValidSVG(t, out)
}

func TestRenderMultipleBarSeriesSplitSlot(t *testing.T) {
	single := &diagram.XYChartDiagram{
		XAxis: diagram.XYAxis{Categories: []string{"a", "b"}},
		Series: []diagram.XYSeries{
			{Type: diagram.XYSeriesBar, Data: []float64{10, 20}},
		},
	}
	two := &diagram.XYChartDiagram{
		XAxis: diagram.XYAxis{Categories: []string{"a", "b"}},
		Series: []diagram.XYSeries{
			{Type: diagram.XYSeriesBar, Data: []float64{10, 20}},
			{Type: diagram.XYSeriesBar, Data: []float64{15, 25}},
		},
	}
	sOut, _ := Render(single, nil)
	tOut, _ := Render(two, nil)
	// 2-bar layout must produce exactly two bars per category slot
	// (1 bg + 4 bars = 5 rects); each bar is half the 1-bar width.
	if n := strings.Count(string(tOut), "<rect"); n != 5 {
		t.Errorf("rect count (2 series) = %d, want 5", n)
	}
	sw := firstBarWidth(t, sOut)
	tw := firstBarWidth(t, tOut)
	if !(tw > 0 && sw > 0 && tw < sw*0.6) {
		t.Errorf("two-series bar width %.2f should be ~half of single-series %.2f", tw, sw)
	}
}

// Regression: mixed bar+line charts used to count line series toward
// the bar count, producing half-width bars. One bar series alongside
// one line series must render the bar at full band width — the same
// width as a single-bar chart.
func TestRenderBarWidthIgnoresLineSeries(t *testing.T) {
	soloBar := &diagram.XYChartDiagram{
		XAxis: diagram.XYAxis{Categories: []string{"a", "b"}},
		YAxis: diagram.XYAxis{HasRange: true, Min: 0, Max: 100},
		Series: []diagram.XYSeries{
			{Type: diagram.XYSeriesBar, Data: []float64{50, 60}},
		},
	}
	mixed := &diagram.XYChartDiagram{
		XAxis: diagram.XYAxis{Categories: []string{"a", "b"}},
		YAxis: diagram.XYAxis{HasRange: true, Min: 0, Max: 100},
		Series: []diagram.XYSeries{
			{Type: diagram.XYSeriesBar, Data: []float64{50, 60}},
			{Type: diagram.XYSeriesLine, Data: []float64{40, 55}},
		},
	}
	sOut, _ := Render(soloBar, nil)
	mOut, _ := Render(mixed, nil)
	sw := firstBarWidth(t, sOut)
	mw := firstBarWidth(t, mOut)
	if math.Abs(sw-mw) > 0.01 {
		t.Errorf("bar width differed (solo=%.2f mixed=%.2f) — line series must not shrink bars", sw, mw)
	}
}

// firstBarWidth parses the width attribute of the first <rect ...
// fill="#5470c6"> — the first bar in the default palette.
func firstBarWidth(t *testing.T, svgBytes []byte) float64 {
	t.Helper()
	raw := string(svgBytes)
	// Skip the background rect (fill:#fff).
	idx := strings.Index(raw, `fill:#5470c6`)
	if idx < 0 {
		return 0
	}
	start := strings.LastIndex(raw[:idx], "<rect")
	if start < 0 {
		return 0
	}
	wIdx := strings.Index(raw[start:idx], ` width="`)
	if wIdx < 0 {
		return 0
	}
	wIdx += start + len(` width="`)
	end := strings.Index(raw[wIdx:], `"`)
	if end < 0 {
		return 0
	}
	v, _ := strconv.ParseFloat(raw[wIdx:wIdx+end], 64)
	return v
}

func TestRenderAutoYRange(t *testing.T) {
	d := &diagram.XYChartDiagram{
		XAxis: diagram.XYAxis{Categories: []string{"a", "b", "c"}},
		Series: []diagram.XYSeries{
			{Type: diagram.XYSeriesBar, Data: []float64{5, 10, 15}},
		},
	}
	// No explicit YAxis.HasRange — renderer derives from data.
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	// Tick labels are formatted; at least one non-zero tick should appear.
	raw := string(out)
	if !strings.Contains(raw, ">0<") {
		t.Error("expected a zero tick label")
	}
}

func TestRenderFlatData(t *testing.T) {
	// All data equal: yRange should widen so the bar/line is visible.
	d := &diagram.XYChartDiagram{
		XAxis: diagram.XYAxis{Categories: []string{"a", "b"}},
		Series: []diagram.XYSeries{
			{Type: diagram.XYSeriesLine, Data: []float64{5, 5}},
		},
	}
	if _, err := Render(d, nil); err != nil {
		t.Fatalf("Render: %v", err)
	}
}

func TestRenderSyntheticCategories(t *testing.T) {
	// No categories but a series with N points: synthetic "1".."N".
	d := &diagram.XYChartDiagram{
		Series: []diagram.XYSeries{
			{Type: diagram.XYSeriesLine, Data: []float64{1, 2, 3, 4}},
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	for _, c := range []string{">1<", ">2<", ">3<", ">4<"} {
		if !strings.Contains(raw, c) {
			t.Errorf("synthetic category %s missing", c)
		}
	}
}

func TestRenderWithAxisTitles(t *testing.T) {
	d := &diagram.XYChartDiagram{
		Title: "T",
		XAxis: diagram.XYAxis{Title: "Month", Categories: []string{"a"}},
		YAxis: diagram.XYAxis{Title: "Revenue", HasRange: true, Min: 0, Max: 100},
		Series: []diagram.XYSeries{
			{Type: diagram.XYSeriesBar, Data: []float64{50}},
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	if !strings.Contains(raw, ">Month<") || !strings.Contains(raw, ">Revenue<") {
		t.Error("axis titles missing")
	}
}

// Horizontal is parsed and preserved on the AST but not yet honored by
// the renderer — output must match the vertical rendering byte-for-byte.
// This test pins that intentional no-op so a future `Horizontal` impl
// has to touch the test deliberately.
// Horizontal layout swaps which axis carries the categorical labels:
// vertical puts categories along the bottom (text-anchor=middle,
// dominant=hanging), horizontal puts them down the left side
// (text-anchor=end, dominant=central).
func TestRenderHorizontalLayoutSwapsAxes(t *testing.T) {
	d := &diagram.XYChartDiagram{
		Horizontal: true,
		XAxis:      diagram.XYAxis{Categories: []string{"alpha", "beta"}},
		Series: []diagram.XYSeries{
			{Type: diagram.XYSeriesBar, Data: []float64{1, 2}},
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	// In horizontal mode, the category labels render with text-anchor="end"
	// (right-aligned to the left of the y-axis). Vertical mode uses "middle".
	if !strings.Contains(raw, `text-anchor="end"`) {
		t.Error("horizontal layout should anchor category labels at end (right of y-axis)")
	}
	for _, want := range []string{">alpha<", ">beta<"} {
		if !strings.Contains(raw, want) {
			t.Errorf("expected category label %q in horizontal output", want)
		}
	}
	// Config-level orientation override should also flip the layout
	// even when the AST flag is false.
	d2 := &diagram.XYChartDiagram{
		XAxis: diagram.XYAxis{Categories: []string{"alpha", "beta"}},
		Series: []diagram.XYSeries{
			{Type: diagram.XYSeriesBar, Data: []float64{1, 2}},
		},
	}
	out2, err := Render(d2, &Options{Config: Config{ChartOrientation: OrientationHorizontal}})
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if !strings.Contains(string(out2), `text-anchor="end"`) {
		t.Error("Config.ChartOrientation=Horizontal should flip layout regardless of AST flag")
	}
}

// `xychart-beta` with a numeric x-axis range and no categories should
// render continuous-axis tick labels along the x-axis instead of
// per-category labels.
func TestRenderContinuousXAxis(t *testing.T) {
	d := &diagram.XYChartDiagram{
		XAxis: diagram.XYAxis{Title: "T", HasRange: true, Min: 0, Max: 10},
		YAxis: diagram.XYAxis{HasRange: true, Min: 0, Max: 100},
		Series: []diagram.XYSeries{
			{Type: diagram.XYSeriesLine, Data: []float64{10, 30, 60, 90}},
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	for _, want := range []string{">0<", ">2<", ">4<", ">6<", ">8<", ">10<"} {
		if !strings.Contains(raw, want) {
			t.Errorf("expected continuous x-axis tick %q", want)
		}
	}
}

// ShowDataLabel paints the value over each bar; ShowDataLabelOutsideBar
// flips the placement to the bar's outer edge.
func TestRenderDataLabels(t *testing.T) {
	d := &diagram.XYChartDiagram{
		XAxis: diagram.XYAxis{Categories: []string{"a", "b"}},
		Series: []diagram.XYSeries{
			{Type: diagram.XYSeriesBar, Data: []float64{42, 17}},
		},
	}
	t.Run("inside", func(t *testing.T) {
		on := true
		out, err := Render(d, &Options{Config: Config{ShowDataLabel: &on}})
		if err != nil {
			t.Fatalf("Render: %v", err)
		}
		raw := string(out)
		for _, want := range []string{">42<", ">17<"} {
			if !strings.Contains(raw, want) {
				t.Errorf("expected data label %q", want)
			}
		}
	})
	t.Run("off by default", func(t *testing.T) {
		out, err := Render(d, nil)
		if err != nil {
			t.Fatalf("Render: %v", err)
		}
		if strings.Contains(string(out), ">42<") {
			t.Error("data labels should be off by default")
		}
	})
}

// Per-axis theme overrides bypass the LabelFill aggregate.
func TestRenderPerAxisThemeOverride(t *testing.T) {
	d := &diagram.XYChartDiagram{
		XAxis: diagram.XYAxis{Title: "X", Categories: []string{"a"}},
		YAxis: diagram.XYAxis{Title: "Y"},
		Series: []diagram.XYSeries{
			{Type: diagram.XYSeriesBar, Data: []float64{1}},
		},
	}
	out, err := Render(d, &Options{Theme: Theme{
		XAxisTitleColor: "#ff0000",
		YAxisTitleColor: "#0000ff",
	}})
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	if !strings.Contains(raw, "fill:#ff0000") {
		t.Error("XAxisTitleColor override missing")
	}
	if !strings.Contains(raw, "fill:#0000ff") {
		t.Error("YAxisTitleColor override missing")
	}
}

// Disabling axis show-flags suppresses the corresponding SVG.
// Horizontal layout with line series + multi-bar series + data labels
// exercises the lower coverage paths in renderSeriesHorizontal.
func TestRenderHorizontalSeriesCoverage(t *testing.T) {
	on := true
	d := &diagram.XYChartDiagram{
		Horizontal: true,
		XAxis:      diagram.XYAxis{Categories: []string{"a", "b", "c"}},
		Series: []diagram.XYSeries{
			{Type: diagram.XYSeriesBar, Data: []float64{10, 20, 30}},
			{Type: diagram.XYSeriesBar, Data: []float64{15, 25, 5}},
			{Type: diagram.XYSeriesLine, Data: []float64{12, 18, 22}},
			{Type: diagram.XYSeriesLine, Data: nil}, // empty line: skip path
		},
	}
	out, err := Render(d, &Options{Config: Config{
		ShowDataLabel:           &on,
		ShowDataLabelOutsideBar: &on,
	}})
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if !strings.Contains(string(out), "<polyline") {
		t.Error("expected line polyline in horizontal output")
	}
}

// Aggregate Theme.LabelFill / Theme.AxisStroke overrides rebroadcast
// to every per-axis surface they covered pre-split.
func TestRenderThemeAggregateRebroadcast(t *testing.T) {
	d := &diagram.XYChartDiagram{
		Title: "T",
		XAxis: diagram.XYAxis{Title: "X", Categories: []string{"a"}},
		YAxis: diagram.XYAxis{Title: "Y"},
		Series: []diagram.XYSeries{
			{Type: diagram.XYSeriesBar, Data: []float64{1}},
		},
	}
	out, err := Render(d, &Options{Theme: Theme{
		LabelFill:  "#abc123",
		AxisStroke: "#def456",
	}})
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	if !strings.Contains(raw, "fill:#abc123") {
		t.Error("LabelFill aggregate should rebroadcast to title/label/axis-title surfaces")
	}
	if !strings.Contains(raw, "stroke:#def456") {
		t.Error("AxisStroke aggregate should rebroadcast to axis-line/tick surfaces")
	}
}

// Config.ChartOrientation=Vertical forces vertical layout even when
// the AST flag says horizontal.
func TestRenderConfigForcesVerticalOverridesAST(t *testing.T) {
	d := &diagram.XYChartDiagram{
		Horizontal: true,
		XAxis:      diagram.XYAxis{Categories: []string{"a"}},
		Series: []diagram.XYSeries{
			{Type: diagram.XYSeriesBar, Data: []float64{1}},
		},
	}
	out, err := Render(d, &Options{Config: Config{ChartOrientation: OrientationVertical}})
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	// Vertical category labels anchor "middle".
	if !strings.Contains(string(out), `text-anchor="middle"`) {
		t.Error("OrientationVertical should override AST Horizontal=true")
	}
}

func TestRenderShowFlags(t *testing.T) {
	d := &diagram.XYChartDiagram{
		XAxis: diagram.XYAxis{Categories: []string{"a", "b"}},
		Series: []diagram.XYSeries{
			{Type: diagram.XYSeriesBar, Data: []float64{1, 2}},
		},
	}
	off := false
	out, err := Render(d, &Options{Config: Config{
		XAxis: AxisConfig{ShowAxisLine: &off, ShowTick: &off, ShowLabel: &off},
		YAxis: AxisConfig{ShowAxisLine: &off, ShowTick: &off, ShowLabel: &off},
	}})
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	// With both axis lines + ticks off, the only stroke entries left
	// are gridlines (#e5e5e5 default).
	if strings.Contains(raw, ">a<") || strings.Contains(raw, ">b<") {
		t.Error("ShowLabel=false should suppress category labels")
	}
}

// Data points beyond len(categories) are silently truncated. Pin the
// behavior — the alternative would be synthesizing extra categories,
// which would mask the user's data-shape mistake.
func TestRenderTruncatesDataBeyondCategories(t *testing.T) {
	d := &diagram.XYChartDiagram{
		XAxis: diagram.XYAxis{Categories: []string{"a", "b"}},
		YAxis: diagram.XYAxis{HasRange: true, Min: 0, Max: 10},
		Series: []diagram.XYSeries{
			{Type: diagram.XYSeriesBar, Data: []float64{1, 2, 3, 4}},
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	// 1 background + 2 bars (truncated to nCols) = 3 rects.
	if n := strings.Count(string(out), "<rect"); n != 3 {
		t.Errorf("rect count = %d, want 3 (extra data points should be dropped)", n)
	}
}

func TestRenderClampsOutOfRangeValues(t *testing.T) {
	// Value 200 > yMax=100 should clamp (not overflow plot).
	d := &diagram.XYChartDiagram{
		XAxis: diagram.XYAxis{Categories: []string{"a"}},
		YAxis: diagram.XYAxis{HasRange: true, Min: 0, Max: 100},
		Series: []diagram.XYSeries{
			{Type: diagram.XYSeriesBar, Data: []float64{200}},
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	assertValidSVG(t, out)
}

func TestRenderDeterministic(t *testing.T) {
	d := &diagram.XYChartDiagram{
		Title: "Det",
		XAxis: diagram.XYAxis{Categories: []string{"a", "b", "c"}},
		Series: []diagram.XYSeries{
			{Type: diagram.XYSeriesBar, Data: []float64{1, 2, 3}},
			{Type: diagram.XYSeriesLine, Data: []float64{3, 2, 1}},
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

func TestRenderCustomFontSize(t *testing.T) {
	d := &diagram.XYChartDiagram{
		XAxis: diagram.XYAxis{Title: "Month", Categories: []string{"a"}},
		Series: []diagram.XYSeries{
			{Type: diagram.XYSeriesBar, Data: []float64{1}},
		},
	}
	out, err := Render(d, &Options{FontSize: 20})
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	// Axis title uses fontSize directly; tick labels use fontSize-2.
	if !strings.Contains(string(out), "font-size:20px") {
		t.Error("custom font size not applied to axis title")
	}
	if !strings.Contains(string(out), "font-size:18px") {
		t.Error("custom font size not applied to tick labels")
	}
}

func TestFormatTick(t *testing.T) {
	cases := map[float64]string{
		0:     "0",
		10:    "10",
		1.5:   "1.5",
		1.25:  "1.25",
		1.999: "2",
		-3:    "-3",
	}
	for v, want := range cases {
		if got := formatTick(v); got != want {
			t.Errorf("formatTick(%g) = %q, want %q", v, got, want)
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

// AccTitle/AccDescr emit as <title>/<desc> SVG children.
func TestRenderXYChartAccessibility(t *testing.T) {
	d := &diagram.XYChartDiagram{
		AccTitle: "Quarterly revenue",
		AccDescr: "Sum across product lines",
		Series: []diagram.XYSeries{
			{Type: diagram.XYSeriesBar, Data: []float64{1, 2, 3}},
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	if !strings.Contains(raw, "<title>Quarterly revenue</title>") {
		t.Errorf("expected <title> in output:\n%s", raw)
	}
	if !strings.Contains(raw, "<desc>Sum across product lines</desc>") {
		t.Errorf("expected <desc> in output:\n%s", raw)
	}
}
