package class

import "github.com/julianshen/mmgo/pkg/renderer/svgutil"

// Theme holds per-color-surface values the class renderer consumes.
// Mirrors the shape of pkg/renderer/flowchart/theme.go so the SVG
// dispatcher can map the shared config.ThemeColors palette uniformly.
// Unset fields fall back to DefaultTheme values via resolveTheme.
type Theme struct {
	NodeFill        string
	NodeStroke      string
	NodeText        string
	AnnotationText  string // e.g. the «interface» italic tag
	EdgeStroke      string
	EdgeText        string
	Background      string
	NoteFill        string // sticky-note background (yellow by default)
	NoteStroke      string
	NoteText        string
	NamespaceFill   string // namespace bounding-rect background
	NamespaceStroke string
	NamespaceText   string
}

// DefaultTheme returns the Mermaid-classic class-diagram palette
// (light purple boxes, dark purple borders, black text). Used when
// no explicit theme is wired through Options.
func DefaultTheme() Theme {
	return Theme{
		NodeFill:        "#ECECFF",
		NodeStroke:      "#9370DB",
		NodeText:        "#333",
		AnnotationText:  "#999",
		EdgeStroke:      "#333",
		EdgeText:        "#333",
		Background:      "#fff",
		NoteFill:        "#fff5ad",
		NoteStroke:      "#aaaa33",
		NoteText:        "#333",
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
	svgutil.MergeStr(&th.NodeFill, opts.Theme.NodeFill)
	svgutil.MergeStr(&th.NodeStroke, opts.Theme.NodeStroke)
	svgutil.MergeStr(&th.NodeText, opts.Theme.NodeText)
	svgutil.MergeStr(&th.AnnotationText, opts.Theme.AnnotationText)
	svgutil.MergeStr(&th.EdgeStroke, opts.Theme.EdgeStroke)
	svgutil.MergeStr(&th.EdgeText, opts.Theme.EdgeText)
	svgutil.MergeStr(&th.Background, opts.Theme.Background)
	svgutil.MergeStr(&th.NoteFill, opts.Theme.NoteFill)
	svgutil.MergeStr(&th.NoteStroke, opts.Theme.NoteStroke)
	svgutil.MergeStr(&th.NoteText, opts.Theme.NoteText)
	svgutil.MergeStr(&th.NamespaceFill, opts.Theme.NamespaceFill)
	svgutil.MergeStr(&th.NamespaceStroke, opts.Theme.NamespaceStroke)
	svgutil.MergeStr(&th.NamespaceText, opts.Theme.NamespaceText)
	return th
}
