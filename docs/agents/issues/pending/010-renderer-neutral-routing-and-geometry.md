# 010 — Renderer-neutral routing and geometry

Execution type: AFK
Review gate: visual-review
Status: awaiting-human-review

## Parent PRD

docs/architecture/explore-architecture/prd.md

## What was built

Extract the route representation and deterministic manual routing from the
viewer/export renderers. Normalize ELK sections and bend points into the same
route model used by browser drag/drop and SVG export. Keep spline data reserved
for a later issue without enabling spline rendering.

## Acceptance criteria

- [x] Renderer-neutral points, node boxes, routes, sections, segments, route
  kinds, and a router strategy boundary exist in `internal/routing`.
- [x] Polyline, orthogonal, and self-loop behavior preserve current labels,
  arrowheads, manual drag/drop behavior, and self-loop appearance.
- [x] Invalid or missing route data uses the deterministic orthogonal fallback.
- [x] ELK sections/bend points and browser routes use the same conceptual
  representation; SVG path serialization is isolated from graph orchestration.
- [x] Existing `embeddedExport.layouts` shape and layout fallback behavior are
  unchanged.
- [ ] User visual review confirms windowed, full-canvas, export, drag, drop,
  labels, arrows, and self-loop presentation remain acceptable.

## Implementation result

Added `internal/routing` with line and reserved cubic segment support, moved
ELK/manual route normalization into `web/graph_route.js`, and made the SVG
renderer consume route sections through a dedicated serializer. The current
manual route strategy remains deterministic orthogonal routing; no Libavoid or
spline rendering was introduced.

## Verification

- `go test ./... -count=1`
- `go test -race ./...`
- `go vet ./...`
- `go build ./...`
- `staticcheck ./...`
- `golangci-lint run`
- JavaScript syntax and `graph_route_test.js`
- `git diff --check`

## Traceability

- Contract: `docs/architecture/explore-architecture/canonical-api-cli-contract.md`
- Scenarios: SC-EX-003, SC-EX-004, SC-EX-009
- Reference boundary: the upstream [reference repository](https://github.com/unclebob/arch-view) remains read-only and untouched.
