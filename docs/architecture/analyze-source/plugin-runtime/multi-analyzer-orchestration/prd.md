# Multi-analyzer project orchestration PRD

## Purpose

Let a developer open one mixed-language repository and receive one navigable
architecture result assembled from the applicable analyzer jobs, without
losing useful results when an independent job fails.

## Actors

- **Developer:** opens a repository and inspects a combined or per-project architecture scope.
- **CI/documentation operator:** runs a repeatable combined analysis and consumes status, diagnostics, and output artifacts.
- **Analyzer host:** discovers roots, plans jobs, schedules processes, and aggregates results.
- **Analyzer process:** analyzes exactly one assigned project scope and reports only its own observations.

## Goals

1. Discover supported project roots inside the opened repository.
2. Plan at most one logical analyzer per project-root/language scope unless an explicit selection says otherwise.
3. Run independent jobs concurrently within a bounded worker pool.
4. Preserve collision-safe identities and per-job provenance in the combined result.
5. Keep successful and partial scopes usable when another job fails, times out, or is cancelled.
6. Make job status and progress observable without requiring analyzer-specific progress protocols.
7. Let project configuration constrain each analyzer job's source input with
   deterministic include/exclude globs.
8. Preserve the canonical model and viewer semantics for every individual scope.

## Non-goals

- Inferring relationships between languages or independent project roots.
- Merging duplicate implementations of the same logical analyzer by default.
- Parsing source or deciding language-specific dependency meaning in the host.
- Running target application code, build tools, package managers, or workspace commands.
- Persistent analysis caching across application sessions in the first slice.

## Required workflows

### Discover project roots

Starting at the opened repository, the host traverses directories in lexical
order while applying fixed exclusions. Strong analyzer manifest markers create
root candidates. A nested strong manifest becomes a nested project scope and is
carved out of its parent's effective input. Weak markers do not create roots on
their own. Configured source filters are not applied during this discovery
pass, so include rules cannot hide project markers.

### Plan jobs

Assignments from the assignment capability are applied first. Otherwise the
host evaluates applicable analyzers for each candidate root. The host then
resolves the invocation-root-relative source-scope policy for each analyzer.
One logical analyzer is selected for a root/language pair; different languages
at the same root become independent jobs. Jobs are sorted by relative root,
language, and logical analyzer ID before scheduling.

### Execute jobs

The scheduler runs at most four jobs by default and never more than sixteen.
Each job receives its project root, nested-root exclusions, resolved source
scope, selection, effective options, and a stable job ID. Cancellation stops
queued jobs and terminates active external processes through the parent process
boundary.

### Aggregate and expose

The host namespaces each successful observation, normalizes the combined
collections, derives graph projections, and returns one aggregate model plus
scope summaries. The viewer may select `All` or an individual scope from the
same cached run; selecting a scope does not re-run analysis.

## Functional requirements

| ID | Requirement |
|---|---|
| MAO-FR-001 | Discover project roots only beneath the opened repository and outside the fixed exclusion set. |
| MAO-FR-002 | Apply strong-manifest nested ownership deterministically and exclude nested owned roots from parent job input. |
| MAO-FR-003 | Plan explicit assignments before automatic detection and preserve one logical analyzer per root/language pair by default. |
| MAO-FR-004 | Run jobs through a worker pool with default concurrency four and hard cap sixteen. |
| MAO-FR-005 | Stop queued and active jobs on cancellation; propagate timeout, process, and analyzer diagnostics to the owning scope. |
| MAO-FR-006 | Derive stable scope and observation IDs that cannot collide across roots or analyzers. |
| MAO-FR-007 | Aggregate only analyzer-reported relationships and preserve per-scope provenance. |
| MAO-FR-008 | Return complete, partial, failed, or cancelled aggregate status using the defined usable-result rules. |
| MAO-FR-009 | Emit deterministic job-plan and lifecycle information suitable for UI and CI consumers. |
| MAO-FR-010 | Keep per-scope results available for combined/per-scope projection without re-analysis. |
| MAO-FR-011 | Resolve global exclusion globs and analyzer-scoped include globs relative to the invocation root. |
| MAO-FR-012 | Apply source filters after root discovery, with fixed, nested-root, and configured exclusions taking precedence over includes. |
| MAO-FR-013 | Include the effective source-scope policy and matched source set in deterministic job and cache identity. |

## Non-functional requirements

- Discovery and job ordering are deterministic for the same repository/configuration.
- The host never creates an unbounded number of processes or workers.
- A failed scope cannot delete or mutate another scope's observations.
- The same invocation root, configuration, analyzer registry, and source tree
  produce the same filtered source set for every job.
- Aggregate normalization remains compatible with the language-neutral model boundary.
- A combined model must clearly report partial status and its contributing scopes.

## Success criteria

The scenarios in [acceptance-scenarios.md](acceptance-scenarios.md) pass for
mixed-language fixtures, nested roots, source-filter precedence, failure/
cancellation cases, collision fixtures, cache invalidation, and repeated runs.
