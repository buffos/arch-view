# Python analysis requirements gap analysis

## Resolved

| Area | Resolution |
|---|---|
| Boundary | Analyzer-owned project roots, preferring `pyproject.toml` and supporting common layouts. |
| Node | Python package/module with attached files. |
| Edge | Static import dependency. |
| Resolution | Absolute/relative filesystem resolution where deterministic; dynamic cases become diagnostics. |
| Safety | Never import or execute project code. |
| Defaults | Exclude tests, caches, generated/build/vendor files, `.git`, and directories named `external`. |

## Specification closure and residual risks

Project-marker precedence, source roots, namespace/package rules, static import semantics, stubs, and dynamic diagnostics are defined in the linked exact-spec artifacts. Python-version and namespace-package fixture breadth remain implementation risks.
