# 026 — Clojure project and namespace discovery

## Issue Metadata

- Issue number: `026`
- Owning capability node: `/.okf/capabilities/analyze-source/clojure-compatibility.md`
- Artifact root: `docs/architecture/analyze-source/clojure-compatibility/`
- Issue file: `docs/agents/issues/done/20260827-026-clojure-project-and-namespace-discovery.md`
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
- `docs/architecture/analyze-source/clojure-compatibility/domain-glossary.md`
- `docs/architecture/analyze-source/clojure-compatibility/orchestration-status.md`

## What to build

Add and register the in-process Clojure-family analyzer foundation. It must
detect `deps.edn`, `project.clj`, and `shadow-cljs.edn`, resolve safe source
roots, discover `.clj`, `.cljs`, and `.cljc` files, parse namespace forms into
language-neutral namespace modules, and retain deterministic file/namespace
evidence. The analyzer must return usable module-only analysis through the
existing host/CLI path and must never evaluate forms, load namespaces, or run
project code.

## Acceptance criteria

- [x] The manifest is `org.archview.clojure`, language `clojure`, API-compatible, and declares the three project markers, the specified options, and the `polymorphic_metadata` capability; `arch-view analyzers` lists it deterministically.
- [x] Detection and boundary resolution prefer `deps.edn`, then `project.clj`, then `shadow-cljs.edn`, and source roots follow explicit options > supported configuration > `src` (with a safe repository-root fallback when no `src` exists).
- [x] Configured roots are repository-contained and symlink-safe; default exclusions include generated/build/cache/vendor/external output and test files remain opt-in.
- [x] Eligible `.clj`, `.cljs`, and `.cljc` files produce `clj:<namespace>` modules with structured hierarchy, flavor metadata, file evidence, stable ordering, and no file-derived guessed namespace.
- [x] Missing or malformed `ns` declarations retain file diagnostics and partial results without creating a fabricated module; unreadable or malformed files do not discard usable results from other files.
- [x] Explicit Clojure selection and automatic marker detection reach a valid `analysis-json` result without changing host, canonical model, viewer, or export code.
- [x] Focused tests prove marker/configuration precedence, source-root safety, flavor/test/exclusion behavior, namespace identity/evidence, deterministic repeated output, cancellation, and no target-code execution.

## Artifact sync required

- Application PRD: `none` — this adds an implementation of already-confirmed Clojure scope and does not change actors, workflows, MVP, or non-goals.
- Application architecture summary: `none` for this foundation slice — the common analyzer boundary remains unchanged; the consolidated adapter implementation sequence is recorded by issue 029.
- Owning capability node/artifacts: `required: /.okf/capabilities/analyze-source/clojure-compatibility.md; docs/architecture/analyze-source/clojure-compatibility/orchestration-status.md; docs/architecture/analyze-source/clojure-compatibility/implementation-slice.md`
- Issue registry: `required`; node `issues:` reference: `required`
- Reason/no-impact decision: delivery truth changes now; product scope, canonical schemas, cross-capability boundaries, and rendered UI do not change.

## Human review gate

None. This slice changes backend analysis behavior only and has no rendered UI/UX change.

## Blocked by

None - can start immediately.

## Artifact anchors

- PRD: `CL-FR-001`, `CL-FR-002`, `CL-FR-006`; source-root precedence and no-evaluation rules.
- Domain model: `ClojureProject`, `NamespaceObservation`, and the invariant that invalid files cannot receive a guessed namespace.
- Use cases: `DetectClojureProject`, `ResolveClojureSourceRoots`, `ParseNamespaceForms`, `EmitClojureAnalysisResult`.
- Contract: Clojure manifest, `source_roots`, `platform`, `include_tests`, and `exclude` options; `analysis-json` response parity.

## Acceptance scenarios addressed

- `SC-CL-001 — Discover namespaces`
- `SC-CL-005 — Do not evaluate forms`
- `SC-AS-001 — Analyze a resolvable project`
- `SC-AS-005 — Retain multi-file evidence`
- `SC-AS-006 — Enforce exclusions and safety`
- `SC-AS-007 — Cancellation is not completion`
- `SC-AS-008 — Deterministic repeat`

## Verification obligations

- Policy source: `/.okf/project.md`

| Scenario | Backend boundary | Frontend integration | End-to-end journey |
|---|---|---|---|
| `SC-CL-001` | `passed: internal/clojureanalyzer/analyzer_test.go::TestDiscoverNamespacesFlavorsTestsExclusionsAndEvidence` | `not-applicable` | `passed: cmd/arch-view/clojure_analyzer_test.go::TestClojureCLIExplicitAndAutomaticAnalysisJSON` |
| `SC-CL-005` | `passed: internal/clojureanalyzer/analyzer_test.go::TestDiscoverHonorsCancellationAndDoesNotEvaluateSource` | `not-applicable` | `passed: cmd/arch-view/clojure_analyzer_test.go::TestClojureCLIExplicitAndAutomaticAnalysisJSON` |
| `SC-AS-001` | `passed: internal/clojureanalyzer/analyzer_test.go::TestResolveProjectUsesBoundaryAndSourceRootPrecedence` | `not-applicable` | `passed: cmd/arch-view/clojure_analyzer_test.go::TestClojureCLIExplicitAndAutomaticAnalysisJSON` |
| `SC-AS-005` | `passed: internal/clojureanalyzer/analyzer_test.go::TestDiscoverNamespacesFlavorsTestsExclusionsAndEvidence` | `not-applicable` | `passed: cmd/arch-view/clojure_analyzer_test.go::TestClojureCLIExplicitAndAutomaticAnalysisJSON` |
| `SC-AS-006` | `passed: internal/clojureanalyzer/analyzer_test.go::TestResolveProjectRejectsEscapingAndSymlinkedRoots` | `not-applicable` | `passed: internal/clojureanalyzer/analyzer_test.go::TestDiscoverNamespacesFlavorsTestsExclusionsAndEvidence` |
| `SC-AS-007` | `passed: internal/clojureanalyzer/analyzer_test.go::TestDiscoverHonorsCancellationAndDoesNotEvaluateSource` | `not-applicable` | `not-applicable: backend cancellation policy has no separate viewer journey` |
| `SC-AS-008` | `passed: internal/clojureanalyzer/analyzer_test.go::TestDiscoverPlatformSelectionAndDeterminism` | `not-applicable` | `passed: cmd/arch-view/clojure_analyzer_test.go::TestClojureCLIExplicitAndAutomaticAnalysisJSON` |

## Closure evidence

- `go test ./internal/clojureanalyzer -count=1` passed.
- `go test ./cmd/arch-view ./internal/clojureanalyzer -count=1` passed.
- `git diff --check` passed before closeout.
