# Advanced ELK renderer support future-work register

Status: initial contracted slice `implemented` — issues 080–083 are verified,
visually approved, and archived. Broader candidates remain uncommitted.

This register is the durable place for renderer candidates beyond the
implemented five-feature slice. Broader renderer work must receive a new
bounded contract before issue slicing.

## Candidate workstreams

| ID | Candidate | Why it is separate | Expected boundary |
|---|---|---|---|
| ELK-FW-001 | Ports and port labels | Requires scene nodes/edges to expose port identity, side, and label geometry | Scene contract, ELK adapter, browser/SVG renderers |
| ELK-FW-002 | Edge labels and label-aware routing | Requires route/label placement rules and hit-testing that are not needed by the current edge labels | Routing representation and serializers |
| ELK-FW-003 | Junction points | Requires a first-class junction representation and renderer semantics | Routing representation, SVG/browser renderers |
| ELK-FW-004 | Compound graph geometry | Requires nested layout bounds and cross-hierarchy route rules | Scene projection, layout adapter, navigation/renderers |
| ELK-FW-005 | Broader target-specific ELK options | Requires each option to have a supported target, validation rule, and visual acceptance case | Layout registry and target-aware adapter |
| ELK-FW-006 | Spline-specific refinement | Covers control-point tuning, label placement, and fallback quality beyond the current cubic path | Routing and renderer serializers |

## Entry criteria for issue slicing

Before any `ELK-FW-*` item becomes an implementation issue, implementation
planning must confirm:

1. Which ELK options/geometry are supported by the pinned runtime?
2. What changes, if any, are required in the renderer-neutral scene or route
   contract?
3. Which browser, full-canvas, and export surfaces must render the feature?
4. What remains catalog-only or explicitly unsupported?
5. What deterministic fallback, accessibility behavior, and visual acceptance
   evidence are required?

The delivered PRD, contract/scenario updates, readiness review, and issues
080–083 are linked from the implemented child capability. Candidate work above
is not part of that completed delivery unless explicitly covered there.

## Bounded priority

The recommended order is route/output extensions first (edge labels, junction
points, and spline refinement), structural scene extensions second
(ports/port labels and compound geometry), and broader target-specific options
last. Every selected feature must remain within the pinned ELK runtime,
renderer-neutral scene/route contract, and explicit browser/HTML/SVG fallback
and accessibility behavior.
