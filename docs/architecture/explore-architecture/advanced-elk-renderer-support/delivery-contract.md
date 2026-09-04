# Approved staged delivery contract

Approved implementation plan, 2026-09-04. This amendment supersedes earlier
stage ordering, architecture-only identity, and server-generated geometry
examples in this reference set. It does not claim implementation completion.

## AER-R-001: shared registry and execution

Extend the layout option registry with data-defined feature metadata and
focused JavaScript handlers for edge_labels, junctions, spline_refinement,
ports, and compound. One declarative metadata source feeds Go validation,
the HTTP catalog, and embedded clients. Handler IDs must match that catalog.
Execute negotiate, prepare, layout, normalize/validate, and render phases.
Order dependencies first, then declared order and stable ID. Reject duplicate
registrations, dependency cycles, unknown dependencies, and geometry ownership
conflicts. No last-writer precedence or scene-specific feature dispatch.
Handlers own validation and deterministic affected-portion fallback.

## AER-R-002: persistence and compatibility

Architecture stores layout.features in top-level layout. OKF stores it in
the selected project profile's layout. Legacy architecture omission means off;
OKF omission inherits and an explicit empty array clears inherited features.
Apply changes only the session draft; Save/Save As persists choices. Navigation
preserves effective drafts. Built-ins are immutable. Preserve unknown project
fields, revision checks, idempotency, and atomic writes. Retain existing v1/v2
read support; advanced-feature writes use the established v2 document, including
an empty analysis object where required by the existing v2 contract.

Algorithm changes preserve feature preferences. Apply/Save reject unknown
features and incompatible combinations. A previously saved, known feature
unavailable in the running build is retained as a preference, diagnosed, and
excluded from effective rendering, never silently removed from disk.

## AER-R-003: usable settings first

Both viewers reuse the same settings dialog and controls. Default filtering
shows usable options. The complete catalog remains available through a
secondary filter with distinct algorithm-inapplicable, feature-required, and
not-implemented explanations. Preserve current supported options. Admit
Mr. Tree routing mode, search order, node weighting, Layered component spacing,
and four-sided numeric graph padding only for values verified with the pinned
runtime. Catalog presence alone is not proof of support. Known-broken runtime
values remain unavailable with a reason; do not upgrade ELK.

## AER-R-004: geometry and feature boundaries

arch-view.geometry/v1 uses source identity with kind architecture or okf,
source ID, source revision, and navigation scope. Architecture model identity
is not invented for OKF. Preserve one-to-one visible relationship identity and
deterministic presentation IDs. ELK runs only through the shared browser
runtime, including embedded architecture HTML. No server ELK or new layout
endpoints. Existing layout and OKF profile endpoints remain authoritative.

Layered is the initial advanced-feature algorithm. Spline refinement requires
SPLINES routing. Edge labels use only existing relationship counts. Junctions
are validated shared route points, not relationships. Ports are deterministic
presentation in/out objects, excluded from canonical identity and reading order.
Compound containers use supplied visible hierarchy only; moving a container
moves its visible descendants. Shared geometry handles selection, drag, hit
testing, arrows, and export. Validate finite coordinates, references, connected
routes, hierarchy and bounds; preserve cancellation, supersession, and the last
valid scene. Invalid portions fall back to deterministic orthogonal geometry.

OKF semantic links always remain selected-only, arrowless overlays excluded
from ELK placement. No all-links mode or semantic containment inference.

## AER-R-005: delivery and parity

1. Useful settings, registry foundation, persistence, and compatibility.
2. Better edges: labels, junctions, spline refinement.
3. Presentation ports.
4. Nested containers and supported feature combinations.

Each stage passes automated verification and explicit human visual review
before the next starts. Unsupported implementations cannot be enabled merely
because their metadata exists. Each verified stage has a separate commit.
No automatic issue closeout or capability promotion.

Live architecture, live OKF, embedded architecture HTML and downloaded browser
SVG share geometry functions. Static Go SVG stays deterministic orthogonal and
reports advanced_features_not_applied. No headless OKF export is introduced.
Defaults remain architecture's current layout, OKF Mr. Tree, advanced features
off. New or modified implementation files stay below 600 lines; reuse shared
CSS and BEM-ish classes.

## Verification mapping

- AER-R-001 -> SC-AER-011: ordering, duplicate/cycle/ownership rejection, handler metadata consistency.
- AER-R-002 -> SC-AER-001, SC-AER-008: reloads, isolation, inheritance/explicit clear, unavailable saved preferences, compatibility.
- AER-R-003 -> SC-AER-008: both dialogs, filtering, pinned-runtime option effects and value rejection.
- AER-R-004 -> SC-AER-002 through SC-AER-007, SC-AER-010: geometry, malformed output, interaction and semantics.
- AER-R-005 -> SC-AER-001, SC-AER-009 through SC-AER-011: defaults, parity, downloaded SVG, visual gates.

Required checks: full Go tests/race/vet/build; browser tests/syntax; strict OKF
validation; diff and line audits; real pinned ELK fixtures. At the relevant
stage inspect actual downloaded SVG geometry and normal/full canvas Fit,
wheel zoom, pan, manual moves, selection, focus/Back and responsive settings.
