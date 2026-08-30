# 055 — Add callable complexity and nesting findings

## Issue Metadata

- Issue number: `055`
- Owning capability node: `/.okf/capabilities/code-quality-and-intelligence/deterministic-quality-checks.md`
- Related capability node: `/.okf/capabilities/code-quality-and-intelligence/source-facts-and-symbol-index.md`
- Artifact root: `docs/architecture/code-quality-and-intelligence/deterministic-quality-checks/`
- Issue file: `docs/agents/issues/done/20260830-055-callable-complexity-and-nesting-findings.md`
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

Add registered callable metric providers and exact rules for
`source:callable.max-cyclomatic-complexity` and
`source:callable.max-nesting-depth`. The first provider may be Go-first, but it
must declare its decision vocabulary, syntax basis, formula ID/version, and
capability coverage. Unsupported languages or unavailable metrics remain
explicitly unsupported/not-evaluable.

## Acceptance criteria

- [x] Cyclomatic complexity uses the provider-declared `1 + decision_points`
  formula and records the decision vocabulary, provider, formula ID/version,
  source span/hash, and provenance.
- [x] Nesting depth uses provider-defined control-flow nodes and a declared
  formula/version rather than a generic brace heuristic.
- [x] Each rule compares only a compatible versioned metric and emits an exact
  finding with metric and callable source evidence when its predicate is true.
- [x] Languages or callable shapes without a compatible provider produce
  explicit unsupported/not-evaluable coverage and no pass inferred from an
  empty metric collection.
- [x] Provider/rule registration remains additive and existing rules need no
  central language switch.
- [x] Tests cover DQC-AC-003 and DQC-AC-004, including different decision
  vocabularies, formula revisions, and unsupported coverage.

## Artifact sync required

- Application PRD: `none` — these are specified rule families with no new
  product behavior.
- Application architecture summary: `none` — provider strategy ownership and
  source-index input boundaries are already documented.
- Owning capability node/artifacts: `required: /.okf/capabilities/code-quality-and-intelligence/deterministic-quality-checks.md` and its `orchestration-status.md`.
- Issue registry: `required`; owning node `issues:` reference: `required when .okf exists`.
- OKF index/log: `required` at implementation/batch synchronization; no state
  transition is claimed by this issue alone.
- Reason/no-impact decision: delivery progress changes; topology, product
  scope, and cross-capability architecture remain unchanged.

## Human review gate

None. Provider fixtures and report-contract tests provide the required
verification.

## Blocked by

- Completed dependency: `docs/agents/issues/done/20260830-054-callable-size-findings.md`.

## Specification anchors

- DQC-FR-003 through DQC-FR-005 and DQC-FR-013.
- DQC-AC-003 and DQC-AC-004.
- `source:callable.cyclomatic_complexity` and
  `source:callable.max_nesting_depth` in the canonical formula model.

## User stories addressed

The capability PRD has no numbered user-story section. This slice supports the
developer actor in application Journey 13 by exposing reproducible complexity
and nesting risks without pretending cross-language formulas are identical.

## Verification obligations

- Policy source: `/.okf/project.md`
- Backend boundary: `when-supported`.
- Frontend integration: `not-applicable`.
- End-to-end: `when-supported` for Go fixtures and explicit unsupported
  language coverage.

## Completion evidence

- `go test ./internal/quality ./internal/analyzers/go/sourcefacts -count=1`
- Go extractor complexity/nesting fixtures and incompatible formula/provider
  coverage tests pass.

## Handoff

Callable complexity and nesting findings are complete. Issues 056–058 deliver
the remaining initial documentation and graph/constraint rule families.
