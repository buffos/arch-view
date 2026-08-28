# Multi-analyzer project orchestration canonical API/CLI contract

## Aggregate response

The combined analysis contract is `arch-view.aggregate/v1`:

```json
{
  "schema_version": "arch-view.aggregate/v1",
  "run_id": "run-123",
  "status": "partial",
  "repository": { "root_label": "repo" },
  "scopes": [
    {
      "scope_id": "scope-<sha256>",
      "project_root": "frontend",
      "analyzer": { "id": "org.archview.typescript", "version": "1.0.0", "language": "typescript", "api_version": "arch-view.analyzer/v1" },
      "runtime_source": "packaged",
      "status": "complete",
      "summary": { "module_count": 4, "relationship_count": 3, "diagnostic_count": 0 }
    },
    {
      "scope_id": "scope-<sha256>",
      "project_root": "services/api",
      "analyzer": { "id": "org.archview.python", "version": "1.0.0", "language": "python", "api_version": "arch-view.analyzer/v1" },
      "runtime_source": "packaged",
      "status": "failed",
      "diagnostics": [{ "code": "analyzer_package_launch_failed", "severity": "error", "message": "..." }]
    }
  ],
  "model": { "schema_version": "arch-view.aggregate-model/v1", "project": { "language": "mixed" }, "modules": [], "relationships": [], "references": [], "source_references": [], "diagnostics": [], "derived": {} },
  "diagnostics": [],
  "summary": { "job_count": 2, "usable_scope_count": 1, "failed_scope_count": 1 }
}
```

The aggregate model retains `scope_id` in per-observation provenance and uses
namespaced IDs. Single-scope consumers continue to receive the existing
`arch-view.model/v1` shape.

## Job-plan and progress contract

The host may expose a plan/progress stream with event types:

```json
{
  "schema_version": "arch-view.job-event/v1",
  "sequence": 7,
  "run_id": "run-123",
  "type": "job.completed",
  "job_id": "job-...",
  "scope_id": "scope-...",
  "status": "complete",
  "completed_jobs": 1,
  "total_jobs": 2
}
```

Required lifecycle types are `run.started`, `job.planned`, `job.started`,
`job.completed`, `job.failed`, `job.cancelled`, and `run.completed`. The job
list and snapshots are sorted by scope ID. `sequence` reflects emission order;
clients must use the per-job status and terminal run event rather than infer
completion from timing. Analyzer-internal percentage is optional.

## HTTP mapping

- `POST /v1/analyses` accepts the existing project request plus optional
  combined mode and assignment resolution, and returns a complete, partial,
  failed, or cancelled aggregate result.
- `GET /v1/analyses/{run_id}` returns the aggregate and scope summaries.
- `GET /v1/analyses/{run_id}/scopes` returns sorted scope summaries and
  diagnostics.
- `GET /v1/analyses/{run_id}/projection?scope=all|<scope_id>` returns the
  selected renderer-neutral projection; `scope=all` is the default.
- `GET /v1/analyses/{run_id}/events` may stream job lifecycle events; it does
  not change result semantics.

HTTP status rules remain: `200` for complete or partial usable results, `400`
for invalid requests, `409` for ambiguous/duplicate selection, `422` for
unavailable or invalid assigned analyzers, `500` for aggregate failure, and
`130`/cancelled semantics for caller cancellation through CLI.

## CLI mapping

```text
arch-view analyze --project <repository> --format analysis-json --output <file>
arch-view analyze --project <repository> --scope <scope-id> --format analysis-json --output <file>
arch-view open --project <repository>
```

Without a single-analyzer override, `analyze` plans the combined run. A
partial result exits `0` and carries `status: partial`; a run with no usable
scope uses the existing fatal/unsupported exit mapping. `--analyzer` or
`--language` constrains the selected root according to the assignment contract.

## Aggregation and parity rules

- Only analyzer-reported module, reference, source, relationship, and diagnostic observations are retained.
- Namespace rewriting is deterministic and does not alter local names or relationship types.
- No cross-scope relationship is inferred from matching names, paths, imports, or languages.
- Combined and individual scopes must expose the same facts for a scope; `All` adds other scopes and aggregate status only.
