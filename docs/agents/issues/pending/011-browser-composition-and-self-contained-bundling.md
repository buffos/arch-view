# 011 — Browser composition and self-contained bundling

Execution type: AFK
Review gate: visual-review
Status: awaiting-human-review

## Parent PRD

docs/architecture/explore-architecture/prd.md

## What was built

Convert the live viewer to native browser ES modules with a small entrypoint,
explicit application context, and capability-focused modules. Keep exported
HTML self-contained by bundling the module graph with the pinned esbuild Go
dependency and inlining the result.

## Acceptance criteria

- [x] `web/app.js` is a small bootstrap entrypoint and browser responsibilities
  are split into API, state/navigation, viewport, layout, graph, details, and
  settings modules.
- [x] Live module assets are served from the embedded filesystem and use no
  network CDN dependency.
- [x] HTML export bundles the module graph in memory, rejects external imports,
  and inlines JavaScript without external script or stylesheet dependencies.
- [x] Existing live viewer, export, layout, navigation, and model contracts are
  preserved.
- [x] CLI and HTTP host composition is split into focused command/transport
  files; hand-written production files remain below the approximate 400-line
  target.
- [ ] User visual review confirms live and exported HTML behavior, including
  settings, navigation, viewport controls, drag/drop, and full canvas.

## Implementation result

Added native ESM application modules, embedded live assets, an in-memory
esbuild bundler with external-import rejection, and self-containment checks.
The Go viewer host and CLI command handlers were also decomposed by
responsibility so orchestration remains small.

## Verification

- `go test ./... -count=1`
- `go test -race ./...`
- `go vet ./...`
- `go build ./...`
- `staticcheck ./...`
- `golangci-lint run`
- `node --check` for every non-vendor viewer module
- `node internal/viewer/web/graph_route_test.js`
- `node internal/viewer/web/layout_request_test.js`
- export self-containment and deterministic-output tests
- `git diff --check`

## Traceability

- Contract: `docs/architecture/explore-architecture/canonical-api-cli-contract.md`
- Scenarios: SC-EX-001, SC-EX-003, SC-EXPT-003
- Reference boundary: the upstream [reference repository](https://github.com/unclebob/arch-view) remains read-only and untouched.
