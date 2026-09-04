# 064 — Discover, validate, index, and select independent OKF bundles

## Issue Metadata

- Issue number: `064`
- Owning capability node: `/.okf/capabilities/okf-knowledge-views.md`
- Related capability nodes: `/.okf/capabilities/explore-architecture.md`
- Artifact root: `docs/architecture/okf-knowledge-views/`
- Issue file: `docs/agents/issues/done/20260903-064-okf-bundle-discovery-and-index.md`
- Category: `feature`
- Execution type: `AFK`
- Review gate: `none`
- Suggested state: `done`

## Parent PRD

`docs/architecture/okf-knowledge-views/prd.md`

## Parent artifacts

- `docs/prd.md`
- `docs/architecture/application-architecture-summary.md`
- `docs/architecture/okf-knowledge-views/discovery-notes.md`
- `docs/architecture/okf-knowledge-views/requirements-gap-analysis.md`
- `docs/architecture/okf-knowledge-views/domain-glossary.md`
- `docs/architecture/okf-knowledge-views/canonical-domain-model.md`
- `docs/architecture/okf-knowledge-views/canonical-use-cases.md`
- `docs/architecture/okf-knowledge-views/canonical-api-cli-contract.md`
- `docs/architecture/okf-knowledge-views/acceptance-scenarios.md`
- `docs/architecture/okf-knowledge-views/readiness-review.md`
- `/.okf/project.md`

## What to build

Establish the read-only source boundary for OKF knowledge views. Recursively
discover eligible `.okf` directories within the selected project, apply the
specified exclusions, normalize stable repository-relative POSIX bundle IDs,
validate candidates before marking them selectable, and preserve a lossless
bundle index containing frontmatter, Markdown, links, unknown metadata, and
provenance. Expose deterministic catalog and selected-bundle summary data
through the canonical OKF HTTP boundary, including actionable diagnostics for
invalid, unreadable, unavailable, or independently failing candidates.

Keep catalog ordering deterministic, keep local-link resolution inside the
selected bundle, and never merge bundles or mutate OKF source files. This slice
does not add profile composition or graph rendering; it provides the validated
catalog, one-bundle selection, and source facts consumed by the next slice.

## Acceptance criteria

- [x] Recursive discovery finds eligible `.okf` directories under the project
  while skipping `.git`, dependency, generated, and output directories defined
  by the capability rules.
- [x] Every candidate has a stable normalized repository-relative POSIX bundle
  ID and a status that distinguishes discovered, valid, invalid, unavailable,
  and unreadable candidates.
- [x] Invalid candidates are diagnosed before selection and cannot be silently
  treated as valid or replaced by a different candidate.
- [x] A selected valid bundle is indexed losslessly, preserving concept paths,
  frontmatter, Markdown, links, unknown metadata, and source provenance.
- [x] Local-link resolution is bundle-scoped and cannot read a concept or file
  outside the selected bundle boundary.
- [x] Catalog and summary responses have deterministic ordering, bounded
  diagnostics, and the canonical response/status/error vocabulary.
- [x] Failure in one bundle does not prevent another valid bundle from being
  listed and selected.
- [x] Focused fixtures and contract tests cover multiple bundles, invalid
  candidates, arbitrary source facts, boundary violations, and independent
  failure isolation without writing to source bundles.

## Artifact sync required

- Application PRD: `none` — the discovery, selection, source-boundary, and
  read-only rules are already specified.
- Application architecture summary: `none` — this implements the documented
  discovery/index boundary and does not change the shared viewer architecture.
- Owning capability node/artifacts: `required: /.okf/capabilities/okf-knowledge-views.md`;
  exact-spec artifacts remain authoritative unless implementation proves a
  contract mismatch.
- Issue registry: `required`; the owning node `issues:` reference is required
  when `.okf` exists.
- OKF index/log: `required` for issue creation and delivery synchronization;
  the stale capability-directory index entry must remain repaired.
- Reason/no-impact decision: delivery truth changes; product scope,
  architecture boundaries, topology, and capability state remain unchanged.

