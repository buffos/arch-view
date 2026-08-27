---
type: capability
title: Explore and inspect architecture
description: Let users navigate generated architecture views and inspect the code and dependency evidence behind them.
tags: [viewer, navigation, evidence, layout]
timestamp: 2026-08-27T00:54:56Z
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
  - docs/agents/issues/done/20260826-008-expand-elk-parent-options.md
  - docs/agents/issues/done/20260827-009-elk-node-edge-option-targets.md
  - docs/agents/issues/done/20260827-016-spline-routing-and-cubic-rendering.md
  - docs/agents/issues/done/20260826-010-renderer-neutral-routing-and-geometry.md
  - docs/agents/issues/done/20260826-011-browser-composition-and-self-contained-bundling.md
  - docs/agents/issues/done/20260826-012-scene-projection-capability.md
  - docs/agents/issues/done/20260826-014-elk-option-handler-registry.md
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

Issues 003, 004, 005, 006, 007, and 008 are archived after explicit visual approval of the local-first/reference-boundary baseline, the evidence/inspection workflow, deterministic export parity, semantic/ELK refinement, the ELK settings/project-configuration slice, and the expanded parent-level ELK option tranche. Issue 009 implements the bounded target-aware node/edge priority tranche and is archived after automated verification and the user's visual approval. Refactor issues 010, 011, 012, and 014 are archived after automated implementation verification and the user's integrated visual review. Issue 016 is archived after automated verification and the user's visual approval of general ELK spline routing. The route model's reserved spline segments are now active for the supported layered path; self-contained HTML embeds the profile/catalog and pinned ELK runtime, and browser Download SVG captures the current canvas while Go static SVG remains deterministic orthogonal.
