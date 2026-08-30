# 072 — Add the local live-session CLI bridge

## Issue Metadata

- Issue number: `072`
- Owning capability node: `/.okf/capabilities/code-quality-and-intelligence/live-analysis-and-mcp.md`
- Related capability nodes: `/.okf/capabilities/analyze-source.md`, `/.okf/capabilities/export-and-automate.md`, `/.okf/capabilities/explore-architecture.md`
- Artifact root: `docs/architecture/code-quality-and-intelligence/live-analysis-and-mcp/`
- Issue file: `docs/agents/issues/pending/20260831-072-local-live-session-cli-bridge.md`
- Category: `feature`
- Execution type: `AFK`
- Review gate: `none`
- Suggested state: `ready-for-agent`

## Parent PRD

`docs/architecture/code-quality-and-intelligence/live-analysis-and-mcp/prd.md`

## Parent artifacts

- `docs/prd.md`
- `docs/architecture/application-architecture-summary.md`
- `docs/architecture/code-quality-and-intelligence/live-analysis-and-mcp/canonical-api-cli-contract.md`
- `docs/architecture/code-quality-and-intelligence/live-analysis-and-mcp/canonical-use-cases.md`
- `docs/architecture/code-quality-and-intelligence/live-analysis-and-mcp/acceptance-scenarios.md`

## What to build

Add a documented local CLI bridge for starting and controlling a live session.
The command family should start a long-lived session for a repository, return a
stable local endpoint/session identity, expose status and bounded wait/ensure-
current actions, and use the same session service as later viewer and MCP
adapters. The exact command and flag names must be added to the CLI reference
and covered by CLI tests. Existing one-shot `analyze` and `open` behavior must
remain compatible unless an explicit live mode is selected.

## Acceptance criteria

- [ ] A documented live command starts a configured session and exposes its
  endpoint/session identity without requiring a second terminal workflow to
  reimplement analysis or freshness logic.
- [ ] Status reports `initializing`, `ready`, `degraded`, `stale`, `updating`,
  failed, and `input_unstable` states with revision/freshness diagnostics.
- [ ] A bounded wait/ensure-current action supports `latest_ready` and
  `require_current` semantics and delegates to the server reconciliation path.
- [ ] Context cancellation, interrupt/shutdown, invalid config, and endpoint
  errors leave no orphaned watcher/session resources.
- [ ] The bridge uses the shared live/query/quality services and never executes
  the target application or silently edits source/policy files.
- [ ] CLI tests cover start/status/wait success and failure, currentness
  timeout, no-ready state, invalid roots, and compatibility of existing
  one-shot commands.

## Artifact sync required

- Application PRD: `none` — the local lifecycle journey is already specified
  and this issue exposes it through the planned CLI boundary.
- Application architecture summary: `none` — the CLI is an adapter over the
  specified live session service.
- Owning capability node/artifacts: `required: /.okf/capabilities/code-quality-and-intelligence/live-analysis-and-mcp.md` and its `orchestration-status.md`.
- Issue registry: `required`; owning node `issues:` reference: `required`.
- OKF index/log: `required` at implementation/batch synchronization; no state
  transition is claimed by this issue alone.
- Reason/no-impact decision: capability and delivery progress only; existing
  one-shot CLI product behavior remains backward-compatible.

## Blocked by

- `docs/agents/issues/pending/20260831-067-freshness-reconciliation-and-single-flight.md`

## Specification anchors

- LAM-FR-001, LAM-FR-004, LAM-FR-006, LAM-FR-012, LAM-FR-014, and LAM-FR-016.
- LAM-AC-001, LAM-AC-005, LAM-AC-006, LAM-AC-019, and LAM-AC-021.
- `StartLiveSession`, `GetLiveStatus`, `EnsureCurrentSnapshot`, and the local
  packaging rules in the canonical contract.

## User stories addressed

No numbered capability stories exist. This slice supports the CI/CLI operator
and developer who want one local command to host and inspect a live session,
and it provides the process boundary used by application Journey 14.

## Verification obligations

- Policy source: `/.okf/project.md`
- Backend boundary: `when-supported`.
- Frontend integration: `not-applicable`.
- End-to-end: `when-supported` for a long-lived local session.

## Automated verification

- `go test ./cmd/arch-view ./internal/analysis/... -count=1`
- `go test ./... -count=1`
- `go test -race ./...`
- `go vet ./...`
- `go build ./...`
- `git diff --check`

## Handoff

Issue 073 attaches the project viewer to this live session. Issue 074 uses the
same bridge for the local MCP server.
