---
type: capability
title: Generate architecture models
description: Convert analyzer results into a language-neutral, hierarchical, layered architecture model.
tags: [domain-model, graph, layout]
timestamp: 2026-08-25T18:39:17Z
state: specified
state_changed: 2026-08-25T17:10:00Z
project: /project.md
parent: /project.md
artifact_root: docs/architecture/generate-models
orchestration_status: docs/architecture/generate-models/orchestration-status.md
issues:
  - docs/agents/issues/done/20260825-002-go-package-import-model-pipeline.md
discovery_notes: docs/architecture/generate-models/discovery-notes.md
gap_analysis: docs/architecture/generate-models/requirements-gap-analysis.md
prd: docs/architecture/generate-models/prd.md
glossary: docs/architecture/generate-models/domain-glossary.md
domain_model: docs/architecture/generate-models/canonical-domain-model.md
use_cases: docs/architecture/generate-models/canonical-use-cases.md
contract: docs/architecture/generate-models/canonical-api-cli-contract.md
scenarios: docs/architecture/generate-models/acceptance-scenarios.md
readiness_review: docs/architecture/generate-models/readiness-review.md
---

# Intent

Give every analyzer one stable model for modules, hierarchy, relationships, evidence, abstraction metadata, cycles, and layers.

# Scope

This capability owns normalization, identity, hierarchy segments, aggregation, relationship semantics, cycle detection, layer assignment, and model validation. It does not own source parsing or a specific renderer.

# Relationships

- Parent: [Arch View](../project.md)
- Inputs: [Analyze source code](analyze-source.md)
- Consumers: [Explore and inspect architecture](explore-architecture.md), [Export and automate](export-and-automate.md)
- Artifact plan: [Model orchestration status](../../docs/architecture/generate-models/orchestration-status.md)

# Notes

The model must support dot-separated Clojure namespaces, slash-separated Go import paths, Python packages, and TypeScript module paths without encoding any one naming convention.

# Planning state

This capability is specified. Its canonical model schema, normalization rules, projection semantics, CLI/HTTP contract, and acceptance behavior are linked from the exact-spec artifacts.

## Delivery progress

Issue 002 delivers the first canonical `arch-view.model/v1` normalization, integrity validation, cycle/layer derivation, and hierarchy projection path. Non-local references remain fully retained here; hiding, aggregation, expansion, and import-list presentation belong to the viewer/export consumers. The capability remains `specified` while those consumers are delivered.
