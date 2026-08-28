# Clojure implementation slice: repository to visible architecture view

## Selected frontier

The [Clojure compatibility capability](../../../../.okf/capabilities/analyze-source/clojure-compatibility.md) is a readiness-reviewed `implemented` child of [Analyze source code](../../../../.okf/capabilities/analyze-source.md). The application PRD and application architecture summary agree with the confirmed graph and now record the verified in-process Clojure adapter, so the complete delivery slice is closed.

The user-approved issue sequence is 026–029. It adds one in-process language adapter behind the existing analyzer contract and does not create a new capability, shared concern, canonical model type, viewer branch, or renderer contract.

## Current and target truth

- **Observed in code:** the common `analysis.Analyzer` contract, deterministic host selection, canonical normalization, local viewer, and JSON/HTML/SVG paths exist; Go, Clojure, and Python are registered; the Clojure foundation now resolves safe project roots and emits namespace modules with evidence.
- **Inferred from docs:** the Clojure contract requires static project/configuration discovery, namespace modules, `:require`/`:use`/macro dependencies, `.cljc` platform metadata, polymorphic tags, diagnostics, source evidence, and no evaluation.
- **User-confirmed target:** the Clojure capability should be implemented on the isolated branch through the existing language-neutral path.
- **Observed result:** the target is met; all four approved issues are archived after acceptance and repository verification.
- **Mismatch:** none identified between the specified behavior and the verified implementation.

## Vertical outcome

Given a Clojure-family repository, a developer can:

1. select or auto-detect the Clojure analyzer;
2. discover safe `.clj`, `.cljs`, and `.cljc` namespace modules with structured hierarchy and file evidence;
3. inspect static require/use/macro relationships and unresolved references with aliases, platform metadata, confidence, diagnostics, and source locations;
4. see statically recognized polymorphic forms and dynamic-loading uncertainty without target-code execution; and
5. pass the resulting language-neutral model through the existing viewer and deterministic JSON/HTML/SVG export paths.

## Ordered delivery issues

| Issue | Outcome | Owner | Blocked by | Review gate |
|---|---|---|---|---|
| [026](../../../agents/issues/done/20260827-026-clojure-project-and-namespace-discovery.md) | Registered analyzer, marker/configuration boundary, safe source roots, flavor filtering, namespace modules, and evidence | Clojure analysis + plugin composition | complete | none |
| [027](../../../agents/issues/done/20260827-027-clojure-static-namespace-dependencies.md) | Static require/use/macro observations, local resolution, references, confidence, and partial diagnostics | Clojure analysis | complete | none |
| [028](../../../agents/issues/done/20260827-028-clojure-platform-polymorphism-and-safety.md) | `.cljc` platform conditionals, polymorphic metadata, dynamic-loading diagnostics, and malformed recovery | Clojure analysis | complete | none |
| [029](../../../agents/issues/done/20260827-029-clojure-public-integration-and-deterministic-exports.md) | Public CLI option exposure and shared model/viewer/export integration with deterministic artifacts | Clojure analysis + existing consumers | complete | none |

## Explicit non-goals

- Evaluating, importing, requiring, installing, or introspecting target Clojure code.
- Macro expansion, runtime classpath resolution, call graphs, or complete reader-condition evaluation.
- Changes to canonical model JSON, layout configuration, renderer routing, or the upstream reference repository.
- A Clojure-specific viewer or export implementation.

## Verification surfaces

- **Backend boundary:** manifest/detection, project-marker precedence, source-root safety, namespace parsing, dependency kinds, reader conditionals, polymorphic tags, diagnostics, determinism, and cancellation.
- **Frontend integration:** existing viewer hierarchy, directed relationships, evidence, references, confidence, and diagnostics remain language-neutral; no rendered UI/UX change is planned.
- **End-to-end:** Clojure repository → analysis JSON → canonical model → local viewer and JSON/HTML/SVG exports, with repeated-output checks.
- **Repository/OKF integrity:** full repository gates, strict OKF validation, synchronized issue references, and untouched upstream reference boundary.

## Artifact impact

The four approved issues changed delivery truth, and issue 029's closeout now records the verified implementation in the application PRD, application architecture summary, owning capability, parent orchestration, implementation slice, registry, and OKF log. Product scope, topology, canonical model semantics, and cross-capability boundaries remain unchanged.

## Compiled entrypoint evidence

Issue [034](../../../agents/issues/done/20260828-034-compiled-analyzer-plugin-entrypoints.md)
adds the compiled `org.archview.clojure` entrypoint at
`cmd/analyzers/clojure/main.go`. It constructs the existing
`clojureanalyzer.New()` implementation behind the shared process runner and
passes manifest and canonical-result parity verification without evaluating
target code.
