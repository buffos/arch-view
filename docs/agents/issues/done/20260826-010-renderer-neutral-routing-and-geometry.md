# 010 — Renderer-neutral routing and geometry

Execution type: AFK
Review gate: visual-review
Status: done

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
- [x] User visual review confirms windowed, full-canvas, export, drag, drop,
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

## Artifact sync required

- Application PRD: no impact — the refactor preserves the existing viewer
  behavior and external contracts.
- Application architecture summary: synchronized — the renderer-neutral route
  boundary and deferred spline work are recorded.
- Owning capability artifacts: synchronized in
  `.okf/capabilities/explore-architecture.md` and
  `docs/architecture/explore-architecture/orchestration-status.md`; the
  existing scene contract and acceptance artifacts remain unchanged.
- Delivery truth: the active registry, first implementation slice, and
  `.okf/log.md` are synchronized during closeout.
- Reference boundary: the upstream [reference repository](https://github.com/unclebob/arch-view) remains read-only and untouched.

## Review handoff

The user explicitly approved the visual review on 2026-08-27. It covered the
windowed, full-canvas, and exported route presentation, drag/drop, labels,
arrowheads, and self-loop behavior.

## Closeout result

Closed on 2026-08-27 after explicit user approval. The issue was moved to the
dated delivery archive, its registry row was removed, and the owning
capability and OKF delivery references were synchronized.
