package xychart

import (
	"bytes"
	"encoding/xml"
	"math"
	"regexp"
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

// Each show-flag must be independently honoured. Toggle exactly one
// flag at a time and verify the corresponding SVG element disappears
// while the unaffected ones stay.
func TestRenderShowFlagsIsolation(t *testing.T) {
	off := false
	base := func() *diagram.XYChartDiagram {
		return &diagram.XYChartDiagram{
			XAxis: diagram.XYAxis{Title: "X", Categories: []string{"a", "b"}},
			YAxis: diagram.XYAxis{Title: "Y", HasRange: true, Min: 0, Max: 10},
			Series: []diagram.XYSeries{
				{Type: diagram.XYSeriesBar, Data: []float64{1, 2}},
			},
		}
	}
	defaultOut, err := Render(base(), nil)
	if err != nil {
		t.Fatalf("default render: %v", err)
	}
	defaultRaw := string(defaultOut)
	// Sanity: defaults emit category labels, axis title, ticks, axis lines.
	for _, want := range []string{">a<", ">X<", ">Y<"} {
		if !strings.Contains(defaultRaw, want) {
			t.Fatalf("baseline missing %q", want)
		}
	}

	tests := []struct {
		name      string
		opts      *Options
		mustNot   []string
		mustStill []string
	}{
		{
			name: "XAxis.ShowLabel=off keeps Y labels",
			opts: &Options{Config: Config{XAxis: AxisConfig{ShowLabel: &off}}},
			mustNot:   []string{">a<", ">b<"},
			mustStill: []string{">X<", ">Y<", "<line"},
		},
		{
			name: "YAxis.ShowLabel=off keeps X labels",
			opts: &Options{Config: Config{YAxis: AxisConfig{ShowLabel: &off}}},
			mustNot:   []string{},
			mustStill: []string{">a<", ">b<", ">X<", ">Y<"},
		},
		{
			name: "XAxis.ShowTitle=off keeps Y title",
			opts: &Options{Config: Config{XAxis: AxisConfig{ShowTitle: &off}}},
			mustNot:   []string{">X<"},
			mustStill: []string{">Y<", ">a<"},
		},
		{
			name: "YAxis.ShowTitle=off keeps X title",
			opts: &Options{Config: Config{YAxis: AxisConfig{ShowTitle: &off}}},
			mustNot:   []string{">Y<"},
			mustStill: []string{">X<", ">a<"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			out, err := Render(base(), tc.opts)
			if err != nil {
				t.Fatalf("Render: %v", err)
			}
			raw := string(out)
			for _, s := range tc.mustNot {
				if strings.Contains(raw, s) {
					t.Errorf("expected %q absent, found in output", s)
				}
			}
			for _, s := range tc.mustStill {
				if !strings.Contains(raw, s) {
					t.Errorf("expected %q present, missing", s)
				}
			}
		})
	}
}

// Continuous X with HasRange must clamp out-of-range data points so
// they project onto the plot edges instead of escaping the viewBox or
// producing NaN coordinates.
func TestRenderContinuousXAxisClampsOutOfRange(t *testing.T) {
	d := &diagram.XYChartDiagram{
		XAxis: diagram.XYAxis{HasRange: true, Min: 0, Max: 10},
		YAxis: diagram.XYAxis{HasRange: true, Min: 0, Max: 100},
		Series: []diagram.XYSeries{
			{Type: diagram.XYSeriesLine, Data: []float64{-50, 50, 500}},
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	if strings.Contains(raw, "NaN") {
		t.Error("NaN coordinate leaked into SVG (continuous-X clamp failed)")
	}
	// Polyline must still render — clamped, not dropped.
	if !strings.Contains(raw, "<polyline") {
		t.Error("expected polyline even with out-of-range data")
	}
}

// In horizontal mode, cfg/theme keyed to the AST x-axis (which carries
// categories) must apply to the visible LEFT axis, not to the bottom.
// Regression for the axis-swap bug surfaced in PR review.
func TestRenderHorizontalAxisCfgSwap(t *testing.T) {
	d := &diagram.XYChartDiagram{
		Horizontal: true,
		XAxis:      diagram.XYAxis{Title: "CategoryAxis", Categories: []string{"a", "b"}},
		YAxis:      diagram.XYAxis{Title: "ValueAxis", HasRange: true, Min: 0, Max: 10},
		Series: []diagram.XYSeries{
			{Type: diagram.XYSeriesBar, Data: []float64{1, 2}},
		},
	}
	out, err := Render(d, &Options{Config: Config{
		// Make the X-axis label font distinctly larger than Y so we
		// can tell which one ended up where.
		XAxis: AxisConfig{LabelFontSize: 24, TitleFontSize: 28},
		YAxis: AxisConfig{LabelFontSize: 10, TitleFontSize: 12},
	}})
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	// Categories ("a", "b") should render at 24px (XAxis font),
	// because in horizontal mode the X-axis (categorical) is on the
	// left. Find an "a" label and verify its preceding font-size.
	aIdx := strings.Index(raw, ">a<")
	if aIdx < 0 {
		t.Fatalf("missing category label 'a'")
	}
	prelude := raw[:aIdx]
	fsIdx := strings.LastIndex(prelude, "font-size:")
	if fsIdx < 0 {
		t.Fatalf("could not locate font-size for category label")
	}
	fsEnd := strings.Index(prelude[fsIdx:], "px")
	if !strings.Contains(prelude[fsIdx:fsIdx+fsEnd], "24") {
		t.Errorf("expected category label at XAxis font (24px), got: %q", prelude[fsIdx:fsIdx+fsEnd+2])
	}
	// Bottom value-axis labels ("0", "2", ...) should be at YAxis 10px.
	if !strings.Contains(raw, "font-size:10px") {
		t.Errorf("expected value-axis labels at YAxis font (10px)")
	}
	// Title text colors shouldn't matter, just verify both titles render.
	for _, want := range []string{">CategoryAxis<", ">ValueAxis<"} {
		if !strings.Contains(raw, want) {
			t.Errorf("expected axis title %q", want)
		}
	}
}

// Bars whose values are negative or straddle zero must render with
// non-negative width/height. SVG with negative dimensions is
// undefined and silently dropped by some rasterizers, so any
// regression that recomputes a bar's size from a clamped value
// without an abs() would land here.
func TestRenderBarsCrossingZero(t *testing.T) {
	cases := []struct {
		name string
		d    *diagram.XYChartDiagram
	}{
		{
			name: "vertical, range crosses zero, mixed signs",
			d: &diagram.XYChartDiagram{
				XAxis: diagram.XYAxis{Categories: []string{"a", "b", "c", "d"}},
				YAxis: diagram.XYAxis{HasRange: true, Min: -10, Max: 10},
				Series: []diagram.XYSeries{
					{Type: diagram.XYSeriesBar, Data: []float64{-5, 3, -8, 6}},
				},
			},
		},
		{
			name: "horizontal, range crosses zero, mixed signs",
			d: &diagram.XYChartDiagram{
				Horizontal: true,
				XAxis:      diagram.XYAxis{Categories: []string{"a", "b", "c"}},
				YAxis:      diagram.XYAxis{HasRange: true, Min: -10, Max: 10},
				Series: []diagram.XYSeries{
					{Type: diagram.XYSeriesBar, Data: []float64{-5, 3, -8}},
				},
			},
		},
	}
	rectRe := regexp.MustCompile(`<rect[^>]*width="(-?[0-9.]+)"[^>]*height="(-?[0-9.]+)"`)
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out, err := Render(tc.d, nil)
			if err != nil {
				t.Fatalf("Render: %v", err)
			}
			matches := rectRe.FindAllStringSubmatch(string(out), -1)
			if len(matches) == 0 {
				t.Fatal("no rect elements found")
			}
			for _, m := range matches {
				w, _ := strconv.ParseFloat(m[1], 64)
				h, _ := strconv.ParseFloat(m[2], 64)
				if w < 0 || h < 0 {
					t.Errorf("rect with negative dimensions: width=%s height=%s", m[1], m[2])
				}
			}
		})
	}
}

