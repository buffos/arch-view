# TypeScript analysis canonical domain model

## TypeScriptProject aggregate

Fields: `project_root`, `config_path`, `extends_chain[]`, `compiler_options`, `root_dirs[]`, `include/exclude`, `module_observations[]`, `import_observations[]`.

### TypeScriptModuleObservation

`module_id`, `relative_path`, `extension`, `hierarchy[]`, `source_reference_ids[]`, `tags[]`, `metadata`.

### TypeScriptImportObservation

`from_module_id`, `specifier`, `kind` (`import`, `export`, `reexport`, `type_import`, `require`, `dynamic_import`), `resolved_module_id?`, `reference_id?`, `alias?`, `source_reference_ids[]`, `confidence`.

## Invariants

- One selected config determines the source set and resolution context.
- `extends` is resolved without running scripts.
- A local relationship requires a target inside the selected project boundary.
- A literal dynamic import may be resolved statically; computed loading remains dynamic.
- `node_modules` and generated output do not become local nodes by default.

## Extension points

Project references, package export maps, JavaScript inclusion, bundler-specific resolvers, type/symbol edges, and alternate runtime contexts are explicit capabilities/options.
