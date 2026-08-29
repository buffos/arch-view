# Project analyzer assignments and view selection domain glossary

| Term | Category | Definition | Distinction |
|---|---|---|---|
| Analysis configuration | business object | The `analysis` section of the nearest `.archview.json` that controls project-to-analyzer assignments and source-scope filters. | It is separate from the `layout` section and never changes model facts. |
| Assignment | business object | One mapping from a repository-relative path to one stable logical analyzer ID and optional non-sensitive options. | It is not a deployment descriptor or a language parser rule. |
| Assignment path | value object | Normalized repository-relative POSIX path, using `.` for the repository root. | Absolute paths, parent traversal, and globs are invalid. |
| Source-scope policy | policy object | The global exclusions and analyzer-scoped include rules that determine which source paths each planned job may analyze. | It filters analyzer input after root discovery; it does not select or create project roots. |
| Exclusion glob | value object | A normalized POSIX glob in `analysis.exclude` that removes matching paths from every analyzer job. | It is additive to fixed safety and nested-root exclusions and cannot be negated by an include. |
| Analyzer include rule | business object | One stable logical analyzer ID paired with a non-empty union of source globs. | It is an allowlist for that analyzer only; it is not an assignment or language selector. |
| Invocation root | value object | The normalized directory supplied as `--project` or the equivalent API repository root. | All persisted source-scope globs are relative to this root, not the config file or process working directory. |
| Source-scope glob | value object | A relative path pattern supporting `*`, `?`, character classes, and recursive `**`. | The v2 subset excludes absolute paths, `..`, comments, and `.gitignore` negation. |
| Matching assignment | policy result | The assignment whose path is an ancestor of a discovered project root; the deepest match wins. | It is resolved within one nearest complete configuration file. |
| Automatic detection | workflow/action | Analyzer selection based on discovered project markers when no higher-precedence selection applies. | It is a fallback, not a reason to bypass invalid nearest configuration. |
| Configuration precedence | policy/rule | Ordered choice among CLI selection, deepest assignment, and automatic detection. | It determines selection; it does not merge multiple config files. |
| Analysis scope | business object | One selectable project-root/analyzer result in a combined run. | `All` is a projection over scopes, not another analyzer job. |
| Scope selector | workflow/action | Viewer control that chooses `All` or one cached analysis scope. | It filters existing results and never starts analysis. |
| Effective analyzer options | value object | Analyzer defaults overridden by assignment options and then CLI options. | Layout options and source-scope policy are not part of this value, though both contribute to job/cache identity. |
| Analysis cache entry | business object | Session-scoped result keyed by scope, analyzer package, options, discovery policy, and source fingerprint. | It is invalid when any input in its key changes. |
| Assignment diagnostic | reporting term | A scoped explanation that an assignment cannot produce a usable analyzer job or result. | It is distinct from a malformed configuration error. |

Canonical configuration statuses: `valid`, `invalid`, `active`, `stale`,
`unavailable`.
