# Rust analysis acceptance scenarios

## SC-RS-001 — Select one crate

Given a Cargo workspace with multiple crates, when no crate is selected, then analysis returns an ambiguity diagnostic and does not merge the workspace silently.

Verification: backend-boundary `when-supported`; frontend-integration `not-applicable`; end-to-end `when-supported`.

## SC-RS-002 — Discover modules

Given a crate root with file-backed and inline modules, when analysis runs, then module hierarchy and source evidence are returned.

Verification: backend-boundary `when-supported`; frontend-integration `not-applicable`; end-to-end `when-supported`.

## SC-RS-003 — Emit use relationships

Given local `use` and `pub use` paths, when analysis runs, then proven local observations carry relation-kind metadata and source locations.

Verification: backend-boundary `when-supported`; frontend-integration `not-applicable`; end-to-end `when-supported`.

## SC-RS-004 — Report macro/cfg uncertainty

Given proc-macro/generated or cfg-dependent behavior that static parsing cannot prove, when analysis runs, then uncertainty is visible through references/diagnostics and no fabricated local edge is emitted.

Verification: backend-boundary `when-supported`; frontend-integration `not-applicable`; end-to-end `when-supported`.

## SC-RS-005 — Never execute build tooling

Given a build script with side effects, when analysis runs, then Cargo/rustc/build scripts are not executed.

Verification: backend-boundary `when-supported`; frontend-integration `not-applicable`; end-to-end `when-supported`.
