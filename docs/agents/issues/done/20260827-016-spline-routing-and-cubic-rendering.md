# 016 — Render ELK spline routes in the viewer and exports

Execution type: AFK
Review gate: visual-review
Status: done

## Parent PRD

docs/architecture/explore-architecture/prd.md

## What to build

Activate the existing reserved cubic route representation for general ELK
spline output. When the layered layout profile selects
`org.eclipse.elk.edgeRouting=SPLINES`, normalize valid ELK spline sections and
control points into the renderer-neutral `internal/routing` representation and
render the result consistently in the live browser viewer and self-contained
HTML export. Add a browser Download SVG action that serializes the current
canvas, including its active ELK or manual route geometry.

Keep the extension bounded to the current simple node/edge scene. Preserve the
canonical model, the flat `arch-view.config/v1` options map, the existing
`embeddedExport.layouts` shape, hierarchy/reference policies, and the
deterministic orthogonal/manual route behavior. The established self-loop cubic
route is not reclassified as a general spline route.

The settings catalog must become honest end to end: base `SPLINES` routing is
editable only for the pinned layered algorithm after the browser and export
renderers can consume it. Spline-specific tuning options such as routing mode
or sloppy spacing remain catalog-only unless this issue implements and tests
their complete effect. Do not use Libavoid.

## Acceptance criteria

- [x] The layout catalog and validation allow `org.eclipse.elk.edgeRouting` to
  select `SPLINES` for the pinned layered algorithm only after its renderer
  support is complete; incompatible algorithms and unsupported spline tuning
  options remain visibly unavailable.
- [x] Valid ELK spline sections and control-point data are normalized into the
  existing renderer-neutral route representation as finite cubic segments,
  preserving section order, endpoints, labels, and arrow direction.
- [x] The browser route serializer renders general cubic spline segments while
  retaining the current arrowheads, edge labels, self-loop appearance, and
  deterministic fallback behavior.
- [x] The self-contained HTML export embeds the effective layout profile,
  catalog, and pinned ELK runtime so it recalculates its scene in the browser
  without external assets or module imports. The browser Download SVG action
  serializes the current canvas through the route-capable SVG path serializer.
  Go's static `--format svg` export remains explicitly deterministic
  orthogonal and is intentionally a separate artifact path.
- [x] Missing, malformed, incomplete, or non-finite spline control data falls
  back deterministically to the existing orthogonal route; it never emits an
  invalid path or detached arrowhead.
- [x] Manual node movement and drop continue to use the existing deterministic
  orthogonal routing calculation and do not invoke ELK or Libavoid for every
  pointer update.
- [x] Applying, resetting, saving, discovering, and reloading a profile retain
  their existing semantics; no canonical model fact, hierarchy identity, or
  reference evidence is changed by spline presentation.
- [x] Deterministic backend, browser-module, export-parity, and malformed-route
  tests cover the new mapping and fallback behavior.
- [x] User visual review approves spline routes in windowed, full-canvas, and
  exported views, including labels, arrowheads, self-loops, navigation, and
  fallback cases.

## Implementation result

The layered layout profile now accepts `SPLINES` and the browser ELK adapter
maps ELK's `3n−1` bend/control-point streams into finite piecewise cubic route
segments. Cubic paths use the existing arrow marker and edge-label layer; the
established self-loop remains a self-loop route, and manual node movement/drop
continues to use deterministic orthogonal geometry without re-running ELK.

The browser and SVG serializers consume the same route representation. The
self-contained HTML bundle embeds the effective profile and catalog alongside
the pinned ELK runtime, preserves the existing `embeddedExport.layouts` shape
as a deterministic fallback, and recalculates valid scenes when the file is
opened. The browser Download SVG action captures that current scene, including
active spline routes or deterministic manual orthogonal routes. Go's static
`--format svg` exporter is unchanged in behavior and continues to use its
deterministic orthogonal layout. Malformed spline sections are omitted from the
ELK route map and therefore use the existing orthogonal fallback in the graph
renderer.

## Non-goals

- Ports, port labels, junction symbols, compound/cross-hierarchy spline
  geometry, or per-edge custom spline controls.
- Making every ELK spline-related catalog option editable.
- Spline routing for manual drag/drop or a new renderer such as Canvas/WebGL.
- Making the raw `arch-view export --input` command discover a project
  configuration when it receives only a model file; it embeds built-in
  defaults unless the project-backed `analyze --format html` path supplies a
  discovered profile.
