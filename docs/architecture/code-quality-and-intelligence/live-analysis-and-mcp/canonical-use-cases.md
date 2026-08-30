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
- `ReconcileSourceState`
- `PlanInvalidation`
- `BuildCandidateSnapshot`
- `PublishSnapshotAtomically`
- `WaitForRevision`

### LiveQueryService

- `QueryLatestReady`
- `QuerySpecificRevision`
- `EnsureCurrentSnapshot`
- `SearchExactText`
- `GetBoundedSourceContext`
- `CompareQualityRevisions`

### QualityGateway

- `ReadQualityCatalog`
- `EvaluateQualityRequest`
- `ExecuteQualityPolicyCommand`

The gateway delegates to the deterministic-quality catalog, evaluation, and
policy services. It does not implement rule, profile, suppression, or baseline
semantics a second time.

### MCPAdapter

- `ServeSnapshotStatus`
- `ServeAnalyzerNeutralSearch`
- `ServeExactTextSearch`
- `ServeQualityReport`
- `ServeQualityEvaluation`
- `ServeQualityPolicyCommand`
- `ServeEvidence`
- `ServeCapabilityAwareSemanticQuery`

## `ValidateLiveSession` — command

**Input:** versioned live config, opened repository root, analyzer/source-index
and quality catalogs, watcher/store/query/transport capabilities.

**Rules:** roots are descendants of the opened root, limits are positive and
within hard maxima, freshness policy is bounded, read-only/no-shell/no-
execution defaults hold, requested capabilities may be unavailable only when
that status is explicit, quality profile references are valid, and operation
permissions are a subset of the server policy. Invalid config fails closed.

**Output:** immutable validated session policy.

## `StartLiveSession` — command

**Input:** validated policy and replaceable watcher/snapshot/query adapters.

**Responsibilities:** perform the initial full source scan across configured
analyzer scopes, build source index/model and the requested quality report, then
publish the first revision only after input stability and all requested
artifacts validate.

**Outcome:** `ready` or `degraded` revision, or `initializing`/`failed` with
structured diagnostics when no usable revision exists.

## `NormalizeWatchEvent` / `CoalesceEvents` — commands

**Input:** backend events.

**Rules:** normalize separators and root containment, canonicalize
rename/delete pairs, deduplicate paths, coalesce within the configured bounded
window, and treat overflow/error/ambiguous paths as rescan triggers.

**Outcome:** immutable event group and no source/model mutation.

## `ReconcileSourceState` — query/command boundary

**Input:** live session roots, last verified input fingerprint, and freshness
policy.

**Rules:** perform a cheap manifest/dirty-state check first when possible. A
strict request is satisfied only by an authoritative eligible-source
content/input fingerprint; metadata or a clean watcher is insufficient when it
cannot prove the source unchanged. If the fingerprint differs, identify
changed paths and build the affected scope. If a strict request arrives while
a reconciliation/build is in progress, join that single-flight work.

**Outcome:** unchanged, changed, failed, or unstable reconciliation result.
Only an unchanged result or a successfully published candidate can satisfy
`require_current`.

## `PlanInvalidation` — command

**Input:** normalized event group, reconciliation result, current scope plan,
source/cache identity, and dependency evidence.

**Rules:** choose selective scopes only when affected source identity and
dependency impact are proven by existing orchestration/cache semantics;
otherwise plan a full rescan of the session/affected scope. The plan never
guesses a semantic relation from a file name or language assumption.

**Outcome:** invalidation plan with paths, scopes, mode, fingerprint, and
reason.

## `BuildCandidateSnapshot` — command

**Input:** invalidation plan, current source scope, registered analyzer
assignments, source-index request, quality profile, analyzer orchestration, and
previous ready revision.

**Responsibilities:** run bounded analysis for affected analyzer scopes,
refresh source index/model and requested quality report, normalize the canonical
model, validate all artifacts, and compare input fingerprints before and after
the build.

**Outcome:** unpublished candidate or isolated diagnostics. If input changes
during the build, the candidate is discarded and retried within policy; after
the retry limit, the session reports `input_unstable`.

## `PublishSnapshotAtomically` — command

**Input:** validated stable candidate.

**Rules:** require matching input fingerprints across source/model/quality,
assign the next revision, calculate semantic digest, swap the store pointer
atomically, and publish freshness/event metadata. Equal semantic digest may
acknowledge an event without creating a new semantic revision.

**Outcome:** one immutable live snapshot visible to all readers.

## `QueryLatestReady` / `QuerySpecificRevision` — queries

**Input:** session ID, consistency selector, optional revision/snapshot
selector, structural filters, projection, cursor, and byte/item budget.

**Output:** `QueryEnvelope<T>` containing exact consistency context, freshness,
capability coverage, omissions, diagnostics, budget usage, and cursor.

**Rules:** `latest_ready` returns the latest committed revision and exposes
stale/updating status. `require_current` performs reconciliation before the
query and returns no current success if the source cannot be verified within
the wait budget. A specific revision is immutable; pagination cannot drift to a
newer revision.

