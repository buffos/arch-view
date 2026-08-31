# 078 — Add automatic baseline loading and managed CLI append

## Issue Metadata

- Issue number: `078`
- Owning capability node: `/.okf/capabilities/code-quality-and-intelligence/deterministic-quality-checks.md`
- Related capability nodes: `/.okf/capabilities/export-and-automate.md`, `/.okf/capabilities/code-quality-and-intelligence/live-analysis-and-mcp.md`
- Artifact root: `docs/architecture/code-quality-and-intelligence/deterministic-quality-checks/`
- Issue file: `docs/agents/issues/done/20260831-078-managed-quality-baseline-cli.md`
- Category: `feature`
- Execution type: `AFK`
- Review gate: `none`
- Suggested state: `done`

## Parent PRD

`docs/architecture/code-quality-and-intelligence/deterministic-quality-checks/prd.md`

## Parent artifacts

- `docs/prd.md`
- `docs/architecture/application-architecture-summary.md`
- `docs/architecture/code-quality-and-intelligence/deterministic-quality-checks/canonical-api-cli-contract.md`
- `docs/architecture/code-quality-and-intelligence/deterministic-quality-checks/acceptance-scenarios.md`

## What to build

Make normal profile-based CLI analysis discover the profile's exact canonical
baseline automatically. Keep `--quality-baseline` as an explicit one-run
override and add `--no-quality-baseline` for a clean report.

Add `arch-view quality baseline add` for the repeatable review loop. It reads a
current report, selects one or more reviewed findings, creates or merges the
direct baseline file, updates the profile reference, reports added versus
existing keys, and tells the caller to run analysis again. It must not edit
source code, overwrite an unrelated document, or accept ineligible coverage.
Keep the older standalone `quality baseline` command intact.

## Acceptance criteria

- [x] `analyze --quality-profile ...` loads the exact referenced baseline from
  the target project's `quality-baselines/` directory without requiring a
  baseline flag on every run.
- [x] A missing automatic baseline emits a clear warning and leaves findings
  unsuppressed; invalid or ambiguous resolution fails.
- [x] `--quality-baseline` applies only the explicit file for that run, and
  `--no-quality-baseline` disables suppression for that run; neither changes
  the saved profile.
- [x] `quality baseline add` creates the canonical file when needed, derives a
  stable first ID from its filename when no ID is supplied, merges without
  duplicates, updates the profile reference, and supports expected/explicit
  revisions.
- [x] The command rejects non-active or non-observed findings and keeps the
  legacy standalone command available.
- [x] CLI tests prove first-run active findings, automatic suppression after
  append, and clean output with `--no-quality-baseline`.

## Artifact sync required

- Application PRD: `required: docs/prd.md`.
- Application architecture summary: `required: docs/architecture/application-architecture-summary.md`.
- Owning capability node/artifacts: `required: /.okf/capabilities/code-quality-and-intelligence/deterministic-quality-checks.md`.
- Public documentation: `required: website/cli/` and `website/quality/`.
- Agent skill references: `required: .codex/skills/arch-view/`.
- Issue registry: `required`; owning node `issues:` reference: `required`.
- OKF index/log: `required` at implementation synchronization.
- Reason/no-impact decision: the command adds a managed workflow while
  preserving the old standalone command and the existing analysis JSON shape.

## Blocked by

077 — managed quality-baseline store and merge lifecycle.

## Specification anchors

- DQC-FR-015 through DQC-FR-019.
- DQC-AC-017, DQC-AC-018, DQC-AC-020, and DQC-AC-021.
- `arch-view analyze` and `arch-view quality baseline add` CLI contracts.

## Verification obligations

- Policy source: `/.okf/project.md`
- Backend boundary: `when-supported`.
- Frontend integration: `not-applicable`.
- End-to-end: `when-supported` for a temporary project with managed policy
  files and a generated analysis report.

## Automated verification

- `go test ./cmd/arch-view ./internal/quality/... -count=1`
- `go test ./... -count=1`
- `go test -race ./...`
- `go vet ./...`
- `go build ./...`
- `node --check` for viewer JavaScript modules
- `git diff --check`

## Completion evidence

- `cmd/arch-view/quality.go` resolves profile references and implements the
  explicit/disabled baseline selectors.
- `cmd/arch-view/quality_baseline_add.go` implements managed create/merge,
  safe destinations, eligibility checks, and structured output.
- `cmd/arch-view/quality_test.go` proves automatic suppression after the first
  managed append and clean evaluation with `--no-quality-baseline`.
- `website/cli/analyze.md`, `website/cli/quality-baseline.md`,
  `website/quality/baselines.md`, and the CLI skill references describe the
  normal workflow and its options.

## Handoff

Issue 079 consumes the same baseline loader for live startup and MCP
evaluation.

## Verification result

Focused CLI and quality tests pass. The final repository and documentation
verification for this implementation batch is recorded in the delivery report
returned with issues 077–079.
