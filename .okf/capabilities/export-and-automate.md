---
type: capability
title: Export and automate
description: Make architecture analysis usable from scripts, reports, and repeatable repository workflows.
tags: [cli, export, automation]
timestamp: 2026-08-25T14:56:24Z
state: specified
state_changed: 2026-08-25T17:10:00Z
project: /project.md
parent: /project.md
artifact_root: docs/architecture/export-and-automate
orchestration_status: docs/architecture/export-and-automate/orchestration-status.md
issues:
  - docs/agents/issues/pending/005-deterministic-json-html-svg-export.md
discovery_notes: docs/architecture/export-and-automate/discovery-notes.md
gap_analysis: docs/architecture/export-and-automate/requirements-gap-analysis.md
prd: docs/architecture/export-and-automate/prd.md
glossary: docs/architecture/export-and-automate/domain-glossary.md
domain_model: docs/architecture/export-and-automate/canonical-domain-model.md
use_cases: docs/architecture/export-and-automate/canonical-use-cases.md
contract: docs/architecture/export-and-automate/canonical-api-cli-contract.md
scenarios: docs/architecture/export-and-automate/acceptance-scenarios.md
readiness_review: docs/architecture/export-and-automate/readiness-review.md
---

# Intent

Support headless analysis and stable artifacts so architecture views can be regenerated locally or in CI.

# Scope

This capability includes CLI options, machine-readable architecture output, visual export, deterministic results, reanalysis, and future CI integration. It does not own the analyzer implementation or interactive scene behavior.

# Relationships

- Parent: [Arch View](../project.md)
- Input: [Generate architecture models](generate-models.md)
- Viewer relationship: [Explore and inspect architecture](explore-architecture.md)
- Artifact plan: [Export orchestration status](../../docs/architecture/export-and-automate/orchestration-status.md)

# Planning state

This capability is specified. Its versioned JSON, deterministic HTML/SVG artifacts, status/exit behavior, source-privacy defaults, shared reference-visibility policy, and CI contract are linked from the exact-spec artifacts.
