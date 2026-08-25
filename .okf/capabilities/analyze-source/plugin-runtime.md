---
type: capability
title: Analyzer plugin runtime
description: Discover, configure, execute, and version language analyzers behind a stable contract.
tags: [plugins, extensibility, analysis]
timestamp: 2026-08-25T14:56:24Z
state: specified
state_changed: 2026-08-25T17:10:00Z
project: /project.md
parent: /capabilities/analyze-source.md
discovery_notes: docs/architecture/analyze-source/plugin-runtime/discovery-notes.md
gap_analysis: docs/architecture/analyze-source/plugin-runtime/requirements-gap-analysis.md
orchestration_status: docs/architecture/analyze-source/plugin-runtime/orchestration-status.md
issues:
  - docs/agents/issues/done/20260825-001-analyzer-host-and-go-project-selection.md
prd: docs/architecture/analyze-source/plugin-runtime/prd.md
glossary: docs/architecture/analyze-source/plugin-runtime/domain-glossary.md
domain_model: docs/architecture/analyze-source/plugin-runtime/canonical-domain-model.md
use_cases: docs/architecture/analyze-source/plugin-runtime/canonical-use-cases.md
contract: docs/architecture/analyze-source/plugin-runtime/canonical-api-cli-contract.md
scenarios: docs/architecture/analyze-source/plugin-runtime/acceptance-scenarios.md
readiness_review: docs/architecture/analyze-source/plugin-runtime/readiness-review.md
---

# Intent

Allow new language analyzers to be added without changing the architecture model, layout engine, or viewer.

# Scope

The capability includes analyzer registration, project detection, explicit language selection, configuration, result validation, diagnostics, and a future process boundary for analyzers written outside Go.

# Relationships

- Parent: [Analyze source code](../analyze-source.md)
- Consumes: [Generate architecture models](../generate-models.md)

# Notes

The first implementation may use built-in Go interfaces. A versioned JSON process protocol can follow when external Python or TypeScript plugins are needed.

# Planning state

This child capability is specified. Its host/plugin API, manifest fields, selection rules, lifecycle, safety policy, and staged external protocol are linked from the exact-spec artifacts.
