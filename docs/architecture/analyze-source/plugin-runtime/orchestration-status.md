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
- Issues 039–043 complete the approved multi-analyzer project-orchestration
  batch's deterministic root discovery/job planning, bounded execution/
  lifecycle, namespaced aggregation/status, CLI/HTTP exposure, and cached
  viewer scope selection. The required visual-review gate is approved.

## Next step and artifact impact

Issues 030–038 are complete. Issues 039–043 are verified and archived
multi-analyzer delivery records, including the approved visual review. The
current v1 pilot remains an explicitly supplied, script-based external Python
process with one analyzer selected per run, while the compiled-distribution
child is now the verified packaged runtime path for release execution. The
parent capability remains `specified`; the separate project assignment/view
child owns the readiness-reviewed invocation-root source-scope and assignment
frontier, while the multi-analyzer batch consumes its resolved policy. Issues
044–046 are verified and archived delivery records for that child; issue 047
remains active as `awaiting-human-review` for its required visual gate.

- [Compiled external analyzer distribution](compiled-external-analyzer-distribution/discovery-notes.md)
- [Multi-analyzer project orchestration](multi-analyzer-orchestration/discovery-notes.md)
- [Project analyzer assignments and view selection](project-analyzer-assignments/discovery-notes.md)

Product and architecture synthesis now record the target compiled-binary,
multi-job, assignment/view, and source-scope boundaries. Delivery truth includes
completed issues 034–038 for the compiled-distribution child, completed issues
039–043 for multi-analyzer orchestration, completed issues 044–046 for the
project assignment/configuration child, and active issue 047 for its viewer
review gate. That child remains `specified`; approved
issue dependencies, platform deferrals, and user-confirmed packaged-artifact
trust policy are recorded in the planning map and log.
