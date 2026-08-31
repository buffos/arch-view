# 077 — Implement the managed quality-baseline store and merge lifecycle

## Issue Metadata

- Issue number: `077`
- Owning capability node: `/.okf/capabilities/code-quality-and-intelligence/deterministic-quality-checks.md`
- Related capability nodes: `/.okf/capabilities/code-quality-and-intelligence/live-analysis-and-mcp.md`
- Artifact root: `docs/architecture/code-quality-and-intelligence/deterministic-quality-checks/`
- Issue file: `docs/agents/issues/done/20260831-077-managed-quality-baseline-store-and-merge.md`
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
- `docs/architecture/code-quality-and-intelligence/deterministic-quality-checks/canonical-api-cli-contract.md`
- `docs/architecture/code-quality-and-intelligence/deterministic-quality-checks/acceptance-scenarios.md`

## What to build

Implement the project-local managed baseline boundary. A profile reference must
resolve one exact baseline ID/revision from `quality-baselines/`; the resolver
must never guess from unrelated JSON files. Add deterministic listing and
bounded document reads, idempotent entry merge, conflicting-metadata rejection,
numeric revision advancement, safe direct filenames, expected-revision conflict
detection, and atomic profile/baseline publication with rollback when the pair
cannot be completed. Keep the `arch-view.quality-baseline/v1` schema unchanged.

The lifecycle must accept only active findings with observed coverage. The
policy layer must not suppress unsupported, not-evaluable, partial, stale, or
non-active results.

## Acceptance criteria

- [x] A new managed baseline starts at revision `1.0.0`; later distinct appends
  advance the numeric revision, while legacy revisions receive a deterministic
  numeric suffix and an explicit revision remains supported.
- [x] A duplicate with the same exact finding/rule/profile/formula identity and
  review metadata is idempotent; a duplicate with a different reason or owner
  is rejected as a conflict.
- [x] Baseline resolution matches exact ID and revision only, reports missing
  documents distinctly, and rejects invalid or ambiguous matches.
- [x] Managed baseline entries require active findings with observed coverage;
  unsupported, not-evaluable, partial, stale, suppressed, and resolved inputs
  are not accepted.
- [x] Direct project-local JSON destinations are enforced and a failed paired
  publication does not leave a newly written baseline without its profile
  reference.
- [x] Focused tests cover merge, idempotence, revision changes, expected
  revision conflicts, listing, exact resolution, ambiguity, invalid documents,
  and missing documents.

## Artifact sync required

- Application PRD: `required: docs/prd.md`.
- Application architecture summary: `required: docs/architecture/application-architecture-summary.md`.
- Owning capability node/artifacts: `required: /.okf/capabilities/code-quality-and-intelligence/deterministic-quality-checks.md` and its canonical artifact set.
- Issue registry: `required`; owning node `issues:` reference: `required`.
- OKF index/log: `required` at implementation synchronization.
- Reason/no-impact decision: this extends the previously standalone baseline
  document workflow into a canonical profile-owned lifecycle without changing
  the baseline schema or source-edit boundary.

## Blocked by

None. It builds on the quality profile/report lifecycle from issues 053–059.

## Specification anchors

- DQC-FR-009 and DQC-FR-015 through DQC-FR-018.
- DQC-AC-009, DQC-AC-010, and DQC-AC-017 through DQC-AC-020.
- `Baseline`, `BaselineEntry`, `BaselineRef`, `MergeBaselineEntries`,
  `NextBaselineRevision`, and the project policy store boundary.

## Verification obligations

- Policy source: `/.okf/project.md`
- Backend boundary: `when-supported`.
- Frontend integration: `not-applicable`.
- End-to-end: `when-supported` for a temporary project containing profile and
  baseline directories.

## Automated verification

- `go test ./internal/quality ./internal/quality/policy -count=1`
- `go test ./... -count=1`
- `go test -race ./...`
- `go vet ./...`
- `go build ./...`
- `git diff --check`

## Completion evidence

- `internal/quality/lifecycle_test.go` covers idempotent/conflicting merge and
  revision behavior.
- `internal/quality/policy/store_test.go` covers deterministic listing, exact
  resolution, missing/invalid/ambiguous outcomes.
- `internal/quality/policy/append_test.go` covers creation, merge,
  idempotence, revision advancement, and expected-revision conflict.
- `internal/quality/policy/append.go` validates the current profile again,
  re-reads both documents before publication, and rolls back the baseline if
  profile publication fails; `append_test.go` verifies that rollback boundary.

## Handoff

Issue 078 consumes this lifecycle for the managed CLI. Issue 079 consumes it
for live startup evaluation and MCP read/selection/append operations.

## Verification result

Focused policy and quality tests pass. The final repository and documentation
verification for this implementation batch is recorded in the delivery report
returned with issues 077–079.
