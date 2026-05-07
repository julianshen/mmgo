package quadrant

import "github.com/julianshen/mmgo/pkg/renderer/svgutil"

// Config mirrors the `quadrantChart` block users can supply via
// `%%{init: {quadrantChart: {...}}}%%`.
//
// XAxisPosition / YAxisPosition use distinct types so an X axis
// can't be set to Left/Right and a Y axis can't be set to
// Top/Bottom — the compiler rejects mismatched pairings rather
// than the renderer silently ignoring them.
type Config struct {
	ChartWidth  float64
	ChartHeight float64

	TitleFontSize float64
	TitlePadding  float64

	// QuadrantPadding / QuadrantTextTopPadding mirror Mermaid's
	// per-quadrant spacing knobs. Currently parsed onto the Config
	// for spec parity but the renderer does not yet apply them —
	// Phase C will wire them once the inner-quadrant layout pass
	// lands.
	QuadrantPadding              float64
	QuadrantTextTopPadding       float64
	QuadrantLabelFontSize        float64
	QuadrantInternalBorderStroke float64
	QuadrantExternalBorderStroke float64

	XAxisLabelPadding  float64
	XAxisLabelFontSize float64
	XAxisPosition      XAxisPosition

	YAxisLabelPadding  float64
	YAxisLabelFontSize float64
	YAxisPosition      YAxisPosition

	// PointTextPadding controls the gap between a point's circle
	// edge and its label. Mirrors the Mermaid config knob.
	PointTextPadding   float64
	PointLabelFontSize float64
	PointRadius        float64
}

// Note on zero values: numeric Config fields currently treat `0`
// as "inherit default" via resolveConfig's merge-on-positive
// rule. That means a caller can't explicitly request a zero
// padding / radius / stroke width — the default wins. For most
// of these fields zero is a degenerate value (invisible points,
// no border) so the limitation is acceptable; if a caller needs
// to override to zero, set the value to the smallest meaningful
// positive number for now. A future API revision may switch to
// pointer-typed fields to disambiguate.

// XAxisPosition selects which horizontal edge the X-axis labels
// anchor to. The Auto default lets the renderer flip the labels
// to the top when only the bottom-half quadrants carry data.
type XAxisPosition int8

const (
	XAxisAuto XAxisPosition = iota
	XAxisTop
	XAxisBottom
)

// YAxisPosition is the Y-axis counterpart to XAxisPosition. Auto
// flips the rotated title to the right side when the left half
// of the plot is empty.
type YAxisPosition int8

const (
	YAxisAuto YAxisPosition = iota
	YAxisLeft
	YAxisRight
)

// QuadrantIndex names the math-convention indices into
// Theme.Quadrants so callers don't have to remember which slot
// corresponds to which quadrant.
const (
	QuadrantQ1 = 0 // top-right
	QuadrantQ2 = 1 // top-left
	QuadrantQ3 = 2 // bottom-left
	QuadrantQ4 = 3 // bottom-right
)

// DefaultConfig returns the layout / typography knobs that
// reproduce the historical mmgo renderer geometry. Callers
// override individual fields via Options.Config and
// resolveConfig fills the rest.
func DefaultConfig() Config {
	return Config{
		// 400 (vs Mermaid's 500) keeps the existing example
		// snapshots stable — most users don't override these.
		ChartWidth:                   400,
		ChartHeight:                  400,
		TitleFontSize:                15,
		TitlePadding:                 0,
		QuadrantPadding:              5,
		QuadrantTextTopPadding:       5,
		QuadrantLabelFontSize:        13,
		QuadrantInternalBorderStroke: 1,
		QuadrantExternalBorderStroke: 1.5,
		XAxisLabelPadding:            20,
		XAxisLabelFontSize:           12,
		XAxisPosition:                XAxisAuto,
		YAxisLabelPadding:            20,
		YAxisLabelFontSize:           12,
		YAxisPosition:                YAxisAuto,
		PointTextPadding:             4,
		PointLabelFontSize:           11,
		PointRadius:                  7,
	}
}

// resolveConfig overlays opts.Config on top of DefaultConfig().
// Zero-valued numeric fields keep the default, mirroring the
// merge-on-non-zero convention used by Theme.
func resolveConfig(opts *Options) Config {
	c := DefaultConfig()
	if opts == nil {
		return c
	}
	o := opts.Config
	// If a caller sets only ChartWidth or only ChartHeight, mirror
	// the explicit value into the other so the square plot picks
	// it up. Without this a single `Config.ChartWidth = 700` would
	// be capped by the default ChartHeight of 400 once min() runs
	// at render time, making one-field overrides silently inert.
	oW, oH := o.ChartWidth, o.ChartHeight
	if oW > 0 && oH == 0 {
		oH = oW
	}
	if oH > 0 && oW == 0 {
		oW = oH
	}
	svgutil.MergeFloat(&c.ChartWidth, oW)
	svgutil.MergeFloat(&c.ChartHeight, oH)
	svgutil.MergeFloat(&c.TitleFontSize, o.TitleFontSize)
	svgutil.MergeFloat(&c.TitlePadding, o.TitlePadding)
	svgutil.MergeFloat(&c.QuadrantPadding, o.QuadrantPadding)
	svgutil.MergeFloat(&c.QuadrantTextTopPadding, o.QuadrantTextTopPadding)
	svgutil.MergeFloat(&c.QuadrantLabelFontSize, o.QuadrantLabelFontSize)
	svgutil.MergeFloat(&c.QuadrantInternalBorderStroke, o.QuadrantInternalBorderStroke)
	svgutil.MergeFloat(&c.QuadrantExternalBorderStroke, o.QuadrantExternalBorderStroke)
	svgutil.MergeFloat(&c.XAxisLabelPadding, o.XAxisLabelPadding)
	svgutil.MergeFloat(&c.XAxisLabelFontSize, o.XAxisLabelFontSize)
	svgutil.MergeFloat(&c.YAxisLabelPadding, o.YAxisLabelPadding)
	svgutil.MergeFloat(&c.YAxisLabelFontSize, o.YAxisLabelFontSize)
	svgutil.MergeFloat(&c.PointTextPadding, o.PointTextPadding)
	svgutil.MergeFloat(&c.PointLabelFontSize, o.PointLabelFontSize)
	svgutil.MergeFloat(&c.PointRadius, o.PointRadius)
	if o.XAxisPosition != XAxisAuto {
		c.XAxisPosition = o.XAxisPosition
	}
	if o.YAxisPosition != YAxisAuto {
		c.YAxisPosition = o.YAxisPosition
	}
	return c
}
