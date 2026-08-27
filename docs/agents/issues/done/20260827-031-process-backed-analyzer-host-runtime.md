# 031 — Process-backed analyzer host runtime

## Issue Metadata

- Issue number: 031
- Owning capability node: /.okf/capabilities/analyze-source/plugin-runtime.md
- Artifact root: docs/architecture/analyze-source/plugin-runtime/
- Issue file: docs/agents/issues/done/20260827-031-process-backed-analyzer-host-runtime.md
- Category: feature
- Execution type: AFK
- Review gate: none
- Suggested state: done

## Parent Artifacts

- docs/architecture/analyze-source/plugin-runtime/prd.md
- docs/architecture/analyze-source/plugin-runtime/canonical-domain-model.md
- docs/architecture/analyze-source/plugin-runtime/canonical-use-cases.md
- docs/architecture/analyze-source/plugin-runtime/canonical-api-cli-contract.md
- docs/architecture/analyze-source/plugin-runtime/acceptance-scenarios.md
- docs/architecture/analyze-source/plugin-runtime/readiness-review.md
- docs/architecture/analyze-source/plugin-runtime/implementation-slice.md
- docs/architecture/analyze-source/plugin-runtime/external-protocol-v1.schema.json
- docs/architecture/analyze-source/plugin-runtime/external-plugin-descriptor-v1.schema.json
- docs/agents/issues/done/20260827-030-external-protocol-schema-and-conformance-fixture.md

## What to build

Implement a process-backed analysis.Analyzer adapter under the analysis
boundary. It receives a validated external-plugin descriptor, exposes the
descriptor manifest without launching a process, and runs one child process
for each Detect or Analyze call using the issue 030 protocol codec and
fixture.

The adapter must preserve the existing host semantics: the host still
selects analyzers, resolves option precedence, owns cancellation and result
validation, and normalizes deterministic collections. The process adapter
only translates the Analyzer calls to frames and translates valid responses
back to the existing Go types.

## Acceptance criteria

- [x] A validated descriptor constructs a process-backed Analyzer whose
  Manifest method is deterministic and does not execute the command. Invalid
  manifests, unsupported API versions, empty commands, duplicate options, and
  unsafe descriptor values are rejected before registration.
- [x] Detect launches the command with argv arguments, reads hello first,
  verifies the protocol/API version and exact manifest agreement, sends one
  detect frame, accepts one candidate, requires done, and returns the
  existing DetectionCandidate type.
- [x] Analyze forwards the normalized project root, selection, and effective
  options in one analyze frame, retains streamed diagnostics, accepts one
  canonical result, requires done, and leaves final result validation to the
  common Host path.
- [x] Process launch never invokes a shell or concatenates untrusted
  descriptor values into a shell command. The child working directory and
  argument resolution are deterministic and remain under descriptor control.
- [x] Every process reaches a terminal cleanup path. Context cancellation or
  deadline sends best-effort cancel, terminates the child, waits for it, and
  returns the existing cancelled host outcome; no result arriving after
  cancellation is accepted.
- [x] Handshake and operation reads enforce the documented frame-size and
  bounded stderr limits. Timeout, non-zero exit, malformed JSON, unknown
  frames, wrong request IDs, missing result/candidate, duplicate terminal
  frames, hello mismatch, and invalid status map to stable host errors with
  useful details.
- [x] Stderr is retained only as bounded diagnostic context and never
  contaminates protocol parsing or canonical analysis output. Stdout
  protocol violations fail closed.
- [x] Conformance tests pass with every issue 030 fixture mode, including
  valid complete/partial results, fatal errors, delayed output,
  cancellation, oversized frames, and stdout contamination.
- [x] Existing in-process analyzer tests and all common host result,
  selection, option, cancellation, and deterministic-ordering tests remain
  green; no public CLI flag is added in this issue.

## Artifact sync required

- Application PRD: none — the process adapter realizes the already confirmed
  plugin boundary; issue 033 owns the public workflow and sequence refresh.
