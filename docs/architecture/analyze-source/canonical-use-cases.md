# Analyze source code canonical use cases

## Application-layer goals

Coordinate safe project analysis and return stable observations without leaking parser or language-runtime details.

## Application services

### AnalysisOrchestrator

- `DetectAnalyzer`
- `SelectAnalyzer`
- `RunStaticAnalysis`
- `ValidateAnalysisResult`

### AnalysisReadService

- `GetSupportedAnalyzers`
- `GetAnalysisResult`

## Canonical commands

### DetectAnalyzer

Input: project root and optional detection hints. Output: ranked candidates or `NoAnalyzer`/`AmbiguousAnalyzer` outcome. It reads markers only and does not parse target code beyond what detection requires.

### RunStaticAnalysis

Input: project root, selection, effective options, cancellation context. Responsibilities: select the analyzer, enforce scope/safety, invoke it, attach provenance, validate observations, and return `complete`, `partial`, `failed`, or `cancelled`.

Failure classes: invalid root, no/ambiguous analyzer, unsupported option, analyzer failure, unreadable file, cancellation, host timeout.

### ValidateAnalysisResult

Input: analyzer result. Output: accepted result plus diagnostics. It rejects broken references or malformed required fields while retaining recoverable observation errors.

## Canonical queries

- `GetSupportedAnalyzers` returns manifests and capabilities in deterministic order.
- `GetAnalysisResult` returns a stored/in-memory result by run ID; it never reruns analysis.

## Transaction and retry expectations

Analysis is read-only. A retry may produce a new run ID; it must not mutate the target project. Host normalization is deterministic for identical inputs, options, analyzer version, and source state.

## End-to-end chains

1. Detect → select → resolve options → run → validate → return complete result.
2. Detect → observe unresolved dependency → retain usable result → return partial status and diagnostic.
3. Explicit language → reject incompatible root with a clear selection/boundary diagnostic.
4. Cancellation → stop at safe boundary → return cancelled status without pretending completion.

## Architecture-neutral mapping

The service can be a Go application service, a CLI orchestration function, an HTTP handler, or a process host. None may move parsing or language-specific rules into the application service.
