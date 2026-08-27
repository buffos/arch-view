# 023 — Rust Cargo boundary and analyzer registration

Execution type: AFK
Review gate: none
Status: done

## Parent PRD

`docs/architecture/analyze-source/rust-analysis/prd.md`

## What to build

Add the Rust analyzer's public boundary to the existing analyzer host. A
developer can list the Rust analyzer, detect a root Cargo project, select one
crate by package name or project-relative path, and pass Rust-specific options
through the CLI and common option fingerprinting. A workspace with multiple
candidate crates must require explicit selection. Manifest inspection and
selection remain data-only and must not invoke Cargo or any target code.

## Acceptance criteria

- [x] The Rust manifest validates with ID `org.archview.rust`, language `rust`,
  `Cargo.toml` detection, capabilities `detect`, `static_dependencies`, and
  `cfg_metadata`, and the contract options `crate`, `features`, `target`,
  `include_tests`, `include_examples`, and `exclude`.
- [x] `arch-view analyzers`, explicit `--language rust`, explicit
  `--analyzer org.archview.rust`, and automatic Cargo detection select the Rust
  analyzer with deterministic selection metadata.
- [x] A single-package Cargo project resolves its package name, edition,
  manifest path, and crate root; a multi-package workspace returns the stable
  module-selection error unless `--crate` selects a package or safe
  project-relative crate directory.
- [x] The CLI and `open --project` path pass repeatable Rust features, target,
  include-tests, include-examples, and exclude options without changing the
  existing Go/Python option behavior or layout configuration boundary.
- [x] Malformed, missing, or escaping workspace/package paths produce clear
  host-compatible errors or recoverable diagnostics, and all selected paths
  remain inside the requested project root.
- [x] Tests demonstrate that Cargo manifests containing build scripts or
  side-effecting metadata are read as data only; Cargo, rustc, build scripts,
  proc macros, and target code are never executed.

## Artifact sync required

- Application PRD: `none` — this slice exposes an already-specified analyzer
  contract and does not yet change application product scope or behavior.
- Application architecture summary: `none` — the adapter remains behind the
  existing language-neutral host boundary.
- Owning capability node/artifacts: `required: .okf/capabilities/analyze-source/rust-analysis.md; docs/architecture/analyze-source/rust-analysis/orchestration-status.md; docs/architecture/analyze-source/rust-analysis/implementation-slice.md`.
- Issue registry: `required`; node `issues:` reference: `required`.
- Reason/no-impact decision: delivery and Rust capability progress change;
  product and cross-capability architecture truth remain unchanged until the
  complete analyzer path is available.

## Blocked by

None - can start immediately.

## User stories addressed

The Rust child PRD has no standalone user-story IDs. This slice covers
`RS-FR-001`, `RS-FR-005`, and `SC-RS-001`, `SC-RS-005`.

## Scenario traceability and verification plan

- `RS-FR-001`, `RS-FR-005`, `SC-RS-001`, and `SC-RS-005` are covered by the
  analyzer manifest, project-selection, CLI-option, safety, and host tests in
  `internal/rustanalyzer` and `cmd/arch-view`.
- Focused verification: `go test ./internal/rustanalyzer ./cmd/arch-view
  -count=1`.

## Implementation result

Implemented the Rust analyzer manifest and host registration, Cargo project and
workspace selection, safe crate selectors, Rust CLI/open options, and the
data-only manifest boundary. Cargo, rustc, build scripts, proc macros, and
target code are not invoked.

## Verification record

Acceptance criteria are complete. Focused and repository verification passed:
`go test ./internal/rustanalyzer ./cmd/arch-view -count=1`, `go test ./...
-count=1`, `go test -race ./...`, `go vet ./...`, `go build ./...`,
`staticcheck ./...`, `golangci-lint run`, and strict OKF validation.
