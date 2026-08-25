# Clojure compatibility requirements gap analysis

## Resolved

| Area | Resolution |
|---|---|
| Boundary | Analyzer-owned source paths and Clojure project configuration. |
| Node | Namespace with attached source files. |
| Edge | Static namespace dependency observations. |
| Compatibility | Preserve `ns`, require/use, source evidence, and abstraction metadata ideas from the reference. |
| Core independence | Namespace syntax stays inside the adapter; the model receives structured hierarchy and generic tags. |
| Safety | Parse/read only; never evaluate forms or load the target project. |

## Specification closure and residual risks

Supported project formats, reader-conditionals, namespace dependency kinds, parser recovery, and polymorphic-tag rules are defined in the linked exact-spec artifacts. Reader-conditional and malformed-form fixture breadth remains an implementation risk.
