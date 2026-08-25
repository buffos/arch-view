# 004 — Evidence drill-down and source inspection

Execution type: AFK
Review gate: visual-review
Status: ready-for-agent

## Parent PRD

docs/architecture/explore-architecture/prd.md

## What to build

Complete the first investigation workflow on top of the local overview.

- Add hierarchy drill-down, breadcrumbs, back navigation, search, selection, zoom, pan, and fit behavior.
- Show module metadata, edge direction, relationship evidence, source locations, external/unresolved references, diagnostics, confidence, cycles, and layers.
- Provide a read-only source excerpt endpoint and panel with project-root containment checks.
- Keep selections and evidence IDs stable across scene updates and prevent stale evidence after reanalysis.
- Preserve an accessible list/details path with keyboard navigation and equivalent facts to the graphic view.

## Acceptance criteria

- [ ] A user can drill into a hierarchy group, inspect its modules and relationships, and return with breadcrumbs or back navigation.
- [ ] Search and selection identify a module or relationship and reveal its source evidence and source location.
- [ ] Source excerpts are read-only, line-aware, confined to the analyzed project root, and reject traversal or cross-root requests.
- [ ] Cycles, unresolved/external references, diagnostics, confidence, and layer information are visible and explainable in both graphic and list/details modes.
- [ ] Reanalysis replaces the active model/evidence revision safely; failed reanalysis leaves the prior revision active and never presents stale evidence as current.
- [ ] Keyboard navigation and accessible labels/descriptions expose the same architecture facts as the graphic view.
- [ ] A visual review confirms that navigation, evidence, cycle states, and source inspection remain legible and coherent.

## Artifact sync required

- Application PRD: none — this realizes the already specified investigation workflow.
- Application architecture summary: none — no new boundary or renderer decision is introduced.
- Owning capability node/artifacts: required: .okf/capabilities/explore-architecture.md; docs/architecture/explore-architecture/orchestration-status.md; docs/architecture/analyze-source/go-analysis/implementation-slice.md.
- Issue registry: required; node issues reference required: .okf/capabilities/explore-architecture.md.
- Reason/no-impact decision: delivery truth is being added; product and architecture truth remain unchanged.

## Blocked by

docs/agents/issues/pending/003-local-web-top-level-architecture-view.md

## User stories addressed

- US-EX-001
- US-EX-002
- US-GM-002

## Contract and scenario trace

- Contract: docs/architecture/explore-architecture/canonical-api-cli-contract.md
- Scenarios: SC-EX-002, SC-EX-003, SC-EX-004, SC-EX-005, SC-EX-006, SC-EX-007, SC-EX-008
