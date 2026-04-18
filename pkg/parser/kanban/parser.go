// Package kanban parses Mermaid kanban diagram syntax. Indentation
// distinguishes sections (column-0) from tasks (indented).
//
//	kanban
//	    Todo
//	        [Create tickets]
//	        id[Triage]@{ priority: 'High' }
//	    id4[In progress]
//	        [Design]
//	    Done
//	        [Write tests]@{ assigned: 'alice' }
//
// Elements may optionally carry `id[text]` (ID prefix) and a trailing
// `@{ key: value, key2: 'value2' }` metadata block. Metadata values
// may be single-quoted to include commas and spaces.
package kanban

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	"github.com/julianshen/mmgo/pkg/diagram"
	parserutil "github.com/julianshen/mmgo/pkg/parser"
)

const headerKeyword = "kanban"

func Parse(r io.Reader) (*diagram.KanbanDiagram, error) {
	d := &diagram.KanbanDiagram{}
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 64*1024), 1<<20)

	lineNum := 0
	headerSeen := false
	// sectionIndent is the indent level of the first body line; any
	// line at that level is a section, any deeper indent is a task.
	// -1 means we haven't seen the first body line yet.
	sectionIndent := -1
	currentSection := -1
	taskSeq := 0

	for scanner.Scan() {
		lineNum++
		raw := parserutil.StripComment(scanner.Text())
		trimmed := strings.TrimSpace(raw)
		if trimmed == "" {
			continue
		}
		if !headerSeen {
			if !parserutil.HasHeaderKeyword(trimmed, headerKeyword) {
				return nil, fmt.Errorf("line %d: expected '%s' header, got %q", lineNum, headerKeyword, trimmed)
			}
			headerSeen = true
			continue
		}
		indent := leadingSpaces(raw)
		if sectionIndent == -1 {
			sectionIndent = indent
		}
		// A line shallower than the first body line is almost always
		// an accidental dedent; reject rather than silently reshape
		// the AST. Equal indent is a new section; deeper is a task.
		if indent < sectionIndent {
			return nil, fmt.Errorf("line %d: indent %d is shallower than the section indent %d", lineNum, indent, sectionIndent)
		}
		if indent == sectionIndent {
			id, text, meta, err := parseElement(trimmed)
			if err != nil {
				return nil, fmt.Errorf("line %d: %w", lineNum, err)
			}
			d.Sections = append(d.Sections, diagram.KanbanSection{
				ID:       id,
				Title:    text,
				Metadata: meta,
			})
			currentSection = len(d.Sections) - 1
			continue
		}
		id, text, meta, err := parseElement(trimmed)
		if err != nil {
			return nil, fmt.Errorf("line %d: %w", lineNum, err)
		}
		if id == "" {
			taskSeq++
			id = fmt.Sprintf("t%d", taskSeq)
		}
		d.Sections[currentSection].Tasks = append(d.Sections[currentSection].Tasks, diagram.KanbanTask{
			ID:       id,
			Text:     text,
			Metadata: meta,
		})
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("reading input: %w", err)
	}
	if !headerSeen {
		return nil, fmt.Errorf("missing %s header", headerKeyword)
	}
	return d, nil
}

func leadingSpaces(s string) int {
	n := 0
	for n < len(s) {
		c := s[n]
		if c != ' ' && c != '\t' {
			break
		}
		n++
	}
	return n
}

// parseElement splits a kanban element into (id, text, metadata). The
// source forms accepted:
//   - `text`                          — bare text, no brackets, no id
//   - `[text]`                        — bracketed text
//   - `id[text]`                      — id prefix + bracketed text
//   - `[text]@{ key: value, ... }`    — bracketed text + metadata
//   - `id[text]@{ key: value, ... }`  — all three
//
// The bracketed form is preferred. A bare-text element has no id and
// no metadata.
func parseElement(s string) (id, text string, metadata map[string]string, err error) {
	// Pull trailing `@{ ... }` first so the rest is simpler. Use
	// LastIndex so a `@{` appearing literally earlier in the task
	// text doesn't get mistaken for the metadata start. The closing
	// `}` is located with quote-aware scanning so `'}' inside a
	// quoted value doesn't truncate the body early.
	if at := strings.LastIndex(s, "@{"); at >= 0 {
		end := findMetaClose(s[at+2:])
		if end < 0 {
			return "", "", nil, fmt.Errorf("unterminated '@{' metadata")
		}
		metaBody := s[at+2 : at+2+end]
		if tail := strings.TrimSpace(s[at+2+end+1:]); tail != "" {
			return "", "", nil, fmt.Errorf("unexpected trailing text after metadata: %q", tail)
		}
		m, err := parseMetadata(metaBody)
		if err != nil {
			return "", "", nil, err
		}
		metadata = m
		s = strings.TrimSpace(s[:at])
	}

	// The id prefix ends at the first `[` (if any).
	if lb := strings.Index(s, "["); lb >= 0 {
		rb := strings.LastIndex(s, "]")
		if rb < lb {
			return "", "", nil, fmt.Errorf("missing closing ']'")
		}
		id = strings.TrimSpace(s[:lb])
		text = strings.TrimSpace(s[lb+1 : rb])
		if tail := strings.TrimSpace(s[rb+1:]); tail != "" {
			return "", "", nil, fmt.Errorf("unexpected trailing text after ']': %q", tail)
		}
		if text == "" {
			return "", "", nil, fmt.Errorf("bracketed text is empty")
		}
		return id, text, metadata, nil
	}
	// Bare form: treat the whole thing as text.
	return "", s, metadata, nil
}

// parseMetadata handles `key: value, key2: 'value, with comma'`. Values
// may be wrapped in single or double quotes to preserve commas. An
// empty body (`@{}` or `@{  }`) returns nil.
func parseMetadata(s string) (map[string]string, error) {
	if strings.TrimSpace(s) == "" {
		return nil, nil
	}
	out := make(map[string]string)
	for _, tok := range splitTopLevelCommas(s) {
		tok = strings.TrimSpace(tok)
		if tok == "" {
			continue
		}
		colon := strings.Index(tok, ":")
		if colon < 0 {
			return nil, fmt.Errorf("metadata entry %q missing ':'", tok)
		}
		k := strings.TrimSpace(tok[:colon])
		v := strings.TrimSpace(tok[colon+1:])
		if k == "" {
			return nil, fmt.Errorf("metadata key is empty in %q", tok)
		}
		out[k] = parserutil.Unquote(v)
	}
	if len(out) == 0 {
		return nil, nil
	}
	return out, nil
}

// findMetaClose returns the index of the first `}` outside single or
// double quotes. -1 if quotes are unterminated or no `}` is found —
// both are treated as malformed metadata by the caller.
func findMetaClose(s string) int {
	var quote byte
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case quote != 0:
			if c == quote {
				quote = 0
			}
		case c == '\'' || c == '"':
			quote = c
		case c == '}':
			return i
		}
	}
	return -1
}

// splitTopLevelCommas splits on commas that are outside single or
// double quotes.
func splitTopLevelCommas(s string) []string {
	var out []string
	var cur strings.Builder
	var quote byte
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case quote != 0:
			if c == quote {
				quote = 0
			}
			cur.WriteByte(c)
		case c == '\'' || c == '"':
			quote = c
			cur.WriteByte(c)
		case c == ',':
			out = append(out, cur.String())
			cur.Reset()
		default:
			cur.WriteByte(c)
		}
	}
	if cur.Len() > 0 {
		out = append(out, cur.String())
	}
	return out
}

