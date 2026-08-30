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
    "capabilities": [
      "source:files",
      "source:declarations",
      "source:documentation"
    ]
  },
  "quality_request": {
    "profile_id": "profile:default",
    "profile_version": "1.0.0"
  },
  "watch_policy": {
    "debounce_ms": 250,
    "max_pending_events": 10000,
    "max_parallel_scopes": 4,
    "rescan_interval_ms": 30000
  },
  "freshness_policy": {
    "default_consistency": "latest_ready",
    "settle_ms": 250,
    "max_wait_ms": 5000,
    "max_stability_retries": 2,
    "reconcile_interval_ms": 30000
  },
  "query_policy": {
    "default_max_bytes": 16384,
    "default_max_items": 50,
    "hard_max_bytes": 1048576,
    "hard_max_items": 5000,
    "default_context_lines": 40,
    "hard_context_lines": 1000
  },
  "permission_policy": {
    "default_mode": "read_only",
    "allowed_operations": [
      "status",
      "search",
      "evidence",
      "source_context",
      "ensure_current",
      "quality_evaluate",
      "quality_profile_read",
      "baseline_read"
    ],
    "transport": "stdio",
    "allow_target_execution": false,
    "allow_shell": false,
    "audit_policy_writes": true
  },
  "extensions": []
}
```

The host may store this object in a future `live` configuration section or pass
it to a session API. It must not copy the full quality profile into the live
config; the profile is referenced by ID/version. Analyzer assignments and
language semantics remain owned by the analysis/plugin contract.

## Snapshot status response

```json
{
  "schema_version": "arch-view.query/v1",
  "session_id": "opaque-session-id",
  "snapshot_id": "opaque-snapshot-id",
  "revision": 12,
  "requested_consistency": "require_current",
  "returned_consistency": "current",
  "freshness": {
    "status": "current",
    "reconciliation": "passed",
    "last_ready_revision": 12,
    "changed_paths": []
  },
  "scope_ids": ["opaque-scope-id"],
  "result": {
    "state": "ready",
    "source_input_fingerprint": { "algorithm": "hash:sha-256", "value": "..." },
    "input_verification": "verified",
    "diagnostics": []
  },
  "result_count": 1,
  "omitted_fields": [],
  "capabilities": [],
  "budget": {
    "max_bytes": 16384,
    "max_items": 50,
    "emitted_bytes": 420,
    "emitted_items": 1,
    "truncated": false
  },
  "diagnostics": []
}
```

`freshness.status` describes the session relative to the revision returned.
`require_current` is satisfied only after source reconciliation and stable
input verification. If no ready revision exists, or the source remains
unstable/analysis times out, the server returns structured `no_ready_snapshot`,
`input_unstable`, or `analysis_wait_timeout` status rather than an empty
successful current result.

## Structural and exact-text query requests

```json
{
  "schema_version": "arch-view.query/v1",
  "session_id": "opaque-session-id",
  "consistency": "require_current",
  "revision": null,
  "query": {
    "path_prefix": "src/",
    "language": "language:go",
    "symbol_categories": ["callable"],
    "name": "Handle",
    "documentation_text": "configuration",
    "case_sensitive": true
  },
  "projection": [
    "id",
    "path",
    "name",
    "location",
    "documentation_summary"
  ],
  "max_bytes": 8192,
  "max_items": 20,
  "cursor": null
}
```

The `language` field is an optional filter, not an MCP implementation branch.
All registered analyzers may contribute records. Structural filters are
deterministic over source-index/model/quality records. Default ordering is
scope, path, source span, category, name, and opaque ID. Fuzzy/semantic ranking
is not part of v1. The cursor is opaque and bound to the session, revision,
query, projection, ordering, consistency, and budget.

An exact text search uses the same envelope:

```json
{
  "pattern": "TODO",
  "mode": "literal",
  "path_glob": "**/*",
  "language": null,
  "case_sensitive": false,
  "max_line_bytes": 512,
  "max_bytes": 8192,
  "max_items": 20
}
```

The regex mode is explicitly bounded and uses the server's safe regex policy.
Results contain repository-relative file paths and line/column locations, not
whole files.

## Query result envelope

Every search/finding/evidence response uses `arch-view.query/v1` and includes:

- session, snapshot, revision, and requested/returned consistency;
- freshness and scope IDs;
- compact typed result;
- capability/coverage states;
- result count, omitted fields, budget usage, and optional cursor;
- diagnostics.

If requested callers/callees, documentation, metrics, or other analyzer facts
are unavailable, the response uses `unsupported`, `unknown`, `partial`, or
`not_evaluable` coverage. It does not return an empty success that looks like
“no callers” or “no documentation”.

## Quality evaluation and policy requests

Quality semantics remain owned by the deterministic-quality capability. MCP
maps to its existing catalog, evaluation, comparison, and policy services.

```json
{
  "schema_version": "arch-view.quality-evaluation-request/v1",
  "session_id": "opaque-session-id",
  "consistency": "require_current",
  "scope_ids": ["opaque-scope-id"],
  "profile_id": "profile:full",
  "profile_version": "1.0.0",
  "rule_bindings": [
    {
      "rule_id": "source:file.max-lines",
      "rule_version": "1.0.0",
      "enabled": true,
      "parameters": {
        "namespace": "rule-config:source-file-size",
        "schema_version": "1.0.0",
        "payload": { "operator": "greater_than", "limit": 500, "unit": "unit:line" }
      }
    }
  ],
  "persist": false
}
```

Temporary `rule_bindings` are validated and evaluated without changing the
project profile. A profile save or baseline command is a different operation:

```json
{
  "operation": "create_baseline",
  "session_id": "opaque-session-id",
  "report_revision": 12,
  "profile_id": "profile:full",
  "profile_version": "1.0.0",
  "finding_keys": ["opaque-stable-finding-key"],
  "reason": "Reviewed intentional legacy coupling",
  "destination": "quality-baselines/legacy-coupling.json",
  "authorization": "opaque-authorization"
}
```

Policy commands require an exact compatible current report, complete profile/
rule/formula identity, explicit operation permission, validated destination, a
reason where applicable, and an audit record. `preview_baseline` is read-only;
`create_baseline` never means “suppress everything”.

## MCP server mapping

The default local packaging is an MCP stdio server launched explicitly by the
user/host. `local_http` or `authenticated_http` are opt-in transports and must
enforce the same root/operation policy. Tool names and input semantics are:

| Tool | Required input | Result | Permission |
|---|---|---|---|
| `get_snapshot_status` | session, consistency, optional wait/revision | state, freshness, last-ready/current revision, diagnostics | status |
| `ensure_current_snapshot` | session, optional scopes and wait budget | verified revision or explicit unstable/unavailable result | ensure_current |
| `list_scopes` | session/revision | roots, analyzer IDs, capabilities, status | search |
| `find_files` | structural query, projection, budget | file records and cursor | search |
| `find_symbols` | structural query, projection, budget | symbol records/locations/documentation status | search |
| `find_text` | literal/regex query, root-safe filters, budget | bounded matches and locations | search |
| `get_documentation` | subject/query, projection, budget | documentation candidates/status/spans | search |
| `get_module_facts` | module ref, scope/revision, budget | containment-linked files/symbols/docs | search |
| `get_quality_profiles` | optional scope/revision | profile identities and validation status | quality_profile_read |
| `get_quality_rules` | optional profile and revision | complete catalog, parameters, capability status | quality_profile_read |
| `get_quality_findings` | rule/severity/status/scope filters, consistency, budget | findings, coverage, evidence summaries | search/evidence |
| `get_finding_evidence` | finding ref, context budget | metrics, spans, relations, optional context | evidence |
| `get_callers_callees` | symbol ref, direction, scope/revision | occurrences/relations or explicit unsupported status | search |
| `evaluate_quality` | profile, optional temporary bindings, consistency | immutable quality report or explicit coverage/configuration result | quality_evaluate |
| `compare_quality_reports` | two compatible report revisions | added/unchanged/suppressed/resolved transitions | search/evidence |
| `validate_quality_profile` | profile or bindings | validation result | quality_profile_read |
| `preview_baseline` | report, selected finding keys, reason | dry-run baseline entries | baseline_read |
| `save_quality_profile` / `save_quality_profile_as` | validated profile, destination, authorization | persisted profile identity/audit result | quality_profile_write |
| `create_baseline` | current report, finding keys, destination, authorization | persisted baseline identity/audit result | baseline_write |
| `get_source_context` | indexed entity/span, line/byte budget | bounded text with matching hash | source_context |

No v1 tool writes source files, runs shell commands, executes the target
application, installs packages, or changes analyzer assignments. Read-only
resource URIs may expose only compact projections such as:

- `archview://session/{session_id}/status`
- `archview://session/{session_id}/snapshot/{revision}`
- `archview://session/{session_id}/scopes`
- `archview://session/{session_id}/quality`

