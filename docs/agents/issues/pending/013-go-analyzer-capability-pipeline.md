# 013 — Go analyzer capability pipeline

Execution type: AFK
Review gate: repository-review
Status: awaiting-human-review

## Parent PRD

docs/architecture/analyze-source/go-analysis/prd.md

## What was built

Split the Go analyzer behind its existing `internal/goanalyzer` entrypoint
into scanner, import-classification, and observation-assembly capabilities.
Keep language-specific logic behind the common analyzer contract and keep the
composition root responsible for registering implementations.

## Acceptance criteria

- [x] The pipeline is explicitly `scan → classify imports → assemble common
  analysis result`.
- [x] Scanner owns traversal, AST parsing, build constraints, test/generated
  filtering, and scan results.
- [x] Import classification owns local, standard-library, external, unresolved,
  cgo, and conditional-import decisions.
- [x] Observation assembly owns common modules, evidence, tags, diagnostics,
  confidence, summaries, ordering, and stable IDs.
- [x] No language switches were added; future analyzers can register through
  `analysis.Analyzer` without changing the Go host orchestration.
- [x] Existing IDs, metadata, ordering, diagnostics, analyzer contract, and
  Go analysis behavior are preserved.

## Implementation result

Added `internal/goanalyzer/scanner`, `internal/goanalyzer/imports`, and
`internal/goanalyzer/observations`. The public `goanalyzer.Analyzer` remains
the stable entrypoint and now coordinates the focused capability functions.

## Verification

- `go test ./... -count=1`
- `go test -race ./...`
- `go vet ./...`
- `go build ./...`
- `staticcheck ./...`
- `golangci-lint run`
- `git diff --check`

## Traceability

- Contract: `docs/architecture/analyze-source/go-analysis/canonical-api-cli-contract.md`
- Scenarios: SC-GO-001, SC-GO-003, SC-GO-004
- Reference boundary: `external/` remains ignored, read-only, and untouched.
