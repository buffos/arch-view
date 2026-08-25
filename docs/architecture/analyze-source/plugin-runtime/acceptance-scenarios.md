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

Given a future external analyzer emitting stdout logs or an unknown frame, when the host reads the stream, then it rejects the protocol violation and preserves stderr as logs only.

Verification: backend-boundary `when-supported`; frontend-integration `not-applicable`; end-to-end `not-applicable`.
