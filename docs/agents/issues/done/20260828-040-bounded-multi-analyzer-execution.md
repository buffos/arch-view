# 040 — Execute bounded multi-analyzer jobs with lifecycle control

## Issue Metadata

- Issue number: `040`
- Owning capability node: `/.okf/capabilities/analyze-source/plugin-runtime/multi-analyzer-orchestration.md`
- Artifact root: `docs/architecture/analyze-source/plugin-runtime/multi-analyzer-orchestration/`
- Issue file: `docs/agents/issues/done/20260828-040-bounded-multi-analyzer-execution.md`
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
- `docs/architecture/analyze-source/plugin-runtime/implementation-slice.md`

## What to build

Implement execution of an immutable analyzer job plan through a bounded worker
pool. Run independent jobs concurrently with a default of four workers and a
hard cap of sixteen, pass each job its root, nested exclusions, selection,
effective options, and resolved effective source scope, and retain runtime
provenance. Model the complete job
lifecycle and publish deterministic run/job events. Cancellation must stop
queued work, terminate active external processes through the existing process
boundary, prevent late results from being accepted, and leave every job in a
terminal state. Timeouts, analyzer failures, invalid results, and diagnostics
remain attached to their owning job without erasing independent work.

## Acceptance criteria

- [x] The scheduler consumes only an immutable `arch-view.job-plan/v1` plan,
  starts no more than four jobs by default, never permits configuration to
  exceed the hard cap of sixteen, and preserves deterministic job snapshots
  sorted by scope ID.
- [x] Each job receives its normalized project root, nested-root exclusions,
  analyzer selection, runtime provenance, resolved options, and the effective
  source scope produced from the invocation-root policy; one analyzer job maps
  to one isolated analyzer operation through the existing in-process or
  process-backed contract.
- [x] Job lifecycle transitions follow `planned -> queued -> running ->
  complete|partial|failed|cancelled|skipped`; terminal jobs cannot transition
  again, and the aggregate run does not become terminal until queued/running
  jobs have reached terminal states.
- [x] Cancellation stops queued jobs, terminates active external processes,
  rejects late output/results, and records stable cancellation diagnostics
  without changing already completed independent jobs.
- [x] Timeouts, process failures, analyzer failures, invalid results, and
  analyzer diagnostics are retained per job with the canonical failure code or
  diagnostic metadata; no automatic retry is introduced.
- [x] A job never receives a path removed by a fixed, nested-root, configured,
  or analyzer-specific exclusion, and a change to the effective source set is
  visible in the job/cache identity used by downstream aggregation.
- [x] Lifecycle output includes `run.started`, `job.planned`, `job.started`,
  `job.completed`, `job.failed`, `job.cancelled`, and `run.completed`, with
  monotonic sequence numbers and completed/total counts; analyzer-internal
  percentage remains optional.
- [x] Tests use blocking and failing analyzer fixtures plus the existing
  process adapter to prove concurrency limits, independent success,
  cancellation, timeout cleanup, terminal-state rules, deterministic events,
  and no leaked or late process results.

## Artifact sync required

- Application PRD: `required` — job execution now carries the validated
  source-scope behavior described in the synchronized application workflow.
- Application architecture summary: `required` — the scheduler must preserve
  the plugin-manager source-scope boundary and effective-source-set identity.
- Owning capability node/artifacts: `required: /.okf/capabilities/analyze-source/plugin-runtime/multi-analyzer-orchestration.md` and `docs/architecture/analyze-source/plugin-runtime/multi-analyzer-orchestration/orchestration-status.md`; retain scheduler evidence in the parent implementation record.
- Issue registry: `required`; node `issues:` reference: `required when .okf exists`.
- Reason/impact decision: delivery truth changes and the scheduler now consumes
  the approved source-scope contract; application product topology and model
  ownership remain unchanged.

## Human review gate

None. This slice has no rendered UI/UX change.

## Blocked by

Blocked by `docs/agents/issues/done/20260828-039-multi-project-root-discovery-and-job-planning.md`.

## Artifact anchors

- `MAO-FR-004`, `MAO-FR-005`, and `MAO-FR-009`.
- `SourceScopePolicy` and the effective source-scope field on `AnalyzerJob`.
- `AnalyzerJobScheduler`, `ExecuteAnalyzerPlan`, `CancelAnalyzerPlan`, and
  `PublishJobLifecycle`.
- `AnalyzerJob` lifecycle, resource-budget, cancellation, and event invariants.
- The existing process-backed analyzer lifecycle and child cleanup contract.

## Acceptance scenarios addressed

- `SC-MAO-004` — Run jobs with bounded concurrency
- `SC-MAO-008` — Cancel queued and active work

## User stories addressed

The capability PRD has no numbered user-story section. This slice supports
application journeys 9 and 10's execution foundation and the scheduler use
cases without adding a rendered surface.

## Verification obligations

- Policy source: `/.okf/project.md`

| Scenario | Backend boundary | Frontend integration | End-to-end journey |
| --- | --- | --- | --- |
| `SC-MAO-004` | `implemented` | `not-applicable` | `deferred` |
| `SC-MAO-008` | `implemented` | `not-applicable` | `deferred` |

The end-to-end obligations were deferred at this slice because public CLI and
viewer exposure were owned by downstream issues 042–043. Issue 042 is verified;
issue 043 remains at its required visual-review gate.

## Scenario traceability

| Source rule | Scenario | Issue criterion | Verification evidence |
| --- | --- | --- | --- |
| MAO-FR-004/005/009 | SC-MAO-004 | bounded worker pool, immutable job inputs, and lifecycle events | scheduler concurrency, failure-retention, filtering, and deterministic-event tests |
| Cancellation and process-boundary rules | SC-MAO-008 | queued/active cancellation and terminal-state enforcement | scheduler cancellation test plus existing process-analyzer cancellation/timeout suite |

## Implementation and verification

Implemented the bounded scheduler, immutable job snapshots, per-job source
scope forwarding/filtering, lifecycle events, cancellation propagation, and
failure/late-result protection. Application PRD and architecture truth required
no further change because the scheduler preserves the synchronized plugin
manager/process boundary. Verified with focused orchestration and process
adapter tests, `go test ./... -count=1`, `go test -race ./... -count=1`,
`go vet ./...`, `go build ./...`, `staticcheck ./...`, JavaScript syntax checks,
strict OKF validation, and `git diff --check`.

## Handoff

After this issue passes, issue 041 may consume all terminal job results to build
the collision-safe aggregate model and status.
