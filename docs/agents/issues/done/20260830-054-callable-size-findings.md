# 054 — Add callable body-size findings

## Issue Metadata

- Issue number: `054`
- Owning capability node: `/.okf/capabilities/code-quality-and-intelligence/deterministic-quality-checks.md`
- Related capability node: `/.okf/capabilities/code-quality-and-intelligence/source-facts-and-symbol-index.md`
- Artifact root: `docs/architecture/code-quality-and-intelligence/deterministic-quality-checks/`
- Issue file: `docs/agents/issues/done/20260830-054-callable-size-findings.md`
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

## What to build

Extend the quality evaluator with the exact `source:callable.max-lines` rule.
Use an extractor-supplied callable body span and a versioned formula to derive
physical body-line count. Emit evidence-backed findings only when the
configured predicate is true; when a callable has no usable body span, emit
explicit `not_evaluable` coverage instead of estimating from braces or other
syntax guesses.

## Acceptance criteria

- [x] Callable body line count is derived only from the declared source span,
  source hash, and versioned formula/provider identity.
- [x] Equality and threshold operators follow the validated profile exactly;
  no implicit inclusive/exclusive behavior is introduced.
- [x] A true predicate emits an exact finding with callable subject, observed
  metric, limit/operator, readable source evidence, rule/version, severity,
  and provenance.
- [x] Missing or invalid body spans produce `not_evaluable` coverage and a
  diagnostic without inventing a line count or clean result.
- [x] Scope-qualified subject identity and evidence remain isolated for
  combined/multi-scope reports.
- [x] Tests cover DQC-AC-002, threshold boundaries, malformed spans, and
  legacy source-index omissions.

## Artifact sync required

- Application PRD: `none` — callable-size evaluation is already in the
  synchronized rule catalog and does not change product scope.
- Application architecture summary: `none` — this adds a strategy behind the
  documented quality/source-index boundary.
- Owning capability node/artifacts: `required: /.okf/capabilities/code-quality-and-intelligence/deterministic-quality-checks.md` and its `orchestration-status.md`.
- Issue registry: `required`; owning node `issues:` reference: `required when .okf exists`.
- OKF index/log: `required` at implementation/batch synchronization; no state
  transition is claimed by this issue alone.
- Reason/no-impact decision: delivery progress changes; product boundary,
  topology, and architecture ownership do not.

## Human review gate

None. Backend and report-contract tests are sufficient.

## Blocked by

- Completed dependency: `docs/agents/issues/done/20260830-053-quality-profile-and-file-size-report.md`.

## Specification anchors

- DQC-FR-003 through DQC-FR-007.
- DQC-AC-002.
- `source:callable.body_line_count` and `source:callable.max-lines` in the
  initial rule catalog and formula model.

## User stories addressed

The capability PRD has no numbered user-story section. This slice supports the
developer actor in application Journey 13 by identifying oversized callable
implementations with precise evidence.

## Verification obligations

- Policy source: `/.okf/project.md`
- Backend boundary: `when-supported`.
- Frontend integration: `not-applicable`.
- End-to-end: `when-supported` through source-index symbols/spans and the
  optional quality report.

## Completion evidence

- `go test ./internal/quality ./internal/analyzers/go/sourcefacts -count=1`
- Callable body-span, threshold-boundary, malformed-span, and scope-qualified
  evidence tests pass.

## Handoff

Callable body-size findings are complete. Issue 055's versioned complexity and
nesting formulas are also delivered in this implementation batch.
