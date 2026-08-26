# Arch View product requirements

## Status

This is the application-level planning baseline. It records confirmed product scope and links to the capability map. All capability territories are now `specified`; their exact specification sets are linked below.

## Product

Arch View analyzes supported source repositories and presents their architecture as a navigable graph of modules, relationships, hierarchy, cycles, and layers. The first implementation targets Go and establishes a plugin boundary for Python, TypeScript, Rust, and Clojure support, with configurable presentation layout for repository-specific viewing needs.

The reference implementation is preserved in `external/` for reading only. It is not part of the product source tree and must remain ignored by Git.

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

## Primary user journeys

1. A developer points Arch View at a Go repository and opens its local-first top-level architecture, with optional reference-boundary expansion.
2. The developer drills into a module, inspects its imports and dependency evidence, and opens the related source file.
3. A maintainer identifies a cycle and uses the model output to understand the participating packages.
4. A CI job runs headless analysis and stores a versioned architecture artifact.
5. A later language plugin analyzes a Python, TypeScript, Rust, or Clojure repository using the same model and viewer.
6. A maintainer generates deterministic JSON, HTML, or SVG artifacts for documentation or CI.
7. A developer adjusts ELK layout settings for a repository, saves them back to the active discovered configuration (or uses `Save As` for a custom folder), and reopens the project with the same effective profile when the file is discoverable from the selected target.

## Non-goals

- Manual diagram authoring or collaborative cloud editing.
- Runtime tracing or a complete call graph.
- Automatic architectural approval or refactoring.
- Perfect resolution of dynamic imports, generated code, or runtime dispatch.
- Treating `.archview.json` as a container for analyzer options, canonical model data, viewport state, or manual diagram positions.
- Changes to the reference material under `external/`.

## Capability map

- [Analyze source code](/.okf/capabilities/analyze-source.md)
- [Generate architecture models](/.okf/capabilities/generate-models.md)
- [Explore and inspect architecture](/.okf/capabilities/explore-architecture.md)
- [Export and automate](/.okf/capabilities/export-and-automate.md)

## Capability maturity

- [Analyze source code](/.okf/capabilities/analyze-source.md): `specified`, including specified plugin-runtime, Go, Python, TypeScript, Rust, and Clojure child contracts.
- [Generate architecture models](/.okf/capabilities/generate-models.md): `specified`, including the v1 canonical model and graph-projection contract.
- [Explore and inspect architecture](/.okf/capabilities/explore-architecture.md): `specified`, including the local web/scene/evidence contract.
- [Export and automate](/.okf/capabilities/export-and-automate.md): `specified`, including JSON v1 and HTML/SVG/CI behavior.

## Cross-capability dependencies

Source analyzers produce evidence for the language-neutral model. The model owns stable identity, structured hierarchy, typed relationships, provenance, cycles, and derived layers. Structural containment remains separate from semantic dependency edges. The viewer consumes the model through a renderer-neutral view contract and a presentation-only layout profile. The local host resolves nearest-ancestor `.archview.json` settings and safely persists explicit project preferences, while exporters produce versioned JSON, HTML, and SVG without knowing source syntax. Analyzer configuration remains separate from viewer layout configuration, and the plugin runtime is the only boundary that should know how a language is detected or parsed.

## Implementation sequence

1. Define the neutral architecture model and analyzer contract.
2. Implement Go package and import analysis.
3. Build headless JSON output and model validation.
4. Build the first local web viewer with local-first reference visibility, imports/evidence inspection, and source workflow.
5. Add cycle and layout diagnostics plus user-selectable ELK settings and project configuration discovery.
6. Add Python, TypeScript, and Rust analyzers.
7. Add Clojure compatibility and an external, versioned plugin protocol if third-party analyzers are needed.

The specification set is complete and readiness-reviewed. Issue slicing is active through the normal delivery plan; the Go analyzer, canonical model, headless projection path, local viewer implementation, deterministic JSON/HTML/SVG export, and the issue 007 layout-settings/configuration backend and browser surface are implemented and visually approved. Issue 008's bounded parent-level ELK option tranche is implemented and visually approved; issue 009 follows with target-aware node/edge mapping.

## Verification strategy

The root project uses `when-supported` verification. Applicable backend, frontend, and end-to-end checks should run when their harnesses exist. Any deferred surface must record its reason. Boundary tests should cover analyzer output, model normalization, graph behavior, viewer interactions, and stable export.

## Current planning gaps

No high or medium blocker prevents the specified planning baseline. The following are implementation risks and verification work, not unresolved product decisions:

- Benchmarking cycle/layer algorithms and large-graph rendering.
- Reference-visibility policies, import-list fixtures, and session-layout behavior need representative small and large graph verification.
- Publishing/migrating JSON and NDJSON schemas.
- Parser fixture breadth and dynamic-language precision.
- Cross-environment HTML/SVG determinism and source-serving security tests.
- Pinned-ELK option compatibility, nearest-ancestor configuration fixtures, active-file `Save` versus custom-folder `Save As`, and safe project-settings writes.

## Specification sources

- [Analysis specification](architecture/analyze-source/prd.md), [contract](architecture/analyze-source/canonical-api-cli-contract.md), and [readiness review](architecture/analyze-source/readiness-review.md).
- [Plugin runtime specification](architecture/analyze-source/plugin-runtime/prd.md) and [contract](architecture/analyze-source/plugin-runtime/canonical-api-cli-contract.md).
- [Go](architecture/analyze-source/go-analysis/prd.md), [Python](architecture/analyze-source/python-analysis/prd.md), [TypeScript](architecture/analyze-source/typescript-analysis/prd.md), [Rust](architecture/analyze-source/rust-analysis/prd.md), and [Clojure](architecture/analyze-source/clojure-compatibility/prd.md) specifications.
- [Model specification](architecture/generate-models/prd.md) and [canonical contract](architecture/generate-models/canonical-api-cli-contract.md).
- [Viewer specification](architecture/explore-architecture/prd.md) and [scene contract](architecture/explore-architecture/canonical-api-cli-contract.md).
- [Export specification](architecture/export-and-automate/prd.md) and [export contract](architecture/export-and-automate/canonical-api-cli-contract.md).

## Source references

- [Reference tool README](../external/README.md)
- [Reference project notes](../external/PROJECT_NOTES.md)
- [Arch View planning map](../.okf/index.md)
