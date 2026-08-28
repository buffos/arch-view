# Multi-analyzer project orchestration acceptance scenarios

## SC-MAO-001 — Discover mixed-language roots

**Given** an opened repository containing Go, Python, TypeScript, Rust, and
Clojure manifest roots, **when** the host builds a plan, **then** it discovers
the applicable roots in deterministic order and creates one job per applicable
root/language scope.

Verification: backend-boundary `when-supported`; frontend-integration
`when-supported`; end-to-end `when-supported`.

## SC-MAO-002 — Respect nested project ownership

**Given** a strong manifest root containing a nested strong manifest, **when**
the plan is built, **then** the nested root becomes its own scope and the
parent job receives that subtree as an exclusion, preventing duplicate
ownership.

Verification: backend-boundary `when-supported`; frontend-integration
`not-applicable`; end-to-end `when-supported`.

## SC-MAO-003 — Keep discovery inside the repository

**Given** symlinks, build directories, vendor directories, and files outside
the opened repository, **when** automatic discovery runs, **then** excluded or
external paths are not traversed or scheduled.

Verification: backend-boundary `when-supported`; frontend-integration
`not-applicable`; end-to-end `when-supported`.

## SC-MAO-004 — Run jobs with bounded concurrency

**Given** more than four independent jobs, **when** the plan executes, **then**
no more than four jobs run concurrently by default and no configuration can
raise the hard cap above sixteen.

Verification: backend-boundary `when-supported`; frontend-integration
`not-applicable`; end-to-end `when-supported`.

## SC-MAO-005 — Preserve independent success after failure

**Given** two jobs where one completes and one fails, **when** aggregation ends,
**then** the successful scope and its relationships remain visible, the
aggregate status is `partial`, and the failed scope's diagnostics remain
attached to its scope.

Verification: backend-boundary `when-supported`; frontend-integration
`when-supported`; end-to-end `when-supported`.

## SC-MAO-006 — Namespace colliding observations

**Given** two scopes whose analyzers emit the same local module, reference, or
relationship IDs, **when** results are aggregated, **then** all observations
remain distinct through stable scope-namespaced IDs and no relationship points
to another scope accidentally.

Verification: backend-boundary `when-supported`; frontend-integration
`not-applicable`; end-to-end `when-supported`.

## SC-MAO-007 — Do not infer cross-scope relationships

**Given** two analyzers report modules with matching names or paths but neither
reports a cross-scope relationship, **when** the combined model is built,
**then** no cross-language or cross-root relationship is added.

Verification: backend-boundary `when-supported`; frontend-integration
`when-supported`; end-to-end `when-supported`.

## SC-MAO-008 — Cancel queued and active work

**Given** a plan with queued and running jobs, **when** the caller cancels the
run, **then** queued jobs become `cancelled`, active external processes are
terminated through the process boundary, and no late result is accepted.

Verification: backend-boundary `when-supported`; frontend-integration
`when-supported`; end-to-end `when-supported`.

## SC-MAO-009 — Expose combined and individual scopes

**Given** a completed or partial aggregate with multiple usable scopes, **when**
the user selects `All` or one scope, **then** the viewer changes projection
from cached results without running an analyzer again, and the individual scope
shows the same facts it contributed to `All`.

Verification: backend-boundary `when-supported`; frontend-integration
`when-supported`; end-to-end `when-supported`.

## SC-MAO-010 — Report no-usable-result failure

**Given** every planned job fails or is cancelled before producing a usable
result, **when** aggregation ends, **then** no combined model is exposed and
the run reports `failed` or `cancelled` with all scope diagnostics.

Verification: backend-boundary `when-supported`; frontend-integration
`when-supported`; end-to-end `when-supported`.
