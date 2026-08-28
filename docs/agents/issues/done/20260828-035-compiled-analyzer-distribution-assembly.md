# 035 — Assemble the compiled analyzer distribution

## Issue Metadata

- Issue number: 035
- Owning capability node: /.okf/capabilities/analyze-source/plugin-runtime/compiled-external-analyzer-distribution.md
- Artifact root: docs/architecture/analyze-source/plugin-runtime/compiled-external-analyzer-distribution/
- Issue file: docs/agents/issues/done/20260828-035-compiled-analyzer-distribution-assembly.md
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
- docs/architecture/analyze-source/plugin-runtime/implementation-slice.md
- docs/agents/issues/done/20260828-034-compiled-analyzer-plugin-entrypoints.md

## What to build

Add the reproducible release assembly that turns the compiled entrypoints into
an application-managed analyzer distribution. The root build targets must
compile every enabled analyzer for the supported platform targets
`windows-amd64`, `linux-amd64`, and `darwin-arm64`, then write the executable,
descriptor, digests, and deterministic `analyzers/index.json` under the
canonical package tree.

`make analyzers` builds the analyzer tree for the requested target. `make
release` includes the host application and that same tree. A failed or
incomplete analyzer build must fail the target and leave no replacement
partial index. This issue owns assembly and build inputs. Trusted loading and
runtime selection follow in issues 036 and 037.

## Acceptance criteria

- [x] `make analyzers` compiles all five enabled analyzer entrypoints from the
  issue 034 commands for the requested target platform.
- [x] `make release` packages the host and the same analyzer tree without
  requiring Python, Rust, Node, or another analyzer implementation runtime on
  the end user's machine.
- [x] Each package is written beneath
  `analyzers/<logical-analyzer-id>/<platform>/` and contains one executable
  plus a descriptor that points to that executable with argv metadata.
- [x] The generated `analyzers/index.json` contains the v1 schema and host API
  versions, build identifier, logical ID, analyzer version, language, API
  version, exact platform, normalized relative executable and descriptor paths,
  and lowercase SHA-256 digests for both files.
- [x] Package entries are unique by logical analyzer ID and platform and are
  sorted deterministically. Repeating the build with unchanged source,
  toolchain, target, and inputs produces byte-stable descriptors and index
  content apart from an intentionally controlled build identifier.
- [x] The assembly validates that every enabled analyzer has one complete
  package entry and rejects duplicate IDs, missing artifacts, invalid
  manifests, unsupported platforms, unsafe paths, or digest-generation
  failures.
- [x] A failed analyzer compilation or index validation fails the build and
  does not publish a partial or replacement index over the last complete
  distribution.
- [x] Build tests cover the first matrix and make the toolchain, target,
  enabled-analyzer set, and output root explicit rather than relying on the
  caller's current directory or machine `PATH`.

## Artifact sync required

- Application PRD: none. The release-maintainer workflow and future direction
  are already documented.
- Application architecture summary: none when the output tree, build target,
  and trust anchor match the approved package boundary; required if they do
  not.
- Owning capability node/artifacts: required: `/.okf/capabilities/analyze-source/plugin-runtime/compiled-external-analyzer-distribution.md`; its orchestration status; and the plugin-runtime implementation slice with the build/index issue evidence.
- Issue registry: required; keep this issue linked from the owning node until
  closeout.
- OKF log: required for the delivery record.
- Reason/no-impact decision: delivery truth changes. Product actors and the
  application architecture boundary already include this package tree and
  build target.

## Human review gate

None. This is release assembly and build verification with no rendered UI.

## Blocked by

Issue 034 is complete; this issue is unblocked and may assemble the five
compiled entrypoints into the release tree.

## Artifact anchors

- CED-FR-001 and CED-FR-002.
- `AnalyzerIndex`, `AnalyzerPackage`, `PlatformTarget`, and
  `IntegrityDigest` in the canonical domain model.
- The Build and assemble section of the capability PRD and the package index
  contract.

## Acceptance scenarios addressed

- SC-CED-001 — Assemble the complete analyzer tree
- SC-CED-002 — Select the exact host platform, through artifact metadata
- SC-CED-006 — Avoid language-runtime installation

## User stories addressed

The compiled-distribution PRD has no numbered user-story section. This issue
supports the release-maintainer workflow and parent story US-PR-001 by making
all supported analyzers available as versioned packages.

## Verification obligations

- Policy source: `/.okf/project.md`

| Scenario | Backend boundary | Frontend integration | End-to-end journey |
| --- | --- | --- | --- |
| SC-CED-001/002 | when-supported: build-matrix, index, path, digest, and atomic-publication tests | not-applicable | when-supported: release tree smoke test |
| SC-CED-006 | when-supported: compiled artifact execution with implementation runtimes absent | not-applicable | when-supported: package-to-analysis smoke test |

