# Analyzer plugin runtime discovery notes

## Purpose

Provide one host-controlled lifecycle and contract for language analyzers so new languages do not require changes to the model, graph engine, viewer, or exporters.

## Target boundary

The runtime owns analyzer registration, manifest validation, project detection, explicit language selection, option precedence, lifecycle/cancellation, result validation, normalization handoff, and diagnostics. It does not parse language syntax, calculate layout, render graphics, or export artifacts.

## Confirmed decisions

- Built-in analyzers use an in-process Go interface first.
- A manifest declares stable plugin ID/version, language, detection markers, supported capabilities, options/defaults, and contract/API version.
- Explicit `--language` selection wins. Auto-detection succeeds only when one analyzer is an unambiguous match; no match or ambiguity is a diagnostic failure requiring user direction.
- The host supplies project root and options; the analyzer owns language-specific project-boundary rules.
- CLI options override project configuration, which overrides analyzer defaults. The host passes the resolved options to the analyzer.
- Each run is cancellable, read-only, and isolated from other runs. An analyzer cannot emit UI objects or arbitrary application code execution.
- The host validates, deduplicates, sorts, and hands normalized analyzer results to the architecture-model capability.
- External analyzers are a later versioned NDJSON/JSON-Schema boundary. The future protocol reserves stdout for protocol messages, stderr for logs, and carries version/capability negotiation, cancellation, timeouts, and diagnostics.

## Runtime flow

`register -> validate manifests -> detect/select -> resolve options -> analyze with context -> validate result -> normalize -> return partial result/diagnostics`.

## Open questions for exact specification

- Exact Go interfaces, manifest fields, capability names, and option schema.
- Detection scoring and project-root normalization rules.
- Cancellation, timeout, panic/error recovery, and resource-limit behavior.
- Whether built-in analyzers run in separate goroutines or processes when optional tool-assisted resolution is enabled.
- Exact external protocol envelope and compatibility policy; this remains intentionally deferred until the in-process contract stabilizes.
