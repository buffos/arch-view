---
type: capability
title: Compiled external analyzer distribution
description: Distribute each language analyzer as a versioned compiled executable without requiring its language runtime.
tags: [plugins, distribution, executables, extensibility]
timestamp: 2026-08-28T06:08:59Z
state: implemented
state_changed: 2026-08-28T16:13:46Z
project: /project.md
parent: /capabilities/analyze-source/plugin-runtime.md
artifact_root: docs/architecture/analyze-source/plugin-runtime/compiled-external-analyzer-distribution
discovery_notes: docs/architecture/analyze-source/plugin-runtime/compiled-external-analyzer-distribution/discovery-notes.md
gap_analysis: docs/architecture/analyze-source/plugin-runtime/compiled-external-analyzer-distribution/requirements-gap-analysis.md
orchestration_status: docs/architecture/analyze-source/plugin-runtime/compiled-external-analyzer-distribution/orchestration-status.md
prd: docs/architecture/analyze-source/plugin-runtime/compiled-external-analyzer-distribution/prd.md
glossary: docs/architecture/analyze-source/plugin-runtime/compiled-external-analyzer-distribution/domain-glossary.md
domain_model: docs/architecture/analyze-source/plugin-runtime/compiled-external-analyzer-distribution/canonical-domain-model.md
use_cases: docs/architecture/analyze-source/plugin-runtime/compiled-external-analyzer-distribution/canonical-use-cases.md
contract: docs/architecture/analyze-source/plugin-runtime/compiled-external-analyzer-distribution/canonical-api-cli-contract.md
scenarios: docs/architecture/analyze-source/plugin-runtime/compiled-external-analyzer-distribution/acceptance-scenarios.md
readiness_review: docs/architecture/analyze-source/plugin-runtime/compiled-external-analyzer-distribution/readiness-review.md
issues:
  - docs/agents/issues/done/20260828-034-compiled-analyzer-plugin-entrypoints.md
  - docs/agents/issues/done/20260828-035-compiled-analyzer-distribution-assembly.md
  - docs/agents/issues/done/20260828-036-trusted-analyzer-package-verification.md
  - docs/agents/issues/done/20260828-037-packaged-runtime-selection.md
  - docs/agents/issues/done/20260828-038-compiled-analyzer-parity-and-release-verification.md
---

# Intent

Make every supported language analyzer installable and launchable as a
versioned compiled executable while preserving the common analyzer contract.

# Scope

This capability owns executable entrypoints, platform-specific artifacts,
descriptor packaging, analyzer version compatibility, trusted application-
managed discovery, and the migration path from in-process implementations to
isolated external processes. The release build must compile each analyzer and
assemble the expected `analyzers` directory automatically. It does not redefine
language semantics, the protocol contract, or the canonical architecture model.

# Relationships

- Parent: [Analyzer plugin runtime](/capabilities/analyze-source/plugin-runtime.md)
- Uses: [Analyze source code](/capabilities/analyze-source.md)
- Produces input for: [Generate architecture models](/capabilities/generate-models.md)

# Planning state

This child is implemented and readiness-reviewed. The user-confirmed target is
a compiled external analyzer binary that reuses the same implementation as the
in-process adapter, with no Python, Rust, Node, or other language runtime
required by the end user. The compiled artifact is the production
implementation for a logical analyzer ID; the in-process adapter remains only
for development, tests, or an explicit migration fallback. The exact package
index, platform matrix, SHA-256 integrity policy, build targets, runtime modes,
and acceptance scenarios are linked above. The current external Python
deployment remains a script-based parity pilot and is not treated as the target
distribution model.

## Delivery progress

Issue 034 is complete: the existing Go, Python, TypeScript, Rust, and Clojure
analyzers now have compiled protocol entrypoints backed by one shared process
runner, with manifest, option, diagnostic, cancellation, and result parity
verified through the existing process adapter and host. Issue 035 is complete:
the deterministic package assembly, descriptor/index generation, SHA-256
integrity metadata, exact platform matrix, and atomic `make analyzers`/
`make release` targets are implemented and verified. Issues 036–038 are now
complete: trusted package verification, packaged runtime selection, explicit
fallback policy, runtime provenance, five-analyzer parity, deterministic
assembly, and Windows release behavior are verified. Linux amd64 and Darwin
arm64 execution remain explicitly deferred under the root `when-supported`
policy until matching runners/toolchains are available. This node is
`implemented`; the parent plugin-runtime capability and its later
multi-analyzer and assignment/view children are implemented after their
verified delivery batches.
