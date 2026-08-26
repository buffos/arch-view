# 015 — Canonical model normalization boundary

Execution type: AFK
Review gate: repository-review
Status: awaiting-human-review

## Parent PRD

docs/architecture/generate-models/prd.md

## What was built

Move canonical normalization and validation behind
`internal/model/canonical`, while retaining canonical model types, graph
derivation, and hierarchy projection data in `internal/model`. Preserve the
model JSON schema, merge rules, validation codes, derived projections, IDs,
and ordering.

## Acceptance criteria

- [x] `canonical.Normalize(...)` and `canonical.Validate(...)` are the
  internal normalization boundary used by CLI, viewer, scene, and export.
- [x] Normalization is split into orchestration, collection merging,
  diagnostics/conflict recovery, identity/path utilities, and validation units.
- [x] Graph derivation remains model-owned and is composed explicitly by the
  canonical normalization pipeline.
- [x] `arch-view.model/v1`, stable IDs, normalized paths, merge behavior,
  validation error codes, cycles, feedback relationships, layers, and ordering
  remain unchanged.
- [x] The CLI, viewer host, scene, export, and projection command validate
  through the new canonical package.
- [x] Hand-written production files remain below the approximate 400-line
  target.

## Implementation result

Added `internal/model/canonical` and migrated all internal callers. The model
package now contains data types, graph derivation, stable model-owned identity,
and projection construction; canonicalization no longer lives in a giant
model file. The viewer host and CLI were decomposed as part of the same
architecture cleanup.

## Verification

- `go test ./... -count=1`
- `go test -race ./...`
- `go vet ./...`
- `go build ./...`
- `staticcheck ./...`
- `golangci-lint run`
- JavaScript syntax and pure-module tests
- export self-containment/determinism tests
- `git diff --check`

## Traceability

- Contract: `docs/architecture/generate-models/canonical-api-cli-contract.md`
- Scenarios: SC-GM-001, SC-GM-004, SC-GM-005, SC-GM-007
- Reference boundary: the upstream [reference repository](https://github.com/unclebob/arch-view) remains read-only and untouched.