// With a symmetric range, a positive-value bar and the same-magnitude
// negative-value bar must share an edge at the value-axis baseline
// pixel. Asserts the baseline math is correct, not just the abs()
// dimension guard.
func TestRenderBarsShareBaselineEdge(t *testing.T) {
	d := &diagram.XYChartDiagram{
		XAxis: diagram.XYAxis{Categories: []string{"pos", "neg"}},
		YAxis: diagram.XYAxis{HasRange: true, Min: -10, Max: 10},
		Series: []diagram.XYSeries{
			{Type: diagram.XYSeriesBar, Data: []float64{5, -5}},
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	rectRe := regexp.MustCompile(`<rect[^>]*y="([0-9.]+)"[^>]*height="([0-9.]+)"[^>]*style="fill:(#[0-9a-fA-F]{6});`)
	matches := rectRe.FindAllStringSubmatch(string(out), -1)
	var bars []struct{ y, h float64 }
	for _, m := range matches {
		if m[3] == "#fff" {
			continue
		}
		y, _ := strconv.ParseFloat(m[1], 64)
		h, _ := strconv.ParseFloat(m[2], 64)
		bars = append(bars, struct{ y, h float64 }{y, h})
	}
	if len(bars) != 2 {
		t.Fatalf("expected 2 data bars, got %d", len(bars))
	}
	// pos bar's bottom edge = neg bar's top edge = baseline pixel.
	posBottom := bars[0].y + bars[0].h
	negTop := bars[1].y
	if math.Abs(posBottom-negTop) > 0.01 {
		t.Errorf("expected positive-bar bottom (%.2f) to equal negative-bar top (%.2f) at baseline", posBottom, negTop)
	}
	// Same magnitude → same height.
	if math.Abs(bars[0].h-bars[1].h) > 0.01 {
		t.Errorf("expected equal heights for ±5 bars, got %.2f vs %.2f", bars[0].h, bars[1].h)
	}
}

// Range fully below zero — baseline clamps to plot top edge (yMax)
// and bars hang downward from it. Regression for the
// axisPos-clamps-to-edge case.
func TestRenderBarsAllNegativeRange(t *testing.T) {
	d := &diagram.XYChartDiagram{
		XAxis: diagram.XYAxis{Categories: []string{"a", "b", "c"}},
		YAxis: diagram.XYAxis{HasRange: true, Min: -10, Max: -1},
		Series: []diagram.XYSeries{
			{Type: diagram.XYSeriesBar, Data: []float64{-3, -7, -2}},
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	if strings.Contains(raw, "NaN") {
		t.Error("NaN coordinate leaked into SVG with all-negative range")
	}
	rectRe := regexp.MustCompile(`<rect[^>]*width="(-?[0-9.]+)"[^>]*height="(-?[0-9.]+)"`)
	for _, m := range rectRe.FindAllStringSubmatch(raw, -1) {
		w, _ := strconv.ParseFloat(m[1], 64)
		h, _ := strconv.ParseFloat(m[2], 64)
		if w < 0 || h < 0 {
			t.Errorf("rect with negative dimensions in all-negative range: w=%.2f h=%.2f", w, h)
		}
	}
}

// Outside-bar data labels for negative values must be placed below
// (vertical) or to the left (horizontal) of the bar — not above /
// right where they'd overlap empty space far from the bar.
func TestRenderDataLabelsFollowBarSign(t *testing.T) {
	on := true
	t.Run("vertical: negative bar's outside label sits below", func(t *testing.T) {
		d := &diagram.XYChartDiagram{
			XAxis: diagram.XYAxis{Categories: []string{"only"}},
			YAxis: diagram.XYAxis{HasRange: true, Min: -10, Max: 10},
			Series: []diagram.XYSeries{
				{Type: diagram.XYSeriesBar, Data: []float64{-5}},
			},
		}
		out, err := Render(d, &Options{Config: Config{
			ShowDataLabel: &on, ShowDataLabelOutsideBar: &on,
		}})
		if err != nil {
			t.Fatalf("Render: %v", err)
		}
		raw := string(out)
		// Find the bar's y attribute and the label "-5"'s y attribute;
		// label must sit below the bar's bottom edge.
		rectRe := regexp.MustCompile(`<rect[^>]*y="([0-9.]+)"[^>]*height="([0-9.]+)"[^>]*style="fill:#[0-9a-fA-F]{6}[^"]*"[^>]*>`)
		// Skip the background rect (fill:#fff in default theme).
		var barY, barH float64
		for _, m := range rectRe.FindAllStringSubmatch(raw, -1) {
			y, _ := strconv.ParseFloat(m[1], 64)
			h, _ := strconv.ParseFloat(m[2], 64)
			if h > 0 && h < 500 { // skip background
				barY, barH = y, h
				break
			}
		}
		// The y-axis also has a tick label "-5"; scope to the data
		// label by its smaller font (LabelFontSize - 2 = 12px default)
		// and middle anchor.
		labelRe := regexp.MustCompile(`<text[^>]*y="([0-9.]+)"[^>]*text-anchor="middle"[^>]*font-size:12px[^>]*>-5</text>`)
		m := labelRe.FindStringSubmatch(raw)
		if m == nil {
			t.Fatalf("missing data label '-5' (middle/12px) in:\n%s", raw)
		}
		labelY, _ := strconv.ParseFloat(m[1], 64)
		if labelY <= barY+barH {
			t.Errorf("expected negative-bar outside label below bar bottom %.2f, got y=%.2f", barY+barH, labelY)
		}
	})
	t.Run("horizontal: negative bar's outside label sits left", func(t *testing.T) {
		d := &diagram.XYChartDiagram{
			Horizontal: true,
			XAxis:      diagram.XYAxis{Categories: []string{"only"}},
			YAxis:      diagram.XYAxis{HasRange: true, Min: -10, Max: 10},
			Series: []diagram.XYSeries{
				{Type: diagram.XYSeriesBar, Data: []float64{-5}},
			},
		}
		out, err := Render(d, &Options{Config: Config{
			ShowDataLabel: &on, ShowDataLabelOutsideBar: &on,
		}})
		if err != nil {
			t.Fatalf("Render: %v", err)
		}
		// Scope to the data label (12px) — y-axis tick labels for the
		// continuous value axis at the bottom would also contain "-5"
		// at 14px.
		labelRe := regexp.MustCompile(`<text[^>]*text-anchor="(end|middle|start)"[^>]*font-size:12px[^>]*>-5</text>`)
		m := labelRe.FindStringSubmatch(string(out))
		if m == nil {
			t.Fatalf("missing 12px data label '-5':\n%s", string(out))
		}
		if m[1] != "end" {
			t.Errorf("expected text-anchor=\"end\" for negative-bar outside label in horizontal mode, got %q", m[1])
		}
	})
}

// In horizontal layout, two bar series in the same category must
// occupy non-overlapping y-slots.
func TestRenderHorizontalMultiBarSlotSplit(t *testing.T) {
	d := &diagram.XYChartDiagram{
		Horizontal: true,
		XAxis:      diagram.XYAxis{Categories: []string{"only"}},
		YAxis:      diagram.XYAxis{HasRange: true, Min: 0, Max: 100},
		Series: []diagram.XYSeries{
			{Type: diagram.XYSeriesBar, Data: []float64{50}},
			{Type: diagram.XYSeriesBar, Data: []float64{75}},
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	rectRe := regexp.MustCompile(`<rect[^>]*y="([0-9.]+)"[^>]*style="fill:(#[0-9a-fA-F]{6});`)
	matches := rectRe.FindAllStringSubmatch(string(out), -1)
	ys := map[string]string{}
	for _, m := range matches {
		// Skip the background rect (always #fff or theme background).
		if m[2] == "#fff" || m[2] == "#FFF" {
			continue
		}
		ys[m[1]] = m[2]
	}
	if len(ys) != 2 {
		t.Errorf("expected 2 distinct bar y-coordinates (split slot), got %d (%v)", len(ys), ys)
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
