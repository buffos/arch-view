# Shared ELK presentation ports

## Issue Metadata

- Issue number: `082`
- Owning capability node: `/.okf/capabilities/explore-architecture/advanced-elk-renderer-support.md`
- Artifact root: `docs/architecture/explore-architecture/advanced-elk-renderer-support`
- Issue file: `docs/agents/issues/done/20260905-082-shared-elk-presentation-ports.md`
- Category: `feature`
- Execution type: `AFK`
- Review gate: `visual-review`
- Suggested state: `done`

## Parent Artifacts

- `docs/architecture/explore-architecture/advanced-elk-renderer-support/prd.md`
- `docs/architecture/explore-architecture/advanced-elk-renderer-support/canonical-domain-model.md`
- `docs/architecture/explore-architecture/advanced-elk-renderer-support/canonical-use-cases.md`
- `docs/architecture/explore-architecture/advanced-elk-renderer-support/canonical-api-cli-contract.md`
- `docs/architecture/explore-architecture/advanced-elk-renderer-support/acceptance-scenarios.md`
- `docs/architecture/explore-architecture/advanced-elk-renderer-support/readiness-review.md`
- `docs/architecture/explore-architecture/advanced-elk-renderer-support/delivery-contract.md`

## What to build

Implement the approved stage through the existing shared viewer settings and
renderer pipeline. Architecture and OKF save separately but reuse the same
controls and geometry. No feature-specific switches in scene adapters.

## Acceptance criteria

- [x] Deterministic input/output presentation ports honor positions, sides, labels and endpoints.
- [x] Shared selection, dragging, hit-testing and export preserve semantic identity.
- [x] Layered port options are gated by verified prerequisites.
- [x] Existing defaults, OKF selected-only arrowless semantics and navigation remain unchanged.
- [x] Automated verification and scenario evidence complete; implementation files below 600 lines.
- [x] Required artifact synchronization complete.
- [x] Explicit human visual approval recorded before next stage or closeout.

## Artifact sync required

- Application PRD: required, `docs/prd.md`.
- Application architecture summary: required, `docs/architecture/application-architecture-summary.md`.
- Owning capability artifacts: required, `docs/architecture/explore-architecture/advanced-elk-renderer-support` and `.okf/capabilities/explore-architecture/advanced-elk-renderer-support.md`.
- Issue registry and owning node issues references: required.
- Impact: product and architecture presentation/configuration extension; no topology, analyzer, or canonical-model changes.

## Human review gate

Inspect this stage in both viewers, including responsive settings, normal/full
canvas, Fit, zoom, pan, manual moves, selection and focus/Back. Review generated
geometry and Download SVG when applicable. Automated checks do not waive review.

## Blocked by

Resolved: issue 081 is archived at
`docs/agents/issues/done/20260905-081-shared-elk-advanced-edge-geometry.md`
with explicit human visual approval.

## Artifact anchors

- PRD AER-FR-001 through AER-FR-009 as applicable.
- Domain LayoutFeatureProfile, GeometrySnapshot, policies and lifecycle.
- Use cases ListSupportedRendererFeatures, ValidateFeatureProfile, BuildFeatureAwareLayoutInput, NormalizeELKGeometry.
- Contract existing layout/profile endpoints and browser/export behavior.
- Delivery contract AER-R-001 through AER-R-005; this stage owns only its tranche.

## Acceptance scenarios addressed

- SC-AER-004
- SC-AER-007
- SC-AER-009
- SC-AER-010
- SC-AER-011

## Verification obligations

Policy source: `/.okf/project.md`, when-supported on all three surfaces.
Canonical-rule mapping is in delivery-contract.md; later feature scenarios
remain explicitly owned by their later blocked issues.

| Scenario | Backend boundary | Frontend integration | End-to-end journey |
|---|---|---|---|
| SC-AER-004 | verified | verified | verified |
| SC-AER-007 | verified | verified | verified |
| SC-AER-009 | verified | not-applicable | verified |
| SC-AER-010 | verified | verified | verified |
| SC-AER-011 | verified | verified | verified |

Run full Go tests, race, vet/build, browser tests/syntax, strict OKF validation,
diff and line audits. Record focused test evidence for every addressed rule,
not only a broad green command.

## Evidence

- `docs/agents/reviews/20260905-advanced-elk-stage-3.md`
- The user explicitly approved the Stage 3 visual-review gate on 2026-09-05;
  issue 083 may proceed.
