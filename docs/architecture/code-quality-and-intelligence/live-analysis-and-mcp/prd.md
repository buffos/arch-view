# Live analysis and MCP PRD

## Purpose

Keep a configured source/architecture/quality view current and give developers
and coding assistants a compact, revision-aware query surface. The design
minimizes repeated source loading by returning structured facts first and
bounded source context only on demand.

## Actors

- **Developer:** watches selected folders and sees current files, symbols, and
  quality findings in the viewer.
- **CI/CLI operator:** requests a stable revision or waits for a ready snapshot
  and exports bounded reports.
- **Coding assistant/LLM:** searches structure, documentation, findings, and
  available callers/callees before requesting relevant source context.
- **Watcher backend:** reports filesystem hints through a replaceable adapter.
- **Snapshot coordinator:** coalesces events, runs safe reanalysis, and
  publishes coherent immutable revisions.
- **MCP client:** consumes the shared read-only tools/resources.
- **Future remediation workflow:** receives an explicit finding handoff; it is
  not part of v1 live analysis.

## Goals

1. Watch explicitly configured roots and keep a coherent latest-ready snapshot.
2. Coalesce events, recover from overflow, and avoid mixed-revision results.
3. Reuse existing bounded orchestration/caches when selective invalidation is
   proven safe, and fall back conservatively when it is not.
4. Share one source-index/model/quality read model with viewer, CLI/export, and
   MCP.
5. Provide deterministic structural file/symbol/documentation/finding search.
6. Provide callers/callees or other semantic queries only when advertised by
   source-index capabilities, with explicit unsupported/unknown status.
7. Bound response bytes/items and omit full source content by default.
8. Enforce configured-root, read-only, no-shell, and no-target-execution
   permissions.
9. Report current, stale, updating, degraded, and failed states honestly.
10. Keep a future explicit fix/remediation handoff separate from analysis.

## Non-goals

- Replacing analyzers, source-index extractors, quality rules, or canonical
  model normalization.
- Runtime tracing, execution of target applications, or arbitrary command
  execution from MCP.
- Default full-text/embedding search, semantic ranking, or unbounded source
  retrieval.
- Automatic source edits, commits, refactors, or autonomous LLM fixes.
- Guaranteeing incremental semantic correctness from one filesystem event
  without a validated source/input fingerprint.

## Functional requirements

| ID | Requirement |
|---|---|
| LAM-FR-001 | Validate a versioned live session with explicit allowed roots, source/quality requests, debounce/resource limits, budgets, and permissions. |
| LAM-FR-002 | Normalize backend events to root-safe paths and coalesce them within a bounded debounce policy. |
| LAM-FR-003 | Recover watcher overflow/errors by rescanning rather than committing guessed state. |
| LAM-FR-004 | Build candidate source/model/quality snapshots in isolation and atomically publish only coherent validated revisions. |
| LAM-FR-005 | Serve the last-ready revision while updates run or fail, with explicit freshness/degraded diagnostics. |
| LAM-FR-006 | Use content/source-scope/cache identity to select safe affected scopes; conservatively rescan when impact is uncertain. |
| LAM-FR-007 | Expose one versioned read model to viewer, CLI/export, and MCP without duplicating analysis semantics. |
| LAM-FR-008 | Provide deterministic structural search/filtering with opaque cursors tied to snapshot/query/order/budget. |
| LAM-FR-009 | Provide bounded quality findings/evidence and callers/callees only when capabilities support them. |
| LAM-FR-010 | Include revision, snapshot, scope, freshness, coverage, omitted-fields, and budget metadata in every query response. |
| LAM-FR-011 | Keep MCP v1 read-only, root-safe, no-shell, and no-target-execution. |
| LAM-FR-012 | Require explicit bounded source-context requests and prohibit arbitrary paths outside configured roots. |
| LAM-FR-013 | Make watcher, store, query, and transport adapters replaceable strategies. |
| LAM-FR-014 | Define an explicit future remediation handoff without implementing source mutation here. |

## Initial read operations

- `get_snapshot_status`
- `list_scopes`
- `find_files`
- `find_symbols`
- `get_documentation`
- `get_module_facts`
- `get_quality_findings`
- `get_finding_evidence`
- `get_callers_callees` when capability coverage is observed
- `get_source_context` with explicit byte/line limits

## Non-functional requirements

- **Consistency:** one response is tied to one requested/selected revision.
- **Availability:** transient updates do not remove the last ready view.
- **Determinism:** equal snapshot/query/filter/order/budget inputs produce equal
  result ordering and cursor behavior.
- **Safety:** roots are allowlisted, symlink escapes rejected, operations are
  read-only, and response/source budgets are enforced.
- **Token efficiency:** structured compact fields first; full source explicit.
- **Extensibility:** new watcher/query/transport capabilities register without
  changing the coordinator or existing payloads.

## Success criteria

The scenarios in [acceptance-scenarios.md](acceptance-scenarios.md) pass for
watch lifecycle, debounce/overflow, atomic revisions, partial failures,
structural search, quality/call capability reporting, budgets, permissions,
MCP parity, and no mutation. The application synthesis records this child as a
consumer of the specified source-index and quality contracts.
