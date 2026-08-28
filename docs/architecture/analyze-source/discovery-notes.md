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

The current v1 analyzer capability starts with a repository or project root and
analysis options. It selects one language analyzer and returns:

- Project metadata and analyzer identity.
- Package or module nodes.
- Static dependency relationships.
- Source references, including line and column when available.
- Module metadata and tags.
- Diagnostics and confidence for unresolved or dynamic relationships.

The capability stops before graph layout, scene construction, rendering, and export.

The confirmed future direction adds a set of analyzer jobs for mixed-language
or nested projects. The host will be able to use compiled external analyzer
executables, persist project-relative analyzer assignments, and expose combined
or per-analyzer scopes in the application. These behaviors are future child
capabilities of the plugin runtime and are not part of the current v1 contract.

## Actors and inputs

- A developer or CI job starts analysis through the CLI.
- The host supplies the repository root, optional language override, include and exclude settings, and analysis mode.
- The plugin manager currently detects or selects one project and analyzer; the
  future orchestration frontier plans multiple project/analyzer jobs.
- The analyzer reads source files and project configuration in read-only mode.

## Confirmed product decisions

1. One language and project per current v1 run; multiple analyzer jobs are a
   confirmed future target.
2. Package or module is the default graph node. Files are attached evidence.
3. Static dependency relationships are the first supported relation.
4. Project-local modules are shown by default. External dependencies are metadata or diagnostics.
5. Tests, generated code, vendor directories, caches, build outputs, and the upstream reference repository are not project inputs and are excluded from product analysis by default.
6. Unresolved dependencies produce partial results plus diagnostics.
7. Relationship evidence includes source file and parser-provided line and column when available.
8. Analyzer selection supports auto-detection and explicit override for the
   current v1 run; future configuration can assign multiple analyzer/project
   scopes.
9. Each analyzer owns project-boundary rules.
10. Analyzer manifests declare identity, version, language, detection markers, capabilities, and options.
11. Built-in Go interfaces came first. External analyzers use versioned NDJSON
    and JSON Schema; the future distribution target is compiled executables
    that reuse the same analyzer implementations.
12. The host validates, normalizes, sorts, and deduplicates analyzer output.
13. Analysis never executes the target application or arbitrary project code.

## Child territories

The following analyzer/plugin territories are now specified and have their own
discovery notes, exact-spec artifacts, and readiness reviews:

- [Analyzer plugin runtime](plugin-runtime/discovery-notes.md)
- [Go analysis](go-analysis/discovery-notes.md)
- [Python analysis](python-analysis/discovery-notes.md)
- [TypeScript analysis](typescript-analysis/discovery-notes.md)
- [Rust analysis](rust-analysis/discovery-notes.md)
- [Clojure compatibility](clojure-compatibility/discovery-notes.md)
- [Compiled external analyzer distribution](plugin-runtime/compiled-external-analyzer-distribution/discovery-notes.md)
- [Multi-analyzer project orchestration](plugin-runtime/multi-analyzer-orchestration/discovery-notes.md)
- [Project analyzer assignments and view selection](plugin-runtime/project-analyzer-assignments/discovery-notes.md)

## Implementation and verification focus

The parent exact-spec set is complete. Remaining work is implementation and
verification of language-specific parser fixtures, analyzer-boundary behavior,
large-repository limits, and compatibility with the specified plugin-runtime,
aggregate, assignment, and viewer-scope contracts.

## References

- [Reference tool repository](https://github.com/unclebob/arch-view)
- [Reference dependency extractor](https://github.com/unclebob/arch-view)
- [Parent planning concept](../../../.okf/capabilities/analyze-source.md)
