# Project analyzer assignments and view selection orchestration status

## State

- Planning state: `implemented`.
- The exact-spec pipeline is complete; the source-scope filter extension is
  incorporated in the v2 configuration contract. Approved implementation
  issues 044–047 cover configuration loading/validation, planner integration,
  selective session caching, and the configured viewer journey. All four issues
  are implemented, verified, archived, and complete, including issue 047's
  required visual review.

## Evidence

- `internal/analysis/config` owns strict v1/v2 decoding, nearest-file
  discovery, path/glob safety, manifest-aware options, diagnostics, and
  layout-preserving v2 round trips.
- The planner and public analysis paths consume one normalized assignment and
  source-scope policy, including deterministic CLI/deepest-assignment
  precedence and retained unavailable assignments.
- The session cache reuses immutable per-job snapshots and reports selective
  invalidation metadata through aggregate scope summaries and reanalysis.
- Issue 047's configured mixed-language fixture and viewer metadata wiring are
  automated-test complete, and the declared visual review was approved by the
  user.

## Artifact sync

- **Topology:** This child is linked under `Analyzer plugin runtime` and names
  [Explore and inspect architecture](/capabilities/explore-architecture.md) as
  its viewer consumer.
- **Capability:** Discovery notes record the target assignment and selection
  behavior, configuration ownership, source-scope glob policy, precedence,
  cache policy, and the combined/per-scope viewer boundary.
- **Product and architecture:** The application synthesis records the
  analysis-configuration and viewer-scope boundary; issues 044–046 implement
  the configuration, planning, and cache portions without changing that
  boundary.
- **Delivery:** The approved multi-analyzer issues 039–043 and the assignment
  issues 044–047 are verified and archived. The assignment/view capability is
  now complete and implemented.

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

IMPLEMENTATION VERIFIED AND VISUALLY APPROVED. Issues 044–047 consume the
resolved policy at their configuration, planning, cache, and viewer seams.
The child capability is now `implemented`.

## Artifact impact

- Capability truth: updated with the complete exact-spec set, including the
  v2 source-scope filter policy.
- Product truth: assignment and scope-selection workflows are synchronized.
- Architecture truth: configuration, source filtering, cache, runtime, and
  viewer boundaries are current.
- Delivery truth: issues 044–047 are verified, archived, and complete. The
  node is `implemented` after the approved visual gate.
