# Go analysis discovery notes

## Purpose

Provide the first complete analyzer for Go package architecture without executing the target application.

## Target boundary

The analyzer detects a Go module/project, discovers package directories and source files, resolves static local imports, classifies standard-library and external imports, and returns module/dependency evidence and diagnostics. It stops before canonical graph normalization, layers, rendering, and export.

## Confirmed decisions

- `go.mod` is the primary project boundary. A `go.work` file may provide workspace context, but a run still targets one selected module; multiple candidate modules require explicit selection rather than silently composing a monorepo.
- Package is the default module node; files are attached evidence. Import relationships are `depends_on` edges.
- Normal source files are included; tests, `vendor`, generated files, build output, caches, `.git`, and directories named `external` are excluded by default.
- Build constraints and selected tags are analyzer options. The default result describes the normal configured build view and records the configuration used.
- Local imports resolve against the selected module path and package directories. standard-library, third-party, missing, cgo, and build-conditional cases remain metadata or diagnostics rather than invented local nodes.
- Source evidence carries relative file paths and parser locations when available. Unresolved imports do not fail the complete run.
- The first slice focuses on package dependency graphs, not call graphs, symbol ownership, or runtime behavior.
- Parsing and configuration reading are static. Tool-assisted resolution may be added later behind explicit safe-mode, timeout, and diagnostic controls.

## Open questions for exact specification

- Parser and package-loading implementation, including whether optional Go tooling is used.
- `go.work` selection, build tags, cgo, generated-file detection, and symlink policy.
- Exact standard-library/third-party classification and module identity algorithm.
- Diagnostics and confidence for conditional or unresolved imports.
