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
// Q3 bottom-left, Q4 bottom-right. The test inspects the X of each
// quadrant label text element and compares pairs — top-right's Q1 must
// share X with bottom-right's Q4, etc.
func TestRenderQuadrantPositions(t *testing.T) {
	d := &diagram.QuadrantChartDiagram{
		Quadrant1: "Q1TR", Quadrant2: "Q2TL",
		Quadrant3: "Q3BL", Quadrant4: "Q4BR",
	}
	out, _ := Render(d, nil)
	raw := string(out)
	q1X, q1Y := textCoords(t, raw, "Q1TR")
	q2X, q2Y := textCoords(t, raw, "Q2TL")
	q3X, q3Y := textCoords(t, raw, "Q3BL")
	q4X, q4Y := textCoords(t, raw, "Q4BR")
	// X pairs: Q2/Q3 share left half; Q1/Q4 share right half.
	if !(q2X < q1X) {
		t.Errorf("Q2 (top-left) X %.2f should be less than Q1 (top-right) X %.2f", q2X, q1X)
	}
	if !(q3X < q4X) {
		t.Errorf("Q3 (bottom-left) X %.2f should be less than Q4 (bottom-right) X %.2f", q3X, q4X)
	}
	// Y pairs: Q1/Q2 share top half; Q3/Q4 share bottom half.
	if !(q1Y < q3Y) {
		t.Errorf("Q1 (top) Y %.2f should be less than Q3 (bottom) Y %.2f", q1Y, q3Y)
	}
	if !(q2Y < q4Y) {
		t.Errorf("Q2 (top) Y %.2f should be less than Q4 (bottom) Y %.2f", q2Y, q4Y)
	}
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
	d := &diagram.QuadrantChartDiagram{
		Points: []diagram.QuadrantPoint{{X: 0.5, Y: 0.5}},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	if !strings.Contains(raw, "<circle") {
		t.Error("circle missing")
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
