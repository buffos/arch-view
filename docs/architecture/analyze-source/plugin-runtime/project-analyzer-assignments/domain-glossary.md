# Project analyzer assignments and view selection domain glossary

| Term | Category | Definition | Distinction |
|---|---|---|---|
| Analysis configuration | business object | The `analysis` section of the nearest `.archview.json` that controls project-to-analyzer assignments. | It is separate from the `layout` section and never changes model facts. |
| Assignment | business object | One mapping from a repository-relative path to one stable logical analyzer ID and optional non-sensitive options. | It is not a deployment descriptor or a language parser rule. |
| Assignment path | value object | Normalized repository-relative POSIX path, using `.` for the repository root. | Absolute paths, parent traversal, and globs are invalid. |
| Matching assignment | policy result | The assignment whose path is an ancestor of a discovered project root; the deepest match wins. | It is resolved within one nearest complete configuration file. |
| Automatic detection | workflow/action | Analyzer selection based on discovered project markers when no higher-precedence selection applies. | It is a fallback, not a reason to bypass invalid nearest configuration. |
| Configuration precedence | policy/rule | Ordered choice among CLI selection, deepest assignment, and automatic detection. | It determines selection; it does not merge multiple config files. |
| Analysis scope | business object | One selectable project-root/analyzer result in a combined run. | `All` is a projection over scopes, not another analyzer job. |
| Scope selector | workflow/action | Viewer control that chooses `All` or one cached analysis scope. | It filters existing results and never starts analysis. |
| Effective analyzer options | value object | Analyzer defaults overridden by assignment options and then CLI options. | Layout options are not part of this value. |
| Analysis cache entry | business object | Session-scoped result keyed by scope, analyzer package, options, discovery policy, and source fingerprint. | It is invalid when any input in its key changes. |
| Assignment diagnostic | reporting term | A scoped explanation that an assignment cannot produce a usable analyzer job or result. | It is distinct from a malformed configuration error. |

Canonical configuration statuses: `valid`, `invalid`, `active`, `stale`,
`unavailable`.
