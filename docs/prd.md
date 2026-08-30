# Arch View product requirements

## Status

This is the application-level planning baseline. It records confirmed product scope and links to the capability map. Capability territories are now specified, implemented, or explicitly tracked for implementation; their detailed artifacts are linked below.

## Product

Arch View analyzes supported source repositories and presents their architecture as a navigable graph of modules, relationships, hierarchy, cycles, and layers. The first implementation targets Go and establishes a plugin boundary for Python, Clojure, TypeScript, and Rust support, with configurable presentation layout for repository-specific viewing needs. Go, Python, TypeScript, Rust, and Clojure run as in-process adapters behind the same language-neutral contract, and the current deployment also includes an explicitly supplied external Python process plus trusted packaged compiled analyzer executables behind that contract. In-process adapters remain available for development and explicit migration fallback. The multi-analyzer analysis path is implemented through the verified and user-approved 039–043 delivery slices, including the viewer's mixed-language visual review; project-relative analyzer assignments with selectable scopes and persisted source-scope configuration are implemented through verified issues 044–047, including the approved configured-viewer visual review.

The reference implementation is available in the upstream
[unclebob/arch-view repository](https://github.com/unclebob/arch-view) for
reading only. It is not part of the product source tree.

## Actors

- Developers learning an unfamiliar repository.
- Maintainers investigating dependency structure and cycles.
- Architects reviewing whether code structure matches intended boundaries.
- CI or documentation workflows that need repeatable architecture artifacts.
- Coding assistants and MCP clients that need compact, evidence-backed source and quality information.

## MVP scope

- Analyze a Go repository without executing its application.
- Discover packages, local imports, source files, and dependency evidence.
- Convert analysis into a language-neutral architecture model.
- Detect cycles and assign modules to layers.
- Provide a first navigable architecture view with drill-down and source inspection.
- Produce deterministic headless JSON output.
- Provide self-contained HTML and scalable SVG artifacts from the same neutral model/view contract.
- Serve the first interactive experience as a local web application with progressive disclosure for large graphs.
- Make the default architecture overview local-first: retain standard-library, external, unresolved, and dynamic imports in the model while exposing them through boundary summaries and an accessible imports/evidence view.
- Keep collapsed group presentation honest: non-cycle internal relationships are summarized as counts/details rather than rendered as cycle-like self-loops, while real cycles remain visible.
- Allow developers to inspect and apply the pinned ELK layout catalog and persist presentation preferences in a nearest-ancestor `.archview.json` configuration file.
- Keep analyzer-specific behavior behind a plugin contract.
- Allow a maintainer to opt into an external analyzer through an explicit
  local descriptor without executing target project code or changing the
  model/viewer/export contracts.

## Confirmed extension direction

The current v1 product path remains valid. The compiled external analyzer
distribution is implemented and is the production release path for stable
logical analyzer IDs. The project-assignment and source-scope extension is also
implemented through issues 044–047:

- Persist repository-relative folder or project-root to analyzer assignments
  and let the application switch between individual analyzer scopes and the
  combined view.
- Define source scope in project configuration with global exclusion globs and
  analyzer-ID-scoped include globs relative to the invocation root, with
  deterministic matching and exclusions taking precedence.

The multi-analyzer behavior is now implemented through the multi-analyzer child
delivery path, including its approved viewer review. Persisted assignment
configuration is implemented through the project-assignment child and its
approved issues 044–047; the compiled distribution child is implemented.

The implemented multi-analyzer direction is intentionally specific: repository
opening discovers marker-driven nested project roots and schedules different
language jobs through bounded concurrency; merged identities include relative
root, analyzer ID, and local ID; failures yield partial results; and the viewer
  exposes cached `All` and per-scope views. A separate `analysis` section in
  `.archview.json` for persisted assignments and source scope remains owned by
  the implemented project-assignment child capability.

### Code quality and code intelligence (specified roll-up, source-facts delivery implemented)

The [Code quality and code intelligence](/.okf/capabilities/code-quality-and-intelligence.md)
capability is a pure structural-child roll-up with materialized state
`specified`, derived as the minimum of its three structural children. The parent
is an aggregate/navigation boundary with no standalone PRD or implementation
slice; it extends the product through three child territories:

- The [Source facts and symbol index](architecture/code-quality-and-intelligence/source-facts-and-symbol-index/prd.md) is implemented through issues 048–052 as an optional, scope-first attachment with first-class files, physical line counts, hashes, documentation, named declarations, visibility, containment, provenance, typed extension points, bounded queries, and human-oriented module inspection. The graph presents a compact summary card; a same-tab inspection route progressively loads bounded Files, Symbols, and evidence views, while technical identifiers remain secondary. Its required final visual-review gate is approved.
- The [Deterministic quality checks](architecture/code-quality-and-intelligence/deterministic-quality-checks/prd.md) child is implemented through issues 053–063: configurable size, complexity, documentation, coupling, cycle, dependency-direction, and layer rules; deterministic report lifecycle/baselines; conservative SOLID signals; bounded queries; headless/export projections; and the report-backed viewer. Metrics/rules are versioned registry strategies; unsupported inputs are not passes, and SOLID output is explicitly a signal rather than a provable violation. Issue 063 passed its desktop and responsive visual review.
- The [Live analysis and MCP](architecture/code-quality-and-intelligence/live-analysis-and-mcp/prd.md) child is now specified for configured-folder watching, revision-aware immutable snapshots, compact structural code search, and on-demand quality reports to LLM tools. Reporting is read-only by default; source changes remain an explicit downstream action.

The source-facts child is implemented through its approved visual review. The
deterministic-quality child is `implemented`; issues 053–063 have delivered the quality profile and
catalog, source/graph providers, exact rules, report lifecycle and baseline
workflow, SOLID signals, bounded queries, and headless/export projections. The
live/MCP child remains specified and not implemented. Their exact contracts
keep source facts, quality policy, snapshot lifecycle, structural search, and
transport permissions separately owned.

## Primary user journeys

1. A developer points Arch View at a Go repository and opens its local-first top-level architecture, with optional reference-boundary expansion.
2. The developer drills into a module, inspects its imports and dependency evidence, and opens the related source file.
3. A maintainer identifies a cycle and uses the model output to understand the participating packages.
4. A CI job runs headless analysis and stores a versioned architecture artifact.
5. A language adapter analyzes a Python, TypeScript, Rust, or Clojure repository through the same model and viewer.
6. A maintainer generates deterministic JSON, HTML, or SVG artifacts for documentation or CI.
7. A developer adjusts ELK layout settings for a repository, saves them back to the active discovered configuration (or uses `Save As` for a custom folder), and reopens the project with the same effective profile when the file is discoverable from the selected target.
8. A maintainer supplies an external analyzer descriptor, selects the
   external Python adapter explicitly, and receives the same architecture
   workflow and evidence surfaces as the in-process path.
9. A developer opens a mixed-language repository; applicable compiled analyzer
   executables run concurrently and the resulting scopes appear in one
   architecture session.
10. A developer assigns a repository-relative folder to a specific analyzer
    in project configuration and switches between that scope and the combined
    view from the application.
11. A developer restricts analysis to selected source globs, excludes generated
     or vendor content globally, and confirms that changing those filters
     invalidates the affected cached scopes.
12. A developer selects a module or group, gets a compact summary, and opens a
    same-tab inspection view whose Overview, Structure, Files, Symbols,
    Dependencies, Evidence, and Technical details sections progressively reveal
    constituent files, declarations, documentation, provenance, and source
    metrics. Files and symbols are bounded and filterable; source excerpts
    require an explicit read-only action.
13. A maintainer configures deterministic quality thresholds, sees findings
    highlighted in the architecture/detail views, can see how many files exceed
    a configured line limit and filter to those files, and exports the same
    findings for automation.
14. A coding assistant queries the live MCP surface for current findings,
    symbols, callers/callees where available, and bounded source context.

## Non-goals

- Manual diagram authoring or collaborative cloud editing.
- Runtime tracing or a complete call graph.
- Automatic architectural approval or refactoring.
- Presenting subjective or LLM-generated judgments as deterministic findings;
  SOLID labels and comment usefulness remain signals or coverage checks.
- Implicit source mutation from analysis, watching, or MCP reporting.
- Perfect resolution of dynamic imports, generated code, or runtime dispatch.
- Treating `.archview.json` as a container for canonical model data, viewport
  state, or manual diagram positions. Its separately owned `analysis` section
  remains distinct from layout semantics.
- Changes to the upstream reference implementation.

## Capability map

- [Analyze source code](/.okf/capabilities/analyze-source.md)
- [Generate architecture models](/.okf/capabilities/generate-models.md)
- [Explore and inspect architecture](/.okf/capabilities/explore-architecture.md)
- [Export and automate](/.okf/capabilities/export-and-automate.md)
- [Code quality and code intelligence](/.okf/capabilities/code-quality-and-intelligence.md)

## Capability maturity

- [Analyze source code](/.okf/capabilities/analyze-source.md): `implemented`,
  with the Go, Python, TypeScript, Rust, and Clojure children implemented, the
  first external Python process slice verified, the compiled-distribution child
  implemented, the multi-analyzer orchestration path verified and visually
  approved through issues 039–043, and the project analyzer assignments/view
  selection child implemented through verified issues 044–047, including the
  approved configured-viewer visual review.
- [Generate architecture models](/.okf/capabilities/generate-models.md): `implemented`, including the v1 canonical model and graph-projection contract.
- [Explore and inspect architecture](/.okf/capabilities/explore-architecture.md): `implemented` for the current local web/scene/evidence contract; its [Advanced ELK renderer support child](/.okf/capabilities/explore-architecture/advanced-elk-renderer-support.md) is `specified` for future extensions.
- [Export and automate](/.okf/capabilities/export-and-automate.md): `implemented`, including JSON v1 and deterministic HTML/SVG/CI behavior.
- [Code quality and code intelligence](/.okf/capabilities/code-quality-and-intelligence.md): `specified` as a structural roll-up; its [Source facts and symbol index](architecture/code-quality-and-intelligence/source-facts-and-symbol-index/prd.md) child is implemented through issues 048–052 with final visual approval, while [Deterministic quality checks](architecture/code-quality-and-intelligence/deterministic-quality-checks/prd.md) and [Live analysis and MCP](architecture/code-quality-and-intelligence/live-analysis-and-mcp/prd.md) remain `specified`. The parent has no standalone implementation slice.

## Cross-capability dependencies

Source analyzers produce evidence for the language-neutral model. The model owns stable identity, structured hierarchy, typed relationships, provenance, cycles, and derived layers. Structural containment remains separate from semantic dependency edges. The viewer consumes the model through a renderer-neutral view contract and a presentation-only layout profile. The local host resolves nearest-ancestor `.archview.json` settings and safely persists explicit project preferences, while exporters produce versioned JSON, HTML, and SVG without knowing source syntax. The implemented analyzer runtime plans a set of project/analyzer jobs and applies the validated invocation-root source-scope policy, while configuration keeps persisted analyzer assignments and source filters distinct from viewer layout settings and the plugin runtime remains the only boundary that knows how a language is detected or parsed.

The code-quality/code-intelligence extension consumes analyzer and model facts
without changing the module graph's default granularity. Its roll-up parent
provides aggregate navigation only; its source-facts child attaches an
optional `SourceIndex` beside existing modules,
relationships, source references, diagnostics, and derived data. The index
retains authoritative per-scope snapshots with first-class files, named
symbols, documentation candidates, hash-linked spans, provenance, containment,
and versioned metric slots; a combined view is an explicit projection and does
not infer cross-scope relationships. The viewer keeps this attachment out of
the initial graph request, then uses bounded source-index queries with explicit
module/file containment filters for the inspection route. Registered language extractors own
language semantics, while the core owns validation, opaque IDs, coverage,
canonical ordering, and digest assembly. The specified quality child evaluates
versioned metrics and rules through registries, emits exact findings or
explicitly labeled SOLID signals, and keeps baselines/coverage separate from
source facts. The specified live child owns watcher freshness, conservative
invalidation, atomic coherent revisions, compact structural queries, budgets,
and read-only permissions; MCP remains its transport/query adapter rather than
an owner of analysis semantics.

## Implementation sequence

1. Define the neutral architecture model and analyzer contract.
2. Implement Go package and import analysis.
3. Build headless JSON output and model validation.
4. Build the first local web viewer with local-first reference visibility, imports/evidence inspection, and source workflow.
5. Add cycle and layout diagnostics plus user-selectable ELK settings and project configuration discovery.
6. Add the Python, TypeScript, Rust, and Clojure analyzers through the same common contract.
7. Add the opt-in external, versioned plugin protocol and validate it with
   the external Python parity pilot.
8. Replace the script-based pilot with compiled external analyzer executables
   that reuse the in-process implementations. **Completed through issues
   034–038.**
9. Add multi-project/multi-analyzer job planning, bounded concurrency, result
   merging, and partial-failure behavior. **Implemented through issues
   039–043, including the approved viewer visual review.**
10. Add project-relative analyzer assignments, invocation-root source-scope
    filtering, and application scope selection. **Implemented and verified
    through issues 044–047, including the approved configured-viewer visual
    review.**
11. Specify and then implement first-class source facts and symbol indexing for
    files, documentation, declarations, and deterministic source metrics. The
    source-index contract and issues 048–052 are implemented; final visual
    approval of the module-inspection projection is recorded.
12. Specify and then implement configurable deterministic quality rules,
    findings, architecture constraints, and machine-readable quality reports.
    The quality contract is now specified and readiness-reviewed; delivery is
    not claimed.
13. Specify and then implement live analysis snapshots, folder watching,
    compact structural search, and MCP reporting with explicit freshness and
    source-safety policies. The live/MCP contract is now specified and
    readiness-reviewed; delivery is not claimed.

The specification set is complete and readiness-reviewed. A 2026-08-28
brownfield audit confirmed that the Go analyzer, canonical model, and export
capabilities exhaust their promised scopes, so their planning states are now
`implemented`. Their linked completed issues cover all acceptance scenarios,
`go test ./...` passes, the browser module tests pass, and the export paths
retain their recorded visual approvals. The current Explore scope and the
Python, TypeScript, Rust, and Clojure adapters are also implemented. Issues
030–033 complete the first external plugin-runtime slice. Issues 034–038
complete the compiled distribution capability. The multi-analyzer child is
implemented through issues 039–043, including its approved viewer visual
review, and the project assignment/view child is implemented through verified
issues 044–047, including its approved configured-viewer visual review. The
advanced ELK renderer child remains specified.

## Future planning state

The explicitly supplied external Python deployment remains a compatibility
pilot and does not change model/viewer/export semantics. Release builds use the
implemented trusted compiled external distribution, while in-process adapters
remain explicit development or migration paths. Multi-analyzer orchestration
is implemented, including the approved issue 043 visual review; project
assignment/view selection and its source-scope policy are implemented through
verified issues 044–047, including the approved configured-viewer visual
review. Advanced ELK renderer support remains the specified renderer-only
extension of the implemented viewer. The code-quality and code-intelligence
roll-up remains `specified`; its source-facts child is implemented with its
approved final visual gate, while deterministic-quality and live-analysis/MCP remain
specified. Further implementation work starts at the least-mature remaining
child rather than at the aggregate parent.

## Verification strategy

The root project uses `when-supported` verification. Applicable backend, frontend, and end-to-end checks should run when their harnesses exist. Any deferred surface must record its reason. Boundary tests should cover analyzer output, model normalization, graph behavior, viewer interactions, and stable export.

## Current planning gaps

No high or medium blocker prevents the specified planning baseline. The following are implementation risks and verification work, not unresolved product decisions:

- Benchmarking cycle/layer algorithms and large-graph rendering.
- Reference-visibility policies, import-list fixtures, and session-layout behavior need representative small and large graph verification.
- Migrating JSON and NDJSON schemas beyond the published v1 external-plugin
  contract.
- Benchmarking process frame/stderr limits, timeout defaults, and
  cross-platform external-plugin launch behavior.
- Parser fixture breadth and dynamic-language precision.
- Cross-environment HTML/SVG determinism and source-serving security tests.
- Pinned-ELK option compatibility, nearest-ancestor configuration fixtures, active-file `Save` versus custom-folder `Save As`, and safe project-settings writes.
- Multi-analyzer job discovery, source-scope application, merging, transport,
  and cached viewer projection are implemented and covered by verified issues
  039–043, including the approved visual review. Folder-assignment
  configuration, persisted source-scope policy, and scope-selection dropdown
  behavior are implemented and covered by verified issues 044–047, including
  the approved configured-viewer visual review.
- Linux amd64 and Darwin arm64 packaged-analyzer execution remain deferred
  verification surfaces under the root `when-supported` policy until matching
  runners or toolchains are available.
- The source-facts and symbol-index contract is specified and readiness-reviewed;
  issues 048–052 implement the Go-first attachment, language extractor,
  scope-safe projection, bounded queries, and viewer projection. The final
  visual-review gate is approved.
- The deterministic quality contract is implemented and readiness-reviewed. Its
  implementation is delivered through issues 053–063, including the real Go
  structural-fact producer for SOLID signals and the `quality baseline` CLI
  workflow and the visually reviewed report-backed viewer.
- Live watcher/MCP snapshot behavior, permissions, and transport are specified
  and readiness-reviewed but still require implementation and packaging
  verification.

## Specification sources

- [Analysis specification](architecture/analyze-source/prd.md), [contract](architecture/analyze-source/canonical-api-cli-contract.md), and [readiness review](architecture/analyze-source/readiness-review.md).
- [Plugin runtime specification](architecture/analyze-source/plugin-runtime/prd.md) and [contract](architecture/analyze-source/plugin-runtime/canonical-api-cli-contract.md).
- [Go](architecture/analyze-source/go-analysis/prd.md), [Python](architecture/analyze-source/python-analysis/prd.md), [TypeScript](architecture/analyze-source/typescript-analysis/prd.md), [Rust](architecture/analyze-source/rust-analysis/prd.md), and [Clojure](architecture/analyze-source/clojure-compatibility/prd.md) specifications.
- [Model specification](architecture/generate-models/prd.md) and [canonical contract](architecture/generate-models/canonical-api-cli-contract.md).
- [Viewer specification](architecture/explore-architecture/prd.md) and [scene contract](architecture/explore-architecture/canonical-api-cli-contract.md).
- [Export specification](architecture/export-and-automate/prd.md) and [export contract](architecture/export-and-automate/canonical-api-cli-contract.md).

## Source references

- [Reference tool repository](https://github.com/unclebob/arch-view)
- [Reference project](https://github.com/unclebob/arch-view)
- [Arch View planning map](../.okf/index.md)
