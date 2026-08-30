---
type: capability
title: Analyze source code
description: Discover supported project structures, modules, and dependency evidence through language analyzers.
tags: [analysis, plugins, languages]
timestamp: 2026-08-29T00:00:00Z
state: implemented
state_changed: 2026-08-29T00:00:00Z
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
  - docs/agents/issues/done/20260827-030-external-protocol-schema-and-conformance-fixture.md
  - docs/agents/issues/done/20260827-031-process-backed-analyzer-host-runtime.md
  - docs/agents/issues/done/20260827-032-external-python-analyzer-parity.md
  - docs/agents/issues/done/20260827-033-external-plugin-cli-and-visible-journey.md
  - docs/agents/issues/done/20260828-034-compiled-analyzer-plugin-entrypoints.md
  - docs/agents/issues/done/20260828-035-compiled-analyzer-distribution-assembly.md
  - docs/agents/issues/done/20260828-036-trusted-analyzer-package-verification.md
  - docs/agents/issues/done/20260828-037-packaged-runtime-selection.md
  - docs/agents/issues/done/20260828-038-compiled-analyzer-parity-and-release-verification.md
  - docs/agents/issues/done/20260828-039-multi-project-root-discovery-and-job-planning.md
  - docs/agents/issues/done/20260828-040-bounded-multi-analyzer-execution.md
  - docs/agents/issues/done/20260828-041-namespaced-aggregate-model-and-status.md
  - docs/agents/issues/done/20260828-042-combined-analysis-cli-and-http-exposure.md
  - docs/agents/issues/done/20260828-043-cached-scope-projections-and-viewer-selection.md
  - docs/agents/issues/done/20260829-044-load-and-validate-analysis-configuration.md
  - docs/agents/issues/done/20260829-045-resolve-configured-assignments-and-source-scopes.md
  - docs/agents/issues/done/20260829-046-session-cache-and-selective-invalidation.md
  - docs/agents/issues/done/20260829-047-configured-scope-viewer-journey.md
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

The capability is implemented. Its observation schema, analyzer lifecycle,
safety rules, language adapter contracts, external process pilot, compiled
distribution, multi-analyzer orchestration, project assignments, and
invocation-root source-scope policy are linked from the exact-spec artifacts.
The compiled-distribution child is implemented after
issues 034–038 completed entrypoint migration, distribution assembly, trusted
verification, packaged runtime selection, parity, and release verification;
issues 039–043 complete the multi-analyzer discovery, execution, aggregation,
CLI/HTTP path, and cached viewer scope selection after its approved visual
review. Issues 044–047 completed configuration validation, planner integration,
selective caching, and configured viewer integration, including the approved
visual review. All analysis children are implemented.

## Delivery progress

Issues 001 through 008, 013, and 016 are complete for the first Go analyzer/model, viewer, export, layout, analyzer-pipeline, and bounded spline extension slice. Issues 017 through 019 complete the registered Python project/module-discovery, static import/uncertainty, and shared visible-journey slice after automated verification and explicit visual approval in [017–019](../../docs/architecture/analyze-source/python-analysis/implementation-slice.md). Issues 020–022 complete the registered TypeScript project/module boundary, deterministic discovery, static dependency resolution, uncertainty, evidence, public CLI, and shared visible-journey path after automated verification and explicit visual approval in [the TypeScript slice](../../docs/architecture/analyze-source/typescript-analysis/implementation-slice.md). Issues 023–025 complete the registered Rust Cargo boundary, module/evidence discovery, static relationships, uncertainty, and shared canonical output path in [the Rust implementation slice](../../docs/architecture/analyze-source/rust-analysis/implementation-slice.md). Issues 026–029 complete the registered Clojure project/namespace, static dependency, platform/polymorphic metadata, safety, and shared public-path slice in [the Clojure implementation slice](../../docs/architecture/analyze-source/clojure-compatibility/implementation-slice.md). Issues 030–033 complete the protocol foundation, process host, external Python parity, and public shared path in [the plugin-runtime implementation slice](../../docs/architecture/analyze-source/plugin-runtime/implementation-slice.md). Issues 034–038 complete the compiled-plugin runner, five compiled analyzer entrypoints, package assembly, trusted verification, packaged runtime selection, explicit fallback, parity, and release verification. Issues 039–043 complete the multi-analyzer project-orchestration path, including its approved viewer visual review. Issues 044–047 complete the project-assignment/configuration and configured-viewer path, including its approved visual review. The capability and all of its children are implemented.
