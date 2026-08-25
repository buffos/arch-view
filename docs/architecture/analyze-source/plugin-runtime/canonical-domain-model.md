# Analyzer plugin runtime canonical domain model

## Registry aggregate

`AnalyzerRegistry` owns a set of manifests and implementations.

### AnalyzerManifest

Required fields: `id`, `version`, `language`, `api_version`, `detection_markers[]`, `capabilities[]`, `options[]`.

`id` is lowercase stable reverse-DNS or namespace-like text; `version` is semantic-version text; `api_version` is `arch-view.analyzer/v1` for the first contract.

Each option declares `name`, `type`, `default`, `allowed_values?`, `description`, and `sensitive=false` unless explicitly supported.

### DetectionCandidate

`analyzer_id`, `confidence` (`0..1`), `matched_markers[]`, `boundary_hint?`, `reason`.

### AnalyzerSession

`session_id`, `project_root`, `selection`, `effective_options`, `status`, `started_at?`, `finished_at?`, `result?`, `diagnostics[]`.

Invariants: one selected analyzer; options are validated before running; terminal statuses cannot transition back; cancellation is terminal.

## Extension points

Capabilities, analyzer versions, option schemas, process transports, and future language adapters extend the registry without changing model vocabulary.

## Domain events

`AnalyzerRegistered`, `AnalyzerSelected`, `AnalyzerStarted`, `AnalyzerProducedPartialResult`, `AnalyzerCompleted`, `AnalyzerFailed`, `AnalyzerCancelled`.
