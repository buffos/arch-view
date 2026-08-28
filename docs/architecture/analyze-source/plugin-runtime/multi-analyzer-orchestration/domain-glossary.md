# Multi-analyzer project orchestration domain glossary

| Term | Category | Definition | Distinction |
|---|---|---|---|
| Opened repository | business object | The normalized root directory the user asked Arch View to analyze. | It is the discovery boundary, not necessarily one language project. |
| Project root | business object | A repository-relative directory owned by one analyzer job. | It can be nested inside another project root. |
| Strong manifest marker | policy/metadata | An analyzer-declared project-boundary marker that can establish a project root. | A weak source marker can support detection but cannot create an automatic nested root. |
| Analyzer job | business object | One planned execution for one project root and one logical analyzer ID. | It is the unit of concurrency, cancellation, caching, status, and provenance. |
| Job plan | business object | Deterministically ordered set of analyzer jobs plus discovery diagnostics. | It is created before execution and is not itself an analysis result. |
| Scope ID | value object | Stable identifier derived from repository-relative project root and logical analyzer ID. | It namespaces observations; it is not a UI label. |
| Combined analysis | business object | One aggregate result assembled from independent job results. | It retains per-scope provenance and never invents cross-scope relationships. |
| Scope result | business object | One job's normalized result, status, counts, and diagnostics within a combined analysis. | A failed scope can remain visible as diagnostics without contributing graph nodes. |
| Usable result | policy term | A complete or partial job result whose model observations pass normalization sufficiently to contribute. | A job with only a failure diagnostic is not usable. |
| Partial analysis | state/status | Aggregate status with at least one usable job and at least one non-usable or partial job. | It is different from a fully failed run. |
| Provenance | reporting term | Scope, analyzer, version, runtime source, and local observation origin retained with aggregate data. | It explains where data came from; it does not change relationship semantics. |
| Worker pool | technical policy | Bounded scheduler capacity for concurrent analyzer jobs. | Its size limits execution resources, not the number of discovered scopes. |
| Discovery exclusion | policy/rule | Directory or path excluded from automatic root traversal. | It prevents accidental build/cache/vendor analysis and is not a language dependency rule. |

Canonical job statuses: `planned`, `queued`, `running`, `complete`, `partial`,
`failed`, `cancelled`, `skipped`.
