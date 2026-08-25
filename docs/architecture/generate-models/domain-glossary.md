# Generate architecture models domain glossary

| Term | Category | Canonical definition | Important distinction |
|---|---|---|---|
| Canonical model | domain object | Normalized language-neutral representation consumed by all downstream capabilities. | Not an analyzer's raw observation set or a renderer scene. |
| Module | domain object | Stable project-scoped architecture node. | Not a file or hierarchy group. |
| Hierarchy path | value object | Ordered structural segments used for nesting and drill-down. | Not a dependency edge. |
| Relationship | domain object | Typed semantic association between a module and a module/reference. | Not a visual line alone. |
| Reference | domain object | Non-local/external/unresolved target retained without a local module node. | Not a project module. |
| Evidence | value object | Source references supporting a module or relationship. | Aggregation must not erase it. |
| Projection | domain object | Derived view of the canonical model for cycles, layers, aggregation, or hierarchy. | Must not rewrite canonical meaning. |
| Cycle group | derived object | Strongly connected modules or a self-loop under selected relation types. | Not a layout error. |
| Feedback relation | derived object | Relation temporarily removed from a layering graph to make it acyclic. | Canonical relation remains. |
| Layer plan | derived object | Deterministic module levels derived from a relation graph and algorithm version. | Not a stable module property. |
| Aggregation | workflow | Combines modules/relations for a hierarchy view while retaining counts and contributors. | Not data loss. |
| Partial model | status | Usable model containing recoverable diagnostics. | Not failed. |

## Status vocabulary

Model status is `complete`, `partial`, or `failed`. Diagnostic severity is `info`, `warning`, or `error`; a recoverable error may coexist with `partial`.
