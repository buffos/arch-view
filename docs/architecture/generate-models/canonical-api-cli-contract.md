# Generate architecture models canonical API/CLI contract

## Canonical JSON envelope

```json
{
  "schema_version": "arch-view.model/v1",
  "model_id": "string",
  "status": "complete | partial | failed",
  "project": {"root_label": "string", "boundary": "string", "language": "string"},
  "analyzer": {"id": "string", "version": "string", "api_version": "string"},
  "modules": [{"id":"string","kind":"string","language":"string","name":"string","display_name":"string","hierarchy":[],"source_reference_ids":[],"tags":[],"metadata":{}}],
  "references": [{"id":"string","name":"string","scope":"external | standard_library | unresolved | dynamic","language":"string","metadata":{}}],
  "source_references": [{"id":"string","path":"string","start":{"line":1,"column":1},"end":{"line":1,"column":1},"symbol":"string","kind":"string"}],
  "relationships": [{"id":"string","type":"depends_on","from_module_id":"string","to_module_id":"string","to_reference_id":"string","source_reference_ids":[],"confidence":{"basis":"resolved","score":1},"metadata":{}}],
  "diagnostics": [],
  "derived": {"cycles":[],"feedback_relationship_ids":[],"layers":[],"algorithm_provenance":{}}
}
```

Exactly one of `to_module_id` and `to_reference_id` is present. Optional fields shown with string placeholders are omitted when absent. `score` is optional and ranges from 0 to 1. `depends_on` direction is `from` depends on `to`.

## HTTP mapping

- `POST /v1/models:normalize` → normalize an analysis result; `200` complete/partial, `400` malformed request, `422` invalid model, `500` host failure.
- `POST /v1/models:validate` → validate without deriving projections.
- `GET /v1/models/{model_id}/projection?path=<segments>` → hierarchy projection.

## CLI mapping

- `arch-view model normalize --input <analysis-json> --output <model-json>`.
- `arch-view model validate --input <model-json>`.
- `arch-view model projection --input <model-json> --path <segment>... --output <projection-json>`.

Exit codes: `0` valid complete/partial model; `2` malformed invocation; `3` invalid model; `4` derivation/resource failure.

## Determinism and compatibility

The schema version is required. Stable IDs, sorted collections, normalized paths, algorithm/version provenance, and no implicit timestamps are required. A future schema version must be explicitly negotiated or rejected; silent shape guessing is forbidden.
