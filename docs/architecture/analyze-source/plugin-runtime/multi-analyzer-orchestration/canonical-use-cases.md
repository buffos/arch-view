# Multi-analyzer project orchestration canonical use cases

## Application services

### ProjectDiscoveryService

- `DiscoverProjectRoots`
- `ResolveNestedProjectOwnership`
- `EvaluateAnalyzerCandidates`

### AnalyzerJobPlanner

- `PlanAnalyzerJobs`
- `ApplyExplicitAssignments`
- `ResolveSourceScopePolicy`
- `NamespaceJobIdentity`

### AnalyzerJobScheduler

- `ExecuteAnalyzerPlan`
- `CancelAnalyzerPlan`
- `PublishJobLifecycle`

### AnalysisAggregationService

- `CollectUsableScopeResults`
- `AggregateScopeResults`
- `DeriveCombinedGraph`
- `SelectAnalysisScope`

## Canonical commands and queries

### `DiscoverProjectRoots` — query

**Input:** opened repository, discovery policy, analyzer marker catalog.

**Output:** ordered candidate roots, nested ownership tree, exclusions, and
discovery diagnostics. It never runs target code.

### `PlanAnalyzerJobs` — command

**Input:** candidate roots, assignments, CLI selection, validated source-scope
policy, and analyzer registry.

**Responsibilities:** resolve explicit before automatic selection, eliminate
duplicate logical analyzer choices, resolve each job's source scope, create
scope/job IDs, and produce a stable job order.

**Failure:** invalid plan inputs fail before execution; unavailable assigned
analyzers become scoped job diagnostics and do not erase other jobs.

### `ExecuteAnalyzerPlan` — command

**Input:** immutable job plan and cancellation context.

**Responsibilities:** run at most sixteen concurrent workers, defaulting to
four; pass nested exclusions, effective source scope, and options; capture
result, status, diagnostics, and runtime provenance per job.

**Transaction:** each job is isolated. There is no all-jobs transaction.

**Retry:** no automatic retry; a caller can start a new run. A retried job gets
a new run ID but the same stable scope ID when its scope identity is unchanged.

### `AggregateScopeResults` — command

**Input:** terminal job results.

**Responsibilities:** namespace observations, merge/canonicalize collections,
retain diagnostics and provenance, derive cycles/layers, and calculate the
aggregate status using the domain rules.

### `SelectAnalysisScope` — query

**Input:** aggregate run and `all` or one scope ID.

**Output:** the corresponding projection backed by cached job results. It does
not invoke an analyzer.

## Failure model

- `RepositoryUnreadable`
- `DiscoveryLimitExceeded`
- `NoApplicableAnalyzer`
- `AssignmentAnalyzerUnavailable`
- `JobTimedOut`
- `JobCancelled`
- `JobAnalyzerFailed`
- `JobResultInvalid`
- `AnalysisScopeFilterInvalid`
- `AggregateNormalizationFailed`

Scope failures are data in the aggregate when another usable scope remains;
only aggregate-level discovery/normalization failures prevent a usable model.

## Architecture-neutral mapping

The scheduler may be in-process, process-backed, or distributed later. The
stable surface is the job plan, scope identity, lifecycle, aggregation rules,
and result status—not a particular queue or concurrency primitive.
