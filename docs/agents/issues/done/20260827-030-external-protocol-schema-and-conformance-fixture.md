# 030 — External protocol schema and conformance fixture

## Issue Metadata

- Issue number: 030
- Owning capability node: /.okf/capabilities/analyze-source/plugin-runtime.md
- Artifact root: docs/architecture/analyze-source/plugin-runtime/
- Issue file: docs/agents/issues/done/20260827-030-external-protocol-schema-and-conformance-fixture.md
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

## What to build

Publish and exercise the v1 external analyzer wire contract before adding
process execution to the host. Add the protocol and descriptor schemas,
typed frame/codec support in an analysis-owned package, and a test-only
external-process fixture that can emit valid and intentionally invalid
streams. The fixture must cover detection and analysis without becoming a
shipped or implicitly discovered analyzer.

Keep the protocol envelope separate from the canonical analysis result. The
existing result validator remains authoritative for modules, relationships,
references, source evidence, diagnostics, status, and deterministic ordering.

## Acceptance criteria

- [x] The protocol schema defines hello, detect, analyze, cancel, candidate,
  result, diagnostic, done, and fatal frames with the required fields,
  request identifiers, API version, terminal statuses, and canonical payload
  locations.
- [x] The descriptor schema defines schema_version, a complete validated
  manifest, an argv command, optional arguments, and an optional working
  directory; it does not accept a shell command string or arbitrary
  application callbacks.
- [x] The wire package encodes one JSON object per line, rejects malformed
  JSON, unknown frame types, missing required fields, invalid request IDs,
  duplicate terminal frames, and lines larger than the documented maximum.
- [x] A test-only process fixture supports detect and analyze operations and
  can produce a valid hello/candidate/result/done exchange, streamed
  diagnostics, fatal errors, stdout contamination, unknown frames, malformed
  JSON, wrong request IDs, oversized lines, and delayed output on command.
- [x] Fixture stderr is distinguishable from stdout protocol data, and the
  conformance tests prove that stdout logs are a protocol violation while
  stderr text is not parsed as a result frame.
- [x] The schema examples and fixture behavior document one-process/one-request
  lifecycle, first-frame hello, exactly one candidate or result, and one
  terminal done or fatal frame.
- [x] Existing in-process analyzers, registry behavior, model schemas, and
  CLI output remain unchanged because this issue does not register the
  external fixture publicly.

## Artifact sync required

- Application PRD: none — this issue publishes and tests the already confirmed
  external boundary; the public implementation sequence is refreshed by issue
  033.
- Application architecture summary: required only if the implementation
  reveals a boundary mismatch; otherwise issue 033 records the consolidated
  process-boundary status.
- Owning capability artifacts: required:
  /.okf/capabilities/analyze-source/plugin-runtime.md;
  docs/architecture/analyze-source/plugin-runtime/orchestration-status.md;
  docs/architecture/analyze-source/plugin-runtime/canonical-api-cli-contract.md;
  docs/architecture/analyze-source/plugin-runtime/acceptance-scenarios.md;
  docs/architecture/analyze-source/plugin-runtime/readiness-review.md.
- Schema artifacts: required:
  docs/architecture/analyze-source/plugin-runtime/external-protocol-v1.schema.json;
  docs/architecture/analyze-source/plugin-runtime/external-plugin-descriptor-v1.schema.json.
- Issue registry: required; the owning node issues list must reference the
  pending issue until closeout.
- Reason/no-impact decision: the wire contract is made executable, but no
  product workflow, canonical model field, language adapter, or viewer path
  changes in this foundation issue.

## Human review gate

None. This is a protocol and backend fixture issue with no rendered UI/UX
change.

## Blocked by

None - can start immediately.

## Artifact anchors

- Plugin-runtime PRD requirements PR-FR-001, PR-FR-005, PR-FR-007, and
  PR-FR-008 through PR-FR-010.
- Canonical contract section External NDJSON protocol v1 and the published
  protocol/descriptor schemas.
- Canonical domain model terms ProtocolSession, ProcessPluginDescriptor, and
  AnalyzerSession.
- The implementation slice protocol decisions and verification surfaces.

## Acceptance scenarios addressed

- SC-PR-002 — Reject incompatible manifests
- SC-PR-006 — Cancel safely
- SC-PR-007 — Validate process frames
- SC-PR-008 — Load an explicitly supplied plugin descriptor
- SC-PR-009 — Require manifest agreement during handshake
- SC-PR-010 — Preserve protocol and terminal-state invariants

## Scenario traceability

| Source rule or use case | Acceptance scenario | Issue criterion | Closure evidence |
| --- | --- | --- | --- |
| Register only a compatible manifest; reject an incompatible hello API or manifest before payload acceptance | SC-PR-002, SC-PR-009 | 1, 2, 3, 6 | `go test ./internal/analysis/processprotocol -count=1`: manifest/API validation and mismatch fixtures pass |
| Load only an explicit descriptor with argv command/arguments and no shell or callback surface | SC-PR-008 | 2, 6 | Focused descriptor decode tests and schema-shape test pass |
| One process handles one request; hello is first; request IDs match; exactly one candidate/result and one terminal frame are accepted | SC-PR-007, SC-PR-010 | 1, 3, 4, 6 | Focused stateful session and subprocess fixture tests pass for valid, wrong-ID, duplicate-terminal, late-output, unknown, malformed, and oversized streams |
| Cancellation is represented as a request-scoped protocol frame while child cleanup and late-result rejection remain host-runtime work | SC-PR-006 | 1, 4 | Focused cancel-frame, wait-for-cancel, and delayed-output fixture tests pass; child cleanup/no-late-result ownership is explicitly deferred to issue 031 |
| Stdout is protocol-only and stderr is bounded human-readable context, never result data | SC-PR-007, SC-PR-010 | 4, 5 | Focused fixture test proves stdout contamination fails decoding and stderr remains separate |

## Verification obligations

- Policy source: /.okf/project.md

| Scenario | Backend boundary | Frontend integration | End-to-end journey |
|---|---|---|---|
| SC-PR-002 | implemented: hello manifest/API compatibility tests pass | not-applicable | not-applicable |
| SC-PR-006 | implemented: cancel-frame, wait-for-cancel, and delayed-output tests pass; deferred: child cleanup and no-late-result host behavior owned by issue 031 | not-applicable | not-applicable |
| SC-PR-007 | implemented: protocol codec and subprocess conformance tests pass for valid/invalid frame classes | not-applicable | not-applicable |
| SC-PR-008 | implemented: strict descriptor decode/validation and schema-shape tests pass | not-applicable | not-applicable |
| SC-PR-009 | implemented: hello manifest/API-version and mismatch tests pass | not-applicable | not-applicable |
| SC-PR-010 | implemented: one-request/one-terminal-frame and oversized-line tests pass | not-applicable | not-applicable |

## Verification evidence

- `go test ./internal/analysis/processprotocol -count=1` — passed.
- `go test ./... -count=1` — passed for all repository packages.
- `go vet ./...` — passed.
- Both published JSON schemas parse successfully with PowerShell `ConvertFrom-Json`.
- The test-only fixture is confined to `_test.go` files and is launched only by the conformance tests; no public analyzer registration was added.

## Handoff

Issue 031 may begin only after the schema, frame codec, and fixture are
available. It should use the fixture as the process boundary test double
instead of inventing a second transport contract.
