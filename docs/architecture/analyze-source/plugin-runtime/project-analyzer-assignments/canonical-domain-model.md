# Project analyzer assignments and view selection canonical domain model

## Modeling boundary

This model owns persisted assignment semantics, selection precedence, cache
identity/invalidation, and scope navigation. Layout remains owned by the
viewer layout capability; analyzer package trust remains in compiled
distribution; job execution and aggregation remain in multi-analyzer
orchestration.

## ConfigurationProfile

Fields:

- `schema_version`: `arch-view.config/v1` or `arch-view.config/v2`
- `layout` (existing presentation profile)
- `analysis?` (v2 only)

V1 decodes as a profile with no assignments. V2 is strict and rejects unknown
fields, malformed paths, duplicate paths, invalid option shapes, and invalid
schema versions.

## AnalysisConfiguration

Fields:

- `assignments[]`
- `exclude[]?`: global source-scope globs
- `include[]?`: analyzer-scoped include rules

Assignments are sorted by normalized path for canonical serialization. Include
rules are sorted by `analyzer_id` and their glob arrays are sorted and
deduplicated for canonical serialization. No ancestor configuration is merged.

## SourceScopePolicy

Fields:

- `invocation_root`
- `exclude[]`
- `include[]`
- `policy_version`

`exclude[]` is a global additive exclusion set. Each include rule contains:

- `analyzer_id`
- `globs[]`

An analyzer with no include rule has no additional allowlist. When a rule is
present, matching paths are the union of its globs. Patterns are normalized
POSIX paths relative to `invocation_root`, which is the canonical `--project`
directory or equivalent API root. They are not relative to the configuration
file or process working directory.

The v2 glob subset supports `*`, `?`, character classes, and recursive `**`.
Patterns cannot be absolute, contain parent traversal, or use `.gitignore`
negation/comments. Directory matches apply recursively. Fixed safety
exclusions, nested-root exclusions, and configured exclusions take precedence
over includes. Root discovery and nested ownership are evaluated before this
policy filters analyzer source input. A job's effective source set is its owned
discovered candidates intersected with its analyzer include union, when
present, and then reduced by all applicable exclusions.

## AnalyzerAssignment

Fields:

- `path`
- `analyzer_id`
- `options?`

Invariants: path is `.` or a non-empty repository-relative POSIX path without
absolute roots, `..`, empty segments, or glob characters; analyzer ID is a
stable logical ID; options contain only non-sensitive values declared by that
analyzer manifest.

## AssignmentResolution

Fields: `project_root`, `source`, `assignment_path?`, `analyzer_id?`,
`effective_options`, `source_scope`, `diagnostics[]`.

Resolution order:

1. Explicit CLI analyzer/language selection for the selected root.
2. Deepest assignment whose path is an ancestor of the project root.
3. Automatic detection from the analyzer registry.

If CLI analyzer and language disagree, the request is invalid. A malformed
nearest file stops resolution. An unavailable assigned analyzer creates a
scoped diagnostic and does not silently fall through to a different analyzer.

## AnalysisScope

Fields: `scope_id`, `relative_project_root`, `analyzer`, `runtime_source`,
`source_scope`, `status`, `summary`, `diagnostics[]`, `cache_key`.

The scope ID is the multi-analyzer `ScopeIdentity`; labels are derived from
relative root and analyzer language/ID and are not identity keys.

## AnalysisCacheEntry

The cache is session-scoped and keyed by the SHA-256 of canonical JSON over:

- schema and discovery-policy versions
- normalized relative project root
- logical analyzer ID, version, API version, and package digest/runtime source
- resolved selection source
- canonical effective option values
- canonical effective source-scope policy and the normalized matched source set
- authoritative source-input fingerprint

The source fingerprint is based on sorted included relative paths and content
digests. Assignment edits invalidate scopes whose effective assignment,
project-root ownership, analyzer, options, or source set changes; unrelated
scope entries remain valid.

## Policies and invariants

- The nearest configuration file is one complete file and invalid nearest configuration is surfaced.
- The source-scope policy is resolved relative to the invocation root and is
  applied only after project-root discovery and nested ownership resolution.
- Global/configured exclusions and fixed/nested exclusions always win over
  includes; an include cannot re-include excluded content.
- Scope selection is read-only and cannot trigger analysis.
- `All` is a projection over the current aggregate run, not a cached duplicate job.
- Sensitive analyzer values cannot be persisted in project configuration.
- If a selected scope is removed, active selection returns to `All`.

## Domain events

`AnalysisConfigurationLoaded`, `AnalysisConfigurationRejected`,
`AnalyzerAssignmentResolved`, `AnalysisScopeInvalidated`,
`AnalysisScopesCached`, `AnalysisScopeSelected`.
