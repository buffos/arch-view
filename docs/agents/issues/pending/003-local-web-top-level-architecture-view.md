# 003 — Local web top-level architecture view

Execution type: AFK
Review gate: visual-review
Status: ready-for-agent

## Parent PRD

docs/architecture/explore-architecture/prd.md

## What to build

Build the first visible local web experience over a validated canonical model.

- Implement arch-view open for a model file and for a Go project through the existing analysis path.
- Serve a local browser application with the renderer-neutral scene contract.
- Render the top-level hierarchy, modules, directed relationships, layer labels, cycle indicators, diagnostics, confidence states, and aggregate counts.
- Start with an overview that remains understandable for a small representative Go repository and has a list/details fallback.
- Keep viewer session state separate from the canonical model and keep source access read-only.

Drill-down, source excerpts, and richer evidence interactions are completed in issue 004. Durable artifact generation is completed in issue 005.

## Acceptance criteria

- [ ] arch-view open --model serves a local browser session and loads the canonical model without source parsing.
- [ ] arch-view open --project runs the Go path from issue 002 and opens the resulting top-level architecture view.
- [ ] The top-level scene contains hierarchy-aware visible nodes, directed relationship indicators, layers, cycles, diagnostics, confidence, and stable IDs.
- [ ] The first view is aggregated at the top level and does not require a renderer to invent semantic relationships or hierarchy.
- [ ] A list/details mode exposes the same visible facts as the graphic view and is keyboard reachable.
- [ ] The local host is read-only, does not execute target code, and does not expose source paths outside the analyzed root.
- [ ] A visual review confirms legible labels, edge direction, cycle/diagnostic styling, and a useful result on a representative Go repository.

## Artifact sync required

- Application PRD: none — this is the specified first local viewer journey.
- Application architecture summary: none — the renderer-neutral scene and local web boundary are already defined.
- Owning capability node/artifacts: required: .okf/capabilities/explore-architecture.md; docs/architecture/explore-architecture/orchestration-status.md; docs/architecture/analyze-source/go-analysis/implementation-slice.md.
- Issue registry: required; node issues reference required: .okf/capabilities/explore-architecture.md.
- Reason/no-impact decision: delivery truth is being added; the existing model/view boundary and visual decisions remain authoritative.

## Blocked by

—

## User stories addressed

- US-EX-001
- US-GM-001
- US-GM-002

## Contract and scenario trace

- Contract: docs/architecture/explore-architecture/canonical-api-cli-contract.md; docs/architecture/generate-models/canonical-api-cli-contract.md
- Scenarios: SC-EX-001, SC-EX-005, SC-EX-006, SC-EX-007
