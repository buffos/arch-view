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

### ProcessAnalyzerService

- LoadProcessPluginDescriptor
- RegisterProcessAnalyzer
- RunProcessDetection
- RunProcessAnalysis
- ValidateProcessProtocol
- CancelProcessSession

## Rules and outcomes

- `RegisterAnalyzer` rejects duplicate IDs, invalid manifests, and unsupported API versions.
- `DetectAnalyzers` returns candidates sorted by confidence then ID.
- `SelectAnalyzer` uses explicit language/ID first; otherwise only one highest-confidence candidate may win.
- `RunAnalyzer` creates a session, invokes with cancellation, converts panic/transport failure to diagnostics, and returns a result status.
- `ValidateAnalyzerResult` rejects broken top-level data and retains recoverable observation diagnostics.
- LoadProcessPluginDescriptor rejects invalid schema/version, malformed
  manifests, empty commands, and unsupported descriptor values before
  registration.
- RunProcessDetection and RunProcessAnalysis launch one process, require a
  matching hello manifest, exchange one request-scoped operation, and require
  one candidate or result followed by done or fatal.
- ValidateProcessProtocol rejects unknown or malformed frames, wrong request
  identifiers, stdout logs, oversized lines, duplicate terminal frames, late
  output, and invalid canonical payloads.
- CancelProcessSession terminates the child after best-effort cancel, waits
  for cleanup, and never accepts a result after cancellation or timeout.

## Failure categories

`ManifestInvalid`, `DuplicateAnalyzer`, `ApiIncompatible`, `NoAnalyzer`, `AmbiguousAnalyzer`, `OptionsInvalid`, `Cancelled`, `TimedOut`, `AnalyzerFailed`, `ResultInvalid`.

## Architecture-neutral mapping

The registry may be static, dependency-injected, or process-backed. The
use-case meaning and statuses remain unchanged. The first process-backed
registration is explicit through a local descriptor and is not automatically
read from the analyzed project.
