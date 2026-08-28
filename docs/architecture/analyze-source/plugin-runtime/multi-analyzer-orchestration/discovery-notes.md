# Multi-analyzer project orchestration discovery notes

## Purpose

Make one architecture session represent mixed-language and nested projects by
planning and running all applicable analyzer jobs together.

## Current implementation

- **Observed in code:** `analysis.Host.Run` selects one analyzer and returns one
  `AnalysisResult`; automatic detection currently resolves one highest-
  confidence candidate, and `open` normalizes one result.
- **Inferred from docs:** The canonical model already carries language,
  project, source evidence, diagnostics, and provenance fields that can support
  aggregation, but the execution contract is still single-analyzer.

## User-confirmed target behavior

- The host can discover more than one project root and applicable analyzer.
- Independent compiled analyzer processes can run concurrently with bounded
  resource use.
- Results are merged into one language-neutral model with collision-safe module
  identity and analyzer provenance.
- A failed or partial analyzer does not discard successful results from other
  jobs.
- An explicit assignment or analyzer selection can constrain the job set.

## Specified decisions

- Discovery begins at the opened repository root and recursively evaluates
  analyzer markers within bounded, product-defined exclusions. A strong
  project manifest owns its subtree; a nested manifest or explicit assignment
  creates a nested project scope. Explicit assignments can narrow or override
  automatic candidates.
- Automatic planning schedules at most one logical analyzer per
  project-root/language pair. Same-language implementation duplicates are
  resolved by the stable logical ID and explicit selection policy; different
  languages in one root are independent jobs and may run together.
- Each job is keyed by relative project root plus logical analyzer ID. Merged
  module and relationship identities include that job key and the analyzer's
  local ID. Only relationships explicitly reported by a job are retained;
  cross-language and cross-root relationships are not inferred.
- A configurable bounded worker pool runs jobs concurrently, with a
  recommended default of four and a hard safety cap. Cancellation stops queued
  work and terminates active external processes; timeouts and diagnostics stay
  attached to their job.
- Successful jobs remain visible when another job fails or times out. The
  aggregate is partial unless no usable result exists, with an explicit future
  strict/all-or-nothing mode allowed without changing the default.
- The canonical output is one combined language-neutral model, backed by
  individual job results/provenance so the application can expose per-scope
  projections without re-running analysis.

## Implementation and verification focus

The exact-spec set defines marker traversal and exclusion algorithms, job-plan
and aggregate model fields, deterministic progress events, resource budgets,
and normalization order. Implementation and verification must now exercise
mixed-language repositories, nested roots, partial failures, cancellation,
cache reuse, and deterministic aggregate output.

## Boundary

This frontier owns job planning, concurrent execution, aggregation, identity,
provenance, and partial failure. Analyzer binaries and installation belong to
compiled external distribution; language semantics belong to individual
analyzers; rendering remains downstream in the model and viewer capabilities.
