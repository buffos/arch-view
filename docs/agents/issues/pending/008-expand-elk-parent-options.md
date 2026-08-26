# 008 — Expand ELK parent-level layout option support

Execution type: AFK
Review gate: visual-review
Status: ready-for-agent

## Parent PRD

docs/architecture/explore-architecture/prd.md

## What to build

Expand the Arch View layout adapter's editable surface beyond the first safe
allowlist while keeping the current flat/aggregated scene and
`arch-view.config/v1` profile shape intact.

- Enable a reviewed tranche of ELK options whose catalog target includes
  `PARENTS` and whose effects are represented by the current root graph and
  SVG renderer.
- The initial tranche is `org.eclipse.elk.aspectRatio`,
  `org.eclipse.elk.alignment`, `org.eclipse.elk.spacing.baseValue`,
  `org.eclipse.elk.layered.spacing.edgeEdgeBetweenLayers`,
  `org.eclipse.elk.layered.layering.strategy`,
  `org.eclipse.elk.layered.cycleBreaking.strategy`,
  `org.eclipse.elk.layered.crossingMinimization.strategy`,
  `org.eclipse.elk.layered.nodePlacement.strategy`, and
  `org.eclipse.elk.layered.compaction.connectedComponents`, subject to the
  pinned ELK catalog confirming their current types and applicability.
- Preserve the full catalog. Options outside this tranche remain visible and
  catalog-only until a later issue proves their target, metadata, output, and
  renderer behavior.
- Derive complete typed metadata for each enabled option: enum values,
  numeric bounds, defaults, algorithm applicability, and safe conversion into
  the ELK request.
- Apply the enabled values to the root graph only. Do not silently attach
  node- or edge-targeted options to the root as a shortcut.

## Acceptance criteria

- [ ] The initial parent-level tranche is editable only for algorithms and
  option values supported by the pinned ELK bundle; every other catalog entry
  remains visibly non-editable.
- [ ] The settings catalog exposes the enabled options' exact type, default,
  current value, allowed values or bounds, description, applicability, and
  support state.
- [ ] Invalid enum, numeric, algorithm-incompatible, and unsupported values
  are rejected before Apply or Save and do not partially update the active
  profile.
- [ ] Apply passes canonical option IDs and typed values to the root ELK graph,
  recalculates positions and routes, clears manual positions for the active
  hierarchy path, and leaves canonical model facts unchanged.
- [ ] Reset, active-file Save, Save As, nearest-ancestor discovery, and the
  `arch-view.config/v1` file shape continue to work without storing viewport or
  manual-position state.
- [ ] Worker failure and unsupported layout requests still use the visible
  deterministic fallback and preserve an actionable diagnostic.
- [ ] Focused tests cover each enabled option's metadata, validation,
  applicability, root-level request mapping, invalid input, and model
  immutability; the existing repository gates remain green.
- [ ] A visual review confirms that at least one representative setting from
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
| Typed, applicable ELK presentation settings are editable only when the adapter can honor them | SC-EX-012 | Initial parent-level tranche has complete metadata and support classification | Catalog/validation tests plus browser settings review | planned |
| Applying presentation settings recalculates geometry without semantic mutation | SC-EX-013 | Root options produce positions/routes and preserve the model | Layout request, scene, and immutability tests | planned |
| Invalid or unavailable layout behavior remains visible and safe | SC-EX-016 | Invalid values and worker failures use diagnostics/fallback | Validation and worker-failure tests plus browser review | planned |

## Verification surfaces

- Backend boundary: catalog metadata, typed validation, algorithm applicability,
  root request mapping, and model immutability.
- Frontend integration: editable controls, Apply/Reset, diagnostics, and
  windowed/full-canvas layout changes.
- End-to-end: a project session applies and persists an enabled parent-level
  option while preserving the existing configuration discovery behavior.
- Repository/OKF integrity: `external/` is untouched, links resolve, and the
  issue registry remains synchronized.

## Review handoff

Do not close this issue until automated verification passes and the user has
visually approved the enabled parent-level options, their diagnostics, and
their resulting layouts in normal and full-canvas views.
