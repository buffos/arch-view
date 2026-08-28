# Multi-analyzer project orchestration orchestration status

## State

- Planning state: `specified`.
- The exact-spec pipeline is complete; implementation issues have not yet been created.

## Evidence

- The shared analyzer contract, canonical result format, and external process
  lifecycle are implemented.
- The current host and public `open` path execute one analyzer per request.
- Multiple external descriptors can be registered, but they are not yet
  scheduled as a combined analysis plan.

## Artifact sync

- **Topology:** This child is linked under `Analyzer plugin runtime`.
- **Capability:** Discovery notes separate current single-analyzer behavior from
  the bounded marker-driven job plan, namespaced merge, bounded concurrency,
  combined model, and partial-failure target.
- **Product and architecture:** The application synthesis records the future
  multi-job orchestration boundary.
- **Delivery:** No issues were created or changed in this graph update.

## Exact-spec inventory

- Requirements gap analysis
- Domain glossary
- Capability PRD
- Canonical domain model
- Canonical use-case model
- Canonical API/CLI contract
- Acceptance scenarios
- Architecture readiness review

All artifacts are linked from the capability node and agree on marker traversal,
nested ownership, job identity, worker limits, cancellation, aggregate status,
provenance, progress, and no-inferred-relationship behavior.

## Readiness decision

READY FOR ARCHITECTURE IMPLEMENTATION. The next step is reference-doc issue
slicing after the application synthesis gate is checked for all four specified
nodes.

## Artifact impact

- Capability truth: updated with the complete exact-spec set.
- Product truth: combined/per-scope and partial-result behavior is synchronized.
- Architecture truth: aggregate model and canonical-normalization dependency are current.
- Delivery truth: intentionally unchanged; no issue slicing in this run.
