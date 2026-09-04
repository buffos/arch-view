---
type: capability
title: Configurable OKF knowledge views
description: Visualize one or more OKF knowledge bundles through configurable structural projections, metadata mappings, and ELK-backed interactive views.
tags: [okf, viewer, visualization, configuration]
timestamp: 2026-09-04T00:00:00Z
state: implemented
state_changed: 2026-09-04T00:00:00Z
project: /project.md
parent: /project.md
artifact_root: docs/architecture/okf-knowledge-views
discovery_notes: docs/architecture/okf-knowledge-views/discovery-notes.md
gap_analysis: docs/architecture/okf-knowledge-views/requirements-gap-analysis.md
glossary: docs/architecture/okf-knowledge-views/domain-glossary.md
prd: docs/architecture/okf-knowledge-views/prd.md
domain_model: docs/architecture/okf-knowledge-views/canonical-domain-model.md
use_cases: docs/architecture/okf-knowledge-views/canonical-use-cases.md
contract: docs/architecture/okf-knowledge-views/canonical-api-cli-contract.md
scenarios: docs/architecture/okf-knowledge-views/acceptance-scenarios.md
orchestration_status: docs/architecture/okf-knowledge-views/orchestration-status.md
readiness_review: docs/architecture/okf-knowledge-views/readiness-review.md
issues:
  - docs/agents/issues/done/20260903-064-okf-bundle-discovery-and-index.md
  - docs/agents/issues/done/20260903-065-okf-neutral-projection-viewer.md
  - docs/agents/issues/done/20260903-066-okf-hierarchy-and-semantic-links.md
  - docs/agents/issues/done/20260903-067-okf-profile-composition-and-fog-of-war.md
  - docs/agents/issues/done/20260903-068-okf-navigation-and-scale-safety.md
  - docs/agents/issues/done/20260903-069-okf-concept-inspection-and-markdown-safety.md
  - docs/agents/issues/done/20260903-070-okf-profile-persistence-and-bindings.md
  - docs/agents/issues/done/20260903-071-okf-failure-isolation-and-acceptance.md
---

# Intent

Let developers use Arch View to explore arbitrary OKF knowledge bundles with
their own graph projections, visual conventions, and inspection preferences.

# Scope

This capability covers discovery and consumption of one or more OKF bundles,
configurable structural and relationship projections, user-defined metadata
facets and visual rules, depth-bounded subtree navigation, and read-only
inspection of concept metadata and Markdown content. It reuses the existing
ELK-backed viewer infrastructure and does not change the architecture model or
the OKF source documents.

# Relationships

- Parent: [Arch View](../project.md)
- Related viewer capability: [Explore and inspect architecture](explore-architecture.md)
- Related renderer capability: [Advanced ELK renderer support](explore-architecture/advanced-elk-renderer-support.md)
- Discovery notes: [OKF knowledge-view pre-PRD](../../docs/architecture/okf-knowledge-views/discovery-notes.md)

# Notes

This capability has a readiness-reviewed exact specification and an approved
delivery slice. Issues 064–071 implement the source boundary, neutral and
profiled projections, relationship semantics, navigation, inspection,
configuration lifecycle, and failure isolation in dependency order. The
capability is `implemented`: all eight issues are verified and the user approved
the completed viewer on 2026-09-04. Shared CSS and current-canvas SVG download
were included in that approval. Public OKF CLI/headless export remains outside
the scoped delivery. The project parent has no roll-up state policy, so no
ancestor state transition is required.
