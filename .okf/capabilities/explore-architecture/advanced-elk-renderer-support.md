---
type: capability
title: Advanced ELK renderer support
description: Extend the viewer renderer to honor additional ELK presentation features beyond the implemented v1 route and option set.
tags: [viewer, elk, renderer, layout, future]
timestamp: 2026-08-27T00:00:00Z
state: foggy
state_changed: 2026-08-27T00:00:00Z
project: /project.md
parent: /capabilities/explore-architecture.md
artifact_root: docs/architecture/explore-architecture/advanced-elk-renderer-support
discovery_notes: docs/architecture/explore-architecture/advanced-elk-renderer-support/discovery-notes.md
orchestration_status: docs/architecture/explore-architecture/advanced-elk-renderer-support/orchestration-status.md
---

# Intent

Make additional ELK presentation capabilities visible, selectable, and
rendered correctly when the viewer has the scene and renderer support needed to
honor them.

# Scope

This child owns future renderer/layout work that extends the implemented
Explore capability: ELK target-specific options and geometry that currently
remain catalog-only, richer port and label presentation, junction handling,
compound-graph geometry, and any related route or hit-testing changes. It does
not change canonical model semantics, analyzer behavior, or project
configuration ownership.

# Relationships

- Parent: [Explore and inspect architecture](../explore-architecture.md)
- Future-work register: [Advanced ELK renderer future work](../../../docs/architecture/explore-architecture/advanced-elk-renderer-support/future-work.md)
- Current route foundation: [Renderer-neutral routing issue](../../../docs/agents/issues/done/20260826-010-renderer-neutral-routing-and-geometry.md)

# Planning state

This child is foggy. The candidate work is recorded for discovery, but the
priority, supported subset, scene-contract changes, renderer capability matrix,
and acceptance scenarios have not yet been agreed. No implementation issues
have been created intentionally; fog clearing and exact specification come
first.
