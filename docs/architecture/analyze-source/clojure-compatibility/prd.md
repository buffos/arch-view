# Clojure compatibility PRD

## Purpose

Preserve the useful Clojure-family static architecture behavior of the reference while keeping namespace syntax inside a language adapter.

## Scope and boundary

Read `.clj`, `.cljs`, and `.cljc` files plus configured source paths, parse namespace declarations, extract static `:require`, `:use`, and macro dependency forms, attach evidence, and emit polymorphism metadata. Never evaluate forms, require namespaces, or execute project code.

Source-root precedence: explicit options > `deps.edn`/`project.clj`/`shadow-cljs.edn` configuration > `src`; tests are opt-in. The product `external/` directory is excluded by default.

## Functional requirements

| ID | Requirement |
|---|---|
| CL-FR-001 | Detect configured Clojure-family source roots and flavor metadata. |
| CL-FR-002 | Parse the first `ns` form and report missing/malformed declarations. |
| CL-FR-003 | Emit static namespace/macro dependencies with evidence and aliases. |
| CL-FR-004 | Preserve `.cljc` reader-conditional/platform metadata. |
| CL-FR-005 | Emit `polymorphic` metadata for statically recognized `defprotocol` and `defmulti` forms. |
| CL-FR-006 | Return partial results without evaluating target forms. |

## Non-goals

Runtime namespace loading, macro expansion, complete reader-condition evaluation, call graphs, and renderer-specific abstract/direct classifications.
