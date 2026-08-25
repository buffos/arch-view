# Explore and inspect architecture discovery notes

## Purpose

Turn a generated architecture model into a useful, language-neutral investigation workflow rather than a static picture.

## Reference observations

The read-only reference viewer provides layered diagrams, dependency indicators, cycle listings, hierarchy drill-down, back navigation, scrolling, zooming, reanalysis, and a source window. These interaction ideas are retained, but the target viewer must consume the neutral model and evidence contract instead of Clojure namespace strings.

## Target boundary

The capability consumes the canonical model and graph/view preparation outputs and provides a renderer-neutral interactive view plus a local web presentation. It owns viewer session state, hierarchy navigation, selection, evidence inspection, source inspection, visual states, accessibility, and progressive disclosure. It does not parse source code, discover dependencies, own canonical graph algorithms, or define export file formats.

## Confirmed visual and interaction decisions

1. The first product surface is a local web application: Go owns the CLI/local server boundary and the browser owns rich interaction and graphics. The frontend is not coupled to a language analyzer.
2. The viewer consumes a renderer-neutral view/scene contract. SVG/HTML is the first normal renderer/export surface; Canvas/WebGL can be added behind the same contract for very large graphs.
3. The primary workflow is overview -> hierarchy drill-down -> module/relationship inspection, with breadcrumbs/back, zoom, pan, fit-to-view, and search.
4. Selecting a module or edge reveals metadata, tags, relationship direction, contributing evidence, diagnostics, and source references. Source inspection is read-only and can show file path plus line/column; it never executes or edits target code.
5. Cycles, unresolved targets, external references, and partial confidence are visible visual states, not silently omitted edges.
6. Large graphs use progressive disclosure: hierarchy aggregation and level-of-detail first, details on demand, and interaction state independent of the canonical model.
7. The viewer supports keyboard navigation, accessible labels/contrast, and a non-visual list/details path so the graph is not the only way to inspect architecture.
8. Reanalysis replaces the model while preserving a safe, explainable navigation context where possible; stale evidence is never presented as current.

## Actors and inputs

- A developer or maintainer opens a local analysis session in a browser.
- The viewer receives a canonical model, derived layers/cycles, hierarchy projections, and evidence references.
- The local host serves the viewer assets and read-only evidence access according to configured repository scope.
- Exporters may reuse the renderer-neutral view contract without importing interactive session state.

## Open questions for exact specification

- Frontend framework/bundling and local-server packaging.
- Exact view/scene contract, layout ownership, renderer thresholds, and large-graph performance budgets.
- Theme, color semantics, filtering/search grammar, and detailed visual treatment of relation types.
- Source delivery policy for paths outside the repository, symlinks, unreadable files, and optional embedded source.
- Accessibility acceptance criteria and browser support matrix.
- Reanalysis behavior when module IDs or hierarchy paths change.