## Human review gate

None. This slice has no rendered UI/UX change; the catalog and source boundary
are verified through Go domain, HTTP contract, fixture, and end-to-end tests.

## Blocked by

None. The dependency batch 064–071 is complete; original dependency order is
preserved in the canonical delivery plan.

## Specification anchors

- `FR-01` through `FR-06` in `prd.md`.
- `RefreshBundleCatalog`, `SelectBundle`, `GetBundleCatalog`, and
  `GetBundleIndexSummary` in `canonical-use-cases.md`.
- Bundle identity, validation, lossless indexing, and boundary rules in
  `canonical-domain-model.md` and `canonical-api-cli-contract.md`.
- `SC-001`, `SC-002`, `SC-003`, and `SC-020`.

## User stories addressed

The capability PRD has no numbered user-story section. This slice supports the
Viewer user and Bundle maintainer in `WF-01` by making independent bundles
discoverable, selectable, traceable, and safely diagnosable.

## Verification obligations

- Policy source: `/.okf/project.md`

| Scenario | Backend boundary | Frontend integration | End-to-end journey |
| --- | --- | --- | --- |
| `SC-001` | `when-supported` | `not-applicable` | `when-supported` |
| `SC-002` | `when-supported` | `not-applicable` | `when-supported` |
| `SC-003` | `when-supported` | `not-applicable` | `when-supported` |
| `SC-020` | `when-supported` | `not-applicable` | `when-supported` |

## Automated verification

- Focused discovery, validation, lossless-index, boundary, catalog, and HTTP
  contract tests.
- `go test ./... -count=1`
- `go test -race ./...`
- `go vet ./...`
- `go build ./...`
- `git diff --check`

## Delivery synchronization (2026-09-03)

Implementation is complete for this no-UI slice. Focused discovery, indexing,
boundary, application, and HTTP tests pass, as do the full repository gates.
No visual-review gate applies to issue 064.

## Scenario traceability

| Source rule | Scenario | Issue criterion | Verification evidence |
| --- | --- | --- | --- |
| Recursive discovery and stable bundle identity | `SC-001` | Eligible independent bundles appear in deterministic catalog order and can be selected | discovery fixture and catalog contract tests |
| Candidate validation and diagnostics | `SC-002` | Invalid candidates remain visible with actionable status and are not selectable | invalid-bundle fixture and error assertions |
| Lossless indexing and provenance | `SC-003` | Arbitrary source facts survive indexing without source mutation | round-trip/index assertions and read-only fixture check |
| Bundle boundary and failure isolation | `SC-020` | One failing bundle does not hide or corrupt another valid bundle | multi-bundle isolation test |

## Handoff

Issue 065 consumes the validated catalog, selected bundle identity, immutable
lossless index, and source diagnostics to build the first neutral projection.
The capability remains `specified` until the complete approved issue set is
implemented and verified.

## Approved closeout (2026-09-04)

The user explicitly approved the completed result in this task: "Everything is
fine now. I approve. proceed to commits (one or more )". This closes the human
review gate where applicable. The chronological pending-review and evidence-gap
notes above are superseded by this closeout, not erased.

All acceptance criteria are satisfied by the scenario evidence audit and final
review in `docs/agents/reviews/20260904-okf-scenario-evidence.md` and
`docs/agents/reviews/20260904-okf-working-tree-review.md`, including their later
resolved findings. The shared CSS/SVG follow-up is recorded in
`docs/agents/reviews/20260904-shared-css-svg-followup.md`.

The final Go test, race, vet, build, JavaScript test/syntax, and diff checks pass.
Artifact synchronization includes the application PRD/architecture summary,
owning capability, issue registry, orchestration status, and OKF index/log.
Approved follow-ups include architecture-first mode, shared settings/rendering,
selection-only arrowless semantic links, uncapped Fit, and current-canvas SVG
download. No public OKF CLI, headless OKF export API, or source-editing surface
is introduced.
