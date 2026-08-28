# 035 — Assemble the compiled analyzer distribution

## Issue Metadata

- Issue number: 035
- Owning capability node: /.okf/capabilities/analyze-source/plugin-runtime/compiled-external-analyzer-distribution.md
- Artifact root: docs/architecture/analyze-source/plugin-runtime/compiled-external-analyzer-distribution/
- Issue file: docs/agents/issues/pending/20260828-035-compiled-analyzer-distribution-assembly.md
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

- [ ] `make analyzers` compiles all five enabled analyzer entrypoints from the
  issue 034 commands for the requested target platform.
- [ ] `make release` packages the host and the same analyzer tree without
  requiring Python, Rust, Node, or another analyzer implementation runtime on
  the end user's machine.
- [ ] Each package is written beneath
  `analyzers/<logical-analyzer-id>/<platform>/` and contains one executable
  plus a descriptor that points to that executable with argv metadata.
- [ ] The generated `analyzers/index.json` contains the v1 schema and host API
  versions, build identifier, logical ID, analyzer version, language, API
  version, exact platform, normalized relative executable and descriptor paths,
  and lowercase SHA-256 digests for both files.
- [ ] Package entries are unique by logical analyzer ID and platform and are
  sorted deterministically. Repeating the build with unchanged source,
  toolchain, target, and inputs produces byte-stable descriptors and index
  content apart from an intentionally controlled build identifier.
- [ ] The assembly validates that every enabled analyzer has one complete
  package entry and rejects duplicate IDs, missing artifacts, invalid
  manifests, unsupported platforms, unsafe paths, or digest-generation
  failures.
- [ ] A failed analyzer compilation or index validation fails the build and
  does not publish a partial or replacement index over the last complete
  distribution.
- [ ] Build tests cover the first matrix and make the toolchain, target,
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

## Handoff

Issue 036 can consume the generated index and package tree to implement
trusted loading, exact platform selection, and pre-launch verification.
