package diagram

type GitCommitType int8

const (
	GitCommitNormal GitCommitType = iota
	GitCommitReverse
	GitCommitHighlight
	GitCommitMerge
	// GitCommitCherryPick records that this commit was produced by
	// `cherry-pick id: "..."` rather than a direct `commit`. The
	// renderer draws it with a distinct glyph; CherryPickOf points
	// at the source commit on the donor branch.
	GitCommitCherryPick
)

var gitCommitTypeNames = []string{"normal", "reverse", "highlight", "merge", "cherry_pick"}

func (t GitCommitType) String() string { return enumString(t, gitCommitTypeNames) }

type GitCommit struct {
	ID     string
	Branch string
	Type   GitCommitType
	Tag    string
	// Msg is the optional `commit msg: "..."` body — distinct from
	// the id label that Mermaid renders below the dot.
	Msg string
	// CherryPickOf is set for cherry-pick commits to the id of the
	// commit being copied. CherryPickParent (when present) overrides
	// the parent inference used when the source is a merge commit.
	CherryPickOf     string
	CherryPickParent string
	// Parents are the commit IDs this commit descends from. A normal
	// commit has exactly 1 parent (or 0 for the initial commit). A
	// merge commit has 2: the receiving branch's previous head plus
	// the merged-in branch's head.
	Parents []string
}

type GitGraphDiagram struct {
	// Branches lists branch names in declaration order. The first
	// branch is typically "main" (implicit — created automatically
	// when the first commit lands without a prior `branch` command).
	Branches []string
	// BranchOrder maps a branch name to the explicit `order: N`
	// supplied at declaration time. Renderers should sort lanes by
	// (order asc, declaration index asc) so a high-order branch
	// drops to the bottom of the diagram.
	BranchOrder map[string]int
	Commits     []GitCommit
}

func (*GitGraphDiagram) Type() DiagramType { return GitGraph }

var _ Diagram = (*GitGraphDiagram)(nil)
