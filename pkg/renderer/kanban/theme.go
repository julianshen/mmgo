package kanban

import "github.com/julianshen/mmgo/pkg/renderer/svgutil"

// Theme holds kanban color surfaces.
//
// Priority* fields drive the left-edge stripe drawn on cards
// whose `priority:` metadata matches one of the four documented
// levels (`Very High`, `High`, `Low`, `Very Low`). When unset the
// renderer falls back to the default red→amber→sky→slate ramp.
type Theme struct {
	Background       string
	ColumnFill       string
	ColumnStroke     string
	ColumnTitle      string
	CardFill         string
	CardStroke       string
	CardText         string
	MetaText         string
	PriorityVeryHigh string
	PriorityHigh     string
	PriorityLow      string
	PriorityVeryLow  string
	// TicketLink is the color used for ticket-id text when a card
	// gains an `<a href>` wrap because both `ticket:` metadata and
	// the diagram's TicketBaseURL are present.
	TicketLink string
}

func DefaultTheme() Theme {
	return Theme{
		Background:       "#fff",
		ColumnFill:       "#f3f4f6",
		ColumnStroke:     "#d1d5db",
		ColumnTitle:      "#1f2937",
		CardFill:         "#ffffff",
		CardStroke:       "#d1d5db",
		CardText:         "#111827",
		MetaText:         "#6b7280",
		PriorityVeryHigh: "#dc2626",
		PriorityHigh:     "#f97316",
		PriorityLow:      "#0ea5e9",
		PriorityVeryLow:  "#94a3b8",
		TicketLink:       "#2563eb",
	}
}

func resolveTheme(opts *Options) Theme {
	th := DefaultTheme()
	if opts == nil {
		return th
	}
	svgutil.MergeStr(&th.Background, opts.Theme.Background)
	svgutil.MergeStr(&th.ColumnFill, opts.Theme.ColumnFill)
	svgutil.MergeStr(&th.ColumnStroke, opts.Theme.ColumnStroke)
	svgutil.MergeStr(&th.ColumnTitle, opts.Theme.ColumnTitle)
	svgutil.MergeStr(&th.CardFill, opts.Theme.CardFill)
	svgutil.MergeStr(&th.CardStroke, opts.Theme.CardStroke)
	svgutil.MergeStr(&th.CardText, opts.Theme.CardText)
	svgutil.MergeStr(&th.MetaText, opts.Theme.MetaText)
	svgutil.MergeStr(&th.PriorityVeryHigh, opts.Theme.PriorityVeryHigh)
	svgutil.MergeStr(&th.PriorityHigh, opts.Theme.PriorityHigh)
	svgutil.MergeStr(&th.PriorityLow, opts.Theme.PriorityLow)
	svgutil.MergeStr(&th.PriorityVeryLow, opts.Theme.PriorityVeryLow)
	svgutil.MergeStr(&th.TicketLink, opts.Theme.TicketLink)
	return th
}

// priorityColor returns the theme color for a documented priority
// level, or "" when the value isn't one of the four spec strings.
// Comparison is case-insensitive but space-sensitive ("Very High"
// vs "veryhigh") to match Mermaid's behaviour.
func (t Theme) priorityColor(level string) string {
	switch level {
	case "Very High", "very high", "VERY HIGH":
		return t.PriorityVeryHigh
	case "High", "high", "HIGH":
		return t.PriorityHigh
	case "Low", "low", "LOW":
		return t.PriorityLow
	case "Very Low", "very low", "VERY LOW":
		return t.PriorityVeryLow
	}
	return ""
}
