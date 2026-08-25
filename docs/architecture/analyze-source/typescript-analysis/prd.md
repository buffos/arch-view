# TypeScript analysis PRD

## Purpose

Discover TypeScript project/module architecture while preserving configuration-driven resolution and runtime uncertainty.

## Scope and boundary

Read one selected `tsconfig.json` project and eligible TypeScript source, resolve static import/export relationships and configured aliases, and emit common observations/evidence/diagnostics. Do not run `tsc`, bundlers, package scripts, or target code.

`tsconfig.json` plus its `extends` chain is the primary boundary. Multiple candidate configs require explicit selection. `.ts` and `.tsx` are included by default; `.js/.jsx` require `allowJs` or `include_js=true`.

Defaults exclude test/spec files, generated/outDir/build/cache output, `node_modules`, `.git`, and `external/`.

## Functional requirements

| ID | Requirement |
|---|---|
| TS-FR-001 | Select one unambiguous TypeScript project and resolve config inheritance. |
| TS-FR-002 | Discover configured modules and explicit hierarchy. |
| TS-FR-003 | Resolve static imports, exports, aliases, and supported package exports. |
| TS-FR-004 | Represent dynamic/computed/unresolved loading as references/diagnostics. |
| TS-FR-005 | Preserve import kind, source locations, and deterministic output. |

## Non-goals

Bundler semantics that cannot be proven from config, runtime module loading, compiler/build execution, call graphs, and package-script execution.
