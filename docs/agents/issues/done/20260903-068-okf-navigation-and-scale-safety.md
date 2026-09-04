# 068 — Add bounded depth, focus, Back, truncation, and scale safety

## Issue Metadata

- Issue number: `068`
- Owning capability node: `/.okf/capabilities/okf-knowledge-views.md`
- Related capability nodes: `/.okf/capabilities/explore-architecture.md`
- Artifact root: `docs/architecture/okf-knowledge-views/`
- Issue file: `docs/agents/issues/done/20260903-068-okf-navigation-and-scale-safety.md`
- Category: `feature`
- Execution type: `AFK`
- Review gate: `visual-review`
- Suggested state: `done`

## Parent PRD

`docs/architecture/okf-knowledge-views/prd.md`

## Parent artifacts

- `docs/prd.md`
- `docs/architecture/application-architecture-summary.md`
- `docs/architecture/okf-knowledge-views/prd.md`
- `docs/architecture/okf-knowledge-views/canonical-domain-model.md`
- `docs/architecture/okf-knowledge-views/canonical-use-cases.md`
- `docs/architecture/okf-knowledge-views/canonical-api-cli-contract.md`
- `docs/architecture/okf-knowledge-views/acceptance-scenarios.md`
- `docs/architecture/okf-knowledge-views/readiness-review.md`
- `/.okf/project.md`

## What to build

Add session-owned navigation over the profiled OKF projection. Support the
default depth of 2, valid integer depths of at least 1, an explicit full view,
subtree focus, breadcrumbs, and Back. Enforce profile and application node and
relationship limits, report truncation and hidden counts visibly, preserve
deterministic path order, and provide focus recovery when a requested target is
not visible.

Make projection requests cancellable and supersedable so an older navigation
request cannot replace the active result. Keep semantic links from expanding
containment traversal, retain the last valid scene when a new bounded request
fails, and expose navigation/projection status through the canonical session
queries.

## Acceptance criteria

- [x] Default navigation depth is 2; integer depth values below 1 are rejected,
  and full projection is explicit rather than an accidental unlimited default.
- [x] Subtree focus, breadcrumbs, and Back update one session deterministically
  and restore the prior focus/depth state where specified.
- [x] Depth traversal follows containment only; semantic links do not expand the
  navigable subtree.
- [x] Node and relationship budgets enforce the recommended defaults and hard
  caps before unbounded work occurs.
- [x] Truncation, hidden counts, and recovery affordances are visible and do
  not masquerade as a complete projection.
- [x] Focus on a hidden or removed target recovers according to the canonical
  focus policy and emits a diagnostic when needed.
- [x] In-flight projection requests support cancellation/supersession and stale
  results cannot replace the active scene.
- [x] Visual review covers depth controls, focus/back breadcrumbs, truncation,
  status messaging, keyboard focus, and responsive layout.

## Artifact sync required

- Application PRD: `none` — navigation, limits, truncation, and freshness rules
  are already specified.
- Application architecture summary: `none` — navigation remains session-owned
  and consumes the shared renderer-neutral scene.
- Owning capability node/artifacts: `required: /.okf/capabilities/okf-knowledge-views.md`;
  no semantic exact-spec edit is expected.
- Issue registry: `required`; the owning node `issues:` reference is required.
- OKF index/log: `required` at implementation and visual-review
  synchronization; capability state remains `specified`.
- Reason/no-impact decision: delivery and visual-review progress change;
  product scope, topology, and source ownership remain unchanged.

## Human review gate

Required `visual-review`. Review depth controls, focus affordances,
breadcrumbs/Back, truncation and hidden-count explanations, cancellation/status
feedback, keyboard navigation, accessible labels, and desktop/responsive
density. Record the rendered result and resolve material findings before
closure.

## Blocked by

None. The dependency batch 064–071 is complete; original dependency order is
preserved in the canonical delivery plan.

## Specification anchors

- `FR-14` through `FR-16` and the navigation/scale rules in `prd.md`.
- `SetNavigationDepth`, `FocusSubtree`, `NavigateBack`, and
  `GetNavigationState` in `canonical-use-cases.md`.
- `DepthLimit`, scale-safety, cancellation, and supersession rules in
  `canonical-domain-model.md` and `canonical-api-cli-contract.md`.
- `SC-009`, `SC-010`, `SC-011`, and `SC-019`.

## User stories addressed

The capability PRD has no numbered user-story section. This slice supports the
Viewer user in `WF-03` by making large OKF views navigable, bounded, recoverable,
and explicit about incomplete projections.

## Verification obligations

- Policy source: `/.okf/project.md`

| Scenario | Backend boundary | Frontend integration | End-to-end journey |
| --- | --- | --- | --- |
| `SC-009` | `when-supported` | `when-supported` | `when-supported` |
| `SC-010` | `when-supported` | `when-supported` | `when-supported` |
| `SC-011` | `when-supported` | `when-supported` | `when-supported` |
| `SC-019` | `when-supported` | `when-supported` | `when-supported` |

## Automated verification

- Focused navigation, budget, truncation, cancellation, supersession, session,
  HTTP, and browser tests.
- `go test ./... -count=1`
- `go test -race ./...`
- `go vet ./...`
- `go build ./...`
- `node --check` for every changed viewer JavaScript module.
- `node --test` for all viewer JavaScript tests.
- `git diff --check`

## Delivery synchronization (2026-09-03)

Automated implementation and verification are complete. Agent inspection covered
depth controls, focus/back breadcrumbs, truncation messaging, cancellation
recovery, graph containment, and responsive layout; the required human
visual-review gate remains pending. Do not close or archive this issue until
that review is explicitly approved.

## Scenario traceability

| Source rule | Scenario | Issue criterion | Verification evidence |
| --- | --- | --- | --- |
| At-most depth and explicit full view | `SC-009` | Depth limits are validated and traversal is containment-bounded | navigation and projection tests |
| Budget/truncation/focus recovery | `SC-010` | Limits produce visible truncation and a usable focus recovery path | large-fixture tests and visual review |
| Focus and Back session history | `SC-011` | Subtree navigation and return preserve canonical session state | session/browser interaction tests |
| Freshness/cancellation | `SC-019` | Superseded or cancelled requests cannot replace the current scene | concurrent projection tests |

## Handoff

Issue 069 consumes bounded projections and navigation state for concept detail.
Issue 071 verifies stale-result and processing-failure behavior across the
complete viewer journey.

## Approved closeout (2026-09-04)

The user explicitly approved the completed result in this task: "Everything is
fine now. I approve. proceed to commits (one or more )". This closes the human
review gate where applicable. The chronological pending-review and evidence-gap
notes above are superseded by this closeout, not erased.

All acceptance criteria are satisfied by the scenario evidence audit and final
review in `docs/agents/reviews/20260904-okf-scenario-evidence.md` and
`docs/agents/reviews/20260904-okf-working-tree-review.md`, including their later
resolved findings. The shared CSS/SVG follow-up is recorded in
`docs/agents/reviews/20260904-shared-css-svg-followup.md`.

The final Go test, race, vet, build, JavaScript test/syntax, and diff checks pass.
Artifact synchronization includes the application PRD/architecture summary,
owning capability, issue registry, orchestration status, and OKF index/log.
Approved follow-ups include architecture-first mode, shared settings/rendering,
selection-only arrowless semantic links, uncapped Fit, and current-canvas SVG
download. No public OKF CLI, headless OKF export API, or source-editing surface
is introduced.
