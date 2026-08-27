# 020 — TypeScript project boundary and module discovery

Execution type: AFK
Review gate: none
Status: done

## Parent PRD

`docs/architecture/analyze-source/typescript-analysis/prd.md`

## What to build

Implement the first registered in-process TypeScript analyzer path. Given a repository with one selected `tsconfig` project, detect and select its configuration, resolve JSONC and the safe `extends` chain, apply static compiler/project boundary data, discover eligible source modules, and emit deterministic common observations with file evidence. Register the adapter at the composition root so the existing host and CLI can select it. Keep import relationships empty until issue 021, but make the module-only result valid, status-bearing, and consumable by canonical normalization.

## Acceptance criteria

- [x] The manifest is valid and stable: `id=org.archview.typescript`, `language=typescript`, `api_version=arch-view.analyzer/v1`, markers for `tsconfig.json` and `package.json`, capabilities `detect`, `static_dependencies`, `aliases`, and `exports`, and the typed options from the TypeScript contract.
- [x] Detection is read-only and deterministic; a `tsconfig.json` marker has higher confidence than `package.json`, and an explicit analyzer selection rejects an unsupported root clearly.
- [x] One config is selected per run. Multiple `tsconfig` candidates require the `config` option; the selected config and its `extends` chain remain inside the repository and are read as JSONC data without running a compiler or script.
- [x] Effective `include`, `exclude`, `files`, `rootDir`, `rootDirs`, `outDir`, `allowJs`, package metadata, project references, and runtime context are represented safely enough for source discovery; malformed or unsafe configuration produces recoverable diagnostics.
- [x] `.ts` and `.tsx` files are included by default. `.js` and `.jsx` are included only when project `allowJs` or `include_js=true` permits them. Tests, `node_modules`, `.git`, generated/build/cache/output directories, directories named `external`, and configured exclusions follow the TypeScript contract.
- [x] Modules have stable project-scoped IDs, repository-relative paths, explicit hierarchy, extension/declaration metadata, tags, and source evidence. Files are not promoted into unrelated graph nodes.
- [x] Unreadable files, malformed config, unsafe paths, and empty eligible source sets retain usable output with recoverable diagnostics and partial status where applicable; repeated analysis is byte-stable.
- [x] `arch-view analyzers` and explicit TypeScript analysis use the common host contract without TypeScript-specific branches in `internal/analysis`, `internal/model`, the viewer, or exporters.

## Artifact sync required

- Application PRD: `required: docs/prd.md` — record TypeScript as the active implementation frontier while preserving the existing product scope and language-plugin sequence.
- Application architecture summary: `required: docs/architecture/application-architecture-summary.md` — record the in-process TypeScript adapter under the existing analyzer boundary; no shared consumer boundary changes.
- Owning capability node/artifacts: `required: .okf/capabilities/analyze-source.md; .okf/capabilities/analyze-source/typescript-analysis.md; docs/architecture/analyze-source/orchestration-status.md; docs/architecture/analyze-source/typescript-analysis/orchestration-status.md; docs/architecture/analyze-source/typescript-analysis/implementation-slice.md`.
- Issue registry: `required`; node `issues:` reference: `required` for the parent analysis node and TypeScript child.
- Reason/no-impact decision: delivery and implementation-status truth change; product topology, canonical schemas, viewer semantics, and upstream-reference boundaries do not.

## Blocked by

None - can start immediately.

## User stories addressed

The TypeScript child PRD has no standalone user-story IDs. This issue covers `TS-FR-001`, `TS-FR-002`, `SC-TS-001`, and `SC-TS-005`, plus the shared analyzer selection/safety path.

## Scenario traceability and verification plan

| Source rule / scenario | Issue coverage | Planned evidence |
|---|---|---|
| Config selection and inheritance; `TS-FR-001`, `SC-TS-001` | Detection, explicit config selection, JSONC parsing, safe `extends`, package context, and effective source scope | Analyzer fixtures with comments/trailing commas, inherited options, missing/cyclic/outside configs, and ambiguous candidates |
| Module scope and hierarchy; `TS-FR-002`, `SC-TS-005` | TypeScript/TSX defaults, opt-in JavaScript/JSX, tests/output/cache/external exclusions, stable IDs and evidence | Repository fixture with configured include/exclude, root/out directories, test files, declarations, JS/JSX, and repeated JSON comparison |
| Safety; parent `AS-FR-008` | No compiler, package script, target-code, or outside-root execution/read | Side-effect package/config fixture and static implementation inspection |

## Verification surfaces

- Backend boundary: focused TypeScript analyzer tests, full Go tests, race/vet/build/staticcheck/lint where available.
- Frontend integration: module-only output must normalize through the existing model/viewer contract; no new renderer behavior is expected.
- End-to-end: CLI analysis JSON and model normalization smoke tests where supported.
- Repository/OKF integrity: `git diff --check`, strict OKF validation, synchronized references, and untouched upstream boundary.
