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
- Issue 034 completed the shared compiled-plugin runner and five compiled
  analyzer entrypoints while preserving the existing in-process analyzers.
- Issue 035 completed deterministic package assembly, descriptor/index
  generation, integrity digests, and explicit analyzer/release build targets.
- Issues 036–038 completed trusted package verification, packaged-by-default
  runtime selection, explicit fallback/override handling, runtime provenance,
  five-analyzer parity, deterministic assembly checks, and release behavior.

## Next step and artifact impact

Issues 030–038 are complete. The current v1 pilot remains an explicitly
supplied, script-based external Python process with one analyzer selected per
run, while the compiled-distribution child is now the verified packaged
runtime path for release execution. The parent capability remains `specified`
because its separate multi-analyzer and project assignment/view children are
not implemented.
The multi-analyzer and project assignment/view children remain `specified`
future frontiers with readiness-reviewed exact artifacts, ready for later
delivery issue slicing.

- [Compiled external analyzer distribution](compiled-external-analyzer-distribution/discovery-notes.md)
- [Multi-analyzer project orchestration](multi-analyzer-orchestration/discovery-notes.md)
- [Project analyzer assignments and view selection](project-analyzer-assignments/discovery-notes.md)

Product and architecture synthesis now record the target compiled-binary,
multi-job, and assignment/view boundaries. Delivery truth now includes
completed issues 034–038 for the compiled-distribution child; no delivery
issues have yet been created for the project assignment/view children. The
child state transition to `implemented`, platform deferrals, and
user-confirmed packaged-artifact trust policy are recorded in the planning map
and log.
