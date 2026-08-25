---
type: capability
title: Rust analysis
description: Build architecture input from Rust crates, modules, and use relationships.
tags: [rust, analysis, roadmap]
timestamp: 2026-08-25T15:15:46Z
state: specified
state_changed: 2026-08-25T17:10:00Z
project: /project.md
parent: /capabilities/analyze-source.md
discovery_notes: docs/architecture/analyze-source/rust-analysis/discovery-notes.md
gap_analysis: docs/architecture/analyze-source/rust-analysis/requirements-gap-analysis.md
orchestration_status: docs/architecture/analyze-source/rust-analysis/orchestration-status.md
prd: docs/architecture/analyze-source/rust-analysis/prd.md
glossary: docs/architecture/analyze-source/rust-analysis/domain-glossary.md
domain_model: docs/architecture/analyze-source/rust-analysis/canonical-domain-model.md
use_cases: docs/architecture/analyze-source/rust-analysis/canonical-use-cases.md
contract: docs/architecture/analyze-source/rust-analysis/canonical-api-cli-contract.md
scenarios: docs/architecture/analyze-source/rust-analysis/acceptance-scenarios.md
readiness_review: docs/architecture/analyze-source/rust-analysis/readiness-review.md
---

# Intent

Add static architecture discovery for Rust repositories while preserving crate and module boundaries in the shared model.

# Scope

The analyzer will detect Cargo projects and workspace boundaries, discover crates and modules, resolve local `use` and dependency relationships where possible, retain source evidence, and report generated or unresolved code.

# Relationships

- Parent: [Analyze source code](../analyze-source.md)
- Produces input for: [Generate architecture models](../generate-models.md)

# Planning state

This child capability is specified. Its crate/workspace selection, module/use semantics, feature/cfg policy, macro uncertainty, options, exclusions, and safety behavior are linked from the exact-spec artifacts.
