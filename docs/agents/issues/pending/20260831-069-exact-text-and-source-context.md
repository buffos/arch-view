# 069 — Add exact text search and bounded source context

## Issue Metadata

- Issue number: `069`
- Owning capability node: `/.okf/capabilities/code-quality-and-intelligence/live-analysis-and-mcp.md`
- Related capability nodes: `/.okf/capabilities/code-quality-and-intelligence/source-facts-and-symbol-index.md`, `/.okf/capabilities/explore-architecture.md`, `/.okf/capabilities/export-and-automate.md`
- Artifact root: `docs/architecture/code-quality-and-intelligence/live-analysis-and-mcp/`
- Issue file: `docs/agents/issues/pending/20260831-069-exact-text-and-source-context.md`
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

Add bounded, deterministic literal/regex search and an explicit source-context
follow-up. Search returns only repository-relative paths and line/column
locations, with safe regex limits and the common query budget/cursor envelope.
Source context accepts an indexed entity/span and explicit line/byte limits,
verifies the selected revision and content hash where available, normalizes
the path beneath an allowed root, and returns a small read-only excerpt. It
must never become arbitrary file access or unbounded file retrieval.

## Acceptance criteria

- [ ] `find_text` supports bounded literal and safe-regex modes with path,
  language, scope, case, line-size, item, byte, and cursor limits.
- [ ] Text results are deterministic and contain file/line/column matches, not
  complete source files or fuzzy/embedding-ranked results.
- [ ] `get_source_context` requires an indexed entity/span, selected revision,
  and explicit bounded line/byte limits.
- [ ] Source context validates root containment, selected scope/revision, and
  content hash where available; traversal, arbitrary paths, invalid spans, and
  over-budget requests fail with structured errors.
- [ ] The existing viewer/source boundary is reused rather than duplicated,
  and quality evidence can compose with this read-only context operation.
- [ ] Tests cover literal/regex determinism, large-match truncation, invalid
  regex, traversal, root/scope mismatch, changed content hash, and limits.

## Artifact sync required

- Application PRD: `none` — bounded source navigation is already specified in
  Journey 14.
- Application architecture summary: `none` — the source-context boundary is
  already owned by source-safe inspection and is being reused.
- Owning capability node/artifacts: `required: /.okf/capabilities/code-quality-and-intelligence/live-analysis-and-mcp.md` and its `orchestration-status.md`.
- Issue registry: `required`; owning node `issues:` reference: `required`.
- OKF index/log: `required` at implementation/batch synchronization; no state
  transition is claimed by this issue alone.
- Reason/no-impact decision: capability and delivery progress only; source
  content remains opt-in and read-only.

## Blocked by

- `docs/agents/issues/pending/20260831-068-analyzer-neutral-query-surface.md`

## Specification anchors

- LAM-FR-009, LAM-FR-013, and LAM-FR-014.
- LAM-AC-013 and LAM-AC-014.
- `TextSearchQuery`, `SearchExactText`, `GetBoundedSourceContext`, and
  `Source-context safety` in the canonical contract.

## User stories addressed

No numbered capability stories exist. This slice supports the coding
assistant's exact-symbol/text navigation and evidence-backed, low-context
inspection in application Journey 14.

## Verification obligations

- Policy source: `/.okf/project.md`
- Backend boundary: `when-supported`.
- Frontend integration: `not-applicable`.
- End-to-end: `when-supported` through the local source-context adapter.

## Automated verification

- `go test ./internal/analysis/... ./internal/viewer/... -count=1`
- `go test ./... -count=1`
- `go test -race ./...`
- `go vet ./...`
- `go build ./...`
- `git diff --check`

## Handoff

Issue 070 can expose quality evidence and temporary evaluations while reusing
this bounded context operation.
