# 061 — Expose bounded quality report, findings, and evidence queries

## Issue Metadata

- Issue number: `061`
- Owning capability node: `/.okf/capabilities/code-quality-and-intelligence/deterministic-quality-checks.md`
- Related capability nodes: `/.okf/capabilities/code-quality-and-intelligence/source-facts-and-symbol-index.md`, `/.okf/capabilities/explore-architecture.md`
- Artifact root: `docs/architecture/code-quality-and-intelligence/deterministic-quality-checks/`
- Issue file: `docs/agents/issues/done/20260830-061-quality-report-query-and-evidence.md`
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

Add the bounded `QualityQueryService` and local HTTP/read-model adapters for
quality reports. Support listing reports/coverage/findings with scope, subject,
rule, assessment kind, severity, status, and cursor/limit filters, plus
`GetFindingEvidence` with explicit source-context budgets. Preserve compact
evidence and keep full source context opt-in, read-only, scope-bound, and
path-safe.

## Acceptance criteria

- [x] `ListFindings`, `GetQualityCoverage`, and report retrieval return stable
  envelopes with bounded pagination and the contract's deterministic default
  ordering.
- [x] Finding filters support report/snapshot, scope, subject, rule,
  assessment kind, severity, status, and the file subject filter needed by
  `source:file.max-lines` consumers.
- [x] Evidence queries return source spans, entity/relationship refs, metric
  refs, diagnostics, provenance, and limitations without including complete
  source context by default.
- [x] Explicit source-context requests use the existing read-only, path-safe,
  bounded source boundary and reject invalid or over-budget requests with the
  common error shape.
- [x] Scope, coverage, suppressed/baseline status, signal limitations, and
  partial/not-evaluable state remain visible through the query boundary.
- [x] Local HTTP routes and service tests cover pagination, invalid selectors,
  scope isolation, evidence budgets, legacy/no-report behavior, and all report
  result kinds.

## Artifact sync required

- Application PRD: `none` — bounded quality queries are already described as a
  consumer boundary in the synchronized contract.
- Application architecture summary: `none` — this implements the documented
  quality-report read model and existing safe source boundary.
- Owning capability node/artifacts: `required: /.okf/capabilities/code-quality-and-intelligence/deterministic-quality-checks.md` and its `orchestration-status.md`.
- Issue registry: `required`; owning node `issues:` reference: `required when .okf exists`.
- OKF index/log: `required` at implementation/batch synchronization; no state
  transition is claimed by this issue alone.
- Reason/no-impact decision: delivery progress changes; no MCP transport,
  source mutation, product boundary, or topology change is introduced.

## Human review gate

None. Query, envelope, pagination, scope, and evidence-boundary tests provide
the required verification.

## Blocked by

- Completed dependency: `docs/agents/issues/done/20260830-060-solid-structural-signals.md`.

## Specification anchors

- `ListFindings`, `GetFindingEvidence`, and `GetQualityCoverage`.
- DQC-FR-005, DQC-FR-007, DQC-FR-011, and DQC-FR-012.
- The bounded query and evidence rules in the canonical API/CLI contract.

## User stories addressed

The capability PRD has no numbered user-story section. This slice supports the
developer, maintainer, and future viewer/export actors in application Journey
13 by making the same bounded report readable by multiple consumers.

## Verification obligations

- Policy source: `/.okf/project.md`
- Backend boundary: `when-supported`.
- Frontend integration: `not-applicable`.
- End-to-end: `when-supported` through local HTTP/read-model consumers.

## Completion evidence

- `go test ./internal/quality ./internal/viewer -count=1`
- Query tests cover deterministic pagination, report/scope/file filters,
  evidence references, explicit source-context budgets, invalid limits, and
  missing findings.
- Viewer HTTP tests cover available and missing reports, scope selection,
  findings/coverage routes, bounded evidence, and safe source excerpts.
- The full batch verification passed: `go test ./... -count=1`,
  `go test -race ./...`, `go vet ./...`, `go build ./...`, JavaScript syntax
  and viewer tests, and `git diff --check`.

## Handoff

Issue 062 consumed this boundary for headless and export output. Issue 063
consumes it for the human-oriented viewer projection.
