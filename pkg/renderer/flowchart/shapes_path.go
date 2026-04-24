// Path-based shape geometry helpers for Mermaid's Stage 3 extended
// shapes. Each helper returns an SVG `d` attribute positioned around
// an absolute (cx, cy) center with overall size (w, h). The path
// draws inside the layout-assigned bounding box so text centered at
// (cx, cy) sits on the glyph.
package flowchart

import (
	"fmt"
	"math"
	"strings"
)

// ---------- Cloud / Bang / Bolt --------------------------------------

// cloudPath draws a cloud silhouette as a sequence of arcs along the
// top, right, bottom, and left edges. Each arc is half-circle-ish;
// five lobes on top, three on the bottom gives a balanced outline.
func cloudPath(cx, cy, w, h float64) string {
	left := cx - w/2
	right := cx + w/2
	top := cy - h/2
	bot := cy + h/2
	// Lobe counts chosen so the arcs meet the corners evenly;
	// each arc sweeps outward (fills upward/rightward/etc.).
	const (
		topLobes  = 5.0
		botLobes  = 3.0
		sideLobes = 2.0
	)
	var b strings.Builder
	fmt.Fprintf(&b, "M%.2f,%.2f", left, top)
	// Top edge — bumps going right across a fixed y=top.
	appendCloudArcs(&b, w/topLobes, topLobes, true, left, top, +1)
	// Right edge — bumps going down along x=right.
	appendCloudArcs(&b, h/sideLobes, sideLobes, false, top, right, +1)
	// Bottom edge — bumps going left across y=bot (sweep=1 still curves outward).
	appendCloudArcs(&b, w/botLobes, botLobes, true, right, bot, -1)
	// Left edge — bumps going up along x=left.
	appendCloudArcs(&b, h/sideLobes, sideLobes, false, bot, left, -1)
	b.WriteString(" Z")
	return b.String()
}

// appendCloudArcs emits `count` consecutive SVG arc segments of equal
// radius=step/2 along one edge. When horizontal is true the arcs step
// in x along a fixed y (`fixed`); when false they step in y along a
// fixed x. `dir` is +1 or -1 for forward/backward traversal.
func appendCloudArcs(b *strings.Builder, step, count float64, horizontal bool, startAlong, fixed, dir float64) {
	r := step / 2
	for i := 0.0; i < count; i++ {
		v := startAlong + dir*(i+1)*step
		x, y := v, fixed
		if !horizontal {
			x, y = fixed, v
		}
		fmt.Fprintf(b, " A%.2f,%.2f 0 0,1 %.2f,%.2f", r, r, x, y)
	}
}

// bangPath draws a starburst outline — alternating peak/valley points
// radiating from (cx, cy). 8 peaks (16 vertices) balances "readable
// as an explosion" against keeping enough space inside the inner
// ellipse for the label to center legibly.
func bangPath(cx, cy, w, h float64) string {
	const spikes = 8
	outerRx := w / 2
	outerRy := h / 2
	innerRx := outerRx * 0.6
	innerRy := outerRy * 0.6
	var b strings.Builder
	for i := 0; i < spikes*2; i++ {
		angle := -math.Pi/2 + float64(i)*math.Pi/float64(spikes)
		rx, ry := outerRx, outerRy
		if i%2 == 1 {
			rx, ry = innerRx, innerRy
		}
		x := cx + rx*math.Cos(angle)
		y := cy + ry*math.Sin(angle)
		if i == 0 {
			fmt.Fprintf(&b, "M%.2f,%.2f", x, y)
		} else {
			fmt.Fprintf(&b, " L%.2f,%.2f", x, y)
		}
	}
	b.WriteString(" Z")
	return b.String()
}

