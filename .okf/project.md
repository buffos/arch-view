---
type: project
title: Arch View
description: Analyze supported codebases and generate navigable architecture views.
tags: [architecture, code-analysis, multi-language, golang]
timestamp: 2026-08-25T15:15:46Z
prd: docs/prd.md
architecture_summary: docs/architecture/application-architecture-summary.md
verification:
  mode: when-supported
  deferred_requires_reason: true
  surfaces:
    backend-boundary: when-supported
    frontend-integration: when-supported
    end-to-end: when-supported
---

# Intent

Arch View helps developers understand a codebase by turning source structure and dependencies into navigable architecture views.

# Scope

The product starts with a Go implementation and is designed to add Python, TypeScript, Rust, and Clojure analyzers through a common plugin contract. It includes source analysis, a language-neutral architecture model, interactive exploration, and headless export.

The reference implementation in `external/` is read-only input to the design. It is not product source and must remain ignored by Git.

# Relationships

- Child: [Analyze source code](/capabilities/analyze-source.md)
- Child: [Generate architecture models](/capabilities/generate-models.md)
- Child: [Explore and inspect architecture](/capabilities/explore-architecture.md)
- Child: [Export and automate](/capabilities/export-and-automate.md)
- Application PRD: [Arch View product requirements](../docs/prd.md)
- Application architecture: [Application architecture summary](../docs/architecture/application-architecture-summary.md)

# Planning baseline

The confirmed product boundary is static architecture discovery and visualization. Manual diagram authoring, cloud collaboration, runtime tracing, and automatic architectural judgment are outside the initial product scope.
