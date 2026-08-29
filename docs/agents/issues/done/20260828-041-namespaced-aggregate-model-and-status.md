# 041 — Aggregate namespaced multi-scope results and statuses

## Issue Metadata

- Issue number: `041`
- Owning capability node: `/.okf/capabilities/analyze-source/plugin-runtime/multi-analyzer-orchestration.md`
- Artifact root: `docs/architecture/analyze-source/plugin-runtime/multi-analyzer-orchestration/`
- Issue file: `docs/agents/issues/done/20260828-041-namespaced-aggregate-model-and-status.md`
- Category: `feature`
- Execution type: `AFK`
- Review gate: `none`
- Suggested state: `done`

## Parent Artifacts

- `docs/prd.md`
- `docs/architecture/application-architecture-summary.md`
- `docs/architecture/analyze-source/plugin-runtime/multi-analyzer-orchestration/prd.md`
- `docs/architecture/analyze-source/plugin-runtime/multi-analyzer-orchestration/canonical-domain-model.md`
- `docs/architecture/analyze-source/plugin-runtime/multi-analyzer-orchestration/canonical-use-cases.md`
- `docs/architecture/analyze-source/plugin-runtime/multi-analyzer-orchestration/canonical-api-cli-contract.md`
- `docs/architecture/analyze-source/plugin-runtime/multi-analyzer-orchestration/acceptance-scenarios.md`
- `docs/architecture/analyze-source/plugin-runtime/multi-analyzer-orchestration/readiness-review.md`
- `docs/architecture/generate-models/canonical-api-cli-contract.md`
- `docs/architecture/generate-models/canonical-domain-model.md`
- `docs/architecture/analyze-source/plugin-runtime/implementation-slice.md`

## What to build

Implement the aggregate boundary that turns terminal job results into one
`arch-view.aggregate/v1` response and one `arch-view.aggregate-model/v1`
language-neutral model. Namespace every module, reference, source reference,
relationship, and scope-specific diagnostic with the stable scope identity
before canonical normalization. Preserve per-scope analyzer/version/runtime
provenance, counts, and diagnostics; derive cycles and layers only from the
reported namespaced graph; and never infer relationships across scopes. Apply
the exact complete/partial/failed/cancelled rules while retaining usable
successful scopes when another job fails. Preserve each scope's effective
source-scope fingerprint and matched-source identity so filtered and
unfiltered results cannot collide in aggregate caches.

## Acceptance criteria

- [x] An aggregate contains `run_id`, repository metadata, sorted scope
  summaries, job diagnostics, aggregate diagnostics, status, and an optional
  combined model using the exact `arch-view.aggregate/v1` contract; the model
  reports `language: mixed` and the existing single-scope model contract
  remains unchanged.
- [x] Scope summaries and cache records retain the effective source-scope
  policy/matched-source fingerprint used by the job, and changing that
  fingerprint cannot reuse an aggregate built from a different source set.
- [x] Local observation IDs are rewritten as
  `scope_id::percent-encoded-local-id` for modules, references, source
  references, relationships, and scope-specific diagnostics; collisions across
  roots or analyzers remain distinct and relationship endpoints stay within
  their source scope.
- [x] Aggregate normalization sorts and validates all collections, preserves
  analyzer/project/runtime provenance and recoverable diagnostics, and derives
  graph projections only after namespacing through the canonical model boundary.
- [x] Status rules are exact: `complete` requires every planned job to be
  usable and non-partial; `partial` requires at least one usable scope and at
  least one non-complete scope; `failed` means no usable scope plus a
  non-cancellation failure or no applicable analyzer; `cancelled` means no
  usable scope remained when cancellation stopped the run.
- [x] A failed, partial, or cancelled job cannot remove or mutate another
  scope's observations; no matching names, paths, languages, or imports create
  a cross-scope relationship.
- [x] No combined model is exposed when no usable scope exists, while all
  scope diagnostics remain available for the later transport/viewer layers.
