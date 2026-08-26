# 004 — Evidence drill-down and source inspection

Execution type: AFK
Review gate: visual-review
Status: done

## Parent PRD

docs/architecture/explore-architecture/prd.md

## What to build

Complete the first investigation workflow on top of the local overview.

- Add hierarchy drill-down, breadcrumbs, back navigation, search, selection, zoom, pan, and fit behavior.
- Show module metadata, edge direction, relationship evidence, source locations, external/unresolved references, diagnostics, confidence, cycles, and layers.
- Add a module/group imports list with scope filters for standard-library, external, unresolved, and dynamic references. Keep the default architecture graph local-first and make reference expansion a deliberate toggle.
- Keep optional user-adjusted node positions and viewport state session-scoped and keyed by model revision and hierarchy path; never mutate the canonical model.
- Provide a read-only source excerpt endpoint and panel with project-root containment checks.
- Keep selections and evidence IDs stable across scene updates and prevent stale evidence after reanalysis.
- Preserve an accessible list/details path with keyboard navigation and equivalent facts to the graphic view.

## Visual review refinements

- [x] Give buttons a restrained hover/active treatment and vertically center status pills beside their actions.
- [x] Add an explicit `100%` zoom reset beside `Fit`.
- [x] Remove graph scrollbars from the zoom/pan surface and lock document/root scrolling while the canvas is expanded.
- [x] Redesign the expanded-canvas header into a compact title/actions row plus a context and summary row so breadcrumbs, layer/reference summaries, and viewport controls preserve graph space.
- [x] Cap expanded-canvas `Fit` at `100%` while retaining the larger windowed fit ceiling.
- [x] Provide `Reset layout` to discard session-only manual node positions and restore the calculated layout.
- [x] Recalculate manual-movement edge routes synchronously with deterministic orthogonal geometry during drag and after drop, keeping the same route strategy throughout the interaction.
- [x] Align navigation breadcrumbs with layer/reference pills even when the summary wraps.
- [x] Make `Fit` use the visible content bounds, the SVG base scale, and a centered pan so dense expanded-reference scenes do not get double-scaled into the corner.
- [x] Coalesce drag renders and keep the node on its manual position so it does not flicker between calculated and manual positions; the interaction zoom guardrail is now `40,000%`, while `Fit` remains capped separately. Pan movement is normalized to the SVG base scale through a single `PAN_SPEED` setting.
- [x] Preserve double-click group navigation with immediate single-click node selection by detecting the second click across SVG re-renders; reserve node movement for `Shift`-drag.
- [x] Move the pan/move hint into the graph as an overlay and give breadcrumbs their own dedicated text-link row below the layer/reference summary.

These are implementation refinements of the existing viewport and visual-review scope; the final human visual gate was explicitly approved.

## Acceptance criteria

- [x] A user can drill into a hierarchy group, inspect its modules and relationships, and return with breadcrumbs or back navigation.
- [x] Search and selection identify a module or relationship and reveal its source evidence and source location.
- [x] A selected module or group exposes an accessible imports list with individual target scope, confidence, counts, and source evidence; reference visibility can be switched between hidden, aggregated, and expanded.
- [x] Source excerpts are read-only, line-aware, confined to the analyzed project root, and reject traversal or cross-root requests.
- [x] Cycles, unresolved/external references, diagnostics, confidence, and layer information are visible and explainable in both graphic and list/details modes.
- [x] Reanalysis replaces the active model/evidence revision safely; failed reanalysis leaves the prior revision active and never presents stale evidence as current.
- [x] Keyboard navigation and accessible labels/descriptions expose the same architecture facts as the graphic view.
- [x] Fit, pan, zoom, and optional node movement preserve a readable session layout without changing canonical model data; layout overrides are isolated by model revision and hierarchy path.
- [x] A visual review confirms that navigation, evidence, cycle states, and source inspection remain legible and coherent.

## Artifact sync required

- Application PRD: none — this realizes the already specified investigation workflow.
- Application architecture summary: none — no new boundary or renderer decision is introduced.
- Owning capability node/artifacts: required: .okf/capabilities/explore-architecture.md; docs/architecture/explore-architecture/orchestration-status.md; docs/architecture/analyze-source/go-analysis/implementation-slice.md.
- Issue registry: required; node issues reference required: .okf/capabilities/explore-architecture.md.
- Reason/no-impact decision: delivery truth is being added; product and architecture truth remain unchanged.

## Blocked by

—

## User stories addressed

- US-EX-001
- US-EX-002
- US-GM-002

## Contract and scenario trace

