# Analyze source code discovery notes

## Purpose

Arch View analyzes a repository without executing its application and returns enough structural information to build a navigable architecture model.

## Current reference behavior

The reference tool in the upstream [unclebob/arch-view repository](https://github.com/unclebob/arch-view):

- Scans configured Clojure-family source paths.
- Reads each file's first Clojure form and requires an `ns` declaration.
- Extracts project-local `:require` relationships.
- Records source-file paths for discovered namespaces.
- Marks selected Clojure polymorphic definitions as abstract.
- Passes the resulting graph to cycle detection, layering, projection, and a viewer.

The reference folder is read-only. It is not a product dependency and must remain ignored by Git.

## Target boundary

The analyzer capability starts with a repository or project root and analysis options. It selects one language analyzer and returns:

- Project metadata and analyzer identity.
- Package or module nodes.
- Static dependency relationships.
- Source references, including line and column when available.
- Module metadata and tags.
- Diagnostics and confidence for unresolved or dynamic relationships.

The capability stops before graph layout, scene construction, rendering, and export.

## Actors and inputs

- A developer or CI job starts analysis through the CLI.
- The host supplies the repository root, optional language override, include and exclude settings, and analysis mode.
- The plugin manager detects or selects one project and analyzer.
- The analyzer reads source files and project configuration in read-only mode.

## Confirmed product decisions

1. One language and project per run.
2. Package or module is the default graph node. Files are attached evidence.
3. Static dependency relationships are the first supported relation.
4. Project-local modules are shown by default. External dependencies are metadata or diagnostics.
5. Tests, generated code, vendor directories, caches, build outputs, and the upstream reference repository are not project inputs and are excluded from product analysis by default.
6. Unresolved dependencies produce partial results plus diagnostics.
7. Relationship evidence includes source file and parser-provided line and column when available.
8. Analyzer selection supports auto-detection and explicit override.
9. Each analyzer owns project-boundary rules.
10. Analyzer manifests declare identity, version, language, detection markers, capabilities, and options.
11. Built-in Go interfaces come first. External analyzers use versioned NDJSON and JSON Schema later.
12. The host validates, normalizes, sorts, and deduplicates analyzer output.
13. Analysis never executes the target application or arbitrary project code.

## Child territories

The following analyzer/plugin territories are now bounded and have their own discovery notes and gap analyses:

- [Analyzer plugin runtime](plugin-runtime/discovery-notes.md)
- [Go analysis](go-analysis/discovery-notes.md)
- [Python analysis](python-analysis/discovery-notes.md)
- [TypeScript analysis](typescript-analysis/discovery-notes.md)
- [Rust analysis](rust-analysis/discovery-notes.md)
- [Clojure compatibility](clojure-compatibility/discovery-notes.md)

## Open questions for exact specification

- What is the canonical module and relationship schema?
- Which Go parser and package-resolution strategy is required for the first slice?
- Which manifest fields and capability names are stable in plugin version one?
- How are include and exclude rules represented and overridden?
- What evidence and diagnostic severity levels are required?
- When should tool-assisted, read-only resolution be enabled?

## References

- [Reference tool repository](https://github.com/unclebob/arch-view)
- [Reference dependency extractor](https://github.com/unclebob/arch-view)
- [Parent planning concept](../../../.okf/capabilities/analyze-source.md)
