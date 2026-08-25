# Export and automate orchestration status

## State

- Planning state: `specified`.
- State transition: `bounded -> specified` on 2026-08-25.
- Topology: confirmed as the headless and repeatable-workflow capability.
- Next route: implement or slice the specified JSON/HTML/SVG exporter after application synthesis validation.

## Confirmed boundary

The capability owns CLI export options, versioned machine-readable output, deterministic visual artifacts, headless status, and repeatable automation behavior. It consumes canonical model and renderer-neutral view inputs. It does not parse source, implement analyzers, or own interactive viewer state.

## Confirmed decisions

- Versioned JSON is the canonical durable interchange format and is distinct from the future NDJSON analyzer protocol.
- JSON includes the neutral model, evidence references, diagnostics, cycles, layers, and analysis status.
- Stable IDs/order, normalized paths, explicit versions, and layout provenance make output deterministic; implicit wall-clock timestamps are excluded.
- Self-contained interactive HTML and scalable SVG are the first visual artifacts. Raster output is a later adapter.
- Source content is not embedded by default; exports retain paths and locations and may add explicit future embedding/redaction controls.
- Partial results and warnings remain visible. Fatal configuration/readability/analysis errors produce non-zero exit codes.
- CI starts with reproducible artifact generation and documented exit behavior, not automatic architectural approval.

## Artifact plan

The exact capability set is complete: [PRD](prd.md), [glossary](domain-glossary.md), [canonical domain model](canonical-domain-model.md), [canonical use cases](canonical-use-cases.md), [export contract](canonical-api-cli-contract.md), [acceptance scenarios](acceptance-scenarios.md), and [readiness review](readiness-review.md), alongside discovery and gap artifacts.

## Current and target truth

- Observed in reference: the tool supports headless analysis and writes an EDN architecture representation.
- User-confirmed target: the Go product should support repeatable machine-readable output and future automation.
- Required follow-up: implement versioned JSON, self-contained HTML, deterministic accessible SVG, atomic writes, and documented CI exit behavior.

## Current delivery slice

- Issue 005 implements deterministic JSON, HTML, and SVG artifacts from the validated model/view contract.
- It is part of [the first Go implementation slice](../analyze-source/go-analysis/implementation-slice.md).

## Artifact sync

- Topology: updated in `.okf/`.
- Capability truth: reflected in [discovery notes](discovery-notes.md) and [requirements gap analysis](requirements-gap-analysis.md).
- Exact specification: complete in the linked PRD, glossary, domain model, use cases, contract, scenarios, and readiness review.
- Product truth: reflected in [the application PRD](../../prd.md).
- Architecture truth: reflected in [the application architecture summary](../application-architecture-summary.md).
- Delivery truth: updated with issue 005.
