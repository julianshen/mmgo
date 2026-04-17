// Package gitgraph parses Mermaid gitGraph diagram syntax.
package gitgraph

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	"github.com/julianshen/mmgo/pkg/diagram"
	parserutil "github.com/julianshen/mmgo/pkg/parser"
)

const defaultBranch = "main"

func Parse(r io.Reader) (*diagram.GitGraphDiagram, error) {
	p := &parser{
		diagram:    &diagram.GitGraphDiagram{},
		branchHead: make(map[string]string),
		current:    defaultBranch,
	}
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 1<<20)
	lineNum := 0
	headerSeen := false

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(parserutil.StripComment(scanner.Text()))
		if line == "" {
			continue
		}
		if !headerSeen {
			if !isHeader(line) {
				return nil, fmt.Errorf("line %d: expected 'gitGraph' header, got %q", lineNum, line)
			}
			headerSeen = true
			continue
		}
		if err := p.parseLine(line); err != nil {
			return nil, fmt.Errorf("line %d: %w", lineNum, err)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("reading input: %w", err)
	}
	if !headerSeen {
		return nil, fmt.Errorf("missing gitGraph header")
	}
	return p.diagram, nil
}

// isHeader matches `gitGraph`, `gitGraph:`, `gitGraph LR`,
// `gitGraph TB:`, etc. Mermaid allows several forms.
func isHeader(line string) bool {
	if !strings.HasPrefix(line, "gitGraph") {
		return false
	}
	rest := strings.TrimSpace(strings.TrimPrefix(line, "gitGraph"))
	rest = strings.TrimSuffix(rest, ":")
	rest = strings.TrimSpace(rest)
	// Allow bare `gitGraph`, or trailing direction like "LR"/"TB".
	return rest == "" || rest == "LR" || rest == "TB" || rest == "BT"
}

type parser struct {
	diagram    *diagram.GitGraphDiagram
	branchHead map[string]string // branch -> current head commit ID
	current    string            // currently checked-out branch
	commitSeq  int               // auto-ID counter for unnamed commits
}

func (p *parser) parseLine(line string) error {
	kw, rest := splitKeyword(line)
	switch kw {
	case "commit":
		return p.parseCommit(rest)
	case "branch":
		return p.parseBranch(rest)
	case "checkout":
		return p.parseCheckout(rest)
	case "merge":
		return p.parseMerge(rest)
	}
	return nil
}

// splitKeyword returns the first word of line and the whitespace-trimmed
// remainder. Lines with no whitespace (e.g. `branch` alone) still yield
// the keyword and an empty rest, so argument-less forms can be diagnosed
// as errors downstream.
func splitKeyword(line string) (kw, rest string) {
	if i := strings.IndexAny(line, " \t"); i >= 0 {
		return line[:i], strings.TrimSpace(line[i+1:])
	}
	return line, ""
}

func (p *parser) ensureBranch(name string) {
	for _, b := range p.diagram.Branches {
		if b == name {
			return
		}
	}
	p.diagram.Branches = append(p.diagram.Branches, name)
}

func (p *parser) parseCommit(rest string) error {
	p.ensureBranch(p.current)
	commit := diagram.GitCommit{
		Branch: p.current,
		Type:   diagram.GitCommitNormal,
	}
	parseCommitAttrs(rest, &commit)
	if commit.ID == "" {
		p.commitSeq++
		commit.ID = fmt.Sprintf("c%d", p.commitSeq)
	}
	if head, ok := p.branchHead[p.current]; ok {
		commit.Parents = []string{head}
	}
	p.branchHead[p.current] = commit.ID
	p.diagram.Commits = append(p.diagram.Commits, commit)
	return nil
}

// parseCommitAttrs parses `id: "x"`, `tag: "v1"`, `type: HIGHLIGHT` etc.
// Attributes are space-separated and the key/value is colon-separated.
func parseCommitAttrs(s string, c *diagram.GitCommit) {
	// Split on spaces but respect quoted strings.
	for _, tok := range tokenizeAttrs(s) {
		colon := strings.Index(tok, ":")
		if colon < 0 {
			continue
		}
		key := strings.TrimSpace(tok[:colon])
		val := strings.TrimSpace(tok[colon+1:])
		val = strings.Trim(val, "\"")
		switch key {
		case "id":
			c.ID = val
		case "tag":
			c.Tag = val
		case "type":
			c.Type = parseCommitType(val)
		}
	}
}

