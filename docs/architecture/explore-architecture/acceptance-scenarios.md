# Explore and inspect architecture acceptance scenarios

## SC-EX-001 — Open top-level view

Given a valid model, when a developer opens it locally, then the browser shows an overview scene with hierarchy, layers, relation indicators, diagnostics, and an accessible title/reading path.

Verification: backend-boundary `when-supported`; frontend-integration `when-supported`; end-to-end `when-supported`.

## SC-EX-002 — Drill down and return

Given a visible hierarchy group, when the developer enters it and then uses breadcrumbs/back, then the selected hierarchy path and scroll/viewport context are restored when still valid.

Verification: backend-boundary `when-supported`; frontend-integration `when-supported`; end-to-end `when-supported`.

## SC-EX-003 — Inspect evidence

Given a selected module or relationship, when the developer opens details, then metadata, direction, counts/contributors, diagnostics, and source references are visible.

Verification: backend-boundary `when-supported`; frontend-integration `when-supported`; end-to-end `when-supported`.

## SC-EX-004 — Inspect source safely

Given a source reference inside the project root, when the developer opens it, then a read-only excerpt shows path and location; an outside-root path is rejected visibly.

Verification: backend-boundary `when-supported`; frontend-integration `when-supported`; end-to-end `when-supported`.

## SC-EX-005 — Show cycles and uncertainty

Given cycles, unresolved references, or partial diagnostics, when the view opens, then those states are visually and textually discoverable rather than hidden.

Verification: backend-boundary `when-supported`; frontend-integration `when-supported`; end-to-end `when-supported`.

## SC-EX-006 — Progressive disclosure

Given a graph above the configured detail threshold, when the view opens, then aggregated groups are shown first and details become available on demand without losing contributor traceability.

Verification: backend-boundary `when-supported`; frontend-integration `when-supported`; end-to-end `when-supported`.

## SC-EX-007 — Accessible inspection

Given keyboard-only or non-visual use, when the developer navigates the view, then modules, relations, cycles, diagnostics, and source links are reachable through labels/list/details without requiring pointer graphics.

Verification: backend-boundary `not-applicable`; frontend-integration `when-supported`; end-to-end `when-supported`.

## SC-EX-008 — Safe reanalysis

Given an open session, when reanalysis produces a new valid revision, then the view replaces the model and invalid selections are cleared; when it fails, the last valid revision remains visible with a diagnostic.

Verification: backend-boundary `when-supported`; frontend-integration `when-supported`; end-to-end `when-supported`.
