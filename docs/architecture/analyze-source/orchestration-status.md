# Analyze source code orchestration status

## State

- Planning state: `implemented`.
- State transition: `bounded -> specified` on 2026-08-25; `specified ->
  implemented` on 2026-08-29 after all child delivery slices and the configured
  viewer review were completed.
- Topology: confirmed as the source-analysis capability with a plugin runtime and language-specific child territories for Go, Python, TypeScript, Rust, and Clojure.
- Delivery route: the Python child is implemented after issues 017–019 and its visual-review gate. TypeScript issues 020–022, Rust issues 023–025, and Clojure issues 026–029 are complete after repository verification and explicit visual approval; external plugin implementation issues 030–033 are complete. Issues 034–038 complete the compiled-distribution child, issues 039–043 complete multi-analyzer orchestration, and issues 044–047 complete configured assignment, source-scope, cache, and viewer integration. Analyzer work remains independent of the completed viewer configuration and spline slices.

## Confirmed boundary

The current v1 capability receives a repository or project root plus analysis
options, selects one analyzer, and returns validated modules, relationships,
source evidence, and diagnostics. The implemented extension adds a set of
analyzer jobs for mixed or nested projects and assignment-driven scope
selection. It does not assign layers, render diagrams, or export files.

## Confirmed decisions

- One language and project per current v1 analysis run; the combined path now
  supports multiple analyzer jobs for mixed or nested projects.
- Package or module nodes are the default graph granularity; files remain attached evidence.
- Static dependency relationships are the first supported relationship type.
- Project-local modules appear by default. External, standard-library, and unresolved dependencies become metadata or diagnostics.
- Tests, generated code, vendor directories, caches, build outputs, and directories named `external` are excluded by default.
- Partial results are returned when dependencies cannot be resolved.
- Relationships carry source file and parser-provided line and column evidence when available.
- Analyzer selection supports auto-detection, explicit CLI selection, and
  configured project assignments that select several analyzer/project-root
  pairs.
- Each analyzer owns project-boundary rules.
- Analyzer manifests declare identity, version, language, detection markers, capabilities, and options.
- Built-in Go interfaces remain the in-process path. Explicit external
  analyzers use the versioned NDJSON and JSON Schema process boundary, and
  packaged execution uses compiled external executables built from the same
  analyzer implementations.
- The host validates, normalizes, sorts, and deduplicates analyzer results.
- Target applications and arbitrary project code are never executed during analysis.

## Child maturity

- [Analyzer plugin runtime](plugin-runtime/readiness-review.md): `implemented`.
- [Go analysis](go-analysis/readiness-review.md): `implemented`.
- [Python analysis](python-analysis/readiness-review.md): `implemented`.
- [TypeScript analysis](typescript-analysis/readiness-review.md): `implemented`.
- [Rust analysis](rust-analysis/readiness-review.md): `implemented`, with completed delivery slices 023–025.
- [Clojure compatibility](clojure-compatibility/readiness-review.md): `implemented`, with completed delivery issues 026–029.
- [Compiled external analyzer distribution](plugin-runtime/compiled-external-analyzer-distribution/readiness-review.md): `implemented`, with completed delivery issues 034–038 and documented non-host platform deferrals.
- [Multi-analyzer project orchestration](plugin-runtime/multi-analyzer-orchestration/readiness-review.md): `implemented`, with issues 039–043 verified, archived, and visually approved.
- [Project analyzer assignments and view selection](plugin-runtime/project-analyzer-assignments/readiness-review.md): `implemented`, with issues 044–047 verified, archived, and visually approved.

## Artifact plan

The current capability set is implemented: [PRD](prd.md), [glossary](domain-glossary.md), [canonical domain model](canonical-domain-model.md), [canonical use cases](canonical-use-cases.md), [analyzer contract](canonical-api-cli-contract.md), [acceptance scenarios](acceptance-scenarios.md), and [readiness review](readiness-review.md), alongside the discovery and gap artifacts. The plugin-runtime child adds the published external schemas and [implementation slice](plugin-runtime/implementation-slice.md). The compiled-distribution child has completed implementation issues 034–038; the multi-analyzer child has completed issues 039–043; the project assignment/view child has completed issues 044–047, including its approved visual review.

## Current and target truth

