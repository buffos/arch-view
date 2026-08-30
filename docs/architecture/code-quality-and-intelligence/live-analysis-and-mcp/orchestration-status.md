# Live analysis and MCP orchestration status

## State

- Planning state: `specified` after the bounded-to-specified transition.
- This run refreshes the exact specification in place; it does not create a
  topology change or claim implementation.
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
- **Delivery:** no issue registry entries or implementation claims were added.

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

## Artifact impact assessment for issue slicing

- **Topology:** no impact. The existing specified node owns this batch; no node
  is added, split, promoted, or advanced.
- **Capability:** updated this record and the owning node's delivery references
  with issues 064–076.
- **Product:** no impact at slicing. The application PRD already describes the
  live developer/agent journeys and quality-policy boundary.
- **Architecture:** no impact at slicing. The application architecture summary
  already defines the coordinator, query, quality-delegation, transport, and
  source-safety boundaries.
- **Delivery:** issue files, registry rows, max issue ID, and node references
  are now synchronized. Implementation closure must reassess product and
  architecture truth if behavior differs from the approved contract.

## Readiness decision

READY FOR ARCHITECTURE IMPLEMENTATION. Future delivery may slice watcher and
reconciliation, coordinator/store, analyzer-neutral query projections, exact
text search, MCP stdio, quality-service delegation, policy permissions, and
optional authenticated HTTP independently. Source-edit/remediation remains a
separate future capability and is not implemented here.
