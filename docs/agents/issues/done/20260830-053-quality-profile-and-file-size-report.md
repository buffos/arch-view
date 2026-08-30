# 053 — Add the quality profile, rule catalog, and first file-size report

## Issue Metadata

- Issue number: `053`
- Owning capability node: `/.okf/capabilities/code-quality-and-intelligence/deterministic-quality-checks.md`
- Related capability node: `/.okf/capabilities/code-quality-and-intelligence/source-facts-and-symbol-index.md`
- Artifact root: `docs/architecture/code-quality-and-intelligence/deterministic-quality-checks/`
- Issue file: `docs/agents/issues/done/20260830-053-quality-profile-and-file-size-report.md`
- Category: `feature`
- Execution type: `AFK`
- Review gate: `none`
- Suggested state: `done`

## Parent PRD

`docs/architecture/code-quality-and-intelligence/deterministic-quality-checks/prd.md`

## Parent artifacts

- `docs/prd.md`
- `docs/architecture/application-architecture-summary.md`
- `docs/architecture/code-quality-and-intelligence/deterministic-quality-checks/canonical-domain-model.md`
- `docs/architecture/code-quality-and-intelligence/deterministic-quality-checks/canonical-use-cases.md`
- `docs/architecture/code-quality-and-intelligence/deterministic-quality-checks/canonical-api-cli-contract.md`
- `docs/architecture/code-quality-and-intelligence/deterministic-quality-checks/acceptance-scenarios.md`
- `docs/architecture/code-quality-and-intelligence/deterministic-quality-checks/readiness-review.md`

## What to build

Create the first end-to-end quality path: a versioned `QualityProfile`, an
open metric/rule catalog, profile validation, the optional
`arch-view.quality/v1` report attachment, and a generic evaluation entry point.
Register and evaluate `source:file.max-lines` using the source-index file
line-count fact, explicit threshold semantics, scope-qualified evidence, and
coverage. The report must serialize beside existing analysis/model data while
remaining safely omitted for legacy results.

This is the tracer-bullet slice for the quality engine. It establishes the
strategy and report boundary without adding a viewer, MCP transport, or
quality configuration UI.

## Acceptance criteria

- [x] `QualityProfile`, catalog entries, `QualityEvaluation`, coverage, metric,
  finding, diagnostic, and optional `quality_report` types follow
  `arch-view.quality/v1` and preserve the existing model/analyzer contracts.
- [x] Metric providers and quality rules register additively; duplicate
  same-version registration is idempotent and conflicting implementations are
  rejected without a central language/rule switch.
- [x] Profile validation rejects unknown rules, unsupported versions, malformed
  typed parameters, invalid threshold operators/units, and invalid profile
  identity with structured diagnostics before evaluation.
- [x] `source:file.max-lines` compares the intrinsic source-index line count
  using the configured operator and limit, emits an exact finding only for a
  true predicate, and includes metric, file, scope, rule/version, severity,
  and provenance evidence.
- [x] Missing, partial, unknown, and unsupported source coverage produces
  explicit coverage/not-evaluable output rather than a clean pass or an
  inferred violation.
- [x] Equal inputs produce a valid report representation and a legacy result
  with no quality report remains valid and unchanged for existing consumers.
- [x] Focused backend/model serialization tests cover DQC-AC-001, the catalog
  portion of DQC-AC-013, and optional-report compatibility.

## Artifact sync required

- Application PRD: `none` — this implements the already synchronized quality
  report scope and adds no product behavior beyond the specified contract.
- Application architecture summary: `none` — the optional quality sibling and
  source-index/model boundary are already documented.
- Owning capability node/artifacts: `required: /.okf/capabilities/code-quality-and-intelligence/deterministic-quality-checks.md` and `docs/architecture/code-quality-and-intelligence/deterministic-quality-checks/orchestration-status.md`.
- Issue registry: `required`; owning node `issues:` reference: `required when .okf exists`.
- OKF index/log: `required` when the issue is implemented or the delivery batch
  is otherwise synchronized; no capability-state transition is claimed by
  this issue alone.
- Reason/no-impact decision: delivery and capability progress change, but
  product scope, topology, and cross-capability ownership remain unchanged.

## Human review gate

None. The slice is verified through model, catalog, validation, evaluation,
serialization, and compatibility tests.

## Blocked by

None — the source-index contract, canonical model, and application synthesis
already exist.

## Specification anchors

- DQC-FR-001 through DQC-FR-008, DQC-FR-011, and DQC-FR-013.
- DQC-AC-001, DQC-AC-004, and DQC-AC-013.
- `QualityProfile`, `QualityEvaluation`, `MetricFact`, `QualityFinding`, and
  `QualityCoverage` in the canonical domain model.

## User stories addressed

The capability PRD has no numbered user-story section. This slice supports the
maintainer and rule-author portions of application Journey 13 by establishing a
validated profile and the first evidence-backed file-size result.

## Verification obligations

- Policy source: `/.okf/project.md`
- Backend boundary: `when-supported` for profile/catalog validation,
  file-threshold evaluation, coverage, and optional report serialization.
- Frontend integration: `not-applicable`.
- End-to-end: `when-supported` for analysis/source-index facts reaching the
  quality report.

## Completion evidence

- `go test ./internal/quality ./internal/model -count=1`
- Report validation, catalog idempotency/conflict, file-size coverage, and
  optional-report compatibility tests pass.

## Handoff

The profile/catalog and report foundation are complete. Follow-on work is
tracked in issues 059–063.
