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
| External registration | Explicit local descriptor; no implicit PATH scan or project configuration execution in the pilot. |
| External lifecycle | One process per detect/analyze request, hello manifest agreement, bounded frames/stderr, cancellation, timeout, and child cleanup. |
| External pilot | Existing Python static-analysis semantics ported with Python ast and compared with the in-process baseline. |

## Specification closure and residual risks

The exact Go interface, manifest schema, detection/selection rules, lifecycle
outcomes, descriptor, NDJSON frame semantics, published schemas, and external
Python pilot boundary are defined in the linked exact-spec artifacts.
Remaining risks are v1 migration/distribution beyond this opt-in pilot,
interpreter availability across environments, and benchmark-driven tuning of
process limits.
