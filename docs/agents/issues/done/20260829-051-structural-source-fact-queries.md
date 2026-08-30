# 051 — Add structural source-fact queries and bounded evidence

## Metadata

- Issue: 051
- Type: feature
- Owning capability: `/.okf/capabilities/code-quality-and-intelligence/source-facts-and-symbol-index.md`
- Related capabilities: `/.okf/capabilities/explore-architecture.md`, `/.okf/capabilities/export-and-automate.md`
- Artifact root: `docs/architecture/code-quality-and-intelligence/source-facts-and-symbol-index/`
- Execution: AFK
- Human review: none
- Suggested state: done

## Parent artifacts

- `docs/prd.md`
- `docs/architecture/application-architecture-summary.md`
- `docs/architecture/code-quality-and-intelligence/source-facts-and-symbol-index/prd.md`
- `docs/architecture/code-quality-and-intelligence/source-facts-and-symbol-index/acceptance-scenarios.md`
- `docs/architecture/code-quality-and-intelligence/source-facts-and-symbol-index/canonical-domain-model.md`
- `docs/architecture/code-quality-and-intelligence/source-facts-and-symbol-index/canonical-use-cases.md`
- `docs/architecture/code-quality-and-intelligence/source-facts-and-symbol-index/canonical-api-cli-contract.md`
- `docs/architecture/code-quality-and-intelligence/source-facts-and-symbol-index/orchestration-status.md`
- `internal/viewer/model_http.go`
- `internal/viewer/source_http.go`

## What to build

Add a deterministic, read-only source-fact query service and local read model
over the canonical and aggregate source-index projections. The query boundary
must keep structural source facts separate from quality judgments and must
return compact evidence that can be followed to a safe, bounded source
inspection path.

Support these query families:

- `InspectModuleSourceFacts`, using explicit module containment to return the
  relevant files, file status, sizes, declarations, documentation, visibility,
  provenance, and relations.
- `FindFiles`, `FindSymbols`, and `FindDocumentation`, with structural filters,
  explicit scope selection, maximum-result limits, stable pagination, and
  canonical ordering.
- `GetSourceFactEvidence`, linking facts to hash-checked spans and existing
  source references, with optional bounded source context only when explicitly
  requested.

Default responses must omit complete source text and raw documentation payloads
unless a caller explicitly requests bounded evidence. The service must preserve
per-scope and combined provenance and must expose unsupported, unknown, partial,
and unavailable coverage rather than silently filling gaps. It does not own
ranking, quality scoring, live watching, MCP transport, or source mutation.

## Acceptance criteria

- Module inspection follows explicit containment and returns deterministic file
  lists with sizes, status, declarations, documentation, visibility, hashes,
  provenance, and relations.
- File, symbol, and documentation queries support structural filters, scope,
  maximum-result/page limits, pagination tokens, case policy, and documented
  canonical ordering.
- Evidence responses link facts to hash-checked spans or legacy source
  references and can provide bounded source context through a safe read-only
  boundary.
- Default responses omit complete source and raw documentation; requested
  evidence is bounded or rejected when it exceeds the configured budget.
- Per-scope and combined queries preserve scope identity and provenance without
  inferring relationships from matching paths or names.
- The local HTTP boundary exposes stable source-index projections and
  structured errors for invalid scope, pagination, budget, and unavailable
  coverage cases.
- Fixtures with many files and symbols, unsupported languages, partial
  extraction, metric facts, and repeated queries prove deterministic output,
  pagination, and token-safe payload sizes.

## Artifact synchronization

- Application PRD: no change required; the source-fact query boundary is within
  the synchronized inspection and export scope.
- Application architecture summary: no change required; the read model is an
  implementation of the existing analysis-to-viewer flow.
- Owning capability artifacts: update `orchestration-status.md` only if query
  findings change the approved boundary or sequencing.
- Planning graph and delivery registry: preserve the issue link, owner, and
  dependency state. No topology or capability-state transition is claimed by
  this issue alone.
- No MCP or external protocol surface is introduced by this issue.

## Human review

No human review is required for the default implementation path. Escalate if
the query service needs a new public protocol surface or changes the safe source
inspection boundary.

## Blocked by

None — delivered after issue 050, archived at
`docs/agents/issues/done/20260829-050-scope-safe-source-index-aggregation.md`.

## Specification anchors

- Source-facts requirements: SFI-FR-007, SFI-FR-013.
- Source-facts acceptance scenarios: SFI-AC-003, SFI-AC-014, SFI-AC-015.
- Canonical concepts: source-index query, module containment, structural
  filter, evidence, source span, provenance, scope, and budget.

## User journeys covered

- Journey 2: inspect a module and its source evidence.
- Journey 12: inspect a module's source facts.

## Verification obligations

| Layer | Required evidence |
|---|---|
| Backend | When-supported: query filtering, pagination, scope qualification, evidence bounds, HTTP errors, and legacy omission tests. |
| Frontend | Not applicable to this read-model and local-boundary issue. |
| E2E | When-supported: module inspection and bounded evidence retrieval through the local HTTP boundary. |

## Acceptance verification

- [x] Module inspection and file/symbol/documentation queries use explicit
  containment, structural filters, canonical ordering, stable pagination, and
  bounded limits.
- [x] Evidence preserves scopes, provenance, hash-linked spans, and optional
  read-only bounded source context while default responses omit raw payloads.
- [x] Local HTTP routes return structured errors and preserve legacy models
  that omit `source_index`.

## Implementation and verification

- Added the deterministic `QueryService` for module containment, file/symbol/
  documentation filters, explicit scope selection, stable cursors, bounded
  limits, coverage states, provenance, and evidence.
- Added local HTTP source-index routes and safe, read-only bounded source
  excerpts with structured errors; default query/evidence responses omit raw
  documentation and source text.
- Verified with source-index query tests,
  `TestSourceIndexHTTPQueriesAndBoundedEvidence`,
  `TestAggregateSourceIndexHTTPResolvesConcreteScopeBeforeQuerying`, and
  `TestLegacyModelSourceIndexRemainsOptional`.

| Scenario | Evidence |
| --- | --- |
| `SFI-AC-003`, `SFI-AC-014` | module containment, filtering, pagination, coverage, and evidence tests |
| `SFI-AC-015` | local HTTP route/error/boundary and legacy omission tests |

## Handoff

When complete, issue 052 may render the module source-fact projection in the
existing local viewer and carry the visual review gate.
