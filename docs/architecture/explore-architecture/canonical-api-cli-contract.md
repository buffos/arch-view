# Explore and inspect architecture canonical API/CLI contract

## Local HTTP contract

- `GET /` → local viewer application.
- `GET /v1/models/{model_id}` → canonical model metadata/content.
- `GET /v1/models/{model_id}/projection?path=<segment>&mode=overview|detail|list` → renderer-neutral scene snapshot.
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
  "evidence_links":[],
  "accessibility": {"reading_order":[],"descriptions":{}}
}
```

The first browser renderer is SVG/HTML. Canvas/WebGL adapters may consume the same shape; no renderer may invent semantic relationships.

## CLI mapping

- `arch-view open --model <model.json> [--port <n>]` → start local web viewer.
- `arch-view open --project <path> [--language <id>] [--port <n>]` → analyze then open.

The CLI never enables source editing or target-code execution.

## Interaction parity

Browser interactions must preserve hierarchy paths, selection IDs, evidence IDs, cycle state, diagnostics, and source locations across renderers. Keyboard/list mode must expose the same facts as the graphic view.