- Contract: docs/architecture/explore-architecture/canonical-api-cli-contract.md
- Scenarios: SC-EX-002, SC-EX-003, SC-EX-004, SC-EX-005, SC-EX-006, SC-EX-007, SC-EX-008, SC-EX-009, SC-EX-010

## Scenario traceability and verification

| Source rule / use case | Scenario | Issue criterion | Verification obligation and evidence | State |
|---|---|---|---|---|
| ViewSessionService hierarchy path, breadcrumbs, and safe back navigation | SC-EX-002 | Drill into a group and return | `go test ./internal/viewer -count=1`; browser review of `Open group`, breadcrumbs, and `Back` on the local viewer | implemented; human visual gate approved |
| EvidenceInspectionService for modules, relations, imports, and source locations | SC-EX-003 | Selection reveals evidence and imports | `TestBuildScenePreservesTopLevelSemanticsAndAggregation`; browser selection of a group, imports scope filter, and evidence list | implemented |
| Read-only, root-confined source inspection | SC-EX-004 | Safe line-aware source excerpts | `TestServerServesContainedReadOnlySourceAndRejectsTraversal`; source panel displays path, line range, and read-only state | implemented |
| Cycle, diagnostic, layer, confidence, and reference-scope projection rules | SC-EX-005 | Explain uncertainty and layers in graph/list | `go test ./internal/viewer -count=1` scene/reference tests; accessible list includes cycle and diagnostic items; live browser list/details review | implemented |
| Progressive disclosure and local-first reference visibility | SC-EX-006 | Deliberate reference expansion and imports details | `TestBuildSceneReferenceVisibilityModes`; browser toggles Local-first, Boundary summary, and Show imports | implemented |
| Accessible alternative and keyboard/list parity | SC-EX-007 | Same facts without pointer graphics | Semantic graph roles, keyboard handlers, accessible list/import/evidence buttons; browser DOM snapshot and keyboard-capable controls | implemented |
| Reanalysis replacement and stale-evidence invalidation | SC-EX-008 | Atomic valid replacement; failed run preserves prior model | `TestServerReanalysisReplacesOnlyValidRevision`; browser Reanalyze action completed without error and resets selection/source state | implemented |
| Import boundary detail by scope, confidence, count, and evidence | SC-EX-009 | Individual imports available outside overview graph | `TestBuildSceneReferenceVisibilityModes`; selected-node imports list with standard-library/external/unresolved/dynamic filters | implemented |
| Session-owned viewport/layout keyed by revision and hierarchy path | SC-EX-010 | Fit/pan/zoom/manual positions do not mutate model | Browser zoom changed and survived reload for the same scene key; dense expanded `Fit` used the SVG base scale and centered content at 100%, navigation aligned with wrapped summaries, `100%` reset restored the zoom, a moved node kept a stable manual position while deterministic orthogonal routes refreshed continuously and remained unchanged in strategy after drop, `Reset layout` restored calculated positions, and graph/root overflow were locked without changing canonical model data | implemented |

## Verification summary

- Backend boundary: `go test ./... -count=1`, `go vet ./...`, and source/reanalysis endpoint tests pass.
- Frontend integration: `node --check internal/viewer/web/app.js` passes; the Go server embeds the ELK browser bundle for calculated layout, while manual edge routing stays local and synchronous in the renderer.
- End-to-end: the local CLI opened a freshly analyzed Go repository at `http://127.0.0.1:45663/`; the browser showed the model and completed the dense-fit, alignment, drag-stability, and reset route checks successfully.
- Visual refinement smoke check: hover styles were observed on the full-canvas action, navigation and wrapped layer/reference summaries shared a top alignment, windowed and expanded graphs had no graph scrollbar, dense expanded `Fit` stayed at `100%` and centered the visible content, the explicit `100%` reset restored the zoom, a dragged node kept its manual position without flicker while the same deterministic orthogonal edge calculation was used before and after drop, and `Reset layout` restored the calculated positions. The current review URL emitted no browser errors or warnings.
- Latest interaction adjustment: static JavaScript syntax, Go tests, vet, and build checks pass; the user approved the `40,000%` ceiling, normalized pan speed, double-click navigation, graph hint overlay, dedicated breadcrumb row, and final full-canvas spacing.
- Review loop: the strict code-review loop found and resolved stale reanalysis scene loading, invalid failed-revision replacement, invalid explicit source line ranges, and an unchecked source-file close; the final pass reported no actionable P0–P2 findings.
- Artifact impact: product PRD and application architecture summary are unchanged by this delivery slice; capability and delivery records below carry the implementation truth.
