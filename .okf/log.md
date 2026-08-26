# Update Log

## 2026-08-26

* **Architecture refactor**: Implemented issues 010–015 while preserving the
  canonical model, scene, HTTP, CLI, configuration, and export contracts.
  Routing is now renderer-neutral with deterministic orthogonal/manual and
  existing self-loop behavior; spline segments are reserved but not rendered.
* **Composition**: Split the live viewer into native ES modules, kept
  `app.js` as a small entrypoint, added embedded esbuild bundling with external
  import rejection for self-contained HTML, and decomposed the Go viewer host
  and CLI into focused files.
* **Capabilities**: Isolated scene projection, the Go scanner/import/
  observation pipeline, the ELK option-handler registry, and canonical model
  normalization/validation into explicit packages. Issue 009's node/edge ELK
  target mapping is explicitly deferred by the user.
* **Verification**: `go test ./... -count=1`, race tests, vet, build,
  staticcheck, golangci-lint, JavaScript syntax/pure-module tests, export
  self-containment/determinism tests, and `git diff --check` pass. Issues
  010–015 are recorded as awaiting human/repository review handoffs;
  `external/` remains ignored, read-only, and untouched.

## 2026-08-26

* **Closeout**: Archived issue 008 after the user's explicit approval of the parent-level ELK option tranche in windowed and full-canvas views, including representative settings, diagnostics, fallbacks, and resulting layouts. Removed its active registry row, updated the owning capability and implementation-slice references, and unblocked issue 009 without processing it; `external/` remains untouched.

* **Implementation**: Issue 008 expanded the editable ELK parent-level tranche with typed metadata and validation for aspect ratio, layered spacing, layering, cycle breaking, crossing minimization, node placement, and connected-component compaction. The browser request builder now forwards only catalogued editable `PARENTS` options to the root graph; node/edge-targeted options remain reserved for issue 009.
* **Correction**: Pinned catalog inspection confirmed `org.eclipse.elk.alignment` is node-targeted and that the canonical parent spacing key is `org.eclipse.elk.layered.spacing.baseValue`; neither is silently treated as a root option.
* **Verification**: Focused Go/JavaScript tests, syntax checks, repository gates, and OKF validation pass. Issue 008 is awaiting visual review of representative settings in windowed and full-canvas views; `external/` remains untouched.

* **Closeout**: Archived issue 005 at [the dated delivery record](../docs/agents/issues/done/20260826-005-deterministic-json-html-svg-export.md) after the user's explicit approval of HTML/SVG parity with the local viewer; required capability, orchestration, and implementation-slice references now point to the completed issue, and issue 007 remains the next ready-for-agent frontier.

* **Planning**: Added the user-confirmed ELK layout-settings and project-configuration extension as issue 007 under the existing Explore and inspect architecture capability; no new capability or shared-concern node was needed because the behavior remains viewer-owned presentation policy.
* **Design**: Chose versioned `.archview.json` as the v1 project configuration file. Discovery walks from the selected target directory through its ancestors toward the filesystem root, the nearest file wins without merging, and an invalid nearest file is surfaced instead of bypassed; the configuration changes layout presentation only.
* **Artifact sync**: Refreshed the Explore exact-spec set, application PRD, application architecture summary, capability issue references, registry, and orchestration records. Analyzer options, canonical model semantics, and export behavior are explicitly unchanged by issue 007.
* **Delivery**: Created issue 007 as `ready-for-agent`, with settings-catalog, validation, apply/reset, active-file `Save`, explicit custom-folder `Save As`, configuration discovery, and model-only-session acceptance coverage.
* **Clarification**: Ordinary `Save` overwrites the exact discovered configuration path and never creates a new project-root copy. When no file is active, it requires `Save As`; `Save As` is the only operation that accepts a custom destination folder and makes the written file active for the current session.
* **Implementation**: Issue 007 now serves the complete pinned ELK catalog (11 algorithms, 8 categories, 235 options), validates typed/applicable profiles, runs the selected ELK adapter with returned routes, and keeps deterministic fallback behavior visible.
* **Persistence**: Added nearest-ancestor `.archview.json` resolution with invalid-nearest diagnostics, session Apply/Reset, exact active-file Save, and explicitly confirmed custom-folder Save As. Model-only sessions remain non-persistent.
* **Review gate**: Automated issue 007 checks pass; the issue is `awaiting-human-review` for the settings origin/error states, apply/reset workflow, and graph behavior in windowed and full-canvas views.

