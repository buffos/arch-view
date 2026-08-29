---
type: capability
title: Analyzer plugin runtime
description: Discover, configure, execute, and version language analyzers behind a stable contract.
tags: [plugins, extensibility, analysis]
timestamp: 2026-08-29T00:00:00Z
state: specified
state_changed: 2026-08-25T17:10:00Z
project: /project.md
parent: /capabilities/analyze-source.md
children:
  - /capabilities/analyze-source/plugin-runtime/compiled-external-analyzer-distribution.md
  - /capabilities/analyze-source/plugin-runtime/multi-analyzer-orchestration.md
  - /capabilities/analyze-source/plugin-runtime/project-analyzer-assignments.md
discovery_notes: docs/architecture/analyze-source/plugin-runtime/discovery-notes.md
gap_analysis: docs/architecture/analyze-source/plugin-runtime/requirements-gap-analysis.md
orchestration_status: docs/architecture/analyze-source/plugin-runtime/orchestration-status.md
issues:
  - docs/agents/issues/done/20260825-001-analyzer-host-and-go-project-selection.md
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
prd: docs/architecture/analyze-source/plugin-runtime/prd.md
glossary: docs/architecture/analyze-source/plugin-runtime/domain-glossary.md
domain_model: docs/architecture/analyze-source/plugin-runtime/canonical-domain-model.md
use_cases: docs/architecture/analyze-source/plugin-runtime/canonical-use-cases.md
contract: docs/architecture/analyze-source/plugin-runtime/canonical-api-cli-contract.md
scenarios: docs/architecture/analyze-source/plugin-runtime/acceptance-scenarios.md
readiness_review: docs/architecture/analyze-source/plugin-runtime/readiness-review.md
implementation_slice: docs/architecture/analyze-source/plugin-runtime/implementation-slice.md
---

# Intent

Allow new language analyzers to be added without changing the architecture model, layout engine, or viewer.

# Scope

The current specified scope includes analyzer registration, project detection,
explicit language selection, configuration, result validation, diagnostics, and
an opt-in process boundary for analyzers written outside Go. Its compiled
external analyzer distribution and concurrent multi-analyzer project
orchestration children are implemented. The remaining future frontier is
durable project analyzer assignments with invocation-root source-scope policy
and application-level assignment-driven scope selection.

# Relationships

- Parent: [Analyze source code](../analyze-source.md)
- Child: [Compiled external analyzer distribution](plugin-runtime/compiled-external-analyzer-distribution.md)
- Child: [Multi-analyzer project orchestration](plugin-runtime/multi-analyzer-orchestration.md)
- Child: [Project analyzer assignments and view selection](plugin-runtime/project-analyzer-assignments.md)
- Consumes: [Generate architecture models](../generate-models.md)

# Notes

The first implementation used built-in Go interfaces. The current staged
extension is an opt-in versioned JSON process protocol, validated first by an
external Python parity plugin; it does not replace the built-in analyzers.
Supported analyzers are now also distributed as compiled external executables
through the implemented child capability, with trusted application-managed
selection and explicit in-process/descriptor overrides. Running multiple
applicable analyzers together is implemented through the multi-analyzer child.
Persisting project-relative analyzer assignments plus invocation-root
source-scope policies for application scope selection remain specified child
work and are not claimed as implemented; the approved delivery sequence is
issues 044–047.

## Delivery progress

Issues 030–033 completed the published external protocol and descriptor
schemas, bounded typed frame codec, stateful conformance validator, process-host
lifecycle, external Python parity, and explicit public integration. Issues
034–038 completed the shared compiled-plugin runner, five compiled analyzer
entrypoints, deterministic distribution assembly, trusted package verification,
packaged runtime selection, explicit fallback, parity, and release
verification. The parent remains `specified`; the project assignment/view and
source-scope child remains specified pending issue 047's visual gate. Issues
039–043 completed the
multi-analyzer discovery, bounded execution, aggregation, CLI/HTTP delivery,
and cached viewer scope selection, including its approved visual review; issues
044–046 completed the separate assignment/configuration implementation, while
issue 047 remains active for its visual-review sequence.

# Planning state

This child capability remains specified after its first external process slice,
the compiled-distribution child, and the multi-analyzer child were implemented
and verified. Its host/plugin API, manifest fields, selection rules, lifecycle,
safety policy, published external schemas, external Python implementation, and
packaged distribution boundary are linked from the exact-spec artifacts.
Project assignment/view selection and its source-scope policy remain specified
until the active child issue's visual gate is approved.
