package kanban

// Theme holds kanban color surfaces.
type Theme struct {
	Background   string
	ColumnFill   string
	ColumnStroke string
	ColumnTitle  string
	CardFill     string
	CardStroke   string
	CardText     string
	MetaText     string
}

func DefaultTheme() Theme {
	return Theme{
		Background:   "#fff",
		ColumnFill:   "#f3f4f6",
		ColumnStroke: "#d1d5db",
		ColumnTitle:  "#1f2937",
		CardFill:     "#ffffff",
		CardStroke:   "#d1d5db",
		CardText:     "#111827",
		MetaText:     "#6b7280",
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
	return th
}
