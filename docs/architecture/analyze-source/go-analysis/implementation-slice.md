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
6. write deterministic JSON, self-contained HTML, or accessible SVG artifacts.

## Scope

- In-process Go analyzer host and registry path.
- One Go module per analysis run, with explicit selection for ambiguous multi-module workspaces.
- Package/import graph as the first relationship type.
- Canonical model version arch-view.model/v1.
- Local web overview, navigation, evidence, diagnostics, and accessible list/details inspection.
- Deterministic JSON, HTML, and SVG output.
- Read-only analysis and source inspection.

## Explicit non-goals

- Python, TypeScript, Rust, or Clojure analyzer implementations.
- External process plugins or the future NDJSON protocol.
- Call graphs, runtime tracing, target-code execution, or full type graphs.
- Source editing, source embedding in exports, cloud hosting, or raster output.
- Changes to the reference-only external folder.

## Ordered delivery issues

| Issue | Outcome | Owner | Blocked by | Review gate |
|---|---|---|---|---|
| [001](../../../agents/issues/done/20260825-001-analyzer-host-and-go-project-selection.md) | Host registry, Go manifest, selection, module boundary, and run options | Plugin runtime + Go analysis | None | none |
| [002](../../../agents/issues/pending/002-go-package-import-model-pipeline.md) | Go package/import observations normalized into model v1 | Go analysis + Generate models | 001 | none |
| [003](../../../agents/issues/pending/003-local-web-top-level-architecture-view.md) | First visible local top-level architecture view | Explore architecture | 002 | visual-review |
| [004](../../../agents/issues/pending/004-evidence-drilldown-and-source-inspection.md) | Hierarchy navigation, evidence, diagnostics, cycles, and safe source inspection | Explore architecture | 003 | visual-review |
| [005](../../../agents/issues/pending/005-deterministic-json-html-svg-export.md) | Repeatable JSON, HTML, and SVG artifacts from the same model/view contract | Export + Explore architecture | 003 | visual-review |

## Slice acceptance

- A representative Go repository reaches a visible top-level architecture view through the CLI.
- Local package relationships are traceable to source locations and non-local dependencies remain references or diagnostics.
- Identical inputs and options produce byte-stable model and export artifacts.
- The viewer exposes hierarchy, relationship direction, cycles, diagnostics, and source evidence without executing or editing target code.
- JSON, HTML, and SVG preserve the same model semantics and status.
- All issue dependencies form a linear, acyclic path from analyzer host to visible result and durable artifacts.

## Verification surfaces

- Backend boundary: analyzer manifests, selection, options, package/import observations, model validation, determinism, and status codes.
- Frontend integration: local browser overview, renderer-neutral scene, navigation, accessibility, source-root safety, and visual parity.
- End-to-end: Go repository to model, viewer, and export artifacts.
- Each surface runs when its harness exists; a deferred surface must record the missing harness and retain a manual acceptance path.

## Artifact impact

This slice updates delivery truth only: the implementation-slice record, issue registry, issue files, capability issue references, and orchestration statuses. Product and application-architecture behavior is already specified and does not change.