// boltPath draws a 6-point zigzag lightning bolt filling the bounding
// box. The path follows the classic silhouette: down-and-right from
// top-left, kick back to center, down to bottom-right tip.
func boltPath(cx, cy, w, h float64) string {
	left := cx - w/2
	right := cx + w/2
	top := cy - h/2
	bot := cy + h/2
	// Control fractions tuned so the bolt has a visible thickness
	// rather than collapsing to a thin stroke.
	return fmt.Sprintf(
		"M%.2f,%.2f L%.2f,%.2f L%.2f,%.2f L%.2f,%.2f L%.2f,%.2f L%.2f,%.2f Z",
		cx-w*0.1, top,
		right, top,
		cx+w*0.1, cy,
		right, cy,
		cx-w*0.1, bot,
		left, cy+h*0.1,
	)
}

// ---------- Document family ------------------------------------------

// documentPath: rectangle whose bottom edge is an S-wave (two curves
// meeting at the midpoint). Classic "document" glyph. The `T`
// continuation reflects the first Q's control through (mid, bot),
// which naturally produces a symmetric second lobe dipping below
// the baseline without needing a second explicit control point.
func documentPath(cx, cy, w, h float64) string {
	left := cx - w/2
	right := cx + w/2
	top := cy - h/2
	bot := cy + h/2
	waveAmp := h * 0.08
	mid := cx
	return fmt.Sprintf(
		"M%.2f,%.2f L%.2f,%.2f L%.2f,%.2f Q%.2f,%.2f %.2f,%.2f T%.2f,%.2f Z",
		left, top,
		right, top,
		right, bot,
		(right+mid)/2, bot-2*waveAmp,
		mid, bot,
		left, bot,
	)
}

// ---------- Delay / Horizontal Cylinder / Curved Trapezoid -----------

// delayPath: rectangle with the RIGHT edge rounded like a half-stadium
// — a.k.a. "half-rounded-rectangle" / Mermaid `delay`.
func delayPath(cx, cy, w, h float64) string {
	left := cx - w/2
	right := cx + w/2
	top := cy - h/2
	bot := cy + h/2
	r := h / 2
	return fmt.Sprintf(
		"M%.2f,%.2f L%.2f,%.2f A%.2f,%.2f 0 0,1 %.2f,%.2f L%.2f,%.2f Z",
		left, top,
		right-r, top,
		r, r, right-r, bot,
		left, bot,
	)
}

// horizontalCylinderPath: cylinder lying on its side. Left cap is a
// backward-facing ellipse; body is the middle; right cap is a
// forward-facing ellipse overdrawn so the rim reads. Emits two
// subpaths in a single `d`; see svgutil.CylinderPath for why relying
// on SVG's default nonzero fill rule produces the rim overlay.
func horizontalCylinderPath(cx, cy, w, h float64) string {
	left := cx - w/2
	right := cx + w/2
	top := cy - h/2
	bot := cy + h/2
	rx := w * 0.1
	leftCap := left + rx
	rightCap := right - rx
	return fmt.Sprintf(
		"M%.2f,%.2f L%.2f,%.2f A%.2f,%.2f 0 0,1 %.2f,%.2f L%.2f,%.2f A%.2f,%.2f 0 0,1 %.2f,%.2f Z"+
			" M%.2f,%.2f A%.2f,%.2f 0 0,0 %.2f,%.2f",
		leftCap, top,
		rightCap, top,
		rx, h/2, rightCap, bot,
		leftCap, bot,
		rx, h/2, leftCap, top,
		rightCap, top,
		rx, h/2, rightCap, bot,
	)
}

// curvedTrapezoidPath: trapezoid with curved left/right sides, like
// an old CRT screen. Straight top/bottom; sides bow outward slightly.
func curvedTrapezoidPath(cx, cy, w, h float64) string {
	left := cx - w/2
	right := cx + w/2
	top := cy - h/2
	bot := cy + h/2
	bow := w * 0.05
	return fmt.Sprintf(
		"M%.2f,%.2f L%.2f,%.2f Q%.2f,%.2f %.2f,%.2f L%.2f,%.2f Q%.2f,%.2f %.2f,%.2f Z",
		left, top,
		right, top,
		right+bow, cy,
		right, bot,
		left, bot,
		left-bow, cy,
		left, top,
	)
}

