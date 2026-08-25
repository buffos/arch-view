# Generate architecture models PRD

## Purpose

Convert analyzer observations into a deterministic, language-neutral architecture model and derived graph projections.

## Actors and inputs

The analyzer host supplies observations; the model service validates and normalizes them; viewers and exporters consume the resulting model. Inputs include modules, typed relationships, non-local references, source evidence, metadata, tags, confidence, and diagnostics.

## User stories

- US-GM-001 — As a viewer or exporter, I can consume one stable language-neutral model regardless of analyzer language.
- US-GM-002 — As a maintainer, I can inspect cycles, layers, relationships, and evidence without losing canonical dependency edges.

## Scope

Own stable module identity, structured hierarchy, evidence aggregation, relationship normalization, model validation, cycle detection, feedback-edge derivation, layer assignment, and hierarchical projection. Do not parse source, render graphics, own viewer state, or package exports.

## Rules

- Hierarchy is `string[]` structural data, never punctuation-derived from IDs.
- Modules are stable opaque project-scoped nodes and may have many source references.
- Relationships have explicit `from`/`to` semantics and a typed `type`; `depends_on` is first.
- Non-local targets remain references or diagnostics, not default local module nodes.
- Canonical relationships remain intact even when cycles require derived feedback edges for layering.
- Derived layers record algorithm/provenance and are not module identity.
- Aggregation merges evidence and contributor IDs deterministically.
- Conflicting metadata is retained with a diagnostic rather than silently discarded.

## Functional requirements

| ID | Requirement |
|---|---|
| GM-FR-001 | Validate and normalize analyzer observations into the canonical model. |
| GM-FR-002 | Deduplicate and deterministically order modules, references, relationships, evidence, tags, and diagnostics. |
| GM-FR-003 | Preserve explicit hierarchy and source traceability through aggregation. |
| GM-FR-004 | Represent typed relationships and external/unresolved targets without invented nodes. |
| GM-FR-005 | Detect self-cycles and strongly connected cycle groups. |
| GM-FR-006 | Derive feedback relations and dependency layers without mutating canonical relations. |
| GM-FR-007 | Produce hierarchical projections with aggregated counts and contributor traceability. |
| GM-FR-008 | Return partial model status plus diagnostics for recoverable invalid observations. |

## Non-functional requirements

Deterministic, language-neutral, versionable, evidence-preserving, renderer-independent, and suitable for both interactive and headless consumers.

## Acceptance summary

Detailed behavior is in [acceptance scenarios](acceptance-scenarios.md). Exact field shapes are normative in the [canonical contract](canonical-api-cli-contract.md).
