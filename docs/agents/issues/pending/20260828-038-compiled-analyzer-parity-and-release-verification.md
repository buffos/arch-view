# 038 — Verify compiled analyzer parity and release behavior

## Issue Metadata

- Issue number: 038
- Owning capability node: /.okf/capabilities/analyze-source/plugin-runtime/compiled-external-analyzer-distribution.md
- Artifact root: docs/architecture/analyze-source/plugin-runtime/compiled-external-analyzer-distribution/
- Issue file: docs/agents/issues/pending/20260828-038-compiled-analyzer-parity-and-release-verification.md
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
- docs/agents/issues/pending/20260828-037-packaged-runtime-selection.md

## What to build

Close the compiled-distribution capability with release-matrix, parity, and
safety verification. Exercise the five packaged analyzers through the public
runtime path on every supported platform where the runner exists, compare
their results with the in-process implementations, and prove that package
tampering, missing runtimes, deceptive project-local descriptors, and
cross-platform artifacts fail safely.

Keep verification tied to the published acceptance scenarios. If a platform
runner is unavailable in the current environment, record the deferred
verification and its re-entry condition under the root `when-supported`
policy. Do not weaken the package contract to make a local platform pass.

## Acceptance criteria

- [ ] The supported build matrix produces and validates complete packages for
  Go, Python, TypeScript, Rust, and Clojure on `windows-amd64`,
  `linux-amd64`, and `darwin-arm64`, or records a justified platform-specific
  verification deferral.
- [ ] Packaged and in-process runs for each analyzer agree on normalized
  modules, relationships, references, source evidence, diagnostics, status,
  options, and downstream canonical model meaning apart from runtime
  provenance.
- [ ] Repeated packaged builds and analysis runs with unchanged inputs are
  byte-stable where the contracts require determinism.
- [ ] Tampering with either a descriptor or executable is detected before
  launch and returns `analyzer_package_integrity_mismatch` without implicit
  fallback.
- [ ] Missing package, unsupported platform, manifest mismatch, API mismatch,
  malformed index, and launch failure produce the documented stable errors.
- [ ] A release package runs without the analyzer language runtimes installed,
  and the analyzer still returns its own complete, partial, or diagnostic
  outcome through the common protocol.
- [ ] A target repository containing a plausible descriptor or executable does
  not cause discovery or execution unless the explicit developer/test override
  is supplied.
- [ ] Existing external protocol conformance tests, built-in analyzer tests,
  model normalization, viewer/source, JSON/HTML/SVG export, and deterministic
  output tests remain green.
- [ ] The compiled-distribution orchestration record contains scenario-linked
  verification evidence, any justified deferrals, and the final artifact-sync
  status before the node can be considered implemented.

## Artifact sync required

- Application PRD: none when the verified behavior matches the existing future
  direction; required if parity or release behavior changes actors, workflows,
  business rules, MVP, roadmap, or non-goals.
- Application architecture summary: none when package trust, runtime
  selection, and the process boundary match the approved architecture;
  required if verification exposes a boundary mismatch.
- Owning capability node/artifacts: required: `/.okf/capabilities/analyze-source/plugin-runtime/compiled-external-analyzer-distribution.md`; its orchestration status; the plugin-runtime implementation slice; and any supporting analyzer records that carry parity evidence.
- Issue registry: required; this issue remains linked until closeout and is
  removed from the active registry only during closeout.
- OKF log: required for verification and final state evidence.
- Reason/no-impact decision: verification completes delivery truth. Product
  and architecture truth remain unchanged when all tests confirm the approved
  contracts.

## Human review gate

None. The capability has no new rendered UI. Existing viewer/export regression
coverage remains automated and reuses the unchanged consumer contracts.

## Blocked by

Blocked by `docs/agents/issues/pending/20260828-037-packaged-runtime-selection.md`.

## Artifact anchors

- SC-CED-001 through SC-CED-009 and CED-FR-001 through CED-FR-010.
- The root `when-supported` verification policy in `/.okf/project.md`.
- The parity rules and residual platform/toolchain risks in the capability
  readiness review.
- Existing external Python parity and process-conformance tests as regression
  baselines.

## Acceptance scenarios addressed

- SC-CED-001 through SC-CED-009

## User stories addressed

The compiled-distribution PRD has no numbered user-story section. This issue
verifies the release-maintainer and Arch View user workflows and supports
parent stories US-PR-001, US-PR-002, and US-PR-003.

## Verification obligations

- Policy source: `/.okf/project.md`

| Scenario | Backend boundary | Frontend integration | End-to-end journey |
| --- | --- | --- | --- |
| SC-CED-001/002/003/004/005/006 | when-supported: release matrix, package verification, process launch, parity, and failure tests | not-applicable | when-supported: release package to analysis result |
| SC-CED-007/008 | when-supported: explicit override, no-fallback, and repository-boundary tests | when-supported: existing project-backed open path | when-supported: public CLI and open flows |
| SC-CED-009 | when-supported: all five analyzer parity fixtures and deterministic output checks | when-supported: existing model/viewer/export regression path | when-supported: packaged and in-process results through unchanged consumers |

## Handoff

After this issue passes, the owning node may move to `implemented` only after
the artifact synchronization contract is complete and the scoped work is
actually exhausted. The node must remain `specified` while any material
package, runtime, or parity gap remains.
