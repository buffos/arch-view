# Arch View

Arch View analyzes supported source repositories and presents a navigable,
language-neutral architecture view. The first implementation targets Go and
keeps the analyzer boundary open for Python, TypeScript, Rust, and Clojure.

The original reference implementation is available in the upstream
[unclebob/arch-view repository](https://github.com/unclebob/arch-view). It is
outside this repository and is not modified as part of Arch View work.

## Viewer guide

This is a temporary contributor/user guide. It records the decisions behind
the first viewer so that common questions are answered in the repository as
well as in the UI.

### What the graph means

- A relationship is directed: `from -> to` means the source module depends on
  the target module.
- A `Layer N` label is dependency depth derived from the local relationship
  graph. It is not a language, package, or architectural classification.
- A group can contain modules from several layers. For example, `internal`
  may contain modules in Layers 0, 1, and 2.
- A real cycle is derived from canonical relationships and keeps its cycle
  indicator. A group-level self-loop created only by collapsing several
  different modules is not itself a cycle.

### Why the overview hides some edges

The top-level view is a projection of the canonical model. It is intentionally
local-first: standard-library, external, unresolved, and dynamic imports stay
in the model but are hidden or summarized until the user expands the view.

When a group contains relationships between its own child modules, the
overview does not draw a misleading self-loop. It shows a summary such as
`4 internal relationships`; the individual relationships remain available in
the group detail/list view.

### Diagnostics

A diagnostic is a structured fact emitted by an analyzer or model normalizer
when analysis encounters an error, uncertainty, or inconsistency. It is not a
user note and it is not a runtime log.

Diagnostics have a code, severity (`info`, `warning`, or `error`), message, and
optional subject, repository-relative path/location, evidence IDs, and
recoverability. Examples include an unreadable file, a parse error, an
unresolved import, or conflicting observations.

The current viewer is read-only and does not let a user manually add a
diagnostic. A language plugin adds one to its analysis result and attaches it
to a module ID or source evidence. The viewer then maps it to the affected
module/group. Future user annotations should be modeled separately from
diagnostics.

### Tags

Tags are analyzer-provided module metadata. They describe source facts rather
than user labels. The current Go analyzer emits `test` for packages containing
test files and `generated` for detected generated source. Other language
plugins may emit language-appropriate tags such as `polymorphic` or
`entrypoint` when they have static evidence.

The `include-tests`, `include-generated`, and build-tag options change the
analysis scope; they are not an arbitrary tag editor. A group displays the
union of the tags of its child modules. Future user-defined labels should use
separate annotations/user-tags so rerunning analysis does not overwrite them.

### Confidence and identity

Relationship/reference confidence comes from analyzer evidence and is
aggregated conservatively across contributors. The lowest contributing score
determines the visible relationship state (`high`, `medium`, `low`, or
`unknown`). A local module/group has a stable identity, but that fact is not a
relationship-confidence score; the UI keeps identity stability separate from
relationship confidence.

### Layout decisions

The canonical model owns semantics; the viewer owns geometry. The active
layout adapter is ELK Layered/elkjs (vendored at version 0.12.0) because it
calculates node positions and edge sections/bend points while leaving
rendering to our SVG layer. The viewer serves both the ELK bundle and its
worker locally, so the layout does not depend on a network CDN. The SVG
renderer still owns arrowhead styling and accessibility and consumes the ELK
routes rather than redrawing them with unrelated curves.

ELK can improve node placement, edge crossings, attachment points, and route
geometry. It cannot decide whether a collapsed group self-loop is semantically
meaningful. That decision belongs to the projection layer. If the renderer
uses only ELK node positions and keeps drawing custom edge paths, the routing
problem remains; the renderer must consume ELK's edge routes as well.

The current flow is:

```text
canonical model
  -> visible projection and aggregation
  -> ELK layered node/edge layout
  -> SVG paths and arrowheads
```

Issue 008 expands the editable root-level ELK surface with aspect ratio,
layered base spacing, layered edge-to-edge spacing, layering strategy, cycle
breaking strategy, crossing minimization strategy, node placement strategy,
and connected-component compaction. These options are enabled only when the
pinned catalog marks them as editable `PARENTS` options for the selected
algorithm. `org.eclipse.elk.alignment` remains catalog-only because it targets
nodes; node- and edge-targeted settings are reserved for the target-aware
mapping slice. The canonical layered spacing base key is
`org.eclipse.elk.layered.spacing.baseValue`.

If the ELK worker cannot be created or a layout request fails, the viewer
uses its deterministic layer-based layout as a visible fallback. This keeps
the semantic scene usable while making the layout engine replaceable.

Manual positions are viewer-session state keyed by model revision and
hierarchy path. They never modify the canonical model.

### Project layout settings

The viewer exposes the pinned ELK adapter's 11 algorithms and 235 layout
options through a searchable, grouped settings surface. Editable options are
typed and validated before they are applied. A project-backed session discovers
the nearest `.archview.json` by checking the selected target directory and
then its parents toward the filesystem root; the first file found wins as a
complete profile, with no merging. If no file exists, built-in defaults apply.

The file will contain presentation layout settings only. Analyzer options,
viewport state, manual node positions, and canonical model facts remain
separate. If discovery loads the file from folder X, ordinary `Save` will
atomically overwrite that exact active file and will not create a project-root
copy. If no file was discovered, `Save` will require `Save As`. `Save As` will
be the only operation that accepts a custom destination folder, writes the
fixed `.archview.json` filename there, and makes it active for the current
session. Model-only sessions can apply settings temporarily but cannot persist
a project configuration. Automatic discovery in a later session still follows
the selected target's ancestor chain, so a custom file outside that chain will
need an explicit configuration-selection feature in a later issue. The
settings dialog also reports whether an entry is editable, unsupported, or not
applicable to the selected algorithm. `Apply` and `Reset defaults` are
session actions; applying a profile recalculates ELK node positions and edge
routes and clears manual positions for the current hierarchy path. If ELK
fails, the deterministic fallback remains visible.

The local viewer exposes the settings contract through `GET /v1/layout/options`,
`GET /v1/layout/config`, `POST /v1/layout/apply`, and `POST /v1/layout/reset`.
Persistence uses `PUT /v1/layout/config` for the exact active file and
`PUT /v1/layout/config/save-as` for an explicitly confirmed custom directory.
Model-only sessions can apply settings but cannot persist a project file.

### Scope boundaries

- Analyzers discover source facts and emit modules, relationships, tags,
  evidence, confidence, and diagnostics.
- `internal/model` owns canonical model types, graph derivation, stable
  model-owned identities, and hierarchy projection data. The
  `internal/model/canonical` capability owns normalization, merge/recovery
  diagnostics, and validation.
- `internal/goanalyzer` remains the public Go analyzer entrypoint while its
  implementation is split into scanner, import classification, and common
  observation assembly capabilities. Other languages can register separate
  implementations of the common analyzer contract.
- The viewer projects/aggregates the model and owns layout, interaction, and
  presentation.
- `internal/viewer/scene` owns the renderer-neutral scene contract;
  `internal/routing` owns route geometry and strategy boundaries;
  `internal/viewer/layout` owns the ELK catalog, typed handlers, profiles, and
  persistence. Browser modules and HTTP/CLI composition are kept separate from
  those capabilities.
- Exporters consume the same neutral model/view facts and must not parse source.

The upstream [reference implementation](https://github.com/unclebob/arch-view)
remains outside all of these product boundaries.

### Refactor boundary

Issues 010–015 implement the architecture refactor before general spline
rendering. They preserve the model, scene, configuration, CLI, HTTP, and
export schemas while making the replaceable boundaries explicit. Issue 016
now activates general ELK spline routes for the layered viewer and
self-contained HTML export: valid ELK control-point streams become cubic route
segments, while malformed data and manual movement retain deterministic
orthogonal fallback. The browser Download SVG action serializes the current
canvas. Go's static SVG export remains a separate deterministic orthogonal
artifact.

### Export artifacts

The CLI can write the validated model as canonical JSON, a self-contained
interactive HTML report, or a static accessible SVG:

```text
arch-view analyze --project <path> --format html --output architecture.html
arch-view export --input model.json --format json --output architecture.json
arch-view export --input model.json --format svg --output architecture.svg
```

Visual exports are local-first by default. Use `--reference-visibility
aggregated` or `--reference-visibility expanded` to show non-local boundaries
or individual imports; the canonical JSON always retains every reference and
source location. Project-backed `analyze --format html` embeds the effective
nearest-ancestor `.archview.json` layout profile; model-only `export --input`
embeds built-in defaults. The HTML file recalculates its layout with the pinned
browser ELK runtime when opened and needs no server. Source contents are not
embedded in v1, and an existing output requires `--overwrite`. Use the live
viewer's Download SVG button for a standalone SVG of the current canvas; it is
distinct from the deterministic Go CLI SVG.

### TypeScript/JavaScript project prerequisites

Arch View analyzes TypeScript and JavaScript projects statically. It reads
configuration, package metadata, and source files as data; it does not run
`tsc`, Node.js, package scripts, bundlers, or the target application.

The minimum project prerequisite is a readable `tsconfig.json` in the selected
project root. Pass `--config <path>` when the project contains more than one
TypeScript configuration, or when the intended configuration is in a
subdirectory. The selected configuration must remain inside the project root.
`tsconfig.json` may contain JSONC comments/trailing commas and a safe local
`extends` chain.

`package.json` is optional for local static analysis. When present, Arch View
reads package context and local `exports`, `imports`, `type`, `types`,
`typings`, `module`, and `main` metadata when it can use them to resolve a
module. A `package.json` by itself can be recognized as a TypeScript project
marker, but it is not sufficient to run analysis without a selected readable
`tsconfig.json`. Dependencies do not need to be installed: unavailable or
external packages remain external/unresolved references with diagnostics as
appropriate.

By default, the analyzer includes `.ts` and `.tsx` files. JavaScript and JSX
files require either `compilerOptions.allowJs` in `tsconfig.json` or the
explicit `--include-js` option. Test/spec files and directories are excluded
unless `--include-tests` is supplied. Generated/output, cache, vendor,
`node_modules`, `.git`, and `external` directories are excluded by default;
repeatable `--exclude <glob>` options add project-relative exclusions.

The analyzer reads the relevant static settings from `tsconfig.json`,
including `files`, `include`, `exclude`, `references`, `baseUrl`, `paths`,
`rootDir`, `rootDirs`, `outDir`, `allowJs`, module settings, JSX settings, and
`resolveJsonModule`. It does not perform type checking or compiler-semantic
analysis, and it does not use the host machine's Node/npm environment to make
resolution decisions.

Example:

```text
arch-view analyze --project ./my-app --language typescript --config ./tsconfig.json --include-js --format analysis-json --output analysis.json
```

## Final documentation backlog (working notes)

> This is a capture list for the future final documentation. It is not the
> final user-facing documentation and should be rewritten, shortened, and
> validated against the shipped behavior before publication.

### Python project recognition

The final documentation should answer the question "does Python require
`pyproject.toml`?" directly:

- Python does not specifically require `pyproject.toml`, but the current
  analyzer requires a project marker at the selected root: `pyproject.toml`,
  `setup.cfg`, or `setup.py`.
- Marker precedence is `pyproject.toml` → `setup.cfg` → `setup.py`. If more
  than one exists, only the preferred boundary is read.
- Auto-detection is conservative. A directory containing only `.py` files,
  `requirements.txt`, `Pipfile`, or similar files is not currently inferred to
  be a Python project. Explicit Python selection still requires a supported
  marker today.
- If Python and another analyzer have competing project markers, automatic
  selection can be ambiguous. `--language python` or
  `--analyzer org.archview.python` makes the selection explicit.
- The marker requirement is a detection/boundary policy, not a technical
  requirement for static parsing. A future explicit markerless-Python mode is
  a possible product decision; it must be kept distinct from riskier automatic
  language inference.

### What `pyproject.toml` contributes

The final documentation should explain both the general meaning of the file
and the narrower set of fields consumed by the current analyzer:

- `[project].requires-python` supplies a Python major/minor version for
  version-sensitive static rules and conditional imports. The CLI
  `--python-version` overrides it.
- `[tool.setuptools.packages.find].where`, setuptools package-dir settings,
  Poetry package `from` settings, and supported `source_roots` values can
  provide source roots. The CLI `--source-root` overrides configured roots.
- If no source root is configured, the analyzer falls back to `src/` when it
  exists and then to the project root.
- `[project].name` is useful packaging metadata but currently does not define
  the architecture graph label; the analyzer derives that label from the
  selected directory.
- General build metadata and dependency declarations are not executed or used
  as an environment resolver. The analyzer reads Python syntax statically and
  does not install packages, import the target project, or run `setup.py`.

The minimal fixture used during issue 019 intentionally contains
`where = ['configured_src']` while the CLI supplies `--source-root src` and
`--source-root extensions`. Therefore, in that run, the file primarily proves
the Python project boundary; `name` is unused, `requires-python` is overridden,
and `where` is overridden. This is useful test rationale, not an example to
copy into final user documentation.

### Include in the final documentation

- Supported Python project markers and their precedence.
- The difference between automatic detection and explicit analyzer selection.
- Source-root precedence and the Python CLI options for tests, stubs, excludes,
  and Python version.
- The fields of `pyproject.toml` that currently affect Arch View, alongside a
  clear statement about fields that are ignored by the analyzer.
- Static-analysis safety: no target-code execution, imports, installation, or
  environment-assisted dependency resolution.
- How unresolved, external, standard-library, conditional, and dynamic imports
  appear as references, confidence, diagnostics, or partial results.
- Small valid examples for a root-layout project, a `src/` project, and an
  explicit `--source-root` invocation.

### Keep out of the final documentation

- Test-only tricks such as the deliberately wrong `configured_src` fixture,
  temporary paths, internal test names, and implementation-specific regex or
  parser details.
- Claims that `project.name` or `[project].dependencies` currently drive graph
  discovery or import resolution.
- A claim that a markerless Python directory is supported before that behavior
  is deliberately implemented and tested.
- Internal state-transition mechanics, issue IDs, and planning-map details
  unless a separate contributor/development guide needs them.

### Agent automation and future skills backlog

The existing planning and delivery skills already cover issue processing,
artifact synchronization, closeout, and OKF validation. The remaining
documentation work would benefit from explicit automation skills:

- `final-documentation-curator`: collect user questions and confirmed answers
  into a working backlog, classify each item as user documentation,
  contributor documentation, test rationale, or internal planning, and only
  generate final prose when requested.
- `documentation-gap-auditor`: compare README backlog notes with the PRD,
  canonical contracts, issue evidence, and shipped behavior; report stale,
  unsupported, or missing claims before publication.
- `architecture-example-author`: generate small deterministic CLI/configuration
  examples and safe fixtures for the final documentation without executing
  target-project code.
- `agent-skill-gap-analyzer`: inspect repeated manual steps in a delivery run,
  map them to existing skills, and propose a new skill only when the workflow
  is reusable and independently bounded. The existing `skill-creator` and
  `write-a-skill` skills can then scaffold and refine it.

Automation must preserve the evidence hierarchy: code and tests establish
current behavior, canonical artifacts establish intended product behavior, and
explicit user decisions establish target or approval. An agent may automate
capture, drafting, consistency checks, and fixture generation; it must not
invent visual approval or silently convert a working note into final product
documentation.
