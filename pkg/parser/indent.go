package parser

// TabWidth is the visual width of a '\t' character when computing
// indentation depth. Four matches Mermaid's documentation and the
// common editor default.
const TabWidth = 4

// IndentWidth returns the visual indentation of line: one column per
// space, TabWidth columns per tab. Used by hierarchical parsers
// (kanban, mindmap) that treat indentation as structural. Only the
// relative ordering matters to callers; the absolute value is
// meaningful only in so far as two lines at equal indent share a
// parent.
func IndentWidth(line string) int {
	w := 0
	for _, c := range line {
		switch c {
		case ' ':
			w++
		case '\t':
			w += TabWidth
		default:
			return w
		}
	}
	return w
}
