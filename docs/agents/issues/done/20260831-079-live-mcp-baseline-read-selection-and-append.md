# 079 — Expose managed baselines through live analysis and MCP

## Issue Metadata

- Issue number: `079`
- Owning capability node: `/.okf/capabilities/code-quality-and-intelligence/live-analysis-and-mcp.md`
- Related capability nodes: `/.okf/capabilities/code-quality-and-intelligence/deterministic-quality-checks.md`, `/.okf/capabilities/export-and-automate.md`
- Artifact root: `docs/architecture/code-quality-and-intelligence/live-analysis-and-mcp/`
- Issue file: `docs/agents/issues/done/20260831-079-live-mcp-baseline-read-selection-and-append.md`
- Category: `feature`
- Execution type: `AFK`
- Review gate: `none`
- Suggested state: `done`

## Parent PRD

`docs/architecture/code-quality-and-intelligence/live-analysis-and-mcp/prd.md`

## Parent artifacts

- `docs/prd.md`
- `docs/architecture/application-architecture-summary.md`
- `docs/architecture/code-quality-and-intelligence/live-analysis-and-mcp/canonical-domain-model.md`
- `docs/architecture/code-quality-and-intelligence/live-analysis-and-mcp/canonical-api-cli-contract.md`
- `docs/architecture/code-quality-and-intelligence/live-analysis-and-mcp/acceptance-scenarios.md`
- `docs/architecture/code-quality-and-intelligence/deterministic-quality-checks/canonical-api-cli-contract.md`

## What to build

Expose the managed baseline lifecycle through the analyzer-neutral live/MCP
surface. Add read-only `get_quality_baselines` with safe filename filters,
bounded entries, and pagination. Add `baseline_mode` to temporary evaluation:
`profile` loads the exact profile reference, `none` suppresses nothing, and
`selected` reads an explicit request-scoped list without changing session
state.

Add authorized `append_baseline`. It must use the current compatible report,
delegate merge/revision/profile-reference behavior to the deterministic-quality
policy service, return added/existing finding keys, audit data, and an explicit
reevaluation-needed result. Writes remain disabled by default. Startup live
evaluation and temporary MCP evaluation must use the same baseline loader.

Update the MCP skill and public documentation so an agent knows the baseline
fields and workflow without reading repository implementation code.

## Acceptance criteria

- [x] `get_quality_baselines` lists deterministic baseline IDs, revisions,
  statuses, entry counts, and bounded entries with cursors; unsafe nested or
  traversal filenames are rejected.
- [x] `evaluate_quality` supports `profile`, `none`, and `selected` baseline
  modes. Selected files affect one request only and compatible selected files
  share one baseline identity/revision.
- [x] Default sessions reject `append_baseline`; an explicitly allowed and
  authorized request validates current report/profile/rule/formula identity,
  delegates the merge, and returns audit plus reevaluation metadata.
- [x] Startup quality evaluation and temporary evaluation resolve profile
  baselines through the same loader and preserve missing-warning,
  invalid-error, and ambiguity behavior.
- [x] The MCP tool catalog, skill reference, website installation/tools pages,
  and live/deterministic canonical contracts document the new operations.
- [x] Tests cover baseline listing/reading, pagination, selected temporary
  suppression, append authorization, profile-reference update, and source
  non-mutation.

## Artifact sync required

- Application PRD: `required: docs/prd.md`.
- Application architecture summary: `required: docs/architecture/application-architecture-summary.md`.
- Owning capability node/artifacts: `required: /.okf/capabilities/code-quality-and-intelligence/live-analysis-and-mcp.md` and its canonical artifact set.
- Deterministic-quality contract: `required` because the live surface delegates
  to its baseline loader and merge service.
- Public documentation: `required: website/mcp/` and `website/quality/`.
- Agent skill: `required: .codex/skills/arch-view/`.
- Issue registry: `required`; owning node `issues:` reference: `required`.
- OKF index/log: `required` at implementation synchronization.
- Reason/no-impact decision: this adds bounded policy read/selection and an
  explicitly authorized append; it does not add shell execution, source
  mutation, arbitrary project switching, or a second quality evaluator.

## Blocked by

077 — managed quality-baseline store and merge lifecycle.

The existing quality gateway and policy authorization boundary from issues
070–071 are also required.

## Specification anchors

- LAM-FR-010, LAM-FR-011, and LAM-FR-017 through LAM-FR-020.
- LAM-AC-023 through LAM-AC-031.
- `get_quality_baselines`, `evaluate_quality.baseline_mode`, and
  `append_baseline` MCP contracts.

## Verification obligations

- Policy source: `/.okf/project.md`
- Backend boundary: `when-supported`.
- Frontend integration: `when-supported` for the local live bridge; no new
  browser surface is required by this issue.
- End-to-end: `when-supported` for MCP stdio and the live startup path.

## Automated verification

- `go test ./internal/live ./internal/quality/... ./cmd/arch-view -count=1`
- `go test ./... -count=1`
- `go test -race ./...`
- `go vet ./...`
- `go build ./...`
- `node --check` for viewer JavaScript modules
- `npm run docs:check` from `website/`
- `npm run docs:build` from `website/`
- `git diff --check`

## Completion evidence

- `internal/live/quality.go` implements bounded baseline listing and request-
  scoped evaluation selection.
- `internal/live/baseline.go` and `internal/live/scanner.go` share the profile
  baseline loader for temporary and startup evaluation.
- `internal/live/policy.go` and `internal/live/policy_adapter.go` implement
  authorized append delegation and audit/reevaluation responses.
- `internal/live/mcp.go` advertises the closed schemas and new tools.
- `internal/live/mcp_test.go` covers listing, bounded reading, selected
  suppression, and append behavior.
- `website/mcp/installation.md`, `website/mcp/tools.md`,
  `website/mcp/agent-skill.md`, and `.codex/skills/arch-view/references/mcp.md`
  explain the new request fields and safe workflow.

## Handoff

Issue 076 remains open for its declared final cross-analyzer and human product
approval gate. This issue does not close or advance that gate.

## Verification result

Focused live/MCP tests pass. The final repository and documentation
verification for this implementation batch is recorded in the delivery report
returned with issues 077–079. No human-review gate is claimed for this
implementation issue.
