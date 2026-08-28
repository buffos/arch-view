# Project analyzer assignments and view selection orchestration status

## State

- Planning state: `specified`.
- The exact-spec pipeline is complete; implementation issues have not yet been created.

## Evidence

- The current project configuration and viewer settings are implemented for
  layout presentation only.
- The analyzer host supports explicit analyzer IDs and external descriptors,
  but not persisted folder assignments or a multi-scope application view.

## Artifact sync

- **Topology:** This child is linked under `Analyzer plugin runtime` and names
  [Explore and inspect architecture](/capabilities/explore-architecture.md) as
  its viewer consumer.
- **Capability:** Discovery notes record the target assignment and selection
  behavior, configuration ownership, precedence, cache policy, and the
  combined/per-scope viewer boundary without fixing the exact v2 schema.
- **Product and architecture:** The application synthesis records the future
  analysis-configuration and viewer-scope boundary.
- **Delivery:** No issues were created or changed in this graph update.

## Exact-spec inventory

- Requirements gap analysis
- Domain glossary
- Capability PRD
- Canonical domain model
- Canonical use-case model
- Canonical API/CLI contract
- Acceptance scenarios
- Architecture readiness review

All artifacts are linked from the capability node and agree on v1/v2
configuration compatibility, path safety, precedence, scoped diagnostics,
cache keys/invalidation, API behavior, and the `All`/individual scope UI.

## Readiness decision

READY FOR ARCHITECTURE IMPLEMENTATION. The next step is reference-doc issue
slicing after the application synthesis gate is checked for all four specified
nodes.

## Artifact impact

- Capability truth: updated with the complete exact-spec set.
- Product truth: assignment and scope-selection workflows are synchronized.
- Architecture truth: configuration, cache, runtime, and viewer boundaries are current.
- Delivery truth: intentionally unchanged; no issue slicing in this run.
