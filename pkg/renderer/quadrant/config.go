package quadrant

// Config mirrors the `quadrantChart` block users can supply via
// `%%{init: {quadrantChart: {...}}}%%`. Field names match the
// documented config keys 1:1 so a future init-directive parser
// can decode JSON straight into this struct.
//
// AxisPosition fields select which side of the plot each axis
// label sits on. "auto" lets the renderer flip labels when only
// the top/bottom (resp. left/right) quadrants carry data — the
// behaviour the spec describes for sparsely populated charts.
type Config struct {
	ChartWidth  float64
	ChartHeight float64

	TitleFontSize float64
	TitlePadding  float64

	QuadrantPadding              float64
	QuadrantTextTopPadding       float64
	QuadrantLabelFontSize        float64
	QuadrantInternalBorderStroke float64
	QuadrantExternalBorderStroke float64

	XAxisLabelPadding  float64
	XAxisLabelFontSize float64
	XAxisPosition      AxisPosition

	YAxisLabelPadding  float64
	YAxisLabelFontSize float64
	YAxisPosition      AxisPosition

	PointTextPadding  float64
	PointLabelFontSize float64
	PointRadius       float64
}

// AxisPosition controls which edge of the plot an axis label
// anchors to. Default is the auto-detect fallback (the renderer
// picks based on which quadrants are populated).
type AxisPosition int8

const (
	AxisPositionAuto AxisPosition = iota
	AxisPositionTop
	AxisPositionBottom
	AxisPositionLeft
	AxisPositionRight
)

// DefaultConfig returns the layout / typography knobs that
// reproduce the historical mmgo renderer geometry. Callers
// override individual fields via Options.Config and
// resolveConfig fills the rest.
func DefaultConfig() Config {
	return Config{
		// Keep the historical mmgo plotSide of 400 (Mermaid's
		// default is 500; mmgo never matched it and the example
		// snapshots are pinned to 400×400).
		ChartWidth:                   400,
		ChartHeight:                  400,
		TitleFontSize:                15,
		TitlePadding:                 0, // legacy renderer used `pageMarginY + titleH/2` directly; keep parity
		QuadrantPadding:              5,
		QuadrantTextTopPadding:       5,
		QuadrantLabelFontSize:        13,
		QuadrantInternalBorderStroke: 1,
		QuadrantExternalBorderStroke: 1.5,
		XAxisLabelPadding:            20, // matches the legacy `axisLabelGap` constant
		XAxisLabelFontSize:           12,
		XAxisPosition:                AxisPositionAuto,
		YAxisLabelPadding:            20,
		YAxisLabelFontSize:           12,
		YAxisPosition:                AxisPositionAuto,
		PointTextPadding:             5,
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
	mergeF(&c.QuadrantPadding, o.QuadrantPadding)
	mergeF(&c.QuadrantTextTopPadding, o.QuadrantTextTopPadding)
	mergeF(&c.QuadrantLabelFontSize, o.QuadrantLabelFontSize)
	mergeF(&c.QuadrantInternalBorderStroke, o.QuadrantInternalBorderStroke)
	mergeF(&c.QuadrantExternalBorderStroke, o.QuadrantExternalBorderStroke)
	mergeF(&c.XAxisLabelPadding, o.XAxisLabelPadding)
	mergeF(&c.XAxisLabelFontSize, o.XAxisLabelFontSize)
	mergeF(&c.YAxisLabelPadding, o.YAxisLabelPadding)
	mergeF(&c.YAxisLabelFontSize, o.YAxisLabelFontSize)
	mergeF(&c.PointTextPadding, o.PointTextPadding)
	mergeF(&c.PointLabelFontSize, o.PointLabelFontSize)
	mergeF(&c.PointRadius, o.PointRadius)
	if o.XAxisPosition != AxisPositionAuto {
		c.XAxisPosition = o.XAxisPosition
	}
	if o.YAxisPosition != AxisPositionAuto {
		c.YAxisPosition = o.YAxisPosition
	}
	return c
}
