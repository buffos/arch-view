# Live analysis and MCP architecture readiness review

## Scope reviewed

The review covers the refreshed live-analysis/MCP child, its discovery notes
and gap analysis, PRD, glossary, domain model, use cases, API/CLI/MCP contract,
acceptance scenarios, and the specified source-index and deterministic-quality
contracts.

## Findings

No High or Medium specification findings remain. The watcher lifecycle,
request-time reconciliation, edit-storm handling, stable-input verification,
atomic snapshot publication, freshness states, analyzer-neutral queries,
quality evaluation/policy delegation, MCP operations, budgets, root safety,
transport policy, and remediation boundary are explicit.

## Cross-document consistency

- The PRD covers all registered analyzers, live read-model lifecycle, strict
  freshness, compact navigation, quality evaluation, and explicitly authorized
  quality-policy operations; parsing, quality semantics, source editing, and
  target execution remain outside it.
- The domain model separates live session config, freshness policy, backend
  events, reconciliation, invalidation, immutable snapshots, query envelopes,
  quality gateway, focused adapters, and permissions.
- The use cases define initial scan, event processing, request-time
  reconciliation, stable-input verification, atomic publication, bounded
  queries, quality delegation, baseline safety, report comparison, and MCP
  mappings.
- The contract fixes `arch-view.live/v1` and `arch-view.query/v1`, defines
  `latest_ready` versus `require_current`, exact-text search, operation
  permissions, profile/evaluation/baseline requests, stdio default transport,
  explicit network policy, cursor/budget semantics, and no source-edit or
  shell/target-execution operations.
- The scenarios cover event storms, missed watcher events, concurrent strict
  requests, changes during analysis, unstable inputs, analyzer capability
  coverage, temporary quality settings, policy authorization, current baseline
  creation, revision comparison, and no mutation.
- The application PRD and architecture summary identify this child as a
  downstream consumer of source-index and quality services while preserving
  their ownership boundaries.

## Residual implementation risks

- OS watcher behavior differs by platform; overflow/reconciliation recovery and
  the polling/fake backend need focused tests.
- A cheap manifest check and full content/input fingerprint must not be
  conflated; the implementation must verify the candidate at both sides of the
  analyzer build.
- Persisted snapshot retention and garbage collection need operational policy,
  but do not alter the immutable query contract.
- MCP SDK versioning, authenticated network deployment, and client-specific
  token estimation require packaging verification.
- Profile/baseline writes require a clear host authorization mechanism and
  audit storage; they must remain unavailable under default read-only policy.
- Incremental reanalysis should remain conservative until dependency-impact
  benchmarks prove a narrower invalidation safe.

## Application synthesis gate

The application PRD and architecture summary are synchronized with this live
boundary, including shared snapshot consumption, strict freshness assurance,
analyzer-neutral capabilities, quality-service delegation, explicit policy
permissions, and the no-autonomous-source-fix rule.

## Artifact impact

- **Capability truth:** this exact-spec set defines watcher, reconciliation,
  snapshot, query, quality-control, MCP, budget, security, and remediation
  boundaries.
- **Product truth:** current/not-current compact source and quality information
  is available to developers and coding assistants across analyzers.
- **Architecture truth:** watcher, coordinator, store, analyzers, quality
  services, query provider, MCP adapter, and transports have focused ownership.
- **Delivery truth:** no implementation issues are created in this pass.

## Readiness

READY FOR ARCHITECTURE IMPLEMENTATION
