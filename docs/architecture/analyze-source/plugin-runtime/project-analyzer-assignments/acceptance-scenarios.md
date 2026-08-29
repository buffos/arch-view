# Project analyzer assignments and view selection acceptance scenarios

## SC-PAA-001 — Preserve layout-only configuration compatibility

**Given** an existing `arch-view.config/v1` layout-only file, **when** the host
loads it, **then** layout behavior is unchanged and analysis behaves as if no
assignments were configured.

Verification: backend-boundary `when-supported`; frontend-integration
`when-supported`; end-to-end `when-supported`.

## SC-PAA-002 — Resolve the deepest assignment

**Given** assignments for `.` and `frontend`, **when** the host resolves a
project root under `frontend`, **then** the `frontend` analyzer and options win
over the root assignment.

Verification: backend-boundary `when-supported`; frontend-integration
`not-applicable`; end-to-end `when-supported`.

## SC-PAA-003 — Apply CLI precedence

**Given** a matching project assignment, **when** the user supplies a compatible
explicit CLI analyzer or language, **then** the CLI selection wins; conflicting
CLI values are rejected as a selection error.

Verification: backend-boundary `when-supported`; frontend-integration
`when-supported`; end-to-end `when-supported`.

## SC-PAA-004 — Surface invalid nearest configuration

**Given** an invalid nearest `.archview.json` and a valid ancestor file,
**when** the repository is opened, **then** the host reports the nearest-file
configuration error and does not silently use the ancestor file.

Verification: backend-boundary `when-supported`; frontend-integration
`when-supported`; end-to-end `when-supported`.

## SC-PAA-005 — Reject unsafe or duplicate paths

**Given** an absolute path, parent traversal, glob, or duplicate normalized
assignment path, **when** v2 configuration is loaded, **then** the configuration
is rejected with field/path details and no unsafe path is analyzed.

Verification: backend-boundary `when-supported`; frontend-integration
`not-applicable`; end-to-end `when-supported`.

## SC-PAA-006 — Keep unavailable assignments scoped

**Given** one valid assignment and one unavailable analyzer ID, **when** a
combined run is planned, **then** the valid scope remains runnable and the
unavailable assignment produces a scoped diagnostic rather than being silently
reassigned.

Verification: backend-boundary `when-supported`; frontend-integration
`when-supported`; end-to-end `when-supported`.

## SC-PAA-007 — Cache scope switching

**Given** a completed multi-scope run, **when** the user switches between `All`
and an individual scope, **then** the viewer changes projection from cached
results and no analyzer process is started.

Verification: backend-boundary `when-supported`; frontend-integration
`when-supported`; end-to-end `when-supported`.

## SC-PAA-008 — Invalidate only affected scopes

**Given** cached results for unrelated project roots, **when** one assignment
or its effective options changes, **then** the affected scope cache key is
invalidated, unaffected scope entries remain reusable, and the new run reports
the invalidation reason.

Verification: backend-boundary `when-supported`; frontend-integration
`when-supported`; end-to-end `when-supported`.

## SC-PAA-009 — Recover selection when a scope disappears

**Given** the viewer is showing one scope, **when** reanalysis removes that
scope, **then** the active selection returns to `All` and the user can still see
the aggregate diagnostics.

Verification: backend-boundary `when-supported`; frontend-integration
`when-supported`; end-to-end `when-supported`.

## SC-PAA-010 — Keep layout and analysis independent

**Given** a layout profile and an analyzer assignment in the same v2 file,
**when** either is changed, **then** layout changes affect presentation only and
assignment changes affect analyzer planning/cache only; neither changes the
other's semantics.

Verification: backend-boundary `when-supported`; frontend-integration
`when-supported`; end-to-end `when-supported`.

## SC-PAA-011 — Apply invocation-root source filters

**Given** a v2 configuration with global `exclude` globs and a selected
invocation root, **when** a combined plan is built, **then** matching source
paths are excluded from every analyzer job, paths are normalized relative to
the invocation root, directory matches apply recursively, and fixed safety or
nested-root exclusions remain effective.

Verification: backend-boundary `when-supported`; frontend-integration
`not-applicable`; end-to-end `when-supported`.

## SC-PAA-012 — Apply analyzer-scoped includes

**Given** analyzer include rules for Go and TypeScript, **when** a mixed
repository is analyzed, **then** each analyzer receives only the union of its
own matching include globs, an analyzer without a rule receives no additional
allowlist, and a configured exclude wins over an include. Root markers remain
visible to discovery even when they do not match a source include.

Verification: backend-boundary `when-supported`; frontend-integration
`not-applicable`; end-to-end `when-supported`.

## SC-PAA-013 — Reject unsafe or ambiguous source filters

**Given** an absolute, parent-traversing, empty, backslash-separated, negated,
malformed, or duplicate-analyzer include rule, **when** v2 configuration is
loaded, **then** the nearest configuration is rejected with field/pattern
details and no unsafe source path is analyzed.

Verification: backend-boundary `when-supported`; frontend-integration
`not-applicable`; end-to-end `when-supported`.

## SC-PAA-014 — Invalidate scopes when filters change

**Given** cached results for unrelated analyzer scopes, **when** a global
exclude or analyzer include rule changes the effective source set, **then** the
affected scope cache key is invalidated, unaffected scope entries remain
reusable where their matched source set is unchanged, and the run reports the
filter invalidation reason.

Verification: backend-boundary `when-supported`; frontend-integration
`when-supported`; end-to-end `when-supported`.
