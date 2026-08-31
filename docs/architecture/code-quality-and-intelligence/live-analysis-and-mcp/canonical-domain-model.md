# Live analysis and MCP canonical domain model

## Modeling boundary

This model owns live-session lifecycle, filesystem event normalization,
request-time freshness reconciliation, snapshot consistency, analyzer-neutral
query envelopes, MCP operations, budgets, and operation permissions. Source
facts, analyzer semantics, graph meaning, quality rules/profile/baseline
semantics, and source mutation remain owned by their separate capabilities.

## LiveSessionConfig

```text
LiveSessionConfig {
  schema_version: "arch-view.live/v1"
  session_id: OpaqueId
  repository_root: RelativePath
  watch_roots: WatchRoot[]
  source_index_request: SourceIndexRequest
  quality_request?: QualityRequest
  watch_policy: WatchPolicy
  freshness_policy: FreshnessPolicy
  query_policy: QueryPolicy
  permission_policy: PermissionPolicy
  extensions: ExtensionBlock[]
}

WatchRoot {
  path: RelativePath
  recursive: boolean
}

SourceIndexRequest {
  enabled: boolean
  capabilities: NamespacedId[]
}

QualityRequest {
  profile_id: NamespacedId
  profile_version: string
}

WatchPolicy {
  debounce_ms: integer >= 0
  max_pending_events: integer > 0
  max_parallel_scopes: integer > 0
  rescan_interval_ms?: integer > 0
}

FreshnessPolicy {
  default_consistency: "latest_ready" | "require_current"
  settle_ms: integer >= 0
  max_wait_ms: integer >= 0
  max_stability_retries: integer > 0
  reconcile_interval_ms?: integer > 0
}

QueryPolicy {
  default_max_bytes: integer > 0
  default_max_items: integer > 0
  hard_max_bytes: integer > 0
  hard_max_items: integer > 0
  default_context_lines: integer >= 0
  hard_context_lines: integer >= 0
}

PermissionPolicy {
  default_mode: "read_only"
  allowed_operations: Operation[]
  transport: "stdio" | "local_http" | "authenticated_http"
  allow_target_execution: false
  allow_shell: false
  audit_policy_writes: true
}

Operation = "status" | "search" | "evidence" | "source_context" |
  "ensure_current" | "quality_evaluate" | "quality_profile_read" |
  "quality_profile_write" | "baseline_read" | "baseline_write"
```

Roots are normalized relative to the opened repository and cannot escape it.
The config is separate from layout, analyzer assignment semantics, quality
profile semantics, and source-index facts. The quality profile is referenced,
not copied into live configuration. `allowed_operations` controls the MCP
surface; the default grants read operations only.

## Watch events, reconciliation, and invalidation

```text
WatchEvent {
  backend_id: NamespacedId
  kind: "create" | "modify" | "delete" | "rename" | "overflow" | "error"
  path: RelativePath
  old_path?: RelativePath
  sequence?: string
}

NormalizedEvent {
  kind: "create" | "modify" | "delete" | "rename" | "overflow" | "error"
  path: RelativePath
  old_path?: RelativePath
  event_group_id: OpaqueId
}

ReconciliationResult {
  status: "unchanged" | "changed" | "failed" | "unstable"
  manifest_fingerprint: ContentDigest
  content_fingerprint?: ContentDigest
  changed_paths: RelativePath[]
  attempts: integer > 0
}

InvalidationPlan {
  event_group_id: OpaqueId
  affected_paths: RelativePath[]
  affected_scope_ids: OpaqueId[]
  mode: "selective" | "full_rescan"
  source_fingerprint_before?: ContentDigest
  reason?: string
}
```

