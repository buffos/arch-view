# Arch View

Arch View analyzes supported source repositories and presents a navigable,
language-neutral architecture view. The first implementation targets Go and
keeps the analyzer boundary open for Python, TypeScript, Rust, and Clojure.

`external/` is read-only reference material. It is intentionally ignored by
Git and must not be modified as part of Arch View work.

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
- The model normalizes identities, hierarchy, cycles, and dependency layers.
- The viewer projects/aggregates the model and owns layout, interaction, and
  presentation.
- Exporters consume the same neutral model/view facts and must not parse source.

The `external/` reference implementation remains outside all of these product
boundaries.

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
source location. Source contents are not embedded in v1, and an existing
output requires `--overwrite`.
