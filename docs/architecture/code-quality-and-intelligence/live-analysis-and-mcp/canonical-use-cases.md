# Live analysis and MCP canonical use cases

## Application services

### LiveSessionService

- `ValidateLiveSession`
- `StartLiveSession`
- `StopLiveSession`
- `GetLiveStatus`

### SnapshotCoordinator

- `NormalizeWatchEvent`
- `CoalesceEvents`
- `PlanInvalidation`
- `BuildCandidateSnapshot`
- `PublishSnapshotAtomically`

### LiveQueryService

- `QueryLatestReady`
- `QuerySpecificRevision`
- `GetBoundedSourceContext`

### MCPAdapter

- `ServeSnapshotStatus`
- `ServeStructuralSearch`
- `ServeQualityReport`
- `ServeEvidence`
- `ServeCapabilityAwareSemanticQuery`

## `ValidateLiveSession` — command

**Input:** versioned live config, opened repository root, source-index/quality
catalogs, watcher/store/transport capabilities.

**Rules:** roots are descendants of the opened root, limits are positive and
within hard maxima, read-only/no-shell/no-execution invariants hold, requested
capabilities exist or are explicitly allowed to be unavailable, and the
quality profile reference is valid. Invalid config fails closed.

**Output:** immutable validated session policy.

## `StartLiveSession` — command

**Input:** validated policy and replaceable watcher/snapshot/query adapters.

**Responsibilities:** perform the initial full source scan, build source index,
  canonical model, and requested quality report, then publish the first
  revision only after validation.

**Outcome:** `ready` or `degraded` revision, or `initializing`/`failed` with
structured diagnostics when no usable revision exists.

## `NormalizeWatchEvent` / `CoalesceEvents` — commands

**Input:** backend events.

**Rules:** normalize separators and root containment, canonicalize rename/delete
  pairs, deduplicate paths, coalesce within the configured bounded window, and
  treat overflow/error/ambiguous paths as rescan triggers.

**Outcome:** immutable event group and no source/model mutation.

## `PlanInvalidation` — command

**Input:** normalized event group, current scope plan, source/cache identity,
  and dependency evidence.

**Rules:** choose selective scopes only when affected source identity and
  dependency impact are proven by existing orchestration/cache semantics;
  otherwise plan a full rescan of the session/affected scope. The plan never
  guesses a semantic relation from a file name.

**Outcome:** invalidation plan with paths, scopes, mode, fingerprint, and
reason.

## `BuildCandidateSnapshot` — command

**Input:** invalidation plan, current source scope, source-index request, quality
profile, analyzer orchestration, and previous ready revision.

**Responsibilities:** run bounded analysis, rebuild/refresh source index and
quality report, normalize the canonical model, validate all artifacts, and
produce one unpublished candidate.

**Outcome:** candidate snapshot or isolated diagnostics. The previous ready
revision remains available during the operation.

## `PublishSnapshotAtomically` — command

**Input:** validated candidate.

**Rules:** require matching input fingerprint across source/model/quality,
  assign the next revision, calculate semantic digest, swap the store pointer
  atomically, and publish freshness/event metadata. Equal semantic digest may
  acknowledge the event without creating a new semantic revision.

**Outcome:** one immutable live snapshot visible to all readers.

## `QueryLatestReady` / `QuerySpecificRevision` — queries

**Input:** session ID, optional revision/snapshot selector, structural filters,
  projection, cursor, and byte/item budget.

**Output:** `QueryEnvelope<T>` containing result, exact consistency context,
  freshness, coverage, omissions, diagnostics, budget usage, and cursor.

**Rules:** latest means latest ready, not an unpublished candidate. A specific
  revision is immutable; pagination cannot drift to a newer revision.

## `GetBoundedSourceContext` — query

**Input:** indexed entity/span, selected revision, line/byte budget.

**Rules:** validate scope/root/hash, clamp to hard maximum, and read only the
  requested bounded range. Arbitrary paths, shell commands, and writes are
  rejected.

## MCP operation mappings

| MCP operation | Shared service | Default output |
|---|---|---|
| `get_snapshot_status` | `GetLiveStatus` | state, latest/last-ready revision, changed paths, diagnostics |
| `list_scopes` | snapshot query | scope IDs, project roots, analyzer/status provenance |
| `find_files` | `QueryLatestReady` | paths, language, roles, size/hash/status |
| `find_symbols` | `QueryLatestReady` | names, categories, locations, visibility, docs IDs |
| `get_documentation` | `QueryLatestReady` | bounded normalized docs, subject, format/status/evidence |
| `get_module_facts` | `QueryLatestReady` | contained files/symbols/docs and architecture links |
| `get_quality_findings` | `QueryLatestReady` | findings, status/severity, coverage, evidence summary |
| `get_finding_evidence` | `QueryLatestReady` + evidence | spans, metrics, relations, bounded context if requested |
| `get_callers_callees` | capability-aware query | results or explicit unsupported/unknown status |
| `get_source_context` | `GetBoundedSourceContext` | bounded source lines/bytes and matching hash |

MCP remains a transport adapter over the shared services. Viewer and CLI can
use the same query provider and must not receive different semantic results.

## Future remediation handoff

`RequestRemediationHandoff` is intentionally not a v1 write operation. A future
capability may accept a finding key, selected revision, proposed patch/dry-run,
and explicit user authorization, then return an auditable handoff. The live
watcher must treat any external edit as a new filesystem event; it never edits
source itself.

## Failure model

- `LiveConfigInvalid`
- `WatchRootInvalid`
- `WatchBackendUnavailable`
- `WatchEventOutOfScope`
- `WatchEventOverflow`
- `SnapshotBuildFailed`
- `SnapshotValidationFailed`
- `SnapshotPublishConflict`
- `NoReadySnapshot`
- `RevisionUnavailable`
- `QueryBudgetExceeded`
- `QueryCursorInvalid`
- `SourceContextOutOfScope`
- `MCPOperationUnsupported`
- `MCPPermissionDenied`

Failures preserve the last ready revision where possible and appear in status,
freshness, or query diagnostics.
