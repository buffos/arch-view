---
type: capability
title: Analyze source code
description: Discover supported project structures, modules, and dependency evidence through language analyzers.
tags: [analysis, plugins, languages]
timestamp: 2026-08-25T18:39:17Z
state: specified
state_changed: 2026-08-25T17:10:00Z
project: /project.md
parent: /project.md
children:
  - /capabilities/analyze-source/plugin-runtime.md
  - /capabilities/analyze-source/go-analysis.md
  - /capabilities/analyze-source/python-analysis.md
  - /capabilities/analyze-source/typescript-analysis.md
  - /capabilities/analyze-source/rust-analysis.md
  - /capabilities/analyze-source/clojure-compatibility.md
artifact_root: docs/architecture/analyze-source
discovery_notes: docs/architecture/analyze-source/discovery-notes.md
gap_analysis: docs/architecture/analyze-source/requirements-gap-analysis.md
orchestration_status: docs/architecture/analyze-source/orchestration-status.md
implementation_slice: docs/architecture/analyze-source/go-analysis/implementation-slice.md
issues:
  - docs/agents/issues/done/20260825-001-analyzer-host-and-go-project-selection.md
  - docs/agents/issues/done/20260825-002-go-package-import-model-pipeline.md
prd: docs/architecture/analyze-source/prd.md
glossary: docs/architecture/analyze-source/domain-glossary.md
domain_model: docs/architecture/analyze-source/canonical-domain-model.md
use_cases: docs/architecture/analyze-source/canonical-use-cases.md
contract: docs/architecture/analyze-source/canonical-api-cli-contract.md
scenarios: docs/architecture/analyze-source/acceptance-scenarios.md
readiness_review: docs/architecture/analyze-source/readiness-review.md
---

# Intent

Analyze a repository without executing its application and produce reliable structural and dependency information for the architecture model.

# Scope

This capability owns project detection, source scope, language-specific parsing, dependency resolution, evidence, diagnostics, and analyzer extension points. It does not own graph layout or user interface behavior.

# Relationships

- Parent: [Arch View](../project.md)
- Child: [Analyzer plugin runtime](analyze-source/plugin-runtime.md)
- Child: [Go analysis](analyze-source/go-analysis.md)
- Child: [Python analysis](analyze-source/python-analysis.md)
- Child: [TypeScript analysis](analyze-source/typescript-analysis.md)
- Child: [Rust analysis](analyze-source/rust-analysis.md)
- Child: [Clojure compatibility](analyze-source/clojure-compatibility.md)
- Downstream model: [Generate architecture models](generate-models.md)
- Artifact plan: [Analysis orchestration status](../../docs/architecture/analyze-source/orchestration-status.md)

# Notes

The reference tool reads Clojure forms and currently recognizes Clojure-family source files. The target design keeps that behavior inside a language adapter and exposes a language-neutral result.

The capability is specified. Its observation schema, analyzer lifecycle, safety rules, language adapter contracts, and external protocol direction are linked from the exact-spec artifacts.

## Delivery progress

Issues 001 and 002 are complete for the first Go analyzer/model slice. Viewer and export work remains in issues 003–005; the capability remains `specified` while that scoped delivery continues.
