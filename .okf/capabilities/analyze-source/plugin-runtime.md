---
type: capability
title: Analyzer plugin runtime
description: Discover, configure, execute, and version language analyzers behind a stable contract.
tags: [plugins, extensibility, analysis]
timestamp: 2026-08-27T20:48:23Z
state: specified
state_changed: 2026-08-25T17:10:00Z
project: /project.md
parent: /capabilities/analyze-source.md
discovery_notes: docs/architecture/analyze-source/plugin-runtime/discovery-notes.md
gap_analysis: docs/architecture/analyze-source/plugin-runtime/requirements-gap-analysis.md
orchestration_status: docs/architecture/analyze-source/plugin-runtime/orchestration-status.md
issues:
  - docs/agents/issues/done/20260825-001-analyzer-host-and-go-project-selection.md
  - docs/agents/issues/done/20260827-030-external-protocol-schema-and-conformance-fixture.md
  - docs/agents/issues/done/20260827-031-process-backed-analyzer-host-runtime.md
  - docs/agents/issues/done/20260827-032-external-python-analyzer-parity.md
  - docs/agents/issues/done/20260827-033-external-plugin-cli-and-visible-journey.md
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

The capability includes analyzer registration, project detection, explicit
language selection, configuration, result validation, diagnostics, and an
opt-in process boundary for analyzers written outside Go.

# Relationships

- Parent: [Analyze source code](../analyze-source.md)
- Consumes: [Generate architecture models](../generate-models.md)

# Notes

The first implementation used built-in Go interfaces. The next staged
extension is an opt-in versioned JSON process protocol, validated first by an
external Python parity plugin; it does not replace the built-in analyzers.

## Delivery progress

Issues 030–033 completed the published external protocol and descriptor
schemas, bounded typed frame codec, stateful conformance validator, process-host
lifecycle, external Python parity, and explicit public integration. This
capability remains `specified` until a subsequent external plugin-runtime
frontier is explicitly chosen.

# Planning state

This child capability remains specified after its first external process slice
was implemented and verified. Its host/plugin API, manifest fields, selection
rules, lifecycle, safety policy, published external schemas, and external
Python implementation are linked from the exact-spec artifacts. Future work
such as broader plugin distribution or discovery requires a new explicitly
chosen frontier.
