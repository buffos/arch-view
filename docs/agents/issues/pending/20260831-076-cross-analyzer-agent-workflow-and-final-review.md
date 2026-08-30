# 076 — Verify the cross-analyzer agent workflow and final product behavior

## Issue Metadata

- Issue number: `076`
- Owning capability node: `/.okf/capabilities/code-quality-and-intelligence/live-analysis-and-mcp.md`
- Related capability nodes: `/.okf/capabilities/analyze-source.md`, `/.okf/capabilities/analyze-source/plugin-runtime/multi-analyzer-orchestration.md`, `/.okf/capabilities/code-quality-and-intelligence/source-facts-and-symbol-index.md`, `/.okf/capabilities/code-quality-and-intelligence/deterministic-quality-checks.md`, `/.okf/capabilities/explore-architecture.md`, `/.okf/capabilities/export-and-automate.md`
- Artifact root: `docs/architecture/code-quality-and-intelligence/live-analysis-and-mcp/`
- Issue file: `docs/agents/issues/pending/20260831-076-cross-analyzer-agent-workflow-and-final-review.md`
- Category: `feature`
- Execution type: `AFK`
- Review gate: `product-approval`
- Suggested state: `ready-for-agent`

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

- [ ] A mixed-analyzer fixture proves one language-neutral surface with
  analyzer/capability coverage and explicit unsupported/unknown states.
- [ ] The full freshness path proves missed events, edit storms, changes during
  analysis, single-flight behavior, last-ready retention, and `input_unstable`
  handling.
- [ ] Viewer, CLI, stdio MCP, and enabled HTTP transport return equivalent
  semantic facts for the same revision/query while retaining their transport-
  specific envelopes and security policies.
- [ ] The agent loop locates a finding/symbol/text match, retrieves bounded
  evidence/context, edits outside MCP, waits for a verified revision, and
  compares findings without treating partial coverage as resolution.
- [ ] Temporary quality settings do not persist; profile/baseline writes are
  denied by default and succeed only with explicit authorization and audit
  evidence.
- [ ] Installation and troubleshooting documentation is tested from a clean
  checkout using the documented commands; no documentation claims an absent
  command or unsupported analyzer behavior.
- [ ] Automated checks pass and the final product review records desktop,
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

- `docs/agents/issues/pending/20260831-073-live-viewer-integration.md`
- `docs/agents/issues/pending/20260831-074-mcp-stdio-server-and-documentation.md`
- `docs/agents/issues/pending/20260831-075-authenticated-http-mcp-transport.md`

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

## Human product-approval gate

After automated verification, inspect the documented local installation and
the viewer/MCP workflow manually. Confirm that the user can start a session,
understand current versus stale data, search across analyzers, retrieve bounded
context, evaluate quality, and see safe permission failures. Record the
reviewed commands, environment, result, screenshots or terminal evidence, and
any corrections before moving this issue to closure.

## Handoff

If all scoped behavior and artifact synchronization are complete, the live
analysis/MCP node may be assessed for `implemented` state through the normal
verification/closeout workflow. If not, record the mismatch and keep the node
`specified`.
