# 060 — Add conservative SOLID structural signals

## Issue Metadata

- Issue number: `060`
- Owning capability node: `/.okf/capabilities/code-quality-and-intelligence/deterministic-quality-checks.md`
- Related capability nodes: `/.okf/capabilities/code-quality-and-intelligence/source-facts-and-symbol-index.md`, `/.okf/capabilities/generate-models.md`
- Artifact root: `docs/architecture/code-quality-and-intelligence/deterministic-quality-checks/`
- Issue file: `docs/agents/issues/pending/20260830-060-solid-structural-signals.md`
- Category: `feature`
- Execution type: `AFK`
- Review gate: `none`
- Suggested state: `ready-for-agent`

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

Add registered structural signal rules for SRP, OCP, LSP, ISP, and DIP using
observable source/model facts. Every result must be explicitly classified as a
`signal`, carry deterministic indicators, evidence, heuristic provenance, and
limitations, and avoid exact-violation or proof language.

## Acceptance criteria

- [ ] All five `signal:solid.*` rules register through the same open catalog
  and emit the common quality-result shape.
- [ ] Every signal has `assessment_kind: signal`, appropriate informational
  severity, structural evidence, deterministic indicator values, and explicit
  limitations.
- [ ] No signal uses exact-violation wording, claims responsibility intent or
  substitutability is proven, or converts compiler/type facts into a SOLID
  verdict.
- [ ] Unsupported or insufficient structural facts produce explicit coverage
  or not-evaluable output without a clean pass or exact violation.
- [ ] Signal outputs participate in canonical ordering, digesting, scope
  provenance, and report comparison without changing exact-finding semantics.
- [ ] Tests cover DQC-AC-011, DQC-AC-012, conservative thresholds, evidence,
  limitations, and unsupported inputs.

## Artifact sync required

- Application PRD: `none` — the PRD already defines SOLID output as advisory
  signals and no new product boundary is introduced.
- Application architecture summary: `none` — the existing quality sibling and
  evidence ownership remain sufficient.
- Owning capability node/artifacts: `required: /.okf/capabilities/code-quality-and-intelligence/deterministic-quality-checks.md` and its `orchestration-status.md`.
- Issue registry: `required`; owning node `issues:` reference: `required when .okf exists`.
- OKF index/log: `required` at implementation/batch synchronization; no state
  transition is claimed by this issue alone.
- Reason/no-impact decision: delivery progress changes; product scope,
  topology, and architecture ownership remain unchanged.

## Human review gate

None. The signal boundary is verified through fixtures, labels, evidence, and
contract tests; presentation calibration belongs to downstream consumers.

## Blocked by

- Blocked by `docs/agents/issues/pending/20260830-059-deterministic-report-lifecycle-and-baselines.md`.

## Specification anchors

- DQC-FR-007, DQC-FR-008, DQC-FR-010, and DQC-FR-013.
- DQC-AC-011, DQC-AC-012, and DQC-AC-013.
- The SOLID signal boundary in the canonical domain model and API contract.

## User stories addressed

The capability PRD has no numbered user-story section. This slice supports the
architect/maintainer actor in application Journey 13 by exposing reviewable
structural indicators without pretending they are automated design approval.

## Verification obligations

- Policy source: `/.okf/project.md`
- Backend boundary: `when-supported`.
- Frontend integration: `not-applicable`.
- End-to-end: `when-supported` for report generation and signal provenance.

## Handoff

Issue 061 exposes exact findings, signals, coverage, and bounded evidence to
local consumers. The live/MCP capability remains outside this batch.

