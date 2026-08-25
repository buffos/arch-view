# Rust analysis requirements gap analysis

## Resolved

| Area | Resolution |
|---|---|
| Boundary | One selected Cargo crate per run; workspace ambiguity requires selection. |
| Node | Crate/module with attached files. |
| Edge | Static module/use/dependency observations normalized as typed dependencies. |
| Generated behavior | Macro/build/conditional uncertainty becomes diagnostics/confidence. |
| Defaults | Exclude tests, target/build/cache/generated files, `.git`, and `external/`. |
| Safety | Read manifests/source; do not execute Cargo, rustc, build scripts, or application code initially. |

## Specification closure and residual risks

Crate selection, feature/target options, module/use relation metadata, macro/cfg diagnostics, and static safety are defined in the linked exact-spec artifacts. Macro/cfg fixture breadth remains an implementation risk.
