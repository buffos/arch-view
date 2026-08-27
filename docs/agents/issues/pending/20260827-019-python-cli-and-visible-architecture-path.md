# 019 — Python CLI and visible architecture path

Execution type: AFK
Review gate: visual-review
Status: ready-for-agent

## Parent PRD

`docs/architecture/analyze-source/python-analysis/prd.md`

## What to build

Complete the public Python journey on top of issues 017 and 018. Expose the Python contract's effective options through the existing analyze command, verify explicit and automatic analyzer selection, and exercise the unchanged canonical model, local viewer, hierarchy/evidence inspection, self-contained HTML, static SVG, and browser-current-canvas export paths with a representative Python repository. The user should see the same language-neutral architecture workflow, with Python-specific uncertainty represented in the existing diagnostics/references/evidence surfaces.

This is an integration and acceptance slice, not a new Python-specific viewer. Keep analyzer registration at the composition root, keep analyzer options separate from layout configuration, and do not change the model or export schemas.

## Acceptance criteria

- [ ] The CLI can supply the Python options from the child contract, including repeatable `source_roots`, `python_version`, `include_stubs`, `include_tests`, and `exclude`, while preserving existing Go flags, option precedence, validation, cancellation, and deterministic fingerprints.
- [ ] `arch-view analyzers` lists both built-in Go and Python manifests in deterministic order; explicit `--language python`, explicit analyzer selection, and auto-detection on a Python-only root select the Python analyzer with stable selection metadata. Mixed-marker ambiguity follows the existing host contract.
- [ ] A representative Python project can run through `analyze --format analysis-json`, canonical `model normalize`/`validate`/`projection`, and the local viewer without special casing in `internal/model`, `internal/viewer/scene`, layout, or browser rendering.
- [ ] The viewer exposes Python package/module hierarchy, directed relationships, source locations, references, confidence, and dynamic/unresolved diagnostics through the existing graph and list/details workflow; no stale or cross-root evidence is introduced.
- [ ] The same Python model produces deterministic JSON, self-contained HTML, and static SVG artifacts. HTML has no external module/style/runtime dependencies, and browser Download SVG remains the current-canvas path; no Python runtime is embedded or required.
- [ ] Repeated CLI/model/export runs with unchanged source, analyzer version, options, and configuration are byte-stable. Existing Go fixtures and output behavior remain byte-stable.
- [ ] User visual review approves the Python top-level and drilled hierarchy views in windowed and full-canvas modes, including edge direction, package/module labels, references, evidence/diagnostic details, pan/zoom/fit/reset, and the existing export paths.

## Artifact sync required

- Application PRD: `required: docs/prd.md` — record the completed Python visible journey and update the implementation sequence/status; no new product scope is introduced.
- Application architecture summary: `required: docs/architecture/application-architecture-summary.md` — record that a second in-process language adapter reaches the existing model/viewer/export path without host or renderer branching.
- Owning capability node/artifacts: `required: .okf/capabilities/analyze-source.md; .okf/capabilities/analyze-source/python-analysis.md; docs/architecture/analyze-source/orchestration-status.md; docs/architecture/analyze-source/python-analysis/orchestration-status.md; docs/architecture/analyze-source/python-analysis/implementation-slice.md`.
- Issue registry: `required`; node `issues:` reference: `required`.
- Reason/no-impact decision: delivery, capability maturity, and implementation status change; public canonical schemas and viewer behavior remain shared contracts.

## Blocked by

Blocked by `docs/agents/issues/pending/20260827-018-python-static-import-resolution-and-uncertainty.md`.

## User stories addressed

The Python child PRD has no standalone user-story IDs. This issue covers the end-to-end portions of `PY-FR-001` through `PY-FR-005`, `SC-PY-001` through `SC-PY-005`, and parent journeys `SC-AS-001`, `SC-AS-002`, `SC-AS-004`, `SC-AS-005`, `SC-AS-006`, and `SC-AS-008`.

## Scenario traceability and verification plan

| Source rule / scenario | Issue coverage | Planned evidence |
|---|---|---|
| Public selection/options; SC-AS-004 and SC-PY-001 | CLI flags and host selection reach the Python analyzer with effective options | Command tests for manifests, explicit/automatic selection, precedence, and fingerprints |
| Canonical model/viewer path; SC-AS-001, SC-AS-005 | Python observations normalize and render through existing language-neutral contracts | CLI integration, model validation/projection, viewer server, and details/evidence assertions |
| Partial uncertainty; SC-AS-002 and SC-PY-003 | Dynamic/unresolved imports remain usable and visible in graph/details/diagnostics | Partial fixture through analysis, model, viewer, and export paths |
| Artifact parity/determinism; SC-AS-008 | JSON, HTML, SVG, and browser-current-canvas artifacts preserve semantics and stable output | Self-containment, repeated-output, and export parity tests |
| User review; existing viewer journeys | Windowed/full-canvas Python graph is readable and navigable | User visual review after automated gates pass |

## Verification surfaces

- Backend boundary: full Go verification suite plus focused CLI/analyzer/model/export tests.
- Frontend integration: Python hierarchy, imports, diagnostics, source evidence, navigation, layout controls, and current-canvas export in the existing viewer.
- End-to-end: Python repository → analysis → model → viewer/HTML/SVG; any unavailable harness must be recorded with its manual review path.
- Repository/OKF integrity: `git diff --check`, strict OKF validation, synchronized artifacts, and untouched upstream reference boundary.
