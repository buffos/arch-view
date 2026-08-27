# Explore and inspect architecture PRD

## Purpose

Give developers a reachable investigation workflow over the canonical architecture model instead of a static diagram.

## Actors

- Developer learning a repository.
- Maintainer investigating dependencies/cycles.
- Architect reviewing module boundaries and evidence.

## User stories

- US-EX-001 — As a developer, I can open a local top-level architecture view and navigate the repository structure.
- US-EX-002 — As a maintainer, I can inspect module, relationship, source, cycle, and diagnostic evidence in read-only mode.
- US-EX-003 — As a developer, I can inspect and adjust the available ELK layout options for my use case and keep those presentation preferences for the project.

## Scope

The first surface is a local web application. It presents a renderer-neutral view of the canonical model, supports hierarchy navigation, selection, evidence/source inspection, cycle/diagnostic visibility, progressive disclosure, reference-boundary filtering, reanalysis, accessible non-graph inspection, and user-configurable ELK presentation layout. It does not parse source, discover dependencies, change the model, or define export formats.

## Workflows

1. Open a model at its top-level hierarchy.
2. Search/select a module or relationship.
3. Drill into a hierarchy group and return with breadcrumbs/back.
4. Inspect module metadata, edge direction, evidence, diagnostics, and source locations.
5. Highlight/report cycles and unresolved/external references.
6. Reanalyze a project and replace stale model/evidence safely.
7. Open layout settings, inspect the pinned ELK option catalog, edit a validated layout profile, and apply it to the current scene.
8. Save project layout preferences back to the active discovered `.archview.json`, use `Save As` for a new custom-folder destination when needed, see the configuration origin, and reset the session to built-in defaults when needed.

## Rules

- Viewer session state is separate from canonical model state.
- Structural hierarchy controls drill-down; dependency relations control edge highlighting and cycle indicators.
- Large graphs begin aggregated and reveal details on demand.
- The default overview prioritizes analyzed project modules and groups. Standard-library, external, unresolved, and dynamic references remain in the canonical model but are hidden or summarized at the boundary unless the user expands them.
- Reference visibility is a view policy with `hidden`, `aggregated`, and `expanded` modes; it never deletes or rewrites canonical relationships.
- Individual imports are evidence/detail facts. They are available through a searchable list/details view for the selected module or group rather than crowding the default architecture graph.
- Relationships between child modules that collapse into a group self-loop are shown as an internal-relationship count when they are not a real cycle; the canonical relationships remain available through details/evidence.
- Node identity stability and relationship confidence are separate inspection facts. A local module/group is not assigned a computed confidence score merely because its ID is stable.
- Layout, viewport, and optional user positioning belong to viewer session state. They never mutate the canonical model and may be persisted only with a model/revision/hierarchy key.
- Layout settings are presentation policy. The settings surface exposes the pinned ELK algorithms/options with their types, defaults, current values, descriptions, and applicability; unknown or invalid values cannot be applied.
- The layout adapter applies validated options at their catalog target: parent options go to the root graph, supported node options are copied to each eligible visible node, and supported edge options are copied to each eligible visible edge. Options requiring unsupported scene features remain catalog-only.
- Project configuration is versioned `.archview.json`. Discovery checks the selected target directory and its ancestors toward the filesystem root; the nearest file wins as a complete profile, with no v1 merging. A malformed or unsupported nearest file is surfaced as a configuration diagnostic instead of silently falling through to another file.
- Ordinary `Save` is an explicit atomic replacement of the active discovered `.archview.json`; it never creates a new file or copies settings to the project root. When no file is active, `Save` is unavailable and the user must choose `Save As`. `Save As` is the only operation that accepts a custom destination folder, writes the fixed `.archview.json` filename atomically after explicit confirmation, and makes it active for the current session. A model-only session can apply settings temporarily but cannot persist project configuration. Analyzer options, viewport state, and manual node positions are not stored in this file.
- Applying a new layout profile recalculates the current scene and routes with the selected layout adapter, discarding manual positions for that hierarchy path as `Reset layout` does. If the adapter is unavailable, the deterministic fallback remains available and the failure is visible.
- Source access is read-only and confined to the analyzed project root.
- Stale evidence is never presented as current after reanalysis.
- The graph has accessible labels, keyboard reachability, contrast, and list/details fallback.

## Functional requirements

| ID | Requirement |
|---|---|
| EX-FR-001 | Open a canonical model in a local browser session. |
| EX-FR-002 | Render hierarchy, modules, typed relations, layers, cycles, diagnostics, and confidence states. |
| EX-FR-003 | Support search, selection, zoom, pan, fit, drill-down, breadcrumbs, and back. |
| EX-FR-004 | Show source evidence and read-only source locations. |
| EX-FR-005 | Use progressive disclosure/aggregation for large graphs. |
| EX-FR-006 | Provide accessible list/details inspection alongside graphics. |
| EX-FR-007 | Reanalyze/reload without exposing stale or cross-root evidence. |
| EX-FR-008 | Filter or aggregate non-local references in the overview while exposing their individual imports and evidence through list/details inspection. |
| EX-FR-009 | Provide a readable layout with fit, pan, zoom, and session-scoped layout state without changing canonical semantics. |
| EX-FR-010 | Expose a searchable, grouped catalog of the pinned ELK algorithms/options with type, default, current value, description, and applicability. |
| EX-FR-011 | Validate and explicitly apply a layout profile, recalculate node positions and edge routes, and provide reset-to-default/session behavior without mutating the canonical model. |
| EX-FR-012 | Discover the nearest versioned `.archview.json` from the selected target directory upward; save back to the active discovered file without creating another file, and support explicit custom-folder `Save As` with clear origin/error reporting. |

## Non-goals

Editing source or diagrams, collaboration, cloud hosting, runtime tracing, automatic architecture judgment, storing manual diagram positions as project semantics, analyzer configuration in the layout file, and renderer-specific behavior in the canonical model.

## Current delivery status

Issue 007 implements the specified layout catalog, profile application/reset,
nearest-ancestor configuration discovery, active-file `Save`, and explicit
custom-folder `Save As`. Issue 008 extends the catalog with the bounded,
root-safe parent-level option tranche; its automated checks and visual review
are complete. Issue 009 implements target-aware mapping for the bounded simple
node/edge priority tranche; its automated checks and visual review are
complete. Refactor issues 010, 011, 012, and 014 preserve
the viewer contract while isolating routing, browser composition, scene
projection, and ELK option handling; their automated checks and review
handoffs are complete. Issue 016 implements the bounded layered spline route
path without expanding the canonical model or configuration schema;
self-contained HTML embeds the profile/catalog and pinned runtime, the browser
Download SVG captures the current canvas, and Go static SVG remains
deterministic orthogonal. Automated verification and visual review are
complete, and the issue is archived.
