# Explore and inspect architecture canonical domain model

## ViewSession aggregate

Fields: `session_id`, `model_id`, `project_root_label`, `model_revision`, `state`, `status` (`open`, `stale`, `closed`).

### ViewState

`hierarchy_path[]`, `selected_node_id?`, `selected_relationship_id?`, `search_query?`, `filters`, `viewport` (`zoom`, `pan_x`, `pan_y`), `layout_overrides?`, `display_mode` (`overview`, `detail`, `list`), `reanalysis_revision`.

`filters` includes `reference_visibility` (`hidden`, `aggregated`, `expanded`; `hidden` is the default for the overview) and optional reference-scope filters for `standard_library`, `external`, `unresolved`, and `dynamic`. `layout_overrides` contains optional session-scoped positions keyed by visible ID and is never part of canonical model state.

### SceneSnapshot

`model_id`, `hierarchy_path[]`, `visible_nodes[]`, `visible_relationships[]`, `cycle_indicators[]`, `layer_labels[]`, `diagnostic_indicators[]`, `reference_summary`, `reference_details[]`, `renderer_hints`, `evidence_links[]`.

`VisibleNode`: `id`, `kind` (`module`, `group`, or `reference`), `label`, `module_ids[]`, `hierarchy_path[]`, `reference_scope?`, `tags[]`, `cycle_state`, `diagnostic_state`, `identity_state`, `confidence_state`, `layer?`, `internal_relationship_ids[]?`, `counts`, `accessible_label`.

`counts` includes `module_count`, visible incident `relationship_count`, optional
`internal_relationship_count`, `evidence_count`, and `diagnostic_count`.

`VisibleRelationship`: `id`, `type`, `from_visible_id`, `to_visible_id/reference_id`, `target_scope?`, `count`, `contributor_relationship_ids[]`, `cycle_state`, `confidence_state`, `evidence_ids[]`.

`ReferenceDetail`: `id`, `name`, `scope`, `from_visible_ids[]`, `relationship_ids[]`, `evidence_ids[]`, `count`, `confidence_state`, `confidence_basis?`, `confidence_score?`, `accessible_label`.

## Policies and invariants

- Every visible canonical ID resolves to the current model revision.
- Aggregated nodes/relations preserve contributor IDs and evidence links.
- A non-cycle relationship between child modules that collapses into a group
  self-loop is represented as `internal_relationship_ids[]` and
  `internal_relationship_count` on the group rather than as a cycle-like
  visible edge. Real canonical cycles remain visible through cycle state and
  cycle indicators.
- `identity_state` describes stable projection/node identity. It is distinct
  from `confidence_state`; local module/group identity does not imply a
  relationship-confidence score.
- The overview may hide or aggregate non-local references, but reference summaries and list/details inspection preserve their scope, confidence, import identity, and evidence.
- Standard-library, external, unresolved, and dynamic references are distinct states; `reference` describes target ownership, not confidence.
- A session cannot expose source outside its configured project root.
- Source inspection is read-only; path traversal, symlink escape, and unreadable files return diagnostics.
- Reanalysis increments model revision and clears selections that no longer resolve.
- Scene snapshots may be rendered by SVG, Canvas, WebGL, or accessible list views without changing semantics. Layout algorithms and manual positions remain renderer/session concerns.

## Domain events

`ViewOpened`, `HierarchyEntered`, `SelectionChanged`, `EvidenceInspected`, `SourceInspected`, `ViewReanalyzed`, `ViewBecameStale`, `ViewClosed`.

## Extension points

Renderer adapters, themes, filters, layout algorithms, alternate accessibility views, and large-graph level-of-detail policies.
