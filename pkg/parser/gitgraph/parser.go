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
		branchSeen: make(map[string]bool),
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

// isHeader accepts the several forms Mermaid allows: bare `gitGraph`,
// trailing colon, or an explicit direction token.
func isHeader(line string) bool {
	if !strings.HasPrefix(line, "gitGraph") {
		return false
	}
	rest := strings.TrimSpace(strings.TrimPrefix(line, "gitGraph"))
	rest = strings.TrimSuffix(rest, ":")
	rest = strings.TrimSpace(rest)
	return rest == "" || rest == "LR" || rest == "TB" || rest == "BT"
}

type parser struct {
	diagram    *diagram.GitGraphDiagram
	branchHead map[string]string // branch -> current head commit ID
	branchSeen map[string]bool   // O(1) dedupe for ensureBranch
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

// splitKeyword returns the first whitespace-delimited word and the
// trimmed remainder. Bare keywords (`branch` alone) yield an empty
// rest so downstream can error on missing arguments.
func splitKeyword(line string) (kw, rest string) {
	if i := strings.IndexAny(line, " \t"); i >= 0 {
		return line[:i], strings.TrimSpace(line[i+1:])
	}
	return line, ""
}

func (p *parser) ensureBranch(name string) {
	if p.branchSeen[name] {
		return
	}
	p.branchSeen[name] = true
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
func parseCommitAttrs(s string, c *diagram.GitCommit) {
	for _, tok := range tokenizeAttrs(s) {
		colon := strings.Index(tok, ":")
		if colon < 0 {
			continue
		}
		key := strings.TrimSpace(tok[:colon])
		val := strings.Trim(strings.TrimSpace(tok[colon+1:]), "\"")
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

// tokenizeAttrs splits `id: "my commit" tag: "v1"` on whitespace that
// separates two complete `key: value` pairs. Whitespace inside the
// value of a pending pair (before a value byte is seen) is folded into
// the current token so `id : "x"` still parses. Quotes are preserved
// so `parseCommitAttrs` can strip them later.
func tokenizeAttrs(s string) []string {
	var tokens []string
	var cur strings.Builder
	inQuote := false
	sawColon := false
	sawValue := false
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c == '"':
			inQuote = !inQuote
			cur.WriteByte(c)
			if sawColon {
				sawValue = true
			}
		case inQuote:
			cur.WriteByte(c)
		case c == ' ' || c == '\t':
			if sawColon && sawValue {
				tokens = append(tokens, strings.TrimSpace(cur.String()))
				cur.Reset()
				sawColon, sawValue = false, false
				continue
			}
			cur.WriteByte(c)
		case c == ':':
			sawColon = true
			cur.WriteByte(c)
		default:
			if sawColon {
				sawValue = true
			}
			cur.WriteByte(c)
		}
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
	if head, ok := p.branchHead[p.current]; ok {
		p.branchHead[name] = head
	}
	p.current = name
	return nil
}

// stripOptionalOrder drops a trailing `order: N` clause, which Mermaid
// uses only to influence lane ordering — mmgo preserves declaration
// order, so the clause is informational.
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
		parseCommitAttrs(strings.TrimPrefix(rest, parts[0]), &commit)
	}
	if commit.ID == "" {
		p.commitSeq++
		commit.ID = fmt.Sprintf("m%d", p.commitSeq)
	}

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
