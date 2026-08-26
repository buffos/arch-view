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

## SC-EX-012 — Inspect the ELK layout catalog

Given an open viewer session, when the developer opens layout settings, then a searchable and grouped catalog exposes the pinned ELK algorithms and options with type, default, current value, description, applicability, and renderer-support information, and unsupported entries cannot be applied as if they were valid.

Verification: backend-boundary `when-supported`; frontend-integration `when-supported`; end-to-end `when-supported`.

## SC-EX-013 — Apply and reset a layout profile

Given a valid layout profile, when the developer applies it, then the current scene is recalculated with the selected layout adapter, the renderer consumes the resulting node positions and edge routes, manual positions for that hierarchy path are cleared, and canonical model facts remain unchanged. Reset returns the active session to built-in defaults without silently deleting a project file.

Verification: backend-boundary `when-supported`; frontend-integration `when-supported`; end-to-end `when-supported`.

## SC-EX-014 — Discover the nearest project configuration

Given a target directory with zero or more ancestor `.archview.json` files, when a project-backed viewer session opens, then Arch View checks the target directory and walks toward the filesystem root, selects the nearest file as one complete profile without merging, reports its origin, and uses built-in defaults when no file exists.

Verification: backend-boundary `when-supported`; frontend-integration `when-supported`; end-to-end `when-supported`.

## SC-EX-015 — Persist project layout settings safely

Given a project-backed session and a valid edited profile, when discovery loaded `.archview.json` from folder X and the developer chooses ordinary `Save`, then Arch View atomically overwrites that exact active file, reports the saved origin/path, creates no project-root copy, and loads the same settings on the next project session. Given no discovered active file, ordinary `Save` is unavailable or returns `save_as_required` without creating a file. When the developer explicitly chooses `Save As`, Arch View accepts a user-selected custom destination folder, atomically writes the fixed `.archview.json` filename there, makes it active for the current session, and reports the destination. Only `Save As` can choose a destination; neither operation edits source files.

Verification: backend-boundary `when-supported`; frontend-integration `when-supported`; end-to-end `when-supported`.

## SC-EX-016 — Handle invalid or model-only configuration

Given a malformed/unsupported nearest `.archview.json` or a model-only session, when layout configuration is loaded or saved, then the viewer reports an actionable diagnostic, does not silently fall through to a farther file, uses safe defaults where necessary, and disables project persistence when no project root is available. A model-only session may still apply settings for the current session, but cannot use either `Save` or `Save As`.

Verification: backend-boundary `when-supported`; frontend-integration `when-supported`; end-to-end `when-supported`.

## Issue 007 verification status

The backend portions of SC-EX-012 through SC-EX-016 are covered by catalog,
validation, discovery, destination, and model-only endpoint tests. The browser
settings surface and its windowed/full-canvas workflow have passed the declared
human visual review; no scenario changes the language-neutral model contract.
