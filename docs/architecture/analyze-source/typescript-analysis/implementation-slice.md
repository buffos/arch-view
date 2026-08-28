# TypeScript implementation slice: repository to visible architecture view

## Selected frontier

The [TypeScript analysis capability](../../../../.okf/capabilities/analyze-source/typescript-analysis.md) was the next specified language frontier after the completed Python adapter. The existing analyzer host, canonical model, viewer, and export contracts consumed the language-neutral adapter without a boundary change; the TypeScript slice is now complete.

This slice adds TypeScript behind the existing `analysis.Analyzer` contract. It does not add a new capability node, change the canonical model schema, or move TypeScript-specific behavior into the host, model, viewer, or exporter.

## Current and target truth

- **Observed in code:** the generic analyzer contract, deterministic host selection, canonical normalization, local viewer, and JSON/HTML/SVG paths exist; Go and Python analyzers are registered at the composition root.
- **Inferred from docs:** the TypeScript child contract requires safe `tsconfig` selection/inheritance, configured source scope, module discovery, static import/export resolution, aliases, package-export context, dynamic uncertainty, evidence, diagnostics, and deterministic output.
- **User-confirmed target:** TypeScript is implemented gradually through the same common contract and must reach the existing model/viewer journey without language-specific changes to shared consumers.
- **Mismatch:** the TypeScript contract is specified but no adapter or delivery slice existed before this work.

## Vertical outcome

Given a TypeScript repository, a developer can:

1. select or auto-detect one TypeScript project configuration;
2. resolve its static `extends` chain and effective source scope without running a compiler, bundler, package script, or target code;
3. discover deterministic TypeScript/TSX modules and opt-in JavaScript/JSX modules with source evidence;
4. see proven local imports, exports, aliases, and supported package exports as directed relationships;
5. see unresolved, external, excluded, and computed dynamic loading as references, confidence, and recoverable diagnostics rather than fabricated local edges;
6. normalize the result through the existing language-neutral model and open it in the current viewer; and
7. produce deterministic JSON, HTML, and SVG artifacts through the existing export path.

## Scope

- In-process TypeScript analyzer implementing `analysis.Analyzer`.
- Root/config candidate detection, explicit config selection, JSONC parsing, safe `extends` traversal, package metadata, compiler options, project references, and runtime condition preference.
- Include/exclude, `rootDir`, `rootDirs`, `outDir`, `allowJs`, tests, generated/cache/vendor/external directories, and repository-root safety rules.
- TypeScript and TSX lexical module/import/export discovery, opt-in JavaScript/JSX support, source locations, deterministic module identity, and file evidence.
- Relative, baseUrl/path-alias, rootDirs, self-package, and conservative package-export resolution.
- Import, type-import, export/re-export, require, literal dynamic-import, computed dynamic-loading, external, unresolved, and excluded observations with confidence and diagnostics.
- Composition-root registration, TypeScript CLI/open options, canonical model normalization, existing viewer/export integration, and regression coverage.

## Explicit non-goals

- Running `tsc`, bundlers, package scripts, or target application code.
- Environment-assisted module resolution, installing dependencies, call graphs, symbol graphs, type checking, or compiler semantic analysis.
- Bundler-only aliases or package-export behavior that cannot be proven from repository-local static data.
- Changes to canonical model JSON, layout configuration, renderer routing, or the upstream reference repository.

## Ordered delivery issues

| Issue | Outcome | Owner | Blocked by | Review gate |
|---|---|---|---|---|
| [020](../../../agents/issues/done/20260827-020-typescript-project-boundary-and-module-discovery.md) | Registered TypeScript analyzer, safe config selection/inheritance, source scope, deterministic module discovery, and module-only analysis output | TypeScript analysis + plugin composition | None | none |
| [021](../../../agents/issues/done/20260827-021-typescript-static-dependencies-and-uncertainty.md) | Static import/export extraction, aliases, package exports, dynamic uncertainty, references, evidence, confidence, and partial results | TypeScript analysis | 020 | none |
| [022](../../../agents/issues/done/20260827-022-typescript-public-and-visible-analysis-path.md) | Public CLI/open options and shared analysis → model → viewer/export path with deterministic regression coverage | TypeScript analysis + existing model/viewer/export consumers | None | visual-review (approved) |

## Slice acceptance

- A representative TypeScript repository reaches a visible architecture view through the existing CLI and local viewer.
- One configured project is selected deterministically; ambiguous configs require explicit selection and inherited configuration is resolved safely.
- `.ts`/`.tsx` modules are discovered by default; `.js`/`.jsx` require project `allowJs` or explicit `include_js`; tests and generated/output/cache/external directories follow the specified defaults.
- Proven static relationships retain import/export kind, aliases, package/runtime provenance, source locations, confidence, and deterministic aggregation.
- Computed or unsupported loading and unresolved aliases/package exports remain visible without invented local modules.
- The canonical model, viewer, export, and host contracts remain language-neutral; existing Go and Python behavior remains unchanged.
- No target code, compiler, bundler, package script, or installation step is executed.

## Verification surfaces

- **Backend boundary:** manifest/detection, config selection and inheritance, safe path handling, module scope, parser/lexer behavior, import/export resolution, package-export conditions, diagnostics, options, deterministic output, and cancellation.
- **Frontend integration:** the existing viewer opens the TypeScript canonical model, preserves hierarchy, directed relationships, references, diagnostics, and source evidence without TypeScript-specific rendering logic.
- **End-to-end:** TypeScript repository → analysis JSON → canonical model → local viewer, self-contained HTML, and SVG, with repeated-output checks.
- **Repository/OKF integrity:** full Go/Node verification, strict OKF validation, synchronized issue references, and untouched upstream reference boundary.

## Artifact impact

This is an implementation slice inside the confirmed TypeScript capability, not a topology change. Delivery truth changes through issues 020–022, the implementation slice, registry, node references, orchestration status, and `.okf/log.md`. Product and architecture boundaries remain unchanged: the adapter implements the existing analyzer contract and shared consumers; application documents are refreshed only to record the implementation sequence/status and the explicit no-impact decision.

## Delivery progress

Issues 020–022 are complete and archived after analyzer-boundary, repository,
race, vet, build, static-analysis, strict OKF, and explicit visual-review
verification. The TypeScript capability is `implemented`.

## Compiled entrypoint evidence

Issue [034](../../../agents/issues/done/20260828-034-compiled-analyzer-plugin-entrypoints.md)
adds the compiled `org.archview.typescript` entrypoint at
`cmd/analyzers/typescript/main.go`. It constructs the existing
`tsanalyzer.New()` implementation behind the shared process runner and passes
manifest and result parity verification.
