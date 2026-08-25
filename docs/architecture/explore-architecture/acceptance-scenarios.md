# Explore and inspect architecture acceptance scenarios

## SC-EX-001 — Open local-first top-level view

Given a valid model, when a developer opens it locally, then the browser shows a readable local-first overview scene with hierarchy, layers, relation indicators, non-local boundary summary, diagnostics, and an accessible title/reading path.

Verification: backend-boundary `when-supported`; frontend-integration `when-supported`; end-to-end `when-supported`.

## SC-EX-002 — Drill down and return

Given a visible hierarchy group, when the developer enters it and then uses breadcrumbs/back, then the selected hierarchy path and scroll/viewport context are restored when still valid.

Verification: backend-boundary `when-supported`; frontend-integration `when-supported`; end-to-end `when-supported`.

## SC-EX-003 — Inspect evidence and imports

Given a selected module or relationship, when the developer opens details, then metadata, direction, counts/contributors, diagnostics, source references, and the selected module/group's individual imports are visible in a list-oriented inspection path.

Verification: backend-boundary `when-supported`; frontend-integration `when-supported`; end-to-end `when-supported`.

## SC-EX-004 — Inspect source safely

Given a source reference inside the project root, when the developer opens it, then a read-only excerpt shows path and location; an outside-root path is rejected visibly.

Verification: backend-boundary `when-supported`; frontend-integration `when-supported`; end-to-end `when-supported`.

## SC-EX-005 — Show cycles and uncertainty

Given cycles, unresolved references, or partial diagnostics, when the view opens, then those states are visually and textually discoverable rather than hidden. Standard-library, external, unresolved, and dynamic reference scopes remain distinguishable even when individual references are not in the overview graph.

Verification: backend-boundary `when-supported`; frontend-integration `when-supported`; end-to-end `when-supported`.

## SC-EX-006 — Progressive disclosure and reference boundaries

Given a graph above the configured detail threshold or containing many non-local references, when the view opens, then aggregated local groups and reference-boundary summaries are shown first and details become available on demand without losing contributor traceability.

Verification: backend-boundary `when-supported`; frontend-integration `when-supported`; end-to-end `when-supported`.

## SC-EX-007 — Accessible inspection

Given keyboard-only or non-visual use, when the developer navigates the view, then modules, relations, cycles, diagnostics, source links, and individual imports are reachable through labels/list/details without requiring pointer graphics.

Verification: backend-boundary `not-applicable`; frontend-integration `when-supported`; end-to-end `when-supported`.

## SC-EX-008 — Safe reanalysis

Given an open session, when reanalysis produces a new valid revision, then the view replaces the model and invalid selections are cleared; when it fails, the last valid revision remains visible with a diagnostic.

Verification: backend-boundary `when-supported`; frontend-integration `when-supported`; end-to-end `when-supported`.

## SC-EX-009 — Inspect import boundaries

Given a module with standard-library, external, unresolved, or dynamic imports, when the developer opens the overview, then non-local references are hidden or aggregated by the selected visibility policy; when the developer opens imports/list details or expands the policy, then each import exposes its scope, confidence, count, and source evidence.

Verification: backend-boundary `when-supported`; frontend-integration `when-supported`; end-to-end `when-supported`.

## SC-EX-010 — Keep a readable session layout

Given a dense local graph, when the developer fits, pans, zooms, or moves a node, then the layout remains readable and the session preserves viewport/layout overrides for the same model revision and hierarchy path without mutating canonical model data.

Verification: backend-boundary `not-applicable`; frontend-integration `when-supported`; end-to-end `when-supported`.

## SC-EX-011 — Explain collapsed internal relationships

Given a hierarchy group whose child modules have relationships with one another
but no real cycle, when the group is shown in the overview, then the viewer
does not draw a cycle-like self-loop and instead exposes the number and
canonical IDs of the internal relationships through the node/details path.
Given a real canonical cycle, when the group is shown, then the cycle remains
visually and textually identified. Local node identity is shown separately
from relationship confidence.

Verification: backend-boundary `when-supported`; frontend-integration `when-supported`; end-to-end `when-supported`.
