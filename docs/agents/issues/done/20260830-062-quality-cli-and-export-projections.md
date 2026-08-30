# 062 — Add headless quality-report and export projections

## Issue Metadata

- Issue number: `062`
- Owning capability node: `/.okf/capabilities/code-quality-and-intelligence/deterministic-quality-checks.md`
- Related capability nodes: `/.okf/capabilities/export-and-automate.md`, `/.okf/capabilities/explore-architecture.md`
- Artifact root: `docs/architecture/code-quality-and-intelligence/deterministic-quality-checks/`
- Issue file: `docs/agents/issues/done/20260830-062-quality-cli-and-export-projections.md`
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
- `docs/architecture/export-and-automate/prd.md`
- `docs/architecture/export-and-automate/canonical-api-cli-contract.md`

## What to build

Allow headless analysis to accept a versioned quality profile and emit the same
complete `quality_report` used by local query/viewer consumers. Extend JSON,
HTML, and SVG projections to carry or annotate quality results without
changing architecture semantics. Add caller-owned exit-policy evaluation over
finding severity/status while preserving every finding, coverage result, and
baseline/suppression status in the report.

## Acceptance criteria

- [x] Headless analysis accepts an explicit versioned quality profile and
  produces the canonical optional quality report for the selected scope(s).
- [x] JSON output preserves the report envelope, identity, coverage, exact
  findings, signals, evidence references, diagnostics, and suppression state.
- [x] HTML/SVG projections may annotate architecture subjects from the report
  but do not mutate module, relationship, source, or finding semantics.
- [x] Exit policy is an explicit caller projection over configured severity and
  status; it does not delete or rewrite report findings.
- [x] Omitted quality profiles/reports retain existing output behavior and
  legacy consumers remain compatible.
- [x] Tests cover profile input, deterministic JSON, HTML/SVG inclusion or
  annotation, exit-policy boundaries, and report omission.

## Artifact sync required

- Application PRD: `none` — headless/export quality output is already included
  in the synchronized quality and export boundaries.
- Application architecture summary: `none` — the report remains an optional
  sibling consumed by CLI/export adapters; no new transport is introduced.
- Owning capability node/artifacts: `required: /.okf/capabilities/code-quality-and-intelligence/deterministic-quality-checks.md` and its `orchestration-status.md`.
- Issue registry: `required`; owning node `issues:` reference: `required when .okf exists`.
- OKF index/log: `required` at implementation/batch synchronization; no state
  transition is claimed by this issue alone.
- Reason/no-impact decision: delivery progress changes; no MCP, source
  mutation, topology, or product-scope change is introduced.

## Human review gate

None. Headless, serialization, export, and exit-policy tests provide the
required verification.

## Blocked by

- Completed dependency: `docs/agents/issues/done/20260830-061-quality-report-query-and-evidence.md`.

## Specification anchors

- CLI/export behavior in the canonical API/CLI contract.
- DQC-FR-006 through DQC-FR-012.
- Application Journey 4 and Journey 13.

## User stories addressed

The capability PRD has no numbered user-story section. This slice supports the
CI operator and documentation/export actors in application Journeys 4, 6, and
13 by emitting one complete machine-readable quality report.

## Verification obligations

- Policy source: `/.okf/project.md`
- Backend boundary: `when-supported`.
- Frontend integration: `not-applicable`.
- End-to-end: `when-supported` for headless analysis and deterministic exports.

## Completion evidence

- `go test ./cmd/arch-view ./internal/export -count=1`
- CLI tests cover profile input, report JSON round-trip, HTML/SVG projections,
  omitted-report compatibility, and explicit severity/status exit policies.
- Aggregate runs retain per-scope quality reports while also attaching the
  combined report for viewer scope selection.
- The full batch verification passed: `go test ./... -count=1`,
  `go test -race ./...`, `go vet ./...`, `go build ./...`, JavaScript syntax
  and viewer tests, and `git diff --check`.

## Handoff

Issue 063 completes the human-facing viewer path, including the report-backed
file line-threshold summary and affected-file filter. Live analysis and MCP
remain a separate specified capability.
