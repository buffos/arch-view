---
type: capability
title: Go analysis
description: Build architecture input from Go modules, packages, files, and imports.
tags: [go, analysis, mvp]
timestamp: 2026-08-26T20:56:17Z
state: specified
state_changed: 2026-08-25T17:10:00Z
project: /project.md
parent: /capabilities/analyze-source.md
discovery_notes: docs/architecture/analyze-source/go-analysis/discovery-notes.md
gap_analysis: docs/architecture/analyze-source/go-analysis/requirements-gap-analysis.md
orchestration_status: docs/architecture/analyze-source/go-analysis/orchestration-status.md
implementation_slice: docs/architecture/analyze-source/go-analysis/implementation-slice.md
issues:
  - docs/agents/issues/done/20260825-001-analyzer-host-and-go-project-selection.md
  - docs/agents/issues/done/20260825-002-go-package-import-model-pipeline.md
  - docs/agents/issues/done/20260826-013-go-analyzer-capability-pipeline.md
prd: docs/architecture/analyze-source/go-analysis/prd.md
glossary: docs/architecture/analyze-source/go-analysis/domain-glossary.md
domain_model: docs/architecture/analyze-source/go-analysis/canonical-domain-model.md
use_cases: docs/architecture/analyze-source/go-analysis/canonical-use-cases.md
contract: docs/architecture/analyze-source/go-analysis/canonical-api-cli-contract.md
scenarios: docs/architecture/analyze-source/go-analysis/acceptance-scenarios.md
readiness_review: docs/architecture/analyze-source/go-analysis/readiness-review.md
---

# Intent

Make Go repositories the first fully supported input for Arch View.

# Scope

The analyzer will detect Go project boundaries, discover packages, resolve local imports, retain source evidence, distinguish standard or external dependencies from project dependencies, and report analysis limitations.

# Relationships

- Parent: [Analyze source code](../analyze-source.md)
- Produces input for: [Generate architecture models](../generate-models.md)

# Notes

The first Go slice should prioritize package dependency graphs and a visible architecture result over call-graph completeness.

# Planning state

This child capability is specified. Its module/workspace selection, package/import semantics, options, exclusions, evidence, diagnostics, and safety behavior are linked from the exact-spec artifacts.

## Delivery progress

Issues 001, 002, and 013 established the Go project/analyzer path, including the separated scanning, import classification, and observation assembly pipeline behind the same analyzer entrypoint. The remaining Python, TypeScript, Rust, and Clojure analyzers remain separate future implementations.
