# Compiled external analyzer distribution discovery notes

## Purpose

Define how supported language analyzers become independently launchable,
versioned executable plugins without imposing a language runtime on Arch View
users.

## Current implementation

- **Observed in code:** The common `analysis.Analyzer` contract and the
  process-backed host already exist. The current external descriptor launches
  `python launcher.py`, while the Go, Python, TypeScript, Rust, and Clojure
  analyzers are registered in-process.
- **Inferred from docs:** The v1 process protocol already carries manifests,
  detection, analysis, diagnostics, cancellation, bounded output, and terminal
  status across an isolated child process.

## User-confirmed target behavior

- Each supported analyzer is distributable as a compiled executable plugin.
- The executable reuses the same analyzer implementation as the in-process
  adapter rather than maintaining a second language implementation.
- A user installation does not require Python, Rust, Node, or another analyzer
  language runtime.
- The host discovers trusted, version-compatible binaries through descriptors
  and launches them with the existing protocol.

## Specified decisions

- The release build compiles every analyzer and assembles an application-owned
  `analyzers/<logical-analyzer-id>/<platform>/` tree containing the executable
  and its descriptor. A root build/make target owns this assembly so the
  runtime layout is reproducible rather than hand-created.
- The host discovers descriptors only from its application-managed analyzer
  directory. The target repository is never scanned for executable plugins;
  explicitly supplied descriptors remain available for development and parity
  tests.
- A compiled artifact keeps the same logical analyzer ID as its in-process
  counterpart. The external implementation is preferred in production, while
  the in-process adapter remains available only for development, tests, or an
  explicit migration fallback.
- The descriptor and process hello must agree on logical ID, analyzer version,
  and supported API. The initial platform matrix is `windows-amd64`,
  `linux-amd64`, and `darwin-arm64`; descriptor and executable SHA-256
  digests are verified against the application-managed package index before a
  production launch. There is no remote signer/update service in this slice.
- Production defaults to the packaged runtime. Explicit descriptors and
  in-process adapters are available only through explicit developer/test or
  migration override modes; there is no silent fallback from a bad package.
- One process handles one detect or analyze job and exits. The existing
  cancellation, timeout, bounded stdout/stderr, and cleanup policy remains the
  process boundary.

## Implementation and verification focus

The exact-spec set defines the platform naming scheme, descriptor fields,
package integrity policy, build targets, and explicit runtime override switch.
Implementation and verification must now cover reproducible cross-platform
assembly, package-index validation, upgrade/rollback behavior, and the
developer/test override gate without reopening the compiled-executable
decision.

## Boundary

This frontier covers executable distribution and registration. Protocol
semantics remain owned by the parent plugin-runtime capability; language
semantics remain owned by each analyzer; multi-root execution is owned by the
multi-analyzer orchestration child.