- [x] Tests cover colliding local IDs, provenance, partial success, failed and
  cancelled no-result runs, invalid job results, normalization conflicts,
  deterministic aggregate bytes, no-inference behavior, and parity between a
  single scope's facts and its contribution to `All`.

## Artifact sync required

- Application PRD: `required` — aggregate/cache behavior now preserves the
  configured source scope described in the synchronized product contract.
- Application architecture summary: `required` — aggregate cache identity must
  retain the effective source-set boundary without moving model ownership.
- Owning capability node/artifacts: `required: /.okf/capabilities/analyze-source/plugin-runtime/multi-analyzer-orchestration.md` and `docs/architecture/analyze-source/plugin-runtime/multi-analyzer-orchestration/orchestration-status.md`; synchronize any canonical-model implementation evidence in the parent delivery record.
- Issue registry: `required`; node `issues:` reference: `required when .okf exists`.
- Reason/impact decision: delivery truth changes and source-set identity is now
  an aggregate/cache invariant; product topology and canonical model ownership
  remain unchanged.

## Human review gate

None. This slice has no rendered UI/UX change.

## Blocked by

Blocked by `docs/agents/issues/done/20260828-040-bounded-multi-analyzer-execution.md`.

## Artifact anchors

- `MAO-FR-006` through `MAO-FR-008`.
- `SourceScopePolicy`, effective source-set identity, and cache invalidation
  rules.
- `AnalysisRun`, `CombinedModel`, `ScopeIdentity`, and aggregate status
  invariants.
- `AnalysisAggregationService`, `CollectUsableScopeResults`,
  `AggregateScopeResults`, and `DeriveCombinedGraph`.
- The `arch-view.aggregate/v1` response, namespaced identity, and parity rules.

## Acceptance scenarios addressed

- `SC-MAO-005` — Preserve independent success after failure
- `SC-MAO-006` — Namespace colliding observations
- `SC-MAO-007` — Do not infer cross-scope relationships
- `SC-MAO-010` — Report no-usable-result failure
- `SC-MAO-013` — Include source scope in cache identity

## User stories addressed

The capability PRD has no numbered user-story section. This slice supports
application journey 9 and the combined-analysis/model contract before public
CLI and viewer exposure.

## Verification obligations

- Policy source: `/.okf/project.md`

| Scenario | Backend boundary | Frontend integration | End-to-end journey |
| --- | --- | --- | --- |
| `SC-MAO-005` | `implemented` | `not-applicable` | `deferred` |
| `SC-MAO-006` | `implemented` | `not-applicable` | `deferred` |
| `SC-MAO-007` | `implemented` | `not-applicable` | `deferred` |
| `SC-MAO-010` | `implemented` | `not-applicable` | `deferred` |
| `SC-MAO-013` | `implemented` | `not-applicable` | `deferred` |

The end-to-end obligations were deferred at this slice because public CLI and
viewer exposure were owned by downstream issues 042–043. Issue 042 is verified;
issue 043 remains at its required visual-review gate.

## Scenario traceability

| Source rule | Scenario | Issue criterion | Verification evidence |
| --- | --- | --- | --- |
| MAO-FR-006/007/008 | SC-MAO-005/006/007 | isolated namespaced merge, provenance, and no cross-scope inference | aggregate namespace/parity and partial-success tests |
| Aggregate status and cache identity rules | SC-MAO-010/013 | no-usable-result behavior and source-set identity | status/cancellation and source-fingerprint tests |

## Implementation and verification

Implemented the versioned aggregate response, collision-safe namespacing,
per-scope provenance/diagnostics, canonical post-merge normalization, cached
scope selection, and exact status rules. Product and architecture artifacts
remain aligned with the approved language-neutral aggregate boundary; no
additional product/topology change was required. Verified with focused
orchestration/viewer tests, `go test ./... -count=1`,
`go test -race ./... -count=1`, `go vet ./...`, `go build ./...`,
`staticcheck ./...`, JavaScript syntax checks, strict OKF validation, and
`git diff --check`.

## Handoff

After this issue passes, issue 042 may expose aggregate runs to CLI and HTTP
consumers, and issue 043 may connect the same cached run to the local viewer.
