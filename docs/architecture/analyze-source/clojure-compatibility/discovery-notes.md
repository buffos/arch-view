# Clojure compatibility discovery notes

## Purpose

Preserve the useful architecture-discovery behavior of the Clojure reference analyzer without making Clojure namespace assumptions part of the shared model.

## Target boundary

The adapter discovers Clojure-family source files, reads namespace declarations and static dependency forms, records source evidence, and emits language-specific abstraction/polymorphism metadata. It does not evaluate forms, require namespaces, assign layers, render graphics, or export files.

## Confirmed decisions

- Support Clojure-family source extensions, including `.clj`, `.cljs`, and `.cljc`, through one analyzer with language/flavor metadata.
- Project boundaries come from configured source paths and common project configuration such as `deps.edn`; ambiguous roots require explicit configuration.
- A namespace is the default module node; source files remain attached evidence. Namespace names are converted to structured hierarchy by the analyzer, not split by the core model.
- Static `:require`, `:use`, and related dependency declarations produce typed observations. Aliases and platform conditionals remain metadata/evidence.
- Missing or malformed `ns` declarations, reader conditionals, dynamic loading, and macro-driven behavior produce diagnostics and partial results.
- Selected polymorphic/abstract definitions may be emitted as tags/metadata with evidence, preserving the reference idea without forcing `abstract` or `direct` into the core edge type.
- The adapter reads forms/configuration but never evaluates project code or loads namespaces.
- The `external/` folder remains read-only reference material and is excluded from product analysis by default.

## Open questions for exact specification

- Supported project configuration formats and source-path precedence.
- Reader-conditional/platform selection and alias/refer semantics.
- Exact static forms that create dependency observations and polymorphism tags.
- Parser choice, malformed-form recovery, and source-location precision.
