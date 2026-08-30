# 067 — Reconcile freshness and coalesce rebuilds

## Issue Metadata

- Issue number: `067`
- Owning capability node: `/.okf/capabilities/code-quality-and-intelligence/live-analysis-and-mcp.md`
- Related capability nodes: `/.okf/capabilities/analyze-source/plugin-runtime/multi-analyzer-orchestration.md`, `/.okf/capabilities/code-quality-and-intelligence/source-facts-and-symbol-index.md`, `/.okf/capabilities/code-quality-and-intelligence/deterministic-quality-checks.md`
- Artifact root: `docs/architecture/code-quality-and-intelligence/live-analysis-and-mcp/`
- Issue file: `docs/agents/issues/pending/20260831-067-freshness-reconciliation-and-single-flight.md`
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

## What to build

Make freshness a server-enforced contract. `latest_ready` returns the last
committed revision quickly with explicit stale/updating state. Every
`require_current` request performs request-time reconciliation against an
authoritative eligible-source input fingerprint; a clean watcher or ambiguous
metadata-only manifest is not enough. When the input changed, start or join
one bounded rebuild, verify the source before and after analysis, retry a
bounded number of times, and return `input_unstable` when edits do not settle.
Use existing cache/source-scope identity for selective invalidation only when
dependency impact is proven; otherwise broaden the rescan.

## Acceptance criteria

- [ ] `latest_ready` and `require_current` have distinct, explicit response
  semantics and never claim current data from watcher state alone.
- [ ] `require_current` performs authoritative reconciliation inside the
  request path, including when no watcher event is pending.
- [ ] Concurrent strict requests for the same target join one single-flight
  reconciliation/build and receive the same published revision or the same
  explicit failure.
- [ ] Candidate input is verified before and after analysis. A change during
  analysis discards the candidate and retries within policy; exhausted retries
  preserve the last-ready revision and return `input_unstable`.
- [ ] Selective invalidation uses explicit scope/cache/dependency identity only
  when safe; uncertain impact schedules a broader rescan.
- [ ] Periodic reconciliation protects against missed watcher events without
  publishing duplicate semantic revisions for unchanged input.
- [ ] Tests cover missed events, clean-but-changed metadata, edit storms,
  concurrent strict requests, source changes during analysis, retry limits,
  selective invalidation, and timeout behavior.

## Artifact sync required

- Application PRD: `none` — the current/stale/unstable behavior is already
  specified in application Journey 14.
- Application architecture summary: `none` — reconciliation and single-flight
  ownership are already defined; update before closure only if the realized
  architecture changes.
- Owning capability node/artifacts: `required: /.okf/capabilities/code-quality-and-intelligence/live-analysis-and-mcp.md` and its `orchestration-status.md`.
- Issue registry: `required`; owning node `issues:` reference: `required`.
- OKF index/log: `required` at implementation/batch synchronization; no state
  transition is claimed by this issue alone.
- Reason/no-impact decision: capability and delivery progress only; product
  scope and topology are unchanged.

## Blocked by

- `docs/agents/issues/pending/20260831-066-coherent-revision-store-and-publication.md`

## Specification anchors

- LAM-FR-004, LAM-FR-005, LAM-FR-006, LAM-FR-007, and LAM-FR-015.
- LAM-AC-008, LAM-AC-019, LAM-AC-020, and LAM-AC-021.
- `FreshnessPolicy`, `ReconciliationResult`, `InputVerification`,
  `EnsureCurrentSnapshot`, and `ReconcileSourceState`.

## User stories addressed

No numbered capability stories exist. This slice directly supports the coding
assistant's current-data workflow and the developer's edit-and-see-updated-
analysis workflow in application Journey 14.

## Verification obligations

- Policy source: `/.okf/project.md`
- Backend boundary: `when-supported`.
- Frontend integration: `not-applicable`.
- End-to-end: `when-supported` for watcher, polling, and strict query paths.

## Automated verification

- `go test ./internal/analysis/... ./internal/analysis/orchestration/... -count=1`
- `go test ./... -count=1`
- `go test -race ./...`
- `go vet ./...`
- `go build ./...`
- `git diff --check`

## Handoff

Issue 068 can safely expose revision-bound analyzer-neutral queries. Issue 072
can build the local lifecycle/status bridge on this freshness service.
