# Analyzer plugin runtime canonical use cases

## Application services

### AnalyzerRegistryService

- `RegisterAnalyzer`
- `ListAnalyzers`
- `DetectAnalyzers`
- `SelectAnalyzer`

### AnalyzerExecutionService

- `ResolveAnalyzerOptions`
- `RunAnalyzer`
- `ValidateAnalyzerResult`

## Rules and outcomes

- `RegisterAnalyzer` rejects duplicate IDs, invalid manifests, and unsupported API versions.
- `DetectAnalyzers` returns candidates sorted by confidence then ID.
- `SelectAnalyzer` uses explicit language/ID first; otherwise only one highest-confidence candidate may win.
- `RunAnalyzer` creates a session, invokes with cancellation, converts panic/transport failure to diagnostics, and returns a result status.
- `ValidateAnalyzerResult` rejects broken top-level data and retains recoverable observation diagnostics.

## Failure categories

`ManifestInvalid`, `DuplicateAnalyzer`, `ApiIncompatible`, `NoAnalyzer`, `AmbiguousAnalyzer`, `OptionsInvalid`, `Cancelled`, `TimedOut`, `AnalyzerFailed`, `ResultInvalid`.

## Architecture-neutral mapping

The registry may be static, dependency-injected, or process-backed. The use-case meaning and statuses remain unchanged.
