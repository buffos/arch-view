# TypeScript analysis domain glossary

| Term | Definition |
|---|---|
| TypeScript project | One selected `tsconfig` plus effective inherited configuration and source set. |
| Module | TypeScript/TSX source unit represented as a graph node. |
| Project reference | Configured relationship to another TS project; it does not automatically merge runs. |
| Alias | `baseUrl`/`paths`-configured module name mapping. |
| Import/export observation | Static dependency evidence including import/re-export kind. |
| Runtime context | ESM/CJS/package-export metadata affecting resolution. |
| Dynamic loading | Computed or unresolved loading retained as reference/diagnostic. |
| Generated output | `outDir`, build/cache, or generated source excluded by default. |

An import path is not a module identity until the effective TypeScript resolution rules prove its target.
