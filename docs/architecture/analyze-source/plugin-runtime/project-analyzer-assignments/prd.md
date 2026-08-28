# Project analyzer assignments and view selection PRD

## Purpose

Give developers durable, repository-relative control over analyzer ownership
and a visible way to switch among the combined architecture and individual
project/analyzer scopes.

## Actors

- **Developer:** edits project configuration or chooses a scope in the viewer.
- **CI/documentation operator:** runs analysis with deterministic configuration and consumes the selected or combined output.
- **Analyzer host:** resolves configuration, precedence, effective options, and cache invalidation.
- **Viewer:** presents available scopes and requests a projection without deciding analyzer semantics.

## Goals

1. Keep analyzer assignments in a dedicated `analysis` configuration section.
2. Preserve existing layout-only `.archview.json` behavior.
3. Make path matching and precedence deterministic across operating systems.
4. Let explicit CLI selection override project assignments.
5. Expose `All` and individual project/analyzer scopes from cached results.
6. Reanalyze only affected jobs after assignment or option changes where possible.
7. Surface invalid configuration and unavailable analyzers rather than silently bypassing them.

## Non-goals

- Moving layout settings into the analysis section or allowing analysis to alter layout/model semantics.
- Multiple config-file merging or inheritance across ancestor files.
- Arbitrary executable commands, secrets, target-code execution, or project-local plugin discovery.
- Remote configuration, shared team configuration, or persistent cross-session result cache.
- A UI editor for every configuration field; the first contract must support file/CLI/API consumption and viewer selection.

## Configuration workflow

The host discovers the nearest `.archview.json` from the selected repository
root toward the filesystem root. The nearest file is one complete profile. A
v1 layout-only file is valid with no assignments. A v2 file may contain the
exact `analysis` section defined in [canonical-api-cli-contract.md](canonical-api-cli-contract.md).

For a project root, the host chooses a CLI analyzer/language selection first,
then the deepest matching assignment, then automatic detection. In combined
mode, all valid assignments and unassigned automatic roots become jobs. A
deeper assignment claims its subtree and overrides its ancestor assignment.

## Scope-selection workflow

After a combined run, the viewer receives sorted scope summaries. It shows
`All` plus one entry per discovered job. Selecting an entry requests a
projection from cached results. A failed scope remains selectable for its
diagnostics; if the active scope disappears after reanalysis, the viewer
returns to `All`.

## Functional requirements

| ID | Requirement |
|---|---|
| PAA-FR-001 | Read v1 layout-only and v2 analysis-enabled `.archview.json` files without changing layout semantics. |
| PAA-FR-002 | Validate assignment paths as normalized repository-relative paths and reject duplicate normalized paths. |
| PAA-FR-003 | Resolve CLI selection before deepest matching assignment, then use automatic detection as fallback. |
| PAA-FR-004 | Preserve the nearest-file-wins rule and surface invalid nearest configuration instead of using a farther file. |
| PAA-FR-005 | Validate assignment analyzer IDs and options against available manifests without executing target code. |
| PAA-FR-006 | Build combined analysis from valid assignment and automatic scopes, retaining scoped diagnostics for unavailable assignments. |
| PAA-FR-007 | Cache per-job results with an input-complete key and invalidate only affected jobs after assignment/options/source changes where possible. |
| PAA-FR-008 | Expose `All` and individual scopes with stable IDs, labels, statuses, counts, and diagnostics. |
| PAA-FR-009 | Make scope selection a projection/filter operation that never re-runs an analyzer. |
| PAA-FR-010 | Keep assignment options, analyzer options, layout options, and sensitive values under separate ownership. |

## Non-functional requirements

- The same repository/configuration produces the same assignment resolution and scope ordering.
- Configuration errors are actionable and include path/field details without exposing secrets.
- Scope labels are human-readable, while scope IDs are stable and machine-safe.
- Cache entries are immutable snapshots and cannot silently substitute stale evidence.

## Success criteria

The scenarios in [acceptance-scenarios.md](acceptance-scenarios.md) demonstrate
configuration compatibility, precedence, invalidation, partial scope status,
and user-visible scope switching.
