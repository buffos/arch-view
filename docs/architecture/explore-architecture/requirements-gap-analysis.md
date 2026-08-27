# Explore and inspect architecture requirements gap analysis

## Resolved

| Area | Resolution |
|---|---|
| First surface | Local web application with a Go host and browser presentation. |
| Rendering architecture | Renderer-neutral view/scene contract; SVG/HTML first, Canvas/WebGL later for scale. |
| Core workflow | Overview, drill-down, breadcrumbs/back, zoom/pan/fit, search, selection, evidence inspection. |
| Large graphs | Progressive disclosure and hierarchy aggregation with details on demand. |
| Uncertainty | Cycles, external/unresolved targets, diagnostics, and confidence are visible states. |
| Reference boundary | The overview is local-first; standard-library, external, unresolved, and dynamic references remain canonical but are hidden, aggregated, or expanded through an explicit view policy. |
| Import inspection | Individual imports are detail/evidence facts exposed through an accessible list for the selected module or group rather than default graph nodes. |
| Layout ownership | Layout, fit/pan/zoom, and optional manual positions belong to renderer/session state; they never mutate canonical model semantics. |
| Layout configuration | The viewer provides a searchable catalog of the pinned ELK algorithms/options, validates typed values and applicability, and applies settings explicitly to the current scene. |
| Project configuration discovery | `.archview.json` is discovered from the selected target directory upward; the nearest file wins as a complete profile, with built-in defaults when none exists. |
| Configuration persistence | `Save` overwrites the exact discovered `.archview.json` that is active for the session and never creates a replacement elsewhere; with no active file it requires `Save As`. `Save As` is the only custom-destination operation and atomically writes the fixed `.archview.json` filename after explicit confirmation. Model-only sessions remain session-only and no source file is edited. |
| Configuration boundary | Layout configuration is presentation policy and remains separate from analyzer options, canonical model facts, viewport state, and manual positions. |
| Source inspection | Read-only path/line/column evidence; no execution or editing. |
| Accessibility | Keyboard/labels/contrast and list/details fallback are part of the boundary. |
| Ownership | Viewer owns session/presentation state; model/graph capabilities own semantic data and algorithms. |

## Specification closure and residual risks

The scene schema, local HTTP/CLI surface, source-root safety rules, progressive-disclosure/reference-boundary behavior, accessibility requirements, renderer direction, layout-option catalog, and project-config discovery/persistence rules are defined in the exact-spec artifacts. Frontend framework, theme tokens, and renderer performance benchmarks remain implementation risks behind the renderer-neutral boundary; issue 007's settings-page review is complete.

The implementation risks for the configuration extension are covered by the complete pinned ELK catalog, typed validation, explicit unsupported/non-applicable treatment, platform-specific atomic-write tests, and the approved settings-surface review. Benchmark fixtures remain verification work, while broader ELK renderer support is now explicitly tracked in the [Advanced ELK renderer support future-work register](advanced-elk-renderer-support/future-work.md).

## Readiness

The capability has passed the architecture specification pipeline and readiness review. It may enter implementation/issue slicing after the application synthesis gate is verified; representative graph and security fixtures remain required verification work.