Events are hints. A rename is treated as delete/create unless a safe provider
correlation exists. Overflow, backend error, path ambiguity, or missed
sequence uses `full_rescan`. A strict query first runs reconciliation against
the configured roots. If it finds a change, it starts or joins the one
single-flight build for the resulting target state. A cheap manifest/dirty-state
check is an optimization; an authoritative content/input fingerprint is
required whenever metadata or watcher delivery cannot prove the eligible source
unchanged.

The coordinator verifies the source manifest/content fingerprint before and
after analysis. If the source changes during the bounded build, the candidate
is discarded and retried according to `FreshnessPolicy`; after the retry limit
the result is `unstable`, not current.

## LiveSnapshot

```text
LiveSnapshot {
  schema_version: "arch-view.live/v1"
  snapshot_id: OpaqueId
  revision: integer > 0
  state: "ready" | "degraded"
  source_input_fingerprint: ContentDigest
  input_verification: InputVerification
  source_index_ref?: OpaqueRef
  model_ref?: OpaqueRef
  quality_report_ref?: OpaqueRef
  quality_policy_ref?: OpaqueRef
  scope_ids: OpaqueId[]
  diagnostics: LiveDiagnostic[]
  freshness: Freshness
  semantic_digest: ContentDigest
  extensions: ExtensionBlock[]
}

InputVerification {
  status: "verified" | "unstable" | "unknown"
  manifest_fingerprint: ContentDigest
  content_fingerprint?: ContentDigest
  attempts: integer > 0
}

Freshness {
  status: "current" | "stale" | "updating" | "failed" |
    "initializing" | "input_unstable"
  requested_consistency?: "latest_ready" | "require_current"
  last_ready_revision?: integer > 0
  pending_event_group_id?: OpaqueId
  changed_paths?: RelativePath[]
  reconciliation: "not_requested" | "passed" | "changed" | "failed" | "unstable"
}

OpaqueRef {
  kind: "source_index" | "model" | "quality_report" | "quality_policy"
  id: OpaqueId
}
```

`revision` is monotonic within a session and is assigned only at atomic
publication. A snapshot's source index, model, quality report, scope set,
effective quality reference, and input fingerprint refer to the same source
state. Operational freshness can change while the last-ready semantic snapshot
remains unchanged.

## Session lifecycle

```text
stopped -> initializing -> ready
ready -> stale -> updating -> ready | degraded | input_unstable
ready -> updating -> ready | degraded | input_unstable
degraded -> stale | updating -> ready | degraded | input_unstable
input_unstable -> updating | failed
initializing -> failed
failed -> initializing
```

`stale`/`updating`/`failed`/`input_unstable` describe session freshness around
the last-ready revision. If no ready revision exists, queries return
unavailable status. A failed or unstable update never replaces the last-ready
snapshot with an empty result.

## QueryEnvelope

```text
QueryEnvelope<T> {
  schema_version: "arch-view.query/v1"
  session_id: OpaqueId
  snapshot_id?: OpaqueId
  revision?: integer > 0
  requested_consistency: "specific_revision" | "latest_ready" | "require_current"
  returned_consistency: "specific_revision" | "latest_ready" | "current" | "unavailable"
  freshness: Freshness
  scope_ids: OpaqueId[]
  result: T
  result_count: integer >= 0
  omitted_fields: NamespacedId[]
  capabilities: CapabilityCoverage[]
  budget: BudgetUsage
  next_cursor?: OpaqueCursor
  diagnostics: QueryDiagnostic[]
}

CapabilityCoverage {
  capability: NamespacedId
  status: "observed" | "partial" | "unknown" | "unsupported" |
    "not_evaluable" | "unavailable"
  provider_ids?: NamespacedId[]
  reason?: string
}

BudgetUsage {
  max_bytes: integer > 0
  max_items: integer > 0
  emitted_bytes: integer >= 0
  emitted_items: integer >= 0
  truncated: boolean
}
```

