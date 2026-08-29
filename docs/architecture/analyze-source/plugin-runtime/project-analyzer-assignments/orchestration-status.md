# Project analyzer assignments and view selection orchestration status

## State

- Planning state: `specified`.
- The exact-spec pipeline is complete; the source-scope filter extension is
  incorporated in the v2 configuration contract. Implementation issues have
  not yet been created for this separate assignment/configuration capability.

## Evidence

- The current project configuration and viewer settings are implemented for
  layout presentation only.
- The analyzer host supports explicit analyzer IDs and external descriptors,
  but not persisted folder assignments or a multi-scope application view.
- The analyzers expose some implementation-specific exclusion options, but no
  persisted cross-analyzer include/exclude policy is implemented.

## Artifact sync

- **Topology:** This child is linked under `Analyzer plugin runtime` and names
  [Explore and inspect architecture](/capabilities/explore-architecture.md) as
  its viewer consumer.
- **Capability:** Discovery notes record the target assignment and selection
  behavior, configuration ownership, source-scope glob policy, precedence,
  cache policy, and the combined/per-scope viewer boundary.
- **Product and architecture:** The application synthesis records the future
  analysis-configuration and viewer-scope boundary.
- **Delivery:** No new issue is created for this specification refresh. The
  approved multi-analyzer issues 039–042 are verified and issue 043 remains an
  active visual-review consumer of the resolved source-scope policy;
  assignment/configuration implementation remains a separate future delivery
  slice.

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
configuration compatibility, assignment and glob path safety, invocation-root
anchoring, filter precedence, scoped diagnostics, cache keys/invalidation, API
behavior, and the `All`/individual scope UI.

## Readiness decision

READY FOR ARCHITECTURE IMPLEMENTATION. The source-scope extension is ready for
the later assignment/configuration delivery slice. Issues 039–043 consume the
resolved policy at their defined orchestration seams; assignment configuration
implementation remains separate.

## Artifact impact

- Capability truth: updated with the complete exact-spec set, including the
  v2 source-scope filter policy.
- Product truth: assignment and scope-selection workflows are synchronized.
- Architecture truth: configuration, source filtering, cache, runtime, and
  viewer boundaries are current.
- Delivery truth: no new assignment issue was created; issues 039–042 are
  verified and issue 043 remains active, all consuming the resolved filter
  policy where applicable.
