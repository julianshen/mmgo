package xychart

import "github.com/julianshen/mmgo/pkg/renderer/svgutil"

// ChartOrientation selects vertical vs. horizontal layout. The Auto
// default lets the renderer follow the AST's `Horizontal` flag (set
// by `xychart-beta horizontal` at the source level) and only switches
// when a caller explicitly overrides via Options.Config.
type ChartOrientation int8

const (
	OrientationAuto ChartOrientation = iota
	OrientationVertical
	OrientationHorizontal
)

// AxisConfig mirrors Mermaid's per-axis `xAxis` / `yAxis` knobs.
// Numeric defaults follow the spec page; ShowXxx defaults are tri-state
// via *bool because Go's zero-bool is `false` and we need to
// distinguish "not set" from "explicitly off".
type AxisConfig struct {
	ShowLabel     *bool
	LabelFontSize float64
	LabelPadding  float64

	ShowTitle     *bool
	TitleFontSize float64
	TitlePadding  float64

	ShowTick   *bool
	TickLength float64
	TickWidth  float64

	ShowAxisLine  *bool
	AxisLineWidth float64
}

// Config mirrors `themeVariables.xyChart.*` and the per-axis surface
// from https://mermaid.js.org/syntax/xyChart.html. Pointer-typed bool
// fields preserve the "not set" state so resolveConfig can apply the
// spec default of `true` without callers having to set it explicitly;
// use BoolPtr to construct one tersely.
type Config struct {
	Width                   float64
	Height                  float64
	TitlePadding            float64
	TitleFontSize           float64
	ShowTitle               *bool
	ChartOrientation        ChartOrientation
	ShowDataLabel           *bool
	ShowDataLabelOutsideBar *bool

	XAxis AxisConfig
	YAxis AxisConfig
}

// BoolPtr returns a *bool pointing at b. Needed because zero *bool
// means "inherit default", so an explicit `&false` is the only way to
// override a default-true Show* flag to off.
func BoolPtr(b bool) *bool { return &b }

// DefaultConfig returns the spec defaults from the Mermaid xyChart
// docs.
func DefaultConfig() Config {
	return Config{
		Width:                   700,
		Height:                  500,
		TitlePadding:            10,
		TitleFontSize:           20,
		ShowTitle:               BoolPtr(true),
		ChartOrientation:        OrientationAuto,
		ShowDataLabel:           BoolPtr(false),
		ShowDataLabelOutsideBar: BoolPtr(false),
		XAxis:                   defaultAxisConfig(),
		YAxis:                   defaultAxisConfig(),
	}
}

func defaultAxisConfig() AxisConfig {
	return AxisConfig{
		ShowLabel:     BoolPtr(true),
		LabelFontSize: 14,
		LabelPadding:  5,
		ShowTitle:     BoolPtr(true),
		TitleFontSize: 16,
		TitlePadding:  5,
		ShowTick:      BoolPtr(true),
		TickLength:    5,
		TickWidth:     2,
		ShowAxisLine:  BoolPtr(true),
		AxisLineWidth: 2,
	}
}

func resolveConfig(opts *Options) Config {
	c := DefaultConfig()
	if opts == nil {
		return c
	}
	// opts.FontSize re-scales the axis label/title fonts (and chart
	// title) before opts.Config is merged, so Config field overrides
	// still win.
	if opts.FontSize > 0 {
		c.TitleFontSize = opts.FontSize + 2
		c.XAxis.LabelFontSize = opts.FontSize - 2
		c.XAxis.TitleFontSize = opts.FontSize
		c.YAxis.LabelFontSize = opts.FontSize - 2
		c.YAxis.TitleFontSize = opts.FontSize
	}
	o := opts.Config
	svgutil.MergeFloat(&c.Width, o.Width)
	svgutil.MergeFloat(&c.Height, o.Height)
	svgutil.MergeFloat(&c.TitlePadding, o.TitlePadding)
	svgutil.MergeFloat(&c.TitleFontSize, o.TitleFontSize)
	svgutil.MergeBoolPtr(&c.ShowTitle, o.ShowTitle)
	if o.ChartOrientation != OrientationAuto {
		c.ChartOrientation = o.ChartOrientation
	}
	svgutil.MergeBoolPtr(&c.ShowDataLabel, o.ShowDataLabel)
	svgutil.MergeBoolPtr(&c.ShowDataLabelOutsideBar, o.ShowDataLabelOutsideBar)
	c.XAxis = mergeAxis(c.XAxis, o.XAxis)
	c.YAxis = mergeAxis(c.YAxis, o.YAxis)
	return c
}

func mergeAxis(dst, src AxisConfig) AxisConfig {
	svgutil.MergeBoolPtr(&dst.ShowLabel, src.ShowLabel)
	svgutil.MergeFloat(&dst.LabelFontSize, src.LabelFontSize)
	svgutil.MergeFloat(&dst.LabelPadding, src.LabelPadding)
	svgutil.MergeBoolPtr(&dst.ShowTitle, src.ShowTitle)
	svgutil.MergeFloat(&dst.TitleFontSize, src.TitleFontSize)
	svgutil.MergeFloat(&dst.TitlePadding, src.TitlePadding)
	svgutil.MergeBoolPtr(&dst.ShowTick, src.ShowTick)
	svgutil.MergeFloat(&dst.TickLength, src.TickLength)
	svgutil.MergeFloat(&dst.TickWidth, src.TickWidth)
	svgutil.MergeBoolPtr(&dst.ShowAxisLine, src.ShowAxisLine)
	svgutil.MergeFloat(&dst.AxisLineWidth, src.AxisLineWidth)
	return dst
}

// flag returns the value of *p, or fallback when p is nil. Centralises
// the tri-state unwrap so every render-time check can stay terse.
func flag(p *bool, fallback bool) bool {
	if p == nil {
		return fallback
	}
	return *p
}
