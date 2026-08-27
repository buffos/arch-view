# 017 — Python project boundary and module discovery

Execution type: AFK
Review gate: none
Status: done

## Parent PRD

`docs/architecture/analyze-source/python-analysis/prd.md`

## What to build

Implement the first registered in-process Python analyzer path. Given a Python repository, detect the project without executing it, resolve its effective configuration and source roots, discover eligible package/module files, and emit deterministic common analysis observations with source evidence. Register the analyzer at the composition root so `arch-view analyzers` and `arch-view analyze --language python` can use this module-discovery path. Keep the host generic: no Python branch or language switch belongs in `internal/analysis`.

The slice may emit an empty relationship collection until issue 018, but its module-only result must already be valid, deterministic, status-bearing, and consumable by canonical normalization and the existing viewer/model path.

## Acceptance criteria

- [x] The manifest is valid and stable: `id=org.archview.python`, `language=python`, `api_version=arch-view.analyzer/v1`, detection markers for `pyproject.toml`, `setup.cfg`, and `setup.py`, capabilities `detect`, `static_dependencies`, and `dynamic_diagnostics`, and the typed options from the Python contract.
- [x] The analyzer is registered only at the composition root; analyzer listing, explicit Python selection, and unambiguous auto-detection work through the existing host without modifying host orchestration or adding a Python-specific switch.
- [x] Project-marker precedence is deterministic: `pyproject.toml` is preferred, then supported `setup.cfg`/`setup.py` metadata; `setup.py` is parsed as data and is never imported or executed. Invalid or unreadable configuration produces a recoverable diagnostic where the rest of the project remains usable.
- [x] Effective source roots follow explicit `source_roots` option > supported project configuration > `src/` when present > selected project root. Paths are normalized relative to the project boundary and remain safe within that root.
- [x] Eligible `.py` files are discovered deterministically; `.pyi` files are excluded by default and included only with `include_stubs=true`; test files, `__pycache__`, generated/build/cache/vendor directories, `.git`, `external`, and configured exclusions follow the Python contract.
- [x] Regular packages, namespace packages, package `__init__` evidence, and modules are represented with stable project-scoped IDs, qualified names, explicit hierarchy, kind metadata, tags, and repository-relative source references. A file is evidence and is not silently promoted to an unrelated graph node.
- [x] Syntax, unreadable-file, unsupported-version, and conflicting-layout problems are recoverable diagnostics with partial status where appropriate; unrelated eligible files still produce usable module observations.
- [x] The analyzer and its tests never import, install, execute, or invoke the target Python project or its build scripts. Repeating the same analysis with unchanged source/options produces byte-stable output and stable IDs/order.
- [x] A CLI smoke fixture can run `arch-view analyze --project <python-root> --language python --format analysis-json --output <file>` and normalize the result through `arch-view model normalize` without changing the existing Go path.

## Implementation notes

- Added `internal/pyanalyzer` as an in-process implementation of the common analyzer contract. It reads `pyproject.toml`, `setup.cfg`, or `setup.py` as data, resolves safe source roots, and emits package/module observations with file evidence.
- Python import relationships remain intentionally empty until issue 018. Issue 019 still owns the public Python visible-journey/export acceptance path.
- CLI analyzer options are passed only when explicitly supplied, preserving Go defaults while allowing Python's typed options to resolve through the generic host.

## Artifact sync required

- Application PRD: `required: docs/prd.md` — update the current delivery sequence/status when this Python slice is implemented; product scope itself remains unchanged.
- Application architecture summary: `required: docs/architecture/application-architecture-summary.md` — record the completed in-process Python adapter while preserving the existing analyzer/plugin boundary and composition-root rule.
- Owning capability node/artifacts: `required: .okf/capabilities/analyze-source.md; .okf/capabilities/analyze-source/python-analysis.md; docs/architecture/analyze-source/orchestration-status.md; docs/architecture/analyze-source/python-analysis/orchestration-status.md; docs/architecture/analyze-source/python-analysis/implementation-slice.md`.
- Issue registry: `required`; node `issues:` reference: `required` for the parent analysis node, Python child, and any directly exercised plugin-runtime boundary.
- Reason/no-impact decision: delivery and implementation-status truth change; no canonical model, public schema, topology, or upstream-reference change is intended.

## Blocked by

None - can start immediately.

## User stories addressed

The Python child PRD has no standalone user-story IDs. This issue covers `PY-FR-001`, `PY-FR-002`, `PY-FR-005`, `SC-PY-001`, `SC-PY-004`, and `SC-PY-005`, plus the parent analyzer selection/safety path in `SC-AS-001`, `SC-AS-004`, `SC-AS-006`, and `SC-AS-008`.

## Scenario traceability and verification plan

| Source rule / scenario | Issue coverage | Planned evidence |
|---|---|---|
| Project markers and source-root precedence; SC-PY-001 | Detection selects the correct Python boundary and produces the expected package/module hierarchy | Python marker/config fixtures and analyzer-host selection tests |
| Stub/test/exclusion behavior; SC-PY-005 and SC-AS-006 | File scope follows defaults and explicit options without crossing the project root | Scope fixture with `.py`, `.pyi`, tests, generated/cache/vendor/external paths |
| Read-only parsing; SC-PY-004 | The analyzer never executes target code or setup metadata | Side-effect fixture plus process/command audit and static analyzer tests |
| Evidence and determinism; SC-PY-001 and SC-AS-008 | IDs, paths, hierarchy, tags, source references, and module ordering are stable | Repeated JSON serialization and evidence assertions |

## Verification surfaces

- Backend boundary: `go test ./... -count=1`, race/vet/build/staticcheck/lint, and focused Python analyzer fixtures.
- Frontend integration: not a new renderer feature; the module-only model must be accepted by the existing viewer contract.
- End-to-end: supported where the CLI/model harness exists; deferred coverage must record its fixture/reason.
- Repository/OKF integrity: `git diff --check`, strict OKF validation, and upstream reference-boundary checks.
