# 034 — Port existing analyzers to compiled plugin entrypoints

## Issue Metadata

- Issue number: 034
- Owning capability node: /.okf/capabilities/analyze-source/plugin-runtime/compiled-external-analyzer-distribution.md
- Supporting capability nodes: /.okf/capabilities/analyze-source/go-analysis.md; /.okf/capabilities/analyze-source/python-analysis.md; /.okf/capabilities/analyze-source/typescript-analysis.md; /.okf/capabilities/analyze-source/rust-analysis.md; /.okf/capabilities/analyze-source/clojure-compatibility.md
- Artifact root: docs/architecture/analyze-source/plugin-runtime/compiled-external-analyzer-distribution/
- Issue file: docs/agents/issues/done/20260828-034-compiled-analyzer-plugin-entrypoints.md
- Category: feature
- Execution type: AFK
- Review gate: none
- Suggested state: done

## Parent artifacts

- docs/prd.md
- docs/architecture/application-architecture-summary.md
- docs/architecture/analyze-source/plugin-runtime/compiled-external-analyzer-distribution/prd.md
- docs/architecture/analyze-source/plugin-runtime/compiled-external-analyzer-distribution/canonical-domain-model.md
- docs/architecture/analyze-source/plugin-runtime/compiled-external-analyzer-distribution/canonical-use-cases.md
- docs/architecture/analyze-source/plugin-runtime/compiled-external-analyzer-distribution/canonical-api-cli-contract.md
- docs/architecture/analyze-source/plugin-runtime/compiled-external-analyzer-distribution/acceptance-scenarios.md
- docs/architecture/analyze-source/plugin-runtime/compiled-external-analyzer-distribution/readiness-review.md
- docs/architecture/analyze-source/plugin-runtime/external-protocol-v1.schema.json
- docs/architecture/analyze-source/plugin-runtime/external-plugin-descriptor-v1.schema.json
- docs/architecture/analyze-source/plugin-runtime/implementation-slice.md

## What to build

Port the five existing in-process analyzers to compiled external plugin
entrypoints without creating a second implementation of any language
semantics. Add one shared process runner around `analysis.Analyzer` that uses
the published NDJSON protocol, then add compiled commands for Go, Python,
TypeScript, Rust, and Clojure that construct the existing `New()` analyzer.

Each command must expose the same logical analyzer ID, version, language,
manifest, options, detection behavior, and analysis result as its in-process
counterpart. The compiled process is the production implementation candidate;
the in-process registry remains available for development, tests, and the
explicit migration fallback. Distribution-tree assembly and packaged runtime
selection belong to issues 035 and 037.

## Acceptance criteria

- [x] A shared compiled-plugin runner accepts an `analysis.Analyzer`, emits a
  manifest hello frame, handles detect and analyze requests, forwards options
  without loss, emits diagnostics, and terminates with exactly one done or
  fatal frame.
- [x] Compiled entrypoints exist for `org.archview.go`,
  `org.archview.python`, `org.archview.typescript`, `org.archview.rust`, and
  `org.archview.clojure`, and each entrypoint constructs the existing analyzer
  implementation rather than duplicating language logic.
- [x] Every compiled entrypoint reports a manifest byte-for-byte compatible
  with its in-process analyzer after canonical JSON serialization, including
  logical ID, semantic version, language, API version, capabilities, markers,
  and options.
- [x] The existing process adapter can invoke each compiled entrypoint for a
  representative detect and analyze operation and receive a valid result
  through the existing host validation path.
- [x] Compiled analysis results match the corresponding in-process results for
  representative fixtures after removing only allowed run and runtime
  provenance. Modules, relationships, references, evidence, diagnostics,
  status, and option fingerprints remain equivalent.
- [x] The runner uses argv and the existing process protocol, writes protocol
  frames only to stdout, keeps runtime logs on stderr, and preserves the
  existing cancellation, timeout, frame-size, and child-cleanup rules.
- [x] The five in-process analyzers, their registry listing, existing external
  Python pilot, and all current tests remain unchanged in behavior.

## Artifact sync required

- Application PRD: none. This implements the already confirmed compiled
  analyzer direction and adds no actor, workflow, MVP, or non-goal.
- Application architecture summary: none when the shared runner and compiled
  entrypoints match the existing process boundary; required if implementation
  evidence exposes a boundary mismatch.
- Owning capability node/artifacts: required: `/.okf/capabilities/analyze-source/plugin-runtime/compiled-external-analyzer-distribution.md`; its orchestration status; the plugin-runtime implementation slice; and supporting analyzer implementation records when they gain compiled-entrypoint evidence.
- Issue registry: required; the owning node `issues:` reference must point to
  this pending file until closeout.
- OKF log: required for delivery creation and later completion evidence.
- Reason/no-impact decision: delivery truth changes. Product and architecture
  truth remain unchanged because the runner reuses the published protocol and
  existing analyzer semantics.

## Human review gate

None. This is a compiled process and protocol issue with no rendered UI change.

## Blocked by

None. The published protocol, descriptor schema, process adapter, and current
analyzers are already available.

## Artifact anchors

- CED-FR-001, CED-FR-005, CED-FR-007, and CED-FR-010.
- `AnalyzerDistribution`, `PackageManifest`, `RuntimeSelection`, and the
  one-process/one-operation lifecycle in the canonical domain model.
