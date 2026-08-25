# Arch View application architecture summary

## Status

This is the application-level architecture baseline. It describes the target Go architecture and distinguishes it from the Clojure reference implementation. All capability territories are now `specified`; their exact contracts are the implementation reference set.

## Boundary summary

```text
CLI / local web entrypoint
        |
        v
Plugin manager and project detection
        |
        v
Language analyzers
  Go, Python, TypeScript, Rust, Clojure
        |
        v
Language-neutral architecture model
        |
        +--> graph normalization, cycles, hierarchy, layers
        |
        +--> renderer-neutral view contract
        |
        +--> local web viewer
        +--> versioned JSON, SVG, HTML, and CI outputs
```

## Current reference architecture

The `external/` tool couples its command entrypoint to a Clojure source scanner, graph construction, layer assignment, and a Quil/Swing viewer. Its source analyzer reads Clojure forms, recognizes Clojure-family extensions, extracts `:require` dependencies, and marks selected Clojure abstractions. Its graph and layering logic are the main reusable ideas.

## Target boundaries

### Confirmed analyzer boundary

The analyzer receives a repository or project root plus analysis options, selects one language analyzer, and returns validated modules, static dependency relationships, source evidence, metadata, and diagnostics. It does not assign layers, render diagrams, or export files.

The plugin runtime and language adapters are specified around this contract. The first deployment uses in-process Go analyzers; each adapter owns project-boundary and static-resolution rules, while the host owns selection, option precedence, validation, normalization, and safety. Python, TypeScript, Rust, and Clojure uncertainty is returned as evidence, confidence, or diagnostics rather than fabricated relationships.

The first graph uses package or module nodes. Files remain attached evidence. Project-local modules are shown by default. Tests, generated code, vendor directories, caches, build outputs, and `external/` are excluded by default. Unresolved dependencies produce partial results with diagnostics.

### CLI and orchestration

Owns command parsing, project selection, analyzer selection, configuration, progress, and error reporting. It must not parse language syntax or calculate layout.

### Plugin manager

Owns analyzer registration, project detection, capability negotiation, configuration, result validation, and plugin version compatibility. The first version can use built-in Go implementations. A later process protocol can allow analyzers written in other languages.

### Language analyzers

Each analyzer owns source discovery, syntax parsing, import or dependency resolution, language-specific classification, source locations, and diagnostics. An analyzer returns data, not UI elements or layout coordinates.

The adapters have exact language-specific contracts; parser/library choices may vary behind those contracts.

### Architecture model

Owns stable opaque module identity, language and project metadata, explicit hierarchy segments, source references, relationships, evidence, confidence, tags, analysis diagnostics, and derived cycle/layer projections. Hierarchy is structural and separate from semantic dependency edges. The model must represent one package containing many files and must not assume dot-separated names.

### Graph and view preparation

Owns relationship normalization, aggregation, abstraction classification, cycle detection, hierarchical projection, layer assignment, and layout inputs. It consumes only the neutral model. When hierarchy aggregation creates a non-cycle group self-loop, this boundary provides an internal-relationship summary instead of asking a renderer to imply a cycle.

### Presentation and export

The viewer owns a local web presentation, renderer-neutral scene/view state, interaction, navigation, progressive disclosure, reference-boundary visibility, import/evidence inspection, source inspection, tooltips, layout/session state, and accessibility concerns. The default overview is local-first; non-local references remain canonical but are hidden, aggregated, or expanded by view policy. The current SVG renderer owns arrowhead styling and consumes node positions plus edge sections/bend points from the locally served ELK/elkjs layered adapter. A deterministic layer-based layout remains the replaceable fallback; neither layout choice changes relationship meaning. Exporters consume the same view policy and own versioned JSON, deterministic HTML/SVG, and CI status behavior. Neither presentation nor export knows how a source language is parsed.

## Data flow

1. The CLI selects a repository and either an explicit analyzer or an analyzer detected from project files.
2. The plugin manager runs the analyzer with source-scope and filtering options.
3. The analyzer returns modules, relationships, evidence, diagnostics, and source references.
4. The model validator checks identity and relationship integrity.
5. The graph engine normalizes relationships, computes cycles, and assigns layers.
6. The renderer-neutral viewer or exporter consumes the resulting model and view projection.

## Shared policies

- Analysis is read-only and does not execute the target application.
- Paths and source evidence must remain traceable to the analyzed repository.
- Unresolved or dynamic dependencies are reported with confidence or diagnostics.
- Non-local references are retained in the canonical model; viewer/export projections decide whether they are hidden, aggregated, or expanded, while import evidence remains available in list/details form.
- External and generated code are filtered by explicit policy, not silently discarded.
- Language-specific concepts such as interfaces, protocols, or abstract classes are represented as metadata and relation semantics, not hard-coded into the core.
- Saved models use a versioned, language-neutral JSON interchange format. Visual artifacts are deterministic projections of the same model/view contract; source contents are not embedded by default.

## Verification

The root policy is `when-supported`, with justified deferrals required. The architecture should be verified at analyzer boundaries, model normalization, graph algorithms, viewer integration, export stability, and end-to-end repository analysis when those harnesses exist.

## Implementation sequence

The first architectural slice is the neutral model and plugin contract. The first product slice is Go package analysis connected to headless output and a visible viewer result. The viewer refinement now prioritizes a local-first overview and import/evidence inspection before broader analyzer expansion. Python, TypeScript, Rust, and Clojure analyzers follow the same contract. External process plugins come after the built-in contract has stabilized.

## Residual implementation decisions

- Benchmarking and tuning frontend/rendering thresholds.
- Reference-boundary aggregation, import-list density, and session-scoped layout persistence.
- Publishing/migrating JSON and NDJSON schemas.
- Process sandbox/resource-limit implementation.
- Future call-graph/type-level relation capabilities.

These are implementation and extension risks, not blockers to the specified v1 scope.

## Exact specification sources

- [Analysis parent specification](analyze-source/prd.md) and [canonical contract](analyze-source/canonical-api-cli-contract.md), with readiness-reviewed language/plugin child specifications.
- [Canonical model specification](generate-models/prd.md) and [model contract](generate-models/canonical-api-cli-contract.md).
- [Viewer specification](explore-architecture/prd.md) and [renderer-neutral scene contract](explore-architecture/canonical-api-cli-contract.md).
- [Export specification](export-and-automate/prd.md) and [JSON/HTML/SVG contract](export-and-automate/canonical-api-cli-contract.md).

## Traceability

- [Arch View project concept](../../.okf/project.md)
- [Analyze source code](../../.okf/capabilities/analyze-source.md)
- [Generate architecture models](../../.okf/capabilities/generate-models.md)
- [Explore and inspect architecture](../../.okf/capabilities/explore-architecture.md)
- [Export and automate](../../.okf/capabilities/export-and-automate.md)
- [Application PRD](../prd.md)
