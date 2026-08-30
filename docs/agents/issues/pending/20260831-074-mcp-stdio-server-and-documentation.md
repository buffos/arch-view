# 074 — Ship the MCP stdio server and installation documentation

## Issue Metadata

- Issue number: `074`
- Owning capability node: `/.okf/capabilities/code-quality-and-intelligence/live-analysis-and-mcp.md`
- Related capability nodes: `/.okf/capabilities/analyze-source.md`, `/.okf/capabilities/code-quality-and-intelligence/source-facts-and-symbol-index.md`, `/.okf/capabilities/code-quality-and-intelligence/deterministic-quality-checks.md`, `/.okf/capabilities/export-and-automate.md`
- Artifact root: `docs/architecture/code-quality-and-intelligence/live-analysis-and-mcp/`
- Issue file: `docs/agents/issues/pending/20260831-074-mcp-stdio-server-and-documentation.md`
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
- `docs/architecture/code-quality-and-intelligence/live-analysis-and-mcp/canonical-domain-model.md`
- `docs/architecture/code-quality-and-intelligence/live-analysis-and-mcp/acceptance-scenarios.md`

## What to build

Expose the shared live/query/quality services through a local MCP-compatible
stdio server. Add the versioned tools and read-only resources from the
contract, including freshness, analyzer-neutral navigation, exact text,
source context, quality evaluation, and explicitly authorized policy actions.
Default sessions must be read-only and bounded. Add a real launch command and
plain-English public documentation that explains prerequisites, installation,
MCP-client configuration, all tools, freshness modes, permissions, quality
profiles/baselines, examples, and common failures. The documentation must
describe only behavior that the executable actually provides.

## Acceptance criteria

- [ ] A documented `arch-view mcp` (or equivalent stable command) launches the
  server over stdio and returns valid MCP initialization, tool, resource, and
  error responses without logging protocol noise to stdout.
- [ ] Tools cover snapshot status/currentness, scopes, files, symbols,
  documentation, module facts, callers/callees, exact text, source context,
  quality profiles/rules/findings/evidence/evaluation/comparison, and the
  explicitly permissioned profile/baseline operations.
- [ ] Every response preserves the versioned query/quality envelope, revision
  and freshness context, capability coverage, deterministic ordering, cursor,
  omission, and byte/item budget semantics.
- [ ] Default MCP configuration rejects shell execution, target execution,
  source edits, arbitrary paths, and quality-policy writes; separately granted
  policy permissions are validated and audited.
- [ ] The server handles cancellation, malformed requests, unknown tools,
  unavailable capabilities, no-ready snapshots, unstable input, and analyzer
  failures with structured errors while preserving the last-ready revision.
- [ ] The English documentation site contains installation and connection
  instructions with copyable examples, explains all configuration fields and
  tools in simple language, and includes the stdio safety/permission model.
- [ ] MCP protocol, tool, budget, permission, documentation smoke, and
  existing CLI compatibility tests pass.

## Artifact sync required

- Application PRD: `none` — MCP is already a specified application workflow;
  this issue delivers its transport and user documentation.
- Application architecture summary: `none` — stdio is the specified default
  adapter over shared services; no new ownership boundary is introduced.
- Owning capability node/artifacts: `required: /.okf/capabilities/code-quality-and-intelligence/live-analysis-and-mcp.md` and its `orchestration-status.md`.
- Public documentation: `required: website/` MCP overview, installation,
  configuration, tools, quality, and troubleshooting pages plus inventory/link
  coverage where applicable.
- Issue registry: `required`; owning node `issues:` reference: `required`.
- OKF index/log: `required` at implementation/batch synchronization; no state
  transition is claimed by this issue alone.
- Reason/no-impact decision: delivery and public documentation progress only;
  the transport and operation semantics are already in the exact contract.

## Blocked by

- `docs/agents/issues/pending/20260831-068-analyzer-neutral-query-surface.md`
- `docs/agents/issues/pending/20260831-069-exact-text-and-source-context.md`
- `docs/agents/issues/pending/20260831-070-quality-gateway-and-temporary-evaluation.md`
- `docs/agents/issues/pending/20260831-071-permissioned-quality-policy-operations.md`
- `docs/agents/issues/pending/20260831-072-local-live-session-cli-bridge.md`

## Specification anchors

- LAM-FR-008 through LAM-FR-016.
- LAM-AC-009 through LAM-AC-018 and LAM-AC-022 through LAM-AC-025.
- The MCP operation mapping, stdio packaging, budget, permission, and
  remediation-boundary sections of the canonical contract.
- Application documentation requirements in the human-centered documentation
  site plan.

## User stories addressed

No numbered capability stories exist. This slice supports coding assistants
that need compact, cross-language repository intelligence and developers who
need simple installation and connection instructions.

## Verification obligations

- Policy source: `/.okf/project.md`
- Backend boundary: `when-supported`.
- Frontend integration: `not-applicable`.
- End-to-end: `when-supported` through an actual MCP client or protocol fixture.

## Automated verification

- MCP protocol/tool/resource integration tests
- `go test ./... -count=1`
- `go test -race ./...`
- `go vet ./...`
- `go build ./...`
- `node --check` for all viewer/documentation JavaScript where applicable
- `npm run docs:check` from `website/`
- `npm run docs:build` from `website/`
- `git diff --check`

## Handoff

Issue 075 adds the optional authenticated HTTP adapter over exactly the same
operations. Issue 076 exercises the complete documented workflow across all
registered analyzers.
