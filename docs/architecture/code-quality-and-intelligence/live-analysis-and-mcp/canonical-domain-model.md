# Live analysis and MCP canonical domain model

## Modeling boundary

This model owns live-session lifecycle, filesystem event normalization,
snapshot consistency, structural query envelopes, MCP operations, budgets, and
read-only permissions. Source facts, quality semantics, graph meaning, and
source mutation remain owned by their separate capabilities.

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

QueryPolicy {
  default_max_bytes: integer > 0
  default_max_items: integer > 0
  hard_max_bytes: integer > 0
  hard_max_items: integer > 0
  default_context_lines: integer >= 0
  hard_context_lines: integer >= 0
}

PermissionPolicy {
  read_only: true
  allowed_operations: "status" | "search" | "evidence" | "source_context"[]
  transport: "stdio" | "local_http" | "authenticated_http"
  allow_target_execution: false
  allow_shell: false
}
```

Roots are normalized relative to the opened repository and cannot escape it.
The config is separate from layout, analyzer assignment, quality profile
semantics, and source-index facts. The quality profile is referenced, not
copied into live configuration.

## Watch events and invalidation

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
sequence uses `full_rescan`. The coordinator hashes/rescans before commit.

## LiveSnapshot

```text
LiveSnapshot {
  schema_version: "arch-view.live/v1"
  snapshot_id: OpaqueId
  revision: integer > 0
  state: "ready" | "degraded"
  source_input_fingerprint: ContentDigest
  source_index_ref?: OpaqueRef
  model_ref?: OpaqueRef
  quality_report_ref?: OpaqueRef
  scope_ids: OpaqueId[]
  diagnostics: LiveDiagnostic[]
  freshness: Freshness
  semantic_digest: ContentDigest
  extensions: ExtensionBlock[]
}

Freshness {
  status: "current" | "stale" | "updating" | "failed" | "initializing"
  last_ready_revision?: integer > 0
  pending_event_group_id?: OpaqueId
  changed_paths?: RelativePath[]
}

OpaqueRef {
  kind: "source_index" | "model" | "quality_report"
  id: OpaqueId
}
```

`revision` is monotonic within a session and is assigned only at atomic
publication. A snapshot's source index, model, quality report, scope set, and
fingerprint refer to the same candidate input. Operational freshness can
change while the last-ready semantic snapshot remains unchanged.

## Session lifecycle

```text
stopped -> initializing -> ready
ready -> stale -> updating -> ready | degraded
ready -> updating -> ready | degraded
degraded -> stale | updating -> ready | degraded
initializing -> failed
failed -> initializing
```

`stale`/`updating`/`failed` describe session freshness around the last-ready
revision. If no ready revision exists, queries return unavailable status. A
failed update never replaces the last-ready snapshot with an empty result.

## QueryEnvelope

```text
QueryEnvelope<T> {
  schema_version: "arch-view.query/v1"
  session_id: OpaqueId
  snapshot_id: OpaqueId
  revision: integer > 0
  consistency: "specific_revision" | "latest_ready"
  freshness: Freshness
  scope_ids: OpaqueId[]
  result: T
  result_count: integer >= 0
  omitted_fields: NamespacedId[]
  budget: BudgetUsage
  next_cursor?: OpaqueCursor
  diagnostics: QueryDiagnostic[]
}

BudgetUsage {
  max_bytes: integer > 0
  max_items: integer > 0
  emitted_bytes: integer >= 0
  emitted_items: integer >= 0
  truncated: boolean
}
```

Cursors are opaque and bound to session, snapshot/revision, query, canonical
ordering, and budget. A cursor from another context is rejected rather than
silently restarting.

## Structural queries

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
```

Structural matching is deterministic and operates on source-index/model/
quality facts. Semantic/fuzzy ranking is not implied by this query object.

## MCP adapter boundary

```text
WatchBackend {
  Start(WatchRoot[], WatchPolicy) -> EventStream
  Stop() -> void
}

SnapshotStore {
  ReadLatestReady(session_id) -> LiveSnapshot?
  PublishAtomically(candidate) -> LiveSnapshot
  ReadRevision(session_id, revision) -> LiveSnapshot?
}

QueryProvider {
  Execute(snapshot, StructuralQuery, Budget) -> QueryResult
  ReadEvidence(snapshot, EntityRef, Budget) -> QueryResult
  ReadSourceContext(snapshot, SourceSpan, ContextBudget) -> QueryResult
}

TransportAdapter {
  Serve(ReadOperation) -> QueryEnvelope
}
```

These focused interfaces make OS watcher, polling, in-memory/persisted store,
HTTP, stdio, and future transports substitutable. The coordinator depends on
the abstractions and does not grow transport or filesystem branches.

## MCP operation set

V1 read operations are `get_snapshot_status`, `list_scopes`, `find_files`,
`find_symbols`, `get_documentation`, `get_module_facts`,
`get_quality_findings`, `get_finding_evidence`, `get_callers_callees`, and
`get_source_context`. Each operation declares required capabilities and returns
unsupported/unknown status when they are unavailable.

## Atomic publication invariants

- A candidate is unpublished until all requested source/model/quality artifacts
  validate and share an input fingerprint.
- Readers see either the previous complete revision or the new complete
  revision, never a mixture.
- Revision numbers strictly increase for semantic input changes; a rescan with
  equal semantic digest need not create a new revision.
- Diagnostics and freshness may report a failed event group without replacing
  valid data.
- Query results identify the exact revision and do not silently move to a newer
  revision during pagination.

## Security invariants

- Every path is normalized and resolved beneath an allowed root.
- Symlink/junction traversal outside the root is rejected.
- Default MCP operations cannot execute a command, target application, or
  source mutation.
- Source context is explicit, bounded, and derived from an indexed span/entity;
  arbitrary local paths are not accepted.
- Full source text, hidden files, and quality internals are omitted unless the
  requested capability/permission and budget allow them.
