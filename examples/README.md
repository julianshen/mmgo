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

| Diagram type   | Examples                                  |
|----------------|-------------------------------------------|
| flowchart      | simple, shapes, subgraphs                 |
| sequence       | simple, auth_flow, notes                  |
| pie            | simple, browsers                          |
| class          | simple, mvc                               |
| state          | simple, traffic_light                     |
| er             | simple, blog                              |
| gantt          | simple, release                           |
| mindmap        | simple, project                           |
| timeline       | simple, career                            |
| c4             | context, container                        |
| block          | simple, pipeline                          |
| gitgraph       | simple, feature_branch                    |
| sankey         | simple, budget                            |
| xychart        | sales, compare                            |
| quadrant       | campaigns, priority                       |
| kanban         | simple, team_board                        |
