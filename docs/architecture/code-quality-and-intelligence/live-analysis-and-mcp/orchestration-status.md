# Live analysis and MCP orchestration status

## State

- Planning state: `specified` after the bounded-to-specified transition.
- Issues 064–070 are implemented and verified in the shared live boundary; it
  does not create a topology change or advance the capability state.
- The refreshed exact-spec pipeline covers analyzer-neutral live queries,
  request-time freshness reconciliation, stable-input verification, quality
  evaluation, explicit quality-policy permissions, and the external agent fix
  loop.

## Exact-spec inventory

- Requirements gap analysis
- Domain glossary
- Capability PRD
- Canonical domain model
- Canonical use-case model
- Canonical API/CLI/MCP contract
- Acceptance scenarios
- Architecture readiness review

The artifacts agree on configured roots, all-analyzer capability negotiation,
event normalization, debounce and overflow recovery, request-time
reconciliation, single-flight reanalysis, stable-input verification, atomic
coherent revisions, last-ready retention, deterministic structural and exact-
text search, quality catalog/evaluation/policy delegation, budgets/cursors,
stdio/network transport policy, root safety, operation permissions, and
explicit remediation boundaries.

## Implementation progress

- **064–067:** Live session configuration, asynchronous initial scans, watcher
  normalization/coalescing, immutable revision publication, authoritative
  reconciliation, stable-input verification, selective invalidation, and
  single-flight rebuilds are implemented under `internal/live/`. The live
  source-index enabled flag and requested capabilities also cross the common
  analyzer and process-plugin boundaries; a nil analyzer request preserves
  legacy defaults.
- **068–069:** Revision-bound analyzer-neutral structural queries, capability
  coverage, deterministic cursors/budgets, exact text search, and bounded
  source context are implemented through the local query adapter.
- **070:** Quality catalog/profile reads, findings/evidence, temporary
  evaluation, and report comparison delegate to the existing deterministic
  quality services through `QualityGateway`.
- **071–076:** Permissioned policy writes, local CLI/viewer consumers, MCP
  transports/documentation, cross-analyzer conformance, and final product
  approval remain active delivery work.

## Artifact impact

- **Topology:** no topology or state transition; the node remains `specified`.
- **Capability:** the live/MCP exact contract now includes freshness assurance,
  analyzer-neutral navigation, temporary quality evaluation, and permissioned
  profile/baseline operations.
- **Product:** developers and coding assistants can request current data across
  all registered analyzers and follow a finding through an external fix loop.
- **Architecture:** the coordinator owns reconciliation and snapshot truth;
  analyzers own language facts; deterministic quality owns profiles/rules/
  baselines; MCP delegates through one shared service boundary.
- **Delivery:** issues 064–070 are archived as complete; remaining issue
  blockers and owning-node references are synchronized. The node remains
  `specified` because the approved frontier is not exhausted.

## Approved delivery frontier

The approved dependency-ordered implementation batch is issues 064–076:

- 064–067: live session configuration, watcher events, coherent revisions,
  request-time reconciliation, stable-input verification, and single-flight
  rebuilds.
- 068–071: analyzer-neutral queries, exact text/source context, quality
  gateway/evaluation, and separately permissioned profile/baseline operations.
- 072–075: local CLI/viewer consumers, MCP stdio plus installation
  documentation, and optional authenticated HTTP transport.
- 076: cross-analyzer conformance, external agent fix-loop verification, and
  final product approval.

All slices are `feature`/`AFK`; only issue 076 has the `product-approval`
review gate. Issues 064–075 are `ready-for-agent` and are ordered by their
declared local-file dependencies.

## Artifact impact assessment for implementation batch

- **Topology:** no impact. The existing specified node owns this batch; no node
  is added, split, promoted, or advanced.
- **Capability:** implementation progress is recorded here and in the owning
  node; the state remains `specified` until issues 071–076 are completed.
- **Product:** no impact at slicing. The application PRD already describes the
  live developer/agent journeys and quality-policy boundary.
- **Architecture:** no impact at slicing. The application architecture summary
  already defines the coordinator, query, quality-delegation, transport, and
  source-safety boundaries.
- **Delivery:** issue files 064–070, registry rows, remaining blockers, and
  node references are synchronized. No product or architecture change was
  needed because the implementation preserves the approved boundaries.

## Readiness decision

IMPLEMENTATION IN PROGRESS. Issues 064–070 are complete and verified. Issues
071–076 remain the active frontier for policy permissions, local consumers,
MCP packaging, optional authenticated HTTP, cross-analyzer conformance, and
final product approval. Source-edit/remediation remains a separate future
capability and is not implemented here.
