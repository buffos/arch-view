# Go analysis canonical domain model

## GoProject aggregate

`GoProject` contains selected module identity, module root, optional workspace context, effective build tags, and package observations.

### GoModule

`module_path`, `module_root`, `go_version`, `workspace_path?`.

Invariant: exactly one `module_path` is selected per run.

### GoPackageObservation

`module_id = "go:" + module_path + relative_import_path`, `package_name`, `relative_directory`, `source_reference_ids[]`, `tags[]`, `metadata`.

The root package omits the relative suffix. IDs are stable for the same module path and package directory.

### GoImportObservation

`from_module_id`, `import_path`, `target_scope`, `to_module_id?`, `reference_id?`, `source_reference_ids[]`, `build_constraints[]`.

Target scope: `local`, `standard_library`, `external`, `unresolved`, `cgo`, `conditional`.

## Policies and invariants

- Only eligible files under the selected module are scanned.
- Package declarations that disagree inside one directory produce a diagnostic.
- An import path is a local module edge only when its resolved directory is inside the selected module boundary.
- Build-tag-dependent observations carry the tags that made them visible.
- No Go command or target package initialization is required for the first slice.

## Extension points

Optional tool-assisted resolution, type/call observations, workspace composition, and build-matrix views can add metadata/relationship types later.