func parseCommitType(s string) diagram.GitCommitType {
	switch strings.ToUpper(s) {
	case "REVERSE":
		return diagram.GitCommitReverse
	case "HIGHLIGHT":
		return diagram.GitCommitHighlight
	default:
		return diagram.GitCommitNormal
	}
}

// tokenizeAttrs splits `id: "my commit" tag: "v1"` into
// [`id: "my commit"`, `tag: "v1"`]. Quoted values are preserved.
func tokenizeAttrs(s string) []string {
	var tokens []string
	var cur strings.Builder
	inQuote := false
	// We split on whitespace that follows a completed `key:value` pair.
	// Simpler: track a "has seen colon" state; a space after the value
	// (outside quotes) ends the token.
	afterColon := false
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == '"' {
			inQuote = !inQuote
			cur.WriteByte(c)
			continue
		}
		if !inQuote && (c == ' ' || c == '\t') {
			if afterColon && cur.Len() > 0 {
				// Determine if we've consumed a value yet. Look for
				// trailing non-space after the colon.
				raw := cur.String()
				colon := strings.Index(raw, ":")
				if colon >= 0 && strings.TrimSpace(raw[colon+1:]) != "" {
					tokens = append(tokens, strings.TrimSpace(raw))
					cur.Reset()
					afterColon = false
					continue
				}
			}
			cur.WriteByte(c)
			continue
		}
		if c == ':' && !inQuote {
			afterColon = true
		}
		cur.WriteByte(c)
	}
	if cur.Len() > 0 {
		tokens = append(tokens, strings.TrimSpace(cur.String()))
	}
	return tokens
}

func (p *parser) parseBranch(name string) error {
	name = stripOptionalOrder(name)
	if name == "" {
		return fmt.Errorf("branch name required")
	}
	p.ensureBranch(name)
	// New branch starts at the current branch's head.
	if head, ok := p.branchHead[p.current]; ok {
		p.branchHead[name] = head
	}
	p.current = name
	return nil
}

// stripOptionalOrder drops a trailing "order: N" clause from a branch
// declaration. Mermaid allows `branch feature order: 2`.
func stripOptionalOrder(s string) string {
	if idx := strings.Index(s, " order:"); idx >= 0 {
		return strings.TrimSpace(s[:idx])
	}
	return strings.TrimSpace(s)
}

func (p *parser) parseCheckout(name string) error {
	if name == "" {
		return fmt.Errorf("checkout: branch name required")
	}
	p.ensureBranch(name)
	p.current = name
	return nil
}

func (p *parser) parseMerge(rest string) error {
	// Format: `merge <branch> [id: "x"] [tag: "v1"] [type: HIGHLIGHT]`
	parts := strings.Fields(rest)
	if len(parts) == 0 {
		return fmt.Errorf("merge: branch name required")
	}
	mergedBranch := parts[0]

	p.ensureBranch(p.current)
	p.ensureBranch(mergedBranch)

	commit := diagram.GitCommit{
		Branch: p.current,
		Type:   diagram.GitCommitMerge,
	}
	if len(parts) > 1 {
		parseCommitAttrs(strings.TrimSpace(rest[len(parts[0]):]), &commit)
	}
	if commit.ID == "" {
		p.commitSeq++
		commit.ID = fmt.Sprintf("m%d", p.commitSeq)
	}

	// Merge commit has two parents: current branch head, and merged
	// branch head (if one exists).
	if head, ok := p.branchHead[p.current]; ok {
		commit.Parents = append(commit.Parents, head)
	}
	if head, ok := p.branchHead[mergedBranch]; ok && mergedBranch != p.current {
		commit.Parents = append(commit.Parents, head)
	}
	p.branchHead[p.current] = commit.ID
	p.diagram.Commits = append(p.diagram.Commits, commit)
	return nil
}
