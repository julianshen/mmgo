# Examples

Reference Mermaid diagrams for every supported diagram type, rendered to both
SVG and PNG. Each `.mmd` source file has sibling `.svg` and `.png` outputs
produced by the `mmgo` CLI.

## Regenerating outputs

```sh
go build -o /tmp/mmgo ./cmd/mmgo
for f in examples/*/*.mmd; do
    base="${f%.mmd}"
    /tmp/mmgo -i "$f" -o "${base}.svg" -q
    /tmp/mmgo -i "$f" -o "${base}.png" -q
done
```

## CI guarantee

`pkg/output/svg/examples_test.go` re-renders every `.mmd` file on every CI
run and asserts the committed `.svg` matches byte-for-byte. If a code change
alters rendered output, the test fails and the committed snapshots must be
refreshed as part of the same commit.

## Coverage

| Diagram type | Examples                                                                                                                             |
|--------------|--------------------------------------------------------------------------------------------------------------------------------------|
| flowchart    | arrows, chaining, ci_cd, directions, edge_types, flowchart_practical, labels, nested_subgraphs, shapes, shapes_extended, shapes_mixed, simple, styling, subgraphs |
| sequence     | auth_flow, notes, simple                                                                                                             |
| pie          | browsers, simple                                                                                                                     |
| class        | mvc, simple                                                                                                                          |
| state        | simple, traffic_light                                                                                                                |
| er           | blog, simple                                                                                                                         |
| gantt        | release, simple                                                                                                                      |
| mindmap      | project, simple                                                                                                                      |
| timeline     | career, simple                                                                                                                       |
| c4           | container, context                                                                                                                   |
| block        | pipeline, simple                                                                                                                     |
| gitgraph     | feature_branch, simple                                                                                                               |
| sankey       | budget, simple                                                                                                                       |
| xychart      | compare, sales                                                                                                                       |
| quadrant     | campaigns, priority                                                                                                                  |
| kanban       | simple, team_board                                                                                                                   |
