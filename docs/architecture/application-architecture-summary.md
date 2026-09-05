# Arch View application architecture summary

Advanced ELK implementation follows the approved
[shared feature-registry contract](explore-architecture/advanced-elk-renderer-support/delivery-contract.md).
Architecture and OKF reuse settings, feature negotiation, browser ELK execution
and geometry functions, with separate persisted profiles. Delivery starts with
useful settings and registry/persistence, followed by edges, ports and compound
geometry. Every stage has an automated and human visual gate. No server-side
ELK service or parallel settings endpoint is introduced.
The registry/settings and advanced-edge stages are approved and archived.
Presentation ports are implemented through the shared geometry path and await
human visual approval; compound geometry remains blocked.

## Status

The configurable OKF knowledge-view capability is a first-class root-level,
implemented capability through verified issues 064–071; its exact node-scoped
reference set is linked below and its shared boundaries are summarized here.

This is the application-level architecture baseline. It describes the target Go architecture and distinguishes it from the Clojure reference implementation. Capability territories are now specified, implemented, or explicitly tracked for implementation; the verified in-process Clojure adapter and opt-in external Python process both reach the existing analyzer, model, viewer, and export paths without adding language-specific consumer branches. The compiled external analyzer distribution is implemented as the trusted production release path. Multi-analyzer issues 039–043, project analyzer assignment issues 044–047, and source-index issues 048–052 are verified, archived, and visually approved; the advanced ELK renderer child remains the specified plugin/viewer frontier.
The Code quality and code intelligence capability is a pure structural-child
roll-up with effective state `implemented`, the minimum of its three children.
Its Source facts and symbol index child is implemented through issues 048–052
with final visual approval recorded; Deterministic quality checks is implemented
through issues 053–063 and 077–078; and Live analysis/MCP is implemented through
verified issues 064–079, including issue 076's final cross-analyzer and human
product approval. The parent has no standalone PRD or implementation slice.

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
  +--> project configuration resolver -> analyzer assignments + source-scope policy + viewer layout profile
        |
        +--> local web viewer
        +--> versioned JSON, SVG, HTML, and CI outputs

        +--> code facts and symbol index
                 |
                 +--> deterministic quality findings
                 +--> live analysis snapshots and MCP queries

OKF bundle discovery
        |
        v
lossless OKF index -> selected view profile -> renderer-neutral scene
                                      |
                                      +--> shared ELK-backed viewer
