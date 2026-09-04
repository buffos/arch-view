# Advanced ELK renderer support canonical API/CLI contract

The approved [delivery contract](delivery-contract.md) updates this baseline
for shared architecture/OKF delivery, persistence, stage ordering, and browser-only
ELK execution. Its AER-R rules and scenario mapping are authoritative.

## Feature profile

The v2 layout profile extends the existing `layout` object additively:

```json
{
  "schema_version": "arch-view.config/v2",
  "layout": {
    "algorithm": "layered",
    "options": {
      "org.eclipse.elk.edgeRouting": "SPLINES"
    },
    "features": ["edge_labels", "spline_refinement"]
  }
}
```

Allowed feature values are `edge_labels`, `junctions`, `ports`, `compound`,
and `spline_refinement`. Architecture omission is equivalent to `[]`; OKF
profile omission inherits and explicit `[]` clears inherited features. The
analysis assignment capability may use the same `arch-view.config/v2` file;
its `analysis` section is independent of `layout.features`.

## Feature catalog response

`GET /v1/layout/options` adds:

```json
{
  "features": [
    { "id": "edge_labels", "status": "supported", "surfaces": ["browser", "html", "browser-svg"] },
    { "id": "junctions", "status": "supported", "surfaces": ["browser", "html", "browser-svg"] },
    { "id": "ports", "status": "supported", "surfaces": ["browser", "html", "browser-svg"] },
    { "id": "compound", "status": "supported", "surfaces": ["browser", "html", "browser-svg"] },
    { "id": "spline_refinement", "status": "supported", "surfaces": ["browser", "html", "browser-svg"] }
  ]
}
```

The response must mark broader ELK options as `catalog-only` until they have a
renderer mapping and acceptance fixture.

The example lists final delivery availability, not current stage support.
Each feature also declares order, prerequisites, supported algorithms,
required options, owned geometry, delivery stage and fallback policy. Stage 1
publishes these five features as not-implemented, not selectable.

## Renderer-neutral geometry

The shared browser pipeline creates geometry locally. The following historical
architecture example illustrates the shape; source identity follows the
updated domain model for both architecture and OKF. No server-generated
geometry or ELK execution endpoint is introduced:

```json
{
  "geometry": {
    "schema_version": "arch-view.geometry/v1",
    "model_id": "model-1",
    "model_revision": "model-1",
    "hierarchy_path": [],
    "nodes": [
      { "id": "module-a", "semantic_node_id": "module-a", "bounds": { "x": 10, "y": 20, "width": 190, "height": 82 }, "parent_id": null, "children_ids": [], "ports": [] }
    ],
    "edges": [
      {
        "id": "rel-a",
        "semantic_relationship_id": "rel-a",
        "source_node_id": "module-a",
        "target_node_id": "module-b",
        "labels": [{ "id": "label::rel-a::count", "text": "1", "kind": "count", "bounds": { "x": 90, "y": 50, "width": 8, "height": 16 }, "visible": true }],
        "junctions": [],
        "route": { "kind": "spline", "sections": [], "label": { "x": 94, "y": 58 } }
      }
    ],
    "diagnostics": [],
    "provenance": { "engine": "elkjs", "features": ["edge_labels", "spline_refinement"] }
  }
}
```

The abbreviated example omits the ordinary finite route sections for clarity;
an emitted route must contain connected line/cubic sections. Geometry IDs map
to semantic scene IDs, while ports, labels, junctions, and layout containers
remain presentation-only.

## HTTP behavior

- `POST /v1/layout/apply` accepts `layout.features` and validates feature/option compatibility.
- `GET /v1/layout/config` reports active features, supported surfaces, and geometry diagnostics.
- Projection endpoints return semantic data. The shared browser runtime produces
  and validates geometry, including local fallback diagnostics.
- An unknown feature returns `422` with `renderer_feature_unknown`.

## CLI and export behavior

Browser and self-contained HTML sessions consume advanced geometry. Browser
Download SVG serializes that current geometry. Static Go SVG remains
deterministic orthogonal; if advanced features are requested or present, its
metadata records `advanced_features_not_applied` and it still emits valid
semantic SVG.

## Error and accessibility rules

Errors use the common `{ "error": { "code", "message", "details" } }` shape.
Geometry diagnostics never replace semantic accessibility. Relationship labels
are included in accessible descriptions, ports and junctions are omitted from
the architecture reading order, and compound containers follow existing
hierarchy reading order.
