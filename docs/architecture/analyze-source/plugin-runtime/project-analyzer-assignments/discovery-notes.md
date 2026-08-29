# Project analyzer assignments and view selection discovery notes

## Purpose

Give users durable, repository-relative control over analyzer ownership and a
simple way to move between individual analyzer scopes and the combined view.

## Current implementation

- **Observed in code:** The viewer supports nearest-ancestor `.archview.json`
  discovery for presentation layout, and `open` receives one analyzer result.
- **Observed in code:** Individual language analyzers expose additional
  repository-relative `exclude` options, but the host has no persisted,
  cross-analyzer include/exclude policy in `.archview.json`.
- **Inferred from docs:** Analyzer options and layout settings are intentionally
  separate today; the current configuration file does not persist folder-to-
  analyzer assignments, source-scope filters, or an analyzer-scope selector.

## User-confirmed target behavior

- A configuration section can map repository-relative folders or project roots
  to analyzer IDs.
- Automatic detection remains available as a fallback.
- A dropdown can switch between `All` and individual analyzer/project scopes
  without re-running already completed jobs.
- Changing one assignment reanalyzes only the affected scope where possible.
- The mapping can distinguish implementations such as built-in versus
  compiled external Python analyzers.
- A configuration can add global exclusion globs and analyzer-scoped include
  globs for the source set analyzed by each job.

## Specified decisions

- Assignments live in a separate top-level `analysis` section of the existing
  versioned `.archview.json`; `layout` remains presentation-only. Adding the
  section requires a compatible configuration-schema revision and preserves
  v1 layout-only files as configurations with no assignments.
- The v2 `analysis` section also contains an optional global `exclude` array
  and an optional `include` array of `{ analyzer_id, globs[] }` rules. The
  logical analyzer ID is canonical; language is retained in analyzer metadata
  and is not an ambiguous configuration key.
- Include and exclude patterns are normalized POSIX globs relative to the
  canonical invocation root supplied by `--project` or the equivalent API
  request. They are not rebased to the configuration file's directory or the
  process working directory. A rule's globs are unioned for that analyzer;
  an analyzer without an include rule has no additional allowlist.
- Source-scope filtering is applied after root discovery and nested ownership
  resolution. It cannot hide manifest markers needed to discover project roots.
  Fixed safety exclusions, nested-root exclusions, and configured excludes win
  over includes; includes cannot re-include a path excluded by policy.
- The v2 glob language supports `/`-normalized relative paths, `*`, `?`,
  character classes, and recursive `**` directory matching. Patterns are
  deterministic and do not support `.gitignore` negation, comments, or
  parent-traversal syntax in the first version. Matching a directory includes
  its descendants for source selection and prunes them for exclusion.
- Keys are normalized repository-relative folders or project roots, and values
  reference stable logical analyzer IDs rather than deployment-specific
  `.external` IDs. The repository root is representable as `.`.
- The nearest discovered configuration file remains the one complete file, as
  in current layout resolution. Within that file, the deepest matching
  assignment wins; explicit CLI analyzer/language selection overrides the
  assignment; automatic detection is the fallback. Invalid nearest config is
  surfaced rather than bypassed, and unavailable analyzer IDs produce scoped
  diagnostics.
- The runtime stores a combined model plus cached per-job results. The
  application dropdown offers `All` and individual project/analyzer scopes;
  changing scope filters/selects existing results and does not re-run analysis.
- Editing an assignment or its options invalidates only affected jobs where
  possible. If the active scope disappears, the view returns to `All`; model
  revisions and stale evidence remain explicit.
- Assignment-scoped non-sensitive analyzer options may be stored beside the
  mapping. Analyzer defaults are overridden by assignment options, and CLI
  options override both; sensitive values remain outside persisted project
  configuration.

## Implementation and verification focus

The exact-spec set defines the v2 JSON schema, assignment and source-scope
validation/error codes, precedence for combined CLI/config cases, cache
invalidation keys, and the dropdown/API contract. Implementation and
verification must now cover nearest-file errors, unavailable analyzers,
invalid/unsafe glob patterns, include/exclude precedence, affected-scope
invalidation, cache reuse, and combined/per-scope viewer projections.

## Boundary

The runtime owns assignment semantics, source-scope policy, and analyzer
selection; the viewer owns the dropdown presentation and scope navigation.
Multi-analyzer orchestration applies the resolved filters to each job. Layout
settings remain presentation-only and must not alter analyzer facts or
canonical model meaning.
