# Rust analysis domain glossary

| Term | Definition |
|---|---|
| Cargo crate | One selected Rust package boundary described by `Cargo.toml`. |
| Cargo workspace | Collection of crates; selection is explicit for one run. |
| Rust module | Inline or file-backed module within a crate. |
| `mod` observation | Static module containment/discovery evidence. |
| `use` observation | Static name/path dependency evidence. |
| `pub use` | Public re-export observation, retained as metadata. |
| cfg condition | Conditional compilation expression that may affect visibility. |
| Macro/generated behavior | Source behavior not fully known without expansion/build tooling. |
| Feature view | Explicit Cargo feature selection used for analysis metadata. |

Containment (`mod`) is structural; `use`/dependency observations become semantic relationships after normalization.
