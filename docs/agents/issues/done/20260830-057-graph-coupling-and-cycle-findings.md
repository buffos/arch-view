# 057 — Add graph coupling and cycle findings

## Issue Metadata

- Issue number: `057`
- Owning capability node: `/.okf/capabilities/code-quality-and-intelligence/deterministic-quality-checks.md`
- Related capability node: `/.okf/capabilities/generate-models.md`
- Artifact root: `docs/architecture/code-quality-and-intelligence/deterministic-quality-checks/`
- Issue file: `docs/agents/issues/done/20260830-057-graph-coupling-and-cycle-findings.md`
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
- `docs/architecture/code-quality-and-intelligence/deterministic-quality-checks/canonical-use-cases.md`
- `docs/architecture/code-quality-and-intelligence/deterministic-quality-checks/canonical-api-cli-contract.md`
- `docs/architecture/code-quality-and-intelligence/deterministic-quality-checks/acceptance-scenarios.md`

## What to build

Add exact architecture rules for distinct efferent coupling, distinct afferent
coupling, and canonical cycle participation. Consume only reported
relationships and derived cycle projections from an authoritative model/scope.
Preserve relationship/module evidence and refuse to infer architecture facts
from repeated paths, names, or empty collections.

## Acceptance criteria

- [x] Efferent coupling counts distinct configured target modules and afferent
  coupling counts distinct configured source modules using the selected scope
  and explicit external/reference policy.
- [x] `architecture:no-cycles` evaluates canonical reported cycle projections
  and emits each configured cycle finding once with participating module and
  relationship evidence.
- [x] Repeated names or paths without corresponding graph relationships produce
  no finding.
- [x] Cross-scope evaluation preserves authoritative scope provenance and does
  not invent relationships between independent source snapshots.
- [x] Missing, partial, or unavailable graph projections produce explicit
  coverage/not-evaluable results rather than a clean pass.
- [x] Tests cover canonical cycle evidence, coupling boundaries, duplicate
  relationship targets, scope isolation, and DQC-AC-006.

## Artifact sync required

- Application PRD: `none` — coupling and cycle rules are already in the
  synchronized initial catalog.
- Application architecture summary: `none` — canonical graph ownership and
  quality decoration are already documented.
- Owning capability node/artifacts: `required: /.okf/capabilities/code-quality-and-intelligence/deterministic-quality-checks.md` and its `orchestration-status.md`.
- Issue registry: `required`; owning node `issues:` reference: `required when .okf exists`.
- OKF index/log: `required` at implementation/batch synchronization; no state
  transition is claimed by this issue alone.
- Reason/no-impact decision: delivery progress changes; topology, product
  scope, and graph semantics remain unchanged.

## Human review gate

None. Canonical-model fixtures and report tests are sufficient.

## Blocked by

- Completed dependency: `docs/agents/issues/done/20260830-053-quality-profile-and-file-size-report.md`.

## Specification anchors

- DQC-FR-004, DQC-FR-005, DQC-FR-007, DQC-FR-008, and DQC-FR-011.
- DQC-AC-006.
- `architecture:module.efferent_coupling`,
  `architecture:module.afferent_coupling`, and
  `architecture:module.cycle_participation` in the canonical model.

## User stories addressed

The capability PRD has no numbered user-story section. This slice supports the
architect/maintainer actor in application Journey 13 by surfacing measurable
graph-structural risk from canonical architecture facts.

## Verification obligations

- Policy source: `/.okf/project.md`
- Backend boundary: `when-supported`.
- Frontend integration: `not-applicable`.
- End-to-end: `when-supported` from canonical model normalization to report.

## Completion evidence

- `go test ./internal/quality -count=1`
- Distinct coupling, external-reference policy, canonical-cycle evidence,
  duplicate-cycle suppression, and scope-isolation tests pass.

## Handoff

Graph coupling and cycle findings are complete. Issue 058's explicit
forbidden-dependency and layer-direction policies are also delivered in this
implementation batch.
