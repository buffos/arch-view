---
type: capability
title: Analyze source code
description: Discover supported project structures, modules, and dependency evidence through language analyzers.
tags: [analysis, plugins, languages]
timestamp: 2026-08-27T13:51:53Z
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
  - docs/agents/issues/done/20260826-013-go-analyzer-capability-pipeline.md
  - docs/agents/issues/done/20260827-017-python-project-boundary-and-module-discovery.md
  - docs/agents/issues/done/20260827-018-python-static-import-resolution-and-uncertainty.md
  - docs/agents/issues/done/20260827-019-python-cli-and-visible-architecture-path.md
  - docs/agents/issues/done/20260827-020-typescript-project-boundary-and-module-discovery.md
  - docs/agents/issues/done/20260827-021-typescript-static-dependencies-and-uncertainty.md
  - docs/agents/issues/done/20260827-022-typescript-public-and-visible-analysis-path.md
  - docs/agents/issues/done/20260827-023-rust-cargo-boundary-and-registration.md
  - docs/agents/issues/done/20260827-024-rust-module-discovery-and-evidence.md
  - docs/agents/issues/done/20260827-025-rust-relationships-and-end-to-end-output.md
  - docs/agents/issues/done/20260827-026-clojure-project-and-namespace-discovery.md
  - docs/agents/issues/done/20260827-027-clojure-static-namespace-dependencies.md
  - docs/agents/issues/done/20260827-028-clojure-platform-polymorphism-and-safety.md
  - docs/agents/issues/done/20260827-029-clojure-public-integration-and-deterministic-exports.md
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

Issues 001 through 008, 013, and 016 are complete for the first Go analyzer/model, viewer, export, layout, analyzer-pipeline, and bounded spline extension slice. Issues 017 through 019 complete the registered Python project/module-discovery, static import/uncertainty, and shared visible-journey slice after automated verification and explicit visual approval in [017–019](../../docs/architecture/analyze-source/python-analysis/implementation-slice.md). Issues 020–022 complete the registered TypeScript project/module boundary, deterministic discovery, static dependency resolution, uncertainty, evidence, public CLI, and shared visible-journey path after automated verification and explicit visual approval in [the TypeScript slice](../../docs/architecture/analyze-source/typescript-analysis/implementation-slice.md). Issues 023–025 complete the registered Rust Cargo boundary, module/evidence discovery, static relationships, uncertainty, and shared canonical output path in [the Rust implementation slice](../../docs/architecture/analyze-source/rust-analysis/implementation-slice.md). Issues 026–029 complete the registered Clojure project/namespace, static dependency, platform/polymorphic metadata, safety, and shared public-path slice in [the Clojure implementation slice](../../docs/architecture/analyze-source/clojure-compatibility/implementation-slice.md). The external plugin protocol remains later work, so the parent capability remains `specified`.