Resource content uses the same versioned envelopes and budgets as tools.

## Recommended HTTP mapping

If an HTTP adapter is enabled, it may map the same operations to:

- `GET /v1/live/{session_id}/status`
- `POST /v1/live/{session_id}/ensure-current`
- `GET /v1/live/{session_id}/scopes`
- `POST /v1/live/{session_id}/search/files`
- `POST /v1/live/{session_id}/search/symbols`
- `POST /v1/live/{session_id}/search/text`
- `POST /v1/live/{session_id}/search/documentation`
- `GET /v1/live/{session_id}/quality`
- `POST /v1/live/{session_id}/quality/evaluate`
- `POST /v1/live/{session_id}/quality/compare`
- `POST /v1/live/{session_id}/evidence`
- `POST /v1/live/{session_id}/source-context`

These are transport mappings, not a requirement for the first local
implementation. Authentication, origin policy, root permissions, and policy
write authorization must be explicit for network transport.

## Watch and freshness behavior

- Backend events are hints and are normalized to root-safe relative paths.
- Events within the configured bounded debounce window are coalesced and
  deduplicated.
- Overflow, backend error, ambiguous rename, or missed sequence forces a full
  rescan of the affected session/scope.
- A periodic reconciliation scan protects against missed watcher events even
  when the backend reports no error. For `require_current`, a clean watcher or
  metadata-only manifest is a fast path, not proof, unless its content/input
  identity is authoritative for the eligible source set.
