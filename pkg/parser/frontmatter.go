package parser

import (
	"bufio"
	"bytes"
	"strings"
)

// SplitFrontmatter separates a leading YAML frontmatter block (delimited
// by `---` lines) from the rest of the input. The frontmatter block must
// start on the first non-blank, non-comment line.
//
// Returns (frontmatterBody, rest). frontmatterBody excludes the `---`
// delimiters and is empty if no frontmatter is present. rest is the
// portion of src after the closing `---`, or all of src if there is no
// frontmatter.
//
// Mermaid spec: https://mermaid.js.org/config/configuration.html#frontmatter-config
func SplitFrontmatter(src []byte) (frontmatter, rest []byte) {
	scanner := bufio.NewScanner(bytes.NewReader(src))
	scanner.Buffer(make([]byte, 0, 64*1024), 1<<20)

	// Find the opening `---`.
	var openOffset int
	openFound := false
	pos := 0
	for scanner.Scan() {
		raw := scanner.Bytes()
		lineLen := len(raw) + 1 // +1 for the newline
		trimmed := strings.TrimSpace(string(raw))
		if trimmed == "" || strings.HasPrefix(trimmed, "%%") {
			pos += lineLen
			continue
		}
		if trimmed == "---" {
			openOffset = pos + lineLen
			openFound = true
			break
		}
		// First content line is not `---` — no frontmatter.
		return nil, src
	}
	if !openFound {
		return nil, src
	}

	// Find the closing `---`.
	closeEnd := -1
	pos = openOffset
	for scanner.Scan() {
		raw := scanner.Bytes()
		lineLen := len(raw) + 1
		if strings.TrimSpace(string(raw)) == "---" {
			closeEnd = pos + lineLen
			break
		}
		pos += lineLen
	}
	if closeEnd < 0 {
		// Unterminated frontmatter — treat the whole remainder as body.
		return nil, src
	}
	return src[openOffset : pos], trimLeadingNewlines(src[closeEnd:])
}

func trimLeadingNewlines(b []byte) []byte {
	for len(b) > 0 && (b[0] == '\n' || b[0] == '\r') {
		b = b[1:]
	}
	return b
}

// FrontmatterValue extracts the trimmed value for `key:` from a YAML
// frontmatter body. Returns "" if the key is not present. Quoted values
// (single or double) are unquoted. Other YAML constructs (nested maps,
// lists, multi-line strings) are not interpreted.
func FrontmatterValue(frontmatter []byte, key string) string {
	scanner := bufio.NewScanner(bytes.NewReader(frontmatter))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		rest, ok := strings.CutPrefix(line, key)
		if !ok {
			continue
		}
		rest = strings.TrimLeft(rest, " \t")
		rest, ok = strings.CutPrefix(rest, ":")
		if !ok {
			continue
		}
		val := strings.TrimSpace(rest)
		if len(val) >= 2 {
			if (val[0] == '"' && val[len(val)-1] == '"') || (val[0] == '\'' && val[len(val)-1] == '\'') {
				val = val[1 : len(val)-1]
			}
		}
		return val
	}
	return ""
}
