---
type: capability
title: TypeScript analysis
description: Build architecture input from TypeScript projects, module imports, aliases, and project boundaries.
tags: [typescript, javascript, analysis, roadmap]
timestamp: 2026-08-25T14:56:24Z
state: specified
state_changed: 2026-08-25T17:10:00Z
project: /project.md
parent: /capabilities/analyze-source.md
discovery_notes: docs/architecture/analyze-source/typescript-analysis/discovery-notes.md
gap_analysis: docs/architecture/analyze-source/typescript-analysis/requirements-gap-analysis.md
orchestration_status: docs/architecture/analyze-source/typescript-analysis/orchestration-status.md
prd: docs/architecture/analyze-source/typescript-analysis/prd.md
glossary: docs/architecture/analyze-source/typescript-analysis/domain-glossary.md
domain_model: docs/architecture/analyze-source/typescript-analysis/canonical-domain-model.md
use_cases: docs/architecture/analyze-source/typescript-analysis/canonical-use-cases.md
contract: docs/architecture/analyze-source/typescript-analysis/canonical-api-cli-contract.md
scenarios: docs/architecture/analyze-source/typescript-analysis/acceptance-scenarios.md
readiness_review: docs/architecture/analyze-source/typescript-analysis/readiness-review.md
---

# Intent

Add static architecture discovery for TypeScript projects without hiding configuration-driven module resolution.

# Scope

The analyzer will account for project configuration, package and module boundaries, import and export relationships, path aliases, generated or excluded files, and unresolved dynamic loading.

# Relationships

- Parent: [Analyze source code](../analyze-source.md)
- Produces input for: [Generate architecture models](../generate-models.md)

# Planning state

This child capability is specified. Its config selection/inheritance, module resolution, import/export kinds, alias policy, dynamic uncertainty, options, and safety behavior are linked from the exact-spec artifacts.
