---
type: project
title: Arch View
description: Analyze supported codebases and generate navigable architecture views.
tags: [architecture, code-analysis, multi-language, golang]
timestamp: 2026-08-30T08:15:05Z
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
and Clojure analyzers through a common plugin contract. The confirmed direction
is to distribute supported analyzers as compiled external executables, run
multiple analyzers for one repository, and let project configuration map
folders to analyzer IDs for selectable application views. These analysis and
assignment flows are implemented; advanced renderer extensions remain future
work. The product includes source
analysis, a language-neutral architecture model, interactive exploration, and
headless export. The confirmed future direction also includes deterministic
code-quality checks, a searchable code-intelligence index, and a live MCP
surface for compact, evidence-backed repository queries.

The upstream [reference implementation](https://github.com/unclebob/arch-view)
is read-only input to the design. It is not product source.

# Relationships

- Child: [Analyze source code](/capabilities/analyze-source.md)
- Child: [Generate architecture models](/capabilities/generate-models.md)
- Child: [Explore and inspect architecture](/capabilities/explore-architecture.md)
- Child: [Export and automate](/capabilities/export-and-automate.md)
- Child: [Code quality and code intelligence](/capabilities/code-quality-and-intelligence.md)
- Application PRD: [Arch View product requirements](../docs/prd.md)
- Application architecture: [Application architecture summary](../docs/architecture/application-architecture-summary.md)

# Planning baseline

The confirmed product boundary is static architecture discovery, visualization,
and a future deterministic code-quality/code-intelligence extension. Manual
diagram authoring, cloud collaboration, runtime tracing, and subjective or
LLM-generated architectural judgment remain outside the deterministic product
scope.

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
multi-analyzer orchestration leaf is implemented through the verified and
user-approved issues 039–043, including its mixed-language viewer review. The
project analyzer assignments/view selection leaf is implemented through
issues 044–047, including its approved visual review. The advanced ELK renderer
support leaf remains specified and ready for later delivery issue slicing.
The [Code quality and code intelligence](capabilities/code-quality-and-intelligence.md)
capability is a pure structural-child roll-up with effective state `specified`,
the minimum of its three children. Its three children are readiness-reviewed;
the deterministic-quality child is implemented and live-analysis/MCP has
verified implementation issues 064–075. The source-facts child is implemented
through issues 048–052, including its approved final visual gate; the parent
has no standalone implementation slice or PRD, and remaining implementation is
routed to the other children. Deterministic quality checks delivered and
archived issues 053–063, including the final desktop and responsive visual
review. The live-analysis/MCP child remains `specified` only for issue 076's
open final product-approval gate, so the parent remains `specified`.
