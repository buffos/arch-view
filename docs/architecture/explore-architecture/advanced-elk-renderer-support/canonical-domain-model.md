# Advanced ELK renderer support canonical domain model

The approved [delivery contract](delivery-contract.md) updates this baseline
for shared architecture/OKF delivery, persistence, stage ordering, and browser-only
ELK execution. Its AER-R rules and scenario mapping are authoritative.

## Modeling boundary

This model owns layout feature negotiation and renderer-neutral geometry. The
canonical model and semantic scene remain authoritative for architecture facts;
the project configuration capability owns profile discovery/persistence.

## LayoutFeatureProfile

Fields:

- `schema_version`: `arch-view.config/v2`
- `algorithm`
- `options`
- `features[]`

Allowed features are `edge_labels`, `junctions`, `ports`, `compound`, and
`spline_refinement`. Feature order is canonicalized lexically. An empty list is
backward-compatible with current behavior.

## GeometrySnapshot

Fields:

- `schema_version`: `arch-view.geometry/v1`
- `source`: kind (`architecture|okf`), source ID, source revision, navigation scope.
  Architecture adapters may retain model ID/revision; OKF uses bundle/projection
  identity and never fabricates architecture-model identity.
- `nodes[]`
- `edges[]`
- `diagnostics[]`
- `provenance`

### GeometryNode

Fields: `id`, `bounds`, `parent_id?`, `children_ids[]`, `ports[]`,
`semantic_node_id?`.

Visible semantic nodes retain their scene ID. Layout-only compound containers
use a deterministic `container::<escaped hierarchy path>` ID and no semantic
node ID.

### PresentationPort

Fields: `id`, `node_id`, `role` (`in|out`), `side`, `position`, `bounds?`,
`label?`.

Ports are generated presentation objects. Their IDs are deterministic from the
visible node ID and role; they are never relationship endpoints in the
canonical model.

### GeometryEdge

Fields: `id`, `semantic_relationship_id`, `source_node_id`, `target_node_id`,
`source_port_id?`, `target_port_id?`, `labels[]`, `junctions[]`, `route`.

Exactly one geometry edge exists for each visible semantic relationship that
has renderable endpoints. No geometry edge may reference a relationship not in
the scene.

### EdgeLabel

Fields: `id`, `text`, `kind` (`count`), `bounds`, `position`, `visible`.

The first implementation uses existing relationship count text only. Label
identity is `label::<relationship-id>::count`.

### Junction

Fields: `id`, `position`, `incident_geometry_edge_ids[]`.

A junction is retained only when its coordinate is finite and at least two
visible geometry edges share it within the adapter tolerance. It has no
semantic relationship meaning.

### Route

The existing route shape remains authoritative: `kind`, connected `sections`,
line/cubic segments, and label position. Advanced output may add junction
references but does not permit non-finite points, disconnected sections, or
unknown segment kinds.

## Policies and invariants

- `edge_labels`, ports, junctions, and compound bounds never alter canonical model collections.
- `compound` may only reflect hierarchy already present in the semantic scene.
- `spline_refinement` accepts finite piecewise-cubic sections; malformed output falls back to orthogonal.
- The browser, embedded HTML, and browser SVG use equal geometry snapshots; Go static SVG uses deterministic orthogonal geometry.
- Semantic accessibility describes relationships and hierarchy even when geometry features are disabled or degraded.
- Unsupported feature combinations remain valid scenes with a visible `geometry_feature_unsupported` diagnostic.

## Lifecycle

`requested -> negotiated -> laid_out -> validated -> rendered`.

Feature-specific failures transition to `degraded -> fallback`; they do not
invalidate the canonical scene.

## Domain events

`LayoutFeaturesRequested`, `LayoutFeaturesNegotiated`,
`GeometrySnapshotProduced`, `GeometryValidationFailed`,
`GeometryFallbackApplied`, `GeometryRendered`.
