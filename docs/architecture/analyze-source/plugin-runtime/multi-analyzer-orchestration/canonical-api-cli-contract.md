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
       "source_scope": { "policy_fingerprint": "sha256:...", "matched_source_set_fingerprint": "sha256:..." },
       "status": "complete",
      "summary": { "module_count": 4, "relationship_count": 3, "diagnostic_count": 0 }
    },
    {
      "scope_id": "scope-<sha256>",
      "project_root": "services/api",
      "analyzer": { "id": "org.archview.python", "version": "1.0.0", "language": "python", "api_version": "arch-view.analyzer/v1" },
       "runtime_source": "packaged",
       "source_scope": { "policy_fingerprint": "sha256:...", "matched_source_set_fingerprint": "sha256:..." },
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
namespaced IDs. Scope summaries retain the effective source-scope policy and
matched-source-set fingerprints used for the job. Single-scope consumers continue to receive the existing
`arch-view.model/v1` shape.

## Source-scope policy contract

The job plan includes the resolved source-scope policy:

```json
{
  "policy_version": "arch-view.source-scope/v1",
  "invocation_root": ".",
  "exclude": ["**/generated/**", "**/vendor/**"],
  "include": [
    { "analyzer_id": "org.archview.go", "globs": ["cmd/**", "internal/**"] },
    { "analyzer_id": "org.archview.typescript", "globs": ["frontend/src/**"] }
  ]
}
```

The policy is resolved from the nearest v2 `.archview.json`. `exclude` applies
to every job. A matching `include` rule selects the union of its globs for that
logical analyzer; an analyzer without a rule has no additional allowlist. The
patterns are normalized POSIX paths relative to the invocation root. The v1
glob subset supports `*`, `?`, character classes, and recursive `**`, with
directory matches applying to descendants. Absolute paths, `..`, empty or
malformed patterns, backslashes, comments, and `.gitignore` negation are
invalid. Fixed safety exclusions and nested-root exclusions are applied before
source filtering, and every exclusion wins over an include. For each job,
filtering intersects the job's owned discovered source candidates with its
analyzer include union when present, then removes all applicable exclusions.

The effective source-scope policy and normalized matched source set are part of
the deterministic job identity and authoritative cache fingerprint. Discovery
markers remain eligible for root discovery even when they are outside an
analyzer's source include set.

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
  combined mode, assignment resolution, and the nearest configuration's
  validated source-scope policy, and returns a complete, partial, failed, or
  cancelled aggregate result.
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

Without a single-analyzer override, `analyze` plans the combined run. The
nearest `.archview.json` source-scope policy is anchored to the invocation root
and is included in the plan. A partial result exits `0` and carries
`status: partial`; a run with no usable scope uses the existing fatal/unsupported
exit mapping. `--analyzer` or `--language` constrains the selected root
according to the assignment contract.

## Aggregation and parity rules

- Only analyzer-reported module, reference, source, relationship, and diagnostic observations are retained.
- Namespace rewriting is deterministic and does not alter local names or relationship types.
- No cross-scope relationship is inferred from matching names, paths, imports, or languages.
- Combined and individual scopes must expose the same facts for a scope; `All` adds other scopes and aggregate status only.
