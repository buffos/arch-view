---
type: capability
title: Analyzer plugin runtime
description: Discover, configure, execute, and version language analyzers behind a stable contract.
tags: [plugins, extensibility, analysis]
timestamp: 2026-08-28T06:08:59Z
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
an opt-in process boundary for analyzers written outside Go. Its future
frontier now includes compiled external analyzer distribution, concurrent
multi-analyzer project orchestration, and durable project analyzer assignments
with application-level scope selection.

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
The user-confirmed longer-term direction is to distribute supported analyzers
as compiled external executables, run multiple applicable analyzers together,
and persist project-relative analyzer assignments for application scope
selection. Those future behaviors are represented by specified child nodes and
are not claimed as implemented.

## Delivery progress

Issues 030–033 completed the published external protocol and descriptor
schemas, bounded typed frame codec, stateful conformance validator, process-host
lifecycle, external Python parity, and explicit public integration. Issue 034
completed the shared compiled-plugin runner and five compiled analyzer
entrypoints while preserving the existing in-process implementations. Issue 035
completed deterministic distribution assembly and the explicit analyzer/release
build targets. The parent remains `specified`; the compiled-distribution child
continues through issues 036–038 for trust, runtime selection, and release
verification.
The multi-analyzer and project assignment/view children remain `specified`
future frontiers, ready for later delivery issue slicing.

# Planning state

This child capability remains specified after its first external process slice
was implemented and verified. Its host/plugin API, manifest fields, selection
rules, lifecycle, safety policy, published external schemas, and external
Python implementation are linked from the exact-spec artifacts. Broader
compiled distribution, multi-analyzer execution, and project assignment/view
selection are now specified future frontiers under the child nodes above.
