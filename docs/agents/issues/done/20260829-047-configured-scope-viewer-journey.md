# 047 — Complete the configured mixed-repository viewer journey

## Issue Metadata

- Issue number: `047`
- Owning capability node: `/.okf/capabilities/analyze-source/plugin-runtime/project-analyzer-assignments.md`
- Related consumer node: `/.okf/capabilities/explore-architecture.md`
- Artifact root: `docs/architecture/analyze-source/plugin-runtime/project-analyzer-assignments/`
- Issue file: `docs/agents/issues/done/20260829-047-configured-scope-viewer-journey.md`
- Category: `feature`
- Execution type: `AFK`
- Review gate: `visual-review`
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
- `docs/architecture/analyze-source/plugin-runtime/multi-analyzer-orchestration/canonical-api-cli-contract.md`
- `docs/architecture/analyze-source/plugin-runtime/multi-analyzer-orchestration/acceptance-scenarios.md`
- `docs/architecture/explore-architecture/canonical-api-cli-contract.md`
- `docs/architecture/explore-architecture/acceptance-scenarios.md`

## What to build

Finish the public configured-analysis journey through the existing local
viewer. Drive a mixed-repository fixture with a v2 `.archview.json` through
project-backed `open` and the existing aggregate/scope APIs. Ensure configured
assignment scopes, automatic scopes, unavailable-assignment diagnostics,
partial status, effective source-scope identity, cache metadata, and invalid
nearest-configuration errors reach the already implemented scope selector and
status/details surfaces.

Keep scope selection projection-only: switching between `All` and an
individual configured scope must not run an analyzer. Preserve the existing
layout, navigation, evidence, source inspection, export, reanalysis, and
keyboard-accessible viewer behavior. Add or adjust only the integration needed
to make the configured contract visible; do not rebuild the selector already
delivered by issue 043.

## Acceptance criteria

- [x] Project-backed `open` on a mixed fixture with v2 assignments displays
  `All` plus deterministic configured and automatic project/analyzer scopes,
  with stable labels, IDs, language/analyzer identity, counts, and statuses.
- [x] A valid assignment and an unavailable assigned analyzer appear as
  independent scope outcomes: usable scopes render normally and the failed
  scope remains selectable for its diagnostic without fabricating a model.
- [x] Global exclusions and analyzer-scoped includes are reflected in the
  displayed scope summaries/details, including effective source-scope identity;
  no excluded source is reintroduced by projection or export.
- [x] Invalid nearest configuration is surfaced through the existing error
  surface and does not silently analyze with a farther configuration or stale
  assignment set.
- [x] Switching `All` and individual scopes changes only the cached projection;
  analyzer invocation counts remain unchanged and the active scope remains
  deterministic across refreshes.
- [x] Reanalysis reports cache hits/invalidation reasons through the existing
  status path, and removal of the active scope returns the viewer to `All`
  while retaining aggregate diagnostics.
- [x] Layout settings remain independent from analysis assignments and filters;
  existing graph navigation, evidence/source inspection, exports, keyboard
  access, and v1 layout-only sessions continue to work.
- [x] Automated server/browser tests cover configured scope/status/error
  routing where harnesses are available, and a mixed-language visual fixture is
  prepared for the declared visual-review gate.

## Artifact sync required

- Application PRD: `none` — this completes the already synchronized configured
  scope journey and adds no new product behavior beyond the approved scenarios.
- Application architecture summary: `none` — the viewer remains a projection
  consumer and the plugin manager remains the owner of analyzer semantics,
  filters, and cache policy.
- Owning capability node/artifacts: `required: /.okf/capabilities/analyze-source/plugin-runtime/project-analyzer-assignments.md` and `docs/architecture/analyze-source/plugin-runtime/project-analyzer-assignments/orchestration-status.md`; synchronize the viewer-consumer evidence only if implementation changes that boundary.
- Issue registry: `required`; node `issues:` reference: `required when .okf exists`.
- Reason/no-impact decision: delivery truth changes and the configured
  workflow becomes reviewable. Product topology and exact-spec semantics remain
  unchanged; the visual gate verifies integration of the existing viewer
  contract.

## Human review gate

Visual review is required after automated verification. Review the configured
mixed-language fixture for scope labels and language identity, `All` versus
individual projections, unavailable/partial diagnostics, invalid-configuration
states, cache/reanalysis behavior, keyboard accessibility, and preservation of
existing navigation and details behavior.

User approval: visual review approved on 2026-08-29. The mixed-language
fixture, configured scope labels and identities, partial/unavailable states,
projection-only selection, reanalysis metadata, keyboard access, and existing
navigation/details behavior were accepted.

## Blocked by

Unblocked by and delivered after
`docs/agents/issues/done/20260829-046-session-cache-and-selective-invalidation.md`.

## Artifact anchors

- `PAA-FR-010`, `PAA-FR-011`, and `PAA-FR-012`.
- The completed issue 043 selector/projection path and the existing aggregate
  `/scopes`, `/projection`, and reanalysis contracts.
- `SC-PAA-006`, `SC-PAA-007`, `SC-PAA-009`, and `SC-PAA-010`.
- The root `when-supported` verification policy and required visual-review
  gate for visible behavior.

## User stories addressed

The capability PRD has no numbered user-story section. This slice completes
application journeys 9, 10, and 11 for configured mixed-repository opening,
scope selection, diagnostics, and source-filter feedback.

## Verification obligations

- Policy source: `/.okf/project.md`

| Scenario | Backend boundary | Frontend integration | End-to-end journey |
| --- | --- | --- | --- |
| `SC-PAA-006` | `when-supported` | `when-supported` | `when-supported` |
| `SC-PAA-007` | `when-supported` | `when-supported` | `when-supported` |
| `SC-PAA-009` | `when-supported` | `when-supported` | `when-supported` |
| `SC-PAA-010` | `when-supported` | `when-supported` | `when-supported` |

Where no browser harness exists, the automated frontend obligation is recorded
as deferred with evidence and the declared visual-review gate remains required.

## Implementation and automated verification

Implemented the configured mixed-language fixture and connected its v2
configuration to the existing project-backed analysis, scope selector, status,
details, and reanalysis surfaces. Scope labels now expose analyzer identity;
unavailable assignments remain selectable diagnostics; source-scope and cache
metadata are visible; and the projection path does not invoke analyzers.

Automated verification passed:

- `go test ./cmd/arch-view ./internal/analysis/orchestration -count=1`
- `go test ./... -count=1`
- `go test -race ./... -count=1`
- `go vet ./...`
- `go build ./...`
- JavaScript syntax checks and all available viewer-module tests
- `git diff --check`

The reproducible manual review fixture is
`docs/architecture/analyze-source/plugin-runtime/project-analyzer-assignments/fixtures/configured-mixed/`.
No browser harness is present, so browser automation remains deferred under
the root `when-supported` policy. The required manual visual review was
performed and approved by the user.

## Scenario traceability

| Source rule | Scenario | Issue criterion | Verification evidence |
| --- | --- | --- | --- |
| PAA-FR-006/010 | `SC-PAA-006` | configured and unavailable scopes remain independently visible | aggregate/viewer integration tests and mixed-fixture review |
| PAA-FR-011 | `SC-PAA-007`/`SC-PAA-009` | cached scope projection and reset-to-All behavior | callback-count/browser interaction tests |
| PAA-FR-012 | `SC-PAA-010` | layout and analysis remain separate in the visible workflow | v1/v2 layout/configuration regression and visual review |

## Handoff

Automated verification and the declared visual review are complete. The owning
capability node is now eligible for `implemented` state.
