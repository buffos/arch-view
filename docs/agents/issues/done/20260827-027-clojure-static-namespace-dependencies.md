# 027 — Clojure static namespace dependencies

## Issue Metadata

- Issue number: `027`
- Owning capability node: `/.okf/capabilities/analyze-source/clojure-compatibility.md`
- Artifact root: `docs/architecture/analyze-source/clojure-compatibility/`
- Issue file: `docs/agents/issues/done/20260827-027-clojure-static-namespace-dependencies.md`
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

## What to build

Extend the discovered Clojure namespace modules with static dependency
observations from `:require`, `:use`, and macro dependency clauses. Resolve
only namespace targets proven by the selected source roots; retain unresolved
or non-local targets as references with confidence and diagnostics. Preserve
aliases, referred symbols, dependency kind, source locations, and deterministic
merging through the common `depends_on` relationship contract.

## Acceptance criteria

- [x] `:require` and `:use` clauses create typed static observations, including namespace aliases and referred symbols, with source-reference evidence.
- [x] `:require-macros`, `:use-macros`, `:refer-macros`, and `:include-macros true` are retained as `macro` dependencies without introducing a renderer-specific relation type.
- [x] Proven project-local namespaces become directed `depends_on` relationships to `clj:<namespace>` modules; duplicate observations merge source IDs and metadata deterministically.
- [x] Unresolved, standard, and other non-local namespaces become references rather than fabricated modules; unresolved targets emit recoverable diagnostics and produce `partial` status.
- [x] Relationship/reference metadata preserves dependency kind, aliases, referred symbols, platform information supplied by the parser, resolution scope, and confidence.
- [x] Static dependency parsing remains read-only and never evaluates quoted forms, requires namespaces, expands macros, or consults the target classpath.
- [x] Focused tests cover require/use/macro syntax, local resolution, unresolved references, source locations, merged evidence, partial status, deterministic output, and cancellation.

## Artifact sync required

- Application PRD: `none` — dependency evidence is already within the confirmed static-analysis workflow and does not change product scope.
- Application architecture summary: `none` — this remains inside the language-adapter port and common observation schema; no host/model/viewer boundary changes.
- Owning capability node/artifacts: `required: /.okf/capabilities/analyze-source/clojure-compatibility.md; docs/architecture/analyze-source/clojure-compatibility/orchestration-status.md; docs/architecture/analyze-source/clojure-compatibility/implementation-slice.md`
- Issue registry: `required`; node `issues:` reference: `required`
- Reason/no-impact decision: delivery and capability implementation evidence change; product, architecture boundaries, and rendered UI remain unchanged.

## Human review gate

None. This slice changes backend analysis behavior only and has no rendered UI/UX change.

## Blocked by

- None - issue 026 is complete; can start immediately.

## Artifact anchors

- PRD: `CL-FR-003` and `CL-FR-006`.
- Domain model: `NamespaceDependencyObservation`, typed dependency kinds, confidence, aliases, referred symbols, and static/no-evaluation invariants.
- Use cases: `ExtractStaticNamespaceDependencies`, `EmitClojureAnalysisResult`.
- Contract: common `depends_on` observations, reference/evidence/diagnostic fields, and partial-result rules.

## Acceptance scenarios addressed

- `SC-CL-002 — Extract static dependencies`
- `SC-CL-005 — Do not evaluate forms`
- `SC-AS-002 — Preserve a partial result`
- `SC-AS-004 — Honor explicit selection`
- `SC-AS-005 — Retain multi-file evidence`
- `SC-AS-008 — Deterministic repeat`

## Verification obligations

- Policy source: `/.okf/project.md`

| Scenario | Backend boundary | Frontend integration | End-to-end journey |
|---|---|---|---|
| `SC-CL-002` | `passed: internal/clojureanalyzer/dependencies_test.go::TestStaticDependencyKindsResolutionMetadataAndEvidence` | `not-applicable` | `passed: cmd/arch-view/clojure_analyzer_test.go::TestClojureCLIExplicitAndAutomaticAnalysisJSON` |
| `SC-CL-005` | `passed: internal/clojureanalyzer/dependencies_test.go::TestStaticDependenciesAreDeterministicAndCancellationIsHonored` | `not-applicable` | `passed: internal/clojureanalyzer/dependencies_test.go::TestStaticDependencyKindsResolutionMetadataAndEvidence` |
| `SC-AS-002` | `passed: internal/clojureanalyzer/dependencies_test.go::TestStaticDependencyKindsResolutionMetadataAndEvidence` | `not-applicable` | `not-applicable: no new UI behavior` |
| `SC-AS-004` | `passed: internal/clojureanalyzer/dependencies_test.go::TestStaticDependencyKindsResolutionMetadataAndEvidence` | `not-applicable` | `passed: cmd/arch-view/clojure_analyzer_test.go::TestClojureCLIExplicitAndAutomaticAnalysisJSON` |
| `SC-AS-005` | `passed: internal/clojureanalyzer/dependencies_test.go::TestStaticDependencyKindsResolutionMetadataAndEvidence` | `not-applicable` | `passed: internal/clojureanalyzer/dependencies_test.go::TestStaticDependencyKindsResolutionMetadataAndEvidence` |
| `SC-AS-008` | `passed: internal/clojureanalyzer/dependencies_test.go::TestStaticDependenciesAreDeterministicAndCancellationIsHonored` | `not-applicable` | `passed: cmd/arch-view/clojure_analyzer_test.go::TestClojureCLIExplicitAndAutomaticAnalysisJSON` |

## Closure evidence

- `go test ./internal/clojureanalyzer -count=1` passed.
- `analysis.ValidateAnalysisResult` passed for resolved, standard-library, and unresolved relationship/reference targets.
- `git diff --check` passed before closeout.