// bowTieRectPath: rectangle where the left/right edges are pinched in
// at the middle — Mermaid's `stored-data` glyph.
func bowTieRectPath(cx, cy, w, h float64) string {
	left := cx - w/2
	right := cx + w/2
	top := cy - h/2
	bot := cy + h/2
	pinch := w * 0.1
	return fmt.Sprintf(
		"M%.2f,%.2f L%.2f,%.2f Q%.2f,%.2f %.2f,%.2f L%.2f,%.2f Q%.2f,%.2f %.2f,%.2f Z",
		left, top,
		right, top,
		right-pinch, cy,
		right, bot,
		left, bot,
		left+pinch, cy,
		left, top,
	)
}

// ---------- Tagged / DataStore / Braces ------------------------------

// taggedRectPath: rectangle with a small tag flag in the bottom-right
// corner (a square notch poking out past the main rect).
func taggedRectPath(cx, cy, w, h float64) string {
	left := cx - w/2
	right := cx + w/2
	top := cy - h/2
	bot := cy + h/2
	tag := math.Min(w, h) * 0.2
	return fmt.Sprintf(
		"M%.2f,%.2f L%.2f,%.2f L%.2f,%.2f L%.2f,%.2f L%.2f,%.2f L%.2f,%.2f Z",
		left, top,
		right, top,
		right, bot,
		right-tag, bot,
		right-tag, bot+tag,
		left, bot,
	)
}

// datastorePath: Mermaid `data-store` — two parallel curved arcs on
// the left side of a rectangle, making the shape look like a pipe or
// lying cylinder with a gap. Like horizontalCylinderPath, this emits
// two subpaths in one `d` and relies on the default nonzero fill
// rule to produce the interior arc overlay.
func datastorePath(cx, cy, w, h float64) string {
	left := cx - w/2
	right := cx + w/2
	top := cy - h/2
	bot := cy + h/2
	rx := w * 0.08
	inner := left + rx*2
	return fmt.Sprintf(
		"M%.2f,%.2f L%.2f,%.2f L%.2f,%.2f L%.2f,%.2f A%.2f,%.2f 0 0,1 %.2f,%.2f Z"+
			" M%.2f,%.2f A%.2f,%.2f 0 0,0 %.2f,%.2f",
		inner, top,
		right, top,
		right, bot,
		inner, bot,
		rx, h/2, inner, top,
		inner, top,
		rx, h/2, inner, bot,
	)
}

// bracePath: single left curly brace `{` occupying the box. The brace
// is drawn as a stroked path (the renderer sets fill:none for this
// shape family).
func bracePath(cx, cy, w, h float64, right bool) string {
	top := cy - h/2
	bot := cy + h/2
	thick := w * 0.3
	stem := cx
	tipX := cx + thick
	if right {
		stem = cx
		tipX = cx - thick
	}
	// Two C-curves meeting at the tip (cy) — the tip pokes out.
	return fmt.Sprintf(
		"M%.2f,%.2f Q%.2f,%.2f %.2f,%.2f Q%.2f,%.2f %.2f,%.2f",
		stem, top,
		tipX, (top+cy)/2,
		tipX, cy,
		tipX, (cy+bot)/2,
		stem, bot,
	)
}

// bracesPath: both `{ }` drawn around the layout box. A single `d`
// string; the renderer still emits one <path>.
func bracesPath(cx, cy, w, h float64) string {
	left := bracePath(cx-w/2, cy, w/4, h, false)
	right := bracePath(cx+w/2, cy, w/4, h, true)
	return left + " " + right
}

// cornerTagPath draws the small triangular pennant overlay used by
// `TaggedDocument`. The pennant hangs off the bottom-right corner of
// the bounding box with leg length = tag.
func cornerTagPath(cx, cy, w, h, tag float64) string {
	right := cx + w/2
	bot := cy + h/2
	return fmt.Sprintf(
		"M%.2f,%.2f L%.2f,%.2f L%.2f,%.2f Z",
		right, bot-tag,
		right, bot+tag,
		right-tag, bot,
	)
}
