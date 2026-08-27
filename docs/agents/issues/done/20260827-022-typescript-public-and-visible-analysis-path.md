# 022 — TypeScript public and visible analysis path

Execution type: AFK
Review gate: visual-review
Status: done

## Parent PRD

`docs/architecture/analyze-source/typescript-analysis/prd.md`

## What to build

Complete the TypeScript public journey through the existing composition root and shared consumers. Expose the specified TypeScript options through `analyze` and `open`, verify deterministic analyzer selection and output, normalize TypeScript analysis into the canonical model, and exercise the existing viewer, source-evidence, JSON, HTML, and SVG paths. The visible result must use the current language-neutral model/viewer/export contracts without TypeScript-specific switches.

## Acceptance criteria

- [x] `arch-view analyzers` lists the TypeScript manifest deterministically; explicit `--language typescript`/`--analyzer org.archview.typescript` and unambiguous auto-detection select it through the generic host.
- [x] CLI and open commands expose and forward `--config`, `--include-js`, `--include-tests`, `--runtime`, and `--exclude` only as selected analyzer options, preserving existing Go/Python option behavior.
- [x] A representative TypeScript repository completes `analysis-json` → canonical model normalization/validation → hierarchy projection and opens through the existing local viewer with directed relationships, references, diagnostics, confidence, and source evidence.
- [x] Deterministic JSON, self-contained HTML, and SVG exports preserve TypeScript module/relationship semantics; repeated analysis and export output is byte-stable where the existing contracts require it.
- [x] Viewer/model/export consumers contain no TypeScript-specific parsing or branching, and existing Go/Python tests and behavior remain unchanged.
- [x] The final visual review covers windowed and full-canvas graph readability, hierarchy, labels, directed arrows, references/diagnostics, evidence/source locations, navigation, Fit/reset, and existing export controls; any defect found is regression-tested and reverified.
- [x] All required application, capability, issue-registry, and `.okf` synchronization is complete before the TypeScript node is advanced to `implemented`.

## Artifact sync required

- Application PRD: `required: docs/prd.md` — record the completed TypeScript implementation sequence and visible journey while preserving the product boundary.
- Application architecture summary: `required: docs/architecture/application-architecture-summary.md` — record the completed adapter and shared-consumer verification with no boundary change.
- Owning capability node/artifacts: `required: .okf/capabilities/analyze-source.md; .okf/capabilities/analyze-source/typescript-analysis.md; docs/architecture/analyze-source/orchestration-status.md; docs/architecture/analyze-source/typescript-analysis/orchestration-status.md; docs/architecture/analyze-source/typescript-analysis/implementation-slice.md`.
- Issue registry: `required`; node `issues:` reference: `required`.
- Reason/no-impact decision: delivery and implementation-status truth change; no new topology, canonical schema, viewer contract, layout policy, or upstream-reference scope is intended.

## Blocked by

Blocked by None. Issues 020 and 021 are complete and archived; this is now the next unblocked TypeScript-analysis frontier.

## Automated implementation evidence

The public TypeScript path is implemented and verified by `cmd/arch-view/typescript_visible_journey_test.go`: manifest listing, explicit/automatic selection, option forwarding, deterministic analysis, canonical normalization/validation, hierarchy projection, viewer model/projection/source routes, and repeated JSON/HTML/SVG output are covered. The full repository test, race, vet, build, static-analysis, lint, strict OKF, and diff checks are green.

## Human review approval

The user explicitly approved the running local TypeScript viewer after reviewing
the result in windowed and full-canvas modes. The review covered graph
readability, hierarchy and labels, directed arrows, references/diagnostics,
evidence/source-location drilldown, navigation, Fit/reset, and the existing
export controls. No visual defect was reported.

## User stories addressed

The TypeScript child PRD has no standalone user-story IDs. This issue completes the public portions of `TS-FR-001` through `TS-FR-005`, `SC-TS-001` through `SC-TS-005`, and the shared analyzer → model → viewer/export journey.

## Scenario traceability and verification plan

| Source rule / scenario | Issue coverage | Planned evidence |
|---|---|---|
| Public selection/options; `SC-TS-001`, `SC-TS-005` | Manifest listing, explicit/automatic selection, option forwarding, and deterministic analysis output | CLI composition tests and repeated analysis JSON comparison |
| Shared model/viewer path; `SC-TS-002`, `SC-TS-003` | TypeScript hierarchy, relationships, references, diagnostics, evidence, projection, and source inspection through existing consumers | CLI/model/viewer HTTP integration fixture and canonical validation |
| Dynamic uncertainty; `SC-TS-004` | Partial result remains usable and visible without fabricated edges | Representative unresolved/dynamic fixture and scene/detail assertions |
| Artifact parity; `SC-TS-001`–`SC-TS-005` | JSON/HTML/SVG parity and stable output | Repeated CLI/export output checks, self-containment/semantic markers, and visual review |

## Verification surfaces

- Backend boundary: full Go tests, race, vet, build, staticcheck/lint where available, and focused TypeScript fixtures.
- Frontend integration: existing viewer hierarchy, relationship direction, references, diagnostics, source evidence, navigation, Fit/reset, and export controls.
- End-to-end: TypeScript repository → analysis → canonical model → viewer/HTML/SVG.
- Repository/OKF integrity: `git diff --check`, strict OKF validation, synchronized issue references, and untouched upstream boundary.