- The existing `internal/analysis/processprotocol` and
  `internal/analysis/processanalyzer` packages.
- The five analyzer manifests and constructors under `internal/analyzers/`.

## Acceptance scenarios addressed

- SC-CED-003 — Verify a packaged executable before launch
- SC-CED-005 — Require manifest agreement
- SC-CED-006 — Avoid language-runtime installation
- SC-CED-009 — Preserve analyzer parity

## User stories addressed

The compiled-distribution PRD has no numbered user-story section. This issue
supports parent stories US-PR-001, US-PR-002, and US-PR-003 through the
compiled implementation of the existing analyzer contract.

## Verification obligations

- Policy source: `/.okf/project.md`

| Scenario | Backend boundary | Frontend integration | End-to-end journey |
| --- | --- | --- | --- |
| SC-CED-003/005 | when-supported: runner handshake, manifest, request, result, and terminal-state tests | not-applicable | when-supported: compiled command through the existing process adapter |
| SC-CED-006 | when-supported: run compiled analyzers with language runtimes unavailable | not-applicable | when-supported: representative repository analysis |
| SC-CED-009 | when-supported: per-analyzer parity fixtures and deterministic normalization | not-applicable | when-supported: compiled and in-process results through the common host |

## Handoff

Issue 035 is unblocked and may assemble all five compiled entrypoints into the
release tree. Issues 036–038 remain ordered behind distribution assembly,
package trust, and packaged-runtime selection respectively.

## Scenario traceability

| Canonical rule or use case | Scenario | Acceptance criteria | Verification and closure evidence |
| --- | --- | --- | --- |
| CED-FR-001 stable logical analyzer identity; `AnalyzerDistribution` retains the same analyzer ID across runtime modes | SC-CED-005, SC-CED-009 | 2, 3, 5 | `cmd/arch-view/compiled_analyzers_test.go` builds and launches all five commands, compares manifests, and compares canonical results with only run/runtime provenance removed |
| CED-FR-005 manifest/API compatibility; `PackageManifest` uses the published analyzer API and manifest contract | SC-CED-003, SC-CED-005 | 1, 3, 4, 6 | `internal/analysis/processplugin/runner_test.go` validates hello/session frames; the compiled parity test invokes the existing process adapter and common host validation |
| CED-FR-010 one child per operation with existing NDJSON, cancellation, diagnostics, and terminal-state rules | SC-CED-003, SC-CED-005 | 1, 6 | Runner unit tests cover detect, analyze, streamed diagnostics, cancellation, fatal output, and exactly one terminal frame; existing `processanalyzer` conformance tests remain green |
| `LaunchVerifiedAnalyzerOperation` delegates execution through the language-neutral analyzer contract without changing language semantics | SC-CED-003, SC-CED-009 | 1, 2, 4, 5, 7 | Each command calls the existing analyzer `New()` constructor; the five compiled fixtures pass detect/analyze and in-process parity through `analysis.Host` |
| Package assembly, integrity verification, and packaged-runtime default are downstream lifecycle steps, not reimplemented here (CED-FR-002/003/004/006/007/008/009) | SC-CED-003, SC-CED-006, SC-CED-009 | 2, 5, 6 | Issue 035 owns assembly/indexing, issue 036 owns trust verification, and issue 037 owns runtime selection; this issue supplies the compiled executable candidates those steps consume |

## Implementation and verification

### Implementation

- Added `internal/analysis/processplugin/runner.go`, a shared child-side
  runner for the published NDJSON protocol. It validates the manifest,
  emits hello, forwards detect/analyze requests, canonicalizes declared
  `string[]` option values after JSON decoding, streams diagnostics, handles
  cancellation, and emits one terminal done/fatal frame.
- Added compiled entrypoints at
  `cmd/analyzers/{go,python,typescript,rust,clojure}/main.go`. Each entrypoint
  constructs the existing analyzer with its `New()` function and delegates
  all protocol work to the shared runner.
- Added `internal/analysis/processplugin/runner_test.go` and
  `cmd/arch-view/compiled_analyzers_test.go` for protocol behavior, option
  forwarding, cancellation, compiled process invocation, manifest agreement,
  and five-language parity.

### Verification

Passed on 2026-08-28:

- `go test ./... -count=1`
- `go test ./cmd/arch-view -run TestCompiledAnalyzerEntrypointsMatchInProcessAnalyzers -count=1 -v`
- `go vet ./...`
- `git diff --check`
- `node C:\Users\buffo\.agents\skills\packages\planning\skills\okf-validate\scripts\okf-validate.mjs .okf --strict --json` — conformant, zero errors, zero warnings

The root `when-supported` policy is satisfied for the applicable backend and
end-to-end surfaces. Frontend integration is `not-applicable`: issue 034
does not change the language-neutral model, viewer, or export surfaces.

### Artifact sync and no-impact decision

The compiled-distribution node, its orchestration status, the plugin-runtime
implementation slice, supporting analyzer records, the Analyze source rollup,
the issue registry, and `.okf/log.md` record this completed migration step.
The application PRD and application architecture summary require no semantic
change: the runner reuses the published protocol and existing analyzer
implementations, and package assembly/runtime selection remain in issues
035–037. The compiled-distribution node remains `specified` because issues
035–038 are still required to complete the scoped capability.
