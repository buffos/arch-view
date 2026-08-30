# Source facts and symbol index orchestration status

## State

- Planning state: `specified` after the bounded-to-specified transition.
- The exact-spec pipeline is complete and readiness-reviewed.
- Approved delivery issues 048–052 are linked from the capability node; issues
  048–051 are verified and archived, and issue 052 is implemented pending its
  required final visual inspection.
- The capability remains `specified` until the 052 visual gate and the
  remaining batch artifact-synchronization obligations are approved.

## Evidence and ownership

- Existing analyzers enumerate source scope and return architecture/source
  evidence through the common analysis result.
- The shared syntax provider supplies parse trees and source ranges; registered
  source-fact extractors own language declarations, documentation, visibility,
  and semantic uncertainty.
- The core source-index service owns validation, opaque IDs, deduplication,
  coverage, canonical ordering, and snapshot digest.
- Existing multi-analyzer orchestration supplies independent scope identity and
  source-policy fingerprints. A combined source index retains authoritative
  per-scope snapshots and may expose a derived projection.

## Exact-spec inventory

- Requirements gap analysis
- Domain glossary
- Capability PRD
- Canonical domain model
- Canonical use-case model
- Canonical API/CLI contract
- Acceptance scenarios
- Architecture readiness review

All artifacts agree on the optional `SourceIndex` sibling, file and symbol
records, separate documentation candidates, hash-linked spans, reusable
provenance, registered extractors, open vocabularies, typed extensions,
explicit coverage, deterministic IDs/digests, scope isolation, and compact
projections.

## Artifact impact

- **Capability:** bounded source-facts intent is now an exact implementation
  contract without expanding into quality rules or live MCP.
- **Product:** files, declarations, documentation, line counts, and structural
  queries now have a single source of truth; the rendered viewer approval is
  still pending.
- **Architecture:** old model/source-reference consumers remain compatible;
  quality and live capabilities consume this optional attachment.
- **Delivery:** issues 048–051 are archived with verification evidence. Issue
  052 remains the active `awaiting-human-review` gate for the rendered viewer;
  the source-index capability state has intentionally not advanced.

## Readiness decision

READY FOR ARCHITECTURE IMPLEMENTATION. The approved delivery workflow is
dependency-ordered as follows:

1. Issue 048 established the versioned source-index attachment and deterministic
   file facts.
2. Issue 049 registered the extractor boundary and added Go declarations and
   documentation.
3. Issue 050 carried authoritative snapshots through canonical and aggregate
   analysis without cross-scope inference.
4. Issue 051 added structural queries and bounded source evidence.
5. Issue 052 renders source facts in module inspection and is awaiting its
   required visual review gate.

The deterministic quality and live/MCP child nodes remain specified and are not
included in this delivery batch.
