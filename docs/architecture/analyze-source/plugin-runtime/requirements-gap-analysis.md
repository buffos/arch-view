# Analyzer plugin runtime requirements gap analysis

## Resolved

| Area | Resolution |
|---|---|
| First deployment | In-process Go analyzer interface. |
| Extensibility | Manifest-driven registration and capability negotiation. |
| Selection | Explicit language override or unambiguous auto-detection. |
| Configuration | CLI > project configuration > analyzer defaults; persisted assignment and source-scope policy stays in the project configuration boundary. |
| Safety | Read-only analysis; no target application execution. |
| Result ownership | Analyzer extracts; host validates/normalizes; model owns canonical graph. |
| Failure behavior | Diagnostics and usable partial results where possible. |
| External registration | Explicit local descriptor; no implicit PATH scan or project configuration execution in the pilot. |
| External lifecycle | One process per detect/analyze request, hello manifest agreement, bounded frames/stderr, cancellation, timeout, and child cleanup. |
| External pilot | Existing Python static-analysis semantics ported with Python ast and compared with the in-process baseline. |

## Specification closure and residual risks

The exact Go interface, manifest schema, detection/selection rules, lifecycle
outcomes, descriptor, NDJSON frame semantics, published schemas, external
Python pilot boundary, and the child-owned assignment/source-scope seam are
defined in the linked exact-spec artifacts.
Remaining risks are v1 migration/distribution beyond this opt-in pilot,
interpreter availability across environments, and benchmark-driven tuning of
process limits.

## Future target implementation slices

| Target | Current state | Gap to clarify |
|---|---|---|
| Compiled external analyzers | The external pilot launches `python launcher.py`; built-in analyzers remain in-process. | Exact specification complete; implement executable packaging, shared implementation entrypoints, platform artifacts, checksum trust, versioning, and explicit runtime overrides. |
| Multi-analyzer orchestration | The host selects one analyzer and returns one result per run. | Exact specification complete; implement project-root discovery, concurrent job planning, merge identity/provenance, progress, cache reuse, source-scope application, and partial failure semantics. |
| Project assignments and view selection | `.archview.json` currently owns presentation layout, while analyzer selection is CLI/request scoped and no persisted source-scope policy exists. | Exact specification complete; implement folder-to-analyzer configuration, invocation-root global excludes and analyzer-scoped includes, precedence, nested assignments, cached scopes, combined views, and application dropdown behavior. |

These are user-confirmed future capabilities and are intentionally not folded
into the completed v1 exact-spec contract. They do not reopen the implemented
external Python pilot; each is now a specified child frontier with its own
readiness-reviewed exact specification before delivery work.
