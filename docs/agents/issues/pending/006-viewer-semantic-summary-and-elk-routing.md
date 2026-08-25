# 006 — Viewer semantic summaries and ELK edge routing

Execution type: AFK
Review gate: visual-review
Status: awaiting-human-review

## Parent PRD

docs/architecture/explore-architecture/prd.md

## What to build

Refine the first local viewer after visual review exposed two presentation
problems: a group-level internal relationship is rendered as a cycle-like
self-loop, and the custom geometry produces hard-to-follow edge routes.

- Add the temporary repository README as the contributor-facing explanation of
  layers, diagnostics, tags, confidence, aggregation, and layout ownership.
- Keep real canonical cycles visible, but summarize non-cycle relationships
  between child modules of a collapsed group as an internal-relationship count
  with canonical contributor IDs and evidence preserved.
- Distinguish stable node identity from relationship confidence in the scene and
  inspection UI. Local module/group identity must not be presented as a
  computed confidence score.
- Serve a pinned ELK/elkjs bundle and worker locally, use the layered algorithm
  for projected node placement, and consume its edge sections/bend points in
  the existing SVG renderer without changing canonical relationship direction.
- Retain the deterministic layer-based layout as a replaceable fallback when
  the worker is unavailable or a layout request fails.

## Acceptance criteria

- [x] README.md explains the current viewer semantics and extension boundaries,
  including diagnostics/tags as analyzer facts and ELK as presentation layout.
- [x] A non-cycle group self-loop is absent from the top-level rendered edge set
  and the owning node exposes `N internal relationships` plus contributor IDs.
- [x] A real canonical cycle remains a directed cycle indicator and is not
  collapsed into the internal summary.
- [x] Internal relationship evidence remains reachable through the owning
  node/group and the list/details view.
- [x] Local module/group inspection shows stable identity separately from
  relationship confidence; reference nodes and relationships retain their
  computed confidence states.
- [x] The README records the active ELK layered adapter, local worker asset,
  SVG edge-route consumption, and deterministic fallback boundary.
- [x] Edge direction remains `from -> to`; the top-level `cmd -> internal`
  relationship remains semantically unchanged.
- [x] Routed-edge hit areas remain transparent and do not paint over the graph;
  they are interaction-only stroke targets.
- [x] Existing deterministic layout behavior remains available and all
  backend/frontend checks pass.
- [ ] A visual review confirms the self-loop summary, readable edge routing,
  layer labels, diagnostics/tags/identity wording, and keyboard/list parity.

## Artifact sync required

- Application PRD: required: docs/prd.md — record the clarified top-level
  aggregation and viewer-owned layout behavior.
- Application architecture summary: required:
  docs/architecture/application-architecture-summary.md — record that layout
  routing and collapsed-group presentation belong to the viewer boundary while
  canonical relationships remain model-owned.
- Owning capability node/artifacts: required:
  .okf/capabilities/explore-architecture.md;
  docs/architecture/explore-architecture/orchestration-status.md;
  docs/architecture/explore-architecture/prd.md;
  docs/architecture/explore-architecture/canonical-domain-model.md;
  docs/architecture/explore-architecture/canonical-api-cli-contract.md.
- Issue registry: required; add this issue to the owning capability references.
- Reason/no-impact decision: no new capability or analyzer boundary is
  introduced. The README and viewer slice clarify and refine the existing
  exploration capability; `external/` remains read-only and untouched.

## Blocked by

—

## Verification surfaces

- Backend boundary: scene tests and server asset/contract tests.
- Frontend integration: JavaScript syntax check and locally served ELK bundle
  plus worker asset; the renderer consumes returned node positions and edge
  sections/bend points.
- End-to-end: local `open --project` smoke test and live visual review.
- Repository/OKF integrity: `git diff --check` and strict OKF validation.
