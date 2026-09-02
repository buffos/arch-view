---
type: capability
title: Configurable OKF knowledge views
description: Visualize one or more OKF knowledge bundles through configurable structural projections, metadata mappings, and ELK-backed interactive views.
tags: [okf, viewer, visualization, configuration]
timestamp: 2026-09-02T18:06:48Z
state: bounded
project: /project.md
parent: /project.md
artifact_root: docs/architecture/okf-knowledge-views
discovery_notes: docs/architecture/okf-knowledge-views/discovery-notes.md
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

This capability has a bounded discovery baseline. Its exact contracts and
implementation issues are still future work, but the product boundary,
configuration direction, profile composition, rule extension seam, navigation,
inspection, and safety behavior are defined well enough for structured design.
