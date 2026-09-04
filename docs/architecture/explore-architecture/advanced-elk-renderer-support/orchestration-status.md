# Advanced ELK renderer support orchestration status

## State

- Planning state: `specified`.
- Parent: the implemented [Explore and inspect architecture](../../../../.okf/capabilities/explore-architecture.md) capability.
- Current route: the staged candidate workstreams are exact-specified and the application synthesis gate is current and passed; this is the next eligible delivery node for issue slicing.
- Delivery issues: none yet. Creating issues before the supported renderer subset and acceptance behavior are decided would create speculative work.

## Boundary

This workstream extends presentation behavior only. Canonical model facts,
language analyzers, and project configuration schemas remain outside its
ownership.

## Exact-spec inventory

- Requirements gap analysis
- Domain glossary
- Capability PRD
- Canonical domain model
- Canonical use-case model
- Canonical API/CLI contract
- Acceptance scenarios
- Architecture readiness review

The supported subset is `edge_labels`, `junctions`, `ports`, `compound`, and
`spline_refinement`, with `arch-view.geometry/v1`, explicit fallback, browser/
HTML/browser-SVG parity, and deterministic static-SVG behavior.

## Readiness decision

The child is READY FOR ARCHITECTURE IMPLEMENTATION. It can move to `implemented`
only after every resulting issue and visual review is complete.

## Bounded decisions

- Route/output extensions are the first tranche; structural scene extensions
  follow; broader catalog options remain gated by concrete renderer support.
- Browser, embedded HTML, and browser SVG share the advanced renderer contract.
- Go static SVG keeps its deterministic orthogonal contract unless a later
  exact specification explicitly extends it.
- Invalid or unsupported layout output uses deterministic fallback with a
  visible diagnostic, and semantic accessibility remains available through the
  list/details surface.

## Artifact impact

- Capability truth: updated with the complete exact-spec set.
- Product truth: opt-in presentation features and fallback diagnostics are synchronized.
- Architecture truth: scene/route/layout and cross-surface boundaries are current.
- Delivery truth: intentionally unchanged; no issue slicing in this run.
