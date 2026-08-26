# Go analysis requirements gap analysis

## Resolved

| Area | Resolution |
|---|---|
| Boundary | One `go.mod` module per run; workspace selection is explicit when ambiguous. |
| Node | Go package with attached files. |
| Edge | Static import dependency. |
| Defaults | Exclude tests, vendor, generated/build/cache files, `.git`, and directories named `external`. |
| Resolution | Map local imports to selected module packages; retain non-local/unresolved diagnostics. |
| Safety | Static parsing/configuration first; no application execution. |

## Specification closure and residual risks

Project/workspace selection, static parser boundary, build-view options, module identity, non-local target scopes, and diagnostic behavior are defined in the linked exact-spec artifacts. Parser fixture breadth and build-constraint coverage remain implementation risks.
