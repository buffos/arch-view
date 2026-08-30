# Live analysis and MCP PRD

## Purpose

Keep a configured source/architecture/quality view current across all
registered analyzers and give developers and coding assistants a compact,
revision-aware query and controlled quality-policy surface. The design reduces
repeated repository reading by returning structured facts first and bounded
source context only on demand.

## Actors

- **Developer:** watches selected folders and sees current files, symbols, and
  quality findings in the viewer.
- **CI/CLI operator:** requests a stable revision, waits for current data, or
  exports a bounded report.
- **Coding assistant/LLM:** searches analyzer-neutral structure or exact text,
  follows findings to evidence and source locations, edits through its own
  coding tools, and checks the next revision.
- **Watcher backend:** reports filesystem hints through a replaceable adapter.
- **Snapshot coordinator:** coalesces events, reconciles source state, runs
  safe multi-analyzer reanalysis, and publishes coherent immutable revisions.
- **MCP client:** consumes the shared bounded surface. It receives read access
  by default and may receive separately granted analysis or quality-policy
  operations.
- **Quality policy owner:** explicitly approves profile persistence or baseline
  changes when those operations are enabled.
- **Future remediation workflow:** may receive an explicit finding handoff; it
  is not a source-edit side effect of live analysis.

## Goals

1. Watch explicitly configured roots across all registered analyzers and keep a
   coherent latest-ready snapshot.
2. Treat watcher events as hints, reconcile source state for strict requests,
   and never label an unstable or unverified input current.
3. Coalesce edit storms, recover from overflow, and avoid mixed-revision or
   duplicate in-flight results.
4. Reuse existing bounded orchestration/caches when selective invalidation is
   proven safe, and fall back conservatively when it is not.
5. Share one source-index/model/quality read model with viewer, CLI/export, and
   MCP without duplicating analyzer or quality semantics.
6. Provide deterministic analyzer-neutral file/symbol/documentation,
   relationship, finding, and bounded exact-text search.
7. Provide callers/callees and other semantic queries only when advertised by
   the relevant analyzer/source-index capabilities.
8. Let an assistant request a temporary quality evaluation and inspect the
   complete available rule/profile catalog.
9. Support separately authorized profile saves and baseline creation through
   the deterministic-quality policy services.
10. Bound response bytes/items and omit full source content by default.
11. Enforce configured-root, read-only-by-default, no-shell, and no-target-
    execution permissions.
12. Make the external source-edit/fix loop explicit and observable through
   revision comparison.

## Non-goals

- Replacing analyzers, source-index extractors, quality rules, or canonical
  model normalization.
- Runtime tracing, execution of target applications, or arbitrary command
  execution from MCP.
- Fuzzy/embedding search, semantic ranking, or unbounded source retrieval.
- Automatic source edits, commits, refactors, or autonomous source fixes.
- Automatic baselining of all findings or silently changing quality policy.
- Treating one watcher event as proof of semantic freshness.
- Claiming current data while the source remains unstable during a scan.

## Functional requirements

