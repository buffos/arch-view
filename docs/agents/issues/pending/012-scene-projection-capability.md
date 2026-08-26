# 012 — Scene projection capability

Execution type: AFK
Review gate: visual-review
Status: awaiting-human-review

## Parent PRD

docs/architecture/explore-architecture/prd.md

## What was built

Move scene construction into `internal/viewer/scene` and keep the scene
contract renderer-neutral. Split orchestration, reference aggregation,
relationship/cycle projection, layer/diagnostic indicators, evidence, and
accessibility into cohesive units.

## Acceptance criteria

- [x] `scene.BuildScene`, `scene.BuildSceneWithOptions`, `scene.SceneSnapshot`,
  and `scene.SceneOptions` provide the migrated internal API.
- [x] Scene projection imports only analysis/model capabilities and does not
  depend on browser or export packages.
- [x] Schema `arch-view.scene/v1`, stable IDs/order, visibility/reference
  policies, hierarchy navigation, cycles, layers, diagnostics, evidence, and
  accessibility output are preserved.
- [x] The former giant scene builder is split into cohesive files and tests
  continue to cover aggregation and inspection behavior.
- [ ] User visual review confirms the windowed/full-canvas scene and details
  presentation remain unchanged.

## Implementation result

Created the scene package with focused orchestration, request validation,
reference, node, relationship, indicator, evidence, accessibility, and schema
units. Viewer, export, and tests now consume the package instead of a monolithic
viewer scene file.

## Verification

- `go test ./... -count=1`
- `go test -race ./...`
- `go vet ./...`
- `go build ./...`
- `staticcheck ./...`
- `golangci-lint run`
- `git diff --check`

## Traceability

- Contract: `docs/architecture/explore-architecture/canonical-api-cli-contract.md`
- Scenarios: SC-EX-002, SC-EX-006, SC-EX-007, SC-EX-009
- Reference boundary: the upstream [reference repository](https://github.com/unclebob/arch-view) remains read-only and untouched.
