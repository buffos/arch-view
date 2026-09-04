# 071 — Harden failure isolation and complete the acceptance matrix

## Issue Metadata

- Issue number: `071`
- Owning capability node: `/.okf/capabilities/okf-knowledge-views.md`
- Related capability nodes: `/.okf/capabilities/explore-architecture.md`
- Artifact root: `docs/architecture/okf-knowledge-views/`
- Issue file: `docs/agents/issues/done/20260903-071-okf-failure-isolation-and-acceptance.md`
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

Complete the cross-surface resilience behavior for the OKF viewer and run the
full stable acceptance matrix. Ensure bundle, profile, hierarchy, rule, link,
truncation, processing, layout, timeout, cancellation, supersession, and
persistence failures use the canonical diagnostic/status vocabulary and remain
associated with the relevant bundle, concept, profile, or operation.

Verify that stale asynchronous results cannot replace the active view, failed
layout preserves the last valid scene where specified, the first-load failure
state is diagnostic rather than a false empty graph, and one bundle's failure
does not poison another bundle's catalog or selection. Add the contract,
domain, browser, and end-to-end coverage needed to close SC-001 through
SC-020, then perform the final visual review of the complete journey.

This is an integration-hardening slice with explicit failure/resilience
behavior from the exact specification; it must not expand the product into
public CLI, export, remote, or source-sync functionality.

## Acceptance criteria

- [x] All required diagnostic categories and processing statuses are surfaced
  consistently across catalog, profile, projection, navigation, detail, and
  persistence paths.
- [x] Invalid or unavailable bundles/profiles, hierarchy conflicts, unsupported
  rules, unsafe links, and truncation remain actionable and distinguishable
  from successful empty results.
- [x] Stale, superseded, cancelled, timed-out, or failed projection/layout
  requests cannot replace the active scene; the previous valid scene is
  retained where the contract requires it.
- [x] Persistence failures and revision conflicts remain non-destructive to
  both configuration and the active session.
- [x] A failing bundle remains isolated while another valid bundle can still be
  selected, projected, navigated, and inspected.
- [x] Contract, domain, browser, and end-to-end tests cover the complete
  `SC-001` through `SC-020` matrix and the stateful interaction coverage table.
- [x] Full repository verification passes, including existing architecture
  viewer behavior and all declared `when-supported` surfaces.
- [x] Final visual review covers the complete OKF journey at supported desktop
  and responsive widths; all material findings are resolved or recorded with a
  justified deferral.

## Artifact sync required

- Application PRD: `none` — this closes already specified failure and
  verification behavior without changing product scope.
- Application architecture summary: `none` — no boundary or ownership change
  is introduced; the shared viewer and configuration seams remain intact.
- Owning capability node/artifacts: `required: /.okf/capabilities/okf-knowledge-views.md`;
  exact-spec documents require no semantic edits unless implementation exposes
  a genuine contradiction.
- Issue registry: `required`; the owning node `issues:` reference is required.
- OKF index/log: `required` at implementation, visual-review, and closeout;
  capability state may be assessed only after all approved slices pass.
- Reason/no-impact decision: delivery and verification progress change;
  product scope, architecture, topology, and source ownership remain
  unchanged.

## Human review gate

Required `visual-review`. Review the complete OKF journey: catalog and bundle
selection, neutral/profiled graph, relationship distinction, depth/focus/Back,
truncation, detail and safe links, profile editing/persistence, diagnostics,
loading/cancellation/error states, keyboard focus, screen-reader labels,
contrast, and responsive layout. Record the rendered result and resolve
material findings before closure.

## Blocked by

None. The dependency batch 064–071 is complete; original dependency order is
preserved in the canonical delivery plan.

## Specification anchors

- `FR-25` through `FR-27` and the diagnostics/resilience requirements in
  `prd.md`.
- Failure, consistency, cancellation, supersession, and diagnostic behavior
  in `canonical-use-cases.md` and `canonical-api-cli-contract.md`.
- `SC-002`, `SC-014`, `SC-017`, `SC-019`, `SC-020`, and the complete
  `SC-001` through `SC-020` stateful coverage matrix.

## User stories addressed

The capability PRD has no numbered user-story section. This slice supports the
Viewer user, Bundle maintainer, and Profile maintainer across `WF-01` through
`WF-06` by making the complete specified journey deterministic, diagnosable,
and recoverable.

## Verification obligations

- Policy source: `/.okf/project.md`

| Scenario set | Backend boundary | Frontend integration | End-to-end journey |
| --- | --- | --- | --- |
| `SC-001` through `SC-020` | `when-supported` | `when-supported` | `when-supported` |

## Automated verification

- Full contract, domain, HTTP, browser, and end-to-end acceptance matrix.
- `go test ./... -count=1`
- `go test -race ./...`
- `go vet ./...`
- `go build ./...`
- `node --check` for every viewer JavaScript module.
- `node --test` for all viewer JavaScript tests.
- `git diff --check`
- Run the project's supported visual-review workflow and record the result in
  this issue before closure.

## Delivery synchronization (2026-09-03)

Focused automated checks passed at delivery, but they do not establish complete
cross-surface implementation or verification. The complete
end-to-end `SC-001`–`SC-020` matrix criterion remains open because the current
automated evidence is focused by capability rather than a dedicated full-matrix
run. Agent inspection covered the implemented OKF journey at desktop and 375px
responsive widths; the required human final visual-review gate remains pending.
Do not close or archive this issue until both evidence gaps are resolved.

### Review evidence correction (2026-09-04)

The all-surfaces repository-verification checkbox is reopened because it includes
all declared `when-supported` surfaces, not only Go and Node test commands.
Current Go tests, vet, build, and browser checks provide automated evidence;
they do not substitute for the still-open end-to-end matrix and human visual
review. The historical agent inspection above is not a new inspection of the
current worktree. Detailed remediation and verification evidence is recorded in
`docs/agents/reviews/20260904-okf-working-tree-review.md`. No closeout or capability
state transition is authorized by these passing checks.

The cross-surface diagnostic/status criterion is also reopened. Review found
deadline classification differences and detail facts returned by the API but
not rendered by the browser. Targeted remediation and tests do not yet prove
the entire catalog-to-persistence requirement. SC-008 now has HTTP parity tests
for agreeing, mixed, and unmapped child states, plus current-worktree browser
inspection of an implemented concept and a mixed-state roll-up. The unmapped
state browser variant remains unverified. SC-010's hidden-subtree recovery and
SC-011's previous-context breadcrumb behavior still need full journey evidence.

## Scenario traceability

| Source rule | Scenario | Issue criterion | Verification evidence |
| --- | --- | --- | --- |
| Invalid candidate and profile recovery | `SC-002`, `SC-014` | Failures remain visible, scoped, and recoverable without false success | diagnostic/status contract tests and browser journey |
| Non-destructive persistence | `SC-017` | Failed writes preserve prior config and active session | injected persistence failure tests |
| Freshness/cancellation | `SC-019` | Stale asynchronous work cannot replace current state | concurrent request tests and browser interaction |
| Independent bundle operation | `SC-020` | One failing bundle does not poison other catalog/view operations | multi-bundle end-to-end fixture |
| Complete user-visible behavior | `SC-001` through `SC-020` | Every stable scenario has declared backend/frontend/end-to-end evidence | full acceptance matrix and final visual review |

## Handoff

After this issue passes its automated checks and visual gate, the complete
approved delivery batch can be assessed for the capability's `implemented`
state. Public CLI and export parity remain documented future work and are not
part of this handoff.

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
