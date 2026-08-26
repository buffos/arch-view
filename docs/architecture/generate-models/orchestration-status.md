# Generate architecture models orchestration status

## State

- Planning state: `specified`.
- State transition: `bounded -> specified` on 2026-08-25.
- Topology: confirmed as the language-neutral model and graph-processing capability.
- Next route: implement or slice the specified canonical model after application synthesis validation.

## Confirmed boundary

The capability receives validated analyzer observations and returns a deterministic canonical model plus derived graph-analysis projections. It owns identity, hierarchy, relationship normalization, provenance, aggregation, validation, cycle detection, feedback-edge derivation, and layer assignment. It does not parse source code, manage interactive state, render graphics, or package exports.

## Confirmed decisions

- Hierarchy is an explicit structured path on each module and is separate from dependency edges.
- Modules have stable opaque project-scoped IDs, display labels, language metadata, tags, and many source references.
- Relationships are typed with explicit direction; `depends_on` is the first type and future relation types remain open-ended.
- External, standard-library, unresolved, and dynamic targets are retained as references or diagnostics without becoming default local module nodes.
- Evidence is merged and deduplicated through module and relationship aggregation.
- Canonical relationships are never removed to make layout easier. Cycles, feedback edges, and layers are derived outputs with provenance.
- Language-specific abstraction concepts are metadata/tags, not core model variants.
- Validation, deduplication, and sorting are host responsibilities and produce deterministic output.

## Artifact plan

The exact capability set is complete: [PRD](prd.md), [glossary](domain-glossary.md), [canonical domain model](canonical-domain-model.md), [canonical use cases](canonical-use-cases.md), [model contract](canonical-api-cli-contract.md), [acceptance scenarios](acceptance-scenarios.md), and [readiness review](readiness-review.md), alongside discovery and gap artifacts.

## Current and target truth

- Observed in reference: graph construction, cycle handling, hierarchy projection, and layered layout are separate from Clojure source extraction but still consume namespace-shaped strings.
- User-confirmed target: analyzers for multiple languages must feed one shared model.
- Required follow-up: implement schema validation, deterministic normalization, cycle/layer projections, and hierarchy aggregation according to the exact contract.

## Current delivery slice

- Issue 002 consumed Go observations and implemented the first canonical model pipeline, including normalization, validation, graph derivations, hierarchy projection, and headless CLI access.
- It is part of [the first Go implementation slice](../analyze-source/go-analysis/implementation-slice.md).

## Delivery progress

Issue 002 is complete. The first-slice viewer and export consumers in issues 003 through 006 are complete; issue 007 is a viewer-owned presentation extension and does not change the canonical model boundary.

## Artifact sync

- Topology: updated in `.okf/`.
- Capability truth: reflected in [discovery notes](discovery-notes.md) and [requirements gap analysis](requirements-gap-analysis.md).
- Exact specification: complete in the linked PRD, glossary, domain model, use cases, contract, scenarios, and readiness review.
- Product truth: reflected in [the application PRD](../../prd.md).
- Architecture truth: reflected in [the application architecture summary](../application-architecture-summary.md).
- Delivery truth: issue 002 is archived as complete; the active queue begins with issue 003.
