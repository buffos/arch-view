# 008 — Expand ELK parent-level layout option support

Execution type: AFK
Review gate: visual-review
Status: done

## Parent PRD

docs/architecture/explore-architecture/prd.md

## What to build

Expand the Arch View layout adapter's editable surface beyond the first safe
allowlist while keeping the current flat/aggregated scene and
`arch-view.config/v1` profile shape intact.

- Enable a reviewed tranche of ELK options whose catalog target includes
  `PARENTS` and whose effects are represented by the current root graph and
  SVG renderer.
- The pinned catalog confirms this root-safe tranche:
  `org.eclipse.elk.aspectRatio`,
  `org.eclipse.elk.layered.spacing.baseValue`,
  `org.eclipse.elk.layered.spacing.edgeEdgeBetweenLayers`,
  `org.eclipse.elk.layered.layering.strategy`,
  `org.eclipse.elk.layered.cycleBreaking.strategy`,
  `org.eclipse.elk.layered.crossingMinimization.strategy`,
  `org.eclipse.elk.layered.nodePlacement.strategy`, and
  `org.eclipse.elk.layered.compaction.connectedComponents`, as confirmed by
  the pinned ELK catalog's current types and applicability.
- `org.eclipse.elk.alignment` is deliberately not part of this tranche: the
  pinned catalog targets it at `NODES`, so it remains catalog-only until the
  target-aware mapping in issue 009. The non-layered
  `org.eclipse.elk.spacing.baseValue` identifier is not present in the pinned
  catalog; the layered parent option above is the canonical supported key.
- Preserve the full catalog. Options outside this tranche remain visible and
  catalog-only until a later issue proves their target, metadata, output, and
  renderer behavior.
- Derive complete typed metadata for each enabled option: enum values,
  numeric bounds, defaults, algorithm applicability, and safe conversion into
  the ELK request.
- Apply the enabled values to the root graph only. Do not silently attach
  node- or edge-targeted options to the root as a shortcut.

## Acceptance criteria

- [x] The initial parent-level tranche is editable only for algorithms and
  option values supported by the pinned ELK bundle; every other catalog entry
  remains visibly non-editable.
- [x] The settings catalog exposes the enabled options' exact type, default,
  current value, allowed values or bounds, description, applicability, and
  support state.
- [x] Invalid enum, numeric, algorithm-incompatible, and unsupported values
  are rejected before Apply or Save and do not partially update the active
  profile.
- [x] Apply passes canonical option IDs and typed values to the root ELK graph,
  recalculates positions and routes, clears manual positions for the active
  hierarchy path, and leaves canonical model facts unchanged.
- [x] Reset, active-file Save, Save As, nearest-ancestor discovery, and the
  `arch-view.config/v1` file shape continue to work without storing viewport or
  manual-position state.
- [x] Worker failure and unsupported layout requests still use the visible
  deterministic fallback and preserve an actionable diagnostic.
- [x] Focused tests cover each enabled option's metadata, validation,
  applicability, root-level request mapping, invalid input, and model
  immutability; the existing repository gates remain green.
- [x] A visual review confirms that at least one representative setting from
  each enabled family changes the layout predictably in windowed and
  full-canvas views without degrading edge readability.

## Artifact sync required

- Application PRD: no impact — this is an incremental implementation of the
  already specified layout-settings journey and does not add a new user
  workflow.
- Application architecture summary: no impact — the root ELK adapter,
  renderer-neutral scene, and presentation-only configuration boundary remain
  unchanged.
- Owning capability artifacts: no impact — the existing catalog, typed profile,
  Apply/Reset, and renderer-support contracts already own this behavior; update
  them only if the implementation changes an externally visible contract.
- Delivery truth: required — keep this issue, the registry, the owning
  capability `issues:` list, and `.okf/log.md` synchronized.
- Reference boundary: `external/` remains read-only and untouched.

## Blocked by

—

## User stories addressed

- US-EX-003

## Contract and scenario trace

- Contract: `docs/architecture/explore-architecture/canonical-api-cli-contract.md`
- Scenarios: SC-EX-012, SC-EX-013, SC-EX-016

## Scenario traceability and verification plan

| Source rule / use case | Scenario | Issue criterion | Planned verification | State |
|---|---|---|---|---|
| Typed, applicable ELK presentation settings are editable only when the adapter can honor them | SC-EX-012 | Initial parent-level tranche has complete metadata and support classification | Catalog/validation tests plus browser settings review | approved |
| Applying presentation settings recalculates geometry without semantic mutation | SC-EX-013 | Root options produce positions/routes and preserve the model | Layout request, scene, and immutability tests | approved |
| Invalid or unavailable layout behavior remains visible and safe | SC-EX-016 | Invalid values and worker failures use diagnostics/fallback | Validation and worker-failure tests plus browser review | approved |

## Verification surfaces

- Backend boundary: catalog metadata, typed validation, algorithm applicability,
  root request mapping, and model immutability.
- Frontend integration: editable controls, Apply/Reset, diagnostics, and
  windowed/full-canvas layout changes.
- End-to-end: a project session applies and persists an enabled parent-level
  option while preserving the existing configuration discovery behavior.
- Repository/OKF integrity: `external/` is untouched, links resolve, and the
  issue registry remains synchronized.

## Implementation result

The viewer now enables eight pinned parent-level option families: aspect ratio,
layered base spacing, layered edge-to-edge spacing, layering strategy, cycle
breaking strategy, crossing minimization strategy, node placement strategy,
and connected-component compaction. Their catalog metadata includes exact
algorithm applicability, defaults, enum values, and numeric bounds. The Go
boundary validates finite numeric values, integer shape, exclusive bounds, and
enum membership before Apply/Save.

The browser ELK request builder is isolated and only forwards validated,
catalogued, editable `PARENTS` options to the root graph. Node- and edge-
targeted options are not attached to the root. Existing layout application,
reset, persistence, fallback, and model-immutability behavior are unchanged.

Automated verification passed, including the focused Go catalog/validation
tests, the pure JavaScript root-request test, JavaScript syntax checks, and the
repository gates listed below. The user approved the required visual review of
representative parent-level settings in windowed and full-canvas views,
including diagnostics, fallbacks, and resulting layouts.

### Automated verification

- `go test ./... -count=1`
- `go test -race ./...`
- `go vet ./...`
- `go build ./...`
- `staticcheck ./...`
- `golangci-lint run`
- `node --check internal/viewer/web/layout_request.js`
- `node --check internal/viewer/web/app.js`
- `node internal/viewer/web/layout_request_test.js`
- `git diff --check`

## Human review result

On 2026-08-26 the user explicitly approved issue 008 after reviewing the
enabled parent-level options, their diagnostics, and their resulting layouts
in normal and full-canvas views. The visual-review gate is complete.
