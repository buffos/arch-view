# 068 — Expose analyzer-neutral structural queries

## Issue Metadata

- Issue number: `068`
- Owning capability node: `/.okf/capabilities/code-quality-and-intelligence/live-analysis-and-mcp.md`
- Related capability nodes: `/.okf/capabilities/analyze-source.md`, `/.okf/capabilities/analyze-source/plugin-runtime/multi-analyzer-orchestration.md`, `/.okf/capabilities/code-quality-and-intelligence/source-facts-and-symbol-index.md`, `/.okf/capabilities/explore-architecture.md`
- Artifact root: `docs/architecture/code-quality-and-intelligence/live-analysis-and-mcp/`
- Issue file: `docs/agents/issues/done/20260831-068-analyzer-neutral-query-surface.md`
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

## What to build

Create the shared analyzer-neutral query service and local read adapter over
immutable live revisions. Expose scopes, files, symbols, documentation,
module facts, and callers/callees when the relevant capability exists. Every
response uses the versioned query envelope with snapshot/revision and
requested/returned consistency, deterministic ordering, explicit capability
coverage, omitted fields, byte/item budgets, and opaque cursors. Queries must
project facts from all registered analyzers through one contract rather than
adding language-specific MCP branches.

## Acceptance criteria

- [x] `list_scopes`, `find_files`, `find_symbols`, `get_documentation`,
  `get_module_facts`, and capability-aware `get_callers_callees` are available
  through a shared service and local read adapter.
- [x] The same query over the same revision, filters, projection, ordering,
  and budget returns deterministic records, counts, truncation, and cursors.
- [x] Cursors are bound to session, snapshot/revision, query, projection,
  ordering, and budget; changed contexts are rejected.
- [x] Results identify contributing analyzer/capability coverage across mixed
  Go, Python, TypeScript, Rust, and Clojure scopes where fixtures provide them.
- [x] Unsupported, unknown, partial, and not-evaluable capabilities remain
  explicit instead of being represented as empty successful results.
- [x] `require_current` delegates to the freshness service before querying;
  specific revisions remain immutable and `latest_ready` labels staleness.
- [x] Tests cover deterministic pagination, scope isolation, mixed analyzers,
  legacy models without source indexes, unsupported caller/callee queries,
  cursor rejection, and budget metadata.

## Artifact sync required

- Application PRD: `none` — analyzer-neutral agent navigation is already
  described in Journey 14.
- Application architecture summary: `none` — this implements the specified
  shared query boundary without changing ownership.
- Owning capability node/artifacts: `required: /.okf/capabilities/code-quality-and-intelligence/live-analysis-and-mcp.md` and its `orchestration-status.md`.
- Issue registry: `required`; owning node `issues:` reference: `required`.
- OKF index/log: `required` at implementation/batch synchronization; no state
  transition is claimed by this issue alone.
- Reason/no-impact decision: capability and delivery progress only; source
  index remains the fact owner and topology is unchanged.

## Blocked by

- `docs/agents/issues/pending/20260831-067-freshness-reconciliation-and-single-flight.md`

## Specification anchors

- LAM-FR-008, LAM-FR-009, LAM-FR-012, and LAM-FR-015.
- LAM-AC-009, LAM-AC-010, LAM-AC-012, and LAM-AC-022.
- `QueryEnvelope`, `StructuralQuery`, `QueryLatestReady`,
  `QuerySpecificRevision`, and `MCPAdapter`.

## User stories addressed

No numbered capability stories exist. This slice supports the coding
assistant's low-token cross-language navigation and the viewer/CLI consumers
in application Journey 14.

## Verification obligations

- Policy source: `/.okf/project.md`
- Backend boundary: `when-supported`.
- Frontend integration: `when-supported` for the local read adapter.
- End-to-end: `when-supported` across at least two registered analyzers.

## Automated verification

- `go test ./internal/analysis/... ./internal/viewer/... -count=1`
- `go test ./... -count=1`
- `go test -race ./...`
- `go vet ./...`
- `go build ./...`
- `git diff --check`

## Handoff

Issue 069 adds exact text and safe source context. Issues 070 and 071 add the
quality read and policy paths over the same revision/query boundary.

## Implementation completed

The analyzer-neutral query surface and local adapter now expose scopes, files,
symbols, documentation, module facts, and capability-aware callers/callees.
Responses use revision-bound envelopes, deterministic ordering, explicit
coverage, projections, bounded pages, and opaque cursors. Membership filters
follow only declared containment/declaration relations, and stored queries
retain source language/path context without analyzer-specific branches.

## Verification result

- `go test ./... -count=1`
- `go test -race ./...`
- `go vet ./...`
- `go build ./...`
- `git diff --check`

All checks pass. This issue has no human-review gate.

## Artifact synchronization

- Application PRD and application architecture summary: no impact; Journey 14
  already owns the shared analyzer-neutral query boundary.
- Owning capability and orchestration status: synchronized with this batch;
  the query boundary is complete while later consumers remain active.
- Issue registry and OKF log: synchronized during batch closeout.
