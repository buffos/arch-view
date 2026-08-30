---
type: capability
title: Deterministic quality checks
description: Evaluate reproducible source and architecture metrics against configurable rules and report evidence-backed findings.
tags: [code-quality, metrics, rules, architecture]
timestamp: 2026-08-30T08:33:39Z
state: implemented
state_changed: 2026-08-30T19:45:00Z
project: /project.md
parent: /capabilities/code-quality-and-intelligence.md
artifact_root: docs/architecture/code-quality-and-intelligence/deterministic-quality-checks
discovery_notes: docs/architecture/code-quality-and-intelligence/deterministic-quality-checks/discovery-notes.md
gap_analysis: docs/architecture/code-quality-and-intelligence/deterministic-quality-checks/requirements-gap-analysis.md
orchestration_status: docs/architecture/code-quality-and-intelligence/deterministic-quality-checks/orchestration-status.md
prd: docs/architecture/code-quality-and-intelligence/deterministic-quality-checks/prd.md
glossary: docs/architecture/code-quality-and-intelligence/deterministic-quality-checks/domain-glossary.md
domain_model: docs/architecture/code-quality-and-intelligence/deterministic-quality-checks/canonical-domain-model.md
use_cases: docs/architecture/code-quality-and-intelligence/deterministic-quality-checks/canonical-use-cases.md
contract: docs/architecture/code-quality-and-intelligence/deterministic-quality-checks/canonical-api-cli-contract.md
scenarios: docs/architecture/code-quality-and-intelligence/deterministic-quality-checks/acceptance-scenarios.md
readiness_review: docs/architecture/code-quality-and-intelligence/deterministic-quality-checks/readiness-review.md
issues:
  - docs/agents/issues/done/20260830-053-quality-profile-and-file-size-report.md
  - docs/agents/issues/done/20260830-054-callable-size-findings.md
  - docs/agents/issues/done/20260830-055-callable-complexity-and-nesting-findings.md
  - docs/agents/issues/done/20260830-056-documentation-coverage-findings.md
  - docs/agents/issues/done/20260830-057-graph-coupling-and-cycle-findings.md
  - docs/agents/issues/done/20260830-058-explicit-architecture-constraint-findings.md
  - docs/agents/issues/done/20260830-059-deterministic-report-lifecycle-and-baselines.md
  - docs/agents/issues/done/20260830-060-solid-structural-signals.md
  - docs/agents/issues/done/20260830-061-quality-report-query-and-evidence.md
  - docs/agents/issues/done/20260830-062-quality-cli-and-export-projections.md
  - docs/agents/issues/done/20260830-063-quality-viewer-and-affected-file-filter.md
---

# Intent

Turn measurable source and architecture facts into repeatable quality feedback that can be shown, exported, and checked in automation.

# Scope

This capability covers configurable thresholds, metric calculation, rule evaluation, severity, evidence, baselines, and quality findings. Initial rule territory includes file/function size, cyclomatic complexity, nesting, documentation coverage, coupling, cycles, forbidden dependencies, and layer violations. SOLID-related output is limited to explicitly labeled deterministic signals or heuristics; it is not an automatic architectural verdict.

# Relationships

- Parent: [Code quality and code intelligence](../code-quality-and-intelligence.md)
- Input: [Source facts and symbol index](source-facts-and-symbol-index.md)
- Input: [Generate architecture models](../generate-models.md)
- Consumer: [Explore and inspect architecture](../explore-architecture.md)
- Consumer: [Export and automate](../export-and-automate.md)
- Consumer: [Live analysis and MCP](live-analysis-and-mcp.md)

# Current and target truth

Observed code facts:

- The canonical model already derives cycles, layers, relationships,
  diagnostics, and stable source evidence, but it has no separate quality-
  finding contract.

User-confirmed target behavior:

- Files and functions over configurable limits, excessive complexity, missing
  public documentation, and actual graph-structural problems should be
  highlighted deterministically.
- A configured `source:file.max-lines` rule should produce a scope-safe,
  evidence-backed finding for every file over its explicit line limit. Human
  facing consumers should summarize those findings as a count of affected files
  and offer an explicit filter to show only the affected files; they must use
  the quality report and preserve partial/unknown coverage rather than
  re-evaluating raw source facts.
- SOLID feedback is useful only when clearly labeled as evidence-backed signal
  or heuristic; it must not be presented as a provable violation.

Specified boundary:

- Quality profiles, versioned metric/rule strategies, threshold semantics,
  exact findings, advisory signals, coverage, evidence, baselines, and
  deterministic report identity are defined in the linked exact-spec set.
- Project-backed viewers may explicitly persist a validated complete rule
  selection to an existing quality profile or save it as a new direct JSON
  profile; temporary viewer selections remain session-owned.
- File line-threshold semantics belong to this capability; viewer, CLI, export,
  and MCP surfaces consume the resulting findings and do not own threshold
  evaluation.
- The source-index child owns source facts; this child consumes those facts and
  canonical graph facts. Live freshness, MCP transport, and source mutation
  remain outside this child.

# Delivery

The readiness-reviewed implementation frontier is sliced into issues 053–063.
Issues 053–063 are implemented, verified, and archived: they establish the
versioned profile/catalog, source and graph providers, exact size/complexity/
documentation/coupling/cycle/constraint rules, report lifecycle and baselines,
conservative SOLID signals, bounded queries/evidence, and headless/export
projections. The Go source-index path publishes the structural facts consumed
by the SOLID signals, and the local CLI exposes validated baseline creation.
Issue 063 is verified and archived after desktop and responsive visual review
of the viewer projection. All scoped issues and acceptance criteria are now
implemented, so this capability is `implemented`.
