# Multi-analyzer project orchestration requirements gap analysis

## Scope examined

This pass covers the bounded [multi-analyzer project orchestration](../../../../.okf/capabilities/analyze-source/plugin-runtime/multi-analyzer-orchestration.md) child, the current single-root/single-analyzer host, the canonical model normalizer, the external process boundary, and the confirmed project-assignment direction.

## Confirmed strong areas

- **Observed in code:** `analysis.Host.Run` selects one analyzer and returns one `AnalysisResult`; the process adapter already provides isolated cancellation, timeout, and diagnostic behavior for one operation.
- **Observed in code:** canonical normalization already sorts collections, validates endpoint/evidence references, derives graph projections, and preserves recoverable diagnostics.
- **Inferred from docs:** the model and viewer can preserve language, project, source evidence, diagnostics, and provenance if the aggregation boundary namespaces observations before normalization.
- **User-confirmed target behavior:** one opened repository can produce multiple independent analyzer jobs, run concurrently with bounded resources, and retain successful results when another job fails.

## Blocking gaps resolved by recommendation

| Gap | Impact | Resolution |
|---|---|---|
| Root discovery and nested ownership were not algorithmic | High: different implementations would analyze or duplicate different files | Scan the opened repository lexically, recognize declared strong manifest markers, carve out nested strong roots, and pass child-root exclusions to the parent job. Explicit assignments can create or override a root. |
| Aggregate identity was not stable | High: same local module IDs would collide across languages or roots | Derive a stable scope ID from normalized repository-relative root plus logical analyzer ID; namespace every module, reference, source, and relationship ID with that scope. |
| Partial-run status was not exact | High: one failed analyzer could erase useful architecture or produce contradictory client behavior | Complete means all planned jobs usable; partial means at least one usable result plus one failed/partial/cancelled job; failed/cancelled apply only when no usable result remains. |
| Concurrency and cancellation limits were not exact | High: unbounded process creation could exhaust the host | Default four workers, hard cap sixteen, maximum 128 planned jobs, and cancellation that stops queued jobs and terminates active external processes. |
| Progress behavior was not stable | Medium: UI/CI consumers could not distinguish job lifecycle from analyzer-internal progress | Emit deterministic lifecycle events with sorted job snapshots; analyzer-internal percentage is optional and never required for completion. |

## Deferrable implementation details

- The scheduler may use goroutines, a task queue, or another worker implementation.
- The exact filesystem fingerprint algorithm can be optimized, but cache identity must include the complete analyzer input set and package/version identity.
- Cross-language semantic relationship inference, call graphs, and workspace-specific language semantics remain outside this child.

## Exact assumptions

- Automatic root discovery recognizes strong manifest markers declared by analyzer manifests; weak markers only support detection inside a selected root.
- Symlinks are not traversed by automatic discovery in the first implementation.
- A combined result is an aggregate model with per-scope provenance; it does not pretend that independent analyzers reported cross-language relationships.

## Readiness conclusion

No High or Medium specification gaps remain. The remaining work is to encode the
job-plan, aggregation, progress, and resource rules in exact artifacts.

## Artifact impact

- **Capability:** this report and the linked exact-spec set define discovery, scheduling, identity, aggregation, and failure behavior.
- **Product:** the application PRD must retain the combined-view workflow and partial-result behavior.
- **Architecture:** the application summary must reflect the aggregate model boundary and the dependency on canonical model normalization.
- **Delivery:** no issues are created in this pass.