| ID | Requirement |
|---|---|
| LAM-FR-001 | Validate a versioned live session with explicit roots, analyzer/source capability requests, quality profile reference, freshness policy, debounce/resource limits, budgets, and operation permissions. |
| LAM-FR-002 | Normalize backend events to root-safe paths and coalesce/deduplicate them within a bounded debounce policy. |
| LAM-FR-003 | Recover watcher overflow, backend errors, missed sequences, and ambiguous paths by rescanning rather than committing guessed state. |
| LAM-FR-004 | Reconcile every `require_current` query against an authoritative eligible-source input fingerprint (using a manifest/dirty-state fast path only when it proves the source unchanged) and join an existing in-flight reconciliation when possible. |
| LAM-FR-005 | Build source/model/quality candidates in isolation, verify the input fingerprint before and after analysis, and publish only coherent stable revisions. |
| LAM-FR-006 | Serve the last-ready revision during updates/failures with explicit freshness; return `input_unstable` or unavailable status rather than claiming current data. |
| LAM-FR-007 | Use content/source-scope/cache identity to select safe affected scopes; conservatively broaden the rescan when impact is uncertain. |
| LAM-FR-008 | Expose one versioned analyzer-neutral read model to viewer, CLI/export, and MCP without duplicating analysis semantics. |
| LAM-FR-009 | Provide deterministic structural queries and bounded literal/regex text search with opaque cursors tied to snapshot/query/order/budget. |
| LAM-FR-010 | Provide bounded quality findings/evidence, quality catalog/profile reads, temporary profile evaluation, and report comparison through the quality services. |
| LAM-FR-011 | Allow profile and baseline writes only through explicit operation permissions, exact-version validation, current-report checks, and auditable responses. |
| LAM-FR-012 | Include requested/returned consistency, revision, snapshot, scope, freshness, capabilities/coverage, omitted fields, and budget metadata in every query response. |
| LAM-FR-013 | Require explicit bounded source-context requests and prohibit arbitrary paths outside configured roots. |
| LAM-FR-014 | Keep shell, target execution, and source mutation outside the MCP surface. |
| LAM-FR-015 | Make watcher, store, query, quality delegation, and transport adapters replaceable strategies. |
| LAM-FR-016 | Provide bounded status/wait and quality-report comparison needed for an external `find → inspect → edit → reanalyze → compare` loop. |

## MCP operation groups

### Freshness and lifecycle

- `get_snapshot_status` with `latest_ready` or `require_current` consistency and
  bounded wait.
- `ensure_current_snapshot` for an explicit reconciliation request.

### Analyzer-neutral navigation

- `list_scopes`
- `find_files`
- `find_symbols`
- `find_text`
- `get_documentation`
- `get_module_facts`
- `get_callers_callees` when advertised
- `get_source_context` with explicit byte/line limits

### Quality operations

- `get_quality_profiles`
- `get_quality_rules`
- `get_quality_findings`
- `get_finding_evidence`
- `evaluate_quality` with a profile and optional non-persisted rule bindings
- `compare_quality_reports`

### Explicit quality-policy operations

- `validate_quality_profile`
- `save_quality_profile`
- `save_quality_profile_as`
- `preview_baseline`
- `create_baseline`

Profile and baseline commands delegate to the deterministic-quality services;
they do not create a second policy implementation in the live layer.

## Non-functional requirements

- **Freshness:** `require_current` is enforced by the server, not by client
  convention. A response is current only after authoritative input
  reconciliation and stable fingerprint verification; watcher state or file
  metadata alone is not accepted as proof when it can be ambiguous.
- **Consistency:** one response is tied to one requested/selected revision and
  one effective quality profile/options identity.
- **Availability:** transient updates do not remove the last ready view.
- **Determinism:** equal snapshot/query/filter/order/projection/budget inputs
  produce equal result ordering, counts, and cursor behavior.
- **Safety:** roots are allowlisted, symlink/junction escapes are rejected,
  read operations are default, policy writes are separately granted, and
  response/source budgets are enforced.
- **Token efficiency:** compact fields and counts come first; documentation,
  IDs/hashes, evidence detail, and source text are explicit projections.
- **Extensibility:** new analyzers, watcher/query/quality providers, and
  transports register capabilities without changing the coordinator or
  existing payload meanings.

## Success criteria

The scenarios in [acceptance-scenarios.md](acceptance-scenarios.md) pass for
multi-analyzer capability negotiation, event storms, request-time
reconciliation, stable-input verification, atomic revisions, partial failures,
structural/exact-text search, quality evaluation and policy permissions,
finding comparison, budgets, root safety, MCP parity, and no source mutation.
The application synthesis records this child as a shared consumer of the
source-index and deterministic-quality contracts.
