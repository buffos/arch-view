---
type: capability
title: Live analysis and MCP
description: Keep a configured multi-analyzer source view current and expose compact, analyzer-neutral search, quality evaluation, and controlled quality-policy operations to tools and LLMs.
tags: [mcp, watcher, live-analysis, multi-analyzer, code-search, quality-policy]
timestamp: 2026-08-31T00:00:00Z
state: specified
state_changed: 2026-08-29T00:00:00Z
project: /project.md
parent: /capabilities/code-quality-and-intelligence.md
artifact_root: docs/architecture/code-quality-and-intelligence/live-analysis-and-mcp
discovery_notes: docs/architecture/code-quality-and-intelligence/live-analysis-and-mcp/discovery-notes.md
gap_analysis: docs/architecture/code-quality-and-intelligence/live-analysis-and-mcp/requirements-gap-analysis.md
orchestration_status: docs/architecture/code-quality-and-intelligence/live-analysis-and-mcp/orchestration-status.md
prd: docs/architecture/code-quality-and-intelligence/live-analysis-and-mcp/prd.md
glossary: docs/architecture/code-quality-and-intelligence/live-analysis-and-mcp/domain-glossary.md
domain_model: docs/architecture/code-quality-and-intelligence/live-analysis-and-mcp/canonical-domain-model.md
use_cases: docs/architecture/code-quality-and-intelligence/live-analysis-and-mcp/canonical-use-cases.md
contract: docs/architecture/code-quality-and-intelligence/live-analysis-and-mcp/canonical-api-cli-contract.md
scenarios: docs/architecture/code-quality-and-intelligence/live-analysis-and-mcp/acceptance-scenarios.md
readiness_review: docs/architecture/code-quality-and-intelligence/live-analysis-and-mcp/readiness-review.md
issues:
  - docs/agents/issues/done/20260831-064-live-session-config-and-initial-snapshot.md
  - docs/agents/issues/done/20260831-065-watcher-events-and-coalescing.md
  - docs/agents/issues/done/20260831-066-coherent-revision-store-and-publication.md
  - docs/agents/issues/done/20260831-067-freshness-reconciliation-and-single-flight.md
  - docs/agents/issues/done/20260831-068-analyzer-neutral-query-surface.md
  - docs/agents/issues/done/20260831-069-exact-text-and-source-context.md
  - docs/agents/issues/done/20260831-070-quality-gateway-and-temporary-evaluation.md
  - docs/agents/issues/done/20260831-071-permissioned-quality-policy-operations.md
  - docs/agents/issues/done/20260831-072-local-live-session-cli-bridge.md
  - docs/agents/issues/done/20260831-073-live-viewer-integration.md
  - docs/agents/issues/done/20260831-074-mcp-stdio-server-and-documentation.md
  - docs/agents/issues/done/20260831-075-authenticated-http-mcp-transport.md
  - docs/agents/issues/pending/20260831-076-cross-analyzer-agent-workflow-and-final-review.md
---

# Intent

Give developers and coding assistants a low-token, revision-aware way to query
verified source facts and quality findings across all registered analyzers as a
repository changes.

# Scope

This capability covers configured-folder watching, event coalescing, request-
time freshness reconciliation, coherent analysis snapshots, selective or
bounded reanalysis, analyzer-neutral structural and bounded text search,
quality-report retrieval/evaluation, explicitly authorized quality-profile and
baseline operations, and MCP exposure. The live coordinator owns freshness and
snapshot consistency; analyzers own language semantics; the quality capability
owns rule/profile/baseline semantics; MCP is a bounded transport and query/
command surface. Source edits, autonomous fixes, shell execution, and target
application execution remain outside this capability.

# Relationships

- Parent: [Code quality and code intelligence](../code-quality-and-intelligence.md)
- Input: [Analyze source code](../analyze-source.md)
- Input: [Source facts and symbol index](source-facts-and-symbol-index.md)
- Input: [Deterministic quality checks](deterministic-quality-checks.md)
- Consumer: [Explore and inspect architecture](../explore-architecture.md)
- Consumer: [Export and automate](../export-and-automate.md)

# Current and target truth

Observed code facts:

- `internal/live` implements the analyzer-neutral live session: configured
  roots, polling watcher hints, event coalescing, request-time reconciliation,
  stable-input verification, single-flight rebuilds, immutable revisions, and
  last-ready failure behavior.
- The shared live query and quality gateways expose bounded scopes, files,
  symbols, documentation, text, source context, module relations, quality
  reports, evidence, comparisons, and temporary evaluations. They preserve
  revision, freshness, coverage, cursor, and budget metadata.
- `arch-view live` and `arch-view open --live` provide the local CLI/viewer
  bridge. The viewer pins follow-up model, source, and quality reads to the
  revision advertised by live status, so a refresh cannot mix revisions.
- `arch-view mcp` provides newline-delimited JSON-RPC stdio by default. The
  optional local and authenticated loopback HTTP adapters map to the same
  services and fail closed on unsafe requests.
- Quality profile and baseline writes are separate allowlisted operations.
  They are denied by default, require explicit authorization, validate exact
  report/profile identity, use safe project-relative destinations, and return
  audit data. No live/MCP operation edits source code or runs the target.
- The source-index capability provides analyzer-reported files, declarations,
  documentation, spans, relations, provenance, coverage, and deterministic
  snapshot digests. The quality capability remains the owner of versioned
  rules, profiles, evaluations, findings, coverage, comparisons, and baselines.

User-confirmed target behavior:

- Configured folders should be monitored for changes across all registered
  analyzers, not through a Go-specific MCP path.
- A coding assistant should be able to search structure or exact text, locate
  a symbol or finding, follow its evidence and callers/callees when available,
  and request only the bounded source context needed to act.
- A request that requires current information should verify the source state
  and wait for a coherent revision instead of trusting watcher events alone.
- A model may request a temporary quality configuration, inspect available
  rules/profiles, and—only when explicitly authorized—save a profile or create
  a baseline through the deterministic-quality services.
- Findings are evidence for an explicit downstream source edit. The live
  watcher and MCP surface do not silently edit code or baseline findings.

Implemented boundary:

- The exact-spec set defines configured-root watching, analyzer capability
  negotiation, event coalescing, request-time reconciliation, stable-input
  verification, conservative invalidation, immutable coherent revisions,
  last-ready failure behavior, deterministic structural/exact-text search,
  quality catalog/evaluation/policy delegation, MCP tools/resources, budgets,
  root-safe permissions, stdio-first packaging, and explicit remediation
  handoff.
- Watchers produce change hints. A strict `require_current` query may use a
  cheap dirty/manifest check as a fast path, but must use an authoritative
  content/input fingerprint whenever the watcher or manifest cannot prove the
  eligible source unchanged. It then joins one bounded reanalysis when needed;
  no response is labeled current until the candidate input is verified stable.
- Watchers do not parse; MCP does not own analyzer semantics, source facts,
  graph semantics, quality rules, profile/baseline meaning, or source edits.
  Issues 064–075 implement this boundary. Issue 076 remains the final
  cross-analyzer and product-approval gate.

# Delivery

The approved dependency-ordered implementation frontier is issues 064–076.
Issues 064–071 establish the live session, watcher/reconciliation, immutable
revisions, analyzer-neutral queries, bounded source context, delegated quality
operations, and explicit policy writes. Issues 072–075 implement and verify
the local CLI/viewer bridge, MCP stdio and documentation, and optional
authenticated HTTP transport. Automated conformance evidence for issue 076 is
present; its final human product-approval gate remains open. The capability
remains `specified` until that final gate and the scoped delivery are complete.
