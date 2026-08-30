# 059 — Add deterministic report lifecycle, finding identity, and baselines

## Issue Metadata

- Issue number: `059`
- Owning capability node: `/.okf/capabilities/code-quality-and-intelligence/deterministic-quality-checks.md`
- Related capability nodes: `/.okf/capabilities/code-quality-and-intelligence/source-facts-and-symbol-index.md`, `/.okf/capabilities/generate-models.md`
- Artifact root: `docs/architecture/code-quality-and-intelligence/deterministic-quality-checks/`
- Issue file: `docs/agents/issues/done/20260830-059-deterministic-report-lifecycle-and-baselines.md`
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

Complete the immutable quality-report lifecycle over the rule families from
issues 053–058. Canonically order and digest profiles, providers, metrics,
coverage, findings, evidence, and diagnostics; derive stable finding keys
separate from report-local opaque IDs; preserve provider failures and explicit
coverage; and implement compatible report comparison plus exact-version
baseline/suppression resolution. Provide a user-facing CLI command that turns
selected report findings into a validated baseline document without mutating
the report or silently overwriting an existing file.

## Acceptance criteria

- [x] Equal source/model snapshots, profile, provider/rule/formula versions,
  baseline, and options produce equal semantic metrics, findings, coverage,
  ordering, evaluation fingerprint, and report digest apart from operational
  metadata.
- [x] Finding keys remain stable when evidence line positions shift while
  report-local IDs may change; keys include the declared rule/profile/version
  and scope-qualified subject identity rather than line number alone.
- [x] Provider/rule failures are isolated to diagnostics and partial coverage;
  valid findings and source/model facts remain intact.
- [x] `CompareQualityReports` distinguishes added, unchanged, suppressed, and
  resolved transitions, and never treats absence from a partial/not-evaluable
  report as resolved.
- [x] Baseline entries match exact finding, rule, profile, and formula versions;
  changed versions remain visible until explicitly baselined.
- [x] Matching baselines preserve the finding, evidence, key, and reason with
  `suppressed`/`baseline` status rather than deleting detection.
- [x] Tests cover DQC-AC-008, DQC-AC-009, DQC-AC-010, DQC-AC-014, and DQC-AC-015.
- [x] `arch-view quality baseline` creates a separate validated baseline from
  one finding, repeated finding selections, or all active findings, and
  refuses to overwrite an existing destination unless explicitly requested.

## Artifact sync required

- Application PRD: `none` — report lifecycle, baseline, and suppression
  semantics are already specified and do not change product scope.
- Application architecture summary: `none` — immutable report ownership and
  quality/source-index separation are already documented.
- Owning capability node/artifacts: `required: /.okf/capabilities/code-quality-and-intelligence/deterministic-quality-checks.md` and its `orchestration-status.md`.
- Issue registry: `required`; owning node `issues:` reference: `required when .okf exists`.
- OKF index/log: `required` at implementation/batch synchronization; no state
  transition is claimed by this issue alone.
- Reason/no-impact decision: delivery progress changes; product boundary,
  topology, and cross-capability architecture remain unchanged.

## Human review gate

None. Deterministic fixtures and report-comparison tests provide verification.

## Blocked by

None — issues 054–058 are implemented and archived; this issue is complete.

## Specification anchors

- DQC-FR-005 through DQC-FR-009 and DQC-FR-011 through DQC-FR-012.
- DQC-AC-008 through DQC-AC-010 and DQC-AC-014 through DQC-AC-015.
- `NormalizeQualityReport`, `CompareQualityReports`, `ValidateBaseline`,
  `CreateBaselineEntry`, and `ResolveSuppression`.

## User stories addressed

The capability PRD has no numbered user-story section. This slice supports the
CI operator and maintainer actors in application Journey 13 by making quality
reports reproducible, comparable, and safely suppressible.

## Verification obligations

- Policy source: `/.okf/project.md`
- Backend boundary: `when-supported`.
- Frontend integration: `when-supported`; the project-backed viewer exposes
  bounded baseline creation and explicit profile attachment over the local
  quality bridge.
- End-to-end: `when-supported` for repeated evaluation and report comparison.

## Completion evidence

- `go test ./internal/quality -count=1`
- Lifecycle tests cover line-stable finding keys, exact-version baseline
  suppression, added/unchanged/suppressed/resolved transitions, incomplete
  coverage, and provider-failure isolation.
- `go test ./cmd/arch-view -run TestQualityBaselineCommandCreatesBaselineFromAnalysisReport -count=1`
  covers creating a baseline from an analysis envelope and the overwrite guard.
- The command accepts analysis/model envelopes or a bare quality report and
  writes `arch-view.quality-baseline/v1`; the generated file is then supplied
  to analysis through `--quality-baseline`.
- The project viewer also creates the same validated baseline document from all
  active or explicitly selected findings, protects existing destinations, and
  can attach the baseline to the selected profile before re-evaluation.
- Repository verification for the batch passed: `go test ./... -count=1`,
  `go test -race ./...`, `go vet ./...`, `go build ./...`, JavaScript syntax
  and viewer tests, and `git diff --check`.

## Handoff

Issue 060 added advisory SOLID signals to the completed report lifecycle. Issue
061 exposes all report types through bounded queries and evidence.
