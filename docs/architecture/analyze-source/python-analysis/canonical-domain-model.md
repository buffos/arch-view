# Python analysis canonical domain model

## PythonProject aggregate

Fields: `project_root`, `configuration_files[]`, `source_roots[]`, `python_version?`, `include_stubs`, `module_observations[]`, `import_observations[]`.

### PythonModuleObservation

`module_id`, `qualified_name`, `kind` (`package` or `module`), `hierarchy[]`, `path`, `source_reference_ids[]`, `tags[]`.

### PythonImportObservation

`from_module_id`, `spelling`, `resolved_module_id?`, `reference_id?`, `kind` (`import`, `from`, `reexport?`, `dynamic`), `source_reference_ids[]`, `confidence`.

## Invariants

- Resolution never depends on importing the project.
- Relative import resolution uses the importing package context.
- A dynamic or conditional target cannot become a local edge without static proof.
- Effective source roots and configuration precedence are recorded.
- Unreadable/invalid syntax produces diagnostics while unrelated modules may remain usable.

## Extension points

Type-checker metadata, environment snapshots, importlib plugins, stubs, and symbol/call observations can be added as optional capabilities.
