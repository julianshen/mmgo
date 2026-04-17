package diagram

type GitCommitType int8

const (
	GitCommitNormal GitCommitType = iota
	GitCommitReverse
	GitCommitHighlight
	GitCommitMerge
)

var gitCommitTypeNames = []string{"normal", "reverse", "highlight", "merge"}

func (t GitCommitType) String() string { return enumString(t, gitCommitTypeNames) }

type GitCommit struct {
	ID     string
	Branch string
	Type   GitCommitType
	Tag    string
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
	Commits  []GitCommit
}

func (*GitGraphDiagram) Type() DiagramType { return GitGraph }

var _ Diagram = (*GitGraphDiagram)(nil)
