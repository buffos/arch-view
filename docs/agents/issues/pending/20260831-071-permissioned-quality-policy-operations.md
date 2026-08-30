# 071 — Add permissioned quality-policy operations

## Issue Metadata

- Issue number: `071`
- Owning capability node: `/.okf/capabilities/code-quality-and-intelligence/live-analysis-and-mcp.md`
- Related capability nodes: `/.okf/capabilities/code-quality-and-intelligence/deterministic-quality-checks.md`
- Artifact root: `docs/architecture/code-quality-and-intelligence/live-analysis-and-mcp/`
- Issue file: `docs/agents/issues/pending/20260831-071-permissioned-quality-policy-operations.md`
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
- `docs/architecture/code-quality-and-intelligence/deterministic-quality-checks/canonical-api-cli-contract.md`

## What to build

Expose explicit quality-policy commands through the live quality gateway:
validate a profile, save an existing profile, save a new profile, preview a
baseline, and create a baseline. Default sessions remain read-only. A policy
write requires an allowlisted operation, opaque authorization, exact profile/
rule/formula identity, a verified compatible current report, safe destination,
reason where required, and an audit record. Profile and baseline documents are
the only allowed write targets; no command changes source code or silently
re-evaluates a stale report.

## Acceptance criteria

- [ ] Default read-only sessions reject profile saves and baseline creation
  without modifying project files or live snapshots.
- [ ] Profile save/save-as delegates complete profile validation to the
  deterministic-quality policy service and restricts destinations to the
  configured profile area.
- [ ] Baseline preview is dry-run only and reports the exact selected finding
  keys and policy identities that would be recorded.
- [ ] Baseline creation requires an exact compatible current report, selected
  finding keys, exact rule/profile/formula versions, a reason, safe destination,
  explicit authorization, and non-conflicting write behavior.
- [ ] Every successful policy write returns an auditable result; failed,
  stale, incompatible, unauthorized, or invalid requests leave source,
  reports, and existing policy documents unchanged.
- [ ] Tests cover default denial, authorization, destination traversal,
  profile conflicts, stale/partial reports, baseline selection, audit output,
  overwrite rules, and source non-mutation.

## Artifact sync required

- Application PRD: `none` — explicit quality-policy permissions and baseline
  semantics are already specified in Journey 15.
- Application architecture summary: `none` — policy ownership remains in the
  deterministic-quality service; this adds only the specified adapter path.
- Owning capability node/artifacts: `required: /.okf/capabilities/code-quality-and-intelligence/live-analysis-and-mcp.md` and its `orchestration-status.md`.
- Issue registry: `required`; owning node `issues:` reference: `required`.
- OKF index/log: `required` at implementation/batch synchronization; no state
  transition is claimed by this issue alone.
- Reason/no-impact decision: capability and delivery progress only; policy
  writes are explicit and do not broaden source mutation permissions.

## Blocked by

—

## Specification anchors

- LAM-FR-011, LAM-FR-014, and LAM-FR-015.
- LAM-AC-016, LAM-AC-024, and LAM-AC-025.
- `QualityPolicyCommand`, `ExecuteQualityPolicyCommand`, and the permission/
  audit rules in the canonical contract.
- Deterministic-quality policy/profile/baseline contracts delivered by issues
  `053` and `059`.

## User stories addressed

No numbered capability stories exist. This slice supports the maintainer or
developer in application Journey 15 who explicitly manages profiles and
baselines without allowing an agent to mutate policy by default.

## Verification obligations

- Policy source: `/.okf/project.md`
- Backend boundary: `when-supported`.
- Frontend integration: `not-applicable`.
- End-to-end: `when-supported` for profile and baseline files in a temporary
  repository.

## Automated verification

- `go test ./internal/quality/... ./internal/viewer/... -count=1`
- `go test ./... -count=1`
- `go test -race ./...`
- `go vet ./...`
- `go build ./...`
- `git diff --check`

## Handoff

Issue 074 exposes these operations through MCP after the local CLI/session
bridge and read/query surfaces are available.
