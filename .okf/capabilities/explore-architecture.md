---
type: capability
title: Explore and inspect architecture
description: Let users navigate generated architecture views and inspect the code and dependency evidence behind them.
tags: [viewer, navigation, evidence, layout]
timestamp: 2026-08-26T12:42:04Z
state: specified
state_changed: 2026-08-26T00:00:00Z
project: /project.md
parent: /project.md
artifact_root: docs/architecture/explore-architecture
orchestration_status: docs/architecture/explore-architecture/orchestration-status.md
issues:
  - docs/agents/issues/done/20260826-003-local-web-top-level-architecture-view.md
  - docs/agents/issues/done/20260826-004-evidence-drilldown-and-source-inspection.md
  - docs/agents/issues/done/20260826-005-deterministic-json-html-svg-export.md
  - docs/agents/issues/done/20260826-006-viewer-semantic-summary-and-elk-routing.md
  - docs/agents/issues/done/20260826-007-elk-layout-settings-and-project-config.md
  - docs/agents/issues/pending/008-expand-elk-parent-options.md
  - docs/agents/issues/pending/009-elk-node-edge-option-targets.md
discovery_notes: docs/architecture/explore-architecture/discovery-notes.md
gap_analysis: docs/architecture/explore-architecture/requirements-gap-analysis.md
prd: docs/architecture/explore-architecture/prd.md
glossary: docs/architecture/explore-architecture/domain-glossary.md
domain_model: docs/architecture/explore-architecture/canonical-domain-model.md
use_cases: docs/architecture/explore-architecture/canonical-use-cases.md
contract: docs/architecture/explore-architecture/canonical-api-cli-contract.md
scenarios: docs/architecture/explore-architecture/acceptance-scenarios.md
readiness_review: docs/architecture/explore-architecture/readiness-review.md
---

# Intent

Turn a generated architecture model into a useful investigation workflow rather than a static picture.

# Scope

This capability includes interactive diagram rendering, hierarchy drill-down, zoom and pan, dependency indicators, cycle explanations, source inspection, language-aware labels, configurable presentation layout, and project-scoped layout configuration discovery. It does not own dependency discovery or canonical graph algorithms.

# Relationships

- Parent: [Arch View](../project.md)
- Input: [Generate architecture models](generate-models.md)
- Related automation: [Export and automate](export-and-automate.md)
- Artifact plan: [Exploration orchestration status](../../docs/architecture/explore-architecture/orchestration-status.md)

# Planning state

This capability is specified. Its local web surface, renderer-neutral scene contract, local-first reference visibility, navigation/import evidence behavior, progressive disclosure, source safety, session layout, ELK layout settings, project configuration discovery, and accessibility requirements are linked from the exact-spec artifacts.

## Delivery progress

Issues 003, 004, 005, 006, and 007 are archived after explicit visual approval of the local-first/reference-boundary baseline, the evidence/inspection workflow, deterministic export parity, semantic/ELK refinement, and the ELK settings/project-configuration slice. Issue 004's implementation, compact full-canvas/viewport refinements, centered dense-scene fitting, stable drag rendering, deterministic manual edge routing, reset-layout behavior, and automated/browser verification are complete. Issue 008 is the next ready parent-level ELK option tranche, followed by issue 009's node/edge target mapping; advanced ports, labels, junctions, and compound-graph geometry remain a later specification frontier.