## 2026-08-26

* **Implementation**: Issue 005 now renders validated models as canonical deterministic JSON, self-contained interactive HTML, and script-free accessible SVG. The CLI supports direct export and analyze-to-export with reference visibility/scope controls, source-embedding rejection, partial status, atomic writes, and overwrite protection.
* **Verification**: Issue 005 passed `go test ./... -count=1`, `go test -race ./...`, `go vet ./...`, `go build ./...`, `staticcheck ./...`, `golangci-lint run`, `node --check internal/viewer/web/app.js`, and `git diff --check`. HTTP export parity is not applicable because the current product has no HTTP export endpoint.
* **Delivery**: Issue 005 is awaiting explicit visual review of HTML/SVG parity with the approved local viewer; no issue closeout or commit has been performed.

* **Review**: Completed the strict code-review loop for issue 004. The final pass found no actionable P0–P2 findings after resolving stale reanalysis scene loading, invalid failed-revision replacement, invalid explicit source line ranges, and source-file cleanup.
* **Closeout**: Archived issue 004 after the user's explicit visual approval of navigation, evidence, source inspection, full-canvas spacing, viewport controls, and stable manual routing. Removed its active registry row and updated the dated issue references; issue 005 remains the active unblocked frontier.
* **Verification**: Final delivery gates passed: `go test ./... -count=1`, `go test -race ./...`, `go vet ./...`, `go build ./...`, `staticcheck ./...`, `golangci-lint run`, `node --check internal/viewer/web/app.js`, and `git diff --check`.
* **Documentation**: Added the temporary root README as a contributor-facing viewer guide covering layers, aggregation, diagnostics, tags, identity/confidence, and layout ownership.
* **Implementation**: Added explicit internal-relationship summaries for non-cycle group self-loops, preserved canonical contributor/evidence IDs, and separated stable node identity from relationship confidence in the scene and inspection UI.
* **Implementation**: Issue 006 now serves pinned elkjs 0.12.0 and its worker locally, runs the ELK layered layout, consumes returned edge sections/bend points in the SVG renderer, and retains the deterministic layer layout as fallback.
* **Planning**: Follow-up issue 006 remains under Explore and inspect architecture and is awaiting visual review of the new routing and summaries; the canonical model boundary is unchanged.
* **Bug fix**: Made the SVG edge hit-area stroke-only and transparent. The previous default SVG fill painted black closed shapes beneath routed edges; the fix was verified in aggregated and expanded browser views.
* **Closeout**: Archived issue 003 after the user's explicit visual approval, removed its active registry row, updated the dated OKF issue reference, and unblocked issues 004 and 005. Issue 006 remains active for its separate visual review.
* **Closeout**: Archived issue 006 after the user's explicit visual approval of the semantic summary, ELK routing, transparent edge hit areas, and temporary canonical-cycle visual fixture. Removed its active registry row and updated the dated OKF issue reference; issues 004 and 005 remain the active unblocked frontiers.
* **Verification**: Viewer scene tests, full Go tests, race tests, vet, build, JavaScript syntax check, and diff checks pass. The baseline and issue 006 visual reviews are approved.

## 2026-08-26

* **Planning**: Added issue 008 for a bounded parent-level ELK option-support tranche and issue 009 for target-aware node/edge option mapping under [Explore and inspect architecture](/capabilities/explore-architecture.md). Issue 008 is blocked by 007 and issue 009 is blocked by 008.
* **No impact**: The follow-up issues remain inside the existing viewer capability and preserve the current `arch-view.config/v1` boundary. Advanced ports, labels, junctions, and compound-graph geometry remain a later specification frontier because they would change the scene/renderer contract.
* **Review**: Completed the strict code-review loop for issue 007. The final pass found no actionable P0–P2 findings after adding required-algorithm validation, enforcing layout-request size limits, and serializing configuration writes with session updates. Repository checks and JavaScript syntax checks pass.
* **Closeout**: Archived issue 007 after the user's explicit visual approval of the settings surface, origin/error states, apply/reset workflow, and windowed/full-canvas graph behavior. Removed its active registry row, unblocked issue 008, and synchronized the capability, implementation-slice, index, and orchestration references. `external/` remains untouched.

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
