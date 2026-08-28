# Compiled external analyzer distribution requirements gap analysis

## Scope examined

This pass covers the bounded [compiled external analyzer distribution](../../../../.okf/capabilities/analyze-source/plugin-runtime/compiled-external-analyzer-distribution.md) child, the existing plugin-runtime contract, the current process adapter, the current Python descriptor, and the application synthesis documents.

## Confirmed strong areas

- The analyzer manifest and `arch-view.analyzer/v1` process protocol already define identity, capabilities, options, detection, analysis, diagnostics, cancellation, and terminal outcomes.
- **Observed in code:** the current adapter launches a descriptor command with argv, never through a shell, validates the hello manifest, bounds frames and stderr, and cleans up one child process per operation.
- **Observed in code:** built-in analyzers are registered in-process and the current Python external deployment is a script/interpreter pilot.
- **User-confirmed target behavior:** production analyzers are compiled executables reusing the same logical analyzer identity and implementation semantics, assembled by a reproducible root build target.
- **User-confirmed trust policy:** production discovery is limited to the application-managed analyzer tree; executable integrity is checked; arbitrary project-local plugins are never discovered automatically.

## Blocking gaps resolved

| Gap | Impact | Resolution |
|---|---|---|
| Trust boundary for executable discovery was not exact | High: different implementations could execute arbitrary project-local code | The host reads only the application-owned analyzer index and verifies the packaged descriptor and executable digest before launch. Explicit developer/test descriptors require an opt-in override. |
| Platform selection and missing-platform behavior were unspecified | High: releases could silently select the wrong binary or require a language runtime | Select the exact normalized host platform; fail clearly when no package exists. The initial release matrix is `windows-amd64`, `linux-amd64`, and `darwin-arm64`; the schema is extensible. |
| Package/version agreement was unspecified | High: a descriptor, hello frame, and executable could disagree | The index, descriptor, hello manifest, logical ID, semantic version, language, and API major must agree; incompatible or tampered packages fail before analysis. |
| Build assembly was only a direction | Medium: hand-created output would drift from release expectations | `make analyzers` produces every analyzer package and the index; `make release` assembles the application plus the selected platform analyzer tree. |
| In-process fallback could become silently implicit | High: production behavior would vary after a packaging failure | Packaged external execution is the default. In-process execution is available only through an explicit runtime mode; package failure never silently falls back. |

## Deferrable implementation details

- The exact compiler flags, CI runner matrix, installer format, and code-signing integration can vary without changing the contract. Reproducible inputs, index generation, and digest verification remain required.
- Incremental package download, hot update, rollback, and third-party package registries are outside this capability.

## Assumptions carried into the exact specification

- SHA-256 is the required executable and descriptor digest algorithm in the first package-index version.
- The application installation or release archive is the trust anchor for `analyzers/index.json`; a remote signature or update service is not required in this slice.
- The host loads the analyzer index at startup and does not hot-reload packages during a running session.

## Readiness conclusion

No High or Medium specification gaps remain after the trust-boundary decision. The remaining work is exact artifact definition and implementation verification, not another product clarification round.

## Artifact impact

- **Capability:** this report and the linked exact-spec set define package, build, selection, trust, and fallback behavior.
- **Product:** the application PRD remains synchronized; no new actor or user workflow is introduced beyond runtime availability and safe failure.
- **Architecture:** the application architecture summary must retain the app-managed package boundary, executable verification, and explicit fallback rule.
- **Delivery:** no issues are created in this specification pass.
