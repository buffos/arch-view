---
type: capability
title: Explore and inspect architecture
description: Let users navigate generated architecture views and inspect the code and dependency evidence behind them.
tags: [viewer, navigation, evidence]
timestamp: 2026-08-25T14:56:24Z
state: specified
state_changed: 2026-08-25T17:10:00Z
project: /project.md
parent: /project.md
artifact_root: docs/architecture/explore-architecture
orchestration_status: docs/architecture/explore-architecture/orchestration-status.md
issues:
  - docs/agents/issues/pending/003-local-web-top-level-architecture-view.md
  - docs/agents/issues/pending/004-evidence-drilldown-and-source-inspection.md
  - docs/agents/issues/pending/005-deterministic-json-html-svg-export.md
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

This capability is specified. Its local web surface, renderer-neutral scene contract, navigation/evidence behavior, progressive disclosure, source safety, and accessibility requirements are linked from the exact-spec artifacts.
