package class

// Theme holds per-color-surface values the class renderer consumes.
// Mirrors the shape of pkg/renderer/flowchart/theme.go so the SVG
// dispatcher can map the shared config.ThemeColors palette uniformly.
// Unset fields fall back to DefaultTheme values via resolveTheme.
type Theme struct {
	NodeFill       string
	NodeStroke     string
	NodeText       string
	AnnotationText string // e.g. the «interface» italic tag
	EdgeStroke     string
	EdgeText       string
	Background     string
	NoteFill       string // sticky-note background (yellow by default)
	NoteStroke     string
	NoteText       string
	NamespaceFill   string // namespace bounding-rect background
	NamespaceStroke string
	NamespaceText   string
}

// DefaultTheme returns the Mermaid-classic class-diagram palette
// (light purple boxes, dark purple borders, black text). Used when
// no explicit theme is wired through Options.
func DefaultTheme() Theme {
	return Theme{
		NodeFill:       "#ECECFF",
		NodeStroke:     "#9370DB",
		NodeText:       "#333",
		AnnotationText: "#999",
		EdgeStroke:     "#333",
		EdgeText:       "#333",
		Background:     "#fff",
		NoteFill:       "#fff5ad",
		NoteStroke:     "#aaaa33",
		NoteText:       "#333",
		NamespaceFill:   "#f7f7ff",
		NamespaceStroke: "#9370DB",
		NamespaceText:   "#555",
	}
}

func resolveTheme(opts *Options) Theme {
	th := DefaultTheme()
	if opts == nil {
		return th
	}
	if opts.Theme.NodeFill != "" {
		th.NodeFill = opts.Theme.NodeFill
	}
	if opts.Theme.NodeStroke != "" {
		th.NodeStroke = opts.Theme.NodeStroke
	}
	if opts.Theme.NodeText != "" {
		th.NodeText = opts.Theme.NodeText
	}
	if opts.Theme.AnnotationText != "" {
		th.AnnotationText = opts.Theme.AnnotationText
	}
	if opts.Theme.EdgeStroke != "" {
		th.EdgeStroke = opts.Theme.EdgeStroke
	}
	if opts.Theme.EdgeText != "" {
		th.EdgeText = opts.Theme.EdgeText
	}
	if opts.Theme.Background != "" {
		th.Background = opts.Theme.Background
	}
	if opts.Theme.NoteFill != "" {
		th.NoteFill = opts.Theme.NoteFill
	}
	if opts.Theme.NoteStroke != "" {
		th.NoteStroke = opts.Theme.NoteStroke
	}
	if opts.Theme.NoteText != "" {
		th.NoteText = opts.Theme.NoteText
	}
	if opts.Theme.NamespaceFill != "" {
		th.NamespaceFill = opts.Theme.NamespaceFill
	}
	if opts.Theme.NamespaceStroke != "" {
		th.NamespaceStroke = opts.Theme.NamespaceStroke
	}
	if opts.Theme.NamespaceText != "" {
		th.NamespaceText = opts.Theme.NamespaceText
	}
	return th
}
