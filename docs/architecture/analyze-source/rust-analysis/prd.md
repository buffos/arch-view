# Rust analysis PRD

## Purpose

Provide static architecture discovery for one selected Rust crate while making Cargo workspace, feature, macro, and conditional-compilation uncertainty visible.

## Scope and boundary

Read `Cargo.toml` and Rust source, discover crate/module hierarchy, extract `mod`, `use`, `pub use`, and declared dependency observations, and emit common evidence/diagnostics. Do not run Cargo, rustc, build scripts, proc macros, or target code.

`Cargo.toml` is the project marker. A workspace with multiple crates requires explicit `--crate`; a run never silently merges all workspace members.

Defaults exclude tests, examples/benches unless explicitly selected, target/build/cache/generated files, `.git`, and directories named `external`.

## Functional requirements

| ID | Requirement |
|---|---|
| RS-FR-001 | Detect and select one Cargo crate. |
| RS-FR-002 | Discover crate and module structure from manifests/source. |
| RS-FR-003 | Emit static `mod`, `use`, `pub use`, and dependency observations with evidence. |
| RS-FR-004 | Retain external crates, macros, generated code, and cfg uncertainty as metadata/references/diagnostics. |
| RS-FR-005 | Honor explicit crate/features/target/test options without executing build tooling. |

## Non-goals

Macro expansion, build-script execution, complete cfg evaluation, symbol/call graphs, and automatic workspace composition.
