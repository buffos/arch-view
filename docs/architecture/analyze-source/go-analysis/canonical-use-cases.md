# Go analysis canonical use cases

## GoAnalyzerService

- `DetectGoProject`
- `SelectGoModule`
- `DiscoverGoPackages`
- `ExtractGoImports`
- `ResolveGoImportTargets`
- `EmitGoAnalysisResult`

## Orchestration

1. Read `go.mod` and optional `go.work`.
2. Select one module or return `AmbiguousWorkspace`.
3. Apply source filters and build-view options.
4. Parse eligible files and group by package directory.
5. Resolve local imports by module path/directory mapping.
6. Emit non-local references and diagnostics.
7. Return common analysis result.

Failures: missing/invalid `go.mod`, ambiguous workspace, unreadable file, invalid package declaration, unsupported build constraint, unresolved import, cgo import.

Retry is safe and read-only; identical source/configuration produces deterministic observations.
