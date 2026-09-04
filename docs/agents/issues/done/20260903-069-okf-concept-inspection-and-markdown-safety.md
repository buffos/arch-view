# 069 — Add safe concept inspection and Markdown details

## Issue Metadata

- Issue number: `069`
- Owning capability node: `/.okf/capabilities/okf-knowledge-views.md`
- Related capability nodes: `/.okf/capabilities/explore-architecture.md`
- Artifact root: `docs/architecture/okf-knowledge-views/`
- Issue file: `docs/agents/issues/done/20260903-069-okf-concept-inspection-and-markdown-safety.md`
- Category: `feature`
- Execution type: `AFK`
- Review gate: `visual-review`
- Suggested state: `done`

## Parent PRD

`docs/architecture/okf-knowledge-views/prd.md`

## Parent artifacts

- `docs/prd.md`
- `docs/architecture/application-architecture-summary.md`
- `docs/architecture/okf-knowledge-views/prd.md`
- `docs/architecture/okf-knowledge-views/canonical-domain-model.md`
- `docs/architecture/okf-knowledge-views/canonical-use-cases.md`
- `docs/architecture/okf-knowledge-views/canonical-api-cli-contract.md`
- `docs/architecture/okf-knowledge-views/acceptance-scenarios.md`
- `docs/architecture/okf-knowledge-views/readiness-review.md`
- `/.okf/project.md`

## What to build

Implement the concept-detail query and inspection surface for the selected OKF
bundle. Show overview data, mapped profile metadata, sanitized CommonMark,
unknown frontmatter, containment/semantic links, provenance, and bounded
diagnostics. Support safe local concept links within the selected bundle and
safe external HTTP/HTTPS/mail links in a new context, while rejecting or
stripping HTML, scripts, unsafe schemes, path escapes, and other unsafe markup.

Keep raw Markdown secondary to the sanitized rendering, preserve arbitrary
source facts without treating them as executable content, and make detail
loading compatible with depth/focus state and the shared viewer inspection
patterns.

## Acceptance criteria

- [x] Single-click or equivalent selection opens concept detail with overview,
  mapped metadata, provenance, relationship context, and diagnostics.
- [x] Supported CommonMark headings, emphasis, lists, and code render safely;
  HTML, scripts, and unsafe schemes are stripped or rejected.
- [x] Local concept links resolve only inside the selected bundle and cannot
  escape through traversal or alternate paths.
- [x] External HTTP/HTTPS/mail links are marked as external and open in a safe
  new context; unsupported schemes are diagnosed.
- [x] Unknown frontmatter remains available in a collapsed/secondary section,
  and raw Markdown is secondary rather than the primary rendered surface.
- [x] Detail and link payloads are bounded and retain source/profile/provenance
  identity needed to explain the projection.
- [x] Unsafe/adversarial fixtures cover HTML, scripts, schemes, path escapes,
  missing links, and arbitrary metadata.
- [x] Visual review covers detail hierarchy, safe-link affordances, diagnostics,
  keyboard focus, screen-reader labels, and responsive layout.

## Artifact sync required

- Application PRD: `none` — inspection, Markdown safety, and link-boundary rules
  are already specified.
- Application architecture summary: `none` — detail remains a consumer of the
  lossless index and renderer-neutral projection; no source mutation is added.
- Owning capability node/artifacts: `required: /.okf/capabilities/okf-knowledge-views.md`;
  exact-spec documents remain authoritative.
- Issue registry: `required`; the owning node `issues:` reference is required.
- OKF index/log: `required` at implementation and visual-review
  synchronization; no capability-state transition is claimed.
- Reason/no-impact decision: delivery and visual-review progress change;
  product scope, topology, and source ownership remain unchanged.

## Human review gate

Required `visual-review`. Review the detail panel, Markdown hierarchy, unknown
metadata disclosure, local/external link treatment, diagnostics, focus order,
accessible labels, and desktop/responsive density. Record the rendered result
and resolve material findings before closure.

## Blocked by

None. The dependency batch 064–071 is complete; original dependency order is
preserved in the canonical delivery plan.

## Specification anchors

- `FR-17` and `FR-18` in `prd.md`.
- `GetConceptDetail` in `canonical-use-cases.md`.
- `MarkdownSafety`, `MarkdownFragment`, provenance, and concept-detail payload
  rules in `canonical-domain-model.md` and `canonical-api-cli-contract.md`.
- `SC-003` and `SC-012`.

## User stories addressed

The capability PRD has no numbered user-story section. This slice supports the
Viewer user in `WF-04` by making arbitrary OKF facts inspectable without
silently executing or exposing unsafe content.

## Verification obligations

- Policy source: `/.okf/project.md`

| Scenario | Backend boundary | Frontend integration | End-to-end journey |
| --- | --- | --- | --- |
| `SC-003` | `when-supported` | `when-supported` | `when-supported` |
| `SC-012` | `when-supported` | `when-supported` | `when-supported` |

## Automated verification

- Focused concept-detail, Markdown sanitization, link-boundary, provenance,
  HTTP, and browser tests with adversarial fixtures.
- `go test ./... -count=1`
- `go test -race ./...`
- `go vet ./...`
- `go build ./...`
- `node --check` for every changed viewer JavaScript module.
- `node --test` for all viewer JavaScript tests.
- `git diff --check`

## Delivery synchronization (2026-09-03)

Automated implementation and verification are complete. Agent inspection covered
detail hierarchy, Markdown safety, metadata/raw disclosures, local and external
link affordances, diagnostics, focus labels, and responsive density; the
required human visual-review gate remains pending. Do not close or archive this
issue until that review is explicitly approved.

## Scenario traceability

| Source rule | Scenario | Issue criterion | Verification evidence |
| --- | --- | --- | --- |
| Arbitrary source facts and provenance | `SC-003` | Detail retains source facts and provenance without changing the source bundle | detail payload and filesystem assertions |
| CommonMark/link safety | `SC-012` | Unsafe markup is neutralized and safe local/external links behave as specified | sanitizer/link tests and visual review |

## Handoff

Issue 070 adds durable profile/configuration controls without changing the
read-only inspection boundary. Issue 071 includes detail and link failures in
the final acceptance and regression matrix.

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
