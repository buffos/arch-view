# 046 — Reuse session scope results and selectively invalidate affected jobs

## Issue Metadata

- Issue number: `046`
- Owning capability node: `/.okf/capabilities/analyze-source/plugin-runtime/project-analyzer-assignments.md`
- Related consumer node: `/.okf/capabilities/analyze-source/plugin-runtime/multi-analyzer-orchestration.md`
- Artifact root: `docs/architecture/analyze-source/plugin-runtime/project-analyzer-assignments/`
- Issue file: `docs/agents/issues/done/20260829-046-session-cache-and-selective-invalidation.md`
- Category: `feature`
- Execution type: `AFK`
- Review gate: `none`
- Suggested state: `done`

## Parent Artifacts

- `docs/prd.md`
- `docs/architecture/application-architecture-summary.md`
- `docs/architecture/analyze-source/plugin-runtime/project-analyzer-assignments/prd.md`
- `docs/architecture/analyze-source/plugin-runtime/project-analyzer-assignments/canonical-domain-model.md`
- `docs/architecture/analyze-source/plugin-runtime/project-analyzer-assignments/canonical-use-cases.md`
- `docs/architecture/analyze-source/plugin-runtime/project-analyzer-assignments/canonical-api-cli-contract.md`
- `docs/architecture/analyze-source/plugin-runtime/project-analyzer-assignments/acceptance-scenarios.md`
- `docs/architecture/analyze-source/plugin-runtime/project-analyzer-assignments/readiness-review.md`
- `docs/architecture/analyze-source/plugin-runtime/multi-analyzer-orchestration/canonical-domain-model.md`
- `docs/architecture/analyze-source/plugin-runtime/multi-analyzer-orchestration/canonical-api-cli-contract.md`
- `docs/architecture/analyze-source/plugin-runtime/multi-analyzer-orchestration/acceptance-scenarios.md`
- `docs/architecture/analyze-source/plugin-runtime/multi-analyzer-orchestration/orchestration-status.md`

## What to build

Add an immutable, session-scoped cache around the existing combined job and
aggregate pipeline. A cache entry must be reusable only when its complete
canonical job inputs still match: repository and invocation roots, discovery
policy, assignment/selection source, logical analyzer identity and runtime
provenance, effective options, source-scope policy, matched source set, and
source-content identity. The cache is an in-memory session facility; the PRD's
persistent cross-session cache remains out of scope.

On reanalysis, compare the new plan against the prior session and rerun only
affected jobs where the effective inputs changed. Retain stable scope IDs,
failed/partial diagnostics, and immutable results for unaffected scopes. Report
cache hits and invalidation reasons through the existing run/scope response so
the viewer can continue using cached `All` and individual projections.

## Acceptance criteria

- [x] Cache entries are immutable session snapshots and are never reused when
  any input-complete identity component changes, including analyzer/version/
  API/package/runtime, assignment or selection source, effective options,
  discovery policy, source-scope policy, matched source set, or source content.
- [x] A reanalysis with unchanged job inputs reuses the prior result without
  invoking that analyzer again, while a changed assignment, option, filter, or
  source set invalidates the affected job.
- [x] Changes are selective: unrelated project/analyzer scopes remain reusable
  when their complete inputs and matched source sets are unchanged; each
  invalidation records a stable reason suitable for CLI/HTTP diagnostics.
- [x] Stable scope IDs are preserved for the same relative project root and
  logical analyzer, while job/cache IDs distinguish changed effective inputs
  and cannot substitute stale evidence.
- [x] Aggregate runs retain successful, partial, failed, and unavailable scope
  diagnostics independently of cache hits; no usable result is never replaced
  with a fabricated model.
- [x] Reanalysis responses expose cache-hit information and invalidation
  reasons alongside scope status, source-scope fingerprints, and aggregate
  results without changing the existing projection schema semantics.
- [x] Selecting `All` or an individual scope continues to use cached
  projections only; selection never invokes an analyzer or mutates the cache.
