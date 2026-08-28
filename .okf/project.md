---
type: project
title: Arch View
description: Analyze supported codebases and generate navigable architecture views.
tags: [architecture, code-analysis, multi-language, golang]
timestamp: 2026-08-28T06:08:59Z
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

The product starts with a Go implementation and adds Python, TypeScript, Rust,
and Clojure analyzers through a common plugin contract. The confirmed future
direction is to distribute supported analyzers as compiled external executables,
run multiple analyzers for one repository, and let project configuration map
folders to analyzer IDs for selectable application views. It includes source
analysis, a language-neutral architecture model, interactive exploration, and
headless export.

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

The Go, Python, TypeScript, Rust, and Clojure analysis children are implemented.
The canonical model, export and automation, and current Explore scopes are
implemented as well. Issues 030–033 complete the approved [external analyzer
implementation slice](../docs/architecture/analyze-source/plugin-runtime/implementation-slice.md),
including the process host, external Python parity deployment, and explicit
CLI/shared consumer path. Issues 034–038 complete the compiled entrypoint
migration, deterministic distribution assembly, trusted package verification,
packaged runtime selection, parity, and release verification. The compiled
distribution child is implemented; its Linux amd64 and Darwin arm64 execution
checks remain documented under the root `when-supported` policy. The
multi-analyzer orchestration, project analyzer assignments/view selection, and
advanced ELK renderer support leaves remain ready for delivery issue slicing
without delivery issues.
