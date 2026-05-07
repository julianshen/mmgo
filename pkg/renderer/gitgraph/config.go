package gitgraph

import "github.com/julianshen/mmgo/pkg/renderer/svgutil"

// Config mirrors the `gitGraph` knobs from
// https://mermaid.js.org/syntax/gitgraph.html#configuration. Show*
// flags use *bool because the spec defaults are non-zero (true) and
// callers need to distinguish "explicitly off" from "inherit".
type Config struct {
	ShowBranches      *bool
	ShowCommitLabel   *bool
	RotateCommitLabel *bool
	ParallelCommits   *bool
	// MainBranchOrder is the lane index for the main branch when
	// renderers sort lanes. Defaults to 0 (top of the diagram); set
	// higher to push the main branch below feature branches.
	MainBranchOrder int
}

// DefaultConfig returns the Mermaid spec defaults.
func DefaultConfig() Config {
	return Config{
		ShowBranches:      svgutil.BoolPtr(true),
		ShowCommitLabel:   svgutil.BoolPtr(true),
		RotateCommitLabel: svgutil.BoolPtr(true),
		ParallelCommits:   svgutil.BoolPtr(false),
		MainBranchOrder:   0,
	}
}

func resolveConfig(opts *Options) Config {
	c := DefaultConfig()
	if opts == nil {
		return c
	}
	o := opts.Config
	svgutil.MergeBoolPtr(&c.ShowBranches, o.ShowBranches)
	svgutil.MergeBoolPtr(&c.ShowCommitLabel, o.ShowCommitLabel)
	svgutil.MergeBoolPtr(&c.RotateCommitLabel, o.RotateCommitLabel)
	svgutil.MergeBoolPtr(&c.ParallelCommits, o.ParallelCommits)
	if o.MainBranchOrder != 0 {
		c.MainBranchOrder = o.MainBranchOrder
	}
	return c
}

