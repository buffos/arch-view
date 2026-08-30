# Live analysis and MCP requirements gap analysis

## Scope examined

This pass covers the bounded live-analysis/MCP child, existing analyzer
orchestration and caching, the specified source-index and quality-report
contracts, source-safe inspection, and the future viewer/CLI/LLM consumers.

## Confirmed strong areas

- **Observed in code:** analyzer jobs have bounded execution, cancellation,
  source scopes, cache identity, and partial-failure behavior that a live
  coordinator can reuse.
- **Observed in the specified contracts:** source facts and quality reports are
  immutable, scope-aware, hash/version-backed, and compact by default.
- **Inferred from architecture:** a revision coordinator can publish one
  snapshot envelope consumed by viewer, CLI/export, and MCP without duplicating
  language or quality semantics.
- **User-confirmed target behavior:** configured folders, structural search,
  quality reporting, callers/callees where available, and bounded context are
  needed for low-token LLM workflows.

## Blocking gaps resolved by this specification

| Gap | Impact | Resolution |
|---|---|---|
| Watcher event semantics were undefined | High: missed/duplicate events could produce stale or mixed facts | Normalize events, debounce/coalesce, recover overflow with full rescan, and hash-verify candidate inputs. |
| Snapshot consistency was undefined | High: clients could combine old source facts with new findings | Build candidate snapshots in isolation and atomically commit one coherent revision. |
| Failure behavior could destroy the last usable view | High: a transient parse/process error would remove useful context | Keep last committed ready snapshot and expose degraded diagnostics/freshness. |
| Selective invalidation could be unsound | High: partial reanalysis might miss import/graph changes | Reuse only proven cache/input identities; otherwise rescan the full affected scope. |
| MCP could become a second analysis implementation | High: viewer/CLI/LLM results would diverge | MCP is a transport/query adapter over the shared snapshot/read-model boundary. |
| Search output could exhaust tokens | High: LLM workflows would repeatedly receive source/code blobs | Default structural projections, byte/item budgets, cursors, and explicit source context. |
| Permissions/path scope were undefined | High: an MCP server could expose arbitrary local files or execute commands | Configured-root allowlist, symlink escape prevention, read-only default, no shell/target execution. |
| Current/freshness status was ambiguous | Medium: consumers could mistake stale data for current findings | Explicit session/revision states, source revision, last-ready revision, and diagnostics. |
| Future transports could force core rewrites | Medium: stdio, HTTP, and embedded callers would diverge | Focused transport adapters consume one versioned query envelope. |

## Deferrable implementation details

- The first watcher may use an OS backend, polling, or a test fake behind the
  adapter; event normalization is the stable contract.
- The snapshot store may be in memory or persisted; atomic publish and
  revision/query semantics are mandatory.
- MCP resource URI naming and exact SDK plumbing can vary while tool payloads
  remain versioned and bounded.
- A future semantic/fuzzy search provider may be added as a declared capability
  without changing deterministic structural query behavior.

## Exact assumptions

- A live session's allowed roots must be descendants of the opened repository
  or explicitly selected local roots; paths are normalized before watch/query.
- Symlinks are not followed outside the allowed root. Junction/alias escape is
  treated the same as a path escape.
- Create/modify/delete/rename events are hints. Content hash and source-scope
  rescan determine the committed truth.
- Default debounce is implementation-configurable but bounded; event overflow,
  watcher error, or an untrusted rename forces a full rescan of the affected
  session/scope.
- A revision is published only after source index, canonical model, and any
  requested quality report validate against the same input fingerprint.
- If no ready revision exists, status is `initializing`/`failed` and queries
  return structured unavailable status rather than an empty successful result.
- If an update fails after a ready revision exists, queries default to that
  last-ready revision and include `stale`/`degraded` metadata.
- Default response budget is 64 KiB and 200 items, with a configurable hard
  maximum of 1 MiB and 5,000 items per call. The byte budget is authoritative;
  token estimates are optional adapter metadata.
- Source context requires an indexed entity/span, a matching snapshot or
  explicit latest selector, and a maximum byte/line budget. It cannot accept an
  arbitrary filesystem path outside the allowlist.
- MCP v1 exposes no write/execute tool. A future fix handoff is a separate
  permissioned capability with preview, explicit confirmation, and audit data.

## Readiness conclusion

No High or Medium specification gaps remain for the bounded live/MCP contract.
The remaining work is implementation of watcher adapters, revision store,
query projections, and transport packaging.

## Artifact impact

- **Capability:** this exact-spec set defines lifecycle, invalidation, snapshot,
  query, MCP, budget, and security behavior.
- **Product:** one current/explicitly stale compact view is shared by UI, CLI,
  and LLM tools.
- **Architecture:** watcher, snapshot coordinator, shared read model, and MCP
  adapters have separate ownership.
- **Delivery:** no implementation issues are created in this planning pass.
