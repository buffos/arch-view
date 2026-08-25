# Explore and inspect architecture canonical API/CLI contract

## Local HTTP contract

- `GET /` → local viewer application.
- `GET /v1/models/{model_id}` → canonical model metadata/content.
- `GET /v1/models/{model_id}/projection?path=<segment>&mode=overview|detail|list&reference_visibility=hidden|aggregated|expanded&reference_scope=<scope>` → renderer-neutral scene snapshot. `path` and `reference_scope` may be repeated; valid scopes are `standard_library`, `external`, `unresolved`, and `dynamic`.
- `GET /v1/source?model_id=<id>&path=<relative>&start_line=<n>&end_line=<n>` → read-only source excerpt when path is inside the model project root.
- `POST /v1/reanalysis` with `{ "project_root":"string", "language":"string|null", "options":{} }` → new complete/partial model revision; failed reanalysis leaves the prior revision active.

HTTP errors use the common `{error:{code,message,details}}` shape. `403`/`422` are used for source-root/path policy violations; `404` for unknown model/evidence.

## Renderer-neutral scene shape

```json
{
  "model_id":"string",
  "model_revision":"string",
  "hierarchy_path":[],
  "visible_nodes":[],
  "visible_relationships":[],
  "cycle_indicators":[],
  "diagnostic_indicators":[],
  "layer_labels":[],
  "reference_summary":{},
  "reference_details":[],
  "evidence_links":[],
  "accessibility": {"reading_order":[],"descriptions":{}}
}
```

The default overview uses `reference_visibility=hidden`: it shows project-local modules/groups and a boundary summary while retaining non-local references in the model and evidence list. `aggregated` shows one or more scope boundary nodes; `expanded` shows individual reference nodes. The `list` mode exposes `reference_details` for individual imports with target scope, confidence, contributor IDs, and evidence without requiring them to be rendered as graph nodes.

When a group contains non-cycle relationships between its child modules, the
overview may omit the resulting group self-loop and expose the canonical
contributors through `internal_relationship_ids[]` and
`counts.internal_relationship_count`. This is a presentation projection only;
canonical relationships and evidence remain unchanged. Real cycles are not
eligible for this collapse.

The first browser renderer is SVG/HTML. Canvas/WebGL adapters may consume the same shape; no renderer may invent semantic relationships or decide reference scope from labels.

## CLI mapping

- `arch-view open --model <model.json> [--port <n>]` → start local web viewer.
- `arch-view open --project <path> [--language <id>] [--port <n>]` → analyze then open.
- Viewer options may select reference visibility, but the default remains local-first overview behavior.

The CLI never enables source editing or target-code execution.

## Interaction parity

Browser interactions must preserve hierarchy paths, selection IDs, evidence IDs, cycle state, diagnostics, reference scope, confidence, and source locations across renderers. Keyboard/list mode must expose the same facts as the graphic view, including individual imports when a module or group is selected. Viewport and optional manual positions are session-scoped and keyed by model revision and hierarchy path.
