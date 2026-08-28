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
- External analyzers are an opt-in versioned NDJSON/JSON-Schema boundary loaded
  from an explicitly supplied local descriptor. The protocol reserves stdout
  for protocol messages, stderr for bounded logs, and carries version/capability
  negotiation, detection, analysis, cancellation, terminal status, and
  diagnostics.
- The first external deployment is an existing Python analyzer port using the
  standard-library ast module. It is a parity pilot, not a new language and not
  a replacement for the in-process Python adapter.

## Confirmed future direction

- **User-confirmed target behavior:** Supported analyzers should ultimately be
  shipped as compiled external executables that reuse the same analyzer
  implementations as the current in-process adapters. The end user should not
  need a Python, Rust, Node, or other language runtime.
- **User-confirmed target behavior:** The host should plan and run multiple
  analyzer jobs concurrently for mixed-language or nested projects, then merge
  their language-neutral results.
- **User-confirmed target behavior:** Project configuration should map
  repository-relative folders or project roots to analyzer IDs, with a viewer
  control for switching between individual scopes and the combined view.
- **User-confirmed target behavior:** Compiled external artifacts are the
  production implementation for stable logical analyzer IDs; in-process
  adapters remain only for development, tests, or explicit migration fallback.
- **User-confirmed target behavior:** A reproducible root build target compiles
  each analyzer and assembles the application-managed analyzer directory;
  automatic discovery does not scan target repositories.
- **User-confirmed target behavior:** Multi-analyzer discovery uses bounded
  marker-driven nested project scopes, a bounded concurrent job pool, namespaced
  merge identity, partial-success behavior, and cached per-job results.
- **Observed in code:** The current host selects one analyzer per `Run`, and the
  current external Python descriptor launches `python launcher.py`.
- **Mismatch:** The script-based, single-analyzer pilot does not yet provide
  the confirmed compiled, multi-analyzer, assignment-driven target. This is
  net-new future scope beyond the completed v1 pilot, not a failure of the
  pilot's existing contract.

## Runtime flow

`load descriptor -> validate manifest -> register -> detect/select -> resolve options -> launch one process -> handshake/frame validation -> analyze with context -> validate result -> normalize -> return partial result/diagnostics`.

## Specified future boundaries and implementation follow-up

- Exact Go interfaces, manifest fields, capability names, and option schema.
- Detection scoring and project-root normalization rules.
- Cancellation, timeout, panic/error recovery, and resource-limit behavior.
- Whether built-in analyzers run in separate goroutines or processes when optional tool-assisted resolution is enabled.
- Protocol migration beyond v1 remains outside the pilot. The specified child
  capabilities own executable distribution/package management, multi-analyzer
  scheduling/merging, and analysis assignment configuration. Their exact
  schemas, algorithms, and acceptance scenarios are complete and readiness-
  reviewed; implementation and verification remain future work.
