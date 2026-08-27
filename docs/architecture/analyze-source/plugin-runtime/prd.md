# Analyzer plugin runtime PRD

## Purpose

Allow new language analyzers to be registered, selected, configured, executed, and evolved without changing the canonical model or presentation layers.

## Scope and goals

- Validate analyzer manifests and API compatibility.
- Detect projects and resolve explicit/automatic analyzer selection.
- Apply option precedence and run analyzers with cancellation.
- Validate and normalize results before handing them to the model.
- Provide an in-process Go contract first and an opt-in versioned external
  process boundary once the in-process semantics are stable.
- Validate the boundary with an external Python port of the implemented
  static analyzer before considering additional external languages.

The runtime does not parse source, calculate graph layout, render graphics, or decide language semantics.

## User stories

- US-PR-001 — As a developer, I can list supported analyzers and select one explicitly so that analysis is predictable.
- US-PR-002 — As a developer or CI job, I can run a selected analyzer with effective options, cancellation, and status-bearing errors so that automation is safe.
- US-PR-003 — As a maintainer, I can supply an external analyzer descriptor
  explicitly so that an analyzer written outside Go can produce the same
  architecture result without changing the model or viewer.

## Rules

- Explicit language selection wins; auto-detection must be unambiguous.
- CLI options override project configuration, which overrides manifest defaults.
- A run is read-only and receives a cancellation context.
- Plugin stdout is not a product protocol for built-ins; external process
  plugins reserve stdout for NDJSON and stderr for logs.
- A plugin cannot return UI objects, layout coordinates, or arbitrary executable callbacks.
- External plugins are loaded only from an explicitly supplied local
  descriptor. The pilot performs no implicit PATH scan, project-file plugin
  discovery, or shell command execution.
- One external process handles one detection or analysis request. Its first
  frame is hello, its final frame is done or fatal, and its hello manifest
  must agree with the descriptor.

## Functional requirements

| ID | Requirement |
|---|---|
| PR-FR-001 | Register only manifests compatible with the host API version. |
| PR-FR-002 | Return deterministic detection candidates with reasons and confidence. |
| PR-FR-003 | Reject ambiguous or unsupported selection clearly. |
| PR-FR-004 | Resolve and validate analyzer options before execution. |
| PR-FR-005 | Propagate cancellation, timeout, diagnostics, and panic/error outcomes safely. |
| PR-FR-006 | Validate result references and hand normalized observations to the model. |
| PR-FR-007 | Preserve the external protocol without changing analyzer semantics. |
| PR-FR-008 | Load an explicit process-plugin descriptor and register only a manifest-compatible analyzer. |
| PR-FR-009 | Preserve detection, option, result, diagnostic, and status semantics across the v1 NDJSON boundary. |
| PR-FR-010 | Bound process frames/logs, reject protocol violations, and clean up cancelled or timed-out children. |
| PR-FR-011 | Validate the external boundary with a standard-library-only Python analyzer that is semantically comparable to the in-process adapter. |

## External pilot boundary

The first external deployment is an opt-in Python analyzer with manifest ID
org.archview.python.external. The descriptor and process protocol are defined
in the canonical contract and published schemas. The external plugin uses
Python ast and safe configuration readers, does not execute target code, and
returns the same language-neutral observations as the completed in-process
Python capability for the representative parity fixture.

## Acceptance summary

See [acceptance scenarios](acceptance-scenarios.md). A plugin is ready for
implementation when the manifest, lifecycle, selection, and result boundaries
are honored identically by built-in and process adapters. Issues 030–033 in
the [external implementation slice](implementation-slice.md) deliver the
schema/fixture, process host, Python parity, and public opt-in path.
