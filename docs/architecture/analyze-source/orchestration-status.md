# Analyze source code orchestration status

## State

- Planning state: `specified`.
- State transition: `bounded -> specified` on 2026-08-25.
- Topology: confirmed as the source-analysis capability with a plugin runtime and language-specific child territories for Go, Python, TypeScript, Rust, and Clojure.
- Next route: the Python child is implemented after issues 017–019 and its visual-review gate. Rust is implemented after issues 023–025, and the user-selected Clojure child is implemented after issues 026–029; TypeScript is the next specified language frontier, followed by the external plugin protocol. Analyzer work remains independent of the completed viewer configuration and spline slices.

## Confirmed boundary

The capability receives a repository or project root plus analysis options, selects an analyzer, and returns validated modules, relationships, source evidence, and diagnostics. It does not assign layers, render diagrams, or export files.

## Confirmed decisions

- One language and project per analysis run.
- Package or module nodes are the default graph granularity; files remain attached evidence.
- Static dependency relationships are the first supported relationship type.
- Project-local modules appear by default. External, standard-library, and unresolved dependencies become metadata or diagnostics.
- Tests, generated code, vendor directories, caches, build outputs, and directories named `external` are excluded by default.
- Partial results are returned when dependencies cannot be resolved.
- Relationships carry source file and parser-provided line and column evidence when available.
- Analyzer selection supports auto-detection and an explicit language override.
- Each analyzer owns project-boundary rules.
- Analyzer manifests declare identity, version, language, detection markers, capabilities, and options.
- Built-in Go interfaces come first. External analyzers use versioned NDJSON and JSON Schema later.
- The host validates, normalizes, sorts, and deduplicates analyzer results.
- Target applications and arbitrary project code are never executed during analysis.

## Child maturity

- [Analyzer plugin runtime](plugin-runtime/readiness-review.md): `specified`.
- [Go analysis](go-analysis/readiness-review.md): `specified`.
- [Python analysis](python-analysis/readiness-review.md): `implemented`.
- [TypeScript analysis](typescript-analysis/readiness-review.md): `specified`.
- [Rust analysis](rust-analysis/readiness-review.md): `implemented`, with completed delivery slices 023–025.
- [Clojure compatibility](clojure-compatibility/readiness-review.md): `implemented`, with completed delivery issues 026–029.

## Artifact plan

The exact capability set is complete: [PRD](prd.md), [glossary](domain-glossary.md), [canonical domain model](canonical-domain-model.md), [canonical use cases](canonical-use-cases.md), [analyzer contract](canonical-api-cli-contract.md), [acceptance scenarios](acceptance-scenarios.md), and [readiness review](readiness-review.md), alongside the discovery and gap artifacts.

## Current and target truth

- Observed in reference: the Clojure tool scans Clojure-family files, reads namespace forms, extracts project dependencies, records source files, and marks selected Clojure abstractions.
- User-confirmed target: the repository will become a Go implementation with gradual support for multiple languages through plugins.
- Required follow-up: implement the specified contract, starting with the in-process Go runtime and Go adapter; future protocol/schema publication must preserve this meaning.

## Current delivery slice

- Completed frontiers: [Go repository to visible architecture view](go-analysis/implementation-slice.md), [Python repository to visible architecture view](python-analysis/implementation-slice.md), [Rust repository to language-neutral architecture model](rust-analysis/implementation-slice.md), and [Clojure repository to the shared public path](clojure-compatibility/implementation-slice.md).
- Delivery progress: issues 001 through 008, 013, and 016 are complete for the first Go analyzer/model, viewer, export, layout-configuration, analyzer-pipeline, and bounded presentation-extension work. Issues 017–019 complete the Python project-boundary/module-discovery, static import/uncertainty, and shared visible-journey path after automated verification and visual approval. Issues 023–025 complete the Rust Cargo boundary, module/evidence discovery, static relationships, uncertainty, and shared canonical output path. Clojure issues 026–029 complete project/namespace discovery, static dependencies, platform conditionals, polymorphic metadata, safety handling, and the shared public path; TypeScript and external protocol work remain specified.
- The slice crosses the plugin runtime, Go analyzer, canonical model, local viewer, evidence, and export contracts.

## Artifact sync

- Topology: updated in `.okf/`.
- Capability truth: updated in [discovery notes](discovery-notes.md) and [requirements gap analysis](requirements-gap-analysis.md).
- Exact specification: complete in the linked PRD, glossary, domain model, use cases, contract, scenarios, and readiness review.
- Product truth: reflected in [the application PRD](../../prd.md).
- Architecture truth: reflected in [the application architecture summary](../application-architecture-summary.md).
- Delivery truth: updated with completed issues 001 through 009, 013, 016–019, 023–025, and 026–029. Issues 016 and 019 are archived after automated implementation verification and explicit visual approval; issues 023–025 and 026–029 are archived after Rust and Clojure implementation verification. The Python, Rust, and Clojure nodes are implemented while the parent remains specified for later analyzer and protocol territories.
