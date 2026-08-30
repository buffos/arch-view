# 063 — Add quality findings to the viewer and affected-file filter

## Issue Metadata

- Issue number: `063`
- Owning capability node: `/.okf/capabilities/code-quality-and-intelligence/deterministic-quality-checks.md`
- Related capability nodes: `/.okf/capabilities/explore-architecture.md`, `/.okf/capabilities/code-quality-and-intelligence/source-facts-and-symbol-index.md`
- Artifact root: `docs/architecture/code-quality-and-intelligence/deterministic-quality-checks/`
- Issue file: `docs/agents/issues/done/20260830-063-quality-viewer-and-affected-file-filter.md`
- Category: `feature`
- Execution type: `AFK`
- Review gate: `visual-review`
- Suggested state: `done`

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
facts. The graph controls group view navigation, quality profile selection,
and presentation actions into separate semantic regions. The quality profile
editor lists the complete registered catalog with an enabled/off state and
applies toggles only for the current viewer session. For `source:file.max-lines`, summarize the report's active findings as
the number of affected files and provide an explicit filter that lists only
those file subjects from the same report/scope. Keep partial, unknown, and
unsupported coverage visible and use existing compact/secondary-detail design
boundaries.

## Acceptance criteria

- [x] Viewer and inspection views consume the bounded quality query/report
  boundary and display exact findings separately from advisory SOLID signals,
  analyzer diagnostics, and source facts.
- [x] Severity, status, suppression/baseline state, coverage, limitations, and
  evidence actions remain understandable and accessible without exposing raw
  identifiers in compact summaries.
- [x] A complete report with three active `source:file.max-lines` findings
  shows three affected files and offers a filter listing only those three file
  subjects from the same scope/report; the UI never re-evaluates line counts.
- [x] Partial, unknown, unsupported, missing, and not-evaluable report states
  are distinct and are not shown as zero findings or a clean pass.
- [x] Findings can reach the existing bounded evidence/source action without
  source mutation or unbounded payloads.
- [x] Keyboard navigation, focus states, screen-reader labels, responsive
  layout, and existing graph/inspection behavior remain functional.
- [x] Browser tests cover DQC-AC-016 and representative exact, signal,
  suppressed, partial, unsupported, and empty report states.
- [x] The viewer exposes the complete quality-rule catalog with readable
  categories, enabled/off toggles, catalog defaults, a session-only apply
  path, and explicit save/save-as actions for durable profile changes.
- [x] The graph header separates View, Quality, and Actions controls while
  reusing the shared search, select, button, and state-pill primitives.
- [x] A new visual review at the project's supported target viewport records
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

- Completed dependencies:
  `docs/agents/issues/done/20260830-061-quality-report-query-and-evidence.md`
  and `docs/agents/issues/done/20260830-062-quality-cli-and-export-projections.md`.

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

## Automated verification

- `go test ./... -count=1`
- `go test -race ./...`
- `go vet ./...`
- `go build ./...`
- `node --check` for every viewer JavaScript module
- `node --test` for all viewer JavaScript tests
- `git diff --check`

The automated checks pass. The viewer now loads quality data through the
bounded report/query boundary, keeps the compact graph summary free of raw
quality identifiers, exposes exact findings separately from advisory signals,
and derives the affected-file filter only from active
`source:file.max-lines` findings. It also exposes a deterministic rule catalog
and rule selection so a human can see and change every available check, apply
the change temporarily, or explicitly save it to an existing/new profile.

## Human review completed

The final review covered the normal desktop viewport and a 390×844 responsive
viewport. The compact graph card, inspection overview, coverage wording,
evidence action, complete rule catalog, visible keyboard focus, Back
navigation, and responsive layout were readable and usable without page-level
horizontal overflow. The review found and corrected a material aggregate-scope
bug: the affected-file control initially combined module and report file
selectors, which broadened the result to unrelated files, and then exposed an
identity mismatch between analyzer snapshots and the aggregate source
projection. Aggregate quality evaluation now uses the projection identity and
the affected-only request sends only exact report file subjects. A live rerun
showed 23 of 23 repository findings and no unrelated files; the three-finding
acceptance fixture follows the same bounded path. No material contrast,
density, accessibility, or layout findings remain.

## Handoff

After this issue is verified and its visual gate is approved, the deterministic
quality child can be assessed for `implemented` state. The parent roll-up
remains governed by its other structural children, and live analysis/MCP
consumes this report contract later.
