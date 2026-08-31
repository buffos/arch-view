# Live analysis and MCP orchestration status

## State

- Planning state: `implemented` after completion of issue 076's final
  product-approval gate.
- Issues 064–079 are implemented and verified in the shared live boundary;
  issue 076 records the final cross-analyzer and human product approval.
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
- **071:** Permissioned profile and baseline operations are implemented through
  the shared quality-policy service. Default denial, explicit authorization,
  exact report/profile identity, safe destinations, audit results, and atomic
  non-conflicting writes are covered by policy tests.
- **072:** `arch-view live start|status|wait|ensure-current` and the documented
  local endpoint bridge are implemented. Analyzer-specific CLI options are
  passed through the shared multi-analyzer scanner.
- **073:** `open --live` and `NewLiveServer` attach the viewer to the same live
  session. Status drives revision-pinned model, source, and quality reads; the
  last ready revision remains visible during updates.
- **074:** `arch-view mcp` provides the documented stdio JSON-RPC tools and
  resources over the shared adapter. The website contains installation,
  configuration, tool, quality, safety, freshness, and troubleshooting
  guidance.
- **075:** The opt-in local/authenticated HTTP adapter maps the same query and
  quality services, enforces loopback/authentication/origin/request limits,
  supports encoded opaque IDs, and returns structured transport errors.
- **077–079:** Managed quality baselines now have canonical profile references,
  automatic discovery, deterministic revisions, idempotent append and atomic
  profile/baseline publication. The CLI and MCP expose the same lifecycle;
  MCP can list/read baselines, select one for a temporary evaluation, and
  append reviewed findings only through authorized audited policy writes.
- **076:** Automated mixed-analyzer, freshness, transport, viewer, CLI, and
  documentation evidence is complete. The user approved the final desktop/
  responsive viewer, MCP usability, status wording, and safety-boundary review
  on 2026-08-31; the issue records the approval and is closed.

## Artifact impact

- **Topology:** no topology change; the node advanced from `specified` to
  `implemented` after its scoped delivery was exhausted.
- **Capability:** the live/MCP exact contract now includes freshness assurance,
  analyzer-neutral navigation, temporary quality evaluation, and permissioned
  profile/baseline operations.
- **Product:** developers and coding assistants can request current data across
  all registered analyzers and follow a finding through an external fix loop.
- **Architecture:** the coordinator owns reconciliation and snapshot truth;
  analyzers own language facts; deterministic quality owns profiles/rules/
  baselines; MCP delegates through one shared service boundary.
- **Delivery:** issues 064–079 are archived as complete. Issue 076's owning-
  node reference, approval record, and final inspection evidence are
  synchronized.

## Approved delivery frontier

The approved dependency-ordered implementation batch is issues 064–079:

- 064–067: live session configuration, watcher events, coherent revisions,
  request-time reconciliation, stable-input verification, and single-flight
  rebuilds.
- 068–071: analyzer-neutral queries, exact text/source context, quality
  gateway/evaluation, and separately permissioned profile/baseline operations.
- 072–075: local CLI/viewer consumers, MCP stdio plus installation
  documentation, and optional authenticated HTTP transport.
- 077–079: managed baseline storage and merge, the automatic CLI lifecycle,
  and live/MCP baseline read, selection, and append operations.
- 076: cross-analyzer conformance, external agent fix-loop verification, and
  final product approval.

All slices are `feature`/`AFK`; only issue 076 had the `product-approval`
review gate. Issues 064–079 are verified and archived in dependency order.

## Artifact impact assessment for implementation batch

- **Topology:** no impact. The existing specified node owns this batch; no node
  is added, split, promoted, or advanced.
- **Capability:** implementation progress is recorded here and in the owning
  node; issue 076 completed the final scoped product review and advanced the
  node to `implemented`.
- **Product:** no impact at slicing. The application PRD already describes the
  live developer/agent journeys and quality-policy boundary.
- **Architecture:** no impact at slicing. The application architecture summary
  already defines the coordinator, query, quality-delegation, transport, and
  source-safety boundaries.
- **Delivery:** issue files 064–079, registry rows, blockers, and node
  references are synchronized. No product or architecture change was needed
  because the implementation and final review preserve the approved
  boundaries.

## Readiness decision

IMPLEMENTATION AND PRODUCT REVIEW COMPLETE. Issues 064–079 are complete and
verified. The user-approved final review covered cross-analyzer conformance,
viewer/MCP usability, status wording, and safety boundaries.
Source-edit/remediation remains a separate future capability and is not
implemented here.
