# Live analysis and MCP discovery notes

## Purpose

Keep a configured Arch View source/quality view current and expose the same
compact, evidence-backed snapshot to the viewer, CLI/export, and LLM tools.
The live surface must lower repeated repository-reading cost without making
the MCP server a second parser or quality engine.

## Observed in code

- The local host already performs bounded analyzer orchestration, per-scope
  caching, source-safe inspection, canonical model generation, and deterministic
  model/view output.
- The source-index child now defines compact files, symbols, documentation,
  spans, relations, provenance, scope snapshots, and deterministic digests.
- The deterministic-quality child now defines immutable quality reports,
  versioned rules/metrics, coverage, findings, baselines, and signal labeling.
- No folder watcher, live revision coordinator, MCP server, MCP tool/resource
  contract, or snapshot publication protocol exists in the current product.

## User-confirmed target behavior

- Configured folders should be monitored for selected quality violations.
- An LLM should be able to request concise findings, file/symbol search,
  callers/callees where available, documentation, and bounded source context.
- A shared search/reporting surface should reduce token spending by returning
  structured facts before source text.
- Findings should be reported to the LLM for an explicit downstream fix, not
  silently edited by the watcher or MCP server.

## Specified decisions

- A live session has an independent versioned configuration containing allowed
  roots, source/quality capability requests, debounce/resource limits,
  projection budgets, and read-only permissions. It is separate from layout,
  analyzer assignments, quality profile semantics, and source-index facts.
- A watcher adapter normalizes create/modify/delete/rename/overflow events into
  root-safe paths. Events are coalesced during a bounded debounce window;
  overflow or missed events trigger a complete rescan rather than guessed
  incremental state.
- The coordinator builds a candidate source/model/index/quality snapshot in
  isolation. The last committed ready snapshot remains served while work is in
  progress. A candidate is atomically committed only after validation and
  digest checks.
- Revisions are monotonic within a live session. A revision is internally
  consistent: source index, model, quality report, and freshness metadata refer
  to the same input set. Failed reanalysis keeps the last good snapshot and
  publishes explicit degraded diagnostics.
- Selective reanalysis may invalidate affected analyzer scopes when the
  existing cache/input identity proves it safe; otherwise the coordinator
  performs a full scope rescan. Incremental graph/semantic inference is never
  assumed from one filesystem event.
- The MCP server is a read-only transport/query adapter. Local stdio is the
  default packaging; network transports require explicit configuration and
  authentication/permission policy. No shell execution, target application
  execution, or source mutation is exposed in v1.
- Structural search is deterministic over source-index/model/quality facts:
  paths, languages, roles, symbols, documentation, relations, scopes, and
  findings. Full-text/semantic ranking is a separate future capability.
- Every MCP response carries snapshot ID/revision, freshness, scope, omitted
  fields, bounded result metadata, and continuation information. Byte/item
  budgets are deterministic; token counts are an adapter concern.
- Source context is requested by entity/span and explicit byte/line budget,
  remains root-safe, and is never included in default search/report responses.
- MCP exposes findings and evidence, not autonomous fixes. A later remediation
  workflow may return a patch proposal/dry-run and require explicit user
  authorization as a separate capability.
- Watch backends, snapshot stores, query providers, and transports are focused
  replaceable strategies. The coordinator is closed for modification when a
  new filesystem backend, query, transport, or report type is added.

## Boundary

This capability owns live configuration, watch lifecycle, event normalization,
debounce/coalescing, invalidation, snapshot consistency/revisions, compact
structural queries, MCP tools/resources, budgets, and read-only permissions. It
does not own language parsing, source facts, quality rules, graph semantics, or
source editing.

## Initial MCP surface

Resources: latest snapshot status, scope list, module facts, source-index
projections, and quality-report projections. Tools: `get_snapshot_status`,
`list_scopes`, `find_files`, `find_symbols`, `get_documentation`,
`get_module_facts`, `get_quality_findings`, `get_finding_evidence`,
`get_callers_callees` when advertised, and `get_source_context` with a budget.
All return explicit unsupported/unknown coverage and are read-only.

## Implementation and verification focus

Implement session/config validation, a polling/fake watcher contract,
event-coalescing state machine, immutable snapshot store, revision-aware query
envelopes, stdio MCP adapter, root-safe source context, and deterministic
budgets. Verify atomic visibility during updates, overflow recovery, partial
failure retention, stale/degraded status, query consistency, scope/path
containment, unsupported call graphs, and no mutation/command execution.
