# 014 — ELK option handler registry

Execution type: AFK
Review gate: visual-review
Status: done

## Parent PRD

docs/architecture/explore-architecture/prd.md

## What was built

Move layout catalog, typed profile validation, discovery/persistence, and
session behavior into `internal/viewer/layout`. Replace central option-value
switching with a canonical-ID handler registry while preserving the flat
`.archview.json` contract and issue 009's deferred node/edge scope.

## Acceptance criteria

- [x] Catalog metadata and layout session/configuration behavior are isolated
  from HTTP transport.
- [x] Editable parent options use registry handlers for enrichment, renderer
  support, algorithm applicability, and typed validation.
- [x] Unsupported or unregistered options remain catalog-only and are never
  silently attached to the root graph.
- [x] Parent option application, reset, active-file Save, Save As, nearest-
  ancestor discovery, fallback, and manual-position clearing are unchanged.
- [x] Node/edge target mapping remains deferred to issue 009.
- [x] User visual review confirms settings origin/error states, Apply/Reset,
  and representative resulting layouts in windowed and full-canvas views.

## Implementation result

Created the layout capability package with catalog, profile/configuration,
session, atomic persistence, and handler-registry units. HTTP handlers now
only translate requests and responses, while supported parent options are
declared in the registry and target metadata remains authoritative.

## Verification

- `go test ./... -count=1`
- `go test -race ./...`
- `go vet ./...`
- `go build ./...`
- `staticcheck ./...`
- `golangci-lint run`
- JavaScript layout request tests
- `git diff --check`

## Traceability

- Contract: `docs/architecture/explore-architecture/canonical-api-cli-contract.md`
- Scenarios: SC-EX-012, SC-EX-013, SC-EX-016
- Reference boundary: the upstream [reference repository](https://github.com/unclebob/arch-view) remains read-only and untouched.

## Artifact sync required

- Application PRD: no impact — the existing settings journey and configuration
  contract are preserved.
- Application architecture summary: synchronized — the layout catalog,
  handler registry, and persistence boundary are recorded.
- Owning capability artifacts: synchronized in
  `.okf/capabilities/explore-architecture.md` and
  `docs/architecture/explore-architecture/orchestration-status.md`; the
  existing layout contract and acceptance artifacts remain unchanged.
- Delivery truth: the active registry, first implementation slice, and
  `.okf/log.md` are synchronized during closeout.
- Reference boundary: the upstream [reference repository](https://github.com/unclebob/arch-view) remains read-only and untouched.

## Review handoff

The user explicitly approved the visual review on 2026-08-27. It covered
settings origin and error states, Apply, Reset, persistence behavior, and
representative layouts in windowed and full-canvas views.

## Closeout result

Closed on 2026-08-27 after explicit user approval. The issue was moved to the
dated delivery archive, its registry row was removed, and the owning
capability and OKF delivery references were synchronized.
