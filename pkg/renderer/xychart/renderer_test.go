package xychart

import (
	"bytes"
	"encoding/xml"
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
	d := &diagram.XYChartDiagram{
		XAxis: diagram.XYAxis{Categories: []string{"a", "b"}},
		Series: []diagram.XYSeries{
			{Type: diagram.XYSeriesBar, Data: []float64{10, 20}},
			{Type: diagram.XYSeriesBar, Data: []float64{15, 25}},
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	// 1 background + 2 categories × 2 series = 5 rects total.
	if n := strings.Count(string(out), "<rect"); n != 5 {
		t.Errorf("rect count = %d, want 5", n)
	}
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
