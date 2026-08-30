# 050 — Carry source-index snapshots through canonical and aggregate analysis

## Metadata

- Issue: 050
- Type: feature
- Owning capability: `/.okf/capabilities/code-quality-and-intelligence/source-facts-and-symbol-index.md`
- Related capabilities: `/.okf/capabilities/analyze-source/plugin-runtime/multi-analyzer-orchestration.md`, `/.okf/capabilities/generate-models.md`
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
- `docs/architecture/code-quality-and-intelligence/source-facts-and-symbol-index/readiness-review.md`
- `docs/architecture/analyze-source/plugin-runtime/multi-analyzer-orchestration/`
- `internal/analysis/host.go`
- `internal/model/canonical/normalize.go`
- `internal/analysis/orchestration/aggregation.go`

## What to build

Carry the optional source-index attachment through every analysis path that can
produce an `AnalysisResult`: direct analysis, planned jobs, scheduler-backed
execution, project-backed analysis, local HTTP access, and reanalysis. The
attachment must be built from the resolved source scope and must preserve the
authoritative per-scope snapshot rather than reconstructing facts from a
combined model.

Define the canonical and aggregate boundaries for source facts:

- Keep each usable job result's snapshot, scope identity, project identity,
  source-policy identity, source-set fingerprint, analyzer identity, extractor
  registry, requested capabilities, and coverage state together.
- Make the combined projection explicitly derived, deterministic, and
  scope-qualified. Do not merge files, symbols, documentation, or relations by
  path or display name across scopes.
- Preserve valid snapshots and diagnostics when another scope is partial,
  failed, cancelled, or unavailable.
- Keep the existing external protocol version and legacy result behavior
  compatible when the optional attachment is omitted.

This issue owns integration and aggregation semantics. It does not add new
extractors, quality rules, live watching, MCP transport, or source editing.

## Acceptance criteria

- Direct analysis, planned analysis, scheduler execution, project-backed
  analysis, local HTTP analysis, and reanalysis all use the same source-index
  attachment path and resolved source scope.
- Canonical normalization preserves an optional `SourceIndex` without changing
  existing model semantics or dropping unknown/unsupported coverage states.
- Every usable job snapshot contains scope, project, source policy, source-set,
  analyzer, extractor, requested-capability, and coverage metadata.
- Aggregate results retain authoritative per-scope snapshots and expose a
  deterministic, scope-qualified derived projection with no path/name-based
  cross-scope inference.
- Partial, failed, cancelled, and unavailable scopes produce diagnostics while
  preserving valid sibling snapshots.
- Existing external protocol consumers remain compatible; omitted source-index
  attachments continue to validate and serialize as before.
- Combined CLI, HTTP, canonical-model, and export tests prove deterministic
  IDs and projection order, partial-scope behavior, optional omission, and the
  absence of complete source text in the aggregate payload.

## Artifact synchronization

- Application PRD: no change required; the synchronized application boundary
  already includes model generation and source inspection without changing the
  product contract.
- Application architecture summary: no change required; this issue realizes
  the existing per-scope and aggregate analysis flow.
- Owning capability artifacts: update `orchestration-status.md` only if
  execution findings change delivery sequencing or boundary assumptions.
- Planning graph and delivery registry: preserve the issue link, owner, and
  dependency state. No topology or capability-state transition is claimed by
  this issue alone.
- No new external contract version is permitted.

## Human review

No human review is required for the default implementation path. Escalate if
aggregation requires a change to the application protocol boundary, source
scope identity model, or existing viewer contract.

## Blocked by

None — delivered after issues 048 and 049, both archived in
`docs/agents/issues/done/`.

## Specification anchors

- Source-facts requirements: SFI-FR-001, SFI-FR-009, SFI-FR-010, SFI-FR-011,
  SFI-FR-012, SFI-FR-013.
- Source-facts acceptance scenarios: SFI-AC-009, SFI-AC-010, SFI-AC-011,
  SFI-AC-012, SFI-AC-013.
- Canonical concepts: scope context, source set, analyzer job, source index,
  source-index snapshot, combined projection, coverage, and provenance.

## User journeys covered

- Journey 9: planned multi-analyzer analysis.
- Journey 10: project-backed analysis.
- Journey 11: local HTTP analysis and reanalysis.
- Journey 12: inspect a module's source facts.

## Verification obligations

| Layer | Required evidence |
|---|---|
| Backend | When-supported: direct, planned, scheduler, project, HTTP, aggregate, partial-scope, and legacy-omission tests. |
| Frontend | Not applicable to this integration-only issue. |
| E2E | When-supported: prove a multi-scope analysis reaches the viewer/export boundary with scope-qualified source facts. |

## Acceptance verification

- [x] Direct, planned, scheduler-backed, project-backed, HTTP, and reanalysis
  paths use the same optional source-index attachment.
- [x] Canonical normalization and aggregate assembly preserve authoritative
  per-scope snapshots, diagnostics, coverage, and legacy omission behavior.
- [x] The derived projection is deterministic and scope-qualified, with no
  path/name-based cross-scope inference or complete source text.

## Implementation and verification

- Carried `SourceIndex` through canonical normalization, model cloning,
  orchestration jobs, scheduler/project/reanalysis results, aggregate model
  assembly, and local HTTP selection.
- Added independent authoritative snapshots plus a deterministic,
  scope-qualified combined projection. Projection mapping copies nested spans
  before rewriting IDs and preserves partial sibling diagnostics.
- Verified with `TestAggregateNamespacesObservationsAndPreservesScopeParity`,
  `TestAggregateStatusRulesAndSourceIdentity`,
  `TestAggregateSourceProjectionDoesNotMutateCachedScope`, the source-index
  HTTP aggregate test, export/model tests, and the full repository gates.

| Scenario | Evidence |
| --- | --- |
| `SFI-AC-009`, `SFI-AC-010` | aggregate scope/projection and deterministic identity tests |
| `SFI-AC-011`, `SFI-AC-012`, `SFI-AC-013` | partial-scope retention, canonical normalization, optional omission, and export/HTTP tests |

## Handoff

When complete, issue 051 may build the structural query and bounded evidence
read model on top of the stable aggregate contract.
