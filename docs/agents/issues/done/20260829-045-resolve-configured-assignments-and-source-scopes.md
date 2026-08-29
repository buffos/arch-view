# 045 — Resolve configured assignments and source scopes

## Issue Metadata

- Issue number: `045`
- Owning capability node: `/.okf/capabilities/analyze-source/plugin-runtime/project-analyzer-assignments.md`
- Related consumer node: `/.okf/capabilities/analyze-source/plugin-runtime/multi-analyzer-orchestration.md`
- Artifact root: `docs/architecture/analyze-source/plugin-runtime/project-analyzer-assignments/`
- Issue file: `docs/agents/issues/done/20260829-045-resolve-configured-assignments-and-source-scopes.md`
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

Connect the validated v2 configuration from issue 044 to the existing
multi-analyzer planner and every public analysis entrypoint. Use one shared
resolution path for `arch-view analyze`, project-backed `open`, HTTP analysis,
and reanalysis so the nearest configuration and invocation-root filters cannot
drift between clients.

For each discovered project root, apply explicit CLI analyzer/language
selection first, then the deepest matching repository-relative assignment,
then automatic detection. In combined mode, plan all valid assigned scopes and
automatic roots that remain unassigned. Preserve unavailable assigned analyzers
as scoped diagnostics, apply the existing post-discovery source-scope seam, and
ensure fixed, nested-root, and configured exclusions always win over includes.
Keep assignment options, analyzer options, layout options, and command-line
exclusions under their existing ownership boundaries.

## Acceptance criteria

- [x] `analyze`, project-backed `open`, HTTP analysis, and reanalysis load the
  same nearest configuration and pass one normalized assignment/source-scope
  policy into the existing planner; layout-only v1 behavior remains unchanged.
- [x] Resolution order is deterministic: compatible CLI selection overrides a
  matching assignment, the deepest assignment path wins over ancestors, and
  automatic detection is used only when no explicit assignment or CLI choice
  applies. Conflicting CLI analyzer/language values produce the documented
  selection error.
- [x] Combined planning creates jobs for valid assigned roots and unassigned
  automatic roots, while an unavailable assigned analyzer remains a scoped
  diagnostic and is never silently replaced by another analyzer.
- [x] Assignment options follow the documented precedence and are forwarded to
  the selected job without becoming layout options or leaking sensitive values.
- [x] Global excludes apply to every job and analyzer-scoped includes apply only
  to their logical analyzer. Filtering occurs after root discovery and nested
  ownership, directory patterns match descendants, and fixed/nested/configured
  exclusions cannot be re-included.
- [x] Effective source scopes and their policy/matched-source fingerprints are
  carried into the existing immutable job plan and aggregate scope summaries.
- [x] Configuration, assignment, filter, and selection failures use stable
  documented error/diagnostic codes at CLI and HTTP boundaries; valid sibling
  scopes continue when one assignment is unavailable.
- [x] Tests cover deepest and nested assignments, CLI precedence/conflicts,
  automatic fallback, unavailable scopes, mixed-language planning, invocation
  root anchoring, include/exclude precedence, v1 compatibility, and parity
  across CLI, HTTP, `open`, and reanalysis entrypoints.

## Artifact sync required

- Application PRD: `none` — this is implementation of the already documented
  assignment, filter, and combined-view journeys; no product scope or rule is
  being added.
- Application architecture summary: `none` — the plugin-manager/planner and
  viewer-consumer boundaries are already synchronized; record any boundary
  change before closure rather than silently broadening this issue.
- Owning capability node/artifacts: `required: /.okf/capabilities/analyze-source/plugin-runtime/project-analyzer-assignments.md` and `docs/architecture/analyze-source/plugin-runtime/project-analyzer-assignments/orchestration-status.md`; the multi-analyzer consumer record may receive implementation evidence without changing its ownership.
- Issue registry: `required`; node `issues:` reference: `required when .okf exists`.
- Reason/no-impact decision: delivery truth changes when the planner receives
  durable configuration. The product topology, canonical model, and existing
  multi-analyzer ownership remain unchanged because this issue uses their
  approved input seam.

## Human review gate

None. This slice changes analysis planning and transport behavior but no
rendered UI; it is verified through planner, CLI, HTTP, and reanalysis tests.

## Blocked by

Unblocked by and delivered after
`docs/agents/issues/done/20260829-044-load-and-validate-analysis-configuration.md`.

## Artifact anchors

- `PAA-FR-003`, `PAA-FR-006`, and `PAA-FR-008`.
- The existing `AnalyzerAssignment`, `SourceScopePolicy`, nested ownership,
  `AnalyzerJobPlanner`, and `runCombinedAnalysisWithPolicy` seams.
- `SC-PAA-002`, `SC-PAA-003`, `SC-PAA-006`, `SC-PAA-011`, and `SC-PAA-012`.
- The completed 039–043 multi-analyzer plan, scheduler, aggregate, and
  transport contracts; no replacement viewer selector is introduced here.

## User stories addressed

The capability PRD has no numbered user-story section. This slice supports
application journeys 9, 10, and 11 by making configured assignments and source
filters drive the existing combined analysis path.

## Verification obligations

- Policy source: `/.okf/project.md`

| Scenario | Backend boundary | Frontend integration | End-to-end journey |
| --- | --- | --- | --- |
| `SC-PAA-002` | `when-supported` | `not-applicable` | `when-supported` |
| `SC-PAA-003` | `when-supported` | `not-applicable` | `when-supported` |
| `SC-PAA-006` | `when-supported` | `not-applicable` | `when-supported` |
| `SC-PAA-011` | `when-supported` | `not-applicable` | `when-supported` |
| `SC-PAA-012` | `when-supported` | `not-applicable` | `when-supported` |

The existing viewer consumes the resulting scopes; configured viewer states
and visible diagnostics are completed and reviewed in issue 047.

## Implementation and verification

Implemented the shared configured-analysis path for CLI analysis,
project-backed `open`, HTTP analysis, and reanalysis. The planner now applies
CLI selection, deepest assignment, and automatic detection in deterministic
order; retains unavailable assignments as scoped failures; and carries the
effective source scope and fingerprints into plans and summaries. The
configured mixed-language integration fixture covers sibling success,
unavailable assignment, source filtering, HTTP scope projection, and
reanalysis cache reuse.

Verification passed:

- `go test ./cmd/arch-view ./internal/analysis/orchestration -count=1`
- `go test ./... -count=1`
- `go vet ./...`
- `go build ./...`
- `git diff --check`

## Scenario traceability

| Source rule | Scenario | Issue criterion | Verification evidence |
| --- | --- | --- | --- |
| PAA-FR-003 | `SC-PAA-002`/`SC-PAA-003` | deepest assignment and CLI precedence | planner and CLI selection tests |
| PAA-FR-006 | `SC-PAA-006` | valid sibling jobs plus scoped unavailable-assignment diagnostic | mixed-plan and transport error tests |
| PAA-FR-007/008 | `SC-PAA-011`/`SC-PAA-012` | invocation-root filters and exclusion precedence | effective-source-set planner tests |

## Handoff

Issue 046 added session cache reuse and affected-job invalidation on top of the
configured job inputs.
