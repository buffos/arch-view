# Explore and inspect architecture canonical domain model

## ViewSession aggregate

Fields: `session_id`, `model_id`, `project_root_label`, `model_revision`, `state`, `status` (`open`, `stale`, `closed`).

### ViewState

`hierarchy_path[]`, `selected_node_id?`, `selected_relationship_id?`, `search_query?`, `filters`, `viewport` (`zoom`, `pan_x`, `pan_y`), `display_mode` (`overview`, `detail`, `list`), `reanalysis_revision`.

### SceneSnapshot

`model_id`, `hierarchy_path[]`, `visible_nodes[]`, `visible_relationships[]`, `cycle_indicators[]`, `layer_labels[]`, `diagnostic_indicators[]`, `renderer_hints`, `evidence_links[]`.

`VisibleNode`: `id`, `kind` (`module` or `group`), `label`, `module_ids[]`, `hierarchy_path[]`, `tags[]`, `cycle_state`, `diagnostic_state`, `layer?`, `accessible_label`.

`VisibleRelationship`: `id`, `type`, `from_visible_id`, `to_visible_id/reference_id`, `count`, `contributor_relationship_ids[]`, `cycle_state`, `evidence_ids[]`.

## Policies and invariants

- Every visible canonical ID resolves to the current model revision.
- Aggregated nodes/relations preserve contributor IDs and evidence links.
- A session cannot expose source outside its configured project root.
- Source inspection is read-only; path traversal, symlink escape, and unreadable files return diagnostics.
- Reanalysis increments model revision and clears selections that no longer resolve.
- Scene snapshots may be rendered by SVG, Canvas, WebGL, or accessible list views without changing semantics.

## Domain events

`ViewOpened`, `HierarchyEntered`, `SelectionChanged`, `EvidenceInspected`, `SourceInspected`, `ViewReanalyzed`, `ViewBecameStale`, `ViewClosed`.

## Extension points

Renderer adapters, themes, filters, layout algorithms, alternate accessibility views, and large-graph level-of-detail policies.
