# Shared ELK settings and feature registry

## Issue Metadata

- Issue number: `080`
- Owning capability node: `/.okf/capabilities/explore-architecture/advanced-elk-renderer-support.md`
- Artifact root: `docs/architecture/explore-architecture/advanced-elk-renderer-support`
- Issue file: `docs/agents/issues/pending/080-shared-elk-settings-and-feature-registry.md`
- Category: `feature`
- Execution type: `AFK`
- Review gate: `visual-review`
- Suggested state: `awaiting-human-review`

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

- [x] Shared catalog metadata, deterministic handler ordering and conflict rejection.
- [x] Feature persistence, inheritance, explicit clearing, compatibility and unavailable-feature fallback.
- [x] Usable-first shared settings with full-catalog filter and distinct availability reasons.
- [x] Pinned-runtime verified Mr. Tree controls, Layered component spacing and four numeric padding controls.
- [x] Existing defaults, OKF selected-only arrowless semantics and navigation remain unchanged.
- [x] Automated verification and scenario evidence complete; implementation files below 600 lines.
- [x] Required artifact synchronization complete.
- [ ] Explicit human visual approval recorded before next stage or closeout.

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

None - can start immediately

## Artifact anchors

- PRD AER-FR-001 through AER-FR-009 as applicable.
- Domain LayoutFeatureProfile, GeometrySnapshot, policies and lifecycle.
- Use cases ListSupportedRendererFeatures, ValidateFeatureProfile, BuildFeatureAwareLayoutInput, NormalizeELKGeometry.
- Contract existing layout/profile endpoints and browser/export behavior.
- Delivery contract AER-R-001 through AER-R-005; this stage owns only its tranche.

## Acceptance scenarios addressed

- SC-AER-001
- SC-AER-008
- SC-AER-011

## Verification obligations

Policy source: `/.okf/project.md`, when-supported on all three surfaces.
Canonical-rule mapping is in delivery-contract.md; later feature scenarios
remain explicitly owned by their later blocked issues.

| Scenario | Backend boundary | Frontend integration | End-to-end journey |
|---|---|---|---|
| SC-AER-001 | implemented: layout/features_test.go; OKF layout_features_test.go | implemented: layout_form_test.js; existing OKF/viewport suites | implemented: live fixture Apply/focus/Back and Save/reload |
| SC-AER-008 | implemented: layout_features_http_test.go; layout/runtime_test.go | implemented: layout_form_test.js; real pinned runtime fixture | implemented: both dialogs, compatibility and responsive checks |
| SC-AER-011 | implemented: layout/features_test.go | implemented: layout_features_test.js; shared runtime tests | implemented: both scene adapters use the shared pipeline; visual approval pending |

Evidence: docs/agents/reviews/20260904-advanced-elk-stage-1.md.
Pinned Mr. Tree NONE/BFS values fail the actual runtime fixtures and remain
unavailable with reasons. CONSTRAINT weighting awaits per-node constraints.
Features owned by stages 2–4 remain unavailable, not partially implemented.

Run full Go tests, race, vet/build, browser tests/syntax, strict OKF validation,
diff and line audits. Record focused test evidence for every addressed rule,
not only a broad green command.
