# Analyzer plugin runtime orchestration status

## State

- Planning state: `specified`.
- State transition: `bounded -> specified` on 2026-08-25.
- Exact-spec set is complete and readiness-reviewed.

## Artifact inventory

- [Discovery notes](discovery-notes.md)
- [Requirements gap analysis](requirements-gap-analysis.md)
- [PRD](prd.md)
- [Domain glossary](domain-glossary.md)
- [Canonical domain model](canonical-domain-model.md)
- [Canonical use cases](canonical-use-cases.md)
- [Canonical API/CLI contract](canonical-api-cli-contract.md)
- [Acceptance scenarios](acceptance-scenarios.md)
- [Readiness review](readiness-review.md)
- [External protocol schema](external-protocol-v1.schema.json)
- [External plugin descriptor schema](external-plugin-descriptor-v1.schema.json)
- [External analyzer implementation slice](implementation-slice.md)

## Current delivery slice

- Issue 001 completed the in-process host and Go project-selection boundary.
- It is part of [the first Go implementation slice](../go-analysis/implementation-slice.md).
- Issue 030 completed the published external protocol/descriptor schemas,
  bounded typed frame codec, stateful conformance validator, and test-only
  subprocess fixture in [the external implementation slice](implementation-slice.md).
- Issues 031–033 completed the opt-in external process boundary, external
  Python parity pilot, and explicit CLI/shared model-viewer-export path in
  [the external implementation slice](implementation-slice.md).

## Next step and artifact impact

Issues 030–033 are complete. Product and architecture truth remain unchanged;
delivery truth now records the verified process host, external Python parity,
and public shared-path integration. The capability remains `specified` because
no subsequent external plugin-runtime frontier has been selected.
