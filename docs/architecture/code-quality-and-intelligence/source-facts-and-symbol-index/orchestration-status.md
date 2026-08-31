# Source facts and symbol index orchestration status

## State

- Planning state: `implemented` after verified delivery and approved visual
  review.
- The exact-spec pipeline is complete and readiness-reviewed.
- Approved delivery issues 048–052 are linked from the capability node; issues
  048–052 are verified, archived, and approved, including issue 052's required
  final visual inspection.
- The capability's scoped implementation and artifact synchronization are
  complete. The parent roll-up is now `implemented`; its deterministic-quality
  and live-analysis/MCP children are also implemented.

## Evidence and ownership

- Existing analyzers enumerate source scope and return architecture/source
  evidence through the common analysis result.
- The shared syntax provider supplies parse trees and source ranges; registered
  source-fact extractors own language declarations, documentation, visibility,
  callable body spans, provider-owned source metrics, and semantic uncertainty.
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
records, separate documentation candidates, hash-linked spans, callable body
spans, versioned provider metrics, reusable provenance, registered extractors,
open vocabularies, typed extensions, explicit coverage, deterministic
IDs/digests, scope isolation, and compact projections.

## Artifact impact

- **Capability:** bounded source-facts intent is now an exact implementation
  contract without expanding into quality rules or live MCP.
- **Product:** files, declarations, documentation, line counts, callable body
  spans, provider metrics, and structural queries now have a single source of
  truth; the rendered viewer approval is recorded and the human-oriented
  inspection journey is complete.
- **Architecture:** old model/source-reference consumers remain compatible;
  quality and live capabilities consume this optional attachment.
- **Delivery:** issues 048–052 are archived with verification evidence and
  issue 052's rendered-viewer gate is approved; the source-index capability is
  now `implemented`.

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
5. Issue 052 renders source facts in module inspection and its required visual
   review gate is approved.

The downstream quality delivery adds the Go `source:callable.metrics`
provider facts used by issues 054–055. This does not move quality policy into
the source-index child: thresholds and findings remain owned by deterministic
quality checks.

The deterministic quality and live/MCP child nodes are implemented in their
own delivery batches and are not part of this source-index delivery scope.