- [x] If the previously active scope is absent from the new run, the active
  selection resets to `All` while the new aggregate diagnostics remain visible.
- [x] Tests use invocation-counting analyzer fixtures to prove unchanged-scope
  reuse, affected-only reruns for assignment/filter/options/source changes,
  failed-scope retention, stable IDs, cache metadata, and stale-scope recovery.

## Artifact sync required

- Application PRD: `none` — cache scope and invalidation behavior are already
  defined by the approved product contract and this issue adds no new user
  workflow or retention promise.
- Application architecture summary: `none` — the session-cache boundary,
  plugin-manager ownership, and viewer projection boundary are already
  specified; persistent caching remains explicitly out of scope.
- Owning capability node/artifacts: `required: /.okf/capabilities/analyze-source/plugin-runtime/project-analyzer-assignments.md` and `docs/architecture/analyze-source/plugin-runtime/project-analyzer-assignments/orchestration-status.md`; coordinate evidence with the multi-analyzer orchestration status without changing its owner.
- Issue registry: `required`; node `issues:` reference: `required when .okf exists`.
- Reason/no-impact decision: delivery truth changes and the existing job
  fingerprint groundwork becomes an executable session-cache contract. Product
  topology and architecture ownership remain unchanged.

## Human review gate

None. This slice has no new rendered UI/UX and is independently verifiable by
cache, reanalysis, aggregate, and projection tests.

## Blocked by

Unblocked by and delivered after
`docs/agents/issues/done/20260829-045-resolve-configured-assignments-and-source-scopes.md`.

## Artifact anchors

- `PAA-FR-009`, `PAA-FR-010`, and `PAA-FR-011`.
- The existing `AnalyzerJob` fingerprints, `AnalysisRun` scope maps,
  `SelectAnalysisScope`, and `/v1/reanalysis` callback boundary.
- `SC-PAA-007`, `SC-PAA-008`, `SC-PAA-009`, and `SC-PAA-014`.
- The non-goal excluding persistent cross-session result caching.

## User stories addressed

The capability PRD has no numbered user-story section. This slice supports
application journeys 10 and 11 by making repeated configured analysis reuse
safe cached scope results and rerun only affected work.

## Verification obligations

- Policy source: `/.okf/project.md`

| Scenario | Backend boundary | Frontend integration | End-to-end journey |
| --- | --- | --- | --- |
| `SC-PAA-007` | `when-supported` | `not-applicable` | `when-supported` |
| `SC-PAA-008` | `when-supported` | `not-applicable` | `when-supported` |
| `SC-PAA-009` | `when-supported` | `not-applicable` | `when-supported` |
| `SC-PAA-014` | `when-supported` | `not-applicable` | `when-supported` |

The existing viewer selector is already implemented; issue 047 verifies the
configured run and visible recovery states through its review gate.

## Implementation and verification

Implemented the immutable session-scoped cache around planned analyzer jobs
and aggregate projections. Complete job inputs now determine cache identity;
unchanged jobs are reused, affected jobs receive stable invalidation reasons,
and cached results are deep-cloned before use. Scope summaries and reanalysis
responses expose cache-hit and invalidation metadata while selection remains
projection-only.

Verification passed:

- `go test ./internal/analysis/orchestration ./cmd/arch-view -count=1`
- `go test ./... -count=1`
- `go test -race ./... -count=1`
- `go vet ./...`
- `go build ./...`
- `git diff --check`

## Scenario traceability

| Source rule | Scenario | Issue criterion | Verification evidence |
| --- | --- | --- | --- |
| PAA-FR-009 | `SC-PAA-008`/`SC-PAA-014` | complete input keys and affected-only invalidation | invocation-count, fingerprint, and reanalysis tests |
| PAA-FR-010/011 | `SC-PAA-007`/`SC-PAA-009` | cached projection and active-scope recovery | aggregate projection and reanalysis tests |

## Handoff

Issue 047 now completes the configured mixed-repository viewer journey and
retains its required visual review gate.
