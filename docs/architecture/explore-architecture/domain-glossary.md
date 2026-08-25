# Explore and inspect architecture domain glossary

| Term | Definition | Distinction |
|---|---|---|
| View session | One local user exploration context over one model. | Not the analysis run or canonical model. |
| View state | Current hierarchy path, selection, filters, viewport, and presentation options. | Ephemeral and replaceable. |
| Hierarchy path | Selected structural segments used for drill-down. | Not a dependency path/cycle. |
| Scene snapshot | Renderer-neutral visible nodes, groups, relations, labels, styles, and evidence links. | Not SVG/Canvas implementation. |
| Visible node | Module or aggregated hierarchy group currently shown. | May represent many canonical modules. |
| Visible relation | Aggregated typed relation between visible nodes with contributors/counts. | Derived from canonical relations. |
| Evidence panel | Details for selected module/relation/diagnostic. | Not an editable inspector. |
| Source inspection | Read-only display of repository source around an evidence location. | No execution or editing. |
| Progressive disclosure | Showing aggregate/summary first and details on demand. | Not data loss. |
| Stale evidence | Evidence belonging to a replaced model/source snapshot. | Must not be displayed as current. |
| Accessible alternative | Keyboard/list/details path to the same inspectable facts. | Complements, not replaces, graphics. |
