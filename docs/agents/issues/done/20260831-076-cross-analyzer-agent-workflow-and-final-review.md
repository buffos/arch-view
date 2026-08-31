# 076 — Verify the cross-analyzer agent workflow and final product behavior

## Issue Metadata

- Issue number: `076`
- Owning capability node: `/.okf/capabilities/code-quality-and-intelligence/live-analysis-and-mcp.md`
- Related capability nodes: `/.okf/capabilities/analyze-source.md`, `/.okf/capabilities/analyze-source/plugin-runtime/multi-analyzer-orchestration.md`, `/.okf/capabilities/code-quality-and-intelligence/source-facts-and-symbol-index.md`, `/.okf/capabilities/code-quality-and-intelligence/deterministic-quality-checks.md`, `/.okf/capabilities/explore-architecture.md`, `/.okf/capabilities/export-and-automate.md`
- Artifact root: `docs/architecture/code-quality-and-intelligence/live-analysis-and-mcp/`
- Issue file: `docs/agents/issues/done/20260831-076-cross-analyzer-agent-workflow-and-final-review.md`
- Category: `feature`
- Execution type: `AFK`
- Review gate: `product-approval`
- Suggested state: `done`

## Parent PRD

`docs/architecture/code-quality-and-intelligence/live-analysis-and-mcp/prd.md`

## Parent artifacts

- `docs/prd.md`
- `docs/architecture/application-architecture-summary.md`
- `docs/architecture/code-quality-and-intelligence/live-analysis-and-mcp/canonical-api-cli-contract.md`
- `docs/architecture/code-quality-and-intelligence/live-analysis-and-mcp/acceptance-scenarios.md`
- `docs/architecture/code-quality-and-intelligence/live-analysis-and-mcp/readiness-review.md`

## What to build

Create the final conformance fixture and product-level acceptance path for the
live/MCP capability. Exercise a mixed repository containing the registered Go,
Python, TypeScript, Rust, and Clojure analyzers through session start, missed
watcher/edit storms, strict currentness, analyzer-neutral search, bounded
source context, quality findings/evaluation, authorized baseline behavior, and
viewer/CLI/MCP parity. Demonstrate the external coding-agent loop:
`find → inspect → edit with normal tools → wait for a verified revision →
compare quality reports`. Record all deferred platform limitations and the
final human approval evidence in this issue.

## Acceptance criteria

- [x] A mixed-analyzer fixture proves one language-neutral surface with
  analyzer/capability coverage and explicit unsupported/unknown states.
- [x] The full freshness path proves missed events, edit storms, changes during
  analysis, single-flight behavior, last-ready retention, and `input_unstable`
  handling.
- [x] Viewer, CLI, stdio MCP, and enabled HTTP transport return equivalent
  semantic facts for the same revision/query while retaining their transport-
  specific envelopes and security policies.
- [x] The agent loop locates a finding/symbol/text match, retrieves bounded
  evidence/context, edits outside MCP, waits for a verified revision, and
  compares findings without treating partial coverage as resolution.
- [x] Temporary quality settings do not persist; profile/baseline writes are
  denied by default and succeed only with explicit authorization and audit
  evidence.
- [x] Installation and troubleshooting documentation is tested from a clean
  checkout using the documented commands; no documentation claims an absent
  command or unsupported analyzer behavior.
- [x] Automated checks pass and the final product review records desktop,
  responsive viewer behavior, MCP usability, status wording, safety boundaries,
  and any accepted limitations before closure.

## Artifact sync required

- Application PRD: `required: docs/prd.md` if the end-to-end review reveals a
  product behavior mismatch; otherwise record explicit no-impact evidence in
  this issue.
- Application architecture summary: `required: docs/architecture/application-architecture-summary.md` if the realized service/transport boundary differs; otherwise record explicit no-impact evidence.
- Owning capability node/artifacts: `required: /.okf/capabilities/code-quality-and-intelligence/live-analysis-and-mcp.md` and its `orchestration-status.md`.
- Public documentation: `required: website/` if installation, tool, or
  troubleshooting behavior changes during conformance work.
- Issue registry: `required`; owning node `issues:` reference: `required`.
- OKF index/log: `required` at final implementation/review synchronization;
  the live node may advance only after the scoped behavior and required
  artifacts are genuinely complete.
- Reason/no-impact decision: this is the final product-level gate, so any
  discovered contract or product mismatch must be synchronized before closure.

## Blocked by

- None. Issues 073–079 are complete.

The declared product-approval gate is complete; no delivery dependency remains.

