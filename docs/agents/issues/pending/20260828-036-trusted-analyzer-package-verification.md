# 036 — Load and verify trusted analyzer packages

## Issue Metadata

- Issue number: 036
- Owning capability node: /.okf/capabilities/analyze-source/plugin-runtime/compiled-external-analyzer-distribution.md
- Artifact root: docs/architecture/analyze-source/plugin-runtime/compiled-external-analyzer-distribution/
- Issue file: docs/agents/issues/pending/20260828-036-trusted-analyzer-package-verification.md
- Category: feature
- Execution type: AFK
- Review gate: none
- Suggested state: ready-for-agent

## Parent artifacts

- docs/prd.md
- docs/architecture/application-architecture-summary.md
- docs/architecture/analyze-source/plugin-runtime/compiled-external-analyzer-distribution/prd.md
- docs/architecture/analyze-source/plugin-runtime/compiled-external-analyzer-distribution/canonical-domain-model.md
- docs/architecture/analyze-source/plugin-runtime/compiled-external-analyzer-distribution/canonical-use-cases.md
- docs/architecture/analyze-source/plugin-runtime/compiled-external-analyzer-distribution/canonical-api-cli-contract.md
- docs/architecture/analyze-source/plugin-runtime/compiled-external-analyzer-distribution/acceptance-scenarios.md
- docs/architecture/analyze-source/plugin-runtime/compiled-external-analyzer-distribution/readiness-review.md
- docs/agents/issues/pending/20260828-035-compiled-analyzer-distribution-assembly.md

## What to build

Implement the application-managed package index and verification boundary.
Read only the installation's `analyzers/index.json`, select the exact
normalized host platform, validate package paths and metadata, verify descriptor
and executable SHA-256 digests, and confirm descriptor manifest compatibility
before a package becomes available to the host.

Expose a safe listing/query path that reports package availability and
rejection reasons without launching any executable. Reject traversal,
symlink/substitution, missing, cross-platform, tampered, incompatible, or
duplicate packages with the stable error codes in the contract. A failed
packaged verification must not cause implicit in-process fallback.

## Acceptance criteria

- [ ] The loader validates `schema_version`, `host_api_version`, package
  uniqueness, logical IDs, semantic versions, languages, API versions,
  platform names, relative paths, and lowercase 64-character SHA-256 digests.
- [ ] Executable and descriptor paths normalize beneath the application-owned
  analyzer root and cannot escape through `..`, absolute paths, alternate
  separators, or symlink substitution.
- [ ] Selection accepts only the exact normalized host platform and returns
  `analyzer_package_not_found` or `analyzer_platform_unsupported` when the
  requested package is absent. It never chooses a nearest or cross-platform
  binary.
- [ ] Verification computes both file digests before launch, loads the
  descriptor, validates its manifest, and checks logical ID, version, language,
  API major, and descriptor/executable paths against the selected index entry.
- [ ] The package query/list path does not start a process, read the target
  repository for plugin descriptors, or mutate the registry when an entry is
  rejected.
- [ ] Stable errors cover malformed index, missing package, unsupported
  platform, integrity mismatch, manifest mismatch, and API incompatibility.
  Error details identify the package and failing validation without exposing
  uncontrolled command text.
- [ ] A verified package can be handed to the existing process adapter, while
  an invalid package is rejected before `exec.Cmd.Start` or equivalent launch.
- [ ] Focused tests cover valid packages, duplicate entries, traversal,
  symlink substitution, missing files, descriptor tampering, executable
  tampering, manifest mismatch, API mismatch, and exact platform behavior.

## Artifact sync required

- Application PRD: none. The trust policy, package listing behavior, and
  failure outcomes are already part of the confirmed future workflow.
- Application architecture summary: none when the application-managed trust
  boundary is implemented as specified; required if package verification or
  discovery crosses that boundary.
- Owning capability node/artifacts: required: `/.okf/capabilities/analyze-source/plugin-runtime/compiled-external-analyzer-distribution.md`; its orchestration status; and the plugin-runtime implementation slice with verification evidence.
- Issue registry: required; keep this issue linked from the owning node until
  closeout.
- OKF log: required for the delivery record.
- Reason/no-impact decision: delivery truth changes. Product scope and the
  canonical analyzer/model contract do not change.

## Human review gate

None. The issue is a backend trust and package-query boundary with no rendered
UI change.

## Blocked by

Blocked by `docs/agents/issues/pending/20260828-035-compiled-analyzer-distribution-assembly.md`.

## Artifact anchors

- CED-FR-003, CED-FR-004, CED-FR-005, CED-FR-006, and CED-FR-009.
- `VerifyAnalyzerPackage`, `ListAvailableAnalyzers`, `PlatformTarget`,
  `TrustPolicy`, and the `indexed -> rejected/unavailable` lifecycle.
- The stable error table in the canonical API/CLI contract.

## Acceptance scenarios addressed

- SC-CED-002 — Select the exact host platform
- SC-CED-003 — Verify a packaged executable before launch
- SC-CED-004 — Reject a tampered package
- SC-CED-008 — Keep discovery inside the application boundary

## User stories addressed

The compiled-distribution PRD has no numbered user-story section. This issue
supports parent stories US-PR-001 and US-PR-002 by making packaged analyzer
availability safe and inspectable.

## Verification obligations

- Policy source: `/.okf/project.md`

| Scenario | Backend boundary | Frontend integration | End-to-end journey |
| --- | --- | --- | --- |
| SC-CED-002/003/004 | when-supported: index, platform, path, digest, manifest, and pre-launch tests | not-applicable | when-supported: verified package handed to the existing process host |
| SC-CED-008 | when-supported: no-target-repository-discovery and no-execution listing tests | not-applicable | when-supported: application startup/listing against a repository containing a deceptive descriptor |

## Handoff

Issue 037 can use the verified package query and selection boundary to make
packaged execution the default while preserving explicit development modes.
