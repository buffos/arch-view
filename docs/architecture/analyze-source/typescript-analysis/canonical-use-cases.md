# TypeScript analysis canonical use cases

## TypeScriptAnalyzerService

- `DetectTypeScriptProject`
- `SelectTypeScriptConfig`
- `ResolveEffectiveTsConfig`
- `DiscoverTypeScriptModules`
- `ResolveStaticTypeScriptDependencies`
- `EmitTypeScriptAnalysisResult`

## Orchestration

Select config → resolve `extends`/compiler options → apply include/exclude and JS policy → parse modules → resolve aliases/package context/import/export forms → emit local observations and non-local/dynamic diagnostics.

Failures: no/ambiguous config, invalid extends chain, unsupported compiler option, unreadable file, unresolved alias/package export, dynamic loading. Recoverable failures produce partial status.

No compiler or package script execution is part of the use case.
