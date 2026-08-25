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

## Scope

The first surface is a local web application. It presents a renderer-neutral view of the canonical model, supports hierarchy navigation, selection, evidence/source inspection, cycle/diagnostic visibility, progressive disclosure, reference-boundary filtering, reanalysis, and accessible non-graph inspection. It does not parse source, discover dependencies, change the model, or define export formats.

## Workflows

1. Open a model at its top-level hierarchy.
2. Search/select a module or relationship.
3. Drill into a hierarchy group and return with breadcrumbs/back.
4. Inspect module metadata, edge direction, evidence, diagnostics, and source locations.
5. Highlight/report cycles and unresolved/external references.
6. Reanalyze a project and replace stale model/evidence safely.

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

## Non-goals

Editing source or diagrams, collaboration, cloud hosting, runtime tracing, automatic architecture judgment, and renderer-specific behavior in the canonical model.
