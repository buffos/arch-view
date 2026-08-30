---
type: capability
title: Live analysis and MCP
description: Keep a configured source view current and expose compact structural search and quality reports to tools and LLMs.
tags: [mcp, watcher, incremental-analysis, code-search]
timestamp: 2026-08-29T00:00:00Z
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
---

# Intent

Give developers and coding assistants a low-token, revision-aware way to query current source facts and quality findings as repositories change.

# Scope

This capability covers configured-folder watching, debounced invalidation, analysis snapshots, selective or bounded reanalysis, compact structural search, quality-report retrieval, and MCP exposure. The watcher/index service owns freshness and snapshot consistency; MCP is a transport and query surface. Source edits and autonomous fixes are outside the initial scope.

# Relationships

- Parent: [Code quality and code intelligence](../code-quality-and-intelligence.md)
- Input: [Analyze source code](../analyze-source.md)
- Input: [Source facts and symbol index](source-facts-and-symbol-index.md)
- Input: [Deterministic quality checks](deterministic-quality-checks.md)
- Consumer: [Explore and inspect architecture](../explore-architecture.md)
- Consumer: [Export and automate](../export-and-automate.md)

# Current and target truth

Observed code facts:

- The local host already performs bounded analyzer orchestration, caching,
  source-safe inspection, and deterministic model/view generation; no MCP
  server or folder watcher is part of the current product contract.

User-confirmed target behavior:

- Configured folders should be monitored for selected violations, and an LLM
  should be able to request concise findings, symbol/file search,
  callers/callees where available, and bounded source context.

Specified boundary:

- The exact-spec set defines configured-root watching, event coalescing,
  conservative invalidation, immutable coherent revisions, last-ready failure
  behavior, deterministic structural search, MCP tools/resources, byte/item
  budgets, root-safe permissions, stdio-first packaging, and explicit
  remediation handoff.
- Watchers do not parse; MCP does not own language semantics, quality policy,
  or source edits. Implementation and packaging remain future delivery work.
