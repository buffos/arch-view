# Live analysis and MCP canonical API/CLI contract

## Version and configuration

The live contract is independently versioned as `arch-view.live/v1`. It is
separate from layout, analyzer assignment, source-index, and quality profile
semantics.

```json
{
  "schema_version": "arch-view.live/v1",
  "session_id": "opaque-session-id",
  "repository_root": ".",
  "watch_roots": [
    { "path": "src", "recursive": true }
  ],
  "source_index_request": {
    "enabled": true,
    "capabilities": ["source:files", "source:declarations", "source:documentation"]
  },
  "quality_request": {
    "profile_id": "profile:default",
    "profile_version": "1.0.0"
  },
  "watch_policy": {
    "debounce_ms": 250,
    "max_pending_events": 10000,
    "max_parallel_scopes": 4
  },
  "query_policy": {
    "default_max_bytes": 65536,
    "default_max_items": 200,
    "hard_max_bytes": 1048576,
    "hard_max_items": 5000,
    "default_context_lines": 80,
    "hard_context_lines": 1000
  },
  "permission_policy": {
    "read_only": true,
    "allowed_operations": ["status", "search", "evidence", "source_context"],
    "transport": "stdio",
    "allow_target_execution": false,
    "allow_shell": false
  },
  "extensions": []
}
```

The host may store this object in a future `live` configuration section or pass
it to a session API. It must not copy the full quality profile into the live
config; the profile is referenced by ID/version.

## Snapshot status response

```json
{
  "schema_version": "arch-view.query/v1",
  "session_id": "opaque-session-id",
  "snapshot_id": "opaque-snapshot-id",
  "revision": 12,
  "consistency": "latest_ready",
  "freshness": {
    "status": "stale",
    "last_ready_revision": 12,
    "pending_event_group_id": "opaque-event-group",
    "changed_paths": ["src/catalog/service.go"]
  },
  "scope_ids": ["opaque-scope-id"],
  "result": {
    "state": "ready",
    "source_input_fingerprint": { "algorithm": "hash:sha-256", "value": "..." },
    "diagnostics": []
  },
  "result_count": 1,
  "omitted_fields": [],
  "budget": {
    "max_bytes": 65536,
    "max_items": 200,
    "emitted_bytes": 420,
    "emitted_items": 1,
    "truncated": false
  },
  "diagnostics": []
}
```

`freshness.status` describes the session relative to its last ready revision;
the response always identifies the exact revision returned. If no ready
revision exists, the server returns a structured `no_ready_snapshot` result
instead of an empty successful snapshot.

## Structural query request

```json
{
  "schema_version": "arch-view.query/v1",
  "session_id": "opaque-session-id",
  "revision": 12,
  "query": {
    "path_prefix": "src/",
    "language": "language:go",
    "symbol_categories": ["callable"],
    "name": "Handle",
    "case_sensitive": true
  },
  "projection": ["id", "path", "name", "location", "documentation_summary"],
  "max_bytes": 32768,
  "max_items": 100,
  "cursor": null
}
```

Structural filters are deterministic over source-index/model/quality records.
Default ordering is scope, path, source span, category, name, and opaque ID.
Fuzzy/semantic ranking is not part of v1. The cursor is opaque and bound to the
session, revision, query, projection, ordering, and budget.

## Query result envelope

Every search/finding/evidence response uses `arch-view.query/v1` and includes:

- session, snapshot, revision, and consistency selector;
- freshness and scope IDs;
- compact typed result;
- result count, omitted fields, budget usage, and optional cursor;
- capability/coverage and diagnostics.

If the requested callers/callees or documentation capability is unavailable,
the response uses `unsupported`/`unknown` coverage. It does not return an empty
success that looks like “no callers” or “no documentation”.

## MCP server mapping

The default local packaging is an MCP stdio server launched explicitly by the
user/host. `local_http` or `authenticated_http` are opt-in transports and must
enforce the same root/operation policy. Tool names and input semantics are:

| Tool | Required input | Result |
|---|---|---|
| `get_snapshot_status` | session, optional revision | state, freshness, last-ready revision, diagnostics |
| `list_scopes` | session, optional revision | scope IDs, roots, analyzer provenance/status |
| `find_files` | structural query, projection, budget | file records and continuation cursor |
| `find_symbols` | structural query, projection, budget | symbol records/locations/visibility |
| `get_documentation` | subject/query, projection, budget | documentation candidates/status/evidence |
| `get_module_facts` | module ref, scope/revision, budget | containment-linked files/symbols/docs |
| `get_quality_findings` | rule/severity/status/scope filters, budget | findings, coverage, evidence summaries |
| `get_finding_evidence` | finding ref, context budget | metrics, spans, relations, optional context |
| `get_callers_callees` | symbol ref, direction, scope/revision | occurrences/relations or explicit unsupported status |
| `get_source_context` | entity/span, line/byte budget | bounded text with matching file hash |

No v1 tool writes files, runs shell commands, executes the target application,
installs packages, or changes configuration. Resource URIs may expose read-only
projections such as:

- `archview://session/{session_id}/status`
- `archview://session/{session_id}/snapshot/{revision}`
- `archview://session/{session_id}/scopes`
- `archview://session/{session_id}/quality`

Resource content uses the same versioned envelopes and budgets as tools.

## Recommended HTTP mapping

If an HTTP adapter is enabled, it may map the same operations to:

- `GET /v1/live/{session_id}/status`
- `GET /v1/live/{session_id}/scopes`
- `POST /v1/live/{session_id}/search/files`
- `POST /v1/live/{session_id}/search/symbols`
- `POST /v1/live/{session_id}/search/documentation`
- `GET /v1/live/{session_id}/quality`
- `POST /v1/live/{session_id}/evidence`
- `POST /v1/live/{session_id}/source-context`

These are transport mappings, not a requirement for the first local
implementation. Authentication, origin policy, and root permissions must be
explicit for network transport.

## Watch/revision behavior

- Backend events are hints and are normalized to root-safe relative paths.
- Events within the configured bounded debounce window are coalesced.
- Overflow, backend error, ambiguous rename, or missed sequence forces a full
  rescan of the affected session/scope.
- A candidate source/model/quality bundle is validated before publication.
- Readers continue to receive the previous ready revision during an update.
- A failed update keeps the previous revision and reports `stale`/`degraded`
  diagnostics; it never publishes an empty replacement.
- A rescan with equal semantic input/digest may acknowledge the event without
  incrementing the semantic revision.
- A specific-revision query is immutable and pagination cannot drift.

## Source-context safety

Source context requests must contain an indexed entity/span, selected
revision/snapshot, and explicit line/byte limits. The server verifies file
content hash where available, clamps limits to the hard maximum, normalizes the
path beneath an allowed root, and returns the matching scope/provenance. An
arbitrary path or path traversal is rejected.

## Errors

The common error envelope remains:

```json
{
  "error": {
    "code": "mcp_permission_denied",
    "message": "The requested operation is not enabled for this read-only session",
    "details": {}
  }
}
```

Stable live/query error codes include `live_config_invalid`,
`watch_root_invalid`, `watch_event_out_of_scope`, `watch_event_overflow`,
`no_ready_snapshot`, `revision_unavailable`, `query_budget_exceeded`,
`query_cursor_invalid`, `source_context_out_of_scope`,
`mcp_operation_unsupported`, and `mcp_permission_denied`.

## Determinism and token policy

The source/model/quality snapshot IDs, revision, query/filter/projection,
canonical ordering, and byte/item budget determine the semantic response. The
server omits raw source and verbose diagnostics unless explicitly requested.
Token estimates may be returned as advisory metadata but cannot override the
authoritative byte/item limits. Viewer, CLI, and MCP use the same query
provider and therefore cannot disagree about structural facts.

## Remediation handoff

There is no v1 write tool. A future `request_remediation_handoff` capability
must include finding key, selected revision, proposed patch/dry-run, explicit
authorization, and audit data. Any resulting edit is observed as a new
filesystem event and re-evaluated; it is not an implicit MCP side effect.