```

## Current reference architecture

The upstream [unclebob/arch-view reference implementation](https://github.com/unclebob/arch-view) couples its command entrypoint to a Clojure source scanner, graph construction, layer assignment, and a Quil/Swing viewer. Its source analyzer reads Clojure forms, recognizes Clojure-family extensions, extracts `:require` dependencies, and marks selected Clojure abstractions. Its graph and layering logic are the main reusable ideas.

## Target boundaries

### OKF knowledge-view boundary

The configurable OKF knowledge-view capability is a first-class root-level
boundary alongside the architecture capabilities. It consumes one validated
OKF directory bundle at a time without converting it into the canonical
architecture model. The exact source and behavior are defined in the
[node-scoped PRD](okf-knowledge-views/prd.md), [domain model](okf-knowledge-views/canonical-domain-model.md),
[use-case model](okf-knowledge-views/canonical-use-cases.md),
[canonical contract](okf-knowledge-views/canonical-api-cli-contract.md), and
[acceptance scenarios](okf-knowledge-views/acceptance-scenarios.md).

The architecture boundary is:

1. project discovery/validation and lossless indexing preserve paths,
   frontmatter, Markdown, links, unknown metadata, and provenance;
2. profile resolution and in-process registry strategies interpret the index
   into separate containment and semantic-link layers;
3. projection produces a renderer-neutral scene with presentation decisions,
   legend data, and diagnostics;
4. the existing viewer and ELK layout/rendering path consume that scene.

The optional OKF section of the nearest project configuration owns graph/profile
bindings and project-local profiles while preserving unrelated layout and
analysis settings. Configuration writes validate before atomic replacement.
Source bundles are read-only; local link resolution, Markdown rendering, rule
execution, and application hard caps are security/safety boundaries. Existing
architecture-view behavior remains unchanged when the OKF surface is unused.

The local OKF application boundary composes focused catalog, session/navigation,
inspection, profile-lifecycle, and diagnostic/extension services. The exported
`application.Service` supplies their existing API to the viewer; it does not
implement those use cases itself. The services retain one synchronized
publication state so refresh and configuration writes invalidate sessions
atomically. Profile validation, projection construction, concept inspection,
configuration writing, and report assembly keep their existing focused helpers.
The shared state is internal and does not expose new transport operations.

### Confirmed analyzer boundary

The current v1 analyzer path receives a repository or project root plus
analysis options, selects one language analyzer, and returns validated modules,
static dependency relationships, source evidence, metadata, and diagnostics. It
does not assign layers, render diagrams, or export files. The implemented
multi-analyzer path plans a set of project-root/analyzer jobs, runs applicable
jobs concurrently, and merges their validated language-neutral observations
before model normalization. The host also resolves the validated
invocation-root source-scope policy and passes each job its effective source
set; source filtering does not change marker-driven root discovery.

The plugin runtime and language adapters are specified around this contract. The
current deployment uses in-process Go, Python, TypeScript, Rust, and Clojure
analyzers, an explicitly loaded external Python process, and trusted packaged
compiled analyzer executables for release execution. The compiled executables
reuse the same analyzer implementations and retain the same logical analyzer
IDs; in-process adapters remain explicit development or migration paths. Each
adapter owns project-boundary and static-resolution rules, while the host owns
assignment resolution, marker-driven job planning, option precedence,
validation, normalization, merge identity, and process safety. Python,
TypeScript, Rust, and Clojure uncertainty is returned as evidence, confidence,
or diagnostics rather than fabricated relationships.

The first graph uses package or module nodes. Files remain attached evidence. Project-local modules are shown by default. Tests, generated code, vendor directories, caches, build outputs, and directories named `external` are excluded by default. Unresolved dependencies produce partial results with diagnostics.

### CLI and orchestration

Owns command parsing, project selection, analyzer-job planning, analyzer
assignment and source-scope configuration, progress, and error reporting. It does not parse
language syntax or calculate layout. Presentation layout configuration is
resolved by the local host/viewer boundary and remains separate from analyzer
options and assignment semantics.

### Plugin manager

Owns analyzer registration, project detection, capability negotiation,
assignment and source-scope policy resolution, job planning, result validation, and plugin version
compatibility. Built-in analyzers remain in-process during migration and only as
an explicit fallback. The implemented release distribution is assembled by a
reproducible root build/make target under an application-managed
`analyzers/<id>/<platform>/` tree and registers compiled external executables
from trusted descriptors. The
host validates manifest/hello agreement, launches each job with argv rather than
a shell, reserves stdout for versioned NDJSON, treats stderr as bounded logs,
and owns cancellation, timeout, bounded concurrency, and child cleanup. One
detect or analyze job uses one external process.

The distribution boundary also owns application release identity. Release
builds inject the semantic application version, source commit, build timestamp,
and build ID into the host executable and publish a root `release.json` beside
the trusted `analyzers/index.json`. The release manifest binds the host and
analyzer index to their digests and target platform; analyzer versions and
protocol/schema versions remain independently versioned. A `vMAJOR.MINOR.PATCH`
tag is the release trigger, while successful artifact publication is the
release completion point.

### Language analyzers

Each analyzer owns source discovery, syntax parsing, import or dependency resolution, language-specific classification, source locations, and diagnostics. An analyzer returns data, not UI elements or layout coordinates.

The adapters have exact language-specific contracts; parser/library choices may vary behind those contracts. The in-process Python adapter now extracts absolute, package, re-export, and relative imports through a conservative static pass, resolves only filesystem-proven local modules under effective source roots, and emits standard-library, external, unresolved, conditional, and dynamic references with source evidence, confidence, and recoverable diagnostics. The in-process Rust adapter reads one Cargo crate and statically discovers reachable crate/module hierarchy, `use`/`pub use` relationships, dependency declarations, source evidence, and cfg/macro/generated uncertainty without executing Cargo or target code. The in-process Clojure adapter resolves `deps.edn`, `project.clj`, and `shadow-cljs.edn` boundaries, emits namespace modules and static require/use/macro relationships, preserves `.cljc` platform and polymorphic metadata, and reports dynamic loading or malformed forms as recoverable evidence. The adapters never import or execute the analyzed project; unresolved or dynamic behavior remains visible rather than becoming a fabricated local relationship.

The external Python pilot lives as a separately launched plugin deployment,
uses Python ast and safe configuration readers, and has been verified against
the in-process Python adapter for semantic parity. It is explicitly selected
through a local descriptor; it is not a new language, an implicit project
plugin, or a replacement for the built-in adapter.

This pilot is a compatibility proof for the process boundary, while the
implemented compiled-plugin distribution is the production release model. The
implemented multi-analyzer path plans multiple packaged or in-process jobs and
merges their results without adding language-specific branches to the model or
viewer; cached aggregate and individual scope projections are available to the
local viewer.

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

The implemented analysis-assignment extension adds a separately owned
`analysis` section to the versioned project configuration. It maps
repository-relative folders or project roots to stable logical analyzer IDs and
defines global exclusion globs plus analyzer-ID-scoped include globs relative
to the invocation root. It remains distinct from layout options. The nearest
configuration file remains the one complete file; the deepest matching
assignment wins, explicit CLI selection overrides it, and automatic detection
is the fallback. Source filtering occurs after root discovery, with fixed,
nested-root, and configured exclusions winning over includes. Invalid nearest
configuration or unavailable analyzer IDs are surfaced. The viewer can expose
cached individual scopes and a combined view through a dropdown, but it does
not decide analyzer semantics.

### Code quality and code intelligence

This boundary is represented by a pure structural-child roll-up. It consumes
analyzer observations and canonical model facts without changing the
architecture graph's default module-level granularity; the source-facts child
adds an optional, independently
versioned `SourceIndex` sibling beside modules, relationships, source
references, diagnostics, and derived data. It retains authoritative
per-scope snapshots containing files, intrinsic size facts, named declarations,
documentation candidates, hash-linked spans, provenance, containment, and
typed extension/metric slots. A combined result retains per-scope authority
and may add a deterministic projection without inferring cross-scope
relationships.

Registered language extractors own declaration, visibility, documentation, and
semantic uncertainty. The core source-index service owns validation, opaque ID
assignment, coverage, canonical ordering, and digest assembly. The deterministic
quality child evaluates versioned metric/rule strategies and emits structured
exact findings or explicitly labeled advisory SOLID signals with rule IDs,
thresholds, severity, revision, coverage, and source evidence. Its quality
profile is independent from analyzer, assignment, layout, and live settings.
Human-facing consumers may summarize scope-safe file line-threshold findings and
filter to affected files, while the quality report remains the sole source of
threshold truth. The Go analyzer publishes the syntax-observable structural
facts required by the SOLID signal rules. Baselines are separate validated
quality documents. The managed CLI workflow and live/MCP policy service resolve
one canonical baseline from the selected profile's exact ID/revision, merge
reviewed active observed findings, advance the revision, update the profile
reference, and reject missing/invalid/ambiguous or conflicting policy states.
The legacy standalone `arch-view quality baseline` command remains for
compatibility; it does not silently become a profile reference.

The implemented live-analysis child owns folder watching, event coalescing,
request-time freshness reconciliation, conservative invalidation, stable-input
verification, cache identity, and atomic revision replacement. Viewer,
CLI/export, and MCP clients consume the same immutable snapshot/query envelope.
MCP is an analyzer-neutral transport/query adapter over shared services; it
delegates quality evaluation, profile, baseline read/selection/append, and
baseline semantics to the quality child. Read operations are the default,
while analysis and quality-policy operations require separate permissions.
Stdio is the default local transport and network transport is explicit. The
design must not execute target applications or present subjective architectural
approval, semantic comment quality, or SOLID heuristics as proven facts.

The local viewer treats the source index as a progressive-disclosure read model.
Graph startup requests the model with `include_source_index=false`, so the
right-hand card stays a compact summary and does not wait for the complete
source index. Selecting a node or group opens a same-tab, history-backed
inspection route with Overview, Structure, Files, Symbols, Dependencies,
Evidence, and Technical details sections. Files and symbols use bounded
25-item queries with local filters and explicit `module_id`/`file_id`
containment filters; membership is resolved only from declared containment or
declaration relations. File-only provenance is distinct from line locations,
source excerpts are opt-in and read-only, and IDs, hashes, provider versions,
and snapshot data remain secondary technical details. Embedded exports apply
the same bounded read model locally.

## Capability decisions and remaining frontier

The multi-analyzer, assignment, and advanced-renderer capabilities have exact
schemas, contracts, scenarios, and readiness reviews. The compiled-distribution
capability is implemented through issues 034–038. Multi-analyzer issues
039–043 are verified, archived, and visually approved; those issues consume the
resolved source-scope policy. The assignment capability is implemented through
verified, archived, and visually approved issues 044–047. The advanced-renderer
frontier still requires later issue slicing, implementation, and verification:

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
  assignments and invocation-root source-scope policy. Global `exclude` globs
  and analyzer-ID-scoped `include` globs are validated as a deterministic
  explicit subset, applied after root discovery, and included in cache identity.
  The application exposes `All` and individual project/analyzer scopes from
  cached results without re-running analysis when only the active scope changes.
- Configurable OKF knowledge views are a separate first-class root-level
  implemented boundary, verified through issues 064–071 and user-approved on
  2026-09-04. Its exact node-scoped PRD, domain/use-case models,
  canonical contract, acceptance scenarios, and readiness review define a
  lossless bundle index, profile-driven projection, renderer-neutral scene,
  shared ELK layout, progressive inspection, and project-local persistence.
  OKF source/profile state remains separate from the architecture model.
  Dialog/form styles and current-canvas SVG serialization are shared; the SVG
  serializer captures rendered presentation instead of copying scene CSS.
- Advanced ELK support is a renderer-only, staged extension: route/output
  features first, structural scene features second, and broader options only
  with concrete renderer support. Browser, embedded HTML, and browser SVG share
  the renderer-neutral geometry; Go static SVG remains deterministic
  orthogonal unless separately specified.
- The Code quality and code intelligence roll-up is implemented.
  Its Source facts and symbol index child is implemented through issues 048–052
  with final visual approval recorded; Deterministic quality checks is
  implemented through issues 053–063 and 077–078, including issue 063's
  visual review and the managed baseline lifecycle.
  The Live analysis and MCP capability is `implemented` after issue 076's
  final product gate.
  Issues 064–079 implement and verify exact records, provenance, scope
  isolation, analyzer-neutral capability coverage, request-time freshness
  verification, versioned rules, immutable revisions, bounded queries,
  quality-service delegation, canonical baseline read/selection/append,
  shared CLI/viewer/MCP transports, and
  compatibility/permission rules. The aggregate parent
  has no standalone implementation issues; delivery is owned by the children.

The layout catalog, profile validation, option-handler registry, session state,
and persistence are grouped under `internal/viewer/layout`; HTTP handlers only
translate transport envelopes. The live browser uses a small native-ES-module
entrypoint and focused modules. Exported HTML uses an embedded esbuild bundle
with external-import rejection, so no module, script, or stylesheet dependency
escapes the exported artifact.

## Data flow

1. The CLI and project configuration identify a repository, project roots,
   analyzer assignments or automatic-detection candidates, and the validated
   invocation-root source-scope policy.
2. The plugin manager validates descriptors and source-scope patterns,
   discovers compatible analyzers, and creates a bounded set of analyzer jobs.
3. The plugin manager applies each job's effective source scope and runs
   applicable jobs, in-process during migration or
   through compiled external executables using the versioned process boundary.
4. Each analyzer returns modules, relationships, evidence, diagnostics, and
   source references; the host merges results with collision-safe identity and
   provenance while retaining partial failures.
5. The source-index service enumerates each resolved eligible scope, invokes
   registered language extractors, and attaches optional per-scope source
   snapshots with files, symbols, documentation, spans, provenance, coverage,
   and deterministic digest. Combined runs retain those snapshots and may
   construct a derived projection.
6. The canonical model capability normalizes observations, validates identity
   and relationship integrity, and asks the model graph capability to derive
   cycles/layers.
7. The scene capability projects the model into the renderer-neutral interactive contract.
8. The local host resolves project layout configuration and exposes the effective profile/catalog to the viewer.
9. The routing/layout adapters calculate positions and route sections; the renderer-neutral viewer or exporter serializes the resulting view.
10. The graph viewer requests only the model needed for scene rendering. Its
    compact summary card exposes human-scale counts and status; its separate
    accessible scene list preserves keyboard navigation and search.
11. The same-tab inspection controller reads bounded files, symbols,
    documentation, and entity evidence on demand, caches section/filter state,
    and exposes technical identifiers only in the secondary Technical details
    section. Embedded exports answer these queries from their local source index.
12. Language adapters and the source index provide source facts, symbols,
    documentation, and metrics to quality and query consumers.
13. The quality engine evaluates configured deterministic rules and stores
    revision-aware findings with source evidence. Its profile reference loads
    one exact canonical baseline, and the managed append path merges reviewed
    active observed findings with revision/conflict checks.
14. The live snapshot service watches configured folders, reconciles source
    state for strict requests, updates affected multi-analyzer analysis state,
    and exposes the same compact facts to viewer, CLI/export, and MCP. Quality
    requests delegate to the shared quality catalog/evaluation/policy services,
    including bounded baseline reads and request-scoped selection.

### OKF knowledge-view flow

1. The local host recursively discovers eligible OKF bundle directories under
   the project tree, applies standard dependency/generated-output exclusions,
   orders them deterministically, and applies the selected single-graph
   configuration.
2. The OKF reader builds a lossless, namespaced index of concept documents,
   preserving paths, frontmatter, Markdown body, links, and unknown metadata.
3. The selected view profile projects hierarchy and semantic relationships as
   separate layers, derives facets and display roles, and applies explicit
   roll-up policy.
4. The projection is translated into the shared renderer-neutral scene
   contract; ELK calculates layout and the existing viewer handles navigation,
   subtree drill-down, accessibility, and progressive inspection.

## Shared policies

- OKF consumption is lossless and read-only: the reader preserves source
  documents and unknown metadata, while profiles own interpretation and
  presentation. Structural containment and semantic links are not silently
  merged.
- OKF profile mappings must use a bounded, portable vocabulary rather than
  arbitrary executable code. Derived state, facet values, display roles,
  roll-up behavior, and styles remain explainable from source data plus the
  selected profile.
- Analysis is read-only and does not execute the target application.
- Paths and source evidence must remain traceable to the analyzed repository.
- Unresolved or dynamic dependencies are reported with confidence or diagnostics.
- Non-local references are retained in the canonical model; viewer/export projections decide whether they are hidden, aggregated, or expanded, while import evidence remains available in list/details form.
- External and generated code are filtered by explicit policy, not silently discarded.
- Language-specific concepts such as interfaces, protocols, or abstract classes are represented as metadata and relation semantics, not hard-coded into the core.
- Saved models use a versioned, language-neutral JSON interchange format. Visual artifacts are deterministic projections of the same model/view contract; source contents are not embedded by default.
- The optional `SourceIndex` is independently versioned, scope-first, and
  additive. Files, symbols, documentation, occurrences, relations, and
  metrics carry explicit provenance and coverage; unsupported/unknown facts are
  never fabricated as absence. Full source content is not embedded by default.
- The graph viewer does not eagerly download the complete source index. Its
  inspection view uses bounded query pages, explicit containment/declaration
  membership, readable source locations, opt-in bounded excerpts, and a
  secondary technical-details disclosure.
- A project-backed viewer discovers separate `quality-profiles/*.json`
  documents through `GET /v1/quality/profiles`; choosing one invokes the
  in-process `POST /v1/quality/evaluate` bridge over the loaded model or cached
  scope. The viewer can also load the complete registered rule catalog through
  `GET /v1/quality/rules` and apply an enabled/disabled selection through the
  optional session-only `rule_bindings` request field. `Apply & run checks`
  remains session-only; explicit `PUT /v1/quality/profiles/save` and
  `PUT /v1/quality/profiles/save-as` actions validate and persist an existing
  or newly named profile. Quality evaluation remains in-process and does not
  rerun an analyzer. Successful evaluations are cached by scope so the
  same-session inspection route retains the report.
- The viewer's quality report may create a separate baseline for its existing
  project-backed flow. The managed CLI and MCP append workflow is the canonical
  repeatable path: it keeps one baseline per profile, merges reviewed active
  observed findings, advances the revision, updates the profile reference, and
  re-evaluates only when the caller requests the next run.
- Live/MCP may use the same profile/rule catalog and evaluation services with
  temporary non-persisted rule bindings and request-scoped baseline selection.
  Baseline reads are bounded; profile saves and baseline appends are separate
  allowlisted policy operations, require explicit authorization, and are
  rejected for stale or incompatible reports. MCP never implements a second
  quality evaluator or baseline matcher.
- Source-index IDs are opaque and snapshot-local; canonical spans use
  hash-linked UTF-8 byte coordinates. Registered extractors own language
  semantics, while the core owns validation and deterministic assembly.
- `.archview.json` is versioned project configuration with separate `layout` and
  `analysis` ownership. Nearest-ancestor discovery is deterministic and
  invalid nearest configuration is surfaced rather than silently bypassed.
  Ordinary `Save` is explicit, active-file-only, and atomic; only explicit
  `Save As` may select a custom destination folder, with the fixed filename and
  atomic write preserved.
- Analyzer options and layout options have separate ownership and schemas; layout configuration cannot alter canonical model facts or source analysis.
- The separately owned `analysis` section maps repository-relative folders or
  project roots to stable analyzer IDs and carries the invocation-root source
  policy. It preserves layout/analysis separation, deterministic precedence,
  explicit built-in versus compiled external selection, and the rule that
  configured exclusions cannot be re-included.
- Multi-analyzer execution must use collision-safe identities and preserve
  analyzer/project provenance; one failed job must not erase successful or
  partial results from other jobs.
- Given the same source, analyzer implementation, and quality configuration,
  deterministic metrics and findings must be reproducible. Findings remain
  separate from analyzer diagnostics, and exact facts must be distinguishable
  from heuristic or advisory signals.
- Live snapshots replace atomically, carry explicit revisions, input
  fingerprints, reconciliation status, and freshness state, and keep source
  paths root-safe. Strict MCP queries verify current input before returning a
  `current` result; read retrieval is compact and default, while quality-policy
  writes are explicit and audited. Source edits require a separate downstream
  action.
- Quality configuration has separate ownership from analyzer options,
  project assignments, and viewer layout semantics.

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

The compiled external analyzer distribution is implemented through issues
034–038 after application synthesis. The project analyzer assignments and view
selection child is implemented through verified, archived, and visually
approved issues 044–047. The remaining specified frontier is [advanced ELK
renderer support](../../.okf/capabilities/explore-architecture/advanced-elk-renderer-support.md);
Live analysis/MCP is implemented. Advanced ELK is the next eligible visible
delivery frontier.
The multi-analyzer child is implemented through issues 039–043.
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
The source-facts/index contract, Go-first extractor, scope-safe projection,
bounded query boundary, and module viewer are implemented through issues
048–052; the final visual gate is approved. Deterministic quality issues
053–063 and managed baseline issues 077–078 are implemented and verified.
Live analysis/MCP issues 064–079 are implemented and verified, including issue
076's cross-analyzer conformance and final product approval.
The Configurable OKF knowledge-view capability is implemented through the dated
`20260903-064` through `20260903-071` delivery records and its verified visual
review.

## Residual implementation decisions

- Benchmarking and tuning frontend/rendering thresholds.
- Reference-boundary aggregation, import-list density, and session-scoped layout behavior remain verification/tuning concerns. Broader target-specific ELK option support is tracked in the [Advanced ELK renderer support future-work register](explore-architecture/advanced-elk-renderer-support/future-work.md).
- The implemented multi-project/multi-analyzer viewer path has passed issue
  043's declared visual-review gate.
- Implementation of the specified advanced ELK scene/route fields, feature
  fixtures, and cross-surface export parity.
- Benchmark-driven tuning of process frame/stderr limits, timeout defaults,
  and platform-specific external-plugin launch behavior.
- Linux amd64 and Darwin arm64 packaged-analyzer execution remain deferred
  verification surfaces until matching runners or toolchains are available.
- Future call-graph/type-level relation capabilities.
- Broader language coverage and metric-producing extensions for the implemented
  source-index module-inspection projection remain future work.
- Broader language coverage for quality metrics and SOLID structural facts.
- Platform-specific tuning, long-term snapshot retention, and broader MCP
  client packaging remain future work after the verified live watcher/MCP
  snapshot, request-time reconciliation, analyzer-neutral query, quality
  delegation, canonical baseline read/selection/append, policy-permission,
  and transport behavior delivered by issues 064–075 and 079.
- Conservative semantic resolution for calls/implements and confidence
  categories.

These are implementation and extension risks, not blockers to the specified v1 scope.

## Exact specification sources

- [Analysis parent specification](analyze-source/prd.md) and [canonical contract](analyze-source/canonical-api-cli-contract.md), with readiness-reviewed language/plugin child specifications.
- [Canonical model specification](generate-models/prd.md) and [model contract](generate-models/canonical-api-cli-contract.md).
- [Viewer specification](explore-architecture/prd.md) and [renderer-neutral scene contract](explore-architecture/canonical-api-cli-contract.md).
- [Export specification](export-and-automate/prd.md) and [JSON/HTML/SVG contract](export-and-automate/canonical-api-cli-contract.md).
- [OKF knowledge-view discovery notes](okf-knowledge-views/discovery-notes.md), [gap analysis](okf-knowledge-views/requirements-gap-analysis.md), and [glossary](okf-knowledge-views/domain-glossary.md).
- [OKF knowledge-view PRD](okf-knowledge-views/prd.md), [domain model](okf-knowledge-views/canonical-domain-model.md), [use-case model](okf-knowledge-views/canonical-use-cases.md), [canonical contract](okf-knowledge-views/canonical-api-cli-contract.md), [acceptance scenarios](okf-knowledge-views/acceptance-scenarios.md), and [readiness review](okf-knowledge-views/readiness-review.md).

## Traceability

- [Arch View project concept](../../.okf/project.md)
- [Analyze source code](../../.okf/capabilities/analyze-source.md)
- [Compiled external analyzer distribution](../../.okf/capabilities/analyze-source/plugin-runtime/compiled-external-analyzer-distribution.md)
- [Multi-analyzer project orchestration](../../.okf/capabilities/analyze-source/plugin-runtime/multi-analyzer-orchestration.md)
- [Project analyzer assignments and view selection](../../.okf/capabilities/analyze-source/plugin-runtime/project-analyzer-assignments.md)
- [Generate architecture models](../../.okf/capabilities/generate-models.md)
- [Explore and inspect architecture](../../.okf/capabilities/explore-architecture.md)
- [Configurable OKF knowledge views](../../.okf/capabilities/okf-knowledge-views.md)
- [OKF knowledge-view orchestration status](okf-knowledge-views/orchestration-status.md)
- [OKF knowledge-view discovery notes](okf-knowledge-views/discovery-notes.md)
- [OKF knowledge-view requirements gap analysis](okf-knowledge-views/requirements-gap-analysis.md)
- [OKF knowledge-view glossary](okf-knowledge-views/domain-glossary.md)
- [OKF knowledge-view PRD](okf-knowledge-views/prd.md)
- [OKF knowledge-view domain model](okf-knowledge-views/canonical-domain-model.md)
- [OKF knowledge-view use cases](okf-knowledge-views/canonical-use-cases.md)
- [OKF knowledge-view contract](okf-knowledge-views/canonical-api-cli-contract.md)
- [OKF knowledge-view acceptance scenarios](okf-knowledge-views/acceptance-scenarios.md)
- [OKF knowledge-view readiness review](okf-knowledge-views/readiness-review.md)
- [Advanced ELK renderer support](explore-architecture/advanced-elk-renderer-support/future-work.md)
- [Export and automate](../../.okf/capabilities/export-and-automate.md)
- [Code quality and code intelligence](../../.okf/capabilities/code-quality-and-intelligence.md)
- [Source facts and symbol index discovery notes](code-quality-and-intelligence/source-facts-and-symbol-index/discovery-notes.md)
- [Source facts and symbol index requirements gap analysis](code-quality-and-intelligence/source-facts-and-symbol-index/requirements-gap-analysis.md)
- [Source facts and symbol index PRD](code-quality-and-intelligence/source-facts-and-symbol-index/prd.md)
- [Source facts and symbol index domain model](code-quality-and-intelligence/source-facts-and-symbol-index/canonical-domain-model.md)
- [Source facts and symbol index use cases](code-quality-and-intelligence/source-facts-and-symbol-index/canonical-use-cases.md)
- [Source facts and symbol index contract](code-quality-and-intelligence/source-facts-and-symbol-index/canonical-api-cli-contract.md)
- [Source facts and symbol index acceptance scenarios](code-quality-and-intelligence/source-facts-and-symbol-index/acceptance-scenarios.md)
- [Source facts and symbol index readiness review](code-quality-and-intelligence/source-facts-and-symbol-index/readiness-review.md)
- [Deterministic quality checks discovery notes](code-quality-and-intelligence/deterministic-quality-checks/discovery-notes.md)
- [Deterministic quality checks requirements gap analysis](code-quality-and-intelligence/deterministic-quality-checks/requirements-gap-analysis.md)
- [Deterministic quality checks PRD](code-quality-and-intelligence/deterministic-quality-checks/prd.md)
- [Deterministic quality checks domain model](code-quality-and-intelligence/deterministic-quality-checks/canonical-domain-model.md)
- [Deterministic quality checks use cases](code-quality-and-intelligence/deterministic-quality-checks/canonical-use-cases.md)
- [Deterministic quality checks contract](code-quality-and-intelligence/deterministic-quality-checks/canonical-api-cli-contract.md)
- [Deterministic quality checks acceptance scenarios](code-quality-and-intelligence/deterministic-quality-checks/acceptance-scenarios.md)
- [Deterministic quality checks readiness review](code-quality-and-intelligence/deterministic-quality-checks/readiness-review.md)
- [Live analysis and MCP discovery notes](code-quality-and-intelligence/live-analysis-and-mcp/discovery-notes.md)
- [Live analysis and MCP requirements gap analysis](code-quality-and-intelligence/live-analysis-and-mcp/requirements-gap-analysis.md)
- [Live analysis and MCP PRD](code-quality-and-intelligence/live-analysis-and-mcp/prd.md)
- [Live analysis and MCP domain model](code-quality-and-intelligence/live-analysis-and-mcp/canonical-domain-model.md)
- [Live analysis and MCP use cases](code-quality-and-intelligence/live-analysis-and-mcp/canonical-use-cases.md)
- [Live analysis and MCP contract](code-quality-and-intelligence/live-analysis-and-mcp/canonical-api-cli-contract.md)
- [Live analysis and MCP acceptance scenarios](code-quality-and-intelligence/live-analysis-and-mcp/acceptance-scenarios.md)
- [Live analysis and MCP readiness review](code-quality-and-intelligence/live-analysis-and-mcp/readiness-review.md)
- [Application PRD](../prd.md)
