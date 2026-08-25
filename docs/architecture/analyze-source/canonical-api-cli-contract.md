# Analyze source code canonical API/CLI contract

## Contract goals

Expose one stable analysis meaning through local HTTP and CLI while allowing analyzer implementations to vary.

## Transport-neutral request

```json
{
  "project_root": "string",
  "language": "go | python | typescript | rust | clojure | null",
  "include_tests": false,
  "include_generated": false,
  "include_external": false,
  "excludes": ["string"],
  "options": {},
  "safe_mode": true
}
```

`project_root` is required. Boolean defaults are false except `safe_mode`, which defaults true.

## Transport-neutral response

```json
{
  "run_id": "string",
  "status": "complete | partial | failed | cancelled",
  "analyzer": {"id": "string", "version": "string", "language": "string", "api_version": "string"},
  "project": {"root_label": "string", "boundary": "string"},
  "modules": [],
  "relationships": [],
  "references": [],
  "source_references": [],
  "diagnostics": [],
  "summary": {"module_count": 0, "relationship_count": 0, "diagnostic_count": 0}
}
```

Module, relationship, reference, source-reference, diagnostic, and confidence shapes are defined in the canonical domain model and must be serialized without lossy field renaming.

## HTTP mapping

- `GET /v1/analyzers` → supported analyzer manifests.
- `POST /v1/analyses` → run analysis synchronously for the first slice; `200` for complete/partial, `400` for invalid request, `409` for ambiguous selection, `422` for unsupported language/options, `500` for host failure.
- `GET /v1/analyses/{run_id}` → previously returned result; `404` when unknown.

Error body: `{ "error": { "code": "string", "message": "string", "details": {} } }`.

## CLI mapping

- `arch-view analyzers` → list manifests.
- `arch-view analyze --project <path> [--language <id>] [--include-tests] [--include-generated] [--include-external] [--exclude <glob>] --format analysis-json --output <file>`.

Exit codes: `0` complete or partial; `2` invalid request/ambiguous selection; `3` unsupported or unreadable project; `4` fatal analyzer/host failure; `130` cancellation.

## Parity and safety

HTTP and CLI preserve status, diagnostics, identifiers, and result meaning. Paths are normalized relative to the project boundary. The host never runs target application code; any optional tool-assisted mode is explicit and reported.

## Minimum first slice

`arch-view analyze --project <go-root> --format analysis-json --output <file>` with Go analyzer, local modules/imports, evidence, diagnostics, and deterministic output.
