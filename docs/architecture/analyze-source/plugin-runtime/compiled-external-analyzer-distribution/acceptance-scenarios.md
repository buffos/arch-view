# Compiled external analyzer distribution acceptance scenarios

## SC-CED-001 — Assemble the complete analyzer tree

**Given** the supported analyzer set and a supported platform, **when** the
release maintainer runs `make analyzers`, **then** every enabled analyzer has
an executable, descriptor, digest metadata, and one matching index entry, or
the build fails without publishing a partial index.

Verification: backend-boundary `when-supported`; frontend-integration
`not-applicable`; end-to-end `when-supported`.

## SC-CED-002 — Select the exact host platform

**Given** packages for multiple platforms, **when** the host lists or selects
an analyzer, **then** it chooses only the exact normalized host platform and
never runs a package built for another platform.

Verification: backend-boundary `when-supported`; frontend-integration
`not-applicable`; end-to-end `when-supported`.

## SC-CED-003 — Verify a packaged executable before launch

**Given** an index entry whose descriptor and executable match their SHA-256
digests, **when** a detect or analyze operation starts, **then** the package is
verified before the child process is launched and the operation uses the
existing analyzer protocol.

Verification: backend-boundary `when-supported`; frontend-integration
`not-applicable`; end-to-end `when-supported`.

## SC-CED-004 — Reject a tampered package

**Given** a package whose descriptor or executable bytes differ from the
indexed digest, **when** the host selects it, **then** it returns
`analyzer_package_integrity_mismatch`, launches no process, and does not fall
back to in-process execution.

Verification: backend-boundary `when-supported`; frontend-integration
`not-applicable`; end-to-end `when-supported`.

## SC-CED-005 — Require manifest agreement

**Given** a verified executable whose hello manifest disagrees with the index
or descriptor, **when** the host begins an operation, **then** it rejects the
process before accepting a result and reports the stable manifest-mismatch
outcome.

Verification: backend-boundary `when-supported`; frontend-integration
`not-applicable`; end-to-end `when-supported`.

## SC-CED-006 — Avoid language-runtime installation

**Given** a release package containing a compiled analyzer, **when** a user
runs analysis on a matching platform without the analyzer language runtime
installed, **then** the packaged analyzer completes or returns its own analysis
diagnostic without requiring that runtime.

Verification: backend-boundary `when-supported`; frontend-integration
`not-applicable`; end-to-end `when-supported`.

## SC-CED-007 — Make fallback explicit

**Given** a packaged analyzer is missing or fails verification, **when** the
user has not requested an override, **then** analysis fails with a package
diagnostic; **when** the user explicitly selects in-process mode, **then** the
run records that mode and uses the in-process implementation.

Verification: backend-boundary `when-supported`; frontend-integration
`when-supported`; end-to-end `when-supported`.

## SC-CED-008 — Keep discovery inside the application boundary

**Given** a target repository contains a descriptor or executable with an
apparently valid analyzer manifest, **when** the host opens the repository,
**then** it does not discover or execute that file unless the caller supplies
the explicit developer override.

Verification: backend-boundary `when-supported`; frontend-integration
`when-supported`; end-to-end `when-supported`.

## SC-CED-009 — Preserve analyzer parity

**Given** the same fixture and options, **when** the packaged and in-process
implementations of one logical analyzer run, **then** normalized observations,
diagnostics, status, and downstream model/viewer/export meaning agree apart
from runtime provenance.

Verification: backend-boundary `when-supported`; frontend-integration
`when-supported`; end-to-end `when-supported`.
