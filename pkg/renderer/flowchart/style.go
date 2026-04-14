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

func nodeStyleCSS(n diagram.Node, styles []diagram.StyleDef) string {
	for _, s := range styles {
		if s.NodeID == n.ID {
			return s.CSS
		}
	}
	return ""
}
