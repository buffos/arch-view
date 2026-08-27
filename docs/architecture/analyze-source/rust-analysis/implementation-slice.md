# Rust repository to language-neutral architecture model

## Selected frontier

The approved implementation frontier was the specified [Rust analysis capability](../../../../.okf/capabilities/analyze-source/rust-analysis.md), using the existing [analyzer host and plugin contract](../canonical-api-cli-contract.md); it is now implemented.

The Rust exact-spec set is complete and readiness-reviewed. The implementation
is split into three AFK delivery slices so each change is independently
verifiable while preserving the common analysis → canonical model → viewer /
export path.

The implementation keeps `mod` declarations structural: declaration evidence
is attached to the containing/child module metadata and hierarchy, while only
`use`, `pub use`, and Cargo dependency observations become semantic
`depends_on` relationships.

## Vertical outcome

Given a Rust repository containing one selected Cargo crate, a developer can:

1. detect or explicitly select the Rust analyzer and crate;
2. discover crate/module hierarchy and source evidence without executing Cargo;
3. inspect local and non-local relationships with cfg/macro uncertainty visible;
4. normalize the result through the existing language-neutral model; and
5. use the existing CLI, viewer, and deterministic export surfaces without
   Rust-specific presentation code.

## Ordered delivery slices

| Issue | Outcome | Owner | Blocked by | Review gate | State |
|---|---|---|---|---|---|
| [023](../../../agents/issues/done/20260827-023-rust-cargo-boundary-and-registration.md) | Cargo boundary, crate selection, Rust manifest/options, host and CLI registration | Rust analysis + plugin runtime | None | none | done |
| [024](../../../agents/issues/done/20260827-024-rust-module-discovery-and-evidence.md) | Reachable crate/module hierarchy, scope filtering, cfg metadata, and source evidence | Rust analysis | 023 complete | none | done |
| [025](../../../agents/issues/done/20260827-025-rust-relationships-and-end-to-end-output.md) | Local/non-local relationships, uncertainty, partial results, canonical model and public output path | Rust analysis + Generate models | 023, 024 complete | none | done |

## Explicit non-goals

- Cargo metadata/build execution, rustc, build scripts, proc-macro expansion,
  or target application execution.
- Complete cfg evaluation, symbol/type/call graphs, or automatic workspace
  composition.
- Rust-specific model, viewer, layout, or export schemas.

## Verification surfaces

- Backend boundary: manifest/selection/options, module discovery, relationship
  resolution, diagnostics, evidence, safety, deterministic output, and common
  result validation.
- Frontend integration: not applicable to the Rust adapter; the existing
  language-neutral viewer path is covered by the end-to-end model smoke test.
- End-to-end: Rust repository → analysis JSON → canonical model validation and
  projection → existing viewer/export contracts.
- Repository/OKF integrity: `git diff --check`, full Go tests, static checks
  available in the repository, synchronized artifact references, and untouched
  upstream reference boundary.

## Artifact impact

Issues 023–025 changed delivery and Rust capability truth while preserving the
specified analyzer boundary, product scope, and verification policy. Issue 025
completed the required refresh of the application PRD and architecture summary
when the end-to-end adapter became implemented. No Rust-specific model, viewer,
layout, or export schema was introduced.

## Completion record

Issues 023–025 were accepted and archived on 2026-08-27. The Rust capability
transitioned from `specified` to `implemented` after strict OKF validation and
repository verification. No frontend visual-review gate applied to this
backend/CLI adapter; the existing language-neutral viewer/export path remained
unchanged.
