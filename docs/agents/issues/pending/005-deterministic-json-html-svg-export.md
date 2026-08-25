# 005 — Deterministic JSON, HTML, and SVG export

Execution type: AFK
Review gate: visual-review
Status: ready-for-agent

## Parent PRD

docs/architecture/export-and-automate/prd.md

## What to build

Implement repeatable artifacts from the same validated model and renderer-neutral view used by the local viewer.

- Implement arch-view export for JSON, HTML, and SVG and wire the analyze-to-export path.
- Preserve the arch-view.model/v1 envelope, analysis status, diagnostics, evidence IDs, cycles, layers, provenance, and stable ordering.
- Produce self-contained HTML with embedded model/view data and no network dependency.
- Produce script-free accessible SVG with stable module/relationship attributes, titles, descriptions, cycle/diagnostic styling, and deterministic geometry.
- Refuse existing outputs unless overwrite is explicit, write atomically, and return the specified status and exit codes.
- Reject source embedding and invalid models clearly; do not parse source in the exporter.

## Acceptance criteria

- [ ] JSON export is versioned, schema-valid, byte-stable for identical input/options, and preserves partial status and diagnostics.
- [ ] HTML export is self-contained, opens without network access, and exposes overview, hierarchy, search, evidence/details, cycle/diagnostic states, and accessible list/details mode.
- [ ] SVG export is scalable, script-free, accessible, deterministic, and includes stable module/relationship identifiers and recorded layout provenance.
- [ ] Repeated exports from identical models and options produce identical bytes and equivalent semantics across CLI and any HTTP entrypoint.
- [ ] Existing output is protected unless overwrite is supplied; writes are atomic and failed writes do not leave a misleading completed artifact.
- [ ] Invalid models, unsupported source embedding, invalid invocation, render/write failures, and cancellation map to the specified exit/status behavior.
- [ ] A visual review confirms HTML/SVG parity with the local viewer for overview, relationships, cycles, diagnostics, and labels.

## Artifact sync required

- Application PRD: none — JSON, HTML, and SVG are already in the specified MVP.
- Application architecture summary: none — exporters continue to consume the existing model/view contract.
- Owning capability node/artifacts: required: .okf/capabilities/export-and-automate.md; .okf/capabilities/explore-architecture.md; docs/architecture/export-and-automate/orchestration-status.md; docs/architecture/explore-architecture/orchestration-status.md; docs/architecture/analyze-source/go-analysis/implementation-slice.md.
- Issue registry: required; node issues reference required: .okf/capabilities/export-and-automate.md; .okf/capabilities/explore-architecture.md.
- Reason/no-impact decision: delivery truth is being added; no product or architecture decision changes, and source contents remain excluded from v1 artifacts.

## Blocked by

docs/agents/issues/pending/003-local-web-top-level-architecture-view.md

## User stories addressed

- US-EXP-001
- US-EXP-002
- US-EX-001

## Contract and scenario trace

- Contract: docs/architecture/export-and-automate/canonical-api-cli-contract.md; docs/architecture/explore-architecture/canonical-api-cli-contract.md
- Scenarios: SC-EXPT-001, SC-EXPT-002, SC-EXPT-003, SC-EXPT-004, SC-EXPT-005, SC-EXPT-006, SC-EXPT-007, SC-EXPT-008
