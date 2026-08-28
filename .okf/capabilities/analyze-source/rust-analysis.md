---
type: capability
title: Rust analysis
description: Build architecture input from Rust crates, modules, and use relationships.
tags: [rust, analysis, roadmap]
timestamp: 2026-08-28T06:08:59Z
state: implemented
state_changed: 2026-08-27T13:55:10Z
project: /project.md
parent: /capabilities/analyze-source.md
discovery_notes: docs/architecture/analyze-source/rust-analysis/discovery-notes.md
gap_analysis: docs/architecture/analyze-source/rust-analysis/requirements-gap-analysis.md
orchestration_status: docs/architecture/analyze-source/rust-analysis/orchestration-status.md
implementation_slice: docs/architecture/analyze-source/rust-analysis/implementation-slice.md
issues:
  - docs/agents/issues/done/20260827-023-rust-cargo-boundary-and-registration.md
  - docs/agents/issues/done/20260827-024-rust-module-discovery-and-evidence.md
  - docs/agents/issues/done/20260827-025-rust-relationships-and-end-to-end-output.md
  - docs/agents/issues/done/20260828-034-compiled-analyzer-plugin-entrypoints.md
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

This child capability is implemented. Its crate/workspace selection, module/use semantics, feature/cfg policy, macro uncertainty, options, exclusions, and safety behavior are linked from the exact-spec artifacts and the completed delivery record.

## Delivery progress

Issues 023–025 are complete and archived as ordered AFK slices. Issue 023 established the Cargo boundary and host/CLI registration; issue 024 added module discovery and evidence; issue 025 completed relationships, uncertainty, canonical output, and the required application product/architecture refresh. The Rust analyzer is now implemented behind the existing language-neutral host boundary. Issue 034 adds the compiled `org.archview.rust` entrypoint through the shared process runner and verifies parity with the in-process analyzer on a Cargo fixture; Cargo semantics remain owned by the existing adapter.
