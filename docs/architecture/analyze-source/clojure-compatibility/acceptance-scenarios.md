# Clojure compatibility acceptance scenarios

## SC-CL-001 — Discover namespaces

Given valid `.clj`, `.cljs`, or `.cljc` files with `ns` forms, when analysis runs, then namespace modules and structured hierarchy are returned with file evidence.

Verification: backend-boundary `when-supported`; frontend-integration `not-applicable`; end-to-end `when-supported`.

## SC-CL-002 — Extract static dependencies

Given `:require`, `:use`, and macro dependency clauses, when analysis runs, then typed dependency observations contain aliases/kind and source locations.

Verification: backend-boundary `when-supported`; frontend-integration `not-applicable`; end-to-end `when-supported`.

## SC-CL-003 — Preserve reader conditionals

Given a `.cljc` namespace with platform-specific dependencies, when `platform=both`, then observations retain platform metadata and are not treated as runtime-tested facts.

Verification: backend-boundary `when-supported`; frontend-integration `not-applicable`; end-to-end `when-supported`.

## SC-CL-004 — Mark polymorphic forms

Given statically recognized `defprotocol` or `defmulti` forms, when analysis runs, then the module receives `polymorphic` metadata with evidence and the core relation type remains generic.

Verification: backend-boundary `when-supported`; frontend-integration `not-applicable`; end-to-end `when-supported`.

## SC-CL-005 — Do not evaluate forms

Given a namespace whose evaluation would have side effects, when analysis runs, then no form is evaluated or loaded and static results/diagnostics are returned.

Verification: backend-boundary `when-supported`; frontend-integration `not-applicable`; end-to-end `when-supported`.
