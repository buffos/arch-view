# 067 — Compose profiles, declarative rules, state mapping, and fog of war

## Issue Metadata

- Issue number: `067`
- Owning capability node: `/.okf/capabilities/okf-knowledge-views.md`
- Related capability nodes: `/.okf/capabilities/explore-architecture.md`
- Artifact root: `docs/architecture/okf-knowledge-views/`
- Issue file: `docs/agents/issues/done/20260903-067-okf-profile-composition-and-fog-of-war.md`
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
- `docs/architecture/okf-knowledge-views/requirements-gap-analysis.md`
- `docs/architecture/okf-knowledge-views/domain-glossary.md`
- `docs/architecture/okf-knowledge-views/canonical-domain-model.md`
- `docs/architecture/okf-knowledge-views/canonical-use-cases.md`
- `docs/architecture/okf-knowledge-views/canonical-api-cli-contract.md`
- `docs/architecture/okf-knowledge-views/acceptance-scenarios.md`
- `docs/architecture/okf-knowledge-views/readiness-review.md`
- `/.okf/project.md`

## What to build

Implement the profile catalog, validation, selection, and renderer-neutral rule
application boundary. Provide immutable neutral and fog-of-war built-ins,
support project-local profiles with ordered bases, and compose declarative
registered strategies without executing target-provided code. Apply all matching
rules deterministically, diagnose equal-priority conflicts, and fall back to a
neutral/default result when a layer is unsupported, unavailable, or invalid.

Map declared source state separately from effective presentation state, support
explicit roll-up policy, preserve containment/semantic-link configuration, and
emit finite presentation tokens, roles, facets, and legend entries for the
shared viewer. Keep profiles session-scoped in this slice; durable profile
editing and bindings are owned by issue 070.

## Acceptance criteria

- [x] Every valid bundle exposes immutable neutral and fog-of-war built-in
  profiles with stable catalog metadata.
- [x] Two profiles can project the same source into distinct costumes without
  changing source facts, bundle identity, or relationship provenance.
- [x] Project-local profile composition supports ordered bases and declarative
  rules with deterministic precedence, namespaced identifiers, and validation.
- [x] All matching compatible rules compose; incompatible equal-priority
  outcomes produce a diagnostic and neutral/default fallback.
- [x] Unsupported or unavailable profile layers are visible and safely fall
  back without invalidating unrelated applicable layers.
- [x] Declared state and effective presentation state remain distinct; unknown
  source values are neutral, and roll-up is explicit rather than child-count
  inferred.
- [x] Projection output includes renderer-neutral presentation tokens, roles,
  facets, legend entries, and diagnostics consumed by the viewer.
- [x] The profile selector, catalog, legend, and state/fallback wording are
  readable and accessible in desktop and supported responsive views.

## Artifact sync required

- Application PRD: `none` — profile composition, rule safety, state mapping,
  and fog-of-war behavior are already bounded.
- Application architecture summary: `none` — the in-process registry is an
  extension seam behind the renderer-neutral projection boundary.
- Owning capability node/artifacts: `required: /.okf/capabilities/okf-knowledge-views.md`;
  no semantic exact-spec edit is expected.
- Issue registry: `required`; the owning node `issues:` reference is required.
- OKF index/log: `required` at implementation and visual-review
  synchronization; no state transition is claimed by this issue.
- Reason/no-impact decision: delivery and visual-review progress change;
  product scope, topology, source ownership, and shared viewer boundaries
  remain unchanged.

## Human review gate

Required `visual-review`. Review profile selection, neutral/fog contrast,
legend, state/roll-up wording, fallback diagnostics, keyboard focus, accessible
labels, and responsive density. Record the rendered result and resolve material
findings before closure.

## Blocked by

None. The dependency batch 064–071 is complete; original dependency order is
preserved in the canonical delivery plan.

## Specification anchors

- `FR-07` through `FR-13` in `prd.md`.
- `SelectProfile`, `ValidateProfile`, `GetProfileCatalog`, and
  `GetExtensionCatalog` in `canonical-use-cases.md`.
- `RuleComposition`, `ProfileInheritance`, `StateMapping`, `Rollup`, and the
  registry extension seams in `canonical-domain-model.md`.
- `SC-005`, `SC-008`, `SC-013`, and `SC-014`.

## User stories addressed

The capability PRD has no numbered user-story section. This slice supports the
Viewer user and Profile maintainer in `WF-02` and `WF-04` by making multiple
read-only views of the same OKF source deterministic, explainable, and safe.

## Verification obligations

- Policy source: `/.okf/project.md`

| Scenario | Backend boundary | Frontend integration | End-to-end journey |
| --- | --- | --- | --- |
| `SC-005` | `when-supported` | `when-supported` | `when-supported` |
| `SC-008` | `when-supported` | `when-supported` | `when-supported` |
| `SC-013` | `when-supported` | `when-supported` | `when-supported` |
| `SC-014` | `when-supported` | `when-supported` | `when-supported` |

## Automated verification

- Focused profile, registry, composition, state/roll-up, projection, HTTP, and
  browser tests.
- `go test ./... -count=1`
- `go test -race ./...`
- `go vet ./...`
- `go build ./...`
- `node --check` for every changed viewer JavaScript module.
- `node --test` for all viewer JavaScript tests.
- `git diff --check`

## Delivery synchronization (2026-09-03)

Automated implementation and verification are complete. Agent inspection covered
profile selection, legends, fallback wording, state presentation, and desktop
responsive layout; the required human visual-review gate remains pending. Do not
close or archive this issue until that review is explicitly approved.

## Scenario traceability

| Source rule | Scenario | Issue criterion | Verification evidence |
| --- | --- | --- | --- |
| Built-in and local profile projection | `SC-005` | Neutral and fog profiles produce distinct costumes over identical source facts | profile fixture and browser assertions |
| Declared/effective state and roll-up | `SC-008` | Known, unknown, and rolled-up state remain explicit and explainable | state mapping/roll-up tests and legend review |
| Deterministic composition | `SC-013` | Rule order, priority, conflict, and compatible merge behavior are stable | registry composition matrix tests |
| Unavailable/unsupported layer fallback | `SC-014` | A bad layer degrades with diagnostics while the remaining profile stays usable | profile validation and projection tests |

## Handoff

Issues 068–070 consume the profile catalog and resolved projection for bounded
navigation, concept detail, and durable profile configuration. Issue 071
verifies the cross-surface failure and freshness behavior after these paths
exist.

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
