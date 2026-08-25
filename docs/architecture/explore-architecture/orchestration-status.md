# Explore and inspect architecture orchestration status

## State

- Planning state: `specified`.
- State transition: `bounded -> specified` on 2026-08-25.
- Topology: confirmed as the interactive investigation capability.
- Next route: implement or slice the specified local web viewer after application synthesis validation.

## Confirmed boundary

The capability consumes the canonical model and graph/view preparation outputs and provides a renderer-neutral interactive view plus a local web presentation. It owns viewer session state, navigation, selection, evidence/source inspection, visual states, progressive disclosure, and accessibility. It does not parse source, discover dependencies, own canonical graph algorithms, or define export formats.

## Confirmed decisions

- The first surface is a local web application with a Go host and browser presentation.
- A renderer-neutral view/scene contract separates interaction and visual technology from the model; SVG/HTML is first, Canvas/WebGL can follow for scale.
- The workflow includes overview, hierarchy drill-down, breadcrumbs/back, zoom/pan/fit, search, selection, edge highlighting, cycle indicators, and evidence/source inspection.
- Large graphs use hierarchy aggregation and level-of-detail with details on demand.
- The default overview is local-first: non-local references remain in the model but use hidden, aggregated, or expanded visibility policy rather than appearing as one node per import.
- Standard-library, external, unresolved, and dynamic references are distinct scopes; individual imports belong in an accessible list/details path.
- Fit/pan/zoom and optional manual positions are viewer session state keyed by model revision and hierarchy path; they do not change canonical model data.
- Cycles, unresolved/external dependencies, diagnostics, and confidence remain visible.
- Source inspection is read-only with path and locations; no target code is executed or edited.
- Keyboard navigation, accessible labels/contrast, and a list/details path complement the graph.
- Reanalysis replaces stale model/evidence state and preserves navigation only when safe and explainable.

## Artifact plan

The exact capability set is complete: [PRD](prd.md), [glossary](domain-glossary.md), [canonical domain model](canonical-domain-model.md), [canonical use cases](canonical-use-cases.md), [viewer contract](canonical-api-cli-contract.md), [acceptance scenarios](acceptance-scenarios.md), and [readiness review](readiness-review.md), alongside discovery and gap artifacts.

## Current and target truth

- Observed in reference: the viewer supports layers, dependency indicators, drill-down, back navigation, cycle display, scrolling, zooming, reanalysis, and a source window.
- User-confirmed target: the Go product should retain the useful investigation workflow while allowing language-neutral source evidence.
- Required follow-up: implement the local web host, renderer-neutral scene contract, accessible interaction modes, and source-root safety checks.

## Current delivery slice

- Issue 003 delivered the first visible local top-level view with local-first reference policy, semantic confidence labels, and bounded baseline layout; automated acceptance and the required visual review are complete.
- Issue 006 refines the approved baseline with honest non-cycle internal-relationship summaries, separate node identity/relationship confidence wording, the contributor-facing viewer guide, and a locally served ELK/elkjs layered layout with SVG edge-route consumption. The deterministic layer-based layout remains the replaceable fallback.
- Issue 004 adds evidence, imports list/detail inspection, drill-down, source safety, accessible inspection, viewport controls, and session layout state.
- Issue 005 reuses the view contract for visual export parity, including reference visibility and import/evidence metadata.
- Issues 003, 004, 005, and the post-baseline issue 006 are tracked in [the first Go implementation slice](../analyze-source/go-analysis/implementation-slice.md).

## Implementation progress

The first viewer slice serves validated model files and Go project analyses through a loopback-only, GET-only HTTP boundary. Its renderer-neutral scene preserves hierarchy aggregation, directed relationships, layers, cycles, diagnostics, confidence, stable contributor IDs, evidence links, reference-boundary summaries, and accessible import details. Issue 003 now defaults to local-first visibility and supports aggregated/expanded policies. Issue 006 adds explicit internal-relationship summaries for collapsed groups, separates stable node identity from relationship confidence without changing canonical model data, and routes the projected graph through the local ELK adapter with a deterministic fallback.

## Artifact sync

- Topology: updated in `.okf/`.
- Capability truth: reflected in [discovery notes](discovery-notes.md) and [requirements gap analysis](requirements-gap-analysis.md).
- Exact specification: refreshed in the linked PRD, domain model, use cases, contract, scenarios, gap analysis, and readiness review with local-first reference visibility and import inspection.
- Product truth: refreshed in [the application PRD](../../prd.md).
- Architecture truth: refreshed in [the application architecture summary](../application-architecture-summary.md); no new boundary or capability was introduced.
- Delivery truth: issue 003 is archived after visual approval; issue 006 carries the follow-up semantic/presentation refinement and remains in visual review, while issues 004 and 005 are unblocked for their independent scopes.
