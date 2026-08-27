---
type: capability
title: Python analysis
description: Build architecture input from Python packages, modules, and import relationships.
tags: [python, analysis, roadmap]
timestamp: 2026-08-27T20:48:23Z
state: implemented
state_changed: 2026-08-27T12:09:55Z
project: /project.md
parent: /capabilities/analyze-source.md
discovery_notes: docs/architecture/analyze-source/python-analysis/discovery-notes.md
gap_analysis: docs/architecture/analyze-source/python-analysis/requirements-gap-analysis.md
orchestration_status: docs/architecture/analyze-source/python-analysis/orchestration-status.md
implementation_slice: docs/architecture/analyze-source/python-analysis/implementation-slice.md
issues:
  - docs/agents/issues/done/20260827-017-python-project-boundary-and-module-discovery.md
  - docs/agents/issues/done/20260827-018-python-static-import-resolution-and-uncertainty.md
  - docs/agents/issues/done/20260827-019-python-cli-and-visible-architecture-path.md
  - docs/agents/issues/done/20260827-032-external-python-analyzer-parity.md
  - docs/agents/issues/done/20260827-033-external-plugin-cli-and-visible-journey.md
prd: docs/architecture/analyze-source/python-analysis/prd.md
glossary: docs/architecture/analyze-source/python-analysis/domain-glossary.md
domain_model: docs/architecture/analyze-source/python-analysis/canonical-domain-model.md
use_cases: docs/architecture/analyze-source/python-analysis/canonical-use-cases.md
contract: docs/architecture/analyze-source/python-analysis/canonical-api-cli-contract.md
scenarios: docs/architecture/analyze-source/python-analysis/acceptance-scenarios.md
readiness_review: docs/architecture/analyze-source/python-analysis/readiness-review.md
---

# Intent

Add useful static architecture discovery for Python repositories while making uncertainty from dynamic imports visible.

# Scope

The analyzer will detect common project layouts, resolve absolute and relative imports where possible, identify package and module hierarchy, retain evidence, and report unresolved or dynamic dependencies.

# Relationships

- Parent: [Analyze source code](../analyze-source.md)
- Produces input for: [Generate architecture models](../generate-models.md)

# Planning state

This child capability is implemented. Its project-marker precedence, source-root rules, package/module semantics, static resolution, uncertainty behavior, options, and safety policy remain defined by the linked exact-spec artifacts.

## Delivery progress

The [Python repository to visible architecture view](../../../docs/architecture/analyze-source/python-analysis/implementation-slice.md) slice is complete. Issue 017 implements the registered project-boundary and module-discovery foundation, issue 018 implements static imports, local resolution, references, evidence, and uncertainty diagnostics, and issue 019 connects the public CLI to the shared model/viewer/export path. Issues 032–033 add and verify the opt-in external stdlib-only process deployment and shared path without changing the in-process baseline. Automated verification and the declared visual review are complete, so the node is `implemented`.
