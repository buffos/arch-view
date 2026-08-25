# Python analysis canonical use cases

## PythonAnalyzerService

- `DetectPythonProject`
- `ResolvePythonSourceRoots`
- `DiscoverPythonModules`
- `ResolveStaticPythonImports`
- `ReportDynamicPythonImports`
- `EmitPythonAnalysisResult`

## Orchestration

Read effective configuration → establish roots → parse eligible files → derive package/module hierarchy → resolve absolute/relative imports → classify external/dynamic targets → emit common result.

Failures: no root, invalid config, unsupported syntax version, unreadable file, conflicting package layout, unresolved import, dynamic import. Recoverable failures produce partial status.

Retry is read-only and deterministic for the same source/configuration/options.
