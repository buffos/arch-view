# 003 — Local web top-level architecture view

Execution type: AFK
Review gate: visual-review
Status: done

## Parent PRD

docs/architecture/explore-architecture/prd.md

## What to build

Build the first visible local web experience over a validated canonical model.

- Implement arch-view open for a model file and for a Go project through the existing analysis path.
- Serve a local browser application with the renderer-neutral scene contract.
- Render the top-level hierarchy, modules, directed relationships, layer labels, cycle indicators, diagnostics, confidence states, and aggregate counts.
- Start with a local-first overview that remains understandable for a small representative Go repository. Standard-library, external, unresolved, and dynamic references remain in the model but are hidden or summarized by default rather than rendered as one node per import.
- Provide a reference-visibility policy (`hidden`, `aggregated`, `expanded`) and expose individual imports through the list/details path without removing canonical evidence.
- Use a readable baseline layout with fit-to-view-friendly geometry and clear edge direction. Advanced pan/zoom, navigation, and session layout persistence continue in issue 004.
- Keep viewer session state separate from the canonical model and keep source access read-only.

Drill-down, source excerpts, and richer evidence interactions are completed in issue 004. Durable artifact generation is completed in issue 005.

## Acceptance criteria

- [x] arch-view open --model serves a local browser session and loads the canonical model without source parsing.
- [x] arch-view open --project runs the Go path from issue 002 and opens the resulting top-level architecture view.
- [x] The top-level scene contains hierarchy-aware visible nodes, directed relationship indicators, layers, cycles, diagnostics, confidence, reference scope, and stable IDs.
- [x] The first view is aggregated at the top level, prioritizes local modules/groups, and does not require a renderer to invent semantic relationships or hierarchy.
- [x] A list/details mode exposes the same visible facts as the graphic view, including individual imports and reference scopes, and is keyboard reachable.
- [x] The local host is read-only, does not execute target code, and does not expose source paths outside the analyzed root.
- [x] Standard-library, external, unresolved, and dynamic references are distinguishable, and resolved standard-library relationships are not presented as `unknown confidence`.
- [x] A visual review confirms legible labels, edge direction, cycle/diagnostic styling, local-first density, and a useful result on a representative Go repository.

## Artifact sync required

- Application PRD: required: docs/prd.md — the existing progressive-disclosure journey now has an explicit local-first overview and import-inspection policy.
- Application architecture summary: required: docs/architecture/application-architecture-summary.md — reference retention remains model-owned while visibility/filtering is explicitly viewer-owned and shared with exports.
- Owning capability node/artifacts: required: .okf/capabilities/explore-architecture.md; docs/architecture/explore-architecture/orchestration-status.md; docs/architecture/analyze-source/go-analysis/implementation-slice.md.
- Issue registry: required; node issues reference required: .okf/capabilities/explore-architecture.md.
- Reason/no-impact decision: no new capability or boundary is introduced; the synchronized product and architecture summaries record a refinement of the existing progressive-disclosure journey.

## Blocked by

—

## User stories addressed

- US-EX-001
- US-GM-001
- US-GM-002

## Contract and scenario trace

- Contract: docs/architecture/explore-architecture/canonical-api-cli-contract.md; docs/architecture/generate-models/canonical-api-cli-contract.md
- Scenarios: SC-EX-001, SC-EX-005, SC-EX-006, SC-EX-007, SC-EX-009

## Scenario traceability

| Source rule or use case | Acceptance scenario | Issue criterion | Verification artifact | Closure evidence |
| --- | --- | --- | --- | --- |
| Open a validated model through the local web boundary | SC-EX-001 | 1 | Viewer HTTP/model/projection tests | `go test ./... -count=1`: `TestServerServesReadOnlyModelSceneAndBrowserAssets` passed; model and projection endpoints returned the validated model and scene. |
| Run Go analysis before opening a project view | SC-EX-001 | 2 | CLI smoke test against this repository | `go run ./cmd/arch-view open --project . --language go --port 0` served `GET /` and `GET /v1/models/{id}/projection`; the revised default scene returned `reference_visibility=hidden` with 2 local visible nodes and 32 reference details. |
| Preserve structural hierarchy, directed edges, layers, cycles, diagnostics, confidence, reference scope, and contributor identity | SC-EX-005, SC-EX-006, SC-EX-009 | 3 | Scene fixture with aggregated Go-like package graph plus reference classification | `go test ./... -count=1`: `TestBuildScenePreservesTopLevelSemanticsAndAggregation` and `TestBuildSceneReferenceVisibilityModes` passed; local aggregation, reference scope, and confidence are asserted. |
| Begin with local hierarchy aggregation, reference-boundary policy, and semantic aggregation outside the renderer | SC-EX-006, SC-EX-009 | 4 | Renderer-neutral scene policy and representative visual review | Backend and live endpoint checks passed for hidden/aggregated/expanded policies; the user explicitly approved the issue 003 baseline visual review. |
| Expose the same facts through accessible list/details controls, including imports | SC-EX-007, SC-EX-009 | 5 | Embedded HTML/JS boundary test and keyboard/list review | `TestServerServesReadOnlyModelSceneAndBrowserAssets` asserts the visibility control/import list assets; `node --check internal/viewer/web/app.js` passed; the user approved the baseline visual/list review. |
| Keep the local surface read-only and model evidence repository-relative | SC-EX-005, SC-EX-007 | 6 | HTTP method, model validation, and evidence-link tests | `go test ./... -count=1`: read methods are the only accepted methods, evidence links are marked read-only, and the validated model contains repository-relative source paths; no source-serving route is exposed in issue 003. |
| Distinguish reference scope from relationship confidence | SC-EX-005, SC-EX-009 | 7 | Scene/reference classification test and visual review | `TestBuildSceneReferenceVisibilityModes` passed; the live expanded projection classified 32 standard-library references as `high` confidence, and no reference defaulted to `unknown confidence`. |
| Produce a readable local-first overview on a representative Go repository | SC-EX-001, SC-EX-006, SC-EX-009 | 8 | Manual visual review after the revised implementation | User explicitly approved the issue 003 baseline visual review; later ELK and semantic presentation refinement remains tracked by issue 006. |

## Verification surfaces

- Backend boundary (`when-supported`): implemented by scene/server tests, `go test -race ./...`, `go vet ./...`, `golangci-lint run ./...`, and `staticcheck ./...`.
- Frontend integration (`when-supported`): embedded HTML/CSS/JS is served and syntax-checked; the user-approved issue 003 baseline visual review covers the local-first reference policy and layout.
- End-to-end (`when-supported`): `arch-view open --project . --language go --port 0` and `arch-view open --model .arch-view-model.tmp.json --port 0` were smoke-tested through the local HTTP root and projection endpoint; temporary model artifacts were removed afterward.
- Repository/OKF integrity: `git diff --check` and the strict OKF validator must pass before closeout.

## Review handoff

The implementation provides the local-first reference policy, import list path, semantic confidence labels, and a bounded readable baseline. The user explicitly approved the issue 003 visual-review gate in the current task. Issue 006 carries the later semantic-summary, ELK-routing, and visual-refinement work and remains independently awaiting review.