`require_current` is enforced by the server before the query executes. When a
ready revision exists, `snapshot_id` and `revision` are required and identify
the immutable result. If no ready revision exists, the response may omit those
two fields, use `returned_consistency: unavailable`, and return the explicit
`no_ready_snapshot`/`input_unstable`/timeout diagnostic. If a previous ready
revision exists but cannot be refreshed, that revision remains identified and
is explicitly marked not current. Cursors are opaque and bound to session,
snapshot/revision, query, canonical ordering, projection, and budget. A cursor
from another context is rejected rather than silently restarting.

## Structural and text queries

```text
StructuralQuery {
  scope_ids?: OpaqueId[]
  path_prefix?: RelativePath
  path_glob?: string
  language?: NamespacedId
  roles?: NamespacedId[]
  name?: string
  qualified_name?: string
  symbol_categories?: string[]
  documentation_text?: string
  rule_ids?: NamespacedId[]
  severities?: string[]
  statuses?: string[]
  case_sensitive: boolean
  max_bytes?: integer
  max_items?: integer
  cursor?: OpaqueCursor
}

TextSearchQuery {
  pattern: string
  mode: "literal" | "regex"
  path_glob?: string
  language?: NamespacedId
  scope_ids?: OpaqueId[]
  case_sensitive: boolean
  max_line_bytes?: integer
  max_bytes?: integer
  max_items?: integer
  cursor?: OpaqueCursor
}
```

Structural and exact-text matching is deterministic over source-index/model/
quality facts and root-safe source records. Text search returns bounded matches
with file/line/column locations, not whole files. Semantic/fuzzy/embedding
ranking is not implied by either query.

## Quality requests and policy commands

```text
QualityEvaluationRequest {
  session_id: OpaqueId
  consistency: "latest_ready" | "require_current" | "specific_revision"
  revision?: integer
  scope_ids?: OpaqueId[]
  profile_id: NamespacedId
  profile_version: string
  baseline_mode: "profile" | "none" | "selected"
  baseline_files?: RelativeFileName[]
  rule_bindings?: RuleBinding[]
  persist: false
}

QualityPolicyCommand {
  operation: "validate_profile" | "save_profile" | "save_profile_as" |
    "preview_baseline" | "create_baseline" | "append_baseline"
  report_revision?: integer
  profile_id?: NamespacedId
  profile_version?: string
  finding_keys?: StableFindingKey[]
  reason?: string
  expected_baseline_revision?: string
  destination?: RelativePath
  authorization: OpaqueAuthorization
}

StableFindingKey = opaque stable key from `arch-view.quality/v1`
OpaqueAuthorization = host-issued authorization record
```

Temporary rule bindings and selected baseline files are evaluated through the
deterministic-quality `QualityEvaluationService` and do not change the project
profile or session. `profile` baseline mode loads the exact profile reference;
`none` disables suppression; `selected` is a request-scoped list of safe files
that must share one baseline identity/revision. Profile saves and baseline
commands use the existing quality policy services, validate exact
rule/profile/formula versions, require the appropriate operation permission,
and produce auditable command results. A baseline command must use an exact
current compatible report and selected finding keys; it cannot mean “suppress
everything”.

## MCP adapter boundary

