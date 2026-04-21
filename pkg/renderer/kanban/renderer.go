// Package kanban renders a KanbanDiagram as side-by-side columns,
// one per section. Tasks within each section stack vertically as
// rounded cards. Card height scales with text length via naive word
// wrapping to columnWidth.
package kanban

import (
	"encoding/xml"
	"fmt"
	"sort"
	"strings"

	"github.com/julianshen/mmgo/pkg/diagram"
	"github.com/julianshen/mmgo/pkg/textmeasure"
)

type Options struct {
	FontSize float64
	Theme    Theme
}

const (
	defaultFontSize = 13.0
	marginX         = 30.0
	marginY         = 30.0
	columnWidth     = 220.0
	columnGap       = 20.0
	columnHeaderH   = 40.0
	cardPadding     = 10.0
	cardGap         = 10.0
	cardRadius      = 6.0
	lineHeight      = 18.0
	metaLineHeight  = 15.0

)

func Render(d *diagram.KanbanDiagram, opts *Options) ([]byte, error) {
	if d == nil {
		return nil, fmt.Errorf("kanban render: diagram is nil")
	}
	fontSize := defaultFontSize
	if opts != nil && opts.FontSize > 0 {
		fontSize = opts.FontSize
	}
	th := resolveTheme(opts)

	nCols := len(d.Sections)
	viewW := 2*marginX + float64(nCols)*columnWidth + float64(max(nCols-1, 0))*columnGap
	if nCols == 0 {
		viewW = 2 * marginX
	}

	// Layout pass: compute per-card wrapped text and heights so the
	// column heights can be determined before any drawing.
	type cardLayout struct {
		lines    []string
		meta     []string
		height   float64
	}
	type colLayout struct {
		cards  []cardLayout
		height float64
	}
	columns := make([]colLayout, nCols)
	textWidth := columnWidth - 2*cardPadding
	// Derived font sizes clamped so very small Options.FontSize doesn't
	// produce zero-pixel labels.
	titleSize := max(fontSize+1, 1)
	metaSize := max(fontSize-2, 1)

	for i, s := range d.Sections {
		col := colLayout{}
		y := columnHeaderH + cardGap
		for _, task := range s.Tasks {
			c := cardLayout{
				lines: wrapText(task.Text, textWidth, fontSize),
				meta:  formatMetadata(task.Metadata),
			}
			c.height = 2*cardPadding + float64(len(c.lines))*lineHeight
			if len(c.meta) > 0 {
				c.height += float64(len(c.meta)) * metaLineHeight
			}
			col.cards = append(col.cards, c)
			y += c.height + cardGap
		}
		col.height = y
		columns[i] = col
	}

	canvasH := 0.0
	for _, c := range columns {
		if c.height > canvasH {
			canvasH = c.height
		}
	}
	if canvasH == 0 {
		canvasH = columnHeaderH + cardGap
	}
	viewH := 2*marginY + canvasH

	// Exact preallocation: 1 bg + per section (1 column rect + 1
	// title text + per card 1 rect + len(lines) body text + len(meta)
	// meta text).
	total := 1
	for i := range columns {
		total += 2
		for _, c := range columns[i].cards {
			total += 1 + len(c.lines) + len(c.meta)
		}
	}
	children := make([]any, 0, total)
	children = append(children, &rect{
		X: 0, Y: 0,
		Width:  svgFloat(viewW),
		Height: svgFloat(viewH),
		Style:  fmt.Sprintf("fill:%s;stroke:none", th.Background),
	})

	for i, s := range d.Sections {
		colX := marginX + float64(i)*(columnWidth+columnGap)
		col := columns[i]

		children = append(children, &rect{
			X: svgFloat(colX), Y: svgFloat(marginY),
			Width:  svgFloat(columnWidth),
			Height: svgFloat(canvasH),
			RX:     svgFloat(cardRadius), RY: svgFloat(cardRadius),
			Style: fmt.Sprintf("fill:%s;stroke:%s;stroke-width:1", th.ColumnFill, th.ColumnStroke),
		})
		children = append(children, &text{
			X:        svgFloat(colX + columnWidth/2),
			Y:        svgFloat(marginY + columnHeaderH/2),
			Anchor:   "middle",
			Dominant: "central",
			Style: fmt.Sprintf("fill:%s;font-size:%.0fpx;font-weight:bold",
				th.ColumnTitle, titleSize),
			Content: s.Title,
		})

		cardY := marginY + columnHeaderH + cardGap
		for _, c := range col.cards {
			children = append(children, &rect{
				X: svgFloat(colX + cardGap/2),
				Y: svgFloat(cardY),
				Width:  svgFloat(columnWidth - cardGap),
				Height: svgFloat(c.height),
				RX:     svgFloat(cardRadius), RY: svgFloat(cardRadius),
				Style: fmt.Sprintf("fill:%s;stroke:%s;stroke-width:1", th.CardFill, th.CardStroke),
			})
			textY := cardY + cardPadding + lineHeight/2
			for _, ln := range c.lines {
				children = append(children, &text{
					X:        svgFloat(colX + cardGap/2 + cardPadding),
					Y:        svgFloat(textY),
					Dominant: "central",
					Style:    fmt.Sprintf("fill:%s;font-size:%.0fpx", th.CardText, fontSize),
					Content:  ln,
				})
				textY += lineHeight
			}
			metaY := textY + metaLineHeight/2 - lineHeight/2
			for _, m := range c.meta {
				children = append(children, &text{
					X:        svgFloat(colX + cardGap/2 + cardPadding),
					Y:        svgFloat(metaY),
					Dominant: "central",
					Style:    fmt.Sprintf("fill:%s;font-size:%.0fpx", th.MetaText, metaSize),
					Content:  m,
				})
				metaY += metaLineHeight
			}
			cardY += c.height + cardGap
		}
	}

	doc := svgDoc{
		XMLNS:    "http://www.w3.org/2000/svg",
		ViewBox:  fmt.Sprintf("0 0 %.2f %.2f", viewW, viewH),
		Children: children,
	}
	b, err := xml.Marshal(doc)
	if err != nil {
		return nil, fmt.Errorf("kanban render: %w", err)
	}
	return append([]byte("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n"), b...), nil
}

