# Clojure compatibility canonical domain model

## ClojureProject aggregate

Fields: `project_root`, `flavors[]`, `source_roots[]`, `platform` (`clj|cljs|both`), `namespace_observations[]`, `dependency_observations[]`.

### NamespaceObservation

`module_id = "clj:" + namespace`, `namespace`, `flavor`, `hierarchy[]`, `source_reference_ids[]`, `tags[]`, `metadata`.

### NamespaceDependencyObservation

`from_module_id`, `to_namespace`, `kind` (`require`, `use`, `macro`, `platform_conditional`), `alias?`, `refered_symbols[]`, `platforms[]`, `source_reference_ids[]`, `confidence`.

## Invariants

- A source file without a valid namespace form yields a diagnostic and cannot silently receive a guessed namespace.
- `ns` dependencies are static observations; no form is evaluated.
- `.cljc` platform branches retain platform metadata; `platform=both` does not pretend both branches were runtime-tested.
- `defprotocol` and `defmulti` produce `polymorphic` tags only when their forms are statically recognized.
- Namespace identity and hierarchy are adapter output, not core string splitting.

## Extension points

Additional project configs, reader-condition platforms, macro metadata, and symbol relations can be added without changing core module/relationship semantics.
