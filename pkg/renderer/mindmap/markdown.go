package mindmap

import "strings"

type textSegment struct {
	text   string
	bold   bool
	italic bool
}

func parseMarkdown(s string) []textSegment {
	var segments []textSegment
	remaining := s

	for len(remaining) > 0 {
		biIdx := strings.Index(remaining, "***")
		if biIdx >= 0 {
			end := strings.Index(remaining[biIdx+3:], "***")
			if end >= 0 {
				if biIdx > 0 {
					segments = append(segments, textSegment{text: remaining[:biIdx]})
				}
				segments = append(segments, textSegment{
					text:   remaining[biIdx+3 : biIdx+3+end],
					bold:   true,
					italic: true,
				})
				remaining = remaining[biIdx+3+end+3:]
				continue
			}
		}

		boldIdx := indexOfBold(remaining)
		if boldIdx >= 0 {
			if boldIdx > 0 {
				segments = append(segments, textSegment{text: remaining[:boldIdx]})
			}
			after := remaining[boldIdx+2:]
			end := strings.Index(after, "**")
			if end >= 0 {
				segments = append(segments, textSegment{text: after[:end], bold: true})
				remaining = after[end+2:]
				continue
			}
			segments = append(segments, textSegment{text: remaining[boldIdx:]})
			remaining = ""
			continue
		}

		italicIdx := indexOfItalic(remaining)
		if italicIdx >= 0 {
			if italicIdx > 0 {
				segments = append(segments, textSegment{text: remaining[:italicIdx]})
			}
			after := remaining[italicIdx+1:]
			end := strings.Index(after, "*")
			if end >= 0 {
				segments = append(segments, textSegment{text: after[:end], italic: true})
				remaining = after[end+1:]
				continue
			}
			segments = append(segments, textSegment{text: remaining[italicIdx:]})
			remaining = ""
			continue
		}

		segments = append(segments, textSegment{text: remaining})
		remaining = ""
	}

	if len(segments) == 0 {
		segments = append(segments, textSegment{text: s})
	}
	return segments
}

func indexOfBold(s string) int {
	for i := 0; i+1 < len(s); i++ {
		if s[i] == '*' && s[i+1] == '*' {
			if i+2 < len(s) && s[i+2] == '*' {
				return -1
			}
			return i
		}
	}
	return -1
}

func indexOfItalic(s string) int {
	for i := 0; i < len(s); i++ {
		if s[i] == '*' {
			if i+1 < len(s) && s[i+1] == '*' {
				return -1
			}
			if i > 0 && s[i-1] == '*' {
				continue
			}
			return i
		}
	}
	return -1
}
