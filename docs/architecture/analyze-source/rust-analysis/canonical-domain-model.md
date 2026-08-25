# Rust analysis canonical domain model

## RustProject aggregate

Fields: `manifest_path`, `crate_id`, `crate_name`, `workspace_path?`, `features[]`, `target?`, `module_observations[]`, `use_observations[]`, `dependency_observations[]`.

### RustModuleObservation

`module_id`, `crate_name`, `module_path`, `kind` (`crate`, `file_module`, `inline_module`), `hierarchy[]`, `source_reference_ids[]`, `tags[]`.

### RustDependencyObservation

`from_module_id`, `spelling`, `kind` (`mod`, `use`, `pub_use`, `dependency`), `to_module_id?`, `reference_id?`, `cfg_conditions[]`, `source_reference_ids[]`, `confidence`.

## Invariants

- Exactly one crate is selected.
- `mod` containment is not a dependency edge.
- Local targets must resolve inside the selected crate/workspace selection.
- External crates and unresolved macro/cfg targets never become local nodes without proof.
- Feature/target selection and unresolved cfg conditions are recorded.

## Extension points

Macro expansion, Cargo metadata, workspace composition, symbol/type relations, and richer cfg evaluation are optional future capabilities.
