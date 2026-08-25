# Rust analysis canonical use cases

## RustAnalyzerService

- `DetectCargoProject`
- `SelectRustCrate`
- `ResolveRustSourceTree`
- `ExtractRustModuleAndUseObservations`
- `ClassifyRustConditionalTargets`
- `EmitRustAnalysisResult`

## Orchestration

Read manifest/workspace → select crate → apply feature/target/test scope → parse crate root and module declarations → extract `use`/`pub use`/dependency paths → resolve proven local targets → report external/macro/cfg uncertainty → emit common result.

Failures: invalid manifest, ambiguous crate, missing source, unsupported syntax, unresolved path, macro/generated behavior, cfg uncertainty. Recoverable cases produce partial status.

No Cargo/rustc/build-script execution occurs.