```text
WatchBackend {
  Start(WatchRoot[], WatchPolicy) -> EventStream
  Stop() -> void
}

ReconciliationService {
  CheckRoots(session_id) -> ReconciliationResult
  EnsureCurrent(session_id, FreshnessPolicy) -> LiveSnapshot | Unavailable
}

SnapshotStore {
  ReadLatestReady(session_id) -> LiveSnapshot?
  PublishAtomically(candidate) -> LiveSnapshot
  ReadRevision(session_id, revision) -> LiveSnapshot?
  WaitForRevision(session_id, minimum_revision, timeout) -> LiveSnapshot?
}

QueryProvider {
  Execute(snapshot, StructuralQuery, Budget) -> QueryResult
  SearchText(snapshot, TextSearchQuery, Budget) -> QueryResult
  ReadDocumentation(snapshot, EntityRef, Budget) -> QueryResult
  ReadEvidence(snapshot, EntityRef, Budget) -> QueryResult
  ReadSourceContext(snapshot, SourceSpan, ContextBudget) -> QueryResult
  ReadCallersCallees(snapshot, EntityRef, Budget) -> QueryResult
}

QualityGateway {
  ReadCatalog(snapshot, Budget) -> QueryResult
  ReadBaselines(snapshot, BaselineSelector, Budget) -> QueryResult
  Evaluate(snapshot, QualityEvaluationRequest) -> QualityEvaluation
  Compare(revision_a, revision_b, Budget) -> QueryResult
  ExecutePolicyCommand(snapshot, QualityPolicyCommand) -> CommandResult
}

TransportAdapter {
  Serve(ReadOrAuthorizedOperation) -> QueryEnvelope | CommandResult
}
```

These focused interfaces make OS watcher, polling, in-memory/persisted store,
HTTP, stdio, analyzer providers, quality services, and future transports
substitutable. The coordinator depends on the abstractions and does not grow
transport, filesystem, language, or rule branches.

## MCP operation set

V1 exposes the following operation groups:

- **Freshness:** `get_snapshot_status`, `ensure_current_snapshot`.
- **Navigation:** `list_scopes`, `find_files`, `find_symbols`, `find_text`,
  `get_documentation`, `get_module_facts`, `get_callers_callees`,
  `get_source_context`.
- **Quality:** `get_quality_profiles`, `get_quality_rules`,
  `get_quality_baselines`, `get_quality_findings`, `get_finding_evidence`,
  `evaluate_quality`, `compare_quality_reports`.
- **Quality policy:** `validate_quality_profile`, `save_quality_profile`,
  `save_quality_profile_as`, `preview_baseline`, `create_baseline`,
  `append_baseline`.

Each operation declares required analyzer/source/quality capabilities and
permission class. Missing capabilities return explicit coverage status rather
than an empty success. Quality policy operations delegate to the deterministic-
quality services and are disabled unless explicitly allowed.

## Atomic publication and freshness invariants

- A candidate is unpublished until all requested source/model/quality artifacts
  validate and share an input fingerprint.
- Strict queries reconcile the configured roots before selecting a snapshot.
- Concurrent strict requests join one reconciliation/build rather than starting
  duplicate work.
- Readers see either the previous complete revision or the new complete
  revision, never a mixture.
- A candidate whose input changes during analysis is retried or rejected as
  unstable; it is never labeled current.
- Revision numbers strictly increase for semantic input changes; a rescan with
  equal semantic digest need not create a new revision.
- Quality-only evaluations retain the verified source revision and include the
  effective profile/options digest in their report identity.
- Profile/baseline policy writes never mutate an existing report; the next
  evaluation observes the new policy and produces a new report identity.
- A managed baseline append merges into one canonical document, is idempotent
  for identical review metadata, rejects conflicting duplicates, increments
  the numeric revision, updates the profile reference, and returns an explicit
  reevaluation-needed result. Missing automatic baselines warn without
  suppressing; invalid or ambiguous references fail.
- Diagnostics and freshness may report a failed event group without replacing
  valid data.
- Query results identify the exact revision and do not silently move to a newer
  revision during pagination or comparison.

## Security invariants

- Every path is normalized and resolved beneath an allowed root.
- Symlink/junction traversal outside the root is rejected.
- Default MCP operations are read-only and cannot execute a command, target
  application, or source mutation.
- Quality-policy writes can touch only validated project profile/baseline
  destinations, require explicit permission and authorization, and emit audit
  data; they cannot write arbitrary files.
- Source context is explicit, bounded, and derived from an indexed span/entity;
  arbitrary local paths are not accepted.
- Full source text, hidden files, quality internals, and verbose diagnostics
  are omitted unless the requested capability/permission and budget allow them.
