# TypeScript analysis acceptance scenarios

## SC-TS-001 — Select a configured project

Given one `tsconfig.json` with an `extends` chain, when analysis runs, then the effective config and configured source set are used without running a compiler or script.

Verification: backend-boundary `when-supported`; frontend-integration `not-applicable`; end-to-end `when-supported`.

## SC-TS-002 — Resolve aliases

Given `baseUrl`/`paths` aliases that resolve inside the project, when analysis runs, then local dependency observations point to the target modules with alias provenance.

Verification: backend-boundary `when-supported`; frontend-integration `not-applicable`; end-to-end `when-supported`.

## SC-TS-003 — Preserve import/export kinds

Given imports, type imports, exports, and re-exports, when analysis runs, then observations retain their kind and source locations while sharing common dependency semantics.

Verification: backend-boundary `when-supported`; frontend-integration `not-applicable`; end-to-end `when-supported`.

## SC-TS-004 — Report dynamic loading

Given a computed dynamic import or bundler-only alias, when analysis runs, then the result is partial/diagnostic as appropriate and no fabricated local edge is emitted.

Verification: backend-boundary `when-supported`; frontend-integration `not-applicable`; end-to-end `when-supported`.

## SC-TS-005 — Exclude generated dependencies

Given `node_modules`, outDir, tests, and build caches, when defaults apply, then those files are excluded unless explicit options enable an allowed category.

Verification: backend-boundary `when-supported`; frontend-integration `not-applicable`; end-to-end `when-supported`.
