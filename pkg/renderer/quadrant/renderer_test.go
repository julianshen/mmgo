package quadrant

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
	d := &diagram.QuadrantChartDiagram{}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	assertValidSVG(t, out)
}

func TestRenderFullChart(t *testing.T) {
	d := &diagram.QuadrantChartDiagram{
		Title:     "Campaigns",
		XAxisLow:  "Low Reach",
		XAxisHigh: "High Reach",
		YAxisLow:  "Low Engagement",
		YAxisHigh: "High Engagement",
		Quadrant1: "Expand",
		Quadrant2: "Promote",
		Quadrant3: "Re-evaluate",
		Quadrant4: "Improve",
		Points: []diagram.QuadrantPoint{
			{Label: "A", X: 0.3, Y: 0.6},
			{Label: "B", X: 0.7, Y: 0.8},
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	for _, s := range []string{">Campaigns<", ">Low Reach<", ">High Reach<",
		">Low Engagement<", ">Expand<", ">Re-evaluate<", ">A<", ">B<"} {
		if !strings.Contains(raw, s) {
			t.Errorf("expected %s in output", s)
		}
	}
	if n := strings.Count(raw, "<circle"); n != 2 {
		t.Errorf("circle count = %d, want 2", n)
	}
	assertValidSVG(t, out)
}

// Pin Mermaid's math-convention numbering: Q1 top-right, Q2 top-left,
// Q3 bottom-left, Q4 bottom-right. Each label's position is checked
// against the plot's own midlines so the test encodes the positioning
// contract directly, not a side effect.
func TestRenderQuadrantPositions(t *testing.T) {
	d := &diagram.QuadrantChartDiagram{
		Quadrant1: "Q1TR", Quadrant2: "Q2TL",
		Quadrant3: "Q3BL", Quadrant4: "Q4BR",
	}
	out, _ := Render(d, nil)
	raw := string(out)
	// Plot rect is the second <rect> (after the background).
	plotX0, plotY0, side := plotRect(t, raw)
	midX, midY := plotX0+side/2, plotY0+side/2

	want := []struct {
		name        string
		leftOfMidX  bool
		aboveMidY   bool
	}{
		{"Q1TR", false, true},
		{"Q2TL", true, true},
		{"Q3BL", true, false},
		{"Q4BR", false, false},
	}
	for _, w := range want {
		x, y := textCoords(t, raw, w.name)
		if (x < midX) != w.leftOfMidX {
			side := "right"
			if w.leftOfMidX {
				side = "left"
			}
			t.Errorf("%s X=%.2f should be on the %s of midX=%.2f", w.name, x, side, midX)
		}
		if (y < midY) != w.aboveMidY {
			pos := "below"
			if w.aboveMidY {
				pos = "above"
			}
			t.Errorf("%s Y=%.2f should be %s midY=%.2f", w.name, y, pos, midY)
		}
	}
}

// plotRect parses the second <rect> (first is the white background)
// and returns its origin plus side length.
func plotRect(t *testing.T, raw string) (x0, y0, side float64) {
	t.Helper()
	idx := strings.Index(raw, "<rect")
	if idx < 0 {
		t.Fatal("no <rect> found")
	}
	idx = strings.Index(raw[idx+1:], "<rect") + idx + 1
	end := strings.IndexByte(raw[idx:], '>')
	if end < 0 {
		t.Fatal("second <rect> has no closing >")
	}
	attrs := raw[idx : idx+end]
	x0 = parseAttrFloat(t, attrs, "x")
	y0 = parseAttrFloat(t, attrs, "y")
	side = parseAttrFloat(t, attrs, "width")
	return
}

func parseAttrFloat(t *testing.T, s, name string) float64 {
	t.Helper()
	needle := " " + name + `="`
	i := strings.Index(s, needle)
	if i < 0 {
		t.Fatalf("attr %s missing from %q", name, s)
	}
	i += len(needle)
	j := strings.Index(s[i:], `"`)
	if j < 0 {
		t.Fatalf("attr %s unterminated in %q", name, s)
	}
	v, err := strconv.ParseFloat(s[i:i+j], 64)
	if err != nil {
		t.Fatalf("attr %s parse: %v", name, err)
	}
	return v
}

// Y coordinate inversion: a point at (_, 0) must be lower on the SVG
// canvas (larger Y pixel) than a point at (_, 1), because our
// coordinate convention puts (0, 0) at bottom-left but SVG origin is
// top-left.
func TestRenderYCoordinateInversion(t *testing.T) {
	d := &diagram.QuadrantChartDiagram{
		Points: []diagram.QuadrantPoint{
			{Label: "BOT", X: 0.5, Y: 0},
			{Label: "TOP", X: 0.5, Y: 1},
		},
	}
	out, _ := Render(d, nil)
	raw := string(out)
	_, botY := textCoords(t, raw, "BOT")
	_, topY := textCoords(t, raw, "TOP")
	if !(topY < botY) {
		t.Errorf("TOP Y (%.2f) should be above BOT Y (%.2f) on SVG canvas", topY, botY)
	}
}

func TestRenderDeterministic(t *testing.T) {
	d := &diagram.QuadrantChartDiagram{
		Title:     "Det",
		XAxisLow:  "L", XAxisHigh: "H",
		YAxisLow:  "l", YAxisHigh: "h",
		Quadrant1: "1", Quadrant2: "2", Quadrant3: "3", Quadrant4: "4",
		Points: []diagram.QuadrantPoint{
			{Label: "A", X: 0.3, Y: 0.6},
			{Label: "B", X: 0.7, Y: 0.2},
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
			t.Fatalf("iter %d: diverges", i)
		}
	}
}

func TestRenderCustomFontSize(t *testing.T) {
	d := &diagram.QuadrantChartDiagram{
		Title:     "T",
		Quadrant1: "Q",
	}
	out, err := Render(d, &Options{FontSize: 20})
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if !strings.Contains(string(out), "font-size:20px") {
		t.Error("custom font size not applied")
	}
}

// A very small Options.FontSize must not drive derived label sizes
// (fontSize-1 for axis labels, fontSize-2 for point labels) to zero
// or below, which would serialize as invisible or invalid CSS.
func TestRenderSmallFontSizeClamped(t *testing.T) {
	d := &diagram.QuadrantChartDiagram{
		XAxisLow: "L",
		Points:   []diagram.QuadrantPoint{{Label: "A", X: 0.5, Y: 0.5}},
	}
	out, err := Render(d, &Options{FontSize: 2})
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	if strings.Contains(raw, "font-size:0px") || strings.Contains(raw, "font-size:-") {
		t.Errorf("derived font size degenerated to 0 or negative in:\n%s", raw)
	}
}

func TestRenderPointWithoutLabel(t *testing.T) {
	// A label-less point must emit exactly one <circle> and zero
	// <text> elements (no title, no quadrant names, no axis labels
	// means the entire SVG should contain no text).
	d := &diagram.QuadrantChartDiagram{
		Points: []diagram.QuadrantPoint{{X: 0.5, Y: 0.5}},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	if n := strings.Count(raw, "<circle"); n != 1 {
		t.Errorf("circle count = %d, want 1", n)
	}
	if n := strings.Count(raw, "<text"); n != 0 {
		t.Errorf("text count = %d, want 0 (no label means no text)", n)
	}
}

func TestRenderYAxisTitlesUseSVGTransform(t *testing.T) {
	// The renderer uses the SVG transform presentation attribute (not
	// CSS transform) so rotated labels render correctly in non-browser
	// consumers like tdewolff/canvas.
	d := &diagram.QuadrantChartDiagram{
		YAxisLow:  "Low",
		YAxisHigh: "High",
	}
	out, _ := Render(d, nil)
	raw := string(out)
	if strings.Contains(raw, "transform:rotate") {
		t.Error("CSS transform:rotate found; should use SVG transform attr")
	}
	if !strings.Contains(raw, `transform="rotate(`) {
		t.Error("SVG transform=rotate missing on Y-axis labels")
	}
}

func textCoords(t *testing.T, raw, content string) (x, y float64) {
	t.Helper()
	needle := ">" + content + "<"
	idx := strings.Index(raw, needle)
	if idx < 0 {
		t.Fatalf("text %q not found", content)
	}
	start := strings.LastIndex(raw[:idx], "<text")
	if start < 0 {
		t.Fatalf("<text opening for %q not found", content)
	}
	xIdx := strings.Index(raw[start:idx], ` x="`)
	yIdx := strings.Index(raw[start:idx], ` y="`)
	if xIdx < 0 || yIdx < 0 {
		t.Fatalf("x/y attrs missing for %q", content)
	}
	xIdx += start + len(` x="`)
	yIdx += start + len(` y="`)
	xEnd := strings.Index(raw[xIdx:], `"`)
	yEnd := strings.Index(raw[yIdx:], `"`)
	var err error
	x, err = strconv.ParseFloat(raw[xIdx:xIdx+xEnd], 64)
	if err != nil {
		t.Fatalf("parse x: %v", err)
	}
	y, err = strconv.ParseFloat(raw[yIdx:yIdx+yEnd], 64)
	if err != nil {
		t.Fatalf("parse y: %v", err)
	}
	return x, y
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

// Inline point styling reaches the rendered SVG.
func TestRenderPointInlineStyle(t *testing.T) {
	d := &diagram.QuadrantChartDiagram{
		Points: []diagram.QuadrantPoint{
			{Label: "A", X: 0.3, Y: 0.6, Style: diagram.QuadrantPointStyle{
				Color: "#abcdef", Radius: 11, StrokeWidth: 4, StrokeColor: "#000",
			}},
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	for _, want := range []string{"fill:#abcdef", "stroke:#000", "stroke-width:4", `r="11.00"`} {
		if !strings.Contains(raw, want) {
			t.Errorf("expected %q in output", want)
		}
	}
}

// classDef styling flows onto referenced points.
func TestRenderPointClassDef(t *testing.T) {
	d := &diagram.QuadrantChartDiagram{
		Classes: map[string]diagram.QuadrantPointStyle{
			"hot": {Color: "#f00", Radius: 14},
		},
		Points: []diagram.QuadrantPoint{
			{Label: "A", X: 0.5, Y: 0.5, Class: "hot"},
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	if !strings.Contains(raw, "fill:#f00") {
		t.Errorf("expected classDef fill in output")
	}
	if !strings.Contains(raw, `r="14.00"`) {
		t.Errorf("expected classDef radius in output")
	}
}

// AccTitle and AccDescr emit as <title>/<desc> SVG children.
func TestRenderAccessibility(t *testing.T) {
	d := &diagram.QuadrantChartDiagram{
		AccTitle: "Campaign matrix",
		AccDescr: "Reach vs Engagement",
		Points: []diagram.QuadrantPoint{
			{Label: "A", X: 0.5, Y: 0.5},
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	if !strings.Contains(raw, "<title>Campaign matrix</title>") {
		t.Errorf("expected <title> in output:\n%s", raw)
	}
	if !strings.Contains(raw, "<desc>Reach vs Engagement</desc>") {
		t.Errorf("expected <desc> in output:\n%s", raw)
	}
}

func TestRenderQuadrantPerQuadrantFills(t *testing.T) {
	d := &diagram.QuadrantChartDiagram{
		Quadrant1: "Q1", Quadrant2: "Q2",
		Quadrant3: "Q3", Quadrant4: "Q4",
	}
	out, err := Render(d, &Options{Theme: Theme{
		Quadrants: [4]QuadrantPalette{
			{Fill: "#aabbcc"}, // Q1 top-right
			{Fill: "#ccbbaa"}, // Q2 top-left
			{Fill: "#ddeeff"}, // Q3 bottom-left
			{Fill: "#ffeedd"}, // Q4 bottom-right
		},
	}})
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	for _, want := range []string{
		"fill:#aabbcc",
		"fill:#ccbbaa",
		"fill:#ddeeff",
		"fill:#ffeedd",
	} {
		if !strings.Contains(raw, want) {
			t.Errorf("expected per-quadrant fill %q in output", want)
		}
	}
}

func TestRenderQuadrantThemeVariables(t *testing.T) {
	d := &diagram.QuadrantChartDiagram{
		Title:    "T",
		XAxisLow: "Lo",
		Points:   []diagram.QuadrantPoint{{Label: "P", X: 0.5, Y: 0.5}},
	}
	out, err := Render(d, &Options{Theme: Theme{
		BackgroundColor: "#000",
		TitleColor:      "#fff",
		XAxisLabelColor: "#0f0",
		QuadrantPointFill: "#f0f",
	}})
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	for _, want := range []string{
		"fill:#000",
		"fill:#fff",
		"fill:#0f0",
		"fill:#f0f",
	} {
		if !strings.Contains(raw, want) {
			t.Errorf("expected themed color %q in output", want)
		}
	}
}

func TestRenderQuadrantConfigPointRadius(t *testing.T) {
	d := &diagram.QuadrantChartDiagram{
		Points: []diagram.QuadrantPoint{{Label: "P", X: 0.5, Y: 0.5}},
	}
	out, err := Render(d, &Options{Config: Config{PointRadius: 14}})
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if !strings.Contains(string(out), `r="14.00"`) {
		t.Errorf("expected r=\"14.00\" from custom PointRadius")
	}
}

// onlyBottomQuadrantsPopulated triggers the X-axis auto-flip so
// the axis labels move to the top of the plot.
func TestRenderQuadrantXAxisAutoFlip(t *testing.T) {
	// Only Q3/Q4 have labels; X-axis label should auto-flip to top.
	d := &diagram.QuadrantChartDiagram{
		Quadrant3: "BL", Quadrant4: "BR",
		XAxisLow:  "Lo",
		XAxisHigh: "Hi",
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	// `dominant-baseline="auto"` is the top-anchored variant the
	// renderer switches to when the axis sits above the plot.
	if !strings.Contains(raw, `dominant-baseline="auto"`) {
		t.Errorf("expected dominant-baseline=\"auto\" for top-anchored X-axis")
	}
	// Explicit Bottom override defeats auto-flip.
	out2, _ := Render(d, &Options{Config: Config{XAxisPosition: XAxisBottom}})
	if !strings.Contains(string(out2), `dominant-baseline="hanging"`) {
		t.Errorf("explicit XAxisBottom should anchor labels below the plot")
	}
}

// onlyRightQuadrantsPopulated triggers the Y-axis auto-flip; an
// explicit YAxisLeft override defeats it. The signal is the
// rotated label's X coordinate: when flipped right it sits past
// plotX1, when forced left it sits before plotX0.
func TestRenderQuadrantYAxisAutoFlip(t *testing.T) {
	d := &diagram.QuadrantChartDiagram{
		Quadrant1: "TR", Quadrant4: "BR", // only right half labelled
		YAxisLow:  "Lo",
		YAxisHigh: "Hi",
	}
	autoX := yAxisLabelRotationCX(t, d, nil)
	forcedX := yAxisLabelRotationCX(t, d, &Options{Config: Config{YAxisPosition: YAxisLeft}})
	if !(autoX > forcedX) {
		t.Errorf("auto-flip should place Y label to the right of the forced-left position; auto=%.2f forced=%.2f", autoX, forcedX)
	}
}

// yAxisLabelRotationCX renders the chart and extracts the cx of
// the first `transform="rotate(-90 cx cy)"` — the Y-axis label
// rotation pivot, equivalent to the label's X position.
func yAxisLabelRotationCX(t *testing.T, d *diagram.QuadrantChartDiagram, opts *Options) float64 {
	t.Helper()
	out, err := Render(d, opts)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	const marker = `transform="rotate(-90 `
	i := strings.Index(raw, marker)
	if i < 0 {
		t.Fatalf("no rotated Y-axis label in output:\n%s", raw)
	}
	rest := raw[i+len(marker):]
	end := strings.IndexByte(rest, ' ')
	if end < 0 {
		t.Fatal("rotate transform malformed")
	}
	v, err := strconv.ParseFloat(rest[:end], 64)
	if err != nil {
		t.Fatalf("rotate cx parse: %v", err)
	}
	return v
}

// When both opts.FontSize and opts.Config.TitleFontSize are set,
// the explicit Config value wins.
func TestRenderQuadrantConfigBeatsFontSize(t *testing.T) {
	d := &diagram.QuadrantChartDiagram{Title: "T"}
	out, err := Render(d, &Options{
		FontSize: 30,
		Config:   Config{TitleFontSize: 18},
	})
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	// Title renders at the explicit cfg.TitleFontSize=18; the
	// FontSize-derived `30+2=32` MUST NOT appear on the title.
	if !strings.Contains(raw, "font-size:18px;font-weight:bold") {
		t.Errorf("explicit Config.TitleFontSize should win over FontSize:\n%s", raw)
	}
}

// Independent X / Y axis padding: setting different paddings
// produces a non-square outer chrome (left vs bottom pads
// diverge).
func TestRenderQuadrantIndependentAxisGaps(t *testing.T) {
	d := &diagram.QuadrantChartDiagram{
		XAxisLow: "Lo", YAxisLow: "Lo",
	}
	out, err := Render(d, &Options{Config: Config{
		XAxisLabelPadding: 5,
		YAxisLabelPadding: 40,
	}})
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	// Pull the SVG's viewBox attribute and parse the four floats.
	const marker = `viewBox="`
	i := strings.Index(raw, marker)
	if i < 0 {
		t.Fatal("viewBox missing")
	}
	rest := raw[i+len(marker):]
	end := strings.Index(rest, `"`)
	parts := strings.Fields(rest[:end])
	if len(parts) != 4 {
		t.Fatalf("viewBox shape: %q", rest[:end])
	}
	w, _ := strconv.ParseFloat(parts[2], 64)
	h, _ := strconv.ParseFloat(parts[3], 64)
	// Y label padding (40 vs 5) inflates the *left* margin, so
	// the chart's overall width should clearly exceed the height
	// even though plotSide is the same.
	if w <= h {
		t.Errorf("yAxisGap=40 should widen the viewBox; got %vx%v", w, h)
	}
}

// resolveTheme and resolveConfig with nil opts return defaults
// unchanged; partial overrides leave un-set fields at default.
func TestResolveThemeAndConfigPartial(t *testing.T) {
	if got := resolveTheme(nil); got != DefaultTheme() {
		t.Errorf("resolveTheme(nil) drift: %+v", got)
	}
	if got := resolveConfig(nil); got != DefaultConfig() {
		t.Errorf("resolveConfig(nil) drift: %+v", got)
	}
	// Partial theme override: only TitleColor changes; the rest
	// keeps DefaultTheme values.
	got := resolveTheme(&Options{Theme: Theme{TitleColor: "#abc"}})
	if got.TitleColor != "#abc" {
		t.Errorf("TitleColor override missed: %q", got.TitleColor)
	}
	if got.BackgroundColor != DefaultTheme().BackgroundColor {
		t.Errorf("non-overridden BackgroundColor drifted: %q", got.BackgroundColor)
	}
	// Partial config override: only PointRadius changes.
	gotCfg := resolveConfig(&Options{Config: Config{PointRadius: 12}})
	if gotCfg.PointRadius != 12 {
		t.Errorf("PointRadius override missed: %v", gotCfg.PointRadius)
	}
	if gotCfg.ChartWidth != DefaultConfig().ChartWidth {
		t.Errorf("non-overridden ChartWidth drifted: %v", gotCfg.ChartWidth)
	}
}

// Per-quadrant TextFill flows into the rendered SVG so the four
// quadrant captions can each carry their own color.
func TestRenderQuadrantPerQuadrantTextFill(t *testing.T) {
	d := &diagram.QuadrantChartDiagram{
		Quadrant1: "Q1", Quadrant2: "Q2",
		Quadrant3: "Q3", Quadrant4: "Q4",
	}
	out, err := Render(d, &Options{Theme: Theme{
		Quadrants: [4]QuadrantPalette{
			{TextFill: "#111111"}, // QuadrantQ1
			{TextFill: "#222222"}, // QuadrantQ2
			{TextFill: "#333333"}, // QuadrantQ3
			{TextFill: "#444444"}, // QuadrantQ4
		},
	}})
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	for _, want := range []string{"fill:#111111", "fill:#222222", "fill:#333333", "fill:#444444"} {
		if !strings.Contains(raw, want) {
			t.Errorf("expected per-quadrant TextFill %q in output", want)
		}
	}
}

// Data points alone never trigger auto-flip — a chart with
// only points (no labels) keeps the default axis positions
// regardless of where the points land.
func TestQuadrantsPopulatedPointsDoNotTriggerFlip(t *testing.T) {
	d := &diagram.QuadrantChartDiagram{
		Points: []diagram.QuadrantPoint{
			{Label: "Lo", X: 0.2, Y: 0.2},
			{Label: "Hi", X: 0.8, Y: 0.8},
		},
	}
	top, bottom, left, right := quadrantsPopulated(d)
	if top || bottom || left || right {
		t.Errorf("points alone shouldn't mark any half populated; got top=%v bottom=%v left=%v right=%v",
			top, bottom, left, right)
	}
	if onlyBottomQuadrantsPopulated(d) || onlyRightQuadrantsPopulated(d) {
		t.Error("auto-flip should never trigger on point presence alone")
	}
}

// Setting only ChartWidth (or only ChartHeight) widens the plot
// to that single value — without this the default-height of 400
// would clamp the explicit width via the min() rule at render
// time.
func TestResolveConfigSingleAxisDimension(t *testing.T) {
	got := resolveConfig(&Options{Config: Config{ChartWidth: 700}})
	if got.ChartWidth != 700 || got.ChartHeight != 700 {
		t.Errorf("single ChartWidth override should mirror to ChartHeight; got W=%v H=%v", got.ChartWidth, got.ChartHeight)
	}
	got = resolveConfig(&Options{Config: Config{ChartHeight: 800}})
	if got.ChartWidth != 800 || got.ChartHeight != 800 {
		t.Errorf("single ChartHeight override should mirror to ChartWidth; got W=%v H=%v", got.ChartWidth, got.ChartHeight)
	}
	// Both explicit values keep their independent settings.
	got = resolveConfig(&Options{Config: Config{ChartWidth: 700, ChartHeight: 500}})
	if got.ChartWidth != 700 || got.ChartHeight != 500 {
		t.Errorf("both explicit values should be preserved; got W=%v H=%v", got.ChartWidth, got.ChartHeight)
	}
}

// Visible gap between adjacent rects must equal pad exactly
// (½ pad contributed by each neighbour). Asserts both axes plus
// horizontal centring of the quadrant title.
func TestRenderQuadrantPaddingWired(t *testing.T) {
	d := &diagram.QuadrantChartDiagram{
		Title:     "padding test",
		Quadrant1: "Q1", Quadrant2: "Q2", Quadrant3: "Q3", Quadrant4: "Q4",
		Points:    []diagram.QuadrantPoint{{Label: "p", X: 0.5, Y: 0.5}},
	}
	pal := [4]QuadrantPalette{
		{Fill: "#fee2e2"}, {Fill: "#dbeafe"}, {Fill: "#dcfce7"}, {Fill: "#fef3c7"},
	}
	render := func(pad float64) string {
		out, err := Render(d, &Options{
			Theme:  Theme{Quadrants: pal},
			Config: Config{QuadrantPadding: pad},
		})
		if err != nil {
			t.Fatalf("Render(pad=%v): %v", pad, err)
		}
		return string(out)
	}
	const small, big = 5.0, 25.0

	wSmall := attrFloat(t, render(small), "fill:#fee2e2", "width")
	wBig := attrFloat(t, render(big), "fill:#fee2e2", "width")
	if want := 1.5 * (big - small); math.Abs((wSmall-wBig)-want) > 0.5 {
		t.Errorf("expected Q1 width to shrink by ~%vpx (1.5*Δpad), got %v", want, wSmall-wBig)
	}

	svg := render(big)
	// Vertical inner edge: Q1 left − Q2 right == pad.
	q1Left := attrFloat(t, svg, "fill:#fee2e2", "x")
	q2Left := attrFloat(t, svg, "fill:#dbeafe", "x")
	q2Width := attrFloat(t, svg, "fill:#dbeafe", "width")
	if gap := q1Left - (q2Left + q2Width); math.Abs(gap-big) > 0.01 {
		t.Errorf("expected exact gap of %v between Q2 right and Q1 left edges, got %v", big, gap)
	}
	// Horizontal inner edge: Q4 top − Q1 bottom == pad. Catches a
	// transposed-axis regression that swaps the half/pad split on
	// x vs y dimensions.
	q1Top := attrFloat(t, svg, "fill:#fee2e2", "y")
	q1Height := attrFloat(t, svg, "fill:#fee2e2", "height")
	q4Top := attrFloat(t, svg, "fill:#fef3c7", "y")
	if gap := q4Top - (q1Top + q1Height); math.Abs(gap-big) > 0.01 {
		t.Errorf("expected exact gap of %v between Q1 bottom and Q4 top edges, got %v", big, gap)
	}
	// Title horizontal centring: Q1 label x must equal Q1 rect's
	// x-centre.
	q1Width := attrFloat(t, svg, "fill:#fee2e2", "width")
	wantCx := q1Left + q1Width/2
	gotCx := attrFloat(t, svg, ">Q1<", "x")
	if math.Abs(gotCx-wantCx) > 0.01 {
		t.Errorf("expected Q1 label centred at x=%v, got %v", wantCx, gotCx)
	}
}

// Single-rect path (no per-quadrant fills) must ignore
// QuadrantPadding — the underlying plot rect spans plotX0..plotX1
// regardless of the configured padding.
func TestRenderQuadrantPaddingNoFillsUnaffected(t *testing.T) {
	d := &diagram.QuadrantChartDiagram{
		Quadrant1: "Q1", Quadrant2: "Q2", Quadrant3: "Q3", Quadrant4: "Q4",
	}
	out0, err := Render(d, &Options{Config: Config{QuadrantPadding: 0}})
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	out25, err := Render(d, &Options{Config: Config{QuadrantPadding: 25}})
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	// The plot rect is the first style="fill:<PlotFill>". Strip the
	// background rect by skipping the first occurrence of fill:#fff
	// (default Background).
	plotRectRe := regexp.MustCompile(`<rect[^>]*style="fill:#f7f7fa[^"]*"[^>]*>`)
	r0 := plotRectRe.FindString(string(out0))
	r25 := plotRectRe.FindString(string(out25))
	if r0 == "" || r25 == "" {
		t.Fatalf("missing plot rect in output (PlotFill=#f7f7fa)\nout0=%s\nout25=%s", out0, out25)
	}
	if r0 != r25 {
		t.Errorf("single-rect path must ignore QuadrantPadding; got different rects:\n  pad=0:  %s\n  pad=25: %s", r0, r25)
	}
}

// Pad larger than what the plot can absorb must not produce
// inverted/zero-width rects. The renderer clamps cfg.QuadrantPadding
// internally; this test pins that contract by rendering with an
// extreme pad and checking every emitted rect has positive
// dimensions.
func TestRenderQuadrantPaddingClampedAtPlotSize(t *testing.T) {
	d := &diagram.QuadrantChartDiagram{
		Quadrant1: "Q1", Quadrant2: "Q2", Quadrant3: "Q3", Quadrant4: "Q4",
	}
	pal := [4]QuadrantPalette{
		{Fill: "#fee2e2"}, {Fill: "#dbeafe"}, {Fill: "#dcfce7"}, {Fill: "#fef3c7"},
	}
	out, err := Render(d, &Options{
		Theme:  Theme{Quadrants: pal},
		Config: Config{QuadrantPadding: 10000}, // clearly larger than the 400px default plot
	})
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	rectRe := regexp.MustCompile(`<rect[^>]*width="(-?[0-9.]+)"[^>]*height="(-?[0-9.]+)"`)
	for _, m := range rectRe.FindAllStringSubmatch(string(out), -1) {
		w, _ := strconv.ParseFloat(m[1], 64)
		h, _ := strconv.ParseFloat(m[2], 64)
		if w <= 0 || h <= 0 {
			t.Errorf("rect with non-positive dims under extreme pad: w=%v h=%v", w, h)
		}
	}
}

// Default Config (zero value) must render with the spec-default 5px
// gap. A regression that drops the default-resolution path would
// silently produce a gap-less plot here.
func TestRenderQuadrantPaddingDefault(t *testing.T) {
	d := &diagram.QuadrantChartDiagram{
		Quadrant1: "Q1", Quadrant2: "Q2", Quadrant3: "Q3", Quadrant4: "Q4",
	}
	pal := [4]QuadrantPalette{
		{Fill: "#fee2e2"}, {Fill: "#dbeafe"}, {Fill: "#dcfce7"}, {Fill: "#fef3c7"},
	}
	out, err := Render(d, &Options{Theme: Theme{Quadrants: pal}}) // no Config.
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	q1Left := attrFloat(t, string(out), "fill:#fee2e2", "x")
	q2Left := attrFloat(t, string(out), "fill:#dbeafe", "x")
	q2Width := attrFloat(t, string(out), "fill:#dbeafe", "width")
	const specDefault = 5.0
	if gap := q1Left - (q2Left + q2Width); math.Abs(gap-specDefault) > 0.01 {
		t.Errorf("expected spec-default gap of %v between adjacent quadrants, got %v", specDefault, gap)
	}
}

// A 20px padding bump must shift each label's y by 20px.
func TestRenderQuadrantTextTopPaddingWired(t *testing.T) {
	d := &diagram.QuadrantChartDiagram{
		Quadrant1: "Q1", Quadrant2: "Q2", Quadrant3: "Q3", Quadrant4: "Q4",
	}
	out5, err := Render(d, &Options{Config: Config{QuadrantTextTopPadding: 5}})
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	out25, err := Render(d, &Options{Config: Config{QuadrantTextTopPadding: 25}})
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	q1Y5 := attrFloat(t, string(out5), ">Q1<", "y")
	q1Y25 := attrFloat(t, string(out25), ">Q1<", "y")
	if delta := q1Y25 - q1Y5; math.Abs(delta-20) > 0.01 {
		t.Errorf("expected Q1 label y to shift by 20px when padding goes 5→25, got delta=%v", delta)
	}
}

// attrFloat scans `svg` forward to `marker`, walks back to the most
// recent ` <attr>="<num>"`, and returns the parsed float. Lets a
// caller pull any element's attribute without depending on element
// ordering or tag layout.
func attrFloat(t *testing.T, svg, marker, attr string) float64 {
	t.Helper()
	idx := strings.Index(svg, marker)
	if idx < 0 {
		t.Fatalf("attrFloat: marker %q not found", marker)
	}
	head := svg[:idx]
	needle := ` ` + attr + `="`
	at := strings.LastIndex(head, needle)
	if at < 0 {
		t.Fatalf("attrFloat: %s attribute not found before marker %q", attr, marker)
	}
	start := at + len(needle)
	end := strings.Index(head[start:], `"`)
	if end < 0 {
		t.Fatalf("attrFloat: unterminated %s value before marker %q", attr, marker)
	}
	v, err := strconv.ParseFloat(head[start:start+end], 64)
	if err != nil {
		t.Fatalf("attrFloat: malformed %s=%q: %v", attr, head[start:start+end], err)
	}
	return v
}
