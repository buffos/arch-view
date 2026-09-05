# Advanced ELK Stage 2 evidence

Issue: 081, Shared ELK advanced edge geometry.

Status: implemented, verified, and visually approved on 2026-09-05.
The capability remains specified while issue 082 proceeds.

## Delivered behavior

- Architecture relationship counts are sent to ELK only when the scene already
  contains an explicit positive count, then rendered at ELK-returned bounds.
- OKF does not invent counts. Its semantic links remain selected-only,
  arrowless, and excluded from ELK placement.
- Junction markers appear only at finite points shared by at least two visible
  routes. They are presentation aids and carry no semantic relationship role.
- Connected ELK cubic sections survive live rendering and browser SVG download.
  Malformed sections fall back per relationship to deterministic orthogonal
  geometry with an explicit diagnostic.
- Architecture and OKF use the same feature handlers, geometry snapshot,
  route validation, label/junction markup, settings form, and browser ELK
  runtime. Only their scene/source adapters differ.
- Go static SVG remains deterministic orthogonal and reports requested browser
  features as not applied in export provenance.

## Scenario evidence

| Scenario | Evidence |
|---|---|
| SC-AER-002 | Pinned ELK label-bounds fixture and shared label presentation test. |
| SC-AER-003 | Pinned ELK merged-route fixture, incidence validation, and live cyan junction review. |
| SC-AER-006 | Pinned ELK spline fixture, connected-section validation, and live curved-route review. |
| SC-AER-007 | Malformed spline fixture proves per-edge fallback and diagnostic retention. |
| SC-AER-009 | Static SVG test proves deterministic output plus unapplied-feature provenance. |
| SC-AER-010 | No-invented-count fixture and live selected-only arrowless OKF semantic links. |
| SC-AER-011 | Catalog/handler consistency tests and shared architecture/OKF composition roots. |

## Automated verification

- `go test ./... -count=1`: passed.
- `go test -race ./...`: passed.
- `go vet ./...`: passed.
- `go build ./...`: passed.
- All 59 browser tests: passed.
- `node --check` for all browser JavaScript: passed.
- Browser SVG serializer tests retain labels, junctions, and curved route data.
- `git diff --check`: passed.
- Modified implementation-file audit: maximum 439 lines; all are below 600.
- Strict OKF validation: passed.

## Implementer visual pass

- Architecture: verified Layered count labels, shared-route junctions, spline
  refinement, target-only arrows, Fit/full canvas, and unchanged defaults when
  features are disabled.
- OKF: verified the same settings surface and junction renderer, Fit/full
  canvas, no invented count labels, and selected-only arrowless semantic links
  without relayout.
- Regression follow-up: canvas panning now updates the shared viewport transform
  in place. It no longer replaces the SVG during a drag and then calculates
  movement from a detached zero-sized element.
- Visual-review follow-up: changing an advanced feature preserves the expanded
  settings section. The junction control now states that the pinned ELK runtime
  emits inspectable junction points for orthogonal routes, not spline routes.

## Human review

The user approved Stage 2 after checking labels, spline routes, panning, and the
corrected advanced-feature accordion behavior. Junction markers remain
validated output only and may be absent when ELK returns no shared points.
