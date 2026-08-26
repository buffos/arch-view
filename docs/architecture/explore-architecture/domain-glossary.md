# Explore and inspect architecture domain glossary

| Term | Definition | Distinction |
|---|---|---|
| View session | One local user exploration context over one model. | Not the analysis run or canonical model. |
| View state | Current hierarchy path, selection, filters, viewport, and presentation options. | Ephemeral and replaceable. |
| Layout option catalog | The typed, grouped descriptions exposed by the pinned layout adapter, including algorithms, values, defaults, and applicability. | Not every catalog entry is necessarily valid for every selected algorithm or renderer. |
| Layout profile | The effective presentation settings used to calculate node positions and edge routes for a view. | Does not change canonical relationships or source analysis. |
| Project layout configuration | Versioned `.archview.json` preferences for presentation layout. | Separate from analyzer configuration, viewport state, and manual node positions. |
| Configuration discovery | Searching the selected target directory and its ancestors for the nearest `.archview.json`. | v1 selects one complete file; it does not merge multiple files. |
| Configuration origin | The source of effective layout settings: built-in defaults, a discovered project/ancestor file, an explicitly selected custom `Save As` file, or an unsaved session draft. | Explains why a value is active and where it can be changed. |
| Active configuration file | The exact `.archview.json` currently associated with the session and targeted by ordinary `Save`. | It may be in an ancestor folder; `Save` does not create a new file when no active file exists. |
| Save As | An explicit operation that writes the fixed `.archview.json` filename to a user-selected custom folder and makes that file active for the current session. | The only operation that accepts a custom destination. |
| Hierarchy path | Selected structural segments used for drill-down. | Not a dependency path/cycle. |
| Scene snapshot | Renderer-neutral visible nodes, groups, relations, labels, styles, and evidence links. | Not SVG/Canvas implementation. |
| Visible node | Module, aggregated hierarchy group, or explicitly expanded reference currently shown. | May represent many canonical modules; reference ownership is distinct from confidence. |
| Visible relation | Aggregated typed relation between visible nodes with contributors/counts. | Derived from canonical relations. |
| Reference scope | Ownership classification for a non-local target: standard library, external, unresolved, or dynamic. | Not the same as relationship confidence. |
| Reference visibility | Session policy for non-local targets: hidden, aggregated, or expanded. | Changes the scene projection, never the canonical model. |
| Import inspection | List-oriented view of individual imports, target scope, confidence, counts, and evidence for a selected module/group. | Detail/evidence workflow, not the default architecture graph. |
| Layout override | Optional session-scoped position or viewport adjustment keyed by model revision and hierarchy path. | Never mutates canonical model semantics. |
| Evidence panel | Details for selected module/relation/diagnostic. | Not an editable inspector. |
| Source inspection | Read-only display of repository source around an evidence location. | No execution or editing. |
| Progressive disclosure | Showing aggregate/summary first and details on demand. | Not data loss. |
| Stale evidence | Evidence belonging to a replaced model/source snapshot. | Must not be displayed as current. |
| Accessible alternative | Keyboard/list/details path to the same inspectable facts. | Complements, not replaces, graphics. |
