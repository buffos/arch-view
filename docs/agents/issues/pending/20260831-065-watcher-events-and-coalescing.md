# 065 — Normalize and coalesce watcher events

## Issue Metadata

- Issue number: `065`
- Owning capability node: `/.okf/capabilities/code-quality-and-intelligence/live-analysis-and-mcp.md`
- Related capability nodes: `/.okf/capabilities/analyze-source.md`, `/.okf/capabilities/analyze-source/plugin-runtime/multi-analyzer-orchestration.md`
- Artifact root: `docs/architecture/code-quality-and-intelligence/live-analysis-and-mcp/`
- Issue file: `docs/agents/issues/pending/20260831-065-watcher-events-and-coalescing.md`
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
- `docs/architecture/code-quality-and-intelligence/live-analysis-and-mcp/canonical-use-cases.md`
- `docs/architecture/code-quality-and-intelligence/live-analysis-and-mcp/acceptance-scenarios.md`

## What to build

Add the replaceable filesystem-watcher boundary for a live session. Normalize
create/modify/delete/rename events to repository-relative root-safe paths,
deduplicate repeated paths, and emit one bounded event group after the debounce
window. Overflow, backend errors, ambiguous renames, out-of-root paths, and
missed sequences must become explicit rescan triggers rather than guessed
incremental changes. Include a deterministic fake backend for tests and one
supported local watcher implementation behind the same port.

## Acceptance criteria

- [ ] Watcher events normalize path separators, reject root escapes, and retain
  event kind and backend sequence information where available.
- [ ] Repeated edits to the same path and related paths within the configured
  debounce window produce one canonical event group with bounded memory.
- [ ] Create/modify/delete/rename events preserve enough information for safe
  invalidation; ambiguous rename is represented as delete/create or a full
  rescan trigger.
- [ ] Overflow, backend error, out-of-scope path, and missed-sequence signals
  force an explicit full rescan plan and never commit guessed facts.
- [ ] The watcher implementation is replaceable and tests can inject a fake
  backend without sleeping on wall-clock timing.
- [ ] Tests cover edit storms, rename/delete pairs, overflow, path traversal,
  debounce bounds, cancellation, and backend failure.

## Artifact sync required

- Application PRD: `none` — the configured-folder watching journey is already
  specified and this issue does not expand it.
- Application architecture summary: `none` — the watcher port and ownership
  boundary are already specified; update before closure only if implementation
  changes them.
- Owning capability node/artifacts: `required: /.okf/capabilities/code-quality-and-intelligence/live-analysis-and-mcp.md` and its `orchestration-status.md`.
- Issue registry: `required`; owning node `issues:` reference: `required`.
- OKF index/log: `required` at implementation/batch synchronization; no state
  transition is claimed by this issue alone.
- Reason/no-impact decision: capability and delivery progress only; no product
  or topology change.

## Blocked by

- Completed dependency: `docs/agents/issues/pending/20260831-064-live-session-config-and-initial-snapshot.md`.

## Specification anchors

- LAM-FR-002, LAM-FR-003, and LAM-FR-015.
- LAM-AC-003 and LAM-AC-004.
- `WatchEvent`, `NormalizedEvent`, `NormalizeWatchEvent`, `CoalesceEvents`,
  and `InvalidationPlan` in the canonical models.

## User stories addressed

No numbered capability stories exist. This slice supports the developer actor
who edits files while the live session is running and the agent workflow in
application Journey 14.

## Verification obligations

- Policy source: `/.okf/project.md`
- Backend boundary: `when-supported`.
- Frontend integration: `not-applicable`.
- End-to-end: `when-supported` for a watched temporary repository.

## Automated verification

- `go test ./internal/analysis/... ./internal/analysis/orchestration/... -count=1`
- `go test ./... -count=1`
- `go test -race ./...`
- `go vet ./...`
- `git diff --check`

## Handoff

Issue 066 consumes event groups to build and publish coherent revisions. Issue
067 adds authoritative request-time reconciliation because watcher events alone
are not proof of freshness.
