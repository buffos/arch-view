---
type: capability
title: Export and automate
description: Make architecture analysis usable from scripts, reports, and repeatable repository workflows.
tags: [cli, export, automation]
timestamp: 2026-08-28T00:43:49Z
state: implemented
state_changed: 2026-08-28T00:43:49Z
project: /project.md
parent: /project.md
artifact_root: docs/architecture/export-and-automate
orchestration_status: docs/architecture/export-and-automate/orchestration-status.md
issues:
  - docs/agents/issues/done/20260826-005-deterministic-json-html-svg-export.md
  - docs/agents/issues/done/20260826-011-browser-composition-and-self-contained-bundling.md
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

This capability is implemented. Issues 005, 011, and 016 deliver versioned
JSON, self-contained HTML, deterministic static SVG, browser canvas SVG,
status and exit behavior, overwrite safety, source-privacy defaults, and the
shared reference-visibility policy. All nine linked acceptance scenarios have
automated coverage, and the visual export paths have explicit user approval.

## Delivery progress

Issue 005 implemented the shared JSON, self-contained HTML, and static SVG
export path, including deterministic layouts, reference visibility, evidence
metadata, atomic writes, and CLI analyze-to-export wiring. Issue 011 completed
the browser module and bundling boundary. Issue 016 extends HTML with an
embedded profile and catalog plus a pinned ELK runtime, and adds current-canvas
SVG download in the browser. The Go static SVG contract remains deterministic
and orthogonal. Broader reanalysis and CI policy work are outside this node's
current promised scope and need a future feature addition before delivery.
