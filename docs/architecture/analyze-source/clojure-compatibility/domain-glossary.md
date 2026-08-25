# Clojure compatibility domain glossary

| Term | Definition |
|---|---|
| Clojure-family source | `.clj`, `.cljs`, or `.cljc` source analyzed by the adapter. |
| Namespace | Clojure module-level architecture node declared by `ns`. |
| Namespace form | First declaration that establishes name and static dependencies. |
| Require dependency | Static dependency from `:require` or equivalent namespace clause. |
| Macro dependency | Static macro dependency retained with `kind=macro`. |
| Reader conditional | Platform-specific `.cljc` branch metadata. |
| Polymorphic metadata | Tag/evidence for `defprotocol` or `defmulti`; not a core relation type. |
| Source path | Configured directory containing analyzable source. |

Namespace hierarchy is structured by the adapter and passed to the core model; the core never assumes dot-separated names.
