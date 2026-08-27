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

The upstream [reference implementation](https://github.com/unclebob/arch-view)
is read-only input to the design. It is not product source.

# Relationships

- Child: [Analyze source code](/capabilities/analyze-source.md)
- Child: [Generate architecture models](/capabilities/generate-models.md)
- Child: [Explore and inspect architecture](/capabilities/explore-architecture.md)
- Child: [Export and automate](/capabilities/export-and-automate.md)
- Application PRD: [Arch View product requirements](../docs/prd.md)
- Application architecture: [Application architecture summary](../docs/architecture/application-architecture-summary.md)

# Planning baseline

The confirmed product boundary is static architecture discovery and visualization. Manual diagram authoring, cloud collaboration, runtime tracing, and automatic architectural judgment are outside the initial product scope.

# Current delivery frontier

The approved [TypeScript analysis implementation slice](../docs/architecture/analyze-source/typescript-analysis/implementation-slice.md) is complete through issues 020–022, as are the Rust issues 023–025 and Clojure issues 026–029. The project boundary, module discovery, static dependency, uncertainty, evidence, public journey, and visual review are complete for all four in-process language adapters. Issues 030–033 now complete the approved [external analyzer implementation slice](../docs/architecture/analyze-source/plugin-runtime/implementation-slice.md), including the process host, external Python parity deployment, and explicit CLI/shared consumer path. The plugin-runtime child remains specified for a later explicitly chosen frontier.
