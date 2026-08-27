# Analyzer plugin runtime acceptance scenarios

## SC-PR-001 — Register a compatible analyzer

Given a valid unique manifest with a supported API version, when it is registered, then it appears in deterministic analyzer listings.

Verification: backend-boundary `when-supported`; frontend-integration `not-applicable`; end-to-end `when-supported`.

## SC-PR-002 — Reject incompatible manifests

Given a duplicate ID, malformed manifest, or unsupported API version, when registration occurs, then the analyzer is rejected with a stable diagnostic and no partial registration.

Verification: backend-boundary `when-supported`; frontend-integration `not-applicable`; end-to-end `not-applicable`.

## SC-PR-003 — Resolve unambiguous detection

Given one highest-confidence candidate, when auto-detection runs, then that analyzer is selected with marker reasons recorded.

Verification: backend-boundary `when-supported`; frontend-integration `not-applicable`; end-to-end `when-supported`.

## SC-PR-004 — Stop on ambiguity

Given tied highest-confidence candidates, when no explicit language is supplied, then selection fails clearly without running an analyzer.

Verification: backend-boundary `when-supported`; frontend-integration `not-applicable`; end-to-end `when-supported`.

## SC-PR-005 — Apply option precedence

Given defaults, project configuration, and CLI overrides, when a session starts, then effective options follow CLI > project > defaults and are recorded.

Verification: backend-boundary `when-supported`; frontend-integration `not-applicable`; end-to-end `when-supported`.

## SC-PR-006 — Cancel safely

Given a running analyzer, when the host cancels it, then the session is `cancelled`, target code is not executed, and no later completion is accepted.

Verification: backend-boundary `when-supported`; frontend-integration `not-applicable`; end-to-end `when-supported`.

## SC-PR-007 — Validate process frames

Given an external analyzer emitting stdout logs or an unknown frame, when the
host reads the stream, then it rejects the protocol violation and preserves
stderr as logs only.

Verification: backend-boundary `when-supported`; frontend-integration `not-applicable`; end-to-end `not-applicable`.

## SC-PR-008 — Load an explicitly supplied plugin descriptor

Given a local descriptor with a valid manifest and argv command, when the
descriptor is supplied to the host, then the process analyzer is registered
without executing the target project or scanning the machine for plugins.

Verification: backend-boundary `when-supported`; frontend-integration `not-applicable`; end-to-end `when-supported`.

## SC-PR-009 — Require manifest agreement during handshake

Given a process whose hello frame changes the descriptor manifest or declares
an unsupported API version, when a request starts, then the host rejects the
process before accepting a candidate or result.

Verification: backend-boundary `when-supported`; frontend-integration `not-applicable`; end-to-end `when-supported`.

## SC-PR-010 — Preserve protocol and terminal-state invariants

Given one detect or analyze request, when the process emits valid frames, then
the host accepts exactly one candidate or result followed by done or fatal,
retains streamed diagnostics, and rejects wrong request IDs, duplicate
terminal frames, late output, malformed payloads, or oversized lines.

Verification: backend-boundary `when-supported`; frontend-integration `not-applicable`; end-to-end `when-supported`.

## SC-PR-011 — Bound process output and lifecycle

Given an external analyzer that stalls, exceeds output limits, writes logs,
exits abnormally, or is cancelled, when the host ends the session, then the
child and pipes are cleaned up, stderr is bounded, no result is accepted
after cancellation, and the host returns a stable failure or cancelled
outcome.

Verification: backend-boundary `when-supported`; frontend-integration `not-applicable`; end-to-end `when-supported`.

## SC-PR-012 — Preserve external Python analysis parity

Given the representative Python fixture, when the external Python analyzer
and the in-process Python analyzer run with the same options, then their
normalized modules, relationships, references, evidence, diagnostics, and
partial/complete status agree apart from allowed analyzer/run provenance.

Verification: backend-boundary `when-supported`; frontend-integration `not-applicable`; end-to-end `when-supported`.

## SC-PR-013 — Use the unchanged shared product path

Given an explicitly loaded external Python descriptor, when a developer runs
analyze or open, then the result reaches canonical model normalization,
validation, projection, viewer/source inspection, and deterministic JSON,
HTML, and SVG output without a language-specific consumer branch.

Verification: backend-boundary `when-supported`; frontend-integration `when-supported`; end-to-end `when-supported`.