- Changes to canonical model JSON, scene semantics, analyzer contracts, or the
  upstream [reference repository](https://github.com/unclebob/arch-view).

## Scheduling note

Created after Issue 009 closeout on 2026-08-27. Issues 010–015 established and
visually reviewed the renderer-neutral routing, browser composition, export,
scene, and layout boundaries required for this work. This issue is the next
bounded presentation extension; it does not require a new OKF capability node.

## User stories addressed

- US-EX-003

## Contract and scenario trace

- Contract: `docs/architecture/explore-architecture/canonical-api-cli-contract.md`
- Scenarios: SC-EX-013, SC-EX-017

## Scenario traceability and verification plan

| Source rule / use case | Scenario | Issue criterion | Verification evidence | State |
|---|---|---|---|---|
| A selected layered profile may request spline routing when the renderer supports it | SC-EX-013 | Catalog/validation and layout application expose only the supported spline combination | Layout catalog/validation tests, profile request-shape tests, and pinned ELK integration test | implemented |
| ELK spline control data is a presentation route, not a model relationship | SC-EX-017 | ELK sections become cubic route segments with deterministic fallback and unchanged canonical facts | Route normalization tests, model immutability-preserving request construction, and malformed-route tests | implemented |
| Browser and exports share route geometry | SC-EX-017 | Windowed, full-canvas, self-contained HTML, and browser Download SVG retain valid curves, labels, and arrows; Go static SVG remains deterministic orthogonal | Pure JS route tests, embedded-runtime/export tests, SVG serializer tests, deterministic-output tests, and user visual approval | complete |

## Verification surfaces

- Backend boundary: ELK spline-section normalization, catalog/validation, and
  deterministic fallback.
- Frontend integration: cubic path serialization, labels, arrowheads, pan,
  zoom, fit, navigation, and manual-route preservation.
- Export: self-contained HTML opens with embedded ELK/profile data; browser
  Download SVG captures the current route geometry; Go static SVG retains its
  deterministic orthogonal contract.
- Repository/OKF integrity: strict OKF validation, synchronized issue links,
  and an untouched upstream [reference repository](https://github.com/unclebob/arch-view).

## Artifact sync required

- Application PRD: no product-scope change; spline rendering is an already
  identified presentation roadmap item. Record the new delivery issue in the
  current roadmap/status text.
- Application architecture summary: required — record that the reserved
  renderer-neutral cubic boundary is being activated for supported ELK spline
  output, while canonical model and configuration schemas remain unchanged.
- Owning capability artifacts: required — add the spline route contract,
  scenario, fallback rules, and issue reference to the Explore specification
  set.
- Delivery truth: required — synchronize the issue file, registry, first
  implementation slice, owning capability `issues:` list, and `.okf/log.md`.
- Topology: no new capability node; this is an in-node extension of Explore and
  inspect architecture.
- Reference boundary: the upstream [reference repository](https://github.com/unclebob/arch-view)
  remains read-only and untouched.

## Review handoff

Automated verification passed. The user explicitly approved the visual review
of spline routes in the windowed viewer, full-canvas viewer, self-contained
HTML export, and browser Download SVG behavior, including labels, arrowheads,
self-loops, navigation, and deterministic fallback cases.

## Implementation result

The layered layout profile accepts `SPLINES` and the browser ELK adapter maps
valid `3n−1` control-point streams into finite cubic route segments. The live
viewer, self-contained HTML export, and browser Download SVG action consume the
route-capable geometry; manual movement/drop and Go's static SVG export retain
deterministic orthogonal behavior. No Libavoid routing or canonical model
change was introduced.

Verification passed on 2026-08-27:

- `go test ./... -count=1`
- `go test -race ./... -count=1`
- `go vet ./...`
- `go build ./...`
- `staticcheck ./...`
- `golangci-lint run`
- JavaScript syntax, pure-module, and pinned-ELK integration tests
- strict OKF validation
- `git diff --check`

## Artifact sync result

- Application PRD, application architecture summary, Explore specification
  artifacts, implementation slice, registry, and `.okf/` references are
  synchronized with the completed issue.
- The issue is archived after explicit user visual approval; no new capability
  node was required, and the upstream reference repository remains untouched.
