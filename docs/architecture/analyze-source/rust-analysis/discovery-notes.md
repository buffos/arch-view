# Rust analysis discovery notes

## Purpose

Discover Rust crate and module architecture while keeping Cargo workspace composition explicit and preserving unresolved macro/generated behavior as diagnostics.

## Target boundary

The analyzer reads Cargo manifests and Rust source syntax, discovers one selected crate/module tree, resolves local `mod`, `use`, and dependency relationships where static evidence permits, and returns evidence and diagnostics. It does not run Cargo, rustc, build scripts, proc macros, or the target application in the first slice.

## Confirmed decisions

- `Cargo.toml` is the primary project marker. A workspace with multiple crates requires explicit crate selection for the one-project-per-run policy.
- Crate and module are the natural Rust nodes; files remain evidence. Local `use`/module relationships produce static dependencies, while external crates are metadata or non-local references by default.
- `mod`, inline modules, `use`, `pub use`, and declared dependencies are distinct observations so future relation types can preserve the difference without changing the canonical model.
- Tests, target/build output, generated files, caches, `.git`, and `external/` are excluded by default.
- Macro expansion, build-script output, conditional compilation, and proc-macro behavior are represented with diagnostics/confidence unless an explicitly enabled read-only resolver can prove them.
- Manifest/source parsing is static and read-only. Tool-assisted Cargo metadata is a later opt-in with timeouts and safety controls.

## Open questions for exact specification

- Workspace/crate selection, feature and target selection, and conditional compilation policy.
- Exact `mod`/`use`/`pub use` relation mapping and crate/module identity algorithm.
- Macro/generated-source diagnostics and optional tool-assisted resolution.
- Handling of examples, benches, integration tests, and build-script metadata.
