# 063 — Add quality findings to the viewer and affected-file filter

## Issue Metadata

- Issue number: `063`
- Owning capability node: `/.okf/capabilities/code-quality-and-intelligence/deterministic-quality-checks.md`
- Related capability nodes: `/.okf/capabilities/explore-architecture.md`, `/.okf/capabilities/code-quality-and-intelligence/source-facts-and-symbol-index.md`
- Artifact root: `docs/architecture/code-quality-and-intelligence/deterministic-quality-checks/`
- Issue file: `docs/agents/issues/pending/20260830-063-quality-viewer-and-affected-file-filter.md`
- Category: `feature`
- Execution type: `AFK`
- Review gate: `visual-review`
- Suggested state: `ready-for-agent`

## Parent PRD

`docs/architecture/code-quality-and-intelligence/deterministic-quality-checks/prd.md`

## Parent artifacts

- `docs/prd.md`
- `docs/architecture/application-architecture-summary.md`
- `docs/architecture/code-quality-and-intelligence/deterministic-quality-checks/canonical-api-cli-contract.md`
- `docs/architecture/code-quality-and-intelligence/deterministic-quality-checks/acceptance-scenarios.md`
- `docs/architecture/explore-architecture/prd.md`
- `docs/architecture/explore-architecture/canonical-api-cli-contract.md`

## What to build

Project the bounded quality report into the local architecture viewer and
inspection experience. Show exact findings, advisory signals, severity/status,
coverage, and evidence links without recalculating quality from raw source
facts. For `source:file.max-lines`, summarize the report's active findings as
the number of affected files and provide an explicit filter that lists only
those file subjects from the same report/scope. Keep partial, unknown, and
unsupported coverage visible and use existing compact/secondary-detail design
boundaries.

## Acceptance criteria

- [ ] Viewer and inspection views consume the bounded quality query/report
  boundary and display exact findings separately from advisory SOLID signals,
  analyzer diagnostics, and source facts.
- [ ] Severity, status, suppression/baseline state, coverage, limitations, and
  evidence actions remain understandable and accessible without exposing raw
  identifiers in compact summaries.
- [ ] A complete report with three active `source:file.max-lines` findings
  shows three affected files and offers a filter listing only those three file
  subjects from the same scope/report; the UI never re-evaluates line counts.
- [ ] Partial, unknown, unsupported, missing, and not-evaluable report states
  are distinct and are not shown as zero findings or a clean pass.
- [ ] Findings can reach the existing bounded evidence/source action without
  source mutation or unbounded payloads.
- [ ] Keyboard navigation, focus states, screen-reader labels, responsive
  layout, and existing graph/inspection behavior remain functional.
- [ ] Browser tests cover DQC-AC-016 and representative exact, signal,
  suppressed, partial, unsupported, and empty report states.
- [ ] A new visual review at the project's supported target viewport records
  the rendered result and resolves material layout, contrast, density, or
  accessibility findings before closure.

## Artifact sync required

- Application PRD: `none` — Journey 13 and the report-backed human projection
  are already specified; this issue does not expand the product boundary.
- Application architecture summary: `none` — the viewer remains a consumer of
  the bounded quality report and source-index evidence boundary.
- Owning capability node/artifacts: `required: /.okf/capabilities/code-quality-and-intelligence/deterministic-quality-checks.md` and its `orchestration-status.md`.
- Issue registry: `required`; owning node `issues:` reference: `required when .okf exists`.
- OKF index/log: `required` at implementation, visual-review, and batch
  synchronization; no capability-state transition is claimed by this issue
  alone.
- Reason/no-impact decision: delivery and visual-review progress change;
  product scope, topology, and cross-capability ownership remain unchanged.

## Human review gate

Required `visual-review`. Review the viewer at the project's supported desktop
and responsive widths for finding hierarchy, coverage wording, affected-file
filter clarity, evidence affordances, keyboard focus order, screen-reader
labels, and contrast. Record the rendered result and any correction in this
issue before closure.

## Blocked by

- Blocked by `docs/agents/issues/pending/20260830-061-quality-report-query-and-evidence.md`.

## Specification anchors

- DQC-FR-004, DQC-FR-005, DQC-FR-007, DQC-FR-008, DQC-FR-010, and DQC-FR-014.
- DQC-AC-011, DQC-AC-012, and DQC-AC-016.
- Application Journey 13 and the human-facing file line-threshold projection
  requirement in the deterministic-quality PRD.

## User stories addressed

The capability PRD has no numbered user-story section. This slice supports the
developer and maintainer actors in application Journey 13 by turning report
facts into digestible viewer feedback while preserving technical evidence for
deeper inspection.

## Verification obligations

- Policy source: `/.okf/project.md`
- Backend boundary: `when-supported`.
- Frontend integration: `when-supported`.
- End-to-end: `when-supported` from analysis/profile through local viewer.

## Handoff

After this issue is verified and its visual gate is approved, the deterministic
quality child can be assessed for `implemented` state. The parent roll-up
remains governed by its other structural children, and live analysis/MCP
consumes this report contract later.

