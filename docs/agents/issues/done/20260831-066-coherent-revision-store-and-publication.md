# 066 — Publish coherent immutable revisions

## Issue Metadata

- Issue number: `066`
- Owning capability node: `/.okf/capabilities/code-quality-and-intelligence/live-analysis-and-mcp.md`
- Related capability nodes: `/.okf/capabilities/analyze-source/plugin-runtime/multi-analyzer-orchestration.md`, `/.okf/capabilities/code-quality-and-intelligence/source-facts-and-symbol-index.md`, `/.okf/capabilities/code-quality-and-intelligence/deterministic-quality-checks.md`
- Artifact root: `docs/architecture/code-quality-and-intelligence/live-analysis-and-mcp/`
- Issue file: `docs/agents/issues/done/20260831-066-coherent-revision-store-and-publication.md`
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
- `docs/architecture/code-quality-and-intelligence/live-analysis-and-mcp/canonical-use-cases.md`
- `docs/architecture/code-quality-and-intelligence/live-analysis-and-mcp/acceptance-scenarios.md`

## What to build

Create the in-memory/persistable snapshot-store seam and candidate publication
path. A rebuild must assemble source-index, canonical model, quality report,
scope set, input fingerprint, diagnostics, and freshness metadata as one
candidate. Validate the candidate, assign a monotonic revision, and atomically
replace the reader pointer. Readers keep seeing the previous ready revision
while a candidate is built or fails; an equal semantic digest may acknowledge
an event without creating semantic revision noise.

## Acceptance criteria

- [x] Candidate construction uses the existing multi-analyzer orchestration,
  source-index, model, and deterministic-quality services without duplicating
  their semantics.
- [x] A published revision has one matching source input fingerprint and
  coherent source/model/quality/scope references; no reader can observe a
  mixed candidate.
- [x] Publication assigns monotonic session-local revisions and swaps the
  active pointer atomically for concurrent readers.
- [x] Analyzer, source-index, validation, or quality failure preserves the last
  ready revision and exposes stale/degraded diagnostics instead of an empty
  replacement.
- [x] Equal semantic input/digest can acknowledge the event without changing
  the semantic revision, while operational event status remains observable.
- [x] Tests cover concurrent reads, publication races, failed candidates,
  partial multi-analyzer results, digest equality, and revision ordering.

## Artifact sync required

- Application PRD: `none` — immutable revision behavior is already part of the
  synchronized live product contract.
- Application architecture summary: `none` — the snapshot store and atomic
  publication boundary are already specified; update before closure only if
  the implementation changes that boundary.
- Owning capability node/artifacts: `required: /.okf/capabilities/code-quality-and-intelligence/live-analysis-and-mcp.md` and its `orchestration-status.md`.
- Issue registry: `required`; owning node `issues:` reference: `required`.
- OKF index/log: `required` at implementation/batch synchronization; no state
  transition is claimed by this issue alone.
- Reason/no-impact decision: capability and delivery progress only; no new
  product capability or topology is introduced.

## Blocked by

- `docs/agents/issues/pending/20260831-064-live-session-config-and-initial-snapshot.md`
- `docs/agents/issues/pending/20260831-065-watcher-events-and-coalescing.md`

## Specification anchors

- LAM-FR-005, LAM-FR-006, LAM-FR-008, and LAM-FR-015.
- LAM-AC-005, LAM-AC-006, and LAM-AC-007.
- `LiveSnapshot`, `BuildCandidateSnapshot`, and
  `PublishSnapshotAtomically` in the canonical models.

## User stories addressed

No numbered capability stories exist. This slice supports developers and
coding assistants that must read one coherent revision during background
analysis, as described by application Journey 14.

## Verification obligations

- Policy source: `/.okf/project.md`
- Backend boundary: `when-supported`.
- Frontend integration: `not-applicable`.
- End-to-end: `when-supported` for initial and changed multi-analyzer scans.

## Automated verification

- `go test ./internal/analysis/... ./internal/analysis/orchestration/... ./internal/quality/... -count=1`
- `go test ./... -count=1`
- `go test -race ./...`
- `go vet ./...`
- `go build ./...`
- `git diff --check`

## Handoff

Issue 067 adds request-time reconciliation and the single-flight/stable-input
rules that decide when a new candidate is required.

## Implementation completed

The session-local memory store now publishes validated revisions atomically,
assigns monotonic revisions, preserves immutable historical reads, and keeps
the last ready revision when a rebuild fails. Snapshot records carry verified
input identity plus coherent source-index, model, quality, and scope
references. Stored orchestration runs retain their non-serialized scope
selection caches through an independent clone boundary.

## Verification result

- `go test ./... -count=1`
- `go test -race ./...`
- `go vet ./...`
- `go build ./...`
- `git diff --check`

All checks pass. This issue has no human-review gate.

## Artifact synchronization

- Application PRD and application architecture summary: no impact; revision
  publication remains the specified shared coordinator boundary.
- Owning capability and orchestration status: synchronized with this batch;
  the capability remains `specified` because later delivery issues remain.
- Issue registry and OKF log: synchronized during batch closeout.
