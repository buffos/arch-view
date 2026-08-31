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

- OS watcher behavior still differs by platform. The current local watcher is a
  portable polling backend; overflow, missed-event, and startup-gap recovery
  are covered by the shared reconciliation path, but platform-specific
  performance tuning remains future work.
- The live coordinator now checks an authoritative input fingerprint before and
  after each candidate build. A cheap manifest check is still only a fast
  path, never proof of currentness by itself.
- Persisted snapshot retention and garbage collection need operational policy,
  but do not alter the immutable query contract.
- The MCP surface uses the repository's newline-delimited JSON-RPC adapter;
  client-specific SDK packaging and authenticated network deployment remain
  integration concerns. The CLI currently binds HTTP transports to loopback.
- Profile/baseline writes use explicit operation permissions, an opaque host
  authorization callback, safe project-relative destinations, atomic writes,
  and audit results. They remain unavailable under the default read-only
  session policy.
- Incremental reanalysis remains conservative until dependency-impact
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
  is available to developers and coding assistants across registered
  analyzers.
- **Architecture truth:** watcher, coordinator, store, analyzers, quality
  services, query provider, MCP adapter, viewer, CLI, and transports have
  focused ownership and share one revision/query boundary.
- **Delivery truth:** issues 064–075 are implemented and verified. Issue 076
  has automated conformance evidence and is waiting only for its final human
  product-approval inspection.

## Readiness

AUTOMATED IMPLEMENTATION COMPLETE. HUMAN PRODUCT REVIEW PENDING.
