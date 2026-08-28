# Advanced ELK renderer support acceptance scenarios

## SC-AER-001 — Preserve the current default

**Given** a layout profile with no `features`, **when** the scene is laid out,
**then** the current flat scene and route behavior remain valid and no
advanced geometry feature is required.

Verification: backend-boundary `when-supported`; frontend-integration
`when-supported`; end-to-end `when-supported`.

## SC-AER-002 — Render edge labels

**Given** a valid scene and `edge_labels`, **when** layered ELK returns label
bounds, **then** the renderer draws the deterministic relationship-count label
at the returned position and preserves the relationship ID and accessibility
description.

Verification: backend-boundary `when-supported`; frontend-integration
`when-supported`; end-to-end `when-supported`.

## SC-AER-003 — Render supported junctions

**Given** valid ELK junction output shared by at least two visible edge routes,
**when** `junctions` is enabled, **then** a presentation junction marker is
rendered without adding a canonical relationship or hyperedge.

Verification: backend-boundary `when-supported`; frontend-integration
`when-supported`; end-to-end `when-supported`.

## SC-AER-004 — Render presentation ports

**Given** a scene with `ports` enabled, **when** layout completes, **then**
deterministic in/out presentation ports are used for endpoint geometry and
remain absent from the canonical model and architecture reading order.

Verification: backend-boundary `when-supported`; frontend-integration
`when-supported`; end-to-end `when-supported`.

## SC-AER-005 — Render compound hierarchy

**Given** a scene whose visible nodes have nested hierarchy, **when** `compound`
is enabled, **then** parent bounds, child bounds, and cross-hierarchy routes are
rendered with existing hierarchy semantics and no hidden module is invented.

Verification: backend-boundary `when-supported`; frontend-integration
`when-supported`; end-to-end `when-supported`.

## SC-AER-006 — Preserve valid spline refinement

**Given** finite connected ELK spline sections, **when** `spline_refinement` is
enabled, **then** cubic control points are preserved in the geometry snapshot
and rendered identically by live browser, self-contained HTML, and browser SVG.

Verification: backend-boundary `when-supported`; frontend-integration
`when-supported`; end-to-end `when-supported`.

## SC-AER-007 — Fallback malformed geometry

**Given** non-finite, disconnected, or unknown route/feature output, **when**
geometry normalization runs, **then** the affected geometry uses deterministic
orthogonal fallback, a visible geometry diagnostic is retained, and no invalid
SVG path is emitted.

Verification: backend-boundary `when-supported`; frontend-integration
`when-supported`; end-to-end `when-supported`.

## SC-AER-008 — Keep unsupported catalog entries catalog-only

**Given** a catalogued ELK option outside the five admitted features, **when** a
user requests it, **then** the profile is rejected or the option remains
catalog-only according to its catalog status and no partial renderer support is
pretended.

Verification: backend-boundary `when-supported`; frontend-integration
`when-supported`; end-to-end `when-supported`.

## SC-AER-009 — Preserve static SVG contract

**Given** a model and an advanced browser layout profile, **when** static Go SVG
is exported, **then** it remains deterministic orthogonal, retains semantic
labels/accessibility, and reports that advanced browser geometry was not
applied.

Verification: backend-boundary `when-supported`; frontend-integration
`not-applicable`; end-to-end `when-supported`.

## SC-AER-010 — Preserve accessible semantic parity

**Given** any supported feature combination, **when** a user switches between
graphic and list/details presentation, **then** relationship IDs, labels,
hierarchy, evidence, diagnostics, and reading order remain available without
requiring ports or junctions to be interpreted as architecture facts.

Verification: backend-boundary `when-supported`; frontend-integration
`when-supported`; end-to-end `when-supported`.

## SC-AER-011 — Preserve empty/feature-combination determinism

**Given** the same model, profile, pinned runtime, and enabled feature set,
**when** the layout is repeated, **then** geometry JSON and browser/HTML route
meaning are stable, and feature order does not alter the result.

Verification: backend-boundary `when-supported`; frontend-integration
`when-supported`; end-to-end `when-supported`.
