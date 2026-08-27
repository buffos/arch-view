# 029 — Clojure public integration and deterministic exports

## Issue Metadata

- Issue number: `029`
- Owning capability node: `/.okf/capabilities/analyze-source/clojure-compatibility.md`
- Artifact root: `docs/architecture/analyze-source/clojure-compatibility/`
- Issue file: `docs/agents/issues/done/20260827-029-clojure-public-integration-and-deterministic-exports.md`
- Category: `feature`
- Execution type: `AFK`
- Review gate: `none`
- Suggested state: `done`

## Parent Artifacts

- `docs/architecture/analyze-source/clojure-compatibility/prd.md`
- `docs/architecture/analyze-source/clojure-compatibility/canonical-domain-model.md`
- `docs/architecture/analyze-source/clojure-compatibility/canonical-use-cases.md`
- `docs/architecture/analyze-source/clojure-compatibility/canonical-api-cli-contract.md`
- `docs/architecture/analyze-source/clojure-compatibility/acceptance-scenarios.md`
- `docs/architecture/analyze-source/clojure-compatibility/readiness-review.md`
- `docs/prd.md`
- `docs/architecture/application-architecture-summary.md`

## What to build

Expose the completed Clojure adapter through the public CLI and prove the
unchanged shared product journey: Clojure repository → analysis JSON →
canonical model normalization/validation/projection → local viewer and
deterministic JSON/HTML/SVG artifacts. Add the `--platform` option to analyze
and project-backed open/reanalysis paths, preserve option precedence and
selection metadata, and verify that the viewer/export surfaces expose the
language-neutral namespace graph, evidence, references, diagnostics, and
confidence without Clojure-specific rendering branches.

## Acceptance criteria

- [x] `arch-view analyze` and project-backed `arch-view open` accept the Clojure contract's `--platform` option and repeatable source-root/exclude/test options with the existing host precedence and validation behavior.
- [x] Explicit `--language clojure`, explicit `--analyzer org.archview.clojure`, and automatic detection produce stable analyzer/boundary/run/fingerprint metadata; mixed marker selection follows the common host contract.
- [x] A representative `.clj`/`.cljs`/`.cljc` repository reaches `analysis-json`, `model normalize`, `model validate`, and hierarchy projection without model, viewer, layout, or export language switches.
- [x] The local viewer exposes namespace hierarchy, directed dependencies, references, source evidence, confidence, polymorphic metadata, and diagnostics through the existing list/details/source workflow.
- [x] The same Clojure model produces byte-stable canonical JSON, self-contained HTML, and SVG output across repeated runs; no Clojure runtime or source contents are embedded.
- [x] Existing Go/Python CLI, model, viewer, export, and deterministic output tests remain green; focused Clojure visible-journey tests cover the shared path and no-evaluation safety.
- [x] After the complete 026–029 scope is verified, the Clojure capability node advances from `specified` to `implemented` and all linked planning, application, architecture, and delivery records agree.

## Artifact sync required

- Application PRD: `required: docs/prd.md` — record the completed Clojure implementation slice and update implementation status/sequence without adding product scope.
- Application architecture summary: `required: docs/architecture/application-architecture-summary.md` — record the additional in-process adapter reaching the existing model/viewer/export path without host or renderer branching.
- Owning capability node/artifacts: `required: /.okf/capabilities/analyze-source.md; /.okf/capabilities/analyze-source/clojure-compatibility.md; docs/architecture/analyze-source/orchestration-status.md; docs/architecture/analyze-source/clojure-compatibility/orchestration-status.md; docs/architecture/analyze-source/clojure-compatibility/implementation-slice.md`
- Issue registry: `required`; node `issues:` reference: `required`
- Reason/no-impact decision: product behavior remains within the existing static architecture-discovery scope, but consolidated architecture/product implementation status and the Clojure capability maturity change and must be refreshed before closure.

## Human review gate

None. This slice exercises the existing viewer and export surfaces but does not change rendered UI/UX or navigation behavior; no separate visual review is required by the issue-slicing policy.

## Blocked by

- None - issues 026 through 028 are complete; can start immediately.

## Artifact anchors

- PRD: `CL-FR-001` through `CL-FR-006`, non-goals, and common analyzer workflow.
- Domain model: `ClojureProject` observations normalized through the language-neutral model.
- Use cases: `EmitClojureAnalysisResult` and the complete Clojure analyzer orchestration.
- Contract: manifest/options, CLI example, analysis response parity, and common status/diagnostic rules.

## Acceptance scenarios addressed

- `SC-CL-001 — Discover namespaces`
- `SC-CL-002 — Extract static dependencies`
- `SC-CL-003 — Preserve reader conditionals`
- `SC-CL-004 — Mark polymorphic forms`
- `SC-CL-005 — Do not evaluate forms`
- `SC-AS-001 — Analyze a resolvable project`
- `SC-AS-002 — Preserve a partial result`
- `SC-AS-004 — Honor explicit selection`
- `SC-AS-005 — Retain multi-file evidence`
- `SC-AS-006 — Enforce exclusions and safety`
- `SC-AS-008 — Deterministic repeat`