- Observed in reference: the Clojure tool scans Clojure-family files, reads namespace forms, extracts project dependencies, records source files, and marks selected Clojure abstractions.
- User-confirmed target: the repository will become a Go implementation with
  gradual support for multiple languages through plugins, ultimately using
  compiled external analyzer executables and multiple analyzer jobs per
  repository where applicable.
- Required follow-up: the completed in-process and external Python slices
  preserve the common meaning through the published protocol. The implemented
  compiled-distribution child owns packaged analyzer delivery; completed and
  approved issues 039–043 sequence multi-analyzer execution from planning
  through cached viewer scope selection. Issues 044–047 implement the
  configured assignment/source-scope, session-cache, and viewer seam. Issues
  030–033 remain the completed v1 pilot delivery record.

## Current delivery slice

- Completed frontiers: [Go repository to visible architecture view](go-analysis/implementation-slice.md), [Python repository to visible architecture view](python-analysis/implementation-slice.md), [TypeScript repository to language-neutral architecture model](typescript-analysis/implementation-slice.md), [Rust repository to language-neutral architecture model](rust-analysis/implementation-slice.md), [Clojure repository to the shared public path](clojure-compatibility/implementation-slice.md), [external analyzer process to Python parity](plugin-runtime/implementation-slice.md), the compiled external analyzer distribution through issues 034–038, multi-analyzer project orchestration through issues 039–043, and configured assignment/source-scope planning, caching, and viewer integration through issues 044–047.
- Delivery progress: issues 001 through 008, 013, and 016 are complete for the first Go analyzer/model, viewer, export, layout-configuration, analyzer-pipeline, and bounded presentation-extension work. Issues 017–019 complete the Python project-boundary/module-discovery, static import/uncertainty, and shared visible-journey path after automated verification and visual approval. Issues 020–022 complete the TypeScript project-boundary/module-discovery, static dependency/uncertainty, evidence, public CLI, and shared visible-journey slices after automated verification and explicit visual approval. Issues 023–025 complete the Rust Cargo boundary, module/evidence discovery, static relationships, uncertainty, and shared canonical output path. Clojure issues 026–029 complete project/namespace discovery, static dependencies, platform conditionals, polymorphic metadata, safety handling, and the shared public path. Issues 030–033 complete the external protocol/descriptor schema, conformance fixture, process host, external Python parity, explicit CLI, shared viewer/source path, and deterministic exports. Issues 034–038 complete compiled entrypoints, deterministic distribution assembly, trusted package verification, packaged runtime selection, parity, and release verification. Issues 039–043 complete the multi-analyzer project-orchestration path, including the approved viewer visual review. Issues 044–047 complete the verified assignment/view implementation sequence, including the approved visual review. The plugin-runtime parent and project-assignment/view child are implemented.
- The slice crosses the plugin runtime, Go analyzer, canonical model, local viewer, evidence, and export contracts. The approved multi-analyzer sequence also consumes the resolved invocation-root source-scope policy owned by the project-assignment child.

## Artifact sync

- Topology: updated in `.okf/`.
- Capability truth: updated in [discovery notes](discovery-notes.md) and [requirements gap analysis](requirements-gap-analysis.md).
- Exact specification: complete in the linked PRD, glossary, domain model, use cases, contract, scenarios, and readiness review; the plugin-runtime child also links the published external protocol/descriptor schemas and implementation slice. The assignment child defines the persisted source-scope policy; the multi-analyzer child defines its application and cache identity.
- Product truth: reflected in [the application PRD](../../prd.md).
- Architecture truth: reflected in [the application architecture summary](../application-architecture-summary.md).
- Delivery truth: updated with completed issues 001 through 009, 013, and 016–047. Issues 016, 019, and 022 are archived after automated implementation verification and explicit visual approval; issues 020 and 021 are archived after their backend and repository gates; issues 023–025 and 026–029 are archived after Rust and Clojure implementation verification; issues 030–038 are archived after protocol, process-host, external Python parity, compiled-entrypoint, distribution-assembly, trusted package verification, packaged runtime selection, parity, release, repository, and strict OKF verification; issues 039–043 are archived after multi-analyzer implementation, repository verification, and visual approval; issues 044–047 are archived after configuration, planning, cache, viewer, repository, and visual verification. The plugin-runtime parent and project-assignment child are implemented, and all analysis nodes are implemented.
