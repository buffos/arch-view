# Analyzer plugin runtime requirements gap analysis

## Resolved

| Area | Resolution |
|---|---|
| First deployment | In-process Go analyzer interface. |
| Extensibility | Manifest-driven registration and capability negotiation. |
| Selection | Explicit language override or unambiguous auto-detection. |
| Configuration | CLI > project configuration > analyzer defaults. |
| Safety | Read-only analysis; no target application execution. |
| Result ownership | Analyzer extracts; host validates/normalizes; model owns canonical graph. |
| Failure behavior | Diagnostics and usable partial results where possible. |
| External plugins | Later versioned NDJSON/JSON Schema boundary. |

## Specification closure and residual risks

The exact Go interface, manifest schema, detection/selection rules, lifecycle outcomes, and NDJSON frame semantics are defined in the linked exact-spec artifacts. Resource limits and published JSON Schema interoperability remain implementation risks.
