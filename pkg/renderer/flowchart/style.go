package flowchart

import (
	"fmt"
	"strings"

	"github.com/julianshen/mmgo/pkg/diagram"
)

func buildClassCSS(d *diagram.FlowchartDiagram) string {
	if len(d.Classes) == 0 {
		return ""
	}
	var sb strings.Builder
	for name, css := range d.Classes {
		sb.WriteString(fmt.Sprintf(".%s { %s }\n", name, css))
	}
	return sb.String()
}

// nodeStyleCSS concatenates every StyleDef whose NodeID matches n.ID,
// in declaration order, separated by `;`. Mermaid allows multiple
// `style` directives on the same node and the renderer must preserve
// all of them (later directives override earlier ones via CSS
// later-wins semantics, not by silently dropping them).
func nodeStyleCSS(n diagram.Node, styles []diagram.StyleDef) string {
	var matched []string
	for _, s := range styles {
		if s.NodeID == n.ID && s.CSS != "" {
			matched = append(matched, s.CSS)
		}
	}
	return strings.Join(matched, ";")
}
