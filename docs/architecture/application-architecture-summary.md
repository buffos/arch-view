# Arch View application architecture summary

## Status

This is the application-level architecture baseline. It describes the target Go architecture and distinguishes it from the Clojure reference implementation. Capability territories are now specified, implemented, or explicitly tracked for implementation; the verified in-process Clojure adapter and opt-in external Python process both reach the existing analyzer, model, viewer, and export paths without adding language-specific consumer branches. The next specified plugin-runtime frontiers are compiled external analyzer distribution, multi-analyzer project orchestration, and project analyzer assignments with application scope selection.

## Boundary summary

```text
CLI / local web entrypoint
        |
        v
Plugin manager and project detection
        |
        v
Analyzer job plan
  built-in or compiled external analyzer executables
  one or more project-root jobs running concurrently
    descriptor + versioned NDJSON
        |
        v
Language-neutral architecture model
        |
        +--> graph normalization, cycles, hierarchy, layers
        |
        +--> renderer-neutral view contract
        |
        +--> project configuration resolver -> analyzer assignments + viewer layout profile
        |
        +--> local web viewer
        +--> versioned JSON, SVG, HTML, and CI outputs
```

## Current reference architecture

The upstream [unclebob/arch-view reference implementation](https://github.com/unclebob/arch-view) couples its command entrypoint to a Clojure source scanner, graph construction, layer assignment, and a Quil/Swing viewer. Its source analyzer reads Clojure forms, recognizes Clojure-family extensions, extracts `:require` dependencies, and marks selected Clojure abstractions. Its graph and layering logic are the main reusable ideas.

## Target boundaries

### Confirmed analyzer boundary

The current v1 analyzer path receives a repository or project root plus
analysis options, selects one language analyzer, and returns validated modules,
static dependency relationships, source evidence, metadata, and diagnostics. It
does not assign layers, render diagrams, or export files. The future
multi-analyzer path plans a set of project-root/analyzer jobs, runs applicable
jobs concurrently, and merges their validated language-neutral observations
before model normalization.

The plugin runtime and language adapters are specified around this contract. The
current deployment uses in-process Go, Python, TypeScript, Rust, and Clojure
analyzers plus an explicitly loaded external Python process. The specified future
deployment distributes supported analyzers as compiled external executables that
reuse the same analyzer implementations and retain the same logical analyzer
IDs. Each adapter owns project-boundary and static-resolution rules, while the
host owns assignment resolution, marker-driven job planning, option precedence,
validation, normalization, merge identity, and process safety. Python,
TypeScript, Rust, and Clojure uncertainty is returned as evidence, confidence,
or diagnostics rather than fabricated relationships.

The first graph uses package or module nodes. Files remain attached evidence. Project-local modules are shown by default. Tests, generated code, vendor directories, caches, build outputs, and directories named `external` are excluded by default. Unresolved dependencies produce partial results with diagnostics.

### CLI and orchestration

Owns command parsing, project selection, analyzer-job planning, analyzer
assignment configuration, progress, and error reporting. It does not parse
language syntax or calculate layout. Presentation layout configuration is
resolved by the local host/viewer boundary and remains separate from analyzer
options and assignment semantics.

### Plugin manager

Owns analyzer registration, project detection, capability negotiation,
assignment resolution, job planning, result validation, and plugin version
compatibility. Built-in analyzers remain in-process during migration and only as
an explicit fallback. The target distribution is assembled by a reproducible
root build/make target under an application-managed `analyzers/<id>/<platform>/`
tree and registers compiled external executables from trusted descriptors. The
host validates manifest/hello agreement, launches each job with argv rather than
a shell, reserves stdout for versioned NDJSON, treats stderr as bounded logs,
and owns cancellation, timeout, bounded concurrency, and child cleanup. One
detect or analyze job uses one external process.

### Language analyzers

Each analyzer owns source discovery, syntax parsing, import or dependency resolution, language-specific classification, source locations, and diagnostics. An analyzer returns data, not UI elements or layout coordinates.

The adapters have exact language-specific contracts; parser/library choices may vary behind those contracts. The in-process Python adapter now extracts absolute, package, re-export, and relative imports through a conservative static pass, resolves only filesystem-proven local modules under effective source roots, and emits standard-library, external, unresolved, conditional, and dynamic references with source evidence, confidence, and recoverable diagnostics. The in-process Rust adapter reads one Cargo crate and statically discovers reachable crate/module hierarchy, `use`/`pub use` relationships, dependency declarations, source evidence, and cfg/macro/generated uncertainty without executing Cargo or target code. The in-process Clojure adapter resolves `deps.edn`, `project.clj`, and `shadow-cljs.edn` boundaries, emits namespace modules and static require/use/macro relationships, preserves `.cljc` platform and polymorphic metadata, and reports dynamic loading or malformed forms as recoverable evidence. The adapters never import or execute the analyzed project; unresolved or dynamic behavior remains visible rather than becoming a fabricated local relationship.

The external Python pilot lives as a separately launched plugin deployment,
uses Python ast and safe configuration readers, and has been verified against
the in-process Python adapter for semantic parity. It is explicitly selected
through a local descriptor; it is not a new language, an implicit project
plugin, or a replacement for the built-in adapter.

This pilot is a compatibility proof for the process boundary, not the final
distribution model. The future compiled-plugin frontier replaces its
script/interpreter deployment with platform-specific analyzer executables that
reuse the existing implementations. The multi-analyzer frontier then plans
multiple such jobs and merges their results without adding language-specific
branches to the model or viewer.

### Architecture model

Owns stable opaque module identity, language and project metadata, explicit hierarchy segments, source references, relationships, evidence, confidence, tags, analysis diagnostics, and derived cycle/layer projections. Hierarchy is structural and separate from semantic dependency edges. The model must represent one package containing many files and must not assume dot-separated names.

Canonical model data types and graph/projection data remain in
`internal/model`. The canonicalization capability is `internal/model/canonical`:
it owns normalization, collection merging, conflict/recovery diagnostics,
validation, deterministic identity/content checks, and orchestration of graph
derivation. This keeps model data from depending on its own canonicalization
package while giving CLI, viewer, and export callers one explicit validation
boundary.

### Graph and view preparation

Owns relationship normalization, aggregation, abstraction classification, cycle detection, hierarchical projection, layer assignment, and layout inputs. It consumes only the neutral model. When hierarchy aggregation creates a non-cycle group self-loop, this boundary provides an internal-relationship summary instead of asking a renderer to imply a cycle.

The viewer scene projection is isolated in `internal/viewer/scene` and returns
the versioned renderer-neutral scene contract. Route geometry is isolated in
`internal/routing`, where ELK sections, deterministic orthogonal routes, and
the reserved future cubic segment representation meet before browser/SVG
serialization. A new route strategy therefore does not require changes to
graph orchestration or path rendering.

### Presentation and export

The viewer owns a local web presentation, renderer-neutral scene/view state, interaction, navigation, progressive disclosure, reference-boundary visibility, import/evidence inspection, source inspection, tooltips, layout/session state, configurable ELK presentation settings, and accessibility concerns. The default overview is local-first; non-local references remain canonical but are hidden, aggregated, or expanded by view policy. The locally served ELK/elkjs layered adapter now normalizes valid `SPLINES` sections into cubic routes; the browser serializer preserves arrowheads, labels, and the established self-loop route, while malformed spline data falls back to deterministic orthogonal geometry. A deterministic layer-based layout remains the replaceable fallback; neither layout choice changes relationship meaning. Self-contained HTML embeds the effective profile/catalog and pinned ELK runtime, recalculates its scene in the browser, and uses the same route serializer. The browser Download SVG action serializes the current canvas. Go headless SVG remains explicitly deterministic orthogonal and does not consume project layout configuration. Neither presentation nor export knows how a source language is parsed.

### Project configuration and layout settings

The local host resolves the nearest versioned `.archview.json` from the selected target directory through its ancestors toward the filesystem root. A discovered file is selected as one complete profile; v1 does not merge multiple files. The viewer settings surface obtains a typed/catalogued list of the pinned ELK algorithms and options, validates applicability, and applies profiles explicitly to the current scene. Applying a profile may recalculate node positions and edge routes and clears session manual positions for that hierarchy path, but it cannot mutate canonical model semantics.

Configuration writes distinguish two actions. Ordinary `Save` has no destination input and atomically overwrites exactly the active discovered `.archview.json`; it never creates a new file or copies settings to the analyzed project root, and it requires `Save As` when no active file exists. `Save As` is the only operation that accepts a user-selected custom destination folder; it writes the fixed `.archview.json` filename there atomically after explicit confirmation and makes that path active for the current session. Model-only sessions can use session settings but cannot persist a project file. Analyzer options, viewport state, and manual positions are separate from this configuration. Project-backed HTML analysis embeds the discovered profile for browser use; raw model-only export uses defaults because it has no source-root discovery context. Go static SVG continues to use its explicit deterministic contract.

The specified future analysis-assignment extension adds a separately owned
`analysis` section to the versioned project configuration. It maps
repository-relative folders or project roots to stable logical analyzer IDs and
remains distinct from layout options. The nearest configuration file remains
the one complete file; the deepest matching assignment wins, explicit CLI
selection overrides it, and automatic detection is the fallback. Invalid
nearest configuration or unavailable analyzer IDs are surfaced. The viewer can
expose cached individual scopes and a combined view through a dropdown, but it
does not decide analyzer semantics.

## Specified future capability decisions

The compiled-distribution, multi-analyzer, assignment, and advanced-renderer
capabilities are now specified with exact schemas, contracts, scenarios, and
readiness reviews. They are not implemented yet; compiled distribution is in
ordered issue delivery through 034–038, while the other frontiers still
require issue slicing, implementation, and verification:

- Compiled analyzers are the production implementation for stable logical IDs.
  The release build compiles every analyzer and assembles its platform-specific
  executable and descriptor under the application-managed analyzer tree and
  `analyzers/index.json`. Production discovers only checksum-verified packaged
  artifacts; explicit descriptors and in-process adapters are developer/test or
  explicit migration overrides, never silent fallback. The target repository is
  never scanned for executable plugins; each operation launches one isolated
  process.
- Opening a repository discovers marker-driven nested project roots within
  bounded exclusions. A strong manifest owns its subtree unless a nested
  manifest or assignment creates a nested scope. Different language jobs may
  run through a configurable bounded worker pool, with a recommended default of
  four and a hard cap.
- Aggregation produces one language-neutral model backed by per-job results.
  Global identity includes relative project root, logical analyzer ID, and
  local observation ID. Only analyzer-reported relationships are retained; a
  failed job yields a partial model when usable results remain.
- `.archview.json` gains a separate `analysis` section for repository-relative
  assignments. The application exposes `All` and individual project/analyzer
  scopes from cached results without re-running analysis when only the active
  scope changes.
- Advanced ELK support is a renderer-only, staged extension: route/output
  features first, structural scene features second, and broader options only
  with concrete renderer support. Browser, embedded HTML, and browser SVG share
  the renderer-neutral geometry; Go static SVG remains deterministic
  orthogonal unless separately specified.

The layout catalog, profile validation, option-handler registry, session state,
and persistence are grouped under `internal/viewer/layout`; HTTP handlers only
translate transport envelopes. The live browser uses a small native-ES-module
entrypoint and focused modules. Exported HTML uses an embedded esbuild bundle
with external-import rejection, so no module, script, or stylesheet dependency
escapes the exported artifact.

## Data flow

1. The CLI and project configuration identify a repository, project roots, and
   analyzer assignments or automatic-detection candidates.
2. The plugin manager validates descriptors, discovers compatible analyzers,
   and creates a bounded set of analyzer jobs.
3. The plugin manager runs applicable jobs, in-process during migration or
   through compiled external executables using the versioned process boundary.
4. Each analyzer returns modules, relationships, evidence, diagnostics, and
   source references; the host merges results with collision-safe identity and
   provenance while retaining partial failures.
5. The canonical model capability normalizes observations, validates identity
   and relationship integrity, and asks the model graph capability to derive
   cycles/layers.
6. The scene capability projects the model into the renderer-neutral interactive contract.
7. The local host resolves project layout configuration and exposes the effective profile/catalog to the viewer.
8. The routing/layout adapters calculate positions and route sections; the renderer-neutral viewer or exporter serializes the resulting view.

## Shared policies

- Analysis is read-only and does not execute the target application.
- Paths and source evidence must remain traceable to the analyzed repository.
- Unresolved or dynamic dependencies are reported with confidence or diagnostics.
- Non-local references are retained in the canonical model; viewer/export projections decide whether they are hidden, aggregated, or expanded, while import evidence remains available in list/details form.
- External and generated code are filtered by explicit policy, not silently discarded.
- Language-specific concepts such as interfaces, protocols, or abstract classes are represented as metadata and relation semantics, not hard-coded into the core.
- Saved models use a versioned, language-neutral JSON interchange format. Visual artifacts are deterministic projections of the same model/view contract; source contents are not embedded by default.
- `.archview.json` is versioned project configuration with separate `layout` and
  future `analysis` ownership. Nearest-ancestor discovery is deterministic and
  invalid nearest configuration is surfaced rather than silently bypassed.
  Ordinary `Save` is explicit, active-file-only, and atomic; only explicit
  `Save As` may select a custom destination folder, with the fixed filename and
  atomic write preserved.
- Analyzer options and layout options have separate ownership and schemas; layout configuration cannot alter canonical model facts or source analysis.
- The separately owned `analysis` section maps repository-relative folders or
  project roots to stable analyzer IDs. It preserves layout/analysis
  separation, deterministic precedence, and explicit built-in versus compiled
  external selection.
- Multi-analyzer execution must use collision-safe identities and preserve
  analyzer/project provenance; one failed job must not erase successful or
  partial results from other jobs.

## Verification

The root policy is `when-supported`, with justified deferrals required. The architecture should be verified at analyzer boundaries, model normalization, graph algorithms, viewer integration, export stability, and end-to-end repository analysis when those harnesses exist.

## Implementation sequence

The first architectural slice is the neutral model and plugin contract. The
first product slice is Go package analysis connected to headless output and a
visible viewer result. The completed refactor sequence then isolates routing,
browser composition/bundling, scene projection, the Go analyzer pipeline, the
ELK option registry, and canonical model normalization. Issue 017 adds the
first in-process Python project/module-discovery adapter; issue 018 extends it
through static relationships, uncertainty, evidence, and partial results, and
issue 019 completes the shared visible journey after automated verification
and explicit visual approval. The approved TypeScript slice has completed
issues 020–022 for project discovery, static dependencies, and the shared
visible journey. Completed issues 023–025 add the Rust Cargo boundary,
module/evidence discovery, static relationships, uncertainty, and canonical
output path. Issues 026–029 add the in-process Clojure project/namespace
adapter, static relationships, platform/polymorphic and safety metadata, and
shared visible journey through the existing neutral path. Every adapter follows
the same `analysis.Analyzer` contract and registers at the composition root;
adding one does not modify host orchestration.
Issues 030–033 complete the first external runtime slice: published protocol
and descriptor schemas, the process-backed Analyzer adapter, external Python
parity, and explicit CLI/shared-path integration. The process adapter remains
behind the same host/model/viewer/export boundary and does not add external
protocol fields to the canonical model.

The compiled external analyzer distribution is now in ordered delivery through
issues 034–038 after application synthesis. The remaining specified frontiers
are [multi-analyzer project orchestration](../../.okf/capabilities/analyze-source/plugin-runtime/multi-analyzer-orchestration.md), [project analyzer assignments and view selection](../../.okf/capabilities/analyze-source/plugin-runtime/project-analyzer-assignments.md), and [advanced ELK renderer support](../../.okf/capabilities/explore-architecture/advanced-elk-renderer-support.md). Their exact architecture specifications are complete; later delivery issue slicing follows the application synthesis gate.
The 2026-08-28 brownfield audit also confirmed that the existing Go analyzer,
canonical model generation, and export/automation boundaries exhaust their
current exact specifications. Their planning nodes are implemented. This is a
status correction based on completed delivery and passing verification, not an
architecture change.
Issue 009 implements the bounded supported node/edge ELK option tranche at the
layout adapter boundary without changing the renderer-neutral scene or
configuration schemas. Issue 016 activates the reserved cubic route
representation for supported ELK spline output; automated verification and
visual review are complete. The current Explore scope is implemented. Future
renderer extensions are tracked in the [Advanced ELK renderer support future
work register](explore-architecture/advanced-elk-renderer-support/future-work.md).

## Residual implementation decisions

- Benchmarking and tuning frontend/rendering thresholds.
- Reference-boundary aggregation, import-list density, and session-scoped layout behavior remain verification/tuning concerns. Broader target-specific ELK option support is tracked in the [Advanced ELK renderer support future-work register](explore-architecture/advanced-elk-renderer-support/future-work.md).
- Implementation of the specified compiled plugin artifact packaging,
  platform selection, checksum trust policy, and build targets.
- Implementation of the specified multi-project/multi-analyzer scheduling,
  aggregate schema, provenance, progress, and resource budgets.
- Implementation of the specified folder-to-analyzer assignment schema,
  validation, cache keys, and application dropdown/API behavior.
- Implementation of the specified advanced ELK scene/route fields, feature
  fixtures, and cross-surface export parity.
- Benchmark-driven tuning of process frame/stderr limits, timeout defaults,
  and platform-specific external-plugin launch behavior.
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
- [Compiled external analyzer distribution](../../.okf/capabilities/analyze-source/plugin-runtime/compiled-external-analyzer-distribution.md)
- [Multi-analyzer project orchestration](../../.okf/capabilities/analyze-source/plugin-runtime/multi-analyzer-orchestration.md)
- [Project analyzer assignments and view selection](../../.okf/capabilities/analyze-source/plugin-runtime/project-analyzer-assignments.md)
- [Generate architecture models](../../.okf/capabilities/generate-models.md)
- [Explore and inspect architecture](../../.okf/capabilities/explore-architecture.md)
- [Advanced ELK renderer support](explore-architecture/advanced-elk-renderer-support/future-work.md)
- [Export and automate](../../.okf/capabilities/export-and-automate.md)
- [Application PRD](../prd.md)
