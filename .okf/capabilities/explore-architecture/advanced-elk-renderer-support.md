---
type: capability
title: Advanced ELK renderer support
description: Extend the viewer renderer to honor additional ELK presentation features beyond the implemented v1 route and option set.
tags: [viewer, elk, renderer, layout, future]
timestamp: 2026-08-27T00:00:00Z
state: specified
state_changed: 2026-08-28T00:00:00Z
project: /project.md
parent: /capabilities/explore-architecture.md
artifact_root: docs/architecture/explore-architecture/advanced-elk-renderer-support
discovery_notes: docs/architecture/explore-architecture/advanced-elk-renderer-support/discovery-notes.md
gap_analysis: docs/architecture/explore-architecture/advanced-elk-renderer-support/requirements-gap-analysis.md
orchestration_status: docs/architecture/explore-architecture/advanced-elk-renderer-support/orchestration-status.md
prd: docs/architecture/explore-architecture/advanced-elk-renderer-support/prd.md
glossary: docs/architecture/explore-architecture/advanced-elk-renderer-support/domain-glossary.md
domain_model: docs/architecture/explore-architecture/advanced-elk-renderer-support/canonical-domain-model.md
use_cases: docs/architecture/explore-architecture/advanced-elk-renderer-support/canonical-use-cases.md
contract: docs/architecture/explore-architecture/advanced-elk-renderer-support/canonical-api-cli-contract.md
scenarios: docs/architecture/explore-architecture/advanced-elk-renderer-support/acceptance-scenarios.md
readiness_review: docs/architecture/explore-architecture/advanced-elk-renderer-support/readiness-review.md
issues:
  - docs/agents/issues/done/20260904-080-shared-elk-settings-and-feature-registry.md
  - docs/agents/issues/pending/081-shared-elk-advanced-edge-geometry.md
  - docs/agents/issues/pending/082-shared-elk-presentation-ports.md
  - docs/agents/issues/pending/083-shared-elk-nested-containers.md
---

# Intent

Make additional ELK presentation capabilities visible, selectable, and
rendered correctly when the viewer has the scene and renderer support needed to
honor them.

# Scope

This child owns staged renderer/layout work that extends the implemented Explore
capability: useful shared settings and feature registration first, then
edge labels, junctions and spline refinement, followed by presentation ports
and nested containers. It does not change canonical model semantics, analyzer
behavior, or project configuration ownership.

# Relationships

- Parent: [Explore and inspect architecture](../explore-architecture.md)
- Future-work register: [Advanced ELK renderer future work](../../../docs/architecture/explore-architecture/advanced-elk-renderer-support/future-work.md)
- Current route foundation: [Renderer-neutral routing issue](../../../docs/agents/issues/done/20260826-010-renderer-neutral-routing-and-geometry.md)

# Planning state

This child is specified and readiness-reviewed. Its exact renderer-only
extension program admits five opt-in features: edge labels, junctions,
presentation ports, compound geometry, and spline refinement. Browser,
self-contained HTML, and browser SVG share `arch-view.geometry/v1`; Go static
SVG remains deterministic orthogonal and reports when advanced geometry is not
applied. The linked contract defines geometry identity, option gating,
accessibility, validation, fallback, and visual scenarios. Approved stages are tracked by issues
080–083. Stage 1 is verified, visually approved, and archived. Stages 2–4
remain active and each requires automated verification and human visual approval.
