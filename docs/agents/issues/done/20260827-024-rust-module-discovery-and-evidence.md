# 024 — Rust module discovery and source evidence

Execution type: AFK
Review gate: none
Status: done

## Parent PRD

`docs/architecture/analyze-source/rust-analysis/prd.md`

## What to build

For the selected Cargo crate, discover the reachable crate root, file-backed
modules, and inline modules and expose their hierarchy through the common
analysis result. Preserve file and declaration locations as source evidence,
carry crate/feature/target/cfg metadata, and apply the Rust default scope
policy for tests, examples, benches, generated/build output, caches, vendor,
`.git`, `external`, and explicit exclusions.

## Acceptance criteria

- [x] The selected crate emits one crate module plus deterministic file-backed
  and inline module observations with stable IDs, names, display names,
  hierarchy segments, module kind, and attached source-file evidence.
- [x] `mod name;`, `mod name { ... }`, `#[path = "..."]`, `lib.rs`, `main.rs`,
  configured library/binary paths, and reachable nested module layouts are
  resolved conservatively and remain inside the selected crate.
- [x] Source references retain repository-relative paths and positive
  one-based locations for module declarations; repeated analysis produces the
  same module, evidence, tag, and metadata ordering.
- [x] Tests and `cfg(test)` modules are excluded by default and become
  discoverable only with `include_tests`; examples and benches are excluded by
  default and become discoverable with `include_examples`.
- [x] Default and explicit exclusions prevent excluded Rust source from being
  read or emitted, while generated source markers and unresolved module files
  remain visible through tags/diagnostics without fabricating source content.
- [x] Cfg conditions and selected feature/target metadata are attached to the
  affected module observations; conditional compilation is not executed or
  silently treated as proven.

## Artifact sync required

- Application PRD: `none` — module discovery stays within the specified Rust
  capability and common analyzer product boundary.
- Application architecture summary: `none` — no new port, adapter boundary,
  persistence, security policy, or cross-capability contract is introduced.
- Owning capability node/artifacts: `required: .okf/capabilities/analyze-source/rust-analysis.md; docs/architecture/analyze-source/rust-analysis/orchestration-status.md; docs/architecture/analyze-source/rust-analysis/implementation-slice.md`.
- Issue registry: `required`; node `issues:` reference: `required`.
- Reason/no-impact decision: capability delivery truth and implementation
  evidence change; application-level behavior remains incomplete until issue
  025 integrates relationships and the public journey.

## Blocked by

Blocked by `docs/agents/issues/done/20260827-023-rust-cargo-boundary-and-registration.md`; issue 023 was completed and archived before this slice.

## User stories addressed

The Rust child PRD has no standalone user-story IDs. This slice covers
`RS-FR-002`, `RS-FR-005`, and `SC-RS-002`, `SC-RS-005`.

## Scenario traceability and verification plan

- `RS-FR-002`, `RS-FR-005`, `SC-RS-002`, and `SC-RS-005` are covered by the
  module-discovery, source-evidence, scope, exclusion, cfg, and generated-file
  tests in `internal/rustanalyzer`.
- Focused verification: `go test ./internal/rustanalyzer ./cmd/arch-view
  -count=1`.

## Implementation result

Implemented reachable crate-root discovery, inline and file-backed module
hierarchies, `#[path]` resolution, source evidence, deterministic metadata,
scope filtering, cfg/macro/generated uncertainty, and safe unresolved-module
diagnostics without executing Rust tooling.

## Verification record

Acceptance criteria are complete. Focused and repository verification passed:
`go test ./internal/rustanalyzer ./cmd/arch-view -count=1`, `go test ./...
-count=1`, `go test -race ./...`, `go vet ./...`, `go build ./...`,
`staticcheck ./...`, `golangci-lint run`, and strict OKF validation.
