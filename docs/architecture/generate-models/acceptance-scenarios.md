# Generate architecture models acceptance scenarios

## SC-GM-001 — Normalize a valid result

Given valid analyzer observations, when normalization runs, then a `complete` canonical model contains stable modules, typed relations, evidence, diagnostics, and deterministic ordering.

Verification: backend-boundary `when-supported`; frontend-integration `not-applicable`; end-to-end `when-supported`.

## SC-GM-002 — Preserve hierarchy separately

Given modules with explicit hierarchy paths, when the model is normalized, then hierarchy is available for projection but does not create dependency relations or cycles.

Verification: backend-boundary `when-supported`; frontend-integration `not-applicable`; end-to-end `when-supported`.

## SC-GM-003 — Preserve non-local targets

Given an external or unresolved target, when normalization runs, then it is retained as a reference/diagnostic and is not invented as a local module.

Verification: backend-boundary `when-supported`; frontend-integration `not-applicable`; end-to-end `when-supported`.

## SC-GM-004 — Preserve evidence through aggregation

Given duplicate module/relationship observations with different evidence, when a hierarchy projection aggregates them, then counts and contributor/evidence links remain traceable.

Verification: backend-boundary `when-supported`; frontend-integration `not-applicable`; end-to-end `when-supported`.

## SC-GM-005 — Preserve canonical cycles

Given a cyclic dependency graph, when layers are derived, then canonical relations remain intact, the cycle is reported, and only derived feedback relations are used for acyclic layering.

Verification: backend-boundary `when-supported`; frontend-integration `not-applicable`; end-to-end `when-supported`.

## SC-GM-006 — Return partial model

Given recoverable conflicting metadata or an invalid individual observation, when normalization runs, then usable records remain with `partial` status and structured diagnostics.

Verification: backend-boundary `when-supported`; frontend-integration `not-applicable`; end-to-end `when-supported`.

## SC-GM-007 — Reject broken top-level integrity

Given a relationship referring to neither a module nor a reference, when validation runs, then the model is rejected with a stable error code.

Verification: backend-boundary `when-supported`; frontend-integration `not-applicable`; end-to-end `when-supported`.

## SC-GM-008 — Repeat deterministically

Given identical observations and policy, when normalization runs twice, then serialized canonical output is byte-stable apart from explicitly requested metadata.

Verification: backend-boundary `when-supported`; frontend-integration `not-applicable`; end-to-end `when-supported`.
