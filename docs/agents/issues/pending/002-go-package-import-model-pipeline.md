# 002 — Go package/import analysis and canonical model pipeline

Execution type: AFK
Review gate: none
Status: ready-for-agent

## Parent PRD

docs/architecture/analyze-source/go-analysis/prd.md

## What to build

Implement the Go static analyzer and connect its observations to the canonical architecture model.

- Discover eligible Go packages and source files inside the selected module.
- Extract static imports with source file, line, and column evidence.
- Resolve project-local imports to stable Go package IDs.
- Retain standard-library, third-party, missing, cgo, and conditional targets as references or diagnostics.
- Honor module, build-tag, include-tests, include-generated, and exclusion options without executing the target repository.
- Normalize the result into the versioned arch-view.model/v1 envelope, validate relationship integrity, preserve evidence, and report complete or partial status.
- Derive deterministic cycles, feedback relationships, hierarchy projections, and dependency layers without removing canonical relationships.

## Acceptance criteria

- [ ] A representative Go module produces deterministic package nodes with stable IDs based on module path and relative import path.
- [ ] Local imports become directed depends_on relationships and retain source evidence; imports outside the module do not become invented local modules.
- [ ] Default exclusions and explicit options change the analyzed scope predictably, including tests, generated files, vendor, external, and build-conditional files.
- [ ] Unresolved or non-local imports produce references and diagnostics with partial status where appropriate, rather than fabricated dependencies.
- [ ] The model pipeline emits and validates arch-view.model/v1 with sorted collections, normalized paths, preserved evidence, and no implicit timestamps.
- [ ] Self-cycles, strongly connected groups, feedback relationships, and layers are derived deterministically while canonical edges remain intact.
- [ ] The pipeline provides a headless command path that a later viewer can consume without knowing Go syntax.

## Artifact sync required

- Application PRD: none — this implements the already specified Go MVP and neutral-model pipeline.
- Application architecture summary: none — the analyzer-to-model boundary and ownership remain unchanged.
- Owning capability node/artifacts: required: .okf/capabilities/analyze-source.md; .okf/capabilities/analyze-source/go-analysis.md; .okf/capabilities/generate-models.md; docs/architecture/analyze-source/go-analysis/orchestration-status.md; docs/architecture/generate-models/orchestration-status.md; docs/architecture/analyze-source/go-analysis/implementation-slice.md.
- Issue registry: required; node issues reference required: .okf/capabilities/analyze-source.md; .okf/capabilities/analyze-source/go-analysis.md; .okf/capabilities/generate-models.md.
- Reason/no-impact decision: delivery truth is being added; no product or architecture decision changes, and the external reference folder remains untouched.

## Blocked by

None.

## User stories addressed

- US-GO-001
- US-GO-002
- US-GO-003
- US-GM-001
- US-GM-002

## Contract and scenario trace

- Contract: docs/architecture/analyze-source/go-analysis/canonical-api-cli-contract.md; docs/architecture/generate-models/canonical-api-cli-contract.md
- Scenarios: SC-GO-001, SC-GO-002, SC-GO-003, SC-GO-004, SC-GO-005, SC-GM-001, SC-GM-002, SC-GM-003, SC-GM-004, SC-GM-005, SC-GM-006, SC-GM-007, SC-GM-008