- `latest_ready` returns the last committed revision quickly and labels it
  stale/updating when pending changes exist.
- `require_current` performs authoritative reconciliation before query
  execution. If a change is found, it starts or joins one single-flight
  reanalysis and waits within the request budget.
- A candidate is validated against the input manifest/content fingerprint both
  before and after analysis. If files changed during the build, the candidate
  is discarded and retried; repeated changes return `input_unstable`.
- Readers continue to receive the previous ready revision during an update.
- A failed update keeps the previous revision and reports stale/degraded
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
    "message": "The requested operation is not enabled for this session",
    "details": {}
  }
}
```

Stable live/query error codes include `live_config_invalid`,
`watch_root_invalid`, `watch_event_out_of_scope`, `watch_event_overflow`,
`source_reconciliation_failed`, `input_unstable`, `no_ready_snapshot`,
`revision_unavailable`, `analysis_wait_timeout`, `query_budget_exceeded`,
`query_cursor_invalid`, `source_context_out_of_scope`,
`quality_evaluation_invalid`, `quality_policy_permission_denied`,
`quality_profile_conflict`, `baseline_revision_stale`,
`mcp_operation_unsupported`, and `mcp_permission_denied`.

## Determinism and token policy

The source/model/quality snapshot IDs, input fingerprint, revision,
query/filter/projection, canonical ordering, effective profile/options, and
byte/item budget determine the semantic response. The server omits raw source
and verbose diagnostics unless explicitly requested. Operation-specific compact
defaults keep agent responses small; hard byte/item limits are authoritative.
Token estimates may be returned as advisory adapter metadata but cannot override
the byte/item limits. Viewer, CLI, and MCP use the same query/quality providers
and therefore cannot disagree about structural facts or report semantics.

## Remediation boundary

MCP does not edit source. A future `request_remediation_handoff` capability may
include finding key, selected revision, evidence, proposed patch/dry-run,
explicit authorization, and audit data. Any resulting external edit is
observed as a new filesystem event and re-evaluated; it is not an implicit MCP
side effect.
