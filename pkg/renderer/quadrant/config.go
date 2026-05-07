package quadrant

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

	QuadrantLabelFontSize        float64
	QuadrantInternalBorderStroke float64
	QuadrantExternalBorderStroke float64

	XAxisLabelPadding  float64
	XAxisLabelFontSize float64
	XAxisPosition      XAxisPosition

	YAxisLabelPadding  float64
	YAxisLabelFontSize float64
	YAxisPosition      YAxisPosition

	PointLabelFontSize float64
	PointRadius        float64
}

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
		ChartWidth:  400,
		ChartHeight: 400,
		TitleFontSize:                15,
		TitlePadding:                 0,
		QuadrantLabelFontSize:        13,
		QuadrantInternalBorderStroke: 1,
		QuadrantExternalBorderStroke: 1.5,
		XAxisLabelPadding:            20,
		XAxisLabelFontSize:           12,
		XAxisPosition:                XAxisAuto,
		YAxisLabelPadding:            20,
		YAxisLabelFontSize:           12,
		YAxisPosition:                YAxisAuto,
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
	mergeF := func(dst *float64, src float64) {
		if src > 0 {
			*dst = src
		}
	}
	o := opts.Config
	mergeF(&c.ChartWidth, o.ChartWidth)
	mergeF(&c.ChartHeight, o.ChartHeight)
	mergeF(&c.TitleFontSize, o.TitleFontSize)
	mergeF(&c.TitlePadding, o.TitlePadding)
	mergeF(&c.QuadrantLabelFontSize, o.QuadrantLabelFontSize)
	mergeF(&c.QuadrantInternalBorderStroke, o.QuadrantInternalBorderStroke)
	mergeF(&c.QuadrantExternalBorderStroke, o.QuadrantExternalBorderStroke)
	mergeF(&c.XAxisLabelPadding, o.XAxisLabelPadding)
	mergeF(&c.XAxisLabelFontSize, o.XAxisLabelFontSize)
	mergeF(&c.YAxisLabelPadding, o.YAxisLabelPadding)
	mergeF(&c.YAxisLabelFontSize, o.YAxisLabelFontSize)
	mergeF(&c.PointLabelFontSize, o.PointLabelFontSize)
	mergeF(&c.PointRadius, o.PointRadius)
	if o.XAxisPosition != XAxisAuto {
		c.XAxisPosition = o.XAxisPosition
	}
	if o.YAxisPosition != YAxisAuto {
		c.YAxisPosition = o.YAxisPosition
	}
	return c
}
