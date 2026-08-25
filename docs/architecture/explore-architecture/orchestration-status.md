# Explore and inspect architecture orchestration status

## State

- Planning state: `specified`.
- State transition: `bounded -> specified` on 2026-08-25.
- Topology: confirmed as the interactive investigation capability.
- Next route: implement or slice the specified local web viewer after application synthesis validation.

## Confirmed boundary

The capability consumes the canonical model and graph/view preparation outputs and provides a renderer-neutral interactive view plus a local web presentation. It owns viewer session state, navigation, selection, evidence/source inspection, visual states, progressive disclosure, and accessibility. It does not parse source, discover dependencies, own canonical graph algorithms, or define export formats.

## Confirmed decisions

- The first surface is a local web application with a Go host and browser presentation.
- A renderer-neutral view/scene contract separates interaction and visual technology from the model; SVG/HTML is first, Canvas/WebGL can follow for scale.
- The workflow includes overview, hierarchy drill-down, breadcrumbs/back, zoom/pan/fit, search, selection, edge highlighting, cycle indicators, and evidence/source inspection.
- Large graphs use hierarchy aggregation and level-of-detail with details on demand.
- Cycles, unresolved/external dependencies, diagnostics, and confidence remain visible.
- Source inspection is read-only with path and locations; no target code is executed or edited.
- Keyboard navigation, accessible labels/contrast, and a list/details path complement the graph.
- Reanalysis replaces stale model/evidence state and preserves navigation only when safe and explainable.

## Artifact plan

The exact capability set is complete: [PRD](prd.md), [glossary](domain-glossary.md), [canonical domain model](canonical-domain-model.md), [canonical use cases](canonical-use-cases.md), [viewer contract](canonical-api-cli-contract.md), [acceptance scenarios](acceptance-scenarios.md), and [readiness review](readiness-review.md), alongside discovery and gap artifacts.

## Current and target truth

- Observed in reference: the viewer supports layers, dependency indicators, drill-down, back navigation, cycle display, scrolling, zooming, reanalysis, and a source window.
- User-confirmed target: the Go product should retain the useful investigation workflow while allowing language-neutral source evidence.
- Required follow-up: implement the local web host, renderer-neutral scene contract, accessible interaction modes, and source-root safety checks.

## Current delivery slice

- Issue 003 delivers the first visible local top-level view.
- Issue 004 adds evidence, drill-down, source safety, and accessible inspection.
- Issue 005 reuses the view contract for visual export parity.
- All three are part of [the first Go implementation slice](../analyze-source/go-analysis/implementation-slice.md).

## Artifact sync

- Topology: updated in `.okf/`.
- Capability truth: reflected in [discovery notes](discovery-notes.md) and [requirements gap analysis](requirements-gap-analysis.md).
- Exact specification: complete in the linked PRD, glossary, domain model, use cases, contract, scenarios, and readiness review.
- Product truth: reflected in [the application PRD](../../prd.md).
- Architecture truth: reflected in [the application architecture summary](../application-architecture-summary.md).
- Delivery truth: updated with issues 003, 004, and 005.
