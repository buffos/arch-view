# Analyze source code canonical domain model

## Purpose and modeling principles

This model defines the language-neutral observation semantics produced before canonical graph normalization. It models static analysis meaning, not parser classes, storage tables, or UI state.

## Contexts

- **Selection context:** project root, detection, analyzer manifest, effective options.
- **Observation context:** modules, relationships, evidence, metadata, confidence.
- **Reporting context:** diagnostics, result status, summary, provenance.

## Aggregate: AnalysisRun

`AnalysisRun` is the lifecycle root for one project/language attempt.

Fields: `run_id`, `project_root_label`, `effective_language`, `analyzer_id`, `analyzer_version`, `options_fingerprint`, `status`, `started_at?`, `finished_at?`, `result`, `diagnostics`.

Statuses: `created`, `running`, `complete`, `partial`, `failed`, `cancelled`.

Invariants:

- One run has one project root and one effective analyzer.
- `complete` has no error-severity diagnostics; `partial` has usable observations and at least one recoverable issue; `failed` has no promise of a usable result.
- A cancelled run cannot be reported as complete.
- Target source is never mutated or executed.

## Entities and value objects

- **AnalyzerManifest:** stable ID/version, language, API version, detection markers, capabilities, option descriptors.
- **AnalyzerSelection:** explicit or detected choice plus confidence and competing candidates when ambiguous.
- **AnalysisOptions:** include/exclude policy, tests/generated/external switches, language-specific options, safe-mode/tool-assistance flag.
- **ModuleObservation:** stable project-scoped ID, language, kind, canonical name, display name, hierarchy segments, source-reference IDs, tags, metadata.
- **RelationshipObservation:** stable ID, type, source module ID, target module ID or reference ID, scope, evidence IDs, confidence, metadata.
- **Reference:** non-local target with stable ID, name, scope (`external`, `standard_library`, `unresolved`, `dynamic`), language, metadata.
- **SourceReference:** repository-relative path, optional one-based start/end line/column, optional symbol, evidence kind.
- **Diagnostic:** code, severity (`info`, `warning`, `error`), message, optional subject/path/location, recoverable flag, metadata.
- **Confidence:** optional normalized score plus basis (`resolved`, `inferred`, `dynamic`, `unresolved`).

## Policies and invariants

- Analyzer-specific syntax stays inside the adapter; the shared output uses the vocabulary above.
- Every relationship endpoint is either a known module or an explicit non-local reference.
- Evidence IDs must resolve; duplicate evidence is merged deterministically.
- Hierarchy is structured data and is not derived by splitting module IDs.
- Exclusion policy is explicit and reportable.

## Domain events

- `AnalyzerSelected`
- `AnalysisStarted`
- `AnalysisProducedPartialResult`
- `AnalysisCompleted`
- `AnalysisFailed`
- `AnalysisCancelled`

Events are conceptual observations for adapters/telemetry; they do not require a message broker.

## Extension points

Future relation types, language-specific metadata/tags, external process analyzers, optional read-only resolution, and symbol/call observations extend the model without changing the first static dependency meaning.

## Canonical scenarios

Analyze a resolvable project completely; return a partial result for one unresolved import; reject ambiguous analyzer detection; cancel without executing code; preserve evidence for a multi-file module.
