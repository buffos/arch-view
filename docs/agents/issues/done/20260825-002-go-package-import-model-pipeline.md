# 002 — Go package/import analysis and canonical model pipeline

Execution type: AFK
Review gate: none
Status: done

## Parent PRD

docs/architecture/analyze-source/go-analysis/prd.md

## What to build

Implement the Go static analyzer and connect its observations to the canonical architecture model.

- Discover eligible Go packages and source files inside the selected module.
- Extract static imports with source file, line, and column evidence.
- Resolve project-local imports to stable Go package IDs.
- Retain standard-library, third-party, missing, cgo, and conditional targets as references or diagnostics.
- Honor module, build-tag, include-tests, include-generated, and exclusion options without executing the target repository.
- Normalize the result into the versioned arch-view.model/v1 envelope, validate relationship integrity, preserve evidence, and report complete or partial status.
- Derive deterministic cycles, feedback relationships, hierarchy projections, and dependency layers without removing canonical relationships.

## Acceptance criteria

- [x] A representative Go module produces deterministic package nodes with stable IDs based on module path and relative import path.
- [x] Local imports become directed depends_on relationships and retain source evidence; imports outside the module do not become invented local modules.
- [x] Default exclusions and explicit options change the analyzed scope predictably, including tests, generated files, vendor, external, and build-conditional files.
- [x] Unresolved or non-local imports produce references and diagnostics with partial status where appropriate, rather than fabricated dependencies.
- [x] The model pipeline emits and validates arch-view.model/v1 with sorted collections, normalized paths, preserved evidence, and no implicit timestamps.
- [x] Self-cycles, strongly connected groups, feedback relationships, and layers are derived deterministically while canonical edges remain intact.
- [x] The pipeline provides a headless command path that a later viewer can consume without knowing Go syntax.

## Artifact sync required

- Application PRD: docs/prd.md — refreshed its stale delivery-status sentence; product behavior and scope remain unchanged.
- Application architecture summary: none — the analyzer-to-model boundary and ownership remain unchanged.
- Owning capability node/artifacts: required: .okf/capabilities/analyze-source.md; .okf/capabilities/analyze-source/go-analysis.md; .okf/capabilities/generate-models.md; docs/architecture/analyze-source/go-analysis/orchestration-status.md; docs/architecture/generate-models/orchestration-status.md; docs/architecture/analyze-source/go-analysis/implementation-slice.md.
- Issue registry: required; node issues reference required: .okf/capabilities/analyze-source.md; .okf/capabilities/analyze-source/go-analysis.md; .okf/capabilities/generate-models.md.
- Reason/no-impact decision: delivery truth is being added; no product or architecture decision changes, and the external reference folder remains untouched.

## Blocked by

None.

## User stories addressed

- US-GO-001
- US-GO-002
- US-GO-003
- US-GM-001
- US-GM-002

## Contract and scenario trace

- Contract: docs/architecture/analyze-source/go-analysis/canonical-api-cli-contract.md; docs/architecture/generate-models/canonical-api-cli-contract.md
- Scenarios: SC-GO-001, SC-GO-002, SC-GO-003, SC-GO-004, SC-GO-005, SC-GM-001, SC-GM-002, SC-GM-003, SC-GM-004, SC-GM-005, SC-GM-006, SC-GM-007, SC-GM-008

## Scenario traceability

| Source rule or use case | Acceptance scenario | Issue criterion | Verification artifact | Closure evidence |
| --- | --- | --- | --- | --- |
| Discover selected-module packages with stable Go identity | SC-GO-001 | 1 | Go analyzer package fixture | `go test ./... -count=1`: `TestAnalyzeDiscoversPackagesImportsAndEvidence` passed; repeated analysis JSON was byte-stable |
| Preserve explicit module selection and reject workspace ambiguity | SC-GO-003 | 1, 3 | Go workspace selection tests | `go test ./... -count=1`: `TestAnalyzeRejectsAmbiguousWorkspace` and `TestResolveProjectRequiresExplicitModuleForMultiModuleWorkspace` passed |
| Resolve local imports as directed dependencies with source positions | SC-GO-002 | 2 | Go package/import fixture | `go test ./... -count=1`: `TestAnalyzeDiscoversPackagesImportsAndEvidence` passed |
| Exclude tests, generated, vendor, cache/output, external, and build-conditional files by policy | SC-GO-004 | 3 | Scope/options fixture | `go test ./... -count=1`: `TestAnalyzeHonorsSourceExclusionsAndBuildOptions` passed |
| Retain standard-library, external, cgo, and unresolved imports without local fabrication | SC-GO-005 | 2, 4 | Non-local import fixture | `go test ./... -count=1`: `TestAnalyzeDiscoversPackagesImportsAndEvidence` and `TestAnalyzeReportsUnresolvedAndCgoImportsAsPartial` passed |
| Normalize valid analyzer observations into the versioned model envelope | SC-GM-001 | 5 | Model normalization and CLI integration tests | `go test ./... -count=1`: `TestNormalizeProducesDeterministicModelAndCycleProjections` and `TestModelNormalizeAndValidateCommands` passed |
| Keep hierarchy separate from dependency relationships | SC-GM-002 | 5, 6 | Hierarchy projection test | `go test ./... -count=1`: `TestNormalizeProducesDeterministicModelAndCycleProjections` passed |
| Preserve non-local targets as references and diagnostics | SC-GM-003 | 4, 5 | Partial model and import fixtures | `go test ./... -count=1`: `TestAnalyzeReportsUnresolvedAndCgoImportsAsPartial` passed |
| Merge duplicate observations while retaining contributor evidence | SC-GM-004 | 5, 6 | Duplicate observation/projection fixture | `go test ./... -count=1`: `TestNormalizeMergesEvidenceAndPreservesPartialDiagnostics` passed |
| Preserve canonical cycle edges while deriving feedback and layers | SC-GM-005 | 6 | Cyclic graph fixture | `go test ./... -count=1`: `TestNormalizeProducesDeterministicModelAndCycleProjections` passed |
| Report recoverable duplicate conflicts as partial | SC-GM-006 | 4, 5 | Conflicting observation fixture | `go test ./... -count=1`: `TestNormalizeReportsConflictingDuplicateObservationsAsPartial` passed |
| Reject broken relationship endpoints with a stable error code | SC-GM-007 | 5 | Model integrity test | `go test ./... -count=1`: `TestValidateRejectsBrokenModelEndpoint` passed |
| Repeat normalization and analysis deterministically | SC-GM-008 | 1, 5, 6 | Repeated serialization tests | `go test ./... -count=1`: deterministic model and repeated analyzer serialization assertions passed |

## Verification surfaces

- Backend boundary (`when-supported`): covered by `go test ./... -count=1`, `go test -race ./...`, `go vet ./...`, `golangci-lint run ./...`, and `staticcheck ./...`.
- Frontend integration (`not-applicable`): this issue changes no rendered UI, navigation, or frontend integration surface.
- End-to-end (`when-supported`): `TestModelNormalizeAndValidateCommands` exercises analyze → normalize → validate → projection through the headless command path; `go build ./...` passed.
- Repository/OKF integrity: `git diff --check` and `node C:\Users\buffo\.agents\skills\packages\planning\skills\okf-validate\scripts\okf-validate.mjs .okf --strict` passed at closeout.
