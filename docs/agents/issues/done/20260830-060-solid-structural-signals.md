# 060 — Add conservative SOLID structural signals

## Issue Metadata

- Issue number: `060`
- Owning capability node: `/.okf/capabilities/code-quality-and-intelligence/deterministic-quality-checks.md`
- Related capability nodes: `/.okf/capabilities/code-quality-and-intelligence/source-facts-and-symbol-index.md`, `/.okf/capabilities/generate-models.md`
- Artifact root: `docs/architecture/code-quality-and-intelligence/deterministic-quality-checks/`
- Issue file: `docs/agents/issues/done/20260830-060-solid-structural-signals.md`
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

Add registered structural signal rules for SRP, OCP, LSP, ISP, and DIP using
observable source/model facts. Every result must be explicitly classified as a
`signal`, carry deterministic indicators, evidence, heuristic provenance, and
limitations, and avoid exact-violation or proof language. The Go analyzer must
publish the required structural source facts so the rules work through the
real analyzer → source index → quality evaluator path, not only in synthetic
quality fixtures.

## Acceptance criteria

- [x] All five `signal:solid.*` rules register through the same open catalog
  and emit the common quality-result shape.
- [x] Every signal has `assessment_kind: signal`, appropriate informational
  severity, structural evidence, deterministic indicator values, and explicit
  limitations.
- [x] No signal uses exact-violation wording, claims responsibility intent or
  substitutability is proven, or converts compiler/type facts into a SOLID
  verdict.
- [x] Unsupported or insufficient structural facts produce explicit coverage
  or not-evaluable output without a clean pass or exact violation.
- [x] Signal outputs participate in canonical ordering, digesting, scope
  provenance, and report comparison without changing exact-finding semantics.
- [x] Tests cover DQC-AC-011, DQC-AC-012, conservative thresholds, evidence,
  limitations, and unsupported inputs.
- [x] The Go source extractor publishes `source:solid.structure` facts and the
  Go analyzer end-to-end path evaluates all five SOLID signals without
  unsupported/not-evaluable coverage when the source is valid.

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

- Completed dependency: `docs/agents/issues/done/20260830-059-deterministic-report-lifecycle-and-baselines.md`.

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

## Completion evidence

- `go test ./internal/quality -count=1`
- SOLID signal fixtures cover all five registered rules, conservative
  thresholds, advisory wording, structural evidence, explicit limitations,
  and unsupported/partial coverage.
- `go test ./internal/analyzers/go -count=1` covers the source extractor and
  real analyzer-to-quality path; a repository analysis with
  `quality-profiles/full.json` reports `observed` coverage for SRP, OCP, LSP,
  ISP, and DIP.
- The full batch verification passed: `go test ./... -count=1`,
  `go test -race ./...`, `go vet ./...`, `go build ./...`, JavaScript syntax
  and viewer tests, and `git diff --check`.

## Handoff

Issue 061 exposes exact findings, signals, coverage, and bounded evidence to
local consumers. The live/MCP capability remains outside this batch.
