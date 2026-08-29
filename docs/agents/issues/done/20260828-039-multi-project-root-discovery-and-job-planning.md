# 039 — Discover project roots and plan deterministic analyzer jobs

## Issue Metadata

- Issue number: `039`
- Owning capability node: `/.okf/capabilities/analyze-source/plugin-runtime/multi-analyzer-orchestration.md`
- Related capability node: `/.okf/capabilities/analyze-source/plugin-runtime/project-analyzer-assignments.md` (its resolved assignment input is consumed; its persistence and precedence implementation remain out of scope)
- Artifact root: `docs/architecture/analyze-source/plugin-runtime/multi-analyzer-orchestration/`
- Issue file: `docs/agents/issues/done/20260828-039-multi-project-root-discovery-and-job-planning.md`
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
- `docs/architecture/analyze-source/plugin-runtime/project-analyzer-assignments/canonical-api-cli-contract.md`
- `docs/architecture/analyze-source/plugin-runtime/implementation-slice.md`

## What to build

Implement the repository discovery and immutable job-planning path for a
combined analysis. Starting at the opened repository, traverse directories in
lexical order while enforcing the fixed exclusions, repository containment,
symlink policy, and the maximum planned-job budget. Use analyzer-declared
strong markers to build a nested ownership tree; carve nested strong roots out
of parent input, while weak markers only support detection inside an already
selected root. Evaluate applicable registered analyzers, accept resolved
assignment input and current explicit CLI selection without implementing the
assignment capability's `.archview.json` persistence, and produce one logical
analyzer job per root/language pair by default. Create stable scope/job
identities and a deterministically sorted `arch-view.job-plan/v1` plan with
discovery diagnostics before execution begins. Resolve the validated
invocation-root source-scope policy as part of the plan: global exclusion globs
apply to every job, an analyzer-ID include rule limits only that analyzer's
source set, filtering occurs after root discovery and nested ownership, and
fixed/nested/configured exclusions win.

## Acceptance criteria

- [x] Discovery never traverses or schedules paths outside the opened
  repository, fixed exclusions, build/cache/vendor trees, or symlink targets;
  normalized relative paths use `/` and the repository root is represented as
  `.`.
- [x] Strong analyzer manifest markers create candidate roots in deterministic
  lexical order; nested strong roots become independent scopes and are passed
  as exclusions to the owning parent; weak markers cannot create an automatic
  root by themselves.
- [x] Candidate evaluation supports different language analyzers at one root,
  avoids duplicate logical analyzer IDs for the same root/language pair, and
  preserves the resolved selection source (`assignment`, `cli`, or
  `automatic`). Persisted assignment-file parsing is not reimplemented here.
- [x] The planner emits `arch-view.job-plan/v1` with repository root,
  discovery-policy version, immutable jobs, nested-root exclusions, effective
  option fingerprints, and discovery diagnostics; jobs are sorted by normalized
  relative root, language, and logical analyzer ID.
- [x] Scope IDs are stable across runs for the same relative root and logical
  analyzer ID, and job IDs are unique within a run; the implementation follows
  the canonical SHA-256 scope-key and `scope_id::local_id` namespace contract.
- [x] The planner enforces the specified maximum of 128 planned jobs and
  returns a stable discovery-limit diagnostic instead of creating an unbounded
  work queue.
- [x] The plan carries the canonical invocation-root source-scope policy and
  each job receives its deterministic effective source set; configured source
  filters cannot hide markers or re-include fixed, nested-root, or excluded
  content.
- [x] Tests cover mixed-language roots, nested ownership, weak markers,
  exclusions, symlinks, outside-repository paths, duplicate candidates,
  explicit selection/assignment inputs, safe glob validation, include/exclude
  precedence, stable source sets, stable ordering, and repeated-plan
  determinism.

## Artifact sync required

- Application PRD: `required` — the validated project source-scope policy is
  part of the combined-analysis workflow; the synchronized contract is in
  `docs/prd.md`.
