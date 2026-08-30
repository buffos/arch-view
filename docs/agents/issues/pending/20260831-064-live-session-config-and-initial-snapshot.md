# 064 — Configure and start a live session

## Issue Metadata

- Issue number: `064`
- Owning capability node: `/.okf/capabilities/code-quality-and-intelligence/live-analysis-and-mcp.md`
- Related capability nodes: `/.okf/capabilities/analyze-source.md`, `/.okf/capabilities/code-quality-and-intelligence/source-facts-and-symbol-index.md`, `/.okf/capabilities/code-quality-and-intelligence/deterministic-quality-checks.md`
- Artifact root: `docs/architecture/code-quality-and-intelligence/live-analysis-and-mcp/`
- Issue file: `docs/agents/issues/pending/20260831-064-live-session-config-and-initial-snapshot.md`
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

Create the validated live-session service and its initial full-scan path. A
caller supplies a versioned session configuration with repository-relative
roots, registered analyzer/source capabilities, quality-profile reference,
freshness/resource limits, and operation permissions. The service validates it
fail-closed, starts a long-lived session, runs the existing multi-analyzer
orchestration across all registered analyzers, and exposes structured
`initializing`, `ready`, `degraded`, and `failed` status through a small service
boundary that later CLI, viewer, and MCP adapters can share.

## Acceptance criteria

- [ ] `arch-view.live/v1` configuration validates repository-relative roots,
  analyzer/source capability requests, quality-profile references, positive
  bounded limits, and read-only/no-shell/no-target defaults.
- [ ] Absolute roots, traversal, symlink/junction escapes, invalid operation
  permissions, and unsupported configuration fail closed with structured
  diagnostics and no session access.
- [ ] A valid session starts one initial bounded scan through the existing
  multi-analyzer planner/orchestrator and does not contain a Go-specific live
  branch.
- [ ] The service reports `initializing` while the scan runs, publishes a
  usable `ready` or `degraded` result only through the shared snapshot boundary,
  and reports a structured failure when no usable revision exists.
- [ ] Legacy models or scopes without a source index expose explicit
  unavailable/unsupported coverage rather than fabricated source facts.
- [ ] Unit/service tests cover valid and invalid configuration, all registered
  analyzer registration, initial status transitions, cancellation, and
  no-ready failure behavior.

## Artifact sync required

- Application PRD: `none` — this implements the already synchronized live
  session journey and does not change its product boundary.
- Application architecture summary: `none` — the coordinator/session port is
  already specified; implementation must update it before closure only if the
  realized boundary differs.
- Owning capability node/artifacts: `required: /.okf/capabilities/code-quality-and-intelligence/live-analysis-and-mcp.md` and its `orchestration-status.md`.
- Issue registry: `required`; owning node `issues:` reference: `required`.
- OKF index/log: `required` at implementation/batch synchronization; no state
  transition is claimed by this issue alone.
- Reason/no-impact decision: capability and delivery progress only; topology,
  product scope, and cross-capability ownership remain unchanged.

## Blocked by

None - can start immediately. Completed prerequisites are the source-index
issues `048–052` and deterministic-quality issues `053–063`.

## Specification anchors

- LAM-FR-001, LAM-FR-005, LAM-FR-006, LAM-FR-008, and LAM-FR-015.
- LAM-AC-001 and LAM-AC-002.
- `LiveSessionConfig`, `LiveSnapshot`, and `ValidateLiveSession`/
  `StartLiveSession` in the canonical domain/use-case models.

## User stories addressed

The capability PRD has no numbered user-story section. This slice supports the
developer and CI/CLI operator actors and establishes the live-session boundary
used by application Journey 14.

## Verification obligations

- Policy source: `/.okf/project.md`
- Backend boundary: `when-supported`.
- Frontend integration: `not-applicable`.
- End-to-end: `when-supported` for the initial multi-analyzer scan.

## Automated verification

- `go test ./internal/analysis/... ./internal/analysis/orchestration/... -count=1`
- `go test ./... -count=1`
- `go vet ./...`
- `go build ./...`
- `git diff --check`

## Handoff

Issue 065 adds watcher input to this session. Issue 066 hardens its snapshot
publication and last-ready behavior.