## Verification obligations

- Policy source: `/.okf/project.md`

| Scenario | Backend boundary | Frontend integration | End-to-end journey |
|---|---|---|---|
| `SC-CL-001` | `passed: internal/clojureanalyzer/analyzer_test.go::TestDiscoverNamespacesFlavorsTestsExclusionsAndEvidence` | `not-applicable` | `passed: cmd/arch-view/clojure_visible_journey_test.go::TestClojureSharedModelViewerAndDeterministicExports` |
| `SC-CL-002` | `passed: internal/clojureanalyzer/dependencies_test.go::TestStaticDependencyKindsResolutionMetadataAndEvidence` | `not-applicable` | `passed: cmd/arch-view/clojure_visible_journey_test.go::TestClojureSharedModelViewerAndDeterministicExports` |
| `SC-CL-003` | `passed: internal/clojureanalyzer/safety_test.go::TestReaderConditionalsSelectPlatformsAndRetainMetadata` | `not-applicable` | `passed: cmd/arch-view/clojure_visible_journey_test.go::TestClojureSharedModelViewerAndDeterministicExports` |
| `SC-CL-004` | `passed: internal/clojureanalyzer/safety_test.go::TestPolymorphicMetadataAndDynamicLoadingAreSafe` | `not-applicable` | `passed: cmd/arch-view/clojure_visible_journey_test.go::TestClojureSharedModelViewerAndDeterministicExports` |
| `SC-CL-005` | `passed: internal/clojureanalyzer/safety_test.go::TestPolymorphicMetadataAndDynamicLoadingAreSafe` | `not-applicable` | `passed: cmd/arch-view/clojure_visible_journey_test.go::TestClojureSharedModelViewerAndDeterministicExports` |
| `SC-AS-001` | `passed: internal/clojureanalyzer/analyzer_test.go::TestResolveProjectUsesBoundaryAndSourceRootPrecedence` | `passed: cmd/arch-view/clojure_visible_journey_test.go::TestClojureSharedModelViewerAndDeterministicExports` | `passed: cmd/arch-view/clojure_visible_journey_test.go::TestClojureCLIOptionsAndSharedVisibleJourney` |
| `SC-AS-002` | `passed: internal/clojureanalyzer/safety_test.go::TestSplicedAndMalformedReaderConditionalsRemainStatic` | `passed: cmd/arch-view/clojure_visible_journey_test.go::TestClojureSharedModelViewerAndDeterministicExports` | `passed: cmd/arch-view/clojure_visible_journey_test.go::TestClojureSharedModelViewerAndDeterministicExports` |
| `SC-AS-004` | `passed: internal/clojureanalyzer/dependencies_test.go::TestStaticDependencyKindsResolutionMetadataAndEvidence` | `passed: cmd/arch-view/clojure_visible_journey_test.go::TestClojureCLIOptionsAndSharedVisibleJourney` | `passed: cmd/arch-view/clojure_visible_journey_test.go::TestClojureCLIOptionsAndSharedVisibleJourney` |
| `SC-AS-005` | `passed: internal/clojureanalyzer/dependencies_test.go::TestStaticDependencyKindsResolutionMetadataAndEvidence` | `passed: cmd/arch-view/clojure_visible_journey_test.go::TestClojureSharedModelViewerAndDeterministicExports` | `passed: cmd/arch-view/clojure_visible_journey_test.go::TestClojureSharedModelViewerAndDeterministicExports` |
| `SC-AS-006` | `passed: internal/clojureanalyzer/analyzer_test.go::TestResolveProjectRejectsEscapingAndSymlinkedRoots` | `passed: cmd/arch-view/clojure_visible_journey_test.go::TestClojureSharedModelViewerAndDeterministicExports` | `passed: cmd/arch-view/clojure_visible_journey_test.go::TestClojureSharedModelViewerAndDeterministicExports` |
| `SC-AS-008` | `passed: internal/clojureanalyzer/dependencies_test.go::TestStaticDependenciesAreDeterministicAndCancellationIsHonored` | `passed: cmd/arch-view/clojure_visible_journey_test.go::TestClojureSharedModelViewerAndDeterministicExports` | `passed: cmd/arch-view/clojure_visible_journey_test.go::TestClojureSharedModelViewerAndDeterministicExports` |

## Closure evidence

- `go test ./... -count=1` passed.
- Clojure CLI, canonical model, hierarchy projection, viewer, source inspection, JSON/HTML/SVG export, and repeated-output checks passed.
- `git diff --check` passed before closeout.
- Root application PRD and architecture summary were refreshed to record the completed in-process Clojure adapter and the next TypeScript/Rust frontier.
