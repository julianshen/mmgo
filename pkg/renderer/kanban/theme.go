package kanban

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
	if opts.Theme.Background != "" {
		th.Background = opts.Theme.Background
	}
	if opts.Theme.ColumnFill != "" {
		th.ColumnFill = opts.Theme.ColumnFill
	}
	if opts.Theme.ColumnStroke != "" {
		th.ColumnStroke = opts.Theme.ColumnStroke
	}
	if opts.Theme.ColumnTitle != "" {
		th.ColumnTitle = opts.Theme.ColumnTitle
	}
	if opts.Theme.CardFill != "" {
		th.CardFill = opts.Theme.CardFill
	}
	if opts.Theme.CardStroke != "" {
		th.CardStroke = opts.Theme.CardStroke
	}
	if opts.Theme.CardText != "" {
		th.CardText = opts.Theme.CardText
	}
	if opts.Theme.MetaText != "" {
		th.MetaText = opts.Theme.MetaText
	}
	if opts.Theme.PriorityVeryHigh != "" {
		th.PriorityVeryHigh = opts.Theme.PriorityVeryHigh
	}
	if opts.Theme.PriorityHigh != "" {
		th.PriorityHigh = opts.Theme.PriorityHigh
	}
	if opts.Theme.PriorityLow != "" {
		th.PriorityLow = opts.Theme.PriorityLow
	}
	if opts.Theme.PriorityVeryLow != "" {
		th.PriorityVeryLow = opts.Theme.PriorityVeryLow
	}
	if opts.Theme.TicketLink != "" {
		th.TicketLink = opts.Theme.TicketLink
	}
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
