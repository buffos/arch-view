---
type: capability
title: Code quality and code intelligence
description: Build deterministic quality findings and compact, searchable source intelligence on top of architecture analysis.
tags: [code-quality, code-intelligence, metrics, mcp]
timestamp: 2026-08-30T08:15:05Z
state: specified
state_changed: 2026-08-29T00:00:00Z
state_policy:
  mode: rollup
  source: structural_children
  reducer: min
project: /project.md
parent: /project.md
children:
  - /capabilities/code-quality-and-intelligence/source-facts-and-symbol-index.md
  - /capabilities/code-quality-and-intelligence/deterministic-quality-checks.md
  - /capabilities/code-quality-and-intelligence/live-analysis-and-mcp.md
---

# Intent

Extend Arch View from architecture presentation into a deterministic, searchable understanding of source structure and code-quality risk.

# Scope

This capability is the aggregate/navigation boundary for source intelligence and
quality. Its structural children own the source facts, reproducible metrics and
rule findings, and live query/reporting behavior; the parent has no standalone
implementation scope. It consumes analyzer and canonical-model output; it does
not replace language analysis, architecture rendering, or source editing.

# Relationships

- Parent: [Arch View](../project.md)
- Input: [Analyze source code](analyze-source.md)
- Input: [Generate architecture models](generate-models.md)
- Viewer consumer: [Explore and inspect architecture](explore-architecture.md)
- Automation consumer: [Export and automate](export-and-automate.md)
- Child: [Source facts and symbol index](code-quality-and-intelligence/source-facts-and-symbol-index.md)
- Child: [Deterministic quality checks](code-quality-and-intelligence/deterministic-quality-checks.md)
- Child: [Live analysis and MCP](code-quality-and-intelligence/live-analysis-and-mcp.md)

# Notes

The parent is a pure structural-child roll-up. Its materialized `state` is the
minimum of its three structural children (`specified` at present). All three
children are readiness-reviewed. The source-facts and deterministic-quality
children are implemented through their approved batches; live-analysis/MCP
remains specified. The parent has no standalone PRD or issue batch, and
remaining implementation work is routed to the other children.
If parent-only behavior is later introduced, the policy
must explicitly switch to `mode: own` before adding parent-owned artifacts or
advancing its own state.

# Current and target truth

- **Observed in code:** analyzers return modules, relationships, source references, diagnostics, and file-linked evidence through a common contract; Tree-sitter-backed syntax providers are available for the supported languages.
- **User-confirmed target behavior:** Arch View should expose files, documentation, declarations, deterministic metrics, quality findings, and compact code search/reporting without executing target applications.
- **Required follow-up:** Implement the remaining readiness-reviewed child
  contracts in dependency order. The
  source-facts child owns the shared file/symbol/documentation contract
  consumed by quality and live/MCP.

# Boundary

The deterministic product must not claim subjective architectural approval, semantic comment quality, or provable SOLID violations. Such concerns may be represented as explicitly labeled, evidence-backed signals. Source mutation remains an explicit downstream action rather than an implicit analysis side effect.