- Application architecture summary: none unless implementation changes the
  agreed process boundary; record any such mismatch before continuing to
  issue 032.
- Owning capability artifacts: required:
  /.okf/capabilities/analyze-source/plugin-runtime.md;
  docs/architecture/analyze-source/plugin-runtime/orchestration-status.md;
  docs/architecture/analyze-source/plugin-runtime/canonical-api-cli-contract.md;
  docs/architecture/analyze-source/plugin-runtime/acceptance-scenarios.md;
  docs/architecture/analyze-source/plugin-runtime/readiness-review.md.
- Issue registry: required; the owning node issues list and blocker status
  must remain synchronized.
- Reason/no-impact decision: process lifecycle and safety become implemented
  behavior, but product actors, model vocabulary, viewer behavior, and
  built-in analyzer registration remain unchanged.

## Human review gate

None. This issue changes backend process orchestration only and has no
rendered UI/UX change.

## Blocked by

None — issue 030 is complete and its protocol/fixture artifacts are available.

## Artifact anchors

- Canonical API/CLI contract: descriptor loading, external NDJSON frame
  ordering, hello agreement, process limits, stdout/stderr policy, and
  compatibility rules.
- Canonical use cases: RegisterProcessAnalyzer, RunProcessDetection,
  RunProcessAnalysis, and ValidateProcessProtocol.
- Canonical domain model: ProcessPluginDescriptor, ProcessInvocation,
  ProtocolSession, and AnalyzerSession terminal-state invariants.
- Existing Go host: analysis.Analyzer, analysis.Host, Registry, options, and
  ValidateAnalysisResult.

## Acceptance scenarios addressed

- SC-PR-001 — Register a compatible analyzer
- SC-PR-002 — Reject incompatible manifests
- SC-PR-003 — Resolve unambiguous detection
- SC-PR-004 — Stop on ambiguity
- SC-PR-005 — Apply option precedence
- SC-PR-006 — Cancel safely
- SC-PR-007 — Validate process frames
- SC-PR-009 — Require manifest agreement during handshake
- SC-PR-010 — Preserve protocol and terminal-state invariants
- SC-PR-011 — Bound process output and lifecycle

## Verification obligations

- Policy source: /.okf/project.md

| Scenario | Backend boundary | Frontend integration | End-to-end journey |
|---|---|---|---|
| SC-PR-001/002 | passed: descriptor construction, manifest/API validation, duplicate-option, and unsafe-value tests in `internal/analysis/processanalyzer` | not-applicable | not-applicable |
| SC-PR-003/004 | passed: fixture detection and common-host selection tests in `internal/analysis/processanalyzer` | not-applicable | not-applicable |
| SC-PR-005 | passed: effective option forwarding and streamed diagnostic merge tests | not-applicable | not-applicable |
| SC-PR-006/011 | passed: cancellation, timeout, frame/stderr limits, and child-cleanup tests | not-applicable | not-applicable |
| SC-PR-007/009/010 | passed: protocol conformance tests covering all issue 030 fixture modes | not-applicable | not-applicable |

## Implementation and verification

- Added `internal/analysis/processanalyzer`, an argv-only process-backed
  `analysis.Analyzer` with descriptor loading, hello/manifest agreement,
  bounded NDJSON and stderr handling, streamed diagnostic merge, cancellation,
  timeout, and terminal cleanup.
- Added focused conformance coverage for the valid, partial, fatal, delayed,
  cancellation, oversized, malformed, stdout-contamination, mismatch, and
  late-frame fixture behaviors.
- Verification passed: `go test ./internal/analysis/processanalyzer -count=1`,
  `go test ./... -count=1`, `go vet ./...`, and `git diff --check`.

## Handoff

Issue 032 may begin after the adapter can run the fixture and pass the full
process conformance suite. The external Python plugin should be tested
through this adapter rather than by adding Python-specific behavior to the
host.
