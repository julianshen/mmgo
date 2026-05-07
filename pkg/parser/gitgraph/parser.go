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
	src, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("reading input: %w", err)
	}
	p := &parser{
		diagram: &diagram.GitGraphDiagram{
			BranchOrder: make(map[string]int),
		},
		branchHead: make(map[string]string),
		branchSeen: make(map[string]bool),
		current:    defaultBranch,
	}
	// Optional `---\n…\n---` frontmatter at the top supplies the
	// title and (under `config.gitGraph.mainBranchName`) the
	// effective default branch. The flat FrontmatterValue helper
	// finds nested keys by name — sufficient for Phase 2.
	front, body := parserutil.SplitFrontmatter(src)
	if len(front) > 0 {
		if t := parserutil.FrontmatterValue(front, "title"); t != "" {
			p.diagram.Title = t
		}
		if name := parserutil.FrontmatterValue(front, "mainBranchName"); name != "" {
			p.diagram.MainBranchName = name
			p.current = name
		}
	}
	scanner := bufio.NewScanner(strings.NewReader(string(body)))
	scanner.Buffer(make([]byte, 0, 64*1024), 1<<20)
	lineNum := 0
	headerSeen := false
	var accDescrLines []string
	inAccDescrBlock := false

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
			if dir := headerDirection(line); dir != "" {
				p.diagram.Direction = dir
			}
			headerSeen = true
			continue
		}
		if inAccDescrBlock {
			if line == "}" {
				p.diagram.AccDescr = strings.Join(accDescrLines, "\n")
				accDescrLines = accDescrLines[:0]
				inAccDescrBlock = false
				continue
			}
			accDescrLines = append(accDescrLines, line)
			continue
		}
		if line == "accDescr {" || line == "accDescr{" {
			inAccDescrBlock = true
			continue
		}
		if v, ok := parserutil.MatchKeywordValue(line, "accTitle"); ok {
			p.diagram.AccTitle = v
			continue
		}
		if v, ok := parserutil.MatchKeywordValue(line, "accDescr"); ok {
			p.diagram.AccDescr = v
			continue
		}
		if err := p.parseLine(line); err != nil {
			return nil, fmt.Errorf("line %d: %w", lineNum, err)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("reading input: %w", err)
	}
	if inAccDescrBlock {
		return nil, fmt.Errorf("unterminated accDescr { ... } block")
	}
	if !headerSeen {
		return nil, fmt.Errorf("missing gitGraph header")
	}
	return p.diagram, nil
}

// headerResidue strips the `gitGraph` keyword (and optional trailing
// colon) and returns the residual direction token. ok=false means
// the line isn't a gitGraph header at all. Sharing the trim chain
// keeps isHeader and headerDirection in lockstep so a future
// header form (e.g. RL) only needs to change one acceptor.
func headerResidue(line string) (rest string, ok bool) {
	if !parserutil.HasHeaderKeyword(line, "gitGraph") {
		return "", false
	}
	rest = strings.TrimSpace(strings.TrimPrefix(line, "gitGraph"))
	rest = strings.TrimSuffix(rest, ":")
	return strings.TrimSpace(rest), true
}

func isHeader(line string) bool {
	rest, ok := headerResidue(line)
	if !ok {
		return false
	}
	return rest == "" || rest == "LR" || rest == "TB" || rest == "BT"
}

