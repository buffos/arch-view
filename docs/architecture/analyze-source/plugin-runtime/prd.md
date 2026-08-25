# Analyzer plugin runtime PRD

## Purpose

Allow new language analyzers to be registered, selected, configured, executed, and evolved without changing the canonical model or presentation layers.

## Scope and goals

- Validate analyzer manifests and API compatibility.
- Detect projects and resolve explicit/automatic analyzer selection.
- Apply option precedence and run analyzers with cancellation.
- Validate and normalize results before handing them to the model.
- Provide an in-process Go contract first and a versioned external process boundary later.

The runtime does not parse source, calculate graph layout, render graphics, or decide language semantics.

## User stories

- US-PR-001 — As a developer, I can list supported analyzers and select one explicitly so that analysis is predictable.
- US-PR-002 — As a developer or CI job, I can run a selected analyzer with effective options, cancellation, and status-bearing errors so that automation is safe.

## Rules

- Explicit language selection wins; auto-detection must be unambiguous.
- CLI options override project configuration, which overrides manifest defaults.
- A run is read-only and receives a cancellation context.
- Plugin stdout is not a product protocol for built-ins; future process plugins reserve stdout for NDJSON and stderr for logs.
- A plugin cannot return UI objects, layout coordinates, or arbitrary executable callbacks.

## Functional requirements

| ID | Requirement |
|---|---|
| PR-FR-001 | Register only manifests compatible with the host API version. |
| PR-FR-002 | Return deterministic detection candidates with reasons and confidence. |
| PR-FR-003 | Reject ambiguous or unsupported selection clearly. |
| PR-FR-004 | Resolve and validate analyzer options before execution. |
| PR-FR-005 | Propagate cancellation, timeout, diagnostics, and panic/error outcomes safely. |
| PR-FR-006 | Validate result references and hand normalized observations to the model. |
| PR-FR-007 | Preserve a future external protocol without changing analyzer semantics. |

## Acceptance summary

See [acceptance scenarios](acceptance-scenarios.md). A plugin is ready for implementation when the manifest, lifecycle, selection, and result boundaries are honored identically by built-in and future process adapters.