## Specification anchors

- LAM-FR-004 through LAM-FR-016.
- LAM-AC-015 through LAM-AC-026.
- Application Journeys 14 and 15.
- The readiness review's cross-analyzer, freshness, quality-policy, transport,
  safety, and remediation-boundary findings.

## User stories addressed

No numbered capability stories exist. This slice validates the complete
developer, CI/CLI, and coding-assistant journeys before the capability can be
considered implemented.

## Verification obligations

- Policy source: `/.okf/project.md`
- Backend boundary: `when-supported`.
- Frontend integration: `when-supported`.
- End-to-end: `when-supported`; deferred platform surfaces require evidence
  and a reason.

## Automated verification

- `go test ./... -count=1`
- `go test -race ./...`
- `go vet ./...`
- `go build ./...`
- `node --check` for every viewer JavaScript module
- `node --test` for every viewer JavaScript test
- `npm run docs:check` from `website/`
- `npm run docs:build` from `website/`
- `git diff --check`

## Human product-approval gate (completed)

The user approved the documented local installation and viewer/MCP workflow
after the final review checklist was presented. The recorded review covers
session startup, current versus stale data, cross-analyzer search, bounded
context, quality evaluation, safe permission failures, and the source-safety
boundary.

## Handoff

Scoped behavior and artifact synchronization are complete. The live
analysis/MCP node is now `implemented` through the normal verification and
closeout workflow.

## Automated acceptance audit

The automated acceptance criteria are covered by the following evidence:

- `internal/live/cross_analyzer_test.go` builds a mixed Go, Python, TypeScript,
  Rust, and Clojure fixture and verifies analyzer-neutral scopes, capabilities,
  explicit unsupported coverage, bounded files, and current freshness.
- `internal/live/live_test.go` covers initializing/no-ready behavior, watcher
  startup reconciliation, authoritative `require_current`, concurrent
  single-flight joins, revision-bound queries, bounded source context,
  last-ready revision behavior, and explicit capability coverage states.
- `internal/live/policy_test.go`, `internal/live/http_test.go`,
  `internal/live/mcp_test.go`, `internal/live/transport_test.go`,
  `internal/viewer/live_test.go`, and `cmd/arch-view/live_test.go` cover
  temporary quality evaluation, authorization and audit boundaries, transport
  parity, revision and budget envelopes, viewer/CLI integration, and safe
  failures.
- The documented external-agent sequence is exercised through the shared live
  query/quality gateways: locate with structural or exact-text search, request
  bounded context, edit outside MCP, require a current revision, then compare
  reports. Partial coverage remains explicit in every result envelope.
- `npm run docs:check` verifies the command/rule/layout inventory and
  `npm run docs:build` verifies the public installation, tool, security, and
  troubleshooting pages.

Automated checks do not replace the required final human inspection. The user
explicitly approved issue 076 in the Codex task on 2026-08-31 after the final
product-review checklist was presented. The approval closes the product gate;
no new product mismatch or correction was reported.

## Final human inspection record

The approved review covered the documented workflows from the repository root:

```text
go run ./cmd/arch-view live start --project . --port 0 --no-watch
go run ./cmd/arch-view live ensure-current --endpoint http://127.0.0.1:PORT/v1/live/SESSION --timeout 10s
go run ./cmd/arch-view open --project . --live --port 0
go run ./cmd/arch-view mcp --project .
```

The approval covers desktop and responsive viewer behavior, current/stale/
updating wording, analyzer-neutral search and bounded context, quality
evaluation and comparison, default policy-write denial, the authorized policy
audit result, and the safety boundary that MCP cannot edit source, run the
target, or broaden the configured root. The review environment was the Windows
repository checkout used by this Codex task. The command review also corrected
the invalid unqualified form `open --project . --port 0 --no-watch`: the
non-live viewer is `open --project . --port 0`, while a live viewer without a
watcher is `open --project . --port 0 --live --no-watch`. No separate screenshot
or terminal transcript was attached; the explicit user approval is the human
gate evidence.

## Closeout synchronization

- **Application PRD:** no product-scope or behavior mismatch; status references
  were refreshed to record the approved implementation.
- **Application architecture:** no boundary change; status references were
  refreshed to record the approved implementation.
- **Owning capability artifacts:** orchestration status and the capability node
  were refreshed with the final approval and implemented state.
- **Delivery and OKF:** this issue is archived, its registry row is removed,
  its node reference points to the dated done path, and affected roll-up state
  is recomputed.
