# Live analysis and MCP orchestration status

## State

- Planning state: `specified` after the bounded-to-specified transition.
- The exact-spec pipeline is complete and readiness-reviewed.
- No implementation or delivery issue is claimed by this planning pass.

## Exact-spec inventory

- Requirements gap analysis
- Domain glossary
- Capability PRD
- Canonical domain model
- Canonical use-case model
- Canonical API/CLI/MCP contract
- Acceptance scenarios
- Architecture readiness review

The artifacts agree on configured roots, event normalization, debounce and
overflow recovery, conservative invalidation, atomic coherent revisions,
last-ready retention, deterministic structural search, quality/call capability
coverage, budgets/cursors, stdio/network transport policy, root safety,
read-only permissions, and explicit remediation handoff.

## Artifact impact

- **Capability:** live analysis/MCP now has an exact contract downstream of
  source facts and deterministic quality reports.
- **Product:** viewer, CLI, and LLM tools can share current or explicitly stale
  compact data without repeated full-source loading.
- **Architecture:** watch backends, snapshot store, query provider, and
  transport adapters remain open for extension and independently testable.
- **Delivery:** no issue registry entries or implementation claims were added.

## Readiness decision

READY FOR ARCHITECTURE IMPLEMENTATION. Future delivery may slice watcher,
coordinator/store, query projections, MCP stdio, optional authenticated HTTP,
and remediation handoff independently. The remediation handoff remains a
future permissioned capability and is not implemented here.