## Scenario traceability

| Source rule/use case | Canonical scenario | Issue criterion | Planned verification | Closure evidence |
| --- | --- | --- | --- | --- |
| CED-FR-001, CED-FR-002; `AssembleAnalyzerDistribution`; `AnalyzerIndex` and `AnalyzerPackage` completeness, digest, and atomic-publication rules | SC-CED-001 — Assemble the complete analyzer tree | 1, 3, 4, 5, 6, 7, 8 | `internal/analysis/distribution/assembly_test.go` and `release_test.go` cover deterministic packages/indexes, validation failures, explicit Go/C/C++ build inputs, safe replacement, full-release assembly, and last-complete-output preservation; real `make analyzers` smoke test | `go test ./internal/analysis/distribution -count=1` passed; the real Make smoke produced five complete packages and a v1 index |
| `PlatformTarget` exact supported target and normalized package paths | SC-CED-002 — Select the exact host platform | 1, 4, 5, 6, 8 | Supported-platform matrix tests plus unsupported-platform, unsafe-path, duplicate-entry, and exact `GOOS`/`GOARCH` assertions | `TestAssembleSupportsFirstTargetMatrix` and `TestAssembleRejectsInvalidEnabledSetAndPlatform` passed; the real smoke index contains only `windows-amd64` entries with normalized relative paths |
| Release-maintainer workflow and no implementation-runtime installation requirement | SC-CED-006 — Avoid language-runtime installation | 2 | Real `make release` smoke test and compiled package inspection showing the host plus the same analyzer tree with no Python, Rust, Node, or other implementation runtime bundled | `make release` passed in an isolated output root; it contains `arch-view.exe`, the same five analyzer packages, and no implementation-runtime files |

## Implementation and verification

### Implementation

- Added `internal/analysis/distribution/`, a reusable assembly boundary with
  the canonical five-analyzer catalog, exact supported platform matrix,
  explicit build requests, compiled executable generation, descriptor
  creation, SHA-256 digests, strict index decoding/validation, canonical
  package-tree checks, deterministic JSON, safe replacement checks, atomic
  directory publication, and host-plus-analyzer release orchestration.
- Added `cmd/analyzer-distribution/main.go` for the `make analyzers` target and
  `cmd/release-assembly/main.go` for an atomic host-plus-analyzers release.
  Both commands accept explicit repository/output, platform, build ID,
  Go/C/C++ toolchain, and enabled-analyzer inputs.
- Added the root `Makefile` targets `analyzers` and `release`, and ignored
  generated `dist/` output. The release command stages the host application
  and analyzer tree together before replacing the prior release.

### Verification

Passed on 2026-08-28:

- `go test ./internal/analysis/distribution -count=1`
- `go test ./... -count=1`
- `go test -race ./... -count=1`
- `go vet ./...`
- `go build ./...`
- `staticcheck ./...`
- `golangci-lint run`
- `go mod verify`
- `git diff --check`
- `make analyzers GO=go PLATFORM=windows-amd64 BUILD_ID=final-smoke ANALYZER_ROOT=<isolated temporary root>`
- `make release GO=go PLATFORM=windows-amd64 BUILD_ID=final-release-smoke RELEASE_ROOT=<isolated temporary root>`
- Repeated `make analyzers` with unchanged inputs produced byte-stable
  `index.json` content; the observed index SHA-256 was
  `1a3192ab2c24a23b9745bb88da24f716f756b685c3b5a986e3b3fe8c5e75edef`.
- `node C:\Users\buffo\.agents\skills\packages\planning\skills\okf-validate\scripts\okf-validate.mjs .okf --strict --json` — conformant, zero errors, zero warnings.

The root `when-supported` policy is satisfied for the applicable backend and
end-to-end surfaces. Frontend integration is `not-applicable`: issue 035 is
release assembly and does not change model, viewer, or export behavior.

### Artifact sync and no-impact decision

The compiled-distribution capability node, its orchestration status, the
plugin-runtime implementation slice, the project and graph frontiers, the
issue registry, and `.okf/log.md` are updated with this delivery truth. The
application PRD and application architecture summary require no semantic
change: the implementation matches the already approved application-managed
`analyzers/<logical-analyzer-id>/<platform>/` boundary, build target, and
descriptor/index trust anchor. Issues 036–038 remain responsible for trusted
package verification, packaged runtime selection, and final release/parity
closure, so the capability remains `specified`.

## Handoff

Issue 036 is now unblocked and can consume the generated index and package
tree to implement trusted loading, exact platform selection, and pre-launch
verification. Issues 037 and 038 remain ordered behind that trust boundary.
