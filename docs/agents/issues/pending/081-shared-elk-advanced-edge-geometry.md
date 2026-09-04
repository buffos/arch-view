# Shared ELK advanced edge geometry

## Issue Metadata

- Issue number: `081`
- Owning capability node: `/.okf/capabilities/explore-architecture/advanced-elk-renderer-support.md`
- Artifact root: `docs/architecture/explore-architecture/advanced-elk-renderer-support`
- Issue file: `docs/agents/issues/pending/081-shared-elk-advanced-edge-geometry.md`
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

- [x] Existing count labels use ELK bounds without invented counts.
- [x] Validated shared junctions and connected cubic sections render through shared geometry.
- [x] Geometry v1 source identity supports architecture and OKF; invalid output falls back safely.
- [x] Actual downloaded SVG and embedded/static export behavior verified.
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

Resolved: issue 080 is archived at
`docs/agents/issues/done/20260904-080-shared-elk-settings-and-feature-registry.md`
with explicit human visual approval.

## Artifact anchors

- PRD AER-FR-001 through AER-FR-009 as applicable.
- Domain LayoutFeatureProfile, GeometrySnapshot, policies and lifecycle.
- Use cases ListSupportedRendererFeatures, ValidateFeatureProfile, BuildFeatureAwareLayoutInput, NormalizeELKGeometry.
- Contract existing layout/profile endpoints and browser/export behavior.
- Delivery contract AER-R-001 through AER-R-005; this stage owns only its tranche.

## Acceptance scenarios addressed

- SC-AER-002
- SC-AER-003
- SC-AER-006
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
| SC-AER-002 | `layout/features_test.go` | `advanced_edge_features_test.js` | live architecture and OKF review |
| SC-AER-003 | pinned ELK fixture | `advanced_edge_features_test.js` | live junction rendering and browser SVG serialization |
| SC-AER-006 | pinned ELK fixture | `advanced_edge_features_test.js`, `elk_spline_integration_test.js` | live architecture spline review |
| SC-AER-007 | static export provenance test | malformed-output fallback fixture | last-valid scene and diagnostic review |
| SC-AER-009 | `export/layout_features_test.go` | not-applicable | static SVG limitation reported in provenance |
| SC-AER-010 | catalog/profile validation | no-invented-count fixture and OKF graph tests | live selected-only arrowless semantic-link review |
| SC-AER-011 | feature catalog consistency | registry and shared presentation tests | both viewers use the same settings and geometry modules |

Run full Go tests, race, vet/build, browser tests/syntax, strict OKF validation,
diff and line audits. Record focused test evidence for every addressed rule,
not only a broad green command.

## Evidence

See `docs/agents/reviews/20260904-advanced-elk-stage-2.md`. Automated checks
and the implementer visual pass are complete. The issue remains open solely
for the required independent human visual approval.