- Application architecture summary: `required` — the plugin manager owns
  invocation-root source-scope resolution/application; the synchronized
  boundary is in `docs/architecture/application-architecture-summary.md`.
- Owning capability node/artifacts: `required: /.okf/capabilities/analyze-source/plugin-runtime/multi-analyzer-orchestration.md` and `docs/architecture/analyze-source/plugin-runtime/multi-analyzer-orchestration/orchestration-status.md`; update the parent plugin-runtime delivery record if planning ownership changes.
- Issue registry: `required`; node `issues:` reference: `required when .okf exists`.
- Reason/impact decision: the source-scope contract changes implementation
  inputs and cache identity while preserving the existing product topology and
  canonical model ownership. Application product and architecture artifacts
  were refreshed with the child exact specification.

## Human review gate

None. This slice has no rendered UI/UX change; its plan is independently
verifiable through backend fixtures and serialized plan assertions.

## Blocked by

None — the registered analyzer manifests, detection contract, and explicit
selection path already exist.

## Artifact anchors

- `MAO-FR-001` through `MAO-FR-003`.
- `SourceScopePolicy`, invocation-root glob semantics, and post-discovery
  filtering rules from the project-assignment contract.
- `ProjectDiscoveryService`, `ResolveNestedProjectOwnership`,
  `EvaluateAnalyzerCandidates`, `AnalyzerJobPlanner`, and `PlanAnalyzerJobs`.
- `JobPlan`, `AnalyzerJob`, and `ScopeIdentity` fields and invariants.
- The job-plan ordering and selection-source rules in the canonical contract.

## Acceptance scenarios addressed

- `SC-MAO-001` — Discover mixed-language roots
- `SC-MAO-002` — Respect nested project ownership
- `SC-MAO-003` — Keep discovery inside the repository
- `SC-MAO-011` — Filter source input after root discovery
- `SC-MAO-012` — Apply analyzer-scoped include globs deterministically

## User stories addressed

The capability PRD has no numbered user-story section. This slice supports
application journey 9 and the project-discovery/job-planning services without
adding a new user-facing surface.

## Verification obligations

- Policy source: `/.okf/project.md`

| Scenario | Backend boundary | Frontend integration | End-to-end journey |
| --- | --- | --- | --- |
| `SC-MAO-001` | `implemented` | `not-applicable` | `deferred` |
| `SC-MAO-002` | `implemented` | `not-applicable` | `deferred` |
| `SC-MAO-003` | `implemented` | `not-applicable` | `deferred` |
| `SC-MAO-011` | `implemented` | `not-applicable` | `deferred` |
| `SC-MAO-012` | `implemented` | `not-applicable` | `deferred` |

The end-to-end obligations were deferred at this slice because public CLI and
viewer exposure were owned by downstream issues 042–043. Issue 042 is verified;
issue 043 remains at its required visual-review gate.

## Scenario traceability

| Source rule | Scenario | Issue criterion | Verification evidence |
| --- | --- | --- | --- |
| MAO-FR-001/002/003 | SC-MAO-001/002/003 | bounded lexical discovery, nested ownership, and repository containment | `internal/analysis/orchestration/orchestration_test.go` discovery, symlink, and repeated-plan tests |
| Source-scope policy and post-discovery filtering | SC-MAO-011/012 | effective source sets and include/exclude precedence | planner source-scope test, safe-glob test, fingerprint invalidation test |

## Implementation and verification

Implemented deterministic discovery, nested-root ownership, selection
precedence, bounded planning, source-scope resolution, and stable scope/job
identities. Application PRD and architecture truth required no further change:
the implementation follows the synchronized source-scope and plugin-manager
boundary. Verified with focused orchestration tests, `go test ./... -count=1`,
`go test -race ./... -count=1`, `go vet ./...`, `go build ./...`,
`staticcheck ./...`, JavaScript syntax checks, strict OKF validation, and
`git diff --check`.

## Handoff

After this issue passes, issue 040 may execute its immutable plans through the
bounded scheduler. The assignment capability may later supply persisted
resolved assignments and the validated source-scope policy through the input
seam defined here.
