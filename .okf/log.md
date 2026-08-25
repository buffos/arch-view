# Update Log

## 2026-08-26

* **Documentation**: Added the temporary root README as a contributor-facing viewer guide covering layers, aggregation, diagnostics, tags, identity/confidence, and layout ownership.
* **Implementation**: Added explicit internal-relationship summaries for non-cycle group self-loops, preserved canonical contributor/evidence IDs, and separated stable node identity from relationship confidence in the scene and inspection UI.
* **Implementation**: Issue 006 now serves pinned elkjs 0.12.0 and its worker locally, runs the ELK layered layout, consumes returned edge sections/bend points in the SVG renderer, and retains the deterministic layer layout as fallback.
* **Planning**: Follow-up issue 006 remains under Explore and inspect architecture and is awaiting visual review of the new routing and summaries; the canonical model boundary is unchanged.
* **Bug fix**: Made the SVG edge hit-area stroke-only and transparent. The previous default SVG fill painted black closed shapes beneath routed edges; the fix was verified in aggregated and expanded browser views.
* **Closeout**: Archived issue 003 after the user's explicit visual approval, removed its active registry row, updated the dated OKF issue reference, and unblocked issues 004 and 005. Issue 006 remains active for its separate visual review.
* **Verification**: Viewer scene tests, full Go tests, race tests, vet, build, JavaScript syntax check, and diff checks pass. The baseline visual review is approved; issue 006 still requires visual review of the semantic refinement.

## 2026-08-25

* **Implementation**: Completed issue 003's local-first viewer refinement. The scene contract now supports hidden, aggregated, and expanded reference visibility, scope-aware reference nodes/relations, confidence-aware import details, and reference summaries without mutating the canonical model.
* **Browser surface**: Added the reference visibility control, boundary summary, import/details list, readable bounded graph surface, and accessible scope/confidence presentation.
* **Verification**: `go test ./... -count=1`, `go test -race ./...`, `go vet ./...`, `go build ./...`, `golangci-lint run ./...`, `staticcheck ./...`, and `node --check internal/viewer/web/app.js` passed. The issue is now `awaiting-human-review` for visual approval at the running local session.

## 2026-08-25

* **Brownfield refinement**: Visual review of issue 003 found that expanding every non-local reference overwhelms the local architecture overview and that the baseline layout is not yet navigable.
* **Routing**: Kept the work inside the specified Explore and inspect architecture capability; issue 003 owns the local-first overview and reference visibility baseline, issue 004 owns imports/evidence inspection plus navigation and session layout, and issue 005 owns export parity. No new capability node or shared-concern promotion was needed.
* **Specification sync**: Refreshed the viewer and export PRDs, domain/use-case models, contracts, scenarios, glossary, gap/readiness/orchestration artifacts, and application synthesis with hidden/aggregated/expanded reference visibility, scope/confidence distinction, imports list behavior, and session-owned layout rules.
* **Delivery**: Returned issue 003 from `awaiting-human-review` to `ready-for-agent`; issues 004 and 005 remain blocked by it.

## 2026-08-25

* **Completion**: Archived issue 001 as docs/agents/issues/done/20260825-001-analyzer-host-and-go-project-selection.md.
* **Implementation**: Added the in-process analyzer registry, manifest/API compatibility checks, deterministic listing, explicit/automatic selection, option precedence, cancellation/error containment, result validation, Go module/workspace selection, and the analyzers/analyze CLI boundary.
* **Verification**: go test ./..., go test -race ./..., go vet ./..., and CLI smoke checks passed.
* **Progress**: Issue 002 is now unblocked; the plugin runtime and Go analysis nodes remain specified because their scoped implementation work is not exhausted.

## 2026-08-25

* **Frontier selection**: Selected the specified Go analysis child as the first implementation frontier because it is the first supported language and reaches the intended visible product journey.
* **Slice creation**: Created the vertical Go repository to visible architecture view slice, spanning analyzer host, Go package/import analysis, canonical model, local viewer, evidence, and deterministic export.
* **Delivery**: Created ready-for-agent issues 001 through 005 with dependency order, ownership, acceptance criteria, contract/scenario traceability, and visual-review gates for viewer/export work.
* **Artifact sync**: Linked the implementation slice and issue paths from the affected .okf capability nodes and updated orchestration statuses. Product and application-architecture behavior was unchanged.

## 2026-08-25

