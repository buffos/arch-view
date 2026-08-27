# Advanced ELK renderer support discovery notes

## Why this is a separate workstream

The current Explore capability is implemented for its approved v1 scope. The
viewer already exposes the pinned ELK catalog and correctly treats unsupported
scene features as catalog-only. That is a deliberate renderer boundary, not a
promise that every ELK option is already supported.

Keeping this work in a child capability makes every future renderer extension
findable without reopening the completed v1 viewer scope or implying that a
catalog entry is an implemented behavior.

## Observed candidate areas

- Port geometry, port sides, and port labels.
- Edge labels and label-aware placement.
- Junction points and junction-aware route serialization.
- Compound/cross-hierarchy graph geometry.
- Additional node/edge-targeted ELK options after the supported priority
  tranche.
- Renderer capability negotiation and focused fixtures for each supported
  combination.
- Spline-specific tuning and label placement beyond the implemented general
  cubic route path.

These are candidates, not committed requirements. Their exact scope must be
validated against the pinned ELK runtime and the renderer-neutral scene
contract before issue slicing.
