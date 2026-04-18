package quadrant

import (
	"bytes"
	"encoding/xml"
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
