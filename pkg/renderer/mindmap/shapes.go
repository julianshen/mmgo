package mindmap

import (
	"fmt"
	"strings"
)

func defaultNodePath(w, h float64) string {
	rd := 5.0
	return fmt.Sprintf(
		"M%.2f %.2f v%.2f q0,%.2f %.2f,%.2f h%.2f q%.2f,0 %.2f,%.2f v%.2f H%.2f Z",
		-w/2, h/2-rd,
		-(h - 2*rd),
		-rd, rd, -rd,
		w-2*rd,
		rd, rd, rd,
		h-rd,
		-w/2,
	)
}

func cloudPath(w, h float64) string {
	r1 := 0.15 * w
	r2 := 0.25 * w
	r3 := 0.35 * w
	r4 := 0.2 * w
	hw := -w / 2
	hh := -h / 2
	return fmt.Sprintf(
		"M%.2f %.2f "+
			"a%.2f,%.2f 0 0,1 %.2f,%.2f "+
			"a%.2f,%.2f 1 0,1 %.2f,%.2f "+
			"a%.2f,%.2f 1 0,1 %.2f,%.2f "+
			"a%.2f,%.2f 1 0,1 %.2f,%.2f "+
			"a%.2f,%.2f 1 0,1 %.2f,%.2f "+
			"a%.2f,%.2f 1 0,1 %.2f,%.2f "+
			"a%.2f,%.2f 1 0,1 %.2f,%.2f "+
			"a%.2f,%.2f 1 0,1 %.2f,%.2f "+
			"a%.2f,%.2f 1 0,1 %.2f,%.2f "+
			"a%.2f,%.2f 1 0,1 %.2f,%.2f "+
			"H%.2f V%.2f Z",
		hw, hh,
		r1, r1, w*0.25, -w*0.1,
		r3, r3, w*0.4, -w*0.1,
		r2, r2, w*0.35, w*0.2,
		r1, r1, w*0.15, h*0.35,
		r4, r4, -w*0.15, h*0.65,
		r2, r1, -w*0.25, w*0.15,
		r3, r3, -w*0.5, 0.0,
		r1, r1, -w*0.25, -w*0.15,
		r1, r1, -w*0.1, -h*0.35,
		r4, r4, w*0.1, -h*0.65,
		hw, hh,
	)
}

func bangPath(w, h float64) string {
	r := 0.15 * w
	r08 := r * 0.8
	hw := -w / 2
	hh := -h / 2
	return fmt.Sprintf(
		"M%.2f %.2f "+
			"a%.2f,%.2f 1 0,0 %.2f,%.2f "+
			"a%.2f,%.2f 1 0,0 %.2f,%.2f "+
			"a%.2f,%.2f 1 0,0 %.2f,%.2f "+
			"a%.2f,%.2f 1 0,0 %.2f,%.2f "+
			"a%.2f,%.2f 1 0,0 %.2f,%.2f "+
			"a%.2f,%.2f 1 0,0 %.2f,%.2f "+
			"a%.2f,%.2f 1 0,0 %.2f,%.2f "+
			"a%.2f,%.2f 1 0,0 %.2f,%.2f "+
			"a%.2f,%.2f 1 0,0 %.2f,%.2f "+
			"a%.2f,%.2f 1 0,0 %.2f,%.2f "+
			"a%.2f,%.2f 1 0,0 %.2f,%.2f "+
			"a%.2f,%.2f 1 0,0 %.2f,%.2f "+
			"a%.2f,%.2f 1 0,0 %.2f,%.2f "+
			"a%.2f,%.2f 1 0,0 %.2f,%.2f "+
			"H%.2f V%.2f Z",
		hw, hh,
		r, r, w*0.25, -h*0.1,
		r, r, w*0.25, 0.0,
		r, r, w*0.25, 0.0,
		r, r, w*0.25, h*0.1,
		r, r, w*0.15, h*0.33,
		r08, r08, 0.0, h*0.34,
		r, r, -w*0.15, h*0.33,
		r, r, -w*0.25, h*0.15,
		r, r, -w*0.25, 0.0,
		r, r, -w*0.25, 0.0,
		r, r, -w*0.25, -h*0.15,
		r, r, -w*0.1, -h*0.33,
		r08, r08, 0.0, -h*0.34,
		r, r, w*0.1, -h*0.33,
		hw, hh,
	)
}

func hexagonPoints(w, h float64) string {
	hw := w / 2
	hh := h / 2
	m := h / 4
	pts := [6]struct{ x, y float64 }{
		{-hw + m, -hh},
		{hw - m, -hh},
		{hw, 0},
		{hw - m, hh},
		{-hw + m, hh},
		{-hw, 0},
	}
	var sb strings.Builder
	for i, p := range pts {
		if i > 0 {
			sb.WriteString(" ")
		}
		fmt.Fprintf(&sb, "%.2f,%.2f", p.x, p.y)
	}
	return sb.String()
}

func shapeFillColor(section int, depth int, th Theme) string {
	if depth == 0 {
		return th.RootColor
	}
	colors := th.SectionColors
	if len(colors) == 0 {
		return th.EdgeStroke
	}
	return colors[section%len(colors)]
}

func shapeTextColor(depth int, th Theme) string {
	if depth == 0 {
		return th.RootText
	}
	return th.NodeText
}

func edgeStrokeWidth(depth int) float64 {
	w := 17.0 - 3.0*float64(depth)
	if w < 2 {
		w = 2
	}
	return w
}

func edgeColor(section int, th Theme) string {
	colors := th.SectionColors
	if len(colors) == 0 {
		return th.EdgeStroke
	}
	return colors[section%len(colors)]
}
