# Arch View product requirements

## Status

This is the application-level planning baseline. It records confirmed product scope and links to the capability map. Capability territories are now specified, implemented, or explicitly tracked for implementation; their detailed artifacts are linked below.

## Product

Arch View analyzes supported source repositories and presents their architecture as a navigable graph of modules, relationships, hierarchy, cycles, and layers. The first implementation targets Go and establishes a plugin boundary for Python, Clojure, TypeScript, and Rust support, with configurable presentation layout for repository-specific viewing needs. Go, Python, TypeScript, Rust, and Clojure currently run as in-process adapters behind the same language-neutral contract, and the current deployment also includes an explicitly supplied external Python process behind that contract. The confirmed future direction is compiled external analyzer executables, concurrent multi-analyzer analysis, and project-relative analyzer assignments with selectable scopes.

The reference implementation is available in the upstream
[unclebob/arch-view repository](https://github.com/unclebob/arch-view) for
reading only. It is not part of the product source tree.

## Actors

- Developers learning an unfamiliar repository.
- Maintainers investigating dependency structure and cycles.
- Architects reviewing whether code structure matches intended boundaries.
- CI or documentation workflows that need repeatable architecture artifacts.

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

## Confirmed future direction

The current v1 product path remains valid, but the next analyzer/runtime
frontier is explicitly broader:

- Ship supported language analyzers as compiled external executables that reuse
  the existing analyzer implementations and do not require language runtimes
  on the user's machine.
- Detect and run multiple applicable analyzers concurrently across mixed or
  nested project roots, then merge their results into one architecture model.
- Persist repository-relative folder or project-root to analyzer assignments
  and let the application switch between individual analyzer scopes and the
  combined view.

These are future capabilities, not additions to the current MVP acceptance
scope. They are represented by specified child nodes under [Analyzer plugin
runtime](/.okf/capabilities/analyze-source/plugin-runtime.md).

The specified direction is intentionally specific: compiled binaries are the
production implementation for stable logical analyzer IDs and are assembled by
a reproducible build target under an application-managed `analyzers` tree;
repository opening discovers marker-driven nested project roots and schedules
different language jobs through bounded concurrency; merged identities include
relative root, analyzer ID, and local ID; failures yield partial results; and a
separate `analysis` section in `.archview.json` controls assignments while the
viewer exposes cached `All` and per-scope views.

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

## Non-goals

- Manual diagram authoring or collaborative cloud editing.
- Runtime tracing or a complete call graph.
- Automatic architectural approval or refactoring.
- Perfect resolution of dynamic imports, generated code, or runtime dispatch.
- Treating `.archview.json` as a container for canonical model data, viewport
  state, or manual diagram positions. A future version may add a separately
  owned analysis-assignment section without mixing it with layout semantics.
- Changes to the upstream reference implementation.

## Capability map

- [Analyze source code](/.okf/capabilities/analyze-source.md)
- [Generate architecture models](/.okf/capabilities/generate-models.md)
- [Explore and inspect architecture](/.okf/capabilities/explore-architecture.md)
- [Export and automate](/.okf/capabilities/export-and-automate.md)

## Capability maturity

- [Analyze source code](/.okf/capabilities/analyze-source.md): `specified`,
  with the Go, Python, TypeScript, Rust, and Clojure children implemented, the
  first external Python process slice verified, and three specified future
  plugin-runtime children for compiled distribution, multi-analyzer
  orchestration, and project analyzer assignments/view selection.
- [Generate architecture models](/.okf/capabilities/generate-models.md): `implemented`, including the v1 canonical model and graph-projection contract.
- [Explore and inspect architecture](/.okf/capabilities/explore-architecture.md): `implemented` for the current local web/scene/evidence contract; its [Advanced ELK renderer support child](/.okf/capabilities/explore-architecture/advanced-elk-renderer-support.md) is `specified` for future extensions.
- [Export and automate](/.okf/capabilities/export-and-automate.md): `implemented`, including JSON v1 and deterministic HTML/SVG/CI behavior.

## Cross-capability dependencies

Source analyzers produce evidence for the language-neutral model. The model owns stable identity, structured hierarchy, typed relationships, provenance, cycles, and derived layers. Structural containment remains separate from semantic dependency edges. The viewer consumes the model through a renderer-neutral view contract and a presentation-only layout profile. The local host resolves nearest-ancestor `.archview.json` settings and safely persists explicit project preferences, while exporters produce versioned JSON, HTML, and SVG without knowing source syntax. The future analyzer runtime will plan a set of project/analyzer jobs, while configuration will keep analyzer assignments distinct from viewer layout settings and the plugin runtime remains the only boundary that knows how a language is detected or parsed.

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
   that reuse the in-process implementations.
9. Add multi-project/multi-analyzer job planning, bounded concurrency, result
   merging, and partial-failure behavior.
10. Add project-relative analyzer assignments and application scope selection.

The specification set is complete and readiness-reviewed. A 2026-08-28
brownfield audit confirmed that the Go analyzer, canonical model, and export
capabilities exhaust their promised scopes, so their planning states are now
`implemented`. Their linked completed issues cover all acceptance scenarios,
`go test ./...` passes, the browser module tests pass, and the export paths
retain their recorded visual approvals. The current Explore scope and the
Python, TypeScript, Rust, and Clojure adapters are also implemented. Issues
030–033 complete the first external plugin-runtime slice. The compiled
distribution capability is now in approved delivery through issues 034–038.
The multi-analyzer orchestration, project assignment/view selection, and
advanced ELK renderer children remain specified and have no delivery issues
yet.

## Future planning state

The current external deployment remains the explicitly supplied Python pilot
and does not change model/viewer/export semantics. The compiled external
distribution is a specified frontier with approved delivery issues 034–038 and
is not claimed as implemented by issues 030–033. Multi-analyzer orchestration
and project assignment/view selection remain specified future frontiers
without delivery issues. Advanced ELK renderer support is likewise specified
as a renderer-only extension of the implemented viewer.

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
- Exact compiled analyzer artifact packaging, platform selection,
  trust/version policy, multi-analyzer job discovery and merging, and
  folder-assignment configuration/dropdown schemas are specified in the linked
  child artifacts; implementation and verification remain future work.

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
