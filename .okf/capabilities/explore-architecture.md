---
type: capability
title: Explore and inspect architecture
description: Let users navigate generated architecture views and inspect the code and dependency evidence behind them.
tags: [viewer, navigation, evidence]
timestamp: 2026-08-25T14:56:24Z
state: specified
state_changed: 2026-08-26T00:00:00Z
project: /project.md
parent: /project.md
artifact_root: docs/architecture/explore-architecture
orchestration_status: docs/architecture/explore-architecture/orchestration-status.md
issues:
  - docs/agents/issues/done/20260826-003-local-web-top-level-architecture-view.md
  - docs/agents/issues/pending/004-evidence-drilldown-and-source-inspection.md
  - docs/agents/issues/pending/005-deterministic-json-html-svg-export.md
  - docs/agents/issues/done/20260826-006-viewer-semantic-summary-and-elk-routing.md
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

This capability includes interactive diagram rendering, hierarchy drill-down, zoom and pan, dependency indicators, cycle explanations, source inspection, and language-aware labels. It does not own dependency discovery or graph algorithms.

# Relationships

- Parent: [Arch View](../project.md)
- Input: [Generate architecture models](generate-models.md)
- Related automation: [Export and automate](export-and-automate.md)
- Artifact plan: [Exploration orchestration status](../../docs/architecture/explore-architecture/orchestration-status.md)

# Planning state

This capability is specified. Its local web surface, renderer-neutral scene contract, local-first reference visibility, navigation/import evidence behavior, progressive disclosure, source safety, session layout, and accessibility requirements are linked from the exact-spec artifacts.

## Delivery progress

Issues 003 and 006 are archived after explicit visual approval of the local-first/reference-boundary baseline and its semantic/ELK refinement. Issue 004 owns deeper imports/evidence interaction and session layout controls; issue 005 owns export parity. Issues 004 and 005 remain the active unblocked delivery frontiers.
