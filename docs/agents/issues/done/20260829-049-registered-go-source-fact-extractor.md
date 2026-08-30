# 049 — Add the registered Go source-fact extractor

## Issue Metadata

- Issue number: `049`
- Owning capability node: `/.okf/capabilities/code-quality-and-intelligence/source-facts-and-symbol-index.md`
- Related consumer node: `/.okf/capabilities/analyze-source.md`
- Artifact root: `docs/architecture/code-quality-and-intelligence/source-facts-and-symbol-index/`
- Issue file: `docs/agents/issues/done/20260829-049-registered-go-source-fact-extractor.md`
- Category: `feature`
- Execution type: `AFK`
- Review gate: `none`
- Suggested state: `done`

## Parent PRD

`docs/architecture/code-quality-and-intelligence/source-facts-and-symbol-index/prd.md`

## Parent Artifacts

- `docs/prd.md`
- `docs/architecture/application-architecture-summary.md`
- `docs/architecture/code-quality-and-intelligence/source-facts-and-symbol-index/prd.md`
- `docs/architecture/code-quality-and-intelligence/source-facts-and-symbol-index/canonical-domain-model.md`
- `docs/architecture/code-quality-and-intelligence/source-facts-and-symbol-index/canonical-use-cases.md`
- `docs/architecture/code-quality-and-intelligence/source-facts-and-symbol-index/canonical-api-cli-contract.md`
- `docs/architecture/code-quality-and-intelligence/source-facts-and-symbol-index/acceptance-scenarios.md`
- `docs/architecture/code-quality-and-intelligence/source-facts-and-symbol-index/readiness-review.md`
- `internal/analysis/syntax/provider.go`
- `internal/analysis/syntax/go/provider.go`
- `internal/analyzers/go/scanner/scan.go`

## What to build

Add the registered `SourceFactExtractor` strategy boundary and implement the
first Go extractor. Reuse the existing Tree-sitter provider and resolved file
scope. Emit named Go declarations with stable coarse categories, open
language-specific kinds, visibility, hash-linked spans, provenance, and
explicit containment/declaration relations. Retain every language-native
documentation candidate and let the Go extractor choose the deterministic
primary candidate for each selection group.

The core must keep file facts when parsing or extraction is incomplete. A
malformed batch becomes structured partial or unknown coverage and does not
erase valid facts from other files. No quality thresholds, call graph, live
watching, MCP transport, or language-specific branch in core orchestration is
part of this issue.

## Acceptance criteria

- [x] Extractors register explicit IDs, versions, supported languages, and
  namespaced capabilities. Duplicate identical registrations are idempotent;
  conflicting implementations are rejected without adding a core language
  switch.
- [x] The Go extractor emits named functions, methods, types, interfaces,
  constants, variables, and other supported declarations as `SymbolRecord`
  values with coarse categories, open `language_kind` values, visibility, and
  declaration locations.
- [x] Tree-sitter ranges are normalized to one-based, end-exclusive UTF-8 byte
  spans linked to the emitted file content hash, with legacy source-reference
  links retained where they are known.
- [x] Documentation candidates retain their format, subject, evidence,
  attachment basis, precedence, completeness, and provenance. The extractor
  marks one deterministic primary candidate where appropriate and distinguishes
  present, absent, unsupported, unknown, and partial coverage.
- [x] The extractor emits only reported `contains` and `declares` relations for
  module/file/symbol ownership. It does not infer membership from matching
  paths or duplicate architecture dependency relationships.
- [x] A parser or extractor failure leaves mandatory file records intact,
  reports structured diagnostics and coverage, and does not discard valid
  declarations or documentation from unrelated files.
- [x] A Go analysis fixture produces files, declarations, documentation,
  visibility, spans, provenance, and relations in the optional source-index
  attachment through the normal analysis path.

## Artifact sync required

- Application PRD: `none` — Go source-fact extraction is already part of the synchronized source-index scope and adds no new product rule.
- Application architecture summary: `none` — the registered-extractor/core ownership split is already documented; record any discovered boundary change before closure.
- Owning capability node/artifacts: `required: /.okf/capabilities/code-quality-and-intelligence/source-facts-and-symbol-index.md` and `docs/architecture/code-quality-and-intelligence/source-facts-and-symbol-index/orchestration-status.md`.
- Issue registry: `required`; node `issues:` reference: `required when .okf exists`.
- Reason/no-impact decision: delivery truth changes through the extractor implementation. Product scope, topology, and architecture ownership remain unchanged.

## Human review gate

None. The slice is backend-only and is independently verifiable through Go
fixtures, extractor registry tests, and analysis-result assertions.

## Blocked by

None — delivered after issue 048, which is archived at
`docs/agents/issues/done/20260829-048-versioned-source-index-and-file-facts.md`.

## Artifact anchors

- `SFI-FR-004`, `SFI-FR-005`, `SFI-FR-006`, `SFI-FR-007`, `SFI-FR-008`, and `SFI-FR-010`.
- `SFI-AC-004`, `SFI-AC-005`, `SFI-AC-006`, `SFI-AC-007`, `SFI-AC-008`, and `SFI-AC-011`.
- `SourceFactExtractorRegistry`, `SourceFactExtractor`, `FactBatch`,
  `SymbolRecord`, `DocumentationRecord`, `SourceSpan`, and `CodeRelation`.
- The existing Tree-sitter point/range contract and Go analyzer scan output.

## User stories addressed

The capability PRD has no numbered user-story section. This slice supports
application journeys 5 and 12 by making the first supported language's
declarations and documentation available through the common contract.

## Verification obligations

- Policy source: `/.okf/project.md`

| Scenario | Backend boundary | Frontend integration | End-to-end journey |
| --- | --- | --- | --- |
| `SFI-AC-004` | `when-supported` | `not-applicable` | `when-supported` |
| `SFI-AC-005` | `when-supported` | `not-applicable` | `when-supported` |
| `SFI-AC-006` | `when-supported` | `not-applicable` | `when-supported` |
| `SFI-AC-007` | `when-supported` | `not-applicable` | `when-supported` |
| `SFI-AC-008` | `when-supported` | `not-applicable` | `when-supported` |
| `SFI-AC-011` | `when-supported` | `not-applicable` | `when-supported` |

## Implementation and verification

- Added the thread-safe `SourceFactExtractor` registry and registered Go
  extractor in `internal/analysis/sourceindex/registry.go` and
  `internal/analyzers/go/sourcefacts/`.
- Added Go scanner source-file retention and Tree-sitter declaration,
  visibility, documentation, hash-linked span, provenance, and explicit
  containment/declaration relation output through the normal analyzer path.
- Verified parser/extraction isolation, deterministic records, and the Go
  fixture through `internal/analysis/sourceindex/sourceindex_test.go`,
  `internal/analyzers/go/packages_test.go`, and the full Go analyzer suite.

| Scenario | Evidence |
| --- | --- |
| `SFI-AC-004`, `SFI-AC-005`, `SFI-AC-006` | registered extractor and Go declaration/span fixture assertions |
| `SFI-AC-007`, `SFI-AC-008`, `SFI-AC-011` | documentation, explicit containment, partial-isolation, and provenance assertions |

## Handoff

Issue 050 carries the source-index attachment through canonical normalization
and multi-analyzer scope aggregation.
