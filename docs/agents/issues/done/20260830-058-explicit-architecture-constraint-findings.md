# 058 — Add explicit architecture constraint findings

## Issue Metadata

- Issue number: `058`
- Owning capability node: `/.okf/capabilities/code-quality-and-intelligence/deterministic-quality-checks.md`
- Related capability node: `/.okf/capabilities/generate-models.md`
- Artifact root: `docs/architecture/code-quality-and-intelligence/deterministic-quality-checks/`
- Issue file: `docs/agents/issues/done/20260830-058-explicit-architecture-constraint-findings.md`
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

Evaluate `architecture:forbidden-dependency` and
`architecture:layer-direction` from explicit, validated profile constraints.
Resolve selectors only from declared module IDs, stable configured patterns,
tags, or layer assignments. Evaluate only reported graph relationships and
preserve not-evaluable coverage when policy or layer data is absent.

## Acceptance criteria

- [x] Quality profiles validate forbidden-edge and layer-direction constraint
  kinds, typed parameters, selectors, scope, and policy identity before
  evaluation.
- [x] Forbidden-dependency findings report only matching reported edges and
  include source/target module, relationship, rule, policy, and provenance
  evidence.
- [x] Layer-direction findings compare explicit layer assignments and configured
  direction; the engine never infers intended layers or dependency policy from
  names/directories.
- [x] Missing policy or layer assignment produces disabled/not-evaluable
  coverage and diagnostics rather than a clean pass.
- [x] Cross-scope constraints require an explicit aggregate graph and preserve
  scope provenance.
- [x] Tests cover DQC-AC-007, selector validation, reported-edge boundaries,
  absent policy, and mixed-scope behavior.

## Artifact sync required

- Application PRD: `none` — explicit architecture constraints are already in
  the synchronized quality scope; no configuration UI is introduced here.
- Application architecture summary: `none` — constraint evaluation remains a
  quality sibling over canonical graph facts as already documented.
- Owning capability node/artifacts: `required: /.okf/capabilities/code-quality-and-intelligence/deterministic-quality-checks.md` and its `orchestration-status.md`.
- Issue registry: `required`; owning node `issues:` reference: `required when .okf exists`.
- OKF index/log: `required` at implementation/batch synchronization; no state
  transition is claimed by this issue alone.
- Reason/no-impact decision: delivery progress changes; product boundary,
  topology, and architecture ownership remain unchanged.

## Human review gate

None. Explicit policy fixtures and report tests provide the required evidence.

## Blocked by

- Completed dependency: `docs/agents/issues/done/20260830-057-graph-coupling-and-cycle-findings.md`.

## Specification anchors

- DQC-FR-002, DQC-FR-004, DQC-FR-005, DQC-FR-007, and DQC-FR-011.
- DQC-AC-007.
- `ArchitectureConstraint`, `architecture:forbidden-dependency`, and
  `architecture:layer-direction` in the canonical model and contract.

## User stories addressed

The capability PRD has no numbered user-story section. This slice supports the
maintainer/architect actor in application Journey 13 by checking declared
architecture intent without presenting inferred design judgments as facts.

## Verification obligations

- Policy source: `/.okf/project.md`
- Backend boundary: `when-supported`.
- Frontend integration: `not-applicable`.
- End-to-end: `when-supported` from canonical graph and explicit profile
  constraints to report findings.

## Completion evidence

- `go test ./internal/quality -count=1`
- Explicit selector/policy validation, forbidden-edge, exact layer-direction,
  missing-policy, and mixed-scope tests pass.

## Handoff

Explicit architecture constraint findings are complete. Issue 059 will harden
report lifecycle and finding identity across all exact rule families delivered
by this batch.
