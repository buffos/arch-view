# Python analysis acceptance scenarios

## SC-PY-001 — Discover source-root packages

Given a project using a `src/` layout, when analysis runs, then package/module hierarchy is rooted at `src/` and source paths remain evidence.

Verification: backend-boundary `when-supported`; frontend-integration `not-applicable`; end-to-end `when-supported`.

## SC-PY-002 — Resolve absolute and relative imports

Given deterministic absolute and relative imports, when analysis runs, then local `depends_on` observations point to the proven target modules with locations.

Verification: backend-boundary `when-supported`; frontend-integration `not-applicable`; end-to-end `when-supported`.

## SC-PY-003 — Report dynamic imports

Given `importlib` or computed `__import__` usage, when analysis runs, then the result remains usable and reports a dynamic reference/diagnostic rather than fabricating a module edge.

Verification: backend-boundary `when-supported`; frontend-integration `not-applicable`; end-to-end `when-supported`.

## SC-PY-004 — Never execute Python

Given a project whose import-time code would have side effects, when analysis runs, then no import/install/execution occurs and static results are still returned.

Verification: backend-boundary `when-supported`; frontend-integration `not-applicable`; end-to-end `when-supported`.

## SC-PY-005 — Honor stub option

Given `.pyi` files, when `include_stubs=false` they are excluded, and when `include_stubs=true` they are included with stub metadata.

Verification: backend-boundary `when-supported`; frontend-integration `not-applicable`; end-to-end `when-supported`.
