# Explore and inspect architecture canonical API/CLI contract

## Local HTTP contract

- `GET /` → local viewer application.
- `GET /v1/models/{model_id}` → canonical model metadata/content.
- `GET /v1/models/{model_id}/projection?path=<segment>&mode=overview|detail|list&reference_visibility=hidden|aggregated|expanded&reference_scope=<scope>` → renderer-neutral scene snapshot. `path` and `reference_scope` may be repeated; valid scopes are `standard_library`, `external`, `unresolved`, and `dynamic`.
- `GET /v1/source?model_id=<id>&path=<relative>&start_line=<n>&end_line=<n>` → read-only source excerpt when path is inside the model project root.
- `POST /v1/reanalysis` with `{ "project_root":"string", "language":"string|null", "options":{} }` → new complete/partial model revision; failed reanalysis leaves the prior revision active.
- `GET /v1/layout/options` → grouped, typed catalog of the algorithms and options exposed by the pinned ELK adapter, including defaults, allowed values, descriptions, applicability, and renderer support.
- `GET /v1/layout/config` → effective layout profile, configuration origin (`default`, `project`, `ancestor`, `custom`, or `session`), active file location when applicable, save-action availability, and validation diagnostics. Project-backed sessions resolve `.archview.json` from the selected target directory upward; model-only sessions report session-only/default origin.
- `POST /v1/layout/apply` with `{ "schema_version":"arch-view.config/v1", "layout":{ "algorithm":"layered", "options":{} } }` → validate and apply a session profile without writing a project file; the response reports the effective session profile and its persistence availability.
- `POST /v1/layout/reset` → restore the active session to built-in layout defaults without deleting or rewriting a project configuration file.
- `PUT /v1/layout/config` with `{ "schema_version":"arch-view.config/v1", "layout":{ "algorithm":"layered", "options":{} } }` → ordinary `Save`: validate and atomically overwrite the exact active discovered `.archview.json`. The request has no destination field and never creates a file when no active file exists; that case returns `409 save_as_required`.
- `PUT /v1/layout/config/save-as` with `{ "schema_version":"arch-view.config/v1", "layout":{ "algorithm":"layered", "options":{} }, "destination_dir":"string", "confirm":true }` → explicit `Save As`: validate and atomically write the fixed `.archview.json` filename in the user-selected custom folder, then report the new active path. This is the only layout-config operation that accepts a destination. Model-only sessions cannot persist a project file.

HTTP errors use the common `{error:{code,message,details}}` shape. `403` is used for source-root/path policy violations or unavailable model-only persistence; `409` with `save_as_required` is used when ordinary `Save` has no active file; `422` is used for invalid scene/configuration requests; `404` is used for unknown model/evidence.

## Renderer-neutral scene shape

```json
{
  "model_id":"string",
  "model_revision":"string",
  "hierarchy_path":[],
  "visible_nodes":[],
  "visible_relationships":[],
  "cycle_indicators":[],
  "diagnostic_indicators":[],
  "layer_labels":[],
  "reference_summary":{},
  "reference_details":[],
  "evidence_links":[],
  "accessibility": {"reading_order":[],"descriptions":{}}
}
```

The default overview uses `reference_visibility=hidden`: it shows project-local modules/groups and a boundary summary while retaining non-local references in the model and evidence list. `aggregated` shows one or more scope boundary nodes; `expanded` shows individual reference nodes. The `list` mode exposes `reference_details` for individual imports with target scope, confidence, contributor IDs, and evidence without requiring them to be rendered as graph nodes.

When a group contains non-cycle relationships between its child modules, the
overview may omit the resulting group self-loop and expose the canonical
contributors through `internal_relationship_ids[]` and
`counts.internal_relationship_count`. This is a presentation projection only;
canonical relationships and evidence remain unchanged. Real cycles are not
eligible for this collapse.

The first browser renderer is SVG/HTML. Canvas/WebGL adapters may consume the same shape; no renderer may invent semantic relationships or decide reference scope from labels.

