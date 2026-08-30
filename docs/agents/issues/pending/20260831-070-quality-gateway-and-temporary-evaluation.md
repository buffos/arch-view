# 070 — Delegate quality queries and temporary evaluation

## Issue Metadata

- Issue number: `070`
- Owning capability node: `/.okf/capabilities/code-quality-and-intelligence/live-analysis-and-mcp.md`
- Related capability nodes: `/.okf/capabilities/code-quality-and-intelligence/deterministic-quality-checks.md`, `/.okf/capabilities/code-quality-and-intelligence/source-facts-and-symbol-index.md`
- Artifact root: `docs/architecture/code-quality-and-intelligence/live-analysis-and-mcp/`
- Issue file: `docs/agents/issues/pending/20260831-070-quality-gateway-and-temporary-evaluation.md`
- Category: `feature`
- Execution type: `AFK`
- Review gate: `none`
- Suggested state: `ready-for-agent`

## Parent PRD

`docs/architecture/code-quality-and-intelligence/live-analysis-and-mcp/prd.md`

## Parent artifacts

- `docs/prd.md`
- `docs/architecture/application-architecture-summary.md`
- `docs/architecture/code-quality-and-intelligence/live-analysis-and-mcp/canonical-domain-model.md`
- `docs/architecture/code-quality-and-intelligence/live-analysis-and-mcp/canonical-api-cli-contract.md`
- `docs/architecture/code-quality-and-intelligence/live-analysis-and-mcp/acceptance-scenarios.md`
- `docs/architecture/code-quality-and-intelligence/deterministic-quality-checks/canonical-domain-model.md`
- `docs/architecture/code-quality-and-intelligence/deterministic-quality-checks/canonical-api-cli-contract.md`

## What to build

Create the live `QualityGateway` over the implemented deterministic-quality
catalog, query, evaluation, and comparison services. Expose available
profiles/rules and their parameters, bounded findings/coverage/evidence, and
quality-report comparison through the same revision-aware query envelope.
Support `evaluate_quality` against a verified source/model revision with
optional temporary rule bindings and `persist=false`. The live layer must
delegate rule/profile/finding/baseline semantics and must not implement a
second evaluator or matcher.

## Acceptance criteria

- [ ] Profile and complete rule-catalog reads expose version, parameters,
  capability coverage, and supported/unsupported status through bounded
  results.
- [ ] Findings and evidence preserve exact versus signal assessment kind,
  severity, status, coverage, evidence references, rule/formula/provider
  versions, and selected source revision.
- [ ] `evaluate_quality` validates a profile and optional temporary bindings,
  runs the existing quality service against a verified source/model revision,
  and never persists temporary settings.
- [ ] `compare_quality_reports` delegates stable finding-key/version matching
  and distinguishes added, unchanged, suppressed, and resolved findings;
  partial/unsupported coverage cannot imply resolution.
- [ ] Strict quality requests ensure current input before evaluation; stale or
  incompatible reports are explicitly labeled/rejected as appropriate.
- [ ] Tests cover catalog/profile reads, findings/evidence pagination,
  temporary settings, scope isolation, report comparison, missing/legacy
  quality reports, and explicit unsupported coverage.

## Artifact sync required

- Application PRD: `none` — quality catalog/evaluation behavior is already
  specified in application Journey 15 and the live child contract.
- Application architecture summary: `none` — the gateway delegates to the
  implemented deterministic-quality services without changing ownership.
- Owning capability node/artifacts: `required: /.okf/capabilities/code-quality-and-intelligence/live-analysis-and-mcp.md` and its `orchestration-status.md`.
- Issue registry: `required`; owning node `issues:` reference: `required`.
- OKF index/log: `required` at implementation/batch synchronization; no state
  transition is claimed by this issue alone.
- Reason/no-impact decision: capability and delivery progress only; the
  deterministic-quality child remains the rule/profile/report owner.

## Blocked by

- `docs/agents/issues/pending/20260831-067-freshness-reconciliation-and-single-flight.md`
- `docs/agents/issues/pending/20260831-068-analyzer-neutral-query-surface.md`
- `docs/agents/issues/pending/20260831-069-exact-text-and-source-context.md`

## Specification anchors

- LAM-FR-010 and LAM-FR-012.
- LAM-AC-011 and LAM-AC-023.
- `QualityGateway`, `QualityEvaluationRequest`, `ReadQualityCatalog`,
  `EvaluateQualityRequest`, and `CompareQualityRevisions`.
- Deterministic-quality issues `061–063` and their query/evaluation contracts.

## User stories addressed

No numbered capability stories exist. This slice supports the coding
assistant/developer quality workflow in application Journey 15 and keeps the
existing quality semantics shared by viewer, CLI, and MCP.

## Verification obligations

- Policy source: `/.okf/project.md`
- Backend boundary: `when-supported`.
- Frontend integration: `not-applicable`.
- End-to-end: `when-supported` from a verified live revision to quality output.

## Automated verification

- `go test ./internal/quality/... ./internal/analysis/... -count=1`
- `go test ./... -count=1`
- `go test -race ./...`
- `go vet ./...`
- `go build ./...`
- `git diff --check`

## Handoff

Issue 071 adds the separately authorized profile-save and baseline-policy
commands over this gateway.
