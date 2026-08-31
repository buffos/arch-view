# 075 — Add authenticated HTTP MCP transport

## Issue Metadata

- Issue number: `075`
- Owning capability node: `/.okf/capabilities/code-quality-and-intelligence/live-analysis-and-mcp.md`
- Related capability nodes: `/.okf/capabilities/export-and-automate.md`, `/.okf/capabilities/analyze-source.md`
- Artifact root: `docs/architecture/code-quality-and-intelligence/live-analysis-and-mcp/`
- Issue file: `docs/agents/issues/done/20260831-075-authenticated-http-mcp-transport.md`
- Category: `feature`
- Execution type: `AFK`
- Review gate: `none`
- Suggested state: `done`

## Parent PRD

`docs/architecture/code-quality-and-intelligence/live-analysis-and-mcp/prd.md`

## Parent artifacts

- `docs/prd.md`
- `docs/architecture/application-architecture-summary.md`
- `docs/architecture/code-quality-and-intelligence/live-analysis-and-mcp/canonical-api-cli-contract.md`
- `docs/architecture/code-quality-and-intelligence/live-analysis-and-mcp/acceptance-scenarios.md`

## What to build

Add the opt-in HTTP transport for the live/MCP surface. Reuse the same service
interfaces, versioned envelopes, budgets, cursors, permissions, and quality
delegation as stdio. Local HTTP must bind only where explicitly configured;
authenticated HTTP must enforce authentication, origin policy, root/session
authorization, request limits, and policy-write authorization. Transport choice
must not change semantic results or enable shell, target execution, or source
mutation.

## Acceptance criteria

- [x] The documented HTTP adapter exposes the contract's status, currentness,
  scope, search, source-context, quality, evidence, and comparison operations
  with the same request/result semantics as stdio.
- [x] Equivalent stdio and HTTP requests over the same session/revision return
  equivalent semantic data, ordering, coverage, diagnostics, and budgets.
- [x] Unauthenticated, wrong-origin, out-of-root, invalid-session, over-limit,
  and disallowed policy requests fail closed without mutating state.
- [x] Local HTTP is opt-in and has safe bind defaults; authenticated HTTP
  requires explicit credentials/configuration and does not trust a client-
  supplied repository root or permission escalation.
- [x] Cancellation, timeouts, streaming/large-response limits, and server
  shutdown do not leave a live session or in-flight policy operation orphaned.
- [x] Transport parity, authentication/origin, root safety, CORS/CSRF where
  applicable, budget, and permission tests pass.

## Artifact sync required

- Application PRD: `none` — network transport is an already specified optional
  adapter and does not change the application workflow.
- Application architecture summary: `none` — the HTTP adapter remains behind
  the documented transport boundary and shares all service semantics.
- Owning capability node/artifacts: `required: /.okf/capabilities/code-quality-and-intelligence/live-analysis-and-mcp.md` and its `orchestration-status.md`.
- Public documentation: `required: website/` transport, authentication,
  origin, deployment, and security guidance.
- Issue registry: `required`; owning node `issues:` reference: `required`.
- OKF index/log: `required` at implementation/batch synchronization; no state
  transition is claimed by this issue alone.
- Reason/no-impact decision: optional transport delivery only; stdio remains
  the default and no new product semantics are introduced.

## Blocked by

- `docs/agents/issues/pending/20260831-074-mcp-stdio-server-and-documentation.md`

## Specification anchors

- LAM-FR-012, LAM-FR-014, and LAM-FR-015.
- LAM-AC-015, LAM-AC-016, and LAM-AC-017.
- `local_http`/`authenticated_http` transport policy and the recommended HTTP
  operation mapping in the canonical contract.

## User stories addressed

No numbered capability stories exist. This slice supports teams that need a
controlled network connection to the same live intelligence while preserving
the local stdio safety default.

## Verification obligations

- Policy source: `/.okf/project.md`
- Backend boundary: `when-supported`.
- Frontend integration: `not-applicable`.
- End-to-end: `when-supported` for an authenticated local test server.

## Automated verification

- HTTP/MCP parity and authentication integration tests
- `go test ./... -count=1`
- `go test -race ./...`
- `go vet ./...`
- `go build ./...`
- `npm run docs:check` from `website/`
- `git diff --check`

## Handoff

Issue 076 runs the final multi-analyzer and agent-loop acceptance pass over
stdio, viewer, and HTTP where enabled.

## Implementation completed

The opt-in HTTP adapter now maps live status, currentness, scopes, bounded
search/source operations, quality operations, and policy operations onto the
same shared session services as stdio. Local HTTP binds to loopback by default;
authenticated mode requires a token and optional origin allowlist. Session
paths, root-safe queries, request/response limits, encoded opaque IDs, policy
permissions, cancellation, and structured errors are enforced at the transport
boundary.

## Verification result

- HTTP/MCP semantic parity, authentication/origin, root-safety, request-limit,
  encoded-ID, budget, and permission tests
- `go test ./... -count=1`
- `go test -race ./...`
- `go vet ./...`
- `go build ./...`
- `npm run docs:check` from `website/`
- `git diff --check`

All checks pass. This issue has no human-review gate.

The final review compares an actual bounded file query over HTTP and stdio,
not only status metadata. Browser access is same-origin by default;
cross-origin access requires an allowlisted origin, and authenticated CORS
preflight does not require a bearer token before the actual request.

## Artifact synchronization

- The public English website now documents local and authenticated HTTP
  startup, routes, credentials, origins, budgets, and safety behavior.
- The live capability, orchestration status, issue registry, and OKF log are
  synchronized during batch closeout.
- The application PRD and architecture summary have no product/ownership
  impact; HTTP remains an optional adapter over the stdio-equivalent services.
