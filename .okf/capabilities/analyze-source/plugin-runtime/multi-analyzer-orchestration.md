---
type: capability
title: Multi-analyzer project orchestration
description: Detect, run, and combine multiple language analyzers across mixed and nested project roots.
tags: [plugins, orchestration, multi-language, concurrency]
timestamp: 2026-08-29T00:00:00Z
state: implemented
state_changed: 2026-08-29T00:00:00Z
project: /project.md
parent: /capabilities/analyze-source/plugin-runtime.md
artifact_root: docs/architecture/analyze-source/plugin-runtime/multi-analyzer-orchestration
discovery_notes: docs/architecture/analyze-source/plugin-runtime/multi-analyzer-orchestration/discovery-notes.md
gap_analysis: docs/architecture/analyze-source/plugin-runtime/multi-analyzer-orchestration/requirements-gap-analysis.md
orchestration_status: docs/architecture/analyze-source/plugin-runtime/multi-analyzer-orchestration/orchestration-status.md
prd: docs/architecture/analyze-source/plugin-runtime/multi-analyzer-orchestration/prd.md
glossary: docs/architecture/analyze-source/plugin-runtime/multi-analyzer-orchestration/domain-glossary.md
domain_model: docs/architecture/analyze-source/plugin-runtime/multi-analyzer-orchestration/canonical-domain-model.md
use_cases: docs/architecture/analyze-source/plugin-runtime/multi-analyzer-orchestration/canonical-use-cases.md
contract: docs/architecture/analyze-source/plugin-runtime/multi-analyzer-orchestration/canonical-api-cli-contract.md
scenarios: docs/architecture/analyze-source/plugin-runtime/multi-analyzer-orchestration/acceptance-scenarios.md
readiness_review: docs/architecture/analyze-source/plugin-runtime/multi-analyzer-orchestration/readiness-review.md
issues:
  - docs/agents/issues/done/20260828-039-multi-project-root-discovery-and-job-planning.md
  - docs/agents/issues/done/20260828-040-bounded-multi-analyzer-execution.md
  - docs/agents/issues/done/20260828-041-namespaced-aggregate-model-and-status.md
  - docs/agents/issues/done/20260828-042-combined-analysis-cli-and-http-exposure.md
  - docs/agents/issues/done/20260828-043-cached-scope-projections-and-viewer-selection.md
---

# Intent

Allow one repository view to include results from multiple analyzer processes
running at the same time, including mixed-language repositories and nested
projects.

# Scope

This capability owns marker-driven project-root discovery within the opened
repository, analyzer-job planning, bounded concurrency, result aggregation,
globally stable identities, invocation-root source-scope application, cache
identity, provenance, and partial-failure behavior. A strong
project manifest owns its subtree unless a nested manifest or explicit
assignment creates a nested project. It does not own language parsing,
analyzer-specific dependency semantics, or canonical model meaning.

# Relationships

- Parent: [Analyzer plugin runtime](/capabilities/analyze-source/plugin-runtime.md)
- Inputs: [Analyze source code](/capabilities/analyze-source.md)
- Produces: [Generate architecture models](/capabilities/generate-models.md)

# Planning state

This child is specified and readiness-reviewed. The current host executes one
selected analyzer per run; the target is a deterministic job set in which
applicable analyzers can run concurrently through a bounded worker pool
(default four, hard cap sixteen), return isolated diagnostics or partial
results, and contribute to one language-neutral aggregate model. Global
identities include relative project root, logical analyzer ID, and local
observation ID. The exact discovery, scheduling, aggregate, progress,
resource, cancellation, and acceptance contracts are linked above. The
combined model does not infer cross-language or cross-root relationships.
Source filters are resolved from the project-assignment boundary and applied
after discovery; fixed, nested-root, and configured exclusions win.

## Delivery progress

The approved implementation batch is issues 039–043: deterministic root
discovery and job planning, bounded execution and lifecycle control,
namespaced aggregation, CLI/HTTP exposure, and cached `All`/individual scope
selection in the local viewer. All five issues are implemented, verified, and
archived, including the approved mixed-language visual review. Persisted
`.archview.json` analyzer assignments remain owned by the separate
project-analyzer-assignments capability, which also owns the persisted
source-scope policy.

This node is `implemented`: its scoped multi-analyzer delivery and required
artifact synchronization are complete. The separate
project-analyzer-assignments capability is also implemented for persisted
assignment and source-scope configuration.
