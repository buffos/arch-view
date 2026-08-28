# Advanced ELK renderer support requirements gap analysis

## Scope examined

This pass covers the bounded [Advanced ELK renderer support](../../../../.okf/capabilities/explore-architecture/advanced-elk-renderer-support.md) child, the pinned ELK catalog, the current flat scene projection, the renderer-neutral route primitives, the browser/embedded HTML serializers, and deterministic Go SVG export.

## Confirmed strong areas

- **Observed in code:** the pinned catalog already knows about edge labels, ports, junction points, hierarchy handling, compound support, and spline output, while the current scene/layout request uses a flat node/edge shape.
- **Observed in code:** browser and embedded HTML support renderer-neutral line/cubic routes with deterministic orthogonal fallback; Go static SVG deliberately uses its own deterministic orthogonal layout.
- **Inferred from docs:** accessibility and semantic relationship meaning are carried by the scene/list/details path rather than by geometry.
- **User-confirmed target behavior:** advanced ELK support is renderer-only, staged, and must not silently turn catalog entries into unsupported behavior.

## Blocking gaps resolved by recommendation

| Gap | Impact | Resolution |
|---|---|---|
| The first concrete feature set was not fixed | High: implementation could expand into the entire ELK catalog | Specify five opt-in presentation features: `edge_labels`, `junctions`, `ports`, `compound`, and `spline_refinement`; broader target-specific options remain catalog-only until individually admitted. |
| Scene/route representation for geometry was missing | High: browser and SVG implementations could diverge | Add a versioned renderer-neutral geometry snapshot with bounds, compound parent/child links, presentation ports, edge labels, junctions, and the existing line/cubic route sections. |
| Port and junction semantics could be mistaken for architecture facts | High: renderers might invent relationships | Ports and junctions are presentation-only; binary canonical relationships remain the only semantic edges, and junctions render only when supported output identifies shared route points. |
| Surface parity and static SVG behavior were unresolved | High: the same layout choice could produce contradictory artifacts | Live browser, self-contained HTML, and browser Download SVG consume the same advanced geometry. Go static SVG remains deterministic orthogonal and reports that advanced geometry was not applied. |
| Invalid/unsupported ELK output had no exact behavior | High: malformed geometry could break rendering | Validate finite connected geometry; use orthogonal fallback and a visible layout diagnostic on invalid or unsupported output. |
| Accessibility exposure was not explicit | Medium: geometry-only features could become invisible to keyboard/list users | Keep semantic list/details as the authoritative accessible path; include edge-label text in relationship descriptions, and expose compound hierarchy through existing node reading order. |

## Deferrable implementation details

- Exact visual tokens, font metrics, and SVG styling can vary if semantic labels, geometry validity, and fallback behavior remain stable.
- ELK options outside the admitted feature set remain catalog-only and require a later bounded feature/specification pass.

## Exact assumptions

- `arch-view.geometry/v1` is additive to `arch-view.scene/v1`; semantic scene fields remain unchanged.
- Feature selection is opt-in through `layout.features`; an empty list preserves current behavior.
- Feature diagnostics are non-fatal when a valid orthogonal scene can still be rendered.

## Readiness conclusion

No High or Medium specification gaps remain. The admitted feature subset,
geometry fields, option gating, surface parity, accessibility, and fallback
behavior are exact enough for implementation slicing.

## Artifact impact

- **Capability:** this report and the exact-spec set define the renderer feature boundary and geometry contract.
- **Product:** the application PRD remains a static architecture viewer; advanced features are presentation extensions with no model-semantic change.
- **Architecture:** the scene/route boundary, layout registry, browser/export parity, and deterministic fallback are affected.
- **Delivery:** no issues are created in this specification pass.
