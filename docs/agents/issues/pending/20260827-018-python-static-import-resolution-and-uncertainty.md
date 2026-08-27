# 018 — Python static import resolution and uncertainty

Execution type: AFK
Review gate: none
Status: ready-for-agent

## Parent PRD

`docs/architecture/analyze-source/python-analysis/prd.md`

## What to build

Extend the registered Python analyzer from issue 017 with static import extraction, deterministic local resolution, and honest uncertainty reporting. `import`, `from ... import ...`, package re-exports, and relative imports must become common `depends_on` observations only when the effective source roots and importing package context prove the target. External, standard-library, unresolved, conditional, and dynamic behavior must remain references or diagnostics with evidence and confidence; the analyzer must never fabricate local modules or execute Python to improve resolution.

Keep the implementation behind the Python analyzer boundary and the existing `analysis.Analyzer` contract. The canonical model, viewer, and export layers consume the common observations without learning Python syntax.

## Acceptance criteria

- [ ] `import package.module`, `from package import module`, and supported re-export forms produce source-located import observations; proven local targets become directed `depends_on` relationships with stable IDs and deduplicated contributor evidence.
- [ ] Relative imports resolve from the importing package/module context, including parent traversal and package `__init__` targets, only when the filesystem proves the target under an effective source root. Ambiguous or missing targets remain unresolved rather than being guessed.
- [ ] Standard-library and third-party targets are classified as references using deterministic, version-aware static rules; unresolved local-looking targets remain unresolved references/diagnostics. No reference is turned into a local module without proof.
- [ ] `importlib`, computed `__import__`, plugin discovery, and other unprovable dynamic targets produce `reference.scope=dynamic` and a recoverable diagnostic/confidence basis instead of a fabricated edge. Conditional imports retain condition evidence and make the result partial when the target cannot be proven for the selected view.
- [ ] Module, relationship, reference, and source-reference aggregation preserves all contributing import locations, tags, metadata, confidence, and deterministic ordering. Repeating the same run produces byte-stable JSON and canonical model output.
- [ ] Unreadable files, syntax errors, unsupported syntax, unresolved imports, and dynamic imports do not discard unrelated valid observations; partial status and severity/code fields follow the parent analysis contract.
- [ ] A fixture containing import-time side effects proves that resolution is entirely static: no target import, installation, interpreter invocation, or build-script execution occurs.
- [ ] Existing Go analysis and generic host/model/viewer/export behavior remain unchanged; no Python-specific logic is added to canonical normalization, scene projection, layout, or rendering.

## Artifact sync required

- Application PRD: `required: docs/prd.md` — update delivery status when the Python relationship path is implemented; the product workflow and scope remain unchanged.
- Application architecture summary: `required: docs/architecture/application-architecture-summary.md` — record the completed Python adapter's static-resolution/uncertainty behavior under the existing analyzer boundary.
- Owning capability node/artifacts: `required: .okf/capabilities/analyze-source.md; .okf/capabilities/analyze-source/python-analysis.md; docs/architecture/analyze-source/orchestration-status.md; docs/architecture/analyze-source/python-analysis/orchestration-status.md; docs/architecture/analyze-source/python-analysis/implementation-slice.md`.
- Issue registry: `required`; node `issues:` reference: `required`.
- Reason/no-impact decision: delivery and capability progress change; canonical observation/model schemas and all presentation boundaries remain unchanged.

## Blocked by

Blocked by `docs/agents/issues/pending/20260827-017-python-project-boundary-and-module-discovery.md`.

## User stories addressed

The Python child PRD has no standalone user-story IDs. This issue covers `PY-FR-003`, `PY-FR-004`, `PY-FR-005`, `SC-PY-002`, `SC-PY-003`, `SC-PY-004`, and the parent partial-result/evidence scenarios `SC-AS-001`, `SC-AS-002`, `SC-AS-005`, `SC-AS-006`, and `SC-AS-008`.

## Scenario traceability and verification plan

| Source rule / scenario | Issue coverage | Planned evidence |
|---|---|---|
| Absolute/relative static resolution; SC-PY-002 | Local `depends_on` relationships point only to proven package/module targets and retain line/column evidence | Absolute/relative/package-init fixture and relationship assertions |
| Dynamic uncertainty; SC-PY-003 | Dynamic imports remain dynamic references/diagnostics with partial status | `importlib`, computed `__import__`, conditional, and plugin-discovery fixture |
| Evidence aggregation; SC-PY-002 and SC-AS-005 | Repeated imports and multi-file contributors are merged without losing source facts | Relationship/source-reference aggregation tests and deterministic serialization |
| Safety; SC-PY-004 and SC-AS-006 | No Python process or import-time side effect occurs | Side-effect fixture, command audit, and read-only analyzer tests |

## Verification surfaces

- Backend boundary: focused Python analyzer tests plus repository race/vet/build/staticcheck/lint gates.
- Frontend integration: existing language-neutral viewer contract consumes the resulting model; no renderer-specific changes are expected.
- End-to-end: Python analysis → canonical normalization/model projection where supported; deferred harnesses must record a reason.
- Repository/OKF integrity: `git diff --check`, strict OKF validation, and synchronized issue/node references.
