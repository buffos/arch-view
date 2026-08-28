# 037 — Make packaged runtime selection the default

## Issue Metadata

- Issue number: 037
- Owning capability node: /.okf/capabilities/analyze-source/plugin-runtime/compiled-external-analyzer-distribution.md
- Artifact root: docs/architecture/analyze-source/plugin-runtime/compiled-external-analyzer-distribution/
- Issue file: docs/agents/issues/pending/20260828-037-packaged-runtime-selection.md
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
- docs/agents/issues/pending/20260828-036-trusted-analyzer-package-verification.md

## What to build

Integrate packaged analyzers into runtime selection and the public CLI. Add
the transport-neutral `packaged`, `in-process`, and `explicit` runtime modes,
make packaged execution the production default, and keep the selected runtime
and source observable in analysis provenance and diagnostics.

`arch-view analyzers`, `analyze`, and project-backed `open` must use the
application-managed verified package for normal runs. An explicit in-process
mode selects the existing built-in implementation. An explicit local
descriptor requires the developer/test opt-in and remains the only path for
untrusted descriptors. A missing or invalid package is terminal for packaged
selection and never silently falls back. The open-session reanalysis callback
must retain the selected runtime mode.

## Acceptance criteria

- [ ] `analyzers`, `analyze`, and project-backed `open` accept the documented
  analyzer runtime selection. Packaged mode is the default for normal release
  execution.
- [ ] Packaged selection resolves a stable logical analyzer ID through the
  verified application index and launches the selected executable through the
  existing process adapter and NDJSON protocol.
- [ ] `--analyzer-runtime in-process` explicitly selects the built-in analyzer
  for development, tests, or migration and records that choice in result
  provenance.
- [ ] Explicit local descriptors require `--analyzer-runtime explicit` and
  the documented untrusted-plugin opt-in. `--plugin` is rejected in packaged
  mode and target-repository descriptors are never discovered automatically.
- [ ] Missing, unavailable, tampered, incompatible, or failed packaged
  analyzers return the stable package/runtime error outcome and do not switch
  to an in-process implementation implicitly.
- [ ] Descriptor/index/hello logical ID, version, language, API major, and
  runtime source agree before a result is accepted. Existing host result
  validation, option precedence, cancellation, timeout, and cleanup remain
  authoritative.
- [ ] `open --project` retains the packaged or explicit runtime selection for
  reanalysis, cancellation, and error reporting. Model-only `open` remains
  independent of analyzer runtime selection.
- [ ] Analyzer listing reports packaged availability, platform, version, and
  runtime source without launching analyzers. Existing no-flag built-in
  behavior and the external Python pilot remain compatible with their explicit
  modes.
- [ ] Focused CLI, host, and open-session tests cover default packaged mode,
  explicit in-process mode, explicit descriptor mode, missing packages,
  rejected overrides, runtime provenance, and no silent fallback.

## Artifact sync required

- Application PRD: none when the implementation follows the already confirmed
  runtime workflows; required if the default or override behavior changes the
  documented product contract.
- Application architecture summary: none when the plugin manager remains the
  only runtime boundary and the model/viewer/export consumers remain neutral;
  required if that boundary changes.
- Owning capability node/artifacts: required: `/.okf/capabilities/analyze-source/plugin-runtime/compiled-external-analyzer-distribution.md`; its orchestration status; the plugin-runtime implementation slice; and the CLI/runtime records that document the selected mode.
- Issue registry: required; keep this issue linked from the owning node until
  closeout.
- OKF log: required for the delivery record.
- Reason/no-impact decision: delivery truth changes and the future runtime
  behavior becomes executable. The MVP scope, model schema, viewer contract,
  and export semantics remain unchanged when the approved boundary is kept.

## Human review gate

None. The public commands reuse existing viewer and export behavior without a
rendered UI or navigation change.

## Blocked by

Blocked by `docs/agents/issues/pending/20260828-036-trusted-analyzer-package-verification.md`.

## Artifact anchors

- CED-FR-005, CED-FR-007, CED-FR-008, CED-FR-009, and CED-FR-010.
- `SelectAnalyzerRuntime`, `LaunchVerifiedAnalyzerOperation`,
  `RuntimeSelection`, and the explicit override policy.
- The runtime selection and CLI mapping sections of the canonical contract.
- Existing `cmd/arch-view/analyze.go`, `cmd/arch-view/open.go`,
  `cmd/arch-view/plugins.go`, and `internal/analysis/host.go` boundaries.

## Acceptance scenarios addressed

- SC-CED-003 — Verify a packaged executable before launch
- SC-CED-005 — Require manifest agreement
- SC-CED-007 — Make fallback explicit
- SC-CED-008 — Keep discovery inside the application boundary

## User stories addressed

The compiled-distribution PRD has no numbered user-story section. This issue
supports parent stories US-PR-001, US-PR-002, and US-PR-003.

## Verification obligations

- Policy source: `/.okf/project.md`

| Scenario | Backend boundary | Frontend integration | End-to-end journey |
| --- | --- | --- | --- |
| SC-CED-003/005 | when-supported: runtime selection, package verification, hello agreement, and host validation tests | not-applicable | when-supported: packaged executable through analysis JSON |
| SC-CED-007/008 | when-supported: explicit override, no-fallback, and no-target-discovery tests | when-supported: existing project-backed open/reanalysis path | when-supported: packaged and explicit runs through the public CLI |

## Handoff

Issue 038 can run the full release matrix and parity gate after packaged
runtime selection is available from the public commands.
