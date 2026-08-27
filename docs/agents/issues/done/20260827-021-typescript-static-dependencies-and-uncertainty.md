# 021 — TypeScript static dependencies and uncertainty

Execution type: AFK
Review gate: none
Status: done

## Parent PRD

`docs/architecture/analyze-source/typescript-analysis/prd.md`

## What to build

Extend the registered TypeScript analyzer with conservative static import/export extraction and resolution. Proven relative, aliased, root-directory, self-package, and supported package-export targets become common directed `depends_on` observations. Import/export kind, type-only semantics, aliases, runtime conditions, source locations, confidence, and contributor evidence remain visible. External packages, excluded targets, unresolved aliases/exports, and computed dynamic loading remain references or recoverable diagnostics; the analyzer never executes TypeScript or uses an environment resolver.

## Acceptance criteria

- [x] `import`, side-effect imports, type imports, `export ... from`, re-exports, `require`, and literal dynamic imports produce source-located observations with stable deduplicated relationships.
- [x] Relative paths resolve through TypeScript source extensions and directory indexes only when the filesystem and selected source scope prove a local target. `baseUrl`, `paths`, `rootDirs`, and package self-reference aliases retain alias/resolution provenance.
- [x] `package.json` `exports`, `imports`, `types`/`typings`, and runtime-relevant `import`/`require`/`default` conditions are used conservatively. Unsupported or missing package exports remain unresolved rather than becoming guessed local edges.
- [x] Type-only imports, re-export forms, aliases, `require`, and dynamic-import kinds are retained in relationship metadata without changing the common `depends_on` type.
- [x] External, standard-library/runtime, excluded, unresolved, and dynamic targets become references with scope, metadata, confidence, and source evidence. Computed dynamic paths never fabricate a local relationship; literal dynamic paths may resolve locally when proven.
- [x] Unresolved aliases/package exports, computed loading, malformed syntax, and excluded targets retain unrelated valid observations and produce recoverable diagnostics/partial status with source locations.
- [x] Repeated imports and multi-file contributors aggregate all source-reference IDs and deterministic metadata; repeated unchanged analysis is byte-stable.
- [x] The implementation remains entirely behind `internal/tsanalyzer` and the common analyzer contract; canonical normalization and presentation layers remain language-neutral.

## Artifact sync required

- Application PRD: `required: docs/prd.md` — record the TypeScript static relationship/uncertainty implementation status; product scope remains unchanged.
- Application architecture summary: `required: docs/architecture/application-architecture-summary.md` — record static TypeScript resolution and uncertainty behavior under the existing adapter boundary.
- Owning capability node/artifacts: `required: .okf/capabilities/analyze-source.md; .okf/capabilities/analyze-source/typescript-analysis.md; docs/architecture/analyze-source/orchestration-status.md; docs/architecture/analyze-source/typescript-analysis/orchestration-status.md; docs/architecture/analyze-source/typescript-analysis/implementation-slice.md`.
- Issue registry: `required`; node `issues:` reference: `required`.
- Reason/no-impact decision: delivery and capability progress change; canonical observation/model schemas, host orchestration, viewer behavior, and export boundaries remain unchanged.

## Blocked by

Blocked by `docs/agents/issues/done/20260827-020-typescript-project-boundary-and-module-discovery.md`.

## User stories addressed

The TypeScript child PRD has no standalone user-story IDs. This issue covers `TS-FR-003`, `TS-FR-004`, `TS-FR-005`, and `SC-TS-002` through `SC-TS-004`, plus parent partial-result/evidence behavior.

## Scenario traceability and verification plan

| Source rule / scenario | Issue coverage | Planned evidence |
|---|---|---|
| Alias/local resolution; `TS-FR-003`, `SC-TS-002` | Relative, baseUrl/paths, rootDirs, self-package, index, and extension resolution | Fixtures with aliases, declaration files, ambiguous candidates, package boundaries, and outside-root targets |
| Import/export kinds; `TS-FR-005`, `SC-TS-003` | Import, type-import, export/re-export, require, aliases, source positions, and relationship aggregation | Multi-file fixture with repeated contributors and all supported statement forms |
| Dynamic/unsupported behavior; `TS-FR-004`, `SC-TS-004` | Literal/computed dynamic imports, unresolved aliases/exports, external references, diagnostics, confidence, and partial status | Static fixture with computed templates, missing packages, export-map conditions, and invalid syntax |
| Safety and determinism; parent `AS-FR-004`/`AS-FR-006`/`AS-FR-008` | No target-code execution; stable IDs/order and retained valid observations | Side-effect package fixture, repeated analyzer/model JSON comparison, result validation |

## Verification surfaces

- Backend boundary: focused TypeScript analyzer tests plus full repository gates.
- Frontend integration: common model output preserves TypeScript relationships, references, diagnostics, confidence, and evidence without language branches.
- End-to-end: supported through the existing CLI/model path after issue 020 registration.
- Repository/OKF integrity: synchronized issue/node/application records and strict validation.
