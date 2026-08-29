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

## Confirmed layout-settings and project-configuration decisions

9. The viewer exposes a dedicated layout settings surface for the pinned ELK/elkjs adapter. It presents a searchable, grouped catalog of the available layout algorithms and options, including type, default, current value, description, and applicability. Options that the current adapter cannot validate or render are visible as unsupported rather than silently applied.
10. Applying a layout profile is an explicit user action. It recalculates the current scene with the selected ELK settings, consumes the returned node positions and edge routes, and treats the operation like `Reset layout` by discarding manual positions for the affected hierarchy path. The canonical model and relationship semantics are unchanged.
11. Project layout preferences use a versioned `.archview.json` file. For a project-backed session, discovery checks the selected target directory and then each parent directory toward the filesystem root; the nearest file wins as a whole, with no v1 merging. If no file is found, built-in defaults apply. A malformed or unsupported nearest file produces an actionable configuration diagnostic and is not silently bypassed in favor of a farther file.
12. Saving distinguishes `Save` from `Save As`. If discovery loaded `.archview.json` from folder X, ordinary `Save` atomically overwrites that exact active file and never creates or copies a project-root file. If discovery found no file, ordinary `Save` is unavailable and the user must choose `Save As`. `Save As` is the only operation that accepts a custom destination folder; it writes the fixed `.archview.json` filename atomically after explicit confirmation and makes that file active for the current session. A model-only session can apply settings for the current session but has no project persistence boundary.
13. The configuration file stores presentation layout preferences only. Analyzer options, canonical model data, viewport state, and manual node positions remain separate concerns. The same resolver may later be reused by headless/export commands, but issue 007 applies it to the interactive viewer.

## Analyzer-scope consumer

The implemented multi-analyzer plugin-runtime direction provides a combined
model plus individual analyzer/project scopes. The viewer exposes those cached
scopes through a dropdown and preserves the current local-first/evidence
workflow, but it does not own analyzer detection, assignment precedence,
process lifecycle, or language semantics. The persisted assignment/source-scope
contract is a specified child of the plugin-runtime capability and remains
outside the implemented Explore v1 scope.

## Actors and inputs

- A developer or maintainer opens a local analysis session in a browser.
- The viewer receives a canonical model, derived layers/cycles, hierarchy projections, and evidence references.
- The local host serves the viewer assets and read-only evidence access according to configured repository scope.
- Exporters may reuse the renderer-neutral view contract without importing interactive session state.

## Implementation and verification focus

The current Explore exact-spec set is complete. Remaining work is
implementation and verification of the specified viewer contract, browser
packaging, layout/rendering thresholds, source-safety cases, accessibility,
and the future assignment-driven configuration consumer without moving analyzer
semantics into the viewer.

## Issue 007 implementation note

The local viewer now implements the confirmed layout-settings boundary through
the pinned catalog and the versioned `.archview.json` resolver. Automated
validation and persistence checks, plus the normal/full-canvas browser review,
are complete for issue 007. Issue 008 implements the next bounded parent-level
option tranche and its visual review is complete. Issue 009 implements the
bounded target-aware node/edge priority tranche; automated verification and
its normal/full-canvas visual review are complete.

## Issue 016 delivery note

The reserved cubic route segment is now active for the bounded Issue 016
extension. The layered adapter maps valid ELK spline control streams to cubic
routes with live-browser/self-contained-HTML parity and deterministic
orthogonal fallback. Browser Download SVG captures the current canvas; Go
static SVG remains deterministic orthogonal. It is an in-node extension of
this capability, not a new planning node; visual review is complete and the
issue is archived.