* **State change**: Moved all 10 bounded capability nodes to `specified` after the architecture specification pipeline produced PRDs, glossaries, canonical domain/use-case models, contracts, acceptance scenarios, and readiness reviews.
* **Update**: Specified the shared analysis/model contracts, Go/Python/TypeScript/Rust/Clojure adapters, plugin runtime, local web viewer, and JSON/HTML/SVG export behavior.
* **Progress**: State totals are now 0 `foggy`, 0 `bounded`, 10 `specified`, and 0 `implemented`.
* **No impact**: No delivery issues or ADRs were created. Issue slicing remains downstream of the synchronized application synthesis gate.

## 2026-08-25

* **State change**: Moved [Explore and inspect architecture](/capabilities/explore-architecture.md) and [Export and automate](/capabilities/export-and-automate.md) from `foggy` to `bounded` after reviewing their visual, interaction, export, and automation questions together.
* **Update**: Confirmed a local web surface, renderer-neutral view contract, progressive disclosure, accessible evidence inspection, versioned JSON, deterministic HTML/SVG artifacts, and CI-safe partial/fatal status behavior.
* **Progress**: State totals are now 0 `foggy`, 10 `bounded`, 0 `specified`, and 0 `implemented`.
* **No impact**: No delivery issues or ADRs were created; exact capability specifications remain gated.

## 2026-08-25

* **State change**: Moved the analyzer/plugin child territories under [Analyze source code](/capabilities/analyze-source.md) from `foggy` to `bounded`.
* **Update**: Recorded staged plugin runtime rules and bounded decisions for Go, Python, TypeScript, Rust, and Clojure analysis.
* **Progress**: State totals are now 2 `foggy`, 8 `bounded`, 0 `specified`, and 0 `implemented`.
* **Pause point**: Exploration, graphics/user visualization, and export behavior remain foggy by design until those questions are reviewed together.

## 2026-08-25

* **State change**: Moved [Generate architecture models](/capabilities/generate-models.md) from `foggy` to `bounded` after confirming separate structural hierarchy, stable module identity, typed relationships, evidence/provenance, non-local references, cycle preservation, derived layers, and deterministic normalization.
* **Update**: Added discovery notes and a requirements gap analysis for the bounded model capability.
* **No impact**: No delivery issues or ADRs were created; exact schema and algorithm specification remain gated.

## 2026-08-25

* **State change**: Moved [Analyze source code](/capabilities/analyze-source.md) from `foggy` to `bounded` after clarifying its purpose, boundary, inputs, outputs, analyzer contract, and safety rules.
* **Update**: Added discovery notes and a requirements gap analysis for the bounded capability.
* **Update**: Refreshed the application PRD and architecture summary with the confirmed analyzer decisions.
* **Progress**: State totals are 9 `foggy`, 1 `bounded`, 0 `specified`, and 0 `implemented`.
* **No impact**: No delivery issues or ADRs were created. Exact specification and issue slicing remain gated.

## 2026-08-25

* **Update**: Added Rust analysis as a foggy child capability under [Analyze source code](/capabilities/analyze-source.md).
* **Update**: Synchronized the application PRD, architecture summary, orchestration status, and project scope with the Rust roadmap addition.
* **No impact**: No delivery issues or ADRs were created; the new language territory remains unimplemented.

## 2026-08-25

* **Creation**: Created the confirmed Arch View planning graph with one project concept, four top-level capabilities, and five analyzer-support child capabilities.
* **Update**: Set the root verification policy to `when-supported` with justified deferrals required.
* **Update**: Created the initial application PRD and application architecture summary.
* **No impact**: No implementation issues, ADRs, or delivery references were created because all capability nodes remain `foggy`.

## 2026-08-25

* **Completion**: Archived issue 002, [Go package/import analysis and canonical model pipeline](../docs/agents/issues/done/20260825-002-go-package-import-model-pipeline.md), after implementing deterministic Go observations, canonical model normalization/validation, graph derivations, hierarchy projection, and headless CLI access.
* **Artifact sync**: Updated the analyze-source and generate-models capability issue references, implementation/orchestration status, active issue blockers, and acceptance traceability. Product and application-architecture truth remain unchanged because the analyzer-to-model boundary is unchanged.
* **Verification**: Backend and end-to-end obligations passed where supported; frontend integration is not applicable to this backend/model issue. The external reference folder remains untouched.
