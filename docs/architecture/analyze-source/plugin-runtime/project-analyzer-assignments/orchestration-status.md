# Project analyzer assignments and view selection orchestration status

## State

- Planning state: `specified`.
- The exact-spec pipeline is complete; the source-scope filter extension is
  incorporated in the v2 configuration contract. Approved implementation
  issues 044–047 cover configuration loading/validation, planner integration,
  selective session caching, and the configured viewer journey. Issues 044–046
  are implemented, verified, and archived; issue 047 is awaiting its required
  visual review.

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
  automated-test complete; the browser surface still needs the declared visual
  review.

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
- **Delivery:** The approved multi-analyzer issues 039–043 are verified,
  archived, and consume the resolved source-scope policy. Issues 044–046 are
  verified and archived; issue 047 remains registered as
  `awaiting-human-review` with the visual-review gate.

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

IMPLEMENTATION SLICES VERIFIED; VISUAL REVIEW PENDING. Issues 044–046 consume
the resolved policy at their defined configuration, planning, and cache seams.
Issue 047 has completed its automated implementation and is waiting for the
declared visual review before this child can move from `specified` to
`implemented`.

## Artifact impact

- Capability truth: updated with the complete exact-spec set, including the
  v2 source-scope filter policy.
- Product truth: assignment and scope-selection workflows are synchronized.
- Architecture truth: configuration, source filtering, cache, runtime, and
  viewer boundaries are current.
- Delivery truth: issues 044–046 are verified and archived; issue 047 is
  registered as `awaiting-human-review`. The node remains `specified` until
  that visual gate is complete.
