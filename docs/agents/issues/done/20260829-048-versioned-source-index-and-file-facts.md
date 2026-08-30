# 048 — Add the versioned source-index attachment and deterministic file facts

## Issue Metadata

- Issue number: `048`
- Owning capability node: `/.okf/capabilities/code-quality-and-intelligence/source-facts-and-symbol-index.md`
- Related consumer node: `/.okf/capabilities/generate-models.md`
- Artifact root: `docs/architecture/code-quality-and-intelligence/source-facts-and-symbol-index/`
- Issue file: `docs/agents/issues/done/20260829-048-versioned-source-index-and-file-facts.md`
- Category: `feature`
- Execution type: `AFK`
- Review gate: `none`
- Suggested state: `done`

## Parent PRD

`docs/architecture/code-quality-and-intelligence/source-facts-and-symbol-index/prd.md`

## Parent Artifacts

- `docs/prd.md`
- `docs/architecture/application-architecture-summary.md`
- `docs/architecture/code-quality-and-intelligence/source-facts-and-symbol-index/discovery-notes.md`
- `docs/architecture/code-quality-and-intelligence/source-facts-and-symbol-index/requirements-gap-analysis.md`
- `docs/architecture/code-quality-and-intelligence/source-facts-and-symbol-index/domain-glossary.md`
- `docs/architecture/code-quality-and-intelligence/source-facts-and-symbol-index/canonical-domain-model.md`
- `docs/architecture/code-quality-and-intelligence/source-facts-and-symbol-index/canonical-use-cases.md`
- `docs/architecture/code-quality-and-intelligence/source-facts-and-symbol-index/canonical-api-cli-contract.md`
- `docs/architecture/code-quality-and-intelligence/source-facts-and-symbol-index/acceptance-scenarios.md`
- `docs/architecture/code-quality-and-intelligence/source-facts-and-symbol-index/readiness-review.md`

## What to build

Add the independently versioned `arch-view.source-index/v1` attachment beside
the existing analysis and canonical-model collections. Build the first
file-only snapshot from the already resolved eligible source scope. Each file
must retain its normalized repository-relative path, language, roles, raw byte
count, physical line count, SHA-256 content hash, analysis status, and
provenance. The core must assign opaque deterministic IDs, canonicalize
ordering, calculate the semantic snapshot digest, and validate coverage,
extensions, and versioned metric slots.

Keep the attachment optional. Existing analyzers, clients, exports, and model
consumers must continue to work when `source_index` is omitted. This issue does
not extract declarations or documentation yet; it establishes the common
snapshot and file-fact path that the Go extractor will consume.

## Acceptance criteria

- [x] `AnalysisResult` and the canonical model can carry an optional
  `SourceIndex` using the `arch-view.source-index/v1` contract without changing
  the meaning of modules, relationships, source references, diagnostics, or
  derived graph data.
- [x] A resolved eligible source scope produces one normalized `FileRecord`
  per enumerable file, including language, namespaced roles, analysis status,
  provenance, raw byte count, physical line count, and SHA-256 hash.
- [x] Empty files, unterminated files, LF, CRLF, lone-CR, and invalid-UTF-8
  inputs follow the specified raw-byte line-count rule; decoding failure does
  not change byte count or hashing.
- [x] Snapshot-local IDs are opaque and deterministic for equal scope, source,
  producer, capability, and option inputs. Collections and the semantic digest
  use the contract's canonical ordering and exclude operational timestamps.
- [x] Validation rejects unsafe paths, invalid digests, inconsistent spans or
  references, duplicate records, invalid coverage, and malformed typed metric
  or extension metadata with structured diagnostics.
- [x] Unknown typed extension blocks are safely ignored after namespace and
  version validation, and unsupported or unknown coverage is not rewritten as
  absence.
- [x] Direct analysis and model serialization demonstrate a valid file-only
  source-index snapshot while a legacy result without `source_index` remains
  valid and unchanged for existing consumers.

## Artifact sync required

- Application PRD: `none` — this implements the already synchronized source-index file-fact contract and adds no new product scope or rule.
- Application architecture summary: `none` — the optional attachment and core ownership are already documented; record any discovered boundary change before closure.
- Owning capability node/artifacts: `required: /.okf/capabilities/code-quality-and-intelligence/source-facts-and-symbol-index.md` and `docs/architecture/code-quality-and-intelligence/source-facts-and-symbol-index/orchestration-status.md`.
- Issue registry: `required`; node `issues:` reference: `required when .okf exists`.
- Reason/no-impact decision: delivery truth changes through this issue. Topology, product scope, and architecture ownership remain unchanged.

## Human review gate

None. The slice has no rendered UI change and is independently verifiable
through source-index, serialization, validation, and CLI/model tests.

## Blocked by

None — the resolved source-scope boundary, analysis result, and canonical model
contracts already exist.

## Artifact anchors

- `SFI-FR-001`, `SFI-FR-002`, `SFI-FR-003`, `SFI-FR-009`, `SFI-FR-010`, `SFI-FR-012`, and `SFI-FR-013`.
- `SFI-AC-001`, `SFI-AC-002`, `SFI-AC-009`, `SFI-AC-012`, and `SFI-AC-013`.
- `SourceIndex`, `SourceIndexSnapshot`, `FileRecord`, `FileSize`, `FactProvenance`, `CoverageRecord`, and `ContentDigest` in the canonical domain model.
- The existing `analysis.SourceScope`, analysis-result validation, canonical normalization, and JSON export paths.

## User stories addressed

The capability PRD has no numbered user-story section. This slice supports
application journeys 2, 4, and 12 by making compact, deterministic file facts
available through analysis and model output.

## Verification obligations

- Policy source: `/.okf/project.md`

| Scenario | Backend boundary | Frontend integration | End-to-end journey |
| --- | --- | --- | --- |
| `SFI-AC-001` | `when-supported` | `not-applicable` | `when-supported` |
| `SFI-AC-002` | `when-supported` | `not-applicable` | `when-supported` |
| `SFI-AC-009` | `when-supported` | `not-applicable` | `when-supported` |
| `SFI-AC-012` | `when-supported` | `not-applicable` | `when-supported` |
| `SFI-AC-013` | `when-supported` | `not-applicable` | `when-supported` |

## Implementation and verification

- Added the optional `SourceIndex` attachment, versioned v1 records, canonical
  digesting, validation, deterministic IDs, and file-fact assembly in
  `internal/analysis/source_facts.go`, `internal/analysis/source_facts_validation.go`,
  and `internal/analysis/sourceindex/`.
- Integrated the attachment with analysis results and canonical model
  normalization without changing legacy omission behavior.
- Verified with `TestBuildSourceIndexRetainsRawFileFactsAndGoStructure`,
  `TestBuildSourceIndexIsDeterministicAndQueriesFollowContainment`,
  `TestBuildSourceIndexUsesRawPhysicalLineRule`, and the full analysis/model
  test suites.

| Scenario | Evidence |
| --- | --- |
| `SFI-AC-001`, `SFI-AC-002` | source-index builder and raw file-fact tests |
| `SFI-AC-009`, `SFI-AC-012`, `SFI-AC-013` | deterministic projection/query, validation, and legacy optional-attachment tests |

## Handoff

Issue 049 adds the registered Go extractor and declaration/documentation facts
on top of this neutral snapshot path.
