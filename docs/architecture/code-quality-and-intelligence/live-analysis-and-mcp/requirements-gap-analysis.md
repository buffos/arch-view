# Live analysis and MCP requirements gap analysis

## Scope examined

This pass covers the specified live-analysis/MCP child, the implemented
multi-analyzer orchestration and caching, the source-index and deterministic-
quality contracts, source-safe inspection, and the viewer/CLI/coding-assistant
consumers.

## Confirmed strong areas

- **Observed in code:** analyzer jobs already have bounded execution,
  cancellation, source scopes, cache identity, and partial-failure behavior
  that a live coordinator can reuse across languages.
- **Observed in the specified contracts:** source facts and quality reports are
  immutable, scope-aware, hash/version-backed, capability-aware, and compact by
  default. Quality services already define profile validation, temporary rule
  bindings, report comparison, and baseline creation.
- **Inferred from architecture:** one revision coordinator can publish a
  shared snapshot envelope for viewer, CLI/export, and MCP without duplicating
  language or quality semantics.
- **User-confirmed target behavior:** agents need a low-token path from finding
  to exact source location, verified freshness after edits, configurable
  quality evaluation, and explicit baseline control across all analyzers.

## Blocking gaps resolved by this specification

| Gap | Impact | Resolution |
|---|---|---|
| Watcher events could be mistaken for repository truth | A missed event or editor save storm could leave a report falsely marked current | Events are hints; strict requests reconcile a root-safe manifest, verify content/input fingerprints, and publish only stable candidates. |
| Concurrent edits could be published as a coherent-looking snapshot | Source facts and quality findings might describe different file contents | Read/hash verification detects changes during a scan; the coordinator retries or returns `input_unstable` and never labels it current. |
| Repeated strict requests could start duplicate analyses | Agents and UI clients could multiply expensive work | A single-flight coordinator joins requests for the same target revision and coalesces pending events. |
| Freshness behavior was caller-dependent | A client could forget to call status before querying findings | Every query accepts a consistency mode; `require_current` performs the freshness preflight itself. |
| The original surface was easy to read as Go-specific | Other registered analyzers would be second-class MCP providers | Capabilities, scopes, languages, symbols, metrics, and coverage are negotiated and projected generically; no analyzer-specific branch belongs in MCP. |
| Structural search did not cover cheap literal navigation | Agents still needed large file reads for TODOs, strings, and config keys | Add bounded root-safe literal/regex text search with line locations; fuzzy/embedding ranking remains deferred. |
| The agent fix loop stopped at evidence | There was no specified way to observe the result after an external edit | Add bounded wait/revision status and quality-report comparison between revisions. |
| Quality profiles and baselines were only described as read-only MCP | The model could not request a one-off evaluation or perform an authorized policy action | Delegate catalog, evaluation, profile save, baseline preview, and baseline creation to existing quality services with separate permissions and audit data. |
| Baselines could be created from stale reports | A baseline might suppress a finding that no longer describes the source | Require an exact current report/profile/rule version and reject stale or incompatible baseline requests. |
| Token budgets were global rather than agent-shaped | A safe 64 KiB response can still pollute an agent context | Use compact operation-specific projections and explicit include fields; byte/item ceilings remain authoritative. |

## Deferrable implementation details

- The first watcher may use an OS backend, polling, or a test fake behind the
  adapter; event normalization and request-time reconciliation are the stable
  contracts.
- The snapshot store may be in memory or persisted; atomic publish, input
  verification, and revision/query semantics are mandatory.
- MCP resource URI naming and exact SDK plumbing can vary while tool payloads
  remain versioned and bounded.
- Fuzzy, embedding, and semantic ranking may be added later as declared
  capabilities without changing deterministic structural or exact-text search.
- A source-edit/remediation handoff may be added later. It must remain
  separate from live analysis and must not turn MCP into a shell or target
  execution channel.

## Exact assumptions

- A live session's allowed roots must be descendants of the opened repository
  or explicitly selected local roots; paths are normalized before watch/query.
- Symlinks, junctions, and aliases that resolve outside the allowlist are
  rejected.
- Create/modify/delete/rename events are hints. A manifest/dirty-state check is
  an optimization; authoritative content/input fingerprints determine
  committed truth for strict requests.
- Debounce is bounded. A watcher overflow, backend error, ambiguous rename, or
  failed reconciliation forces a full rescan of the affected session/scope.
- A `latest_ready` query may return the last ready revision with explicit
  stale/updating/degraded metadata. A `require_current` query must reconcile,
  wait within its budget, and return unavailable/unstable status if it cannot
  obtain a verified current revision.
- One published revision contains source index, model, quality report,
  effective profile/options, scope set, and input fingerprint for the same
  source state. A profile-only evaluation may reuse a verified source revision,
  but its report fingerprint names the selected profile/options exactly.
- Default response budgets are compact and operation-specific; a configured
  hard byte/item maximum is authoritative. Token estimates are advisory.
- Source context requires an indexed entity/span, selected revision, and
  maximum line/byte budget. Arbitrary paths outside the allowlist are rejected.
- Read operations are enabled by default. Quality evaluation is a separately
  allowlisted operation; profile and baseline writes require explicit policy
  permissions. MCP never edits source, runs shell commands, or executes the
  target application.

## Readiness conclusion

No High or Medium specification gaps remain for the refreshed live/MCP
contract. The remaining work is implementation of watcher/reconciliation,
revision storage, analyzer-neutral query projections, quality-service
delegation, permissions, and transport packaging.

## Artifact impact

- **Capability:** this exact-spec set now defines freshness assurance,
  analyzer-neutral search, quality control operations, policy permissions, and
  the agent fix/reanalysis loop in addition to lifecycle/query behavior.
- **Product:** developers and coding assistants share compact current-or-
  explicitly-not-current data across all registered analyzers.
- **Architecture:** the live coordinator, analyzers, source index, quality
  services, MCP adapter, and coding-agent edit workflow have separate owners.
- **Delivery:** no implementation issues are created in this planning pass.
