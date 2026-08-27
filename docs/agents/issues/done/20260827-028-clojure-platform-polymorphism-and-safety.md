# 028 — Clojure platform, polymorphism, and safety metadata

## Issue Metadata

- Issue number: `028`
- Owning capability node: `/.okf/capabilities/analyze-source/clojure-compatibility.md`
- Artifact root: `docs/architecture/analyze-source/clojure-compatibility/`
- Issue file: `docs/agents/issues/done/20260827-028-clojure-platform-polymorphism-and-safety.md`
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

Complete the Clojure-family adapter's language-specific static behavior. Add
platform selection for `.cljc` reader conditionals, preserve selected and
combined platform metadata, mark statically recognized `defprotocol` and
`defmulti` forms as polymorphic metadata, and surface dynamic loading or
malformed reader forms as recoverable diagnostics. No language-specific
classification may leak into the core relationship type, and no target form
may be evaluated.

## Acceptance criteria

- [x] `platform=clj`, `platform=cljs`, and `platform=both` select the applicable `.clj`, `.cljs`, and `.cljc` observations without treating either `.cljc` branch as runtime-tested fact.
- [x] `#?(:clj ... :cljs ...)`, spliced conditionals, and `:default` branches preserve platform metadata; `platform=both` retains all selected branch evidence and merges equivalent targets deterministically.
- [x] Unsupported or malformed reader-condition selectors produce recoverable diagnostics and do not create guessed dependencies or modules.
- [x] Statically recognized `defprotocol` and `defmulti` forms add `polymorphic` module metadata and source evidence while leaving the core relation type generic.
- [x] Dynamic loading/evaluation forms such as `require`, `load-file`, `load-string`, and `eval` outside static namespace declarations are surfaced as dynamic references/diagnostics and never executed.
- [x] Malformed source forms still return usable namespace/module/dependency observations from other parseable portions, with `partial` status and source locations.
- [x] Focused tests cover platform filtering, reader conditionals, polymorphic tags/evidence, dynamic loading, malformed recovery, no evaluation, deterministic output, and cancellation.

## Artifact sync required

- Application PRD: `none` — platform and metadata behavior are already committed Clojure compatibility scope and do not change product actors or workflows.
- Application architecture summary: `none` — reader/platform and metadata logic remain owned by the Clojure adapter behind the existing contract.
- Owning capability node/artifacts: `required: /.okf/capabilities/analyze-source/clojure-compatibility.md; docs/architecture/analyze-source/clojure-compatibility/orchestration-status.md; docs/architecture/analyze-source/clojure-compatibility/implementation-slice.md`
- Issue registry: `required`; node `issues:` reference: `required`
- Reason/no-impact decision: capability implementation evidence changes; canonical model types, product scope, architecture boundaries, and rendered UI remain unchanged.

## Human review gate

None. This slice changes backend analysis behavior only and has no rendered UI/UX change.

## Blocked by

- None - issues 026 and 027 are complete; can start immediately.

## Artifact anchors

- PRD: `CL-FR-004`, `CL-FR-005`, and `CL-FR-006`.
- Domain model: `.cljc` platform metadata, `polymorphic` tags, diagnostics, confidence, and no-evaluation invariants.
- Use cases: `ExtractPolymorphicMetadata`, `EmitClojureAnalysisResult`; reader/platform behavior in the orchestration flow.
- Contract: `platform` option (`clj|cljs|both`) and generic `depends_on`/diagnostic metadata.

## Acceptance scenarios addressed

- `SC-CL-003 — Preserve reader conditionals`
- `SC-CL-004 — Mark polymorphic forms`
- `SC-CL-005 — Do not evaluate forms`
- `SC-AS-002 — Preserve a partial result`
- `SC-AS-006 — Enforce exclusions and safety`
- `SC-AS-008 — Deterministic repeat`

## Verification obligations

- Policy source: `/.okf/project.md`

| Scenario | Backend boundary | Frontend integration | End-to-end journey |
|---|---|---|---|
| `SC-CL-003` | `passed: internal/clojureanalyzer/safety_test.go::TestReaderConditionalsSelectPlatformsAndRetainMetadata` | `not-applicable` | `passed: cmd/arch-view/clojure_analyzer_test.go::TestClojureCLIExplicitAndAutomaticAnalysisJSON` |
| `SC-CL-004` | `passed: internal/clojureanalyzer/safety_test.go::TestPolymorphicMetadataAndDynamicLoadingAreSafe` | `not-applicable` | `passed: internal/clojureanalyzer/safety_test.go::TestPolymorphicMetadataAndDynamicLoadingAreSafe` |
| `SC-CL-005` | `passed: internal/clojureanalyzer/safety_test.go::TestPolymorphicMetadataAndDynamicLoadingAreSafe` | `not-applicable` | `passed: internal/clojureanalyzer/safety_test.go::TestPolymorphicMetadataAndDynamicLoadingAreSafe` |
| `SC-AS-002` | `passed: internal/clojureanalyzer/safety_test.go::TestSplicedAndMalformedReaderConditionalsRemainStatic` | `not-applicable` | `not-applicable: no new UI behavior` |
| `SC-AS-006` | `passed: internal/clojureanalyzer/analyzer_test.go::TestResolveProjectRejectsEscapingAndSymlinkedRoots` | `not-applicable` | `passed: internal/clojureanalyzer/safety_test.go::TestSplicedAndMalformedReaderConditionalsRemainStatic` |
| `SC-AS-008` | `passed: internal/clojureanalyzer/safety_test.go::TestReaderConditionalsSelectPlatformsAndRetainMetadata` | `not-applicable` | `passed: internal/clojureanalyzer/safety_test.go::TestReaderConditionalsSelectPlatformsAndRetainMetadata` |

## Closure evidence

- `go test ./internal/clojureanalyzer -count=1` passed.
- Platform, reader-conditional, polymorphism, dynamic-safety, malformed-recovery, and no-evaluation tests passed.
- `git diff --check` passed before closeout.
