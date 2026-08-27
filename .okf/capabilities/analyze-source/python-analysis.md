---
type: capability
title: Python analysis
description: Build architecture input from Python packages, modules, and import relationships.
tags: [python, analysis, roadmap]
timestamp: 2026-08-25T14:56:24Z
state: specified
state_changed: 2026-08-25T17:10:00Z
project: /project.md
parent: /capabilities/analyze-source.md
discovery_notes: docs/architecture/analyze-source/python-analysis/discovery-notes.md
gap_analysis: docs/architecture/analyze-source/python-analysis/requirements-gap-analysis.md
orchestration_status: docs/architecture/analyze-source/python-analysis/orchestration-status.md
implementation_slice: docs/architecture/analyze-source/python-analysis/implementation-slice.md
issues:
  - docs/agents/issues/pending/20260827-017-python-project-boundary-and-module-discovery.md
  - docs/agents/issues/pending/20260827-018-python-static-import-resolution-and-uncertainty.md
  - docs/agents/issues/pending/20260827-019-python-cli-and-visible-architecture-path.md
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

This child capability is specified. Its project-marker precedence, source-root rules, package/module semantics, static resolution, uncertainty behavior, options, and safety policy are linked from the exact-spec artifacts.

## Delivery progress

The next implementation slice is [Python repository to visible architecture view](../../../docs/architecture/analyze-source/python-analysis/implementation-slice.md), with issues 017–019 covering project/module discovery, import uncertainty, and public model/viewer/export integration. The node remains `specified` until the complete slice is implemented and reviewed.
