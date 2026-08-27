# 025 — Rust relationships, uncertainty, and end-to-end output

Execution type: AFK
Review gate: none
Status: done

## Parent PRD

`docs/architecture/analyze-source/rust-analysis/prd.md`

## What to build

Complete the Rust repository-to-model journey on top of the Cargo boundary and
module tree. Emit local `use`/`pub use` relationships, declared dependency
observations, non-local references, source evidence, confidence, and
recoverable diagnostics for unresolved paths, cfg conditions, macros, and
generated behavior. Register the finished adapter through the existing CLI and
prove that the unchanged canonical model/viewer/export path can consume its
language-neutral result.

## Acceptance criteria

- [x] Local `use` and `pub use` paths, including relative paths and grouped
  imports where statically provable, resolve to selected crate/module IDs with
  `depends_on` observations, relation-kind metadata, aliases/re-export
  metadata, confidence, and source locations; structural `mod` containment
  remains represented by hierarchy rather than being mistaken for a semantic
  edge.
- [x] Cargo dependencies emit declared dependency observations and references;
  standard-library crates, external crates, workspace/path dependencies, and
  unresolved local paths retain distinct scopes and metadata without merging
  unselected workspace crates into the local module graph.
- [x] Cfg-dependent paths, proc-macro/attribute uncertainty, macro-generated
  source, missing module files, and unresolved imports produce visible
  recoverable diagnostics/references and never create fabricated local edges.
- [x] The result honors Rust options, includes analyzer provenance and
  deterministic IDs/order, returns `complete` only without recoverable issues,
  returns `partial` for usable uncertain results, and passes common analysis
  result validation.
- [x] A representative Rust fixture runs through CLI `analyze` and canonical
  `model normalize`, `model validate`, and `model projection`; the existing
  language-neutral viewer/export path requires no Rust-specific branch.
- [x] Repeated analysis, canonical normalization, and supported JSON/HTML/SVG
  export runs are byte-stable, and a build script or target source with side
  effects is never executed.

## Artifact sync required

- Application PRD: `required: docs/prd.md` — record Rust as an available
  in-process analyzer and update capability maturity/current implementation
  sequence without widening the confirmed product boundary.
- Application architecture summary: `required: docs/architecture/application-architecture-summary.md` — record the Rust adapter's completed path through the existing host, canonical model, viewer, and export boundaries.
- Owning capability node/artifacts: `required: .okf/capabilities/analyze-source.md; .okf/capabilities/analyze-source/rust-analysis.md; docs/architecture/analyze-source/orchestration-status.md; docs/architecture/analyze-source/rust-analysis/orchestration-status.md; docs/architecture/analyze-source/rust-analysis/implementation-slice.md`.
- Issue registry: `required`; node `issues:` reference: `required` until the
  issue is archived at closeout.
- Reason/no-impact decision: the completed end-to-end adapter changes product
  maturity and documented implementation sequencing, while preserving the
  language-neutral analyzer/model/viewer/export contracts.

## Blocked by

Blocked by `docs/agents/issues/done/20260827-023-rust-cargo-boundary-and-registration.md` and `docs/agents/issues/done/20260827-024-rust-module-discovery-and-evidence.md`; both issues were completed and archived before this slice.

## User stories addressed

The Rust child PRD has no standalone user-story IDs. This slice covers
`RS-FR-003` through `RS-FR-005` and `SC-RS-003` through `SC-RS-005`, plus the
parent analysis scenarios `SC-AS-001`, `SC-AS-002`, `SC-AS-004`,
`SC-AS-005`, `SC-AS-006`, and `SC-AS-008`.

## Scenario traceability and verification plan

- `RS-FR-003` through `RS-FR-005`, `SC-RS-003` through `SC-RS-005`, and the
  listed parent analysis scenarios are covered by relationship, uncertainty,
  canonical normalization, CLI, and export-path tests.
- Focused verification: `go test ./internal/rustanalyzer ./cmd/arch-view
  -count=1`, plus the canonical model and JavaScript route checks.

## Implementation result

Implemented deterministic Rust `use`, `pub use`, and Cargo-dependency
observations with local/external scope, aliases, re-exports, confidence,
source evidence, cfg/macro/generated uncertainty, partial-result handling,
canonical normalization, CLI registration, and the unchanged language-neutral
viewer/export path. Structural `mod` containment remains hierarchy metadata,
not a semantic dependency edge.

## Verification record

Acceptance criteria are complete. Focused and repository verification passed:
`go test ./internal/rustanalyzer ./cmd/arch-view -count=1`, `go test ./...
-count=1`, `go test -race ./...`, `go vet ./...`, `go build ./...`,
`staticcheck ./...`, `golangci-lint run`, Node syntax/route/layout checks,
`git diff --check`, and strict OKF validation.
