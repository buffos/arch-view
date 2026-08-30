# 073 — Integrate live sessions with the viewer

## Issue Metadata

- Issue number: `073`
- Owning capability node: `/.okf/capabilities/code-quality-and-intelligence/live-analysis-and-mcp.md`
- Related capability nodes: `/.okf/capabilities/explore-architecture.md`, `/.okf/capabilities/code-quality-and-intelligence/source-facts-and-symbol-index.md`, `/.okf/capabilities/code-quality-and-intelligence/deterministic-quality-checks.md`
- Artifact root: `docs/architecture/code-quality-and-intelligence/live-analysis-and-mcp/`
- Issue file: `docs/agents/issues/pending/20260831-073-live-viewer-integration.md`
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
- `docs/architecture/code-quality-and-intelligence/live-analysis-and-mcp/acceptance-scenarios.md`
- `docs/architecture/explore-architecture/prd.md`
- `docs/architecture/explore-architecture/canonical-api-cli-contract.md`

## What to build

Attach the existing project-backed viewer to a live session in an explicit
live/watch mode. The graph and inspection views must consume the same immutable
query/quality envelopes as CLI and MCP, show revision and freshness status,
retain the last-ready view during updates, and refresh only after a new
coherent revision is published. Existing one-shot model/project opening stays
available and does not silently start a watcher.

## Acceptance criteria

- [ ] An explicit documented live viewer mode starts or attaches to the local
  live session without changing the default one-shot `open` behavior.
- [ ] The viewer exposes session state, revision, current/stale/updating/
  degraded/unstable status, and diagnostics in accessible human-readable
  language.
- [ ] Graph, source inspection, quality findings, and scope selection refresh
  from one published revision; no mixed source/model/quality content appears.
- [ ] During a rebuild the previous ready view remains usable, and a failed or
  unstable rebuild never replaces it with an empty result.
- [ ] The viewer uses the shared bounded query/source/quality services rather
  than duplicating analyzer, freshness, or quality semantics.
- [ ] Browser tests cover live start/attach, status transitions, revision
  refresh, stale/degraded/unstable notices, keyboard focus, responsive layout,
  and backward-compatible one-shot opening.

## Artifact sync required

- Application PRD: `none` — this implements the already synchronized developer
  live-view journey; it does not add a new product concept.
- Application architecture summary: `none` — the viewer remains a consumer of
  the shared live/query/quality boundaries.
- Owning capability node/artifacts: `required: /.okf/capabilities/code-quality-and-intelligence/live-analysis-and-mcp.md` and its `orchestration-status.md`.
- Issue registry: `required`; owning node `issues:` reference: `required`.
- OKF index/log: `required` at implementation/batch synchronization; no state
  transition is claimed by this issue alone.
- Reason/no-impact decision: capability and delivery progress only; existing
  one-shot viewer behavior remains supported.

## Blocked by

- `docs/agents/issues/pending/20260831-068-analyzer-neutral-query-surface.md`
- `docs/agents/issues/pending/20260831-069-exact-text-and-source-context.md`
- `docs/agents/issues/pending/20260831-070-quality-gateway-and-temporary-evaluation.md`
- `docs/agents/issues/pending/20260831-072-local-live-session-cli-bridge.md`

## Specification anchors

- LAM-FR-006, LAM-FR-008, LAM-FR-010, LAM-FR-012, and LAM-FR-016.
- LAM-AC-005, LAM-AC-006, LAM-AC-011, LAM-AC-013, and LAM-AC-015.
- Application Journey 14 and the existing source-index/quality viewer
  boundaries.

## User stories addressed

No numbered capability stories exist. This slice supports the developer who
starts a watched project and needs the human viewer to remain understandable
while analysis changes in the background.

## Verification obligations

- Policy source: `/.okf/project.md`
- Backend boundary: `when-supported`.
- Frontend integration: `when-supported`.
- End-to-end: `when-supported` through project open, edit, reanalysis, and
  refreshed inspection.

## Automated verification

- `go test ./internal/viewer/... ./cmd/arch-view -count=1`
- `node --check` for every viewer JavaScript module
- `node --test` for every viewer JavaScript test
- `go test ./... -count=1`
- `go test -race ./...`
- `go vet ./...`
- `go build ./...`
- `git diff --check`

## Handoff

Issue 076 performs the final cross-analyzer viewer/MCP product inspection after
the transport adapters are complete.
