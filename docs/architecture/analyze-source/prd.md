# Analyze source code PRD

## Purpose

Analyze one supported project statically and return trustworthy, traceable observations for the language-neutral architecture model.

## Goals

- Select one analyzer for one project/language per run.
- Discover project-local modules and static dependency relationships.
- Preserve file and parser-location evidence.
- Make unresolved, dynamic, external, and excluded code visible through diagnostics and metadata.
- Return deterministic, partial results when recoverable problems occur.
- Never execute the target application or arbitrary project code.

## Actors

- Developer: analyzes a repository to understand its structure.
- Maintainer: investigates dependency and resolution problems.
- CI job: runs repeatable headless analysis.
- Analyzer host: selects, configures, validates, and normalizes analyzer output.

## Scope

The capability receives a repository root and analysis options and returns project metadata, analyzer identity, module observations, relationship observations, source evidence, and diagnostics. Package/module nodes are the default granularity; files are evidence.

The capability does not own canonical graph normalization, cycle/layer calculation, rendering, source editing, export packaging, runtime tracing, or complete call-graph analysis.

## Workflows

1. Detect or explicitly select an analyzer.
2. Resolve project boundaries and effective scope options.
3. Read project configuration and source files without executing them.
4. Emit modules, static relationships, evidence, metadata, and diagnostics.
5. Validate, normalize, sort, and return a complete or partial result.

## Rules

- Explicit language selection overrides auto-detection.
- Auto-detection must find exactly one highest-confidence analyzer; ambiguity is a user-visible failure.
- CLI options override project configuration, which overrides analyzer defaults.
- Project-local modules are included by default; external, standard-library, and unresolved targets are references/diagnostics, not local nodes.
- Tests, generated files, vendor directories, build output, caches, `.git`, and directories named `external` are excluded by default.
- Source paths are repository-relative and locations are one-based when provided.
- Recoverable unresolved or dynamic relationships produce `partial`, not total failure.

## Functional requirements

| ID | Requirement |
|---|---|
| AS-FR-001 | Accept a project root and normalized analysis options. |
| AS-FR-002 | Select an analyzer explicitly or through unambiguous detection. |
| AS-FR-003 | Return stable module identities, hierarchy paths, and source references. |
| AS-FR-004 | Return typed static dependency observations with evidence. |
| AS-FR-005 | Return diagnostics with severity, code, subject, and recoverability. |
| AS-FR-006 | Preserve usable results when individual files or relationships cannot be resolved. |
| AS-FR-007 | Produce deterministic collection ordering and analyzer provenance. |
| AS-FR-008 | Refuse target-code execution and report unsupported dynamic behavior. |

## Non-functional requirements

- Read-only with respect to the target repository.
- Cancellation-aware and bounded by host timeouts where supported.
- Language-neutral result vocabulary.
- Evidence must remain traceable after downstream aggregation.
- A future process analyzer must be able to produce the same observation semantics.

## Acceptance summary

The detailed scenarios are in [acceptance scenarios](acceptance-scenarios.md). The capability is ready for implementation when all required observation fields and status/error semantics are honored by built-in and future analyzers.

## Assumptions

- The first complete implementation is Go; other adapters follow the same contract.
- Optional environment/tool-assisted resolution is disabled by default and cannot change the read-only safety promise.

## Current delivery frontier

The Go analyzer/model and the shared viewer/export path are complete for the first slice. The readiness-reviewed [Python analysis capability](python-analysis/prd.md) is also implemented through the [Python repository to visible architecture view slice](python-analysis/implementation-slice.md) and issues 017–019. The Python adapter registers through the existing composition root and common `analysis.Analyzer` contract; the host, canonical model, layout, viewer, and export boundaries remain language-neutral. TypeScript is the next language in the agreed sequence.