## Layout configuration contract

The v1 project file is named `.archview.json` and contains presentation layout preferences only:

```json
{
  "schema_version": "arch-view.config/v1",
  "layout": {
    "algorithm": "layered",
    "options": {
      "elk.direction": "RIGHT",
      "elk.edgeRouting": "ORTHOGONAL"
    }
  }
}
```

Option keys are the pinned adapter's catalog IDs. The local host also accepts
the conventional `elk.*` shorthand on input and normalizes it to the catalog
ID when saving.

When a validated profile is handed to the ELK adapter, each option is emitted
at the graph-element level declared by its catalog targets: `PARENTS` options
are placed on the root graph, `NODES` options are copied uniformly to each
eligible visible node, and `EDGES` options are copied uniformly to each
eligible visible edge. The current editable target-aware tranche is
`org.eclipse.elk.priority` for nodes and edges plus the layered direction,
shortness, and straightness edge-priority options. Options that require ports,
labels, junctions, per-element values, or renderer-owned styling remain
catalog-only and are rejected if supplied as editable profile values. This
mapping does not change the flat `arch-view.config/v1` schema or canonical
model facts.

### Spline routing extension

Issue 016 activates `org.eclipse.elk.edgeRouting=SPLINES` for the pinned
layered adapter. Valid ELK spline sections and control points are normalized
into the existing renderer-neutral route representation as finite cubic
segments, then consumed by the live browser and browser current-canvas SVG
serializers. The
self-contained HTML export embeds the effective profile/catalog and pinned ELK
runtime, so it recalculates the scene in the browser without network access.
The browser Download SVG action serializes the current canvas, including valid
ELK spline routes. Missing or malformed control data uses the deterministic
orthogonal fallback; it never changes canonical relationships or emits an
invalid path. Go's static `--format svg` export remains explicitly
deterministic orthogonal. The configuration schema remains flat and versioned.
Spline-specific tuning, ports, labels, junctions, compound geometry, and
manual spline routing remain outside this bounded extension unless separately
covered by implementation and tests.

Discovery starts at the selected target directory and walks parent directories toward the filesystem root. The nearest file wins as a complete profile; v1 does not merge files. If none exists, built-in defaults apply. An invalid nearest file produces a visible configuration diagnostic and does not silently select a farther file.

The settings surface may maintain an unsaved session draft. Applying a valid profile recalculates the current scene with the selected layout adapter, consumes its node positions and edge routes, and clears manual positions for that hierarchy path. Reset restores built-in defaults for the session without deleting a project file. Ordinary `Save` is explicit and overwrites the exact active discovered file; it never creates a file when none is active. `Save As` is the only operation that accepts a custom destination folder and writes the fixed `.archview.json` filename there after explicit confirmation; the resulting file becomes active for the current session. Automatic discovery in a later session still follows the target-to-filesystem-root ancestor chain, so a custom file outside that chain requires a later explicit configuration-selection capability, which is outside issue 007. Analyzer options, canonical model facts, viewport state, and manual node positions are not persisted in `.archview.json`.

## CLI mapping

- `arch-view open --model <model.json> [--port <n>]` → start local web viewer.
- `arch-view open --project <path> [--language <id>] [--port <n>]` → analyze then open.
- Viewer options may select reference visibility, but the default remains local-first overview behavior. A project-backed `open` session also loads the nearest `.archview.json`; a model-backed session uses defaults and session-only settings.

The CLI never enables source editing or target-code execution.

## Interaction parity

Browser interactions must preserve hierarchy paths, selection IDs, evidence IDs, cycle state, diagnostics, reference scope, confidence, and source locations across renderers. Keyboard/list mode must expose the same facts as the graphic view, including individual imports when a module or group is selected. Viewport and optional manual positions are session-scoped and keyed by model revision and hierarchy path. Layout profiles are presentation settings; applying one may recalculate geometry but cannot rewrite canonical scene facts.
