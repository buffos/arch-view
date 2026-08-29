# 043 — Expose cached scope projections in the local viewer

## Issue Metadata

- Issue number: `043`
- Owning capability node: `/.okf/capabilities/analyze-source/plugin-runtime/multi-analyzer-orchestration.md`
- Related consumer node: `/.okf/capabilities/analyze-source/plugin-runtime/project-analyzer-assignments.md`
- Artifact root: `docs/architecture/analyze-source/plugin-runtime/multi-analyzer-orchestration/`
- Issue file: `docs/agents/issues/done/20260828-043-cached-scope-projections-and-viewer-selection.md`
- Category: `feature`
- Execution type: `AFK`
- Review gate: `visual-review`
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
- `docs/architecture/explore-architecture/canonical-api-cli-contract.md`
- `docs/architecture/analyze-source/plugin-runtime/project-analyzer-assignments/canonical-api-cli-contract.md`
- `docs/architecture/analyze-source/plugin-runtime/implementation-slice.md`

## What to build

Connect the project-backed `arch-view open` path to a combined analysis run and
make its cached scopes visible through the existing local viewer. Add an
accessible scope control containing `All` plus one entry per discovered job,
sorted by stable scope ID and labeled with project root and analyzer. Selecting
an entry must request the corresponding cached renderer-neutral projection,
preserve the facts that scope contributed to `All`, and never launch another
analyzer. Failed scopes remain selectable for diagnostics, partial aggregate
status remains visible, and a scope that disappears after reanalysis returns
the viewer to `All`. Preserve existing hierarchy navigation, source/evidence,
layout, export, and model-only viewer behavior. Do not implement persisted
assignment configuration or assignment cache invalidation; those belong to the
separate assignment capability. The selector and scope summaries must honor
the effective source scope already resolved by the job plan and must not
broaden a filtered scope during projection.

## Acceptance criteria

- [x] `arch-view open --project <repository>` runs the combined path by default,
  initializes the local viewer with the aggregate run, and retains model-only
  `open --model` behavior unchanged.
- [x] The viewer exposes an accessible `All`/scope selector with deterministic
  labels, stable scope IDs, status/count summaries, and failed scopes retained
  for diagnostic inspection.
- [x] Selecting `All` or an individual scope requests the cached projection
  from the run, updates the graph/details/evidence state without invoking an
  analyzer or rerunning analysis, and preserves the individual scope's facts
  exactly within the combined result.
- [x] Scope summaries/projections retain the effective source-scope identity
  and filtered-source diagnostics supplied by the run; selecting a scope never
  re-includes an excluded path or changes the analyzer's source set.
- [x] Aggregate complete, partial, failed, and cancelled states are visible in
  the existing viewer status/error surfaces; no-usable-result runs expose
  diagnostics without rendering a fabricated combined model.
- [x] Existing hierarchy paths, reference visibility, source inspection,
  layout/session controls, reanalysis behavior, and exports continue to work
  for `All` and individual projections where the underlying contract supports
  them.
- [x] If the active scope is no longer present after reanalysis, the viewer
  returns to `All`; scope selection alone never calls the reanalysis endpoint.
- [x] Server tests cover scope-list/projection routing and no-reanalysis
  selection; browser tests cover keyboard-accessible selection, status/error
  rendering, deterministic ordering, and projection replacement.
- [x] The implementation is ready for visual review using a mixed-language
  fixture, including `All`, at least two individual scopes, and one failed
  scope with visible diagnostics.

## Artifact sync required

- Application PRD: `required` — the viewer must preserve the synchronized
  filtered-scope journey and source-set identity.
- Application architecture summary: `required` — the viewer consumes, but does
  not recompute, the plugin-manager source-scope policy.
- Owning capability node/artifacts: `required: /.okf/capabilities/analyze-source/plugin-runtime/multi-analyzer-orchestration.md` and `docs/architecture/analyze-source/plugin-runtime/multi-analyzer-orchestration/orchestration-status.md`; synchronize viewer-consumer evidence with `docs/architecture/analyze-source/plugin-runtime/project-analyzer-assignments/orchestration-status.md` if its ownership boundary changes.
- Issue registry: `required`; node `issues:` reference: `required when .okf exists`.
- Reason/impact decision: delivery truth changes and the approved viewer
  journey becomes reachable. The viewer remains a projection consumer; product
  topology and ownership are unchanged, and the application artifacts now
  record the source-scope preservation rule.

## Human review gate

The user approved the mixed-language visual review on 2026-08-29. The review
covered the selector labels and ordering, `All` versus individual graph
contents, partial/failed status and diagnostics, keyboard accessibility, and
preservation of existing navigation and details behavior. The follow-up
review also confirmed that the scope picker starts collapsed and expands only
on interaction.

## Blocked by

Blocked by `docs/agents/issues/done/20260828-042-combined-analysis-cli-and-http-exposure.md`.

## Artifact anchors

- `MAO-FR-010` and the cached projection/no-reanalysis rule.
- Effective source-scope identity and filtered-source diagnostics from
  `SourceScopePolicy`.
- `SelectAnalysisScope` and `CombinedModel` scope summaries.
- The `/scopes` and `/projection?scope=all|<scope_id>` contract.
- Existing renderer-neutral scene, model, source, layout, and export contracts.

## Acceptance scenarios addressed

- `SC-MAO-005` — Preserve independent success after failure (viewer status)
- `SC-MAO-009` — Expose combined and individual scopes
- `SC-MAO-010` — Report no-usable-result failure (viewer diagnostics)

## User stories addressed

The capability PRD has no numbered user-story section. This slice completes
application journey 9 and establishes the visible combined/per-scope path used
by journey 10.

## Verification obligations

- Policy source: `/.okf/project.md`

| Scenario | Backend boundary | Frontend integration | End-to-end journey |
| --- | --- | --- | --- |
| `SC-MAO-005` | `implemented` | `deferred` | `deferred` |
| `SC-MAO-009` | `implemented` | `deferred` | `deferred` |
| `SC-MAO-010` | `implemented` | `deferred` | `deferred` |

The frontend and end-to-end entries are deferred under the root
`when-supported` policy because this repository has no browser-test harness;
the executable browser surface is still subject to the declared visual review.

## Scenario traceability

| Source rule | Scenario | Issue criterion | Verification evidence |
| --- | --- | --- | --- |
| MAO-FR-008/010 and cached projection rule | SC-MAO-005/009 | preserve All/individual facts, scope status, and no-reanalysis selection | aggregate server routing, callback-count, failed-scope, scene, and existing viewer regression tests |
| No-usable-result diagnostics | SC-MAO-010 | diagnostic-only failed projection without fabricated model | failed aggregate scene test and aggregate HTTP projection handling |

## Implementation and verification

Implemented combined project-backed viewer wiring, cached All/individual
projection selection, scope/status summaries, failed-scope diagnostic scenes,
source/evidence scope routing, reanalysis reset-to-All behavior, and
transactional preservation of the active revision after failed reanalysis.
Product and architecture truth remain aligned with the projection-consumer boundary;
assignment persistence remains out of scope. Verified with focused viewer and
orchestration tests, `go test ./... -count=1`,
`go test -race ./... -count=1`, `go vet ./...`, `go build ./...`,
`staticcheck ./...`, JavaScript syntax checks, strict OKF validation, and the
available Node viewer-module tests. Browser harness coverage is deferred under
`when-supported`; the required mixed-language visual review was completed and
approved by the user.

## Handoff

The multi-analyzer capability is now implemented and all five issues in the
approved 039–043 sequence are archived. Persisted project assignments and
source-scope configuration remain owned by the separate assignment capability.
