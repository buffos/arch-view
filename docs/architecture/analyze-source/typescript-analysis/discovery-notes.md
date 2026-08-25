# TypeScript analysis discovery notes

## Purpose

Discover TypeScript module architecture while preserving the configuration-driven nature of modern module resolution.

## Target boundary

The analyzer reads TypeScript project configuration and source syntax, discovers configured modules, resolves static imports/exports and path aliases where possible, and returns evidence, metadata, confidence, and diagnostics. It stops before graph normalization, rendering, and export and does not run package scripts or application code.

## Confirmed decisions

- `tsconfig.json` and its `extends` chain are the primary project-boundary inputs; `package.json` and project references contribute resolution context.
- One configured project is analyzed per run. Multiple ambiguous configs require explicit selection.
- TypeScript module is the default node. Import/export relationships are represented as typed static dependencies; alias resolution is retained as evidence/metadata.
- The analyzer understands configured `baseUrl`, `paths`, root directories, package boundaries, and relevant ESM/CJS resolution context without assuming one runtime.
- Dynamic `import()`, computed paths, bundler-only aliases, and unresolved package exports produce diagnostics/confidence rather than fabricated edges.
- Tests, generated output, `node_modules`, build/cache directories, `.git`, and `external/` are excluded by default. JavaScript is included only when explicitly enabled by project configuration/options.
- Syntax/configuration parsing is static. Running `tsc`, bundler scripts, or arbitrary package hooks is out of scope for the first slice.

## Open questions for exact specification

- Config selection and `extends`/project-reference traversal rules.
- ESM/CJS/package-exports resolution matrix and JavaScript inclusion policy.
- Export/re-export edge semantics and confidence for bundler-specific behavior.
- Parser/compiler API choice and diagnostics for unsupported syntax/configuration.
