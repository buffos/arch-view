# 070 — Persist OKF bindings and project-local profile lifecycle

## Issue Metadata

- Issue number: `070`
- Owning capability node: `/.okf/capabilities/okf-knowledge-views.md`
- Related capability nodes: `/.okf/capabilities/explore-architecture.md`
- Artifact root: `docs/architecture/okf-knowledge-views/`
- Issue file: `docs/agents/issues/done/20260903-070-okf-profile-persistence-and-bindings.md`
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

Add project-local OKF profile editing and durable bundle/profile bindings. Load
and persist the optional `okf` section in the nearest `.archview.json` while
preserving unrelated layout, analysis, and unknown configuration data. Support
profile validation, Save, Save As without flattening composition, built-in
immutability, rename, delete with explicit reassignment/fallback, and binding
updates for a selected bundle.

Use the canonical atomic-write, revision, conflict, and idempotency semantics.
Failed validation or persistence must leave the previous configuration intact;
source OKF bundles remain read-only. Reuse or extend the existing configuration
boundary without moving layout or analysis ownership into the OKF profile
service.

## Acceptance criteria

- [x] The optional `okf` configuration section can bind a bundle/profile without
  corrupting or flattening unrelated layout, analysis, or unknown fields.
- [x] Project-local profiles can be edited and validated in the viewer, with
  ordered bases and composition preserved on Save and Save As.
- [x] Save updates the active project-local profile atomically; Save As creates
  a distinct profile without duplicate creation on retry.
- [x] Built-in profiles are immutable and cannot be overwritten through the
  project-local editor.
- [x] Rename updates all affected bindings atomically; delete requires an
  explicit replacement or documented fallback and leaves no dangling binding.
- [x] Revision conflicts, invalid profiles, stale writes, and persistence
  failures are structured diagnostics and do not partially mutate the file.
- [x] Bind, Save, Save As, rename, and delete commands obey the canonical
  idempotency and retry semantics.
- [x] Visual review covers editor validation, Save/Save As affordances,
  immutable built-ins, rename/delete confirmation, fallback/conflict states,
  keyboard focus, and responsive layout.

## Artifact sync required

- Application PRD: `none` — profile lifecycle, optional configuration, and
  atomic persistence behavior are already specified.
- Application architecture summary: `none` — this preserves the existing
  configuration boundary and layout/analysis ownership.
- Owning capability node/artifacts: `required: /.okf/capabilities/okf-knowledge-views.md`;
  exact-spec documents remain authoritative.
- Issue registry: `required`; the owning node `issues:` reference is required.
- OKF index/log: `required` at implementation and visual-review
  synchronization; no state transition is claimed by this issue.
- Reason/no-impact decision: delivery and visual-review progress change;
  product scope, topology, source ownership, and existing configuration
  semantics remain unchanged.

## Human review gate

Required `visual-review`. Review profile editing and validation, Save/Save As,
immutable built-in handling, rename/delete and fallback messaging, conflict and
failure states, focus order, accessible labels, and desktop/responsive density.
Record the rendered result and resolve material findings before closure.

## Blocked by

None. The dependency batch 064–071 is complete; original dependency order is
preserved in the canonical delivery plan.

## Specification anchors

- `FR-19` through `FR-24` in `prd.md`.
- `BindProfileToBundle`, `SaveProjectProfile`, `SaveProjectProfileAs`,
  `RenameProjectProfile`, and `DeleteProjectProfile` in
  `canonical-use-cases.md`.
- `OKFViewConfiguration`, `ConfigurationWrite`, revisions, idempotency, and
  atomic persistence rules in `canonical-domain-model.md` and
  `canonical-api-cli-contract.md`.
- `SC-015` through `SC-018`.

## User stories addressed

The capability PRD has no numbered user-story section. This slice supports the
Profile maintainer and Viewer user in `WF-02`, `WF-05`, and `WF-06` by making
project-local profile changes explicit, durable, reversible on failure, and
safe for existing configuration.

## Verification obligations

- Policy source: `/.okf/project.md`

| Scenario | Backend boundary | Frontend integration | End-to-end journey |
| --- | --- | --- | --- |
| `SC-015` | `when-supported` | `when-supported` | `when-supported` |
| `SC-016` | `when-supported` | `when-supported` | `when-supported` |
| `SC-017` | `when-supported` | `when-supported` | `when-supported` |
| `SC-018` | `when-supported` | `when-supported` | `when-supported` |

## Automated verification

- Focused configuration, profile validation, binding, atomic write, revision,
  idempotency, HTTP, and browser tests.
- `go test ./... -count=1`
- `go test -race ./...`
- `go vet ./...`
- `go build ./...`
- `node --check` for every changed viewer JavaScript module.
- `node --test` for all viewer JavaScript tests.
- `git diff --check`

## Delivery synchronization (2026-09-03)

Automated implementation and verification are complete. Agent inspection covered
editor validation, immutable built-ins, Save/Save As, binding, rename/delete,
conflict messaging, focus, and responsive density; the required human
visual-review gate remains pending. Do not close or archive this issue until
that review is explicitly approved.

## Scenario traceability

| Source rule | Scenario | Issue criterion | Verification evidence |
| --- | --- | --- | --- |
| Save and Save As composition | `SC-015` | Active profile saves and variants preserve bases/rules without flattening | configuration round-trip and browser tests |
| Binding integrity | `SC-016` | Rename/delete updates bindings atomically with explicit replacement/fallback | lifecycle transaction tests |
| Non-destructive persistence | `SC-017` | Failed validation or write leaves the prior file and session binding intact | injected failure and filesystem assertions |
| Revision/idempotency | `SC-018` | Stale writes are rejected and retries do not duplicate profiles | concurrency/idempotency contract tests |

## Handoff

Issue 071 verifies the persistence and projection interactions under stale,
cancelled, invalid, and failed processing conditions. No public CLI or export
surface is added by this first delivery.

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
