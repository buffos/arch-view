# Explore and inspect architecture orchestration status

## State

- Planning state: `specified`.
- State transition: `bounded -> specified` on 2026-08-25.
- Topology: confirmed as the interactive investigation capability.
- Next route: issue 009's node/edge target mapping is explicitly deferred. Refactor issues 010, 011, 012, and 014 are implemented, reviewed, and archived; spline routing remains a later specification frontier.

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
- Manual node movement remains session-only; deterministic orthogonal geometry refreshes affected edge paths synchronously without re-running the layered node layout. The same calculation is used during drag and after drop, and `Reset layout` discards those manual positions and restores the calculated layout.
- The viewer exposes a searchable catalog of the pinned ELK algorithms/options with typed validation, explicit apply/reset behavior, and visible applicability/support information.
- Project layout preferences use versioned `.archview.json` configuration. Discovery walks from the selected target directory toward the filesystem root and selects the nearest file without merging; no file means built-in defaults.
- Ordinary `Save` atomically overwrites exactly the active discovered `.archview.json` and never creates a project-root copy; when no file is active it requires `Save As`. `Save As` is the only operation that accepts a custom destination folder, writes the fixed `.archview.json` filename, and makes it active for the current session. Model-only sessions remain session-only, and layout configuration never stores analyzer options, canonical model facts, viewport state, or manual positions.
- Implementation boundaries are explicit: `internal/viewer/scene` owns renderer-neutral scene projection, `internal/routing` owns route primitives/strategies, `internal/viewer/layout` owns catalog/profile/persistence, and browser modules receive explicit application context/state rather than a shared monolithic closure.

## Artifact plan

The exact capability set is complete: [PRD](prd.md), [glossary](domain-glossary.md), [canonical domain model](canonical-domain-model.md), [canonical use cases](canonical-use-cases.md), [viewer contract](canonical-api-cli-contract.md), [acceptance scenarios](acceptance-scenarios.md), and [readiness review](readiness-review.md), alongside discovery and gap artifacts.

## Current and target truth

- Observed in reference: the viewer supports layers, dependency indicators, drill-down, back navigation, cycle display, scrolling, zooming, reanalysis, and a source window.
- User-confirmed target: the Go product should retain the useful investigation workflow while allowing language-neutral source evidence.
- Required follow-up: implement the local web host, renderer-neutral scene contract, accessible interaction modes, and source-root safety checks.

## Current delivery slice

- Issue 003 delivered the first visible local top-level view with local-first reference policy, semantic confidence labels, and bounded baseline layout; automated acceptance and the required visual review are complete.
- Issue 006 refined the approved baseline with honest non-cycle internal-relationship summaries, separate node identity/relationship confidence wording, the contributor-facing viewer guide, and a locally served ELK/elkjs layered layout with SVG edge-route consumption. The deterministic layer-based layout remains the replaceable fallback; its visual review is complete.
- Issue 004 adds evidence, imports list/detail inspection, drill-down, source safety, accessible inspection, viewport controls, session layout state, read-only reanalysis replacement, compact expanded-canvas controls, centered dense-scene fitting, explicit 100% zoom reset behavior, reset-to-calculated layout, and deterministic manual edge routing. Its implementation, automated verification, and declared visual review are complete.
- Issue 005 reuses the view contract for visual export parity, including reference visibility and import/evidence metadata. Its implementation, automated verification, and explicit user-approved visual parity review are complete; the issue is archived.
- Issue 007 implements the ELK option catalog/settings surface, explicit profile application/reset, nearest-ancestor `.archview.json` discovery, active-file `Save`, and explicit custom-folder `Save As`. Its automated verification and declared visual review are complete and it is archived. Issue 008 implements the bounded parent-level option tranche; its automated verification and declared visual review are complete and it is archived. Issue 009 is explicitly deferred. Issues 010, 011, 012, and 014 implement the routing, browser composition, scene, and layout-registry refactor slices and are archived after review approval. Advanced ports, labels, junctions, compound-graph geometry, and spline rendering require later work. Export configuration consumption remains intentionally outside this slice.
- Issues 003–015 are tracked in [the first Go implementation slice](../analyze-source/go-analysis/implementation-slice.md); issues 013 and 015 are owned by the analyzer/model boundaries respectively.

## Implementation progress

Issue 008 extends the viewer's editable ELK parent-option tranche with typed
validation and root-only request mapping while preserving the same
model/scene boundary. Its automated verification and visual review are
complete.

The architecture-refactor sequence is implemented and reviewed. Issue 010 adds the
renderer-neutral route representation and dedicated SVG serialization; issue
011 isolates native browser modules, in-memory esbuild export bundling, and
focused host/CLI composition; issue 012 isolates scene projection; issue 013
splits the Go analyzer pipeline; issue 014 introduces the ELK option handler
registry; and issue 015 moves canonical normalization/validation behind
`internal/model/canonical`. All automated gates pass. Issues 010–015 are
archived after explicit user approval, and issue 009 remains deferred.

The first viewer slice serves validated model files and Go project analyses through a loopback-only HTTP boundary. Its renderer-neutral scene preserves hierarchy aggregation, directed relationships, layers, cycles, diagnostics, confidence, stable contributor IDs, evidence links, reference-boundary summaries, and accessible import details. Issue 003 now defaults to local-first visibility and supports aggregated/expanded policies. Issue 006 adds explicit internal-relationship summaries for collapsed groups, separates stable node identity from relationship confidence without changing canonical model data, and routes the projected graph through the local ELK adapter with a deterministic fallback. Issue 004 adds hierarchy breadcrumbs/back navigation, selected-node import filters, source-root-confined excerpts, atomic reanalysis replacement, keyboard/list parity, full-canvas navigation, compact expanded-canvas layout, an explicit 100% zoom reset, session-scoped viewport/manual positions, reset-to-calculated layout, and deterministic orthogonal routing for manual positions; the canonical model remains unchanged.

## Artifact sync

- Topology: updated in `.okf/`.
- Capability truth: reflected in [discovery notes](discovery-notes.md) and [requirements gap analysis](requirements-gap-analysis.md).
- Exact specification: refreshed in the linked PRD, domain model, use cases, contract, scenarios, gap analysis, and readiness review with local-first reference visibility and import inspection.
- Product truth: refreshed in [the application PRD](../../prd.md).
- Architecture truth: refreshed in [the application architecture summary](../application-architecture-summary.md); no new boundary or capability was introduced.
- Delivery truth: issues 003, 004, 005, 006, 007, and 008 are archived after visual approval; issue 009 is deferred; issues 010–015 are archived after review approval.
- No impact: issues 008–015 do not change capability topology, product scope, or the external layout contract. Issue 009's target-aware mapping and advanced ELK geometry are deferred; issue 010 reserves route shapes needed for future spline support without enabling it.
- No impact: `.archview.json`, analyzer/plugin contracts, canonical model JSON, scene schema, export formats, and external reference behavior remain unchanged. The implementation-only package boundaries are recorded in the issue handoffs and architecture summary.
