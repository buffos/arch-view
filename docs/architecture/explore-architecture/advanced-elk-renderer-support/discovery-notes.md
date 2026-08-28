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

## Specified recommendations

The work is organized as one bounded renderer-extension program with staged
delivery rather than six simultaneous implementation tracks:

1. Route/output extensions: edge labels and label-aware placement, junction
   points, and spline-specific refinement.
2. Structural scene extensions: ports/port labels and compound-graph geometry.
3. Additional target-specific ELK options only when each option has an
   applicable pinned-runtime behavior, a renderer mapping, and an acceptance
   fixture.

The browser viewer, self-contained HTML, and browser Download SVG share the
renderer-neutral scene/route representation. Go static SVG remains its
deterministic orthogonal output in this specified frontier; adding advanced ELK
parity to it requires a separate exact decision. Unsupported or malformed ELK
output falls back deterministically and remains visible as a capability or
layout diagnostic. Accessibility is provided through the existing semantic
list/details path; presentation-only ports and junctions do not become fake
architecture relationships.

## Implementation and verification focus

The exact-spec set enumerates the first concrete ELK features and pinned-runtime
fixtures, defines the geometry fields for each, and sets visual/browser
support and deterministic fallback behavior. Implementation and verification
must now cover the five opt-in features, malformed/disconnected geometry,
browser/HTML/SVG parity, and semantic accessibility.
