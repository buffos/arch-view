# Analyze source code orchestration status

## State

- Planning state: `specified`.
- State transition: `bounded -> specified` on 2026-08-25.
- Topology: confirmed as the source-analysis capability with a plugin runtime and language-specific child territories for Go, Python, TypeScript, Rust, and Clojure.
- Next route: continue the language-neutral analyzer roadmap alongside the next viewer option slices. Analyzer/plugin implementation remains scoped independently from the completed viewer configuration slice.

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
- [Python analysis](python-analysis/readiness-review.md): `specified`.
- [TypeScript analysis](typescript-analysis/readiness-review.md): `specified`.
- [Rust analysis](rust-analysis/readiness-review.md): `specified`.
- [Clojure compatibility](clojure-compatibility/readiness-review.md): `specified`.

## Artifact plan

The exact capability set is complete: [PRD](prd.md), [glossary](domain-glossary.md), [canonical domain model](canonical-domain-model.md), [canonical use cases](canonical-use-cases.md), [analyzer contract](canonical-api-cli-contract.md), [acceptance scenarios](acceptance-scenarios.md), and [readiness review](readiness-review.md), alongside the discovery and gap artifacts.

## Current and target truth

- Observed in reference: the Clojure tool scans Clojure-family files, reads namespace forms, extracts project dependencies, records source files, and marks selected Clojure abstractions.
- User-confirmed target: the repository will become a Go implementation with gradual support for multiple languages through plugins.
- Required follow-up: implement the specified contract, starting with the in-process Go runtime and Go adapter; future protocol/schema publication must preserve this meaning.

## Current delivery slice

- Selected frontier: Go analysis.
- Implementation slice: [Go repository to visible architecture view](go-analysis/implementation-slice.md).
- Delivery progress: issues 001 through 008 are complete for the first Go analyzer/model, viewer, export, and layout-configuration slice. Issue 013 now isolates the Go scanner, import classifier, and observation assembler behind the existing analyzer entrypoint; future language-adapter work remains specified.
- The slice crosses the plugin runtime, Go analyzer, canonical model, local viewer, evidence, and export contracts.

## Artifact sync

- Topology: updated in `.okf/`.
- Capability truth: updated in [discovery notes](discovery-notes.md) and [requirements gap analysis](requirements-gap-analysis.md).
- Exact specification: complete in the linked PRD, glossary, domain model, use cases, contract, scenarios, and readiness review.
- Product truth: reflected in [the application PRD](../../prd.md).
- Architecture truth: reflected in [the application architecture summary](../application-architecture-summary.md).
- Delivery truth: updated with completed issues 001 through 009, 013, and 016; issue 016 is archived after automated implementation verification and explicit visual approval, while future language-adapter work remains in the specified roadmap.
