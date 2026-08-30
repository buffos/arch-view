# 056 — Add public-symbol documentation coverage findings

## Issue Metadata

- Issue number: `056`
- Owning capability node: `/.okf/capabilities/code-quality-and-intelligence/deterministic-quality-checks.md`
- Related capability node: `/.okf/capabilities/code-quality-and-intelligence/source-facts-and-symbol-index.md`
- Artifact root: `docs/architecture/code-quality-and-intelligence/deterministic-quality-checks/`
- Issue file: `docs/agents/issues/done/20260830-056-documentation-coverage-findings.md`
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

Add the exact `source:public-symbol.documentation` coverage rule. Consume
source-index visibility and documentation states without judging prose quality.
Count only eligible subjects whose visibility and documentation capabilities are
observed, and preserve unsupported, unknown, partial, and unavailable subjects
in coverage rather than relabeling them as undocumented.

## Acceptance criteria

- [x] The rule evaluates only public/eligible subjects with observed visibility
  and supported documentation capability.
- [x] Documented and undocumented counts use the specified denominator;
  unknown/unsupported visibility or documentation remains visible in coverage
  and is not silently counted as missing documentation.
- [x] Coverage includes a versioned metric fact, formula/provenance, scope, and
  subject counts; findings remain distinct from analyzer diagnostics.
- [x] A configured threshold or coverage predicate emits exact evidence-backed
  results only when the required facts are observed.
- [x] Combined reports preserve per-scope documentation coverage and do not
  infer documentation from names, paths, or comments not reported by the
  source-index extractor.
- [x] Tests cover DQC-AC-005 and empty, partial, unsupported, and unknown
  documentation/visibility cases.

## Artifact sync required

- Application PRD: `none` — documentation coverage is already specified as a
  deterministic presence check, not a new product boundary.
- Application architecture summary: `none` — the source-index input and
  quality sibling boundary are already synchronized.
- Owning capability node/artifacts: `required: /.okf/capabilities/code-quality-and-intelligence/deterministic-quality-checks.md` and its `orchestration-status.md`.
- Issue registry: `required`; owning node `issues:` reference: `required when .okf exists`.
- OKF index/log: `required` at implementation/batch synchronization; no state
  transition is claimed by this issue alone.
- Reason/no-impact decision: delivery progress changes; product scope,
  topology, and architecture ownership remain unchanged.

## Human review gate

None. Coverage fixtures and report tests provide the required verification.

## Blocked by

- Completed dependency: `docs/agents/issues/done/20260830-053-quality-profile-and-file-size-report.md`.

## Specification anchors

- DQC-FR-003 through DQC-FR-005 and DQC-FR-007.
- DQC-AC-005.
- `source:documentation.coverage` and the source-index documentation/visibility
  states in the canonical domain model.

## User stories addressed

The capability PRD has no numbered user-story section. This slice supports the
developer and maintainer actors in application Journey 13 by making missing
public documentation measurable without claiming comment quality.

## Verification obligations

- Policy source: `/.okf/project.md`
- Backend boundary: `when-supported`.
- Frontend integration: `not-applicable`.
- End-to-end: `when-supported` through source-index documentation and
  visibility records.

## Completion evidence

- `go test ./internal/quality -count=1`
- Documentation coverage tests pass for observed, absent, partial,
  unsupported, unknown, and scope-isolated visibility/documentation states.

## Handoff

Documentation coverage findings are complete. Issue 059 will apply report
lifecycle, deterministic identity, and baseline semantics across these rule
families.
