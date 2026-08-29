# Multi-analyzer project orchestration canonical domain model

## Modeling boundary

This model owns discovery, job planning, execution coordination, aggregation,
and provenance. Analyzer semantics remain in analyzers; package trust remains
in compiled distribution; assignment precedence remains in the assignment
capability; graph derivation remains in canonical model generation.

## Aggregate: AnalysisRun

Fields:

- `run_id`
- `repository_root_label`
- `status`: `complete | partial | failed | cancelled`
- `job_plan`
- `scopes[]`
- `combined_model?`
- `diagnostics[]`
- `summary`

Invariant: `combined_model` exists only when at least one usable scope exists.

## JobPlan

Fields:

- `plan_version`: `arch-view.job-plan/v1`
- `repository_root`
- `invocation_root`
- `discovery_policy_version`
- `source_scope_policy`
- `jobs[]`
- `discovery_diagnostics[]`

Jobs are sorted by normalized relative root, language, and logical analyzer ID.
The plan is immutable once execution begins.

## AnalyzerJob

Fields:

- `job_id`
- `scope_id`
- `relative_project_root`
- `logical_analyzer_id`
- `analyzer_version?`
- `runtime_source`
- `selection_source`: `assignment | cli | automatic`
- `nested_root_exclusions[]`
- `effective_source_scope`
- `source_scope_fingerprint`
- `matched_source_set_fingerprint`
- `effective_options_fingerprint`
- `status`
- `started_at?`, `finished_at?`
- `result?`
- `diagnostics[]`

`job_id` is unique within a run. `scope_id` is stable across runs for the same
relative root and logical analyzer ID.

## SourceScopePolicy

The job plan carries the canonical source-scope policy resolved from the
nearest analysis configuration:

- `invocation_root`
- `exclude[]`
- `include[]` of `{ analyzer_id, globs[] }`
- `policy_version`

The global exclusion set is additive to fixed discovery safety exclusions,
nested-root exclusions, and analyzer-specific exclusion options. A present
analyzer include rule is an allowlist formed by the union of its globs; no rule
means no additional allowlist. Patterns use normalized POSIX paths relative to
the invocation root and support `*`, `?`, character classes, and recursive
`**`. Absolute paths, parent traversal, negation, comments, and malformed
patterns are invalid. Source filtering occurs after discovery and ownership
resolution, and exclusions always win over includes.

## ScopeIdentity

1. Normalize the repository-relative root with `/`; use `.` for the repository root.
2. Form `scope_key = relative_root + "\u0000" + logical_analyzer_id`.
3. Set `scope_id = "scope-" + lowercase(hex(SHA-256(scope_key)))`.
4. Rewrite local observation IDs as `scope_id + "::" + percent-encoded(local_id)`.

Modules, references, source references, relationships, and diagnostics use
their corresponding rewritten IDs. The `::` delimiter is reserved by the
aggregate serializer. Relationship endpoints may target only observations in
the same scope unless a future contract explicitly adds cross-scope links.

## CombinedModel

The aggregate model uses `arch-view.aggregate/v1` and contains:

- repository-level project metadata with `language: mixed`
- `scopes[]` with scope ID, root, analyzer, runtime source, status, counts, and
  effective source-scope fingerprints
- namespaced `modules[]`, `references[]`, `source_references[]`, and `relationships[]`
- aggregate diagnostics with `scope_id` when scope-specific
- derived cycles/layers calculated only over the namespaced reported graph

The aggregate is a language-neutral model extension; it never changes the
meaning of a single-scope `arch-view.model/v1` result.

## Policies and invariants

- Strong nested roots are carved out of parent input; weak markers do not create automatic roots.
- Fixed discovery filtering runs before configured source filtering; configured
  includes cannot hide root markers and cannot re-include excluded content.
- One logical analyzer per root/language is the default; duplicate IDs are not scheduled twice.
- Canonical collections are normalized after namespacing and sorted by stable IDs.
- A scope failure cannot remove a successful scope.
- No cross-language or cross-root relationship is inferred.
- `complete` requires every planned job to be usable and non-partial.
- `partial` requires at least one usable scope and at least one non-complete scope.
- `failed` means no usable scope and at least one non-cancellation failure or no applicable analyzer.
- `cancelled` means no usable scope remained when cancellation stopped the run.

## Lifecycle

`planned -> queued -> running -> complete|partial|failed|cancelled|skipped`.
Terminal job states cannot transition. A run becomes terminal only after all
queued/running jobs have reached terminal state.

## Domain events

`ProjectRootDiscovered`, `AnalyzerJobPlanned`, `AnalyzerJobStarted`,
`AnalyzerJobProgressed`, `AnalyzerJobCompleted`, `AnalyzerJobFailed`,
`AnalyzerJobCancelled`, `AnalysisRunAggregated`, `AnalysisRunCompleted`.