// headerDirection returns the direction token from `gitGraph LR`
// (etc.). Empty means the bare-header form, leaving the AST default.
func headerDirection(line string) diagram.GitGraphDirection {
	rest, ok := headerResidue(line)
	if !ok {
		return ""
	}
	switch rest {
	case "LR":
		return diagram.GitGraphDirLR
	case "TB":
		return diagram.GitGraphDirTB
	case "BT":
		return diagram.GitGraphDirBT
	}
	return ""
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
	case "checkout", "switch":
		// `switch` is documented as an alias for `checkout` in
		// newer Mermaid versions.
		return p.parseCheckout(rest)
	case "merge":
		return p.parseMerge(rest)
	case "cherry-pick":
		return p.parseCherryPick(rest)
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

// parseCommitAttrs parses `id: "x"`, `tag: "v1"`, `type: HIGHLIGHT`,
// `msg: "..."`, plus the cherry-pick variants `parent: "..."`.
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
		case "msg":
			c.Msg = val
		case "parent":
			c.CherryPickParent = val
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

func (p *parser) parseBranch(rest string) error {
	name, after := splitQuotedOrField(rest)
	if name == "" {
		return fmt.Errorf("branch name required")
	}
	if order, ok := extractOrder(after); ok {
		p.diagram.BranchOrder[name] = order
	}
	p.ensureBranch(name)
	if head, ok := p.branchHead[p.current]; ok {
		p.branchHead[name] = head
	}
	p.current = name
	return nil
}

// extractOrder peels an optional `order: N` clause from the tail of
// a `branch <name> order: N` declaration. Returns (n, true) on a
// valid integer, (0, false) when missing or malformed.
func extractOrder(s string) (int, bool) {
	idx := strings.Index(s, "order:")
	if idx < 0 {
		return 0, false
	}
	val := strings.TrimSpace(s[idx+len("order:"):])
	val = strings.TrimSuffix(val, ",")
	n := 0
	for _, r := range val {
		if r < '0' || r > '9' {
			break
		}
		n = n*10 + int(r-'0')
	}
	if val == "" {
		return 0, false
	}
	return n, true
}

// splitQuotedOrField returns the first token (a quoted string if
// present, otherwise a whitespace-delimited field) and the trimmed
// remainder. Lets `branch "feat-X"` and `checkout "release/1.0"`
// carry names containing spaces, slashes, or hyphens.
func splitQuotedOrField(s string) (token, rest string) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", ""
	}
	if s[0] == '"' {
		if end := strings.IndexByte(s[1:], '"'); end >= 0 {
			return s[1 : end+1], strings.TrimSpace(s[end+2:])
		}
		return s[1:], ""
	}
	if i := strings.IndexAny(s, " \t"); i >= 0 {
		return s[:i], strings.TrimSpace(s[i+1:])
	}
	return s, ""
}

func (p *parser) parseCheckout(rest string) error {
	name, _ := splitQuotedOrField(rest)
	if name == "" {
		return fmt.Errorf("checkout: branch name required")
	}
	p.ensureBranch(name)
	p.current = name
	return nil
}

// parseCherryPick handles `cherry-pick id: "..." [tag: "..."]
// [parent: "..."]`. The cherry-pick lands on the current branch as
// a new commit whose CherryPickOf points back at the source.
func (p *parser) parseCherryPick(rest string) error {
	commit := diagram.GitCommit{
		Branch: p.current,
		Type:   diagram.GitCommitCherryPick,
	}
	parseCommitAttrs(rest, &commit)
	if commit.ID == "" {
		return fmt.Errorf("cherry-pick: id is required")
	}
	commit.CherryPickOf = commit.ID
	p.commitSeq++
	commit.ID = fmt.Sprintf("cp%d", p.commitSeq)
	p.ensureBranch(p.current)
	if head, ok := p.branchHead[p.current]; ok {
		commit.Parents = []string{head}
	}
	p.branchHead[p.current] = commit.ID
	p.diagram.Commits = append(p.diagram.Commits, commit)
	return nil
}

func (p *parser) parseMerge(rest string) error {
	mergedBranch, after := splitQuotedOrField(rest)
	if mergedBranch == "" {
		return fmt.Errorf("merge: branch name required")
	}

	p.ensureBranch(p.current)
	p.ensureBranch(mergedBranch)

	commit := diagram.GitCommit{
		Branch: p.current,
		Type:   diagram.GitCommitMerge,
	}
	if after != "" {
		parseCommitAttrs(after, &commit)
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
