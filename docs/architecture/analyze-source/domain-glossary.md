# Analyze source code domain glossary

| Term | Category | Canonical definition | Aliases/discouraged usage |
|---|---|---|---|
| Analysis run | workflow/object | One bounded execution that analyzes one project with one effective analyzer. | Avoid “scan” when referring to the whole run. |
| Project root | value object | Repository path supplied as the analysis boundary. | Repository path; use `project root` in contracts. |
| Analyzer | capability actor | Language-specific component that reads configuration/source and emits observations. | Parser is narrower; plugin is deployment/runtime terminology. |
| Analyzer host | actor/service | Component that selects, configures, validates, and normalizes analyzers. | Plugin manager is an implementation term for part of the host. |
| Module | domain object | Project-level architecture unit such as a Go package, Python module, TS module, Rust crate/module, or Clojure namespace. | Component/package/namespace are language-specific descriptions, not interchangeable core names. |
| Relationship observation | domain object | Analyzer evidence that one module has a typed static relationship to another module or non-local reference. | Edge is a graph projection term. |
| Evidence | domain object | Source/configuration location supporting a module, relationship, or diagnostic. | Provenance may include non-location origin metadata. |
| Diagnostic | domain object | Structured information about an unsupported, unresolved, invalid, or noteworthy condition. | Error is only one severity. |
| Partial result | status | Usable analysis result containing recoverable diagnostics or unresolved observations. | Do not call it failed. |
| Project-local | policy | Belongs inside the selected project boundary and is eligible for a module node. | Internal dependency. |
| Non-local reference | domain object | External, standard-library, unresolved, or dynamic target retained without a local module node. | Dependency stub; use `reference`. |
| Static analysis | policy | Source/configuration interpretation without running target application code. | Runtime tracing is explicitly different. |

## Critical distinctions

- A module is not a source file; one module may have many files.
- A relationship observation is not yet a canonical graph edge; normalization owns deduplication and IDs.
- A diagnostic is not necessarily fatal.
- A project boundary is not automatically a monorepo/workspace composition.
