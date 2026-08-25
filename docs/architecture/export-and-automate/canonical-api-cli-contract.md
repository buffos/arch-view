# Export and automate canonical API/CLI contract

## Export request

```json
{
  "input_model": "path",
  "format": "json | html | svg",
  "output": "path",
  "view_path": [],
  "reference_visibility": "hidden | aggregated | expanded",
  "reference_scopes": ["standard_library | external | unresolved | dynamic"],
  "deterministic": true,
  "embed_source": false
}
```

`input_model`, `format`, and `output` are required. `reference_visibility` defaults to `hidden` for visual overview artifacts. JSON still preserves the complete canonical reference set; HTML/SVG use the selected local-first, aggregated, or expanded view policy. `embed_source=true` is unsupported in v1 and returns `UnsupportedOption`.

## CLI mapping

- `arch-view export --input model.json --format json --output architecture.json`.
- `arch-view export --input model.json --format html --output architecture.html`.
- `arch-view export --input model.json --format svg --output architecture.svg`.
- `--reference-visibility hidden|aggregated|expanded` and repeatable `--reference-scope <scope>` select the visual boundary policy.
- `arch-view analyze --project <path> [--language <id>] --format json|html|svg --output <file>` runs analysis, normalization, and export in one deterministic pipeline.

Common options: `--view-path <segment>...`, `--overwrite`, `--deterministic` (default true). Existing output is refused unless `--overwrite` is supplied.

Exit codes: `0` complete or partial artifact written; `2` invalid invocation/unsupported option; `3` invalid input model; `4` render/write failure; `130` cancellation.

## JSON contract

JSON is the exact `arch-view.model/v1` envelope defined by the model capability. Export must preserve field names, status, diagnostics, evidence IDs, derived cycles/layers, schema version, and deterministic ordering.

## HTML contract

HTML is self-contained: model/view data, CSS, and JavaScript are embedded; no network fetch is required. It exposes a local-first overview, hierarchy navigation, search, evidence/details, cycle/diagnostic states, reference-boundary controls, and accessible list/details/import inspection. Source contents are not embedded; source paths/locations are shown.

## SVG contract

SVG is static and scalable. It includes a title, descriptive metadata, accessible node labels, stable `data-module-id`/`data-relationship-id` attributes, reference scope/confidence metadata, cycle/diagnostic styling, and deterministic geometry derived from recorded layout provenance. It contains no scripts or network references.

## HTTP mapping

`POST /v1/exports` accepts the export request and returns artifact metadata/status. `GET /v1/exports/{job_id}` returns status/diagnostics. HTTP `200` covers written complete/partial artifacts; `400` invalid request; `422` invalid model/unsupported option; `500` render/write failure.

## Determinism and parity

CLI and HTTP preserve format meaning, status vocabulary, diagnostics, hashes, and artifact semantics. The future NDJSON plugin protocol is not an export format.