## `EnsureCurrentSnapshot` — command

**Input:** session ID, optional scope set, wait/stability budget, and optional
minimum revision.

**Rules:** reconcile roots, coalesce pending events, start or join the affected
reanalysis, wait for a stable candidate, and return the published revision or
an explicit unavailable/unstable/failed result. It never returns an
unpublished candidate as current.

## `SearchExactText` / `GetBoundedSourceContext` — queries

**Input:** root-safe literal/regex query or indexed entity/span, selected
revision, and line/byte budget.

**Rules:** exact text search returns bounded file/line/column matches. Source
context validates scope/root/hash, clamps to hard maximum, and reads only the
requested range. Arbitrary paths, traversal, shell commands, and writes are
rejected.

## `EvaluateQualityRequest` — command

**Input:** a verified source/model revision, profile identity, optional
temporary rule bindings, scope selector, and consistency/wait policy.

**Rules:** delegate profile validation and evaluation to the deterministic-
quality services. Temporary bindings are not persisted. A strict request first
ensures the source revision is current; the resulting report records source
snapshot, profile/options digest, rule/provider versions, coverage, findings,
and evidence.

**Outcome:** immutable quality report or explicit configuration/capability/
coverage diagnostics.

## `ExecuteQualityPolicyCommand` — command

**Input:** validated profile save/profile save-as or baseline preview/create
request, explicit permission, selected report revision, and audit context.

**Rules:** delegate to deterministic-quality policy services. Profile changes
validate complete rule bindings and preserve versioned identity. A baseline
preview is read-only. Baseline creation requires an exact compatible current
report, selected finding keys, exact rule/profile/formula versions, a reason,
and a non-conflicting destination. No command changes source code.

**Outcome:** immutable policy document/preview and audit result, or a structured
permission/staleness/conflict/validation failure.

## `CompareQualityRevisions` — query

**Input:** two compatible quality report revisions and bounded projection.

**Rules:** delegate to the deterministic-quality comparison service. Match by
stable finding key and exact rule/formula versions. Partial, unsupported, or
incompatible reports cannot imply that a finding was resolved.

**Outcome:** added, unchanged, suppressed, and resolved finding transitions.

## MCP operation mappings

| MCP operation | Shared service | Default output | Permission class |
|---|---|---|---|
| `get_snapshot_status` | `GetLiveStatus` / `ReconcileSourceState` | state, revision, freshness, changed paths, diagnostics | read/status |
| `ensure_current_snapshot` | `EnsureCurrentSnapshot` | verified revision or explicit unstable/unavailable result | analysis |
| `list_scopes` | snapshot query | scopes, roots, analyzer IDs, capabilities/status | read/search |
| `find_files` / `find_symbols` | `QueryLatestReady` | compact records, locations, docs summaries, cursor | read/search |
| `find_text` | `SearchExactText` | bounded matches and source locations | read/search |
| `get_documentation` | source-index query | documentation candidates/status/spans | read/search |
| `get_module_facts` | `QueryLatestReady` | containment-linked files/symbols/docs and architecture links | read/search |
| `get_quality_profiles` / `get_quality_rules` | `ReadQualityCatalog` | profiles, rule catalog, capability/parameter status | quality read |
| `get_quality_findings` / `get_finding_evidence` | quality query | findings, coverage, metrics, evidence summary | read/evidence |
| `evaluate_quality` | `EvaluateQualityRequest` | immutable report or explicit coverage/configuration result | quality evaluate |
| `compare_quality_reports` | `CompareQualityRevisions` | added/unchanged/suppressed/resolved transitions | quality read |
| `validate_quality_profile` / `preview_baseline` | quality policy service | validation or dry-run result | quality policy read |
| `save_quality_profile` / `save_quality_profile_as` | quality policy service | persisted profile identity and audit result | quality policy write |
| `create_baseline` | quality policy service | persisted baseline identity and audit result | baseline write |
| `get_source_context` | `GetBoundedSourceContext` | bounded source lines/bytes and matching hash | read/source-context |

MCP remains a transport adapter over shared services. Viewer and CLI can use
the same providers and must not receive different semantic results.

## Failure model

- `LiveConfigInvalid`
- `WatchRootInvalid`
- `WatchBackendUnavailable`
- `WatchEventOutOfScope`
- `WatchEventOverflow`
- `SourceReconciliationFailed`
- `InputUnstable`
- `SnapshotBuildFailed`
- `SnapshotValidationFailed`
- `SnapshotPublishConflict`
- `NoReadySnapshot`
- `RevisionUnavailable`
- `AnalysisWaitTimeout`
- `QueryBudgetExceeded`
- `QueryCursorInvalid`
- `SourceContextOutOfScope`
- `QualityEvaluationInvalid`
- `QualityPolicyPermissionDenied`
- `QualityProfileConflict`
- `BaselineRevisionStale`
- `MCPOperationUnsupported`
- `MCPPermissionDenied`

Failures preserve the last ready revision where possible and appear in status,
freshness, query diagnostics, or explicit command results.
