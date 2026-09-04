# 065 — Render a neutral OKF projection through the existing viewer

## Issue Metadata

- Issue number: `065`
- Owning capability node: `/.okf/capabilities/okf-knowledge-views.md`
- Related capability nodes: `/.okf/capabilities/explore-architecture.md`
- Artifact root: `docs/architecture/okf-knowledge-views/`
- Issue file: `docs/agents/issues/done/20260903-065-okf-neutral-projection-viewer.md`
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

Create the first renderer-neutral OKF projection and connect it to the
existing local viewer/ELK path. Provide an immutable neutral built-in profile,
project the selected indexed bundle into the canonical projection snapshot,
and expose the selected profile, source/profile revisions, nodes, relationships,
counts, legend, and diagnostics through the `/v1/okf` session and projection
boundary.

Add the smallest user-visible entry point needed to choose the OKF view and
load the selected bundle in the shared viewer. Keep the OKF scene separate from
the architecture-model scene, preserve the existing ELK layout/rendering
integration, and leave all existing architecture-view behavior unchanged.
Do not add local profile persistence or advanced relationship/profile rules in
this slice.

## Acceptance criteria

- [x] Every valid selected bundle offers an immutable neutral built-in profile.
- [x] Selecting the neutral profile produces a renderer-neutral projection
  snapshot with source/profile revisions, visible nodes, relationships, counts,
  legend, and diagnostics.
- [x] The OKF viewer renders the projection through the existing ELK-backed
  viewer without converting the bundle into the architecture model.
- [x] The OKF entry point clearly distinguishes the selected bundle and OKF
  view from existing architecture-model views.
- [x] The initial neutral projection is read-only and does not merge bundles or
  write source documents.
- [x] Missing or invalid selected-bundle data produces the canonical diagnostic
  state rather than a misleading empty success.
- [x] Existing architecture-view routes, scenes, layout behavior, and browser
  tests remain compatible.
- [x] A visual review covers the normal desktop and supported responsive view,
  including graph readability, selection state, empty/error states, and focus.

## Artifact sync required

- Application PRD: `none` — the neutral OKF journey and shared-viewer boundary
  are already specified.
- Application architecture summary: `none` — the existing ELK viewer remains
  a consumer of a renderer-neutral OKF scene; architecture-model ownership is
  unchanged.
- Owning capability node/artifacts: `required: /.okf/capabilities/okf-knowledge-views.md`;
  no semantic specification edit is expected.
- Issue registry: `required`; the owning node `issues:` reference is required.
- OKF index/log: `required` at implementation and visual-review
  synchronization; no capability-state transition is claimed by this issue.
- Reason/no-impact decision: delivery and visual-review progress change;
  product scope, topology, source ownership, and cross-capability boundaries
  remain unchanged.

## Human review gate

Required `visual-review`. Review the OKF entry point, selected-bundle/profile
state, neutral graph, empty/error states, keyboard focus, accessible labels,
desktop density, and supported responsive layout. Record the rendered result and
resolve material layout, contrast, density, or accessibility findings before
closure.

## Blocked by

None. The dependency batch 064–071 is complete; original dependency order is
preserved in the canonical delivery plan.

## Specification anchors

- `FR-07` and `FR-13` in `prd.md`.
- `SelectProfile` and `GetCurrentProjection` in `canonical-use-cases.md`.
- The renderer-neutral `ProjectionSnapshot` and existing-viewer integration
  rules in `canonical-domain-model.md` and `canonical-api-cli-contract.md`.
- `SC-004`.

## User stories addressed

The capability PRD has no numbered user-story section. This slice supports the
Viewer user in `WF-01` and `WF-02` by making a selected OKF bundle visible in a
read-only neutral graph through the existing viewer.

## Verification obligations

- Policy source: `/.okf/project.md`

| Scenario | Backend boundary | Frontend integration | End-to-end journey |
| --- | --- | --- | --- |
| `SC-004` | `when-supported` | `when-supported` | `when-supported` |

## Automated verification

- Focused projection, session, HTTP, renderer-neutral scene, and viewer tests.
- `go test ./... -count=1`
- `go test -race ./...`
- `go vet ./...`
- `go build ./...`
- `node --check` for every changed viewer JavaScript module.
- `node --test` for all viewer JavaScript tests.
- `git diff --check`

## Delivery synchronization (2026-09-03)

Automated implementation and verification are complete. Desktop and 375px
responsive agent inspection covered the OKF entry point, graph, selection,
empty/error states, focus/back navigation, and accessibility labels; the
required human visual-review gate remains pending. Do not close or archive this
issue until that review is explicitly approved.

## Scenario traceability

| Source rule | Scenario | Issue criterion | Verification evidence |
| --- | --- | --- | --- |
| Neutral built-in profile and projection | `SC-004` | A valid bundle loads as a neutral graph with explicit source/profile identity | projection contract and browser fixture tests |
| Shared ELK viewer boundary | `SC-004` | The renderer-neutral scene is laid out/rendered without architecture-model conversion | viewer integration test and visual review |
| Read-only source consumption | `SC-004` | Loading and changing the view does not mutate the bundle | source snapshot and filesystem assertions |

## Handoff

Issue 066 consumes the neutral projection to normalize explicit containment,
filesystem fallback, and semantic-link layers. Issues 067–070 extend the same
projection/session boundary with profiles, navigation, inspection, and
persistence.

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
