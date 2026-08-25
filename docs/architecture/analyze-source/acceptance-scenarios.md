# Analyze source code acceptance scenarios

## SC-AS-001 — Analyze a resolvable project

Given a supported project with one unambiguous analyzer and readable source, when analysis runs, then it returns `complete`, project-local modules, static relationships, and source evidence without executing target code.

Verification: backend-boundary `when-supported`; frontend-integration `not-applicable`; end-to-end `when-supported`.

## SC-AS-002 — Preserve a partial result

Given a project with one unresolved or dynamic dependency, when analysis runs, then it returns usable observations with status `partial`, a non-fatal diagnostic, and no fabricated local module.

Verification: backend-boundary `when-supported`; frontend-integration `not-applicable`; end-to-end `when-supported`.

## SC-AS-003 — Reject ambiguous detection

Given a root matching multiple analyzers with no explicit language, when analysis starts, then it returns an ambiguity outcome and does not choose an analyzer silently.

Verification: backend-boundary `when-supported`; frontend-integration `not-applicable`; end-to-end `when-supported`.

## SC-AS-004 — Honor explicit selection

Given a valid project and an explicit supported language, when analysis starts, then that analyzer is selected or a clear incompatibility diagnostic is returned.

Verification: backend-boundary `when-supported`; frontend-integration `not-applicable`; end-to-end `when-supported`.

## SC-AS-005 — Retain multi-file evidence

Given one module represented by multiple source files, when analysis completes, then all contributing source references remain attached to that module.

Verification: backend-boundary `when-supported`; frontend-integration `not-applicable`; end-to-end `when-supported`.

## SC-AS-006 — Enforce exclusions and safety

Given tests, generated files, vendor/cache directories, `.git`, and `external/`, when default analysis runs, then they are excluded and the target application is not executed.

Verification: backend-boundary `when-supported`; frontend-integration `not-applicable`; end-to-end `when-supported`.

## SC-AS-007 — Cancellation is not completion

Given a running analysis, when cancellation occurs, then the result is `cancelled` and is never reported as `complete`.

Verification: backend-boundary `when-supported`; frontend-integration `not-applicable`; end-to-end `when-supported`.

## SC-AS-008 — Deterministic repeat

Given unchanged source, analyzer version, options, and configuration, when the same analysis runs twice, then stable fields and collection ordering match.

Verification: backend-boundary `when-supported`; frontend-integration `not-applicable`; end-to-end `when-supported`.
