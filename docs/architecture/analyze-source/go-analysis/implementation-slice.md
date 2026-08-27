# First implementation slice: Go repository to visible architecture view

## Selected frontier

The first implementation frontier is the specified [Go analysis capability](../../../../.okf/capabilities/analyze-source/go-analysis.md), supported by the specified [analyzer plugin runtime](../../../../.okf/capabilities/analyze-source/plugin-runtime.md).

The planning map has no foggy or bounded nodes. Go is the first supported language in the product baseline and is the smallest analyzer path that can reach the intended visible result. The slice therefore crosses the existing contracts instead of stopping at an isolated parser.

## Vertical outcome

Given a Go repository containing a selected module, a developer can:

1. select or auto-detect the Go analyzer;
2. discover packages and static imports without executing the target repository;
3. receive a deterministic language-neutral model with evidence, non-local references, diagnostics, cycles, and layers;
4. open the top-level model in a local web viewer;
5. drill into hierarchy and source evidence; and
6. write deterministic JSON, self-contained HTML, or accessible SVG artifacts; and
7. adjust the pinned ELK layout through the viewer and persist project presentation preferences.

## Scope

- In-process Go analyzer host and registry path.
- One Go module per analysis run, with explicit selection for ambiguous multi-module workspaces.
- Package/import graph as the first relationship type.
- Canonical model version arch-view.model/v1.
- Local web overview, navigation, evidence, diagnostics, and accessible list/details inspection.
- Deterministic JSON, HTML, and SVG output.
- Read-only analysis and source inspection.
- User-selectable ELK layout settings and nearest-ancestor `.archview.json` project configuration for the interactive viewer.

## Explicit non-goals

- Python, TypeScript, Rust, or Clojure analyzer implementations.
- External process plugins or the future NDJSON protocol.
- Call graphs, runtime tracing, target-code execution, or full type graphs.
- Source editing, source embedding in exports, cloud hosting, or raster output.
- Changes to the upstream reference repository.

## Ordered delivery issues

| Issue | Outcome | Owner | Blocked by | Review gate |
|---|---|---|---|---|
| [001](../../../agents/issues/done/20260825-001-analyzer-host-and-go-project-selection.md) | Host registry, Go manifest, selection, module boundary, and run options | Plugin runtime + Go analysis | None | none |
| [002](../../../agents/issues/done/20260825-002-go-package-import-model-pipeline.md) | Go package/import observations normalized into model v1 | Go analysis + Generate models | 001 | none |
| [003](../../../agents/issues/done/20260826-003-local-web-top-level-architecture-view.md) | First visible local top-level architecture view — local-first/reference-boundary refinement | Explore architecture | — | approved |
| [004](../../../agents/issues/done/20260826-004-evidence-drilldown-and-source-inspection.md) | Hierarchy navigation, evidence, diagnostics, safe source inspection, session layout controls, centered fitting, stable drag rendering, and deterministic manual edge routing | Explore architecture | — | approved |
| [005](../../../agents/issues/done/20260826-005-deterministic-json-html-svg-export.md) | Repeatable JSON, HTML, and SVG artifacts from the same model/view contract | Export + Explore architecture | — | approved |
| [006](../../../agents/issues/done/20260826-006-viewer-semantic-summary-and-elk-routing.md) | Post-baseline semantic summaries and ELK edge routing | Explore architecture | — | approved |
| [007](../../../agents/issues/done/20260826-007-elk-layout-settings-and-project-config.md) | User-selectable ELK layout settings and persistent project configuration | Explore architecture | — | done |
| [008](../../../agents/issues/done/20260826-008-expand-elk-parent-options.md) | Expanded ELK parent-level layout option support | Explore architecture | — | approved |
| [009](../../../agents/issues/done/20260827-009-elk-node-edge-option-targets.md) | Target-aware node- and edge-level ELK option mapping | Explore architecture | — | approved |
| [010](../../../agents/issues/done/20260826-010-renderer-neutral-routing-and-geometry.md) | Renderer-neutral route primitives, ELK route normalization, and dedicated path serialization | Explore architecture | — | approved |
| [011](../../../agents/issues/done/20260826-011-browser-composition-and-self-contained-bundling.md) | Native browser modules, embedded esbuild export bundle, focused host/CLI composition | Explore + Export | 010 | approved |
| [012](../../../agents/issues/done/20260826-012-scene-projection-capability.md) | Renderer-neutral scene projection capability and cohesive scene units | Explore architecture | 011 | approved |
| [013](../../../agents/issues/done/20260826-013-go-analyzer-capability-pipeline.md) | Scanner, import classification, and common observation assembly pipeline | Go analysis | 012 | approved |
| [014](../../../agents/issues/done/20260826-014-elk-option-handler-registry.md) | Registry-driven ELK option enrichment/validation and layout transport split | Explore architecture | 012 | approved |
| [015](../../../agents/issues/done/20260826-015-canonical-model-normalization-boundary.md) | Canonical normalization/validation capability boundary | Generate models | 013 | approved |
| [016](../../../agents/issues/done/20260827-016-spline-routing-and-cubic-rendering.md) | ELK spline route normalization and cubic browser/export rendering | Explore architecture | — | approved |

## Current delivery status

The analyzer-to-model path and the revised local viewer implementation are available. Issues 003 through 009 are archived after explicit visual approval of the local-first/reference-boundary baseline, the evidence/inspection workflow, deterministic JSON/HTML/SVG export parity, semantic/ELK refinement, the layout-settings/project-configuration slice, the expanded parent-level option tranche, and target-aware node/edge option mapping. Issues 010–015 implement the architecture-refactor plan and are archived after automated, repository, and user visual review approval; they preserve the analyzer-to-model and export contracts. Issue 016 is archived after automated verification and the user's visual approval of the bounded spline-routing extension.

## Slice acceptance

- A representative Go repository reaches a visible top-level architecture view through the CLI.
- Local package relationships are traceable to source locations and non-local dependencies remain references or diagnostics.
- Identical inputs and options produce byte-stable model and export artifacts.
- The viewer exposes hierarchy, relationship direction, cycles, diagnostics, and source evidence without executing or editing target code.
- JSON, HTML, and SVG preserve the same model semantics and status.
- All issue dependencies form a linear, acyclic path from analyzer host to visible result and durable artifacts; issue 007 depends only on the approved ELK/viewer baseline.

## Verification surfaces

- Backend boundary: analyzer manifests, selection, options, package/import observations, model validation, determinism, and status codes.
- Frontend integration: local browser overview, renderer-neutral scene, navigation, accessibility, source-root safety, and visual parity.
- End-to-end: Go repository to model, viewer, and export artifacts.
- Each surface runs when its harness exists; a deferred surface must record the missing harness and retain a manual acceptance path.

## Artifact impact

This slice updates delivery truth and the synchronized capability records for hierarchy/evidence inspection, source safety, session layout, project layout configuration, target-aware layout option mapping, the staged architecture refactor, and the spline-routing extension. The application PRD and application architecture summary remain behaviorally unchanged in product scope; implementation notes record the new routing, module, scene, analyzer, layout-registry, canonicalization, target-mapping, and cubic-route boundaries. Issue 009's target-aware work remains inside the layout adapter and is not part of the canonical model or export contract. Issue 016 activates the reserved cubic route representation without changing analyzer/plugin contracts or canonical model semantics; self-contained HTML embeds the profile/catalog and pinned ELK runtime, the browser Download SVG captures the current canvas, and static Go SVG remains on its explicit deterministic orthogonal contract.