// wrapText naively splits text at word boundaries to fit within
// `width` pixels at the given font size. Long unbroken words are
// emitted on their own line (allowed to overflow) rather than broken
// mid-word. Returns at least one line even for empty input so the
// card has a non-zero height.
func wrapText(s string, width, fontSize float64) []string {
	if s == "" {
		return []string{""}
	}
	// Translate width budget into a rough character cap using the
	// shared per-rune heuristic. Close enough for wrapping word-level
	// tokens; exact layout still happens at render time.
	oneCharW := textmeasure.EstimateWidth("x", fontSize)
	if oneCharW <= 0 {
		return []string{s}
	}
	maxChars := int(width / oneCharW)
	if maxChars <= 0 {
		return []string{s}
	}
	words := strings.Fields(s)
	if len(words) == 0 {
		return []string{s}
	}
	var lines []string
	var cur strings.Builder
	for _, w := range words {
		if cur.Len() == 0 {
			cur.WriteString(w)
			continue
		}
		if cur.Len()+1+len(w) > maxChars {
			lines = append(lines, cur.String())
			cur.Reset()
			cur.WriteString(w)
			continue
		}
		cur.WriteByte(' ')
		cur.WriteString(w)
	}
	if cur.Len() > 0 {
		lines = append(lines, cur.String())
	}
	return lines
}

// formatMetadata returns deterministic `key: value` lines. Keys are
// sorted with a fixed preference order (priority, assigned, ticket
// first, then everything else alphabetically) so renderers don't
// depend on map iteration.
func formatMetadata(m map[string]string) []string {
	if len(m) == 0 {
		return nil
	}
	priority := []string{"priority", "assigned", "ticket"}
	seen := make(map[string]bool, len(m))
	var keys []string
	for _, k := range priority {
		if _, ok := m[k]; ok {
			keys = append(keys, k)
			seen[k] = true
		}
	}
	var rest []string
	for k := range m {
		if !seen[k] {
			rest = append(rest, k)
		}
	}
	sort.Strings(rest)
	keys = append(keys, rest...)
	out := make([]string, 0, len(keys))
	for _, k := range keys {
		out = append(out, fmt.Sprintf("%s: %s", k, m[k]))
	}
	return out
}

