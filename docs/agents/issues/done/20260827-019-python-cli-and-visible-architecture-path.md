# 019 — Python CLI and visible architecture path

Execution type: AFK
Review gate: visual-review
Status: done

## Parent PRD

`docs/architecture/analyze-source/python-analysis/prd.md`

## What to build

Complete the public Python journey on top of issues 017 and 018. Expose the Python contract's effective options through the existing analyze command, verify explicit and automatic analyzer selection, and exercise the unchanged canonical model, local viewer, hierarchy/evidence inspection, self-contained HTML, static SVG, and browser-current-canvas export paths with a representative Python repository. The user should see the same language-neutral architecture workflow, with Python-specific uncertainty represented in the existing diagnostics/references/evidence surfaces.

This is an integration and acceptance slice, not a new Python-specific viewer. Keep analyzer registration at the composition root, keep analyzer options separate from layout configuration, and do not change the model or export schemas.

## Acceptance criteria

- [x] The CLI can supply the Python options from the child contract, including repeatable `source_roots`, `python_version`, `include_stubs`, `include_tests`, and `exclude`, while preserving existing Go flags, option precedence, validation, cancellation, and deterministic fingerprints.
- [x] `arch-view analyzers` lists both built-in Go and Python manifests in deterministic order; explicit `--language python`, explicit analyzer selection, and auto-detection on a Python-only root select the Python analyzer with stable selection metadata. Mixed-marker ambiguity follows the existing host contract.
- [x] A representative Python project can run through `analyze --format analysis-json`, canonical `model normalize`/`validate`/`projection`, and the local viewer without special casing in `internal/model`, `internal/viewer/scene`, layout, or browser rendering.
- [x] The viewer exposes Python package/module hierarchy, directed relationships, source locations, references, confidence, and dynamic/unresolved diagnostics through the existing graph and list/details workflow; no stale or cross-root evidence is introduced.
- [x] The same Python model produces deterministic JSON, self-contained HTML, and static SVG artifacts. HTML has no external module/style/runtime dependencies, and browser Download SVG remains the current-canvas path; no Python runtime is embedded or required.
- [x] Repeated CLI/model/export runs with unchanged source, analyzer version, options, and configuration are byte-stable. Existing Go fixtures and output behavior remain byte-stable.
- [x] User visual review approves the Python top-level and drilled hierarchy views in windowed and full-canvas modes, including edge direction, package/module labels, references, evidence/diagnostic details, pan/zoom/fit/reset, and the existing export paths.

## Artifact sync required

- Application PRD: `required: docs/prd.md` — record the completed Python visible journey and update the implementation sequence/status; no new product scope is introduced.
- Application architecture summary: `required: docs/architecture/application-architecture-summary.md` — record that a second in-process language adapter reaches the existing model/viewer/export path without host or renderer branching.
- Owning capability node/artifacts: `required: .okf/capabilities/analyze-source.md; .okf/capabilities/analyze-source/python-analysis.md; docs/architecture/analyze-source/orchestration-status.md; docs/architecture/analyze-source/python-analysis/orchestration-status.md; docs/architecture/analyze-source/python-analysis/implementation-slice.md`.
- Issue registry: `required`; node `issues:` reference: `required`.
- Reason/no-impact decision: delivery, capability maturity, and implementation status change; public canonical schemas and viewer behavior remain shared contracts.

## Blocked by

Blocked by None. Issue 018 is complete and archived; this is now the next unblocked Python-analysis frontier.

## User stories addressed

The Python child PRD has no standalone user-story IDs. This issue covers the end-to-end portions of `PY-FR-001` through `PY-FR-005`, `SC-PY-001` through `SC-PY-005`, and parent journeys `SC-AS-001`, `SC-AS-002`, `SC-AS-004`, `SC-AS-005`, `SC-AS-006`, and `SC-AS-008`.

## Scenario traceability and verification plan

| Source rule / scenario | Issue coverage | Executed evidence |
|---|---|---|
| Public selection/options; SC-AS-004 and SC-PY-001 | CLI flags and host selection reach the Python analyzer with effective options | `cmd/arch-view/main_test.go` covers deterministic built-in manifest ordering; `cmd/arch-view/python_visible_journey_test.go` covers explicit language/analyzer selection, Python option precedence, auto-detection, mixed-marker ambiguity, and stable selection metadata/fingerprints; focused and full Go tests pass |
| Canonical model/viewer path; SC-AS-001, SC-AS-005 | Python observations normalize and render through existing language-neutral contracts | `TestPythonVisibleJourneyUsesSharedModelViewerAndExportPaths`: `analysis-json` → `model normalize` → `model validate` → `model projection`, viewer HTTP model/scene/source endpoints, hierarchy, directed relations, and evidence/details assertions |
| Partial uncertainty; SC-AS-002 and SC-PY-003 | Dynamic/unresolved imports remain usable and visible in graph/details/diagnostics | The representative fixture asserts dynamic and unresolved diagnostics, references, confidence, expanded reference scenes, and source locations without cross-root evidence |
| Artifact parity/determinism; SC-AS-008 | JSON, HTML, SVG, and browser-current-canvas artifacts preserve semantics and stable output | Repeated CLI analysis/JSON/HTML/SVG byte comparisons; HTML self-containment and current-canvas download markers; static SVG semantic markers; full Go/Node verification gates pass |
| User review; existing viewer journeys | Windowed/full-canvas Python graph is readable and navigable | The user explicitly approved the corrected live Python viewer after windowed/drilled/full-canvas inspection covered labels, arrows, expanded references, evidence, diagnostics, pan, zoom, reset, Fit, and the existing export controls |

## Verification surfaces

- Backend boundary: full Go verification suite plus focused CLI/analyzer/model/export tests.
- Frontend integration: Python hierarchy, imports, diagnostics, source evidence, navigation, layout controls, and current-canvas export in the existing viewer.
- End-to-end: Python repository → analysis → model → viewer/HTML/SVG; any unavailable harness must be recorded with its manual review path.
- Repository/OKF integrity: `git diff --check`, strict OKF validation, synchronized artifacts, and untouched upstream reference boundary.

## Verification record

The automated acceptance and repository gates pass:

- `go test ./... -count=1`
- `go test -race ./...`
- `go vet ./...`
- `go build ./...`
- `staticcheck ./...`
- `golangci-lint run`
- `node --check internal/viewer/web/app.js`
- `node internal/viewer/web/graph_route_test.js`
- `node internal/viewer/web/elk_spline_integration_test.js`
- `node internal/viewer/web/layout_request_test.js`
- `node internal/viewer/web/viewport_math_test.js`

The declared `visual-review` gate passed after the live review found and corrected a windowed Fit defect. The corrected viewer measured the visible graph container, contained all nodes in windowed and expanded-reference views, retained full-canvas behavior, and passed the complete automated gate set again.

## Visual review approval

The user explicitly approved the visual review on 2026-08-27. The reviewed representative project covered the top-level and drilled `archdemo` hierarchy in windowed and full-canvas modes, directed arrowheads, package/module labels, local-first and expanded references, evidence/source locations, confidence, dynamic/unresolved diagnostics, details, pan, zoom, Fit, reset, and the existing browser export controls. During review, windowed Fit was found to measure an aspect-ratio-expanded SVG instead of the clipped graph viewport; the defect was corrected, regression-tested, reverified live, and included in the approval.
