# Explore and inspect architecture requirements gap analysis

## Resolved

| Area | Resolution |
|---|---|
| First surface | Local web application with a Go host and browser presentation. |
| Rendering architecture | Renderer-neutral view/scene contract; SVG/HTML first, Canvas/WebGL later for scale. |
| Core workflow | Overview, drill-down, breadcrumbs/back, zoom/pan/fit, search, selection, evidence inspection. |
| Large graphs | Progressive disclosure and hierarchy aggregation with details on demand. |
| Uncertainty | Cycles, external/unresolved targets, diagnostics, and confidence are visible states. |
| Source inspection | Read-only path/line/column evidence; no execution or editing. |
| Accessibility | Keyboard/labels/contrast and list/details fallback are part of the boundary. |
| Ownership | Viewer owns session/presentation state; model/graph capabilities own semantic data and algorithms. |

## Specification closure and residual risks

The scene schema, local HTTP/CLI surface, source-root safety rules, progressive-disclosure behavior, accessibility requirements, and renderer direction are defined in the exact-spec artifacts. Frontend framework, theme tokens, and renderer performance benchmarks remain implementation risks behind the renderer-neutral boundary.

## Readiness

The capability has passed the architecture specification pipeline and readiness review. It may enter implementation/issue slicing after the application synthesis gate is verified; representative graph and security fixtures remain required verification work.
