# 066 — Separate containment hierarchy from semantic-link layers

## Issue Metadata

- Issue number: `066`
- Owning capability node: `/.okf/capabilities/okf-knowledge-views.md`
- Related capability nodes: `/.okf/capabilities/explore-architecture.md`
- Artifact root: `docs/architecture/okf-knowledge-views/`
- Issue file: `docs/agents/issues/done/20260903-066-okf-hierarchy-and-semantic-links.md`
- Category: `feature`
- Execution type: `AFK`
- Review gate: `visual-review`
- Suggested state: `done`

## Parent PRD

`docs/architecture/okf-knowledge-views/prd.md`

## Parent artifacts

- `docs/prd.md`
- `docs/architecture/application-architecture-summary.md`
- `docs/architecture/okf-knowledge-views/discovery-notes.md`
- `docs/architecture/okf-knowledge-views/domain-glossary.md`
- `docs/architecture/okf-knowledge-views/canonical-domain-model.md`
- `docs/architecture/okf-knowledge-views/canonical-use-cases.md`
- `docs/architecture/okf-knowledge-views/canonical-api-cli-contract.md`
- `docs/architecture/okf-knowledge-views/acceptance-scenarios.md`
- `docs/architecture/okf-knowledge-views/readiness-review.md`
- `/.okf/project.md`

## What to build

Normalize OKF relationships into independently understandable containment and
semantic-link layers. Prefer explicit parent/children facts, use the specified
filesystem fallback only when explicit hierarchy is absent, preserve
provenance, and emit deterministic diagnostics for self-parenting, cycles,
multiple parents, and conflicting hierarchy facts. Resolve safe local Markdown
links as semantic links while keeping them separate from containment and
preventing cross-bundle traversal.

Carry the distinction through the projection contract and shared viewer so
visibility, styling, summaries, and navigation can treat hierarchy and
semantic links independently. Keep this slice profile-neutral; profile-specific
rule composition is owned by issue 067.

## Acceptance criteria

- [x] Explicit parent/children facts take precedence over filesystem-derived
  containment according to the canonical hierarchy policy.
- [x] Missing explicit parents can use the defined filesystem fallback with
  provenance showing which source supplied the relationship.
- [x] Self-parenting, cycles, multiple parents, and conflicting facts produce
  deterministic diagnostics and do not create ambiguous navigable containment.
- [x] Local Markdown links inside the selected bundle become semantic links;
  unsafe, missing, or out-of-bound links remain diagnosed and cannot escape the
  bundle boundary.
- [x] Containment and semantic links have distinct relationship kinds and can
  be independently represented in the projection and viewer.
- [x] Relationship summaries and visual treatment make the distinction
  understandable without exposing unstable internal identifiers.
- [x] Tests cover explicit hierarchy, fallback, conflict, cycle, semantic-link,
  and independent bundle-boundary cases.
- [x] Visual review confirms readable edge distinction, diagnostics, focus, and
  responsive behavior without regressing architecture views.

## Artifact sync required

- Application PRD: `none` — hierarchy precedence, semantic-link separation, and
  source-boundary behavior are already specified.
- Application architecture summary: `none` — the projection remains
  renderer-neutral and the existing viewer remains a consumer.
- Owning capability node/artifacts: `required: /.okf/capabilities/okf-knowledge-views.md`;
  exact-spec documents remain authoritative.
- Issue registry: `required`; the owning node `issues:` reference is required.
- OKF index/log: `required` at implementation and review synchronization; no
  topology or capability-state change is claimed.
- Reason/no-impact decision: delivery and visual-review progress change;
  product, architecture, and source ownership remain unchanged.

## Human review gate

Required `visual-review`. Review hierarchy versus semantic-link styling,
visibility, summaries, diagnostics, keyboard focus, accessible relationship
labels, and desktop/responsive density. Record the rendered result and resolve
material findings before closure.

## Blocked by

None. The dependency batch 064–071 is complete; original dependency order is
preserved in the canonical delivery plan.

## Specification anchors

- `FR-06`, `FR-10`, and `FR-11` in `prd.md`.
- `HierarchyPrecedence`, `RelationshipVisibility`, and containment invariants
  in `canonical-domain-model.md`.
- `SC-006` and `SC-007`.

## User stories addressed

The capability PRD has no numbered user-story section. This slice supports the
Viewer user and Bundle maintainer in `WF-01`, `WF-02`, and `WF-04` by making
structural ownership and cross-reference meaning visible and explainable.

## Verification obligations

- Policy source: `/.okf/project.md`

| Scenario | Backend boundary | Frontend integration | End-to-end journey |
| --- | --- | --- | --- |
| `SC-006` | `when-supported` | `when-supported` | `when-supported` |
| `SC-007` | `when-supported` | `when-supported` | `when-supported` |

## Automated verification

- Focused hierarchy normalization, link resolution, projection, HTTP, and
  browser tests.
- `go test ./... -count=1`
- `go test -race ./...`
- `go vet ./...`
- `go build ./...`
- `node --check` for every changed viewer JavaScript module.
- `node --test` for all viewer JavaScript tests.
- `git diff --check`

## Delivery synchronization (2026-09-03)

Automated implementation and verification are complete. Agent inspection
covered containment versus semantic-link styling, diagnostics, focus/back
behavior, accessible relationship labels, and responsive density; the required
human visual-review gate remains pending. Do not close or archive this issue
until that review is explicitly approved.

## Scenario traceability

| Source rule | Scenario | Issue criterion | Verification evidence |
| --- | --- | --- | --- |
| Explicit hierarchy, fallback, and conflict policy | `SC-006` | Navigable containment is deterministic and diagnostics preserve provenance | hierarchy fixtures and projection assertions |
| Semantic-link separation | `SC-007` | Cross-references are a separate relationship kind with independent visibility | link fixture, contract test, and visual review |

## Handoff

Issue 067 consumes the normalized relationship layers when applying composable
profiles, state mappings, and presentation rules. Issues 068 and 069 consume
the same relationship semantics for focus/navigation and detail links.

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
