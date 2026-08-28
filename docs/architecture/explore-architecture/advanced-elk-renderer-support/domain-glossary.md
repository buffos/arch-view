# Advanced ELK renderer support domain glossary

| Term | Category | Definition | Distinction |
|---|---|---|---|
| Renderer-neutral geometry | business object | Versioned layout output shared by supported renderers without carrying analyzer semantics. | It is separate from the canonical architecture model and semantic scene. |
| Presentation feature | business object | Opt-in geometry behavior such as labels, ports, junctions, compound bounds, or spline refinement. | It is not a new relationship type or analyzer capability. |
| Edge label | presentation object | Text and bounds associated with a visible relationship for layout and rendering. | The first label is derived from existing relationship facts; it does not create a new edge. |
| Presentation port | presentation object | A renderer endpoint on a visible node used to place or route an edge. | Ports have no independent architecture meaning in this capability. |
| Junction point | presentation object | A coordinate at which supported layout output indicates a shared route branch. | It is not a canonical hyperedge or relationship. |
| Compound geometry | presentation object | Parent/child bounds and cross-hierarchy routes for a nested visual layout. | It reflects existing hierarchy; it does not create hidden model modules. |
| Spline refinement | presentation policy | Validation and rendering of finite piecewise-cubic ELK routes with deterministic fallback. | It extends route quality, not relationship semantics. |
| Feature negotiation | policy/action | Selection of features supported by the pinned ELK adapter and target renderer. | A catalog entry alone is not negotiated support. |
| Geometry diagnostic | reporting term | Visible explanation that a requested feature was unsupported or its output was invalid. | It is separate from analyzer diagnostics and does not alter model status. |
| Semantic parity | policy | Agreement of IDs, relationships, evidence, accessibility, and labels across renderer surfaces. | Pixel-identical output is not required across browser and static Go SVG. |

Canonical feature statuses: `disabled`, `enabled`, `supported`, `degraded`,
`fallback`.
