# Live analysis and MCP architecture readiness review

## Scope reviewed

The review covers the bounded live-analysis/MCP child, its discovery notes and
gap analysis, PRD, glossary, domain model, use cases, API/CLI/MCP contract,
acceptance scenarios, and the specified source-index and quality-report
contracts.

## Findings

No High or Medium specification findings remain. The watcher lifecycle,
invalidation/recovery rules, atomic snapshot publication, freshness states,
query consistency, structural search, MCP operations, budgets, root safety,
transport policy, and remediation boundary are explicit.

## Cross-document consistency

- The PRD limits the child to live read-model lifecycle and query/transport
  behavior; parsing, quality semantics, source editing, and target execution
  remain outside it.
- The domain model separates live session config, backend events, invalidation,
  immutable snapshots, query envelopes, focused adapters, and permissions.
- The use cases define initial scan, event processing, conservative invalidation,
  atomic publication, bounded query, MCP mappings, and explicit remediation
  handoff.
- The contract fixes `arch-view.live/v1` and `arch-view.query/v1`, stdio as the
  default local transport, explicit network policy, tool/resource mappings,
  cursor/budget semantics, and no v1 write/execute operations.
- The scenarios cover current/stale/updating/degraded behavior, overflow,
  failed updates, revision consistency, structural search, capability
  limitations, token budgets, path safety, parity, and no mutation.
- The application PRD and architecture summary identify this child as a
  downstream consumer of source-index and quality contracts.

## Residual implementation risks

- OS watcher behavior differs by platform; overflow/rescan recovery and the
  polling/fake backend need focused tests.
- Persisted snapshot retention and garbage collection need operational policy,
  but do not alter the immutable query contract.
- MCP SDK versioning, authenticated network deployment, and client-specific
  token estimation require packaging verification.
- Incremental reanalysis should remain conservative until dependency-impact
  benchmarks prove a narrower invalidation safe.

## Application synthesis gate

The application PRD and architecture summary are synchronized with this live
boundary, including shared snapshot consumption, freshness/atomicity, compact
structural queries, explicit MCP permissions, and the no-autonomous-fix rule.

## Artifact impact

- **Capability truth:** this exact-spec set defines watcher, snapshot, query,
  MCP, budget, security, and remediation boundaries.
- **Product truth:** current/stale quality and source information has a shared
  low-token read surface for developers and LLMs.
- **Architecture truth:** watcher, coordinator, store, query provider, and
  transports have focused replaceable ownership.
- **Delivery truth:** no implementation issues are created in this pass.

## Readiness

READY FOR ARCHITECTURE IMPLEMENTATION
