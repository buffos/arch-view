# 001 — Analyzer host and Go project selection

Execution type: AFK
Review gate: none
Status: done

## Parent PRD

docs/architecture/analyze-source/plugin-runtime/prd.md

## What to build

Implement the first in-process analyzer host path and register the built-in Go analyzer boundary.

- Provide a manifest registry with API-version compatibility validation.
- Expose deterministic analyzer listing and explicit or automatic selection.
- Build the Go project-boundary request from a repository root, including go.mod and ambiguous go.work handling.
- Resolve the declared Go options and cancellation context into the analyzer request.
- Return stable complete, partial, invalid-selection, unsupported, fatal, and cancelled status outcomes through the parent analysis command boundary.
- Keep the host responsible for selection, option precedence, safety, and result validation; keep source parsing in the Go adapter.

This issue may use a test or minimal analyzer implementation to prove the host boundary. Package discovery and import extraction belong to issue 002.

## Acceptance criteria

- [x] arch-view analyzers lists compatible manifests in deterministic order and rejects incompatible API versions.
- [x] Explicit language selection wins; automatic selection succeeds only for an unambiguous Go project and reports ambiguity or unsupported projects clearly.
- [x] A single go.mod is selected as the project boundary; a multi-module go.work requires an explicit module selection.
- [x] Analyzer options follow CLI, project configuration, then manifest-default precedence, and the request carries cancellation.
- [x] Host failures and partial outcomes map to the specified status vocabulary and exit-code behavior without panics escaping the boundary.
- [x] The host never executes target Go code and passes the analyzer's declared source-scope policy through without silently broadening it; Go-specific exclusions are implemented by issue 002.

## Artifact sync required

- Application PRD: none — this realizes the already specified first analyzer path.
- Application architecture summary: none — the existing plugin-manager and analyzer boundary is unchanged.
- Owning capability node/artifacts: required: .okf/capabilities/analyze-source.md; .okf/capabilities/analyze-source/plugin-runtime.md; .okf/capabilities/analyze-source/go-analysis.md; docs/architecture/analyze-source/orchestration-status.md; docs/architecture/analyze-source/plugin-runtime/orchestration-status.md; docs/architecture/analyze-source/go-analysis/orchestration-status.md; docs/architecture/analyze-source/go-analysis/implementation-slice.md.
- Issue registry: required; node issues reference required: .okf/capabilities/analyze-source.md; .okf/capabilities/analyze-source/plugin-runtime.md; .okf/capabilities/analyze-source/go-analysis.md.
- Reason/no-impact decision: delivery truth is being added; product and architecture truth require no change because the contract and boundary already exist.

## Blocked by

None.

## User stories addressed

- US-PR-001
- US-PR-002
- US-GO-001

## Contract and scenario trace

- Contract: docs/architecture/analyze-source/plugin-runtime/canonical-api-cli-contract.md; docs/architecture/analyze-source/canonical-api-cli-contract.md; docs/architecture/analyze-source/go-analysis/canonical-api-cli-contract.md
- Scenarios: SC-PR-001, SC-PR-002, SC-PR-003, SC-PR-004, SC-PR-005, SC-PR-006, SC-GO-001, SC-GO-003

## Scenario traceability

| Source rule or use case | Acceptance scenario | Issue criterion | Executed verification |
|---|---|---|---|
| PR-FR-001; AnalyzerRegistryService.RegisterAnalyzer | SC-PR-001, SC-PR-002 | Compatible manifests are listed; malformed, duplicate, and incompatible manifests are rejected | `go test ./...` — registry and manifest tests passed |
| PR-FR-002; AnalyzerRegistryService.DetectAnalyzers | SC-PR-003, SC-PR-004 | Auto-detection chooses one unique highest-confidence candidate and rejects ties | `go test ./...` — host selection tests passed |
| PR-FR-003; AnalyzerRegistryService.SelectAnalyzer | SC-AS-003, SC-AS-004 | Explicit language/ID selection wins and incompatible roots return stable errors | `go test ./...`; CLI unsupported-root smoke test returned exit 3 |
| PR-FR-004; AnalyzerExecutionService.ResolveAnalyzerOptions | SC-PR-005 | Effective values follow CLI > project > defaults and receive a deterministic fingerprint | `go test ./...` — option precedence test passed |
| PR-FR-005; AnalyzerExecutionService.RunAnalyzer | SC-PR-006, SC-AS-007 | Cancellation is terminal; panic and analyzer failures do not escape the host | `go test -race ./...` — cancellation and panic tests passed |
| PR-FR-006; AnalyzerExecutionService.ValidateAnalyzerResult | SC-AS-001, SC-AS-002 | Analyzer provenance, result status, relationship targets, and evidence references are validated | `go test ./...` — invalid-result test passed |
| GO-FR-001; DetectGoProject and SelectGoModule | SC-GO-001, SC-GO-003 | go.mod is selected and multi-module go.work requires explicit module selection | `go test ./...`; CLI analysis smoke test produced the Go boundary result |
| AS-FR-007; deterministic host normalization | SC-AS-008 | Stable run identity, options fingerprint, collection shape, and ordering are produced | `go test ./...`; `go run ./cmd/arch-view analyzers` produced deterministic manifest JSON |
| AS-FR-008; static safety policy | SC-AS-006 | The host never executes target code and passes adapter scope policy without broadening it | `go vet ./...`; `go test -race ./...`; Go analyzer uses filesystem reads only |

## Verification summary

- `go test ./...` passed.
- `go test -race ./...` passed.
- `go vet ./...` passed.
- `go run ./cmd/arch-view analyzers` listed the compatible Go manifest.
- `go run ./cmd/arch-view analyze --project . --language go --include-tests --output - --format analysis-json` returned a valid partial result with the selected go.mod boundary and empty arrays for not-yet-implemented observations.
