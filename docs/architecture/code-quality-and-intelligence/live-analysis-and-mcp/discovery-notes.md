# Live analysis and MCP discovery notes

## Purpose

Keep a configured Arch View source/model/quality view current across all
registered analyzers and expose the same compact, evidence-backed read model to
the viewer, CLI/export, and coding assistants. The live surface lowers
repository-reading cost by returning structured facts first and source text
only when requested. It must not become a second parser, analyzer, or quality
engine.

## Observed in code

- The local host already performs bounded multi-analyzer orchestration,
  per-scope caching, source-safe inspection, canonical model generation, and
  deterministic model/view output.
- The source-index child defines compact files, symbols, documentation, spans,
  relations, provenance, scope snapshots, coverage, and deterministic digests.
- The deterministic-quality child defines immutable reports, versioned rules
  and metrics, complete rule catalogs, profiles, temporary rule bindings,
  coverage, findings, comparisons, and baselines.
- No folder watcher, live revision coordinator, MCP server, MCP tool/resource
  contract, or snapshot publication protocol exists in the current product.

## User-confirmed target behavior

- Configured folders should be monitored for changes across all registered
  analyzers and their resolved source scopes.
- A coding assistant should request concise findings, files, symbols,
  documentation, exact text matches, callers/callees where available, and
  bounded source context.
- The assistant should follow a low-token path from finding to evidence to
  exact source location, then edit through its own coding tools and observe a
  new report.
- Watcher events may arrive many times while a file is being edited. The
  watcher should coalesce them, but a strict request must reconcile the actual
  source state before calling its result current.
- A model may list profiles/rules, run a one-off evaluation with explicit
  settings, and use profile/baseline writes only when the session grants those
  operations. It must not silently hide findings or change source files.

## Specified decisions

- A live session has an independent versioned configuration containing allowed
  roots, analyzer/source capability requests, quality profile reference,
  debounce/resource limits, freshness/reconciliation policy, projection
  budgets, and operation permissions. It is separate from layout, analyzer
  assignment semantics, quality rule semantics, and source-index facts.
- A watcher adapter normalizes create/modify/delete/rename/overflow events into
  root-safe paths. Events are deduplicated and coalesced during a bounded
  debounce window. Overflow, backend error, missed sequence, or uncertain path
  ownership triggers a conservative rescan.
- Watcher events are hints, not proof of freshness. A `require_current` query
  may start with a cheap manifest/dirty-state check, but it must use an
  authoritative content/input fingerprint whenever watcher delivery or file
  metadata cannot prove the eligible source unchanged. If the source is dirty,
  the coordinator starts or joins one bounded reanalysis. A `latest_ready`
  query may use the last ready revision, but must label it stale/updating when
  appropriate.
- The coordinator builds a candidate source/model/index/quality snapshot in
  isolation. It verifies the configured input manifest/content fingerprint
  before publication and checks that files did not change during the scan. An
  unstable input is retried within policy or returned as `input_unstable`; it
  is never published as current.
- Revisions are monotonic within a live session. A revision is internally
  consistent: source index, model, quality report, effective quality policy,
  and freshness metadata refer to the same verified input set. Failed
  reanalysis keeps the last good revision and publishes explicit degraded
  diagnostics.
- Selective reanalysis may invalidate affected analyzer scopes when existing
  cache/input identity proves it safe. Otherwise the coordinator rescans the
  affected scope or session. Language-specific invalidation and semantic
  inference remain analyzer/provider responsibilities; the live layer never
  assumes Go-specific facts.
- Structural queries use analyzer-neutral fields such as language, role,
  symbol category, path, documentation, relation, scope, and finding. A
  bounded literal/regex `search_text` operation is allowed for exact source
  navigation; fuzzy/embedding ranking remains outside v1.
- MCP is a transport/query adapter over the shared live and quality services.
  It exposes read operations by default and separately permissioned analysis
  and quality-policy operations. It does not own language semantics, quality
  rules, or baseline matching.
- A model can request a temporary quality evaluation without persisting it. It
  can save an existing or new profile, preview a baseline, create a standalone
  baseline, or append reviewed findings to the canonical profile baseline only
  through explicit allowlisted operations. Baseline changes are tied to an
  exact current report and selected finding keys; they are never an implicit
  “ignore all” operation.
- Every response carries session, snapshot, revision, requested/returned
  consistency, freshness, capabilities/coverage, omissions, budgets, and
  continuation information. Default projections are compact; IDs/hashes and
  full documentation/source are opt-in machine details.
- Local stdio is the default packaging. Network transports require explicit
  configuration, authentication/origin policy, and the same root and operation
  permissions. No shell, target execution, or source mutation is exposed.

## Boundary

This capability owns live configuration, watch lifecycle, event normalization,
debounce/coalescing, request-time reconciliation, invalidation, snapshot
consistency/revisions, analyzer-neutral bounded queries, MCP tools/resources,
budgets, and permissions. It delegates source facts to source-index providers,
quality evaluation/policy to deterministic-quality services, and source edits
to the coding agent or a separate authorized remediation workflow.

## Initial MCP surface

Resources are deliberately small: session status, scope/capability summary,
latest quality summary, and selected snapshot projections. Tools are grouped as
follows:

- freshness/lifecycle: `get_snapshot_status`, `ensure_current_snapshot`;
- navigation: `list_scopes`, `find_files`, `find_symbols`, `find_text`,
  `get_documentation`, `get_module_facts`, `get_callers_callees`,
  `get_source_context`;
- quality: `get_quality_profiles`, `get_quality_rules`,
  `get_quality_baselines`, `get_quality_findings`, `get_finding_evidence`,
  `evaluate_quality`, `compare_quality_reports`;
- explicitly permissioned policy: `validate_quality_profile`,
  `save_quality_profile`, `save_quality_profile_as`, `preview_baseline`,
  `create_baseline`, `append_baseline`.

All operations return explicit unsupported/unknown/partial coverage and never
turn missing analyzer capability into an empty successful result.

## Implementation and verification focus

Implement session/config validation, a polling/fake watcher contract,
event-coalescing and single-flight reconciliation, immutable snapshot storage,
stable-input verification, analyzer-neutral query projections, quality-service
delegation, revision-aware query envelopes, stdio MCP, root-safe source
context, operation permissions, and deterministic budgets. Verify event storms,
missed watcher events, request-time rescans, edits during scans, concurrent
strict requests, atomic visibility, overflow recovery, partial-failure
retention, stale/degraded/unstable status, profile override evaluation,
baseline authorization/currentness, scope/path containment, unsupported
capabilities, and no shell/target execution/source mutation.
