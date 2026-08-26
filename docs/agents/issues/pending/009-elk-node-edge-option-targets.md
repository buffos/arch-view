# 009 — Route node- and edge-targeted ELK options through the scene adapter

Execution type: AFK
Review gate: visual-review
Status: deferred

## Parent PRD

docs/architecture/explore-architecture/prd.md

## What to build

Make the layout adapter honor ELK option target metadata instead of sending
every profile option to the root graph. Keep the project configuration schema
flat and deterministic: one configured value applies uniformly to every
eligible visible node or edge in the current scene.

- Partition validated profile options by catalog targets: `PARENTS` go to the
  root graph, `NODES` go to each eligible child, and `EDGES` go to each
  eligible ELK edge.
- Start with options that have meaningful behavior for the current simple
  node/edge scene, including `org.eclipse.elk.priority` and the layered edge
  priority options `org.eclipse.elk.layered.priority.direction`,
  `org.eclipse.elk.layered.priority.shortness`, and
  `org.eclipse.elk.layered.priority.straightness`, subject to the pinned
  catalog's exact target and type metadata.
- Keep node/edge options that require ports, labels, per-element custom
  values, junction symbols, or renderer-owned styling catalog-only. Do not
  claim support merely because ELK accepts the key.
- Preserve the existing `arch-view.config/v1` `options` map. Per-node custom
  overrides, ports, labels, and compound-graph geometry are separate future
  design work.

## Acceptance criteria

- [ ] The adapter partitions each validated option using its catalog target
  metadata and emits it at the correct ELK graph element level.
- [ ] The initial node/edge tranche is editable only when its value type,
  algorithm applicability, target, and renderer-visible effect are covered by
  tests; all other target-specific entries remain catalog-only.
- [ ] A configured value is applied uniformly and deterministically to the
  eligible visible nodes or edges, with no hidden per-element state or change
  to the v1 configuration schema.
- [ ] Options are never silently misplaced on the root graph, and output-only
  ELK metadata such as junction points is never exposed as an editable input.
- [ ] Apply, Reset, Save, Save As, nearest-ancestor discovery, fallback
  behavior, and manual-position clearing retain their existing semantics.
- [ ] Focused tests assert root/node/edge request placement, typed validation,
  algorithm compatibility, unsupported target rejection, deterministic output,
  and canonical model immutability.
- [ ] A visual review confirms that supported node/edge-targeted settings do
  not introduce invalid attachments, unreadable routes, or renderer-only
  styling surprises in windowed and full-canvas views.

## Artifact sync required

- Application PRD: no impact — the user-facing workflow remains the existing
  layout-settings journey.
- Application architecture summary: required if the renderer-neutral layout
  input contract changes; otherwise record no impact in the implementation
  result. The adapter must remain the owner of ELK target mapping and the
  canonical model must remain unaware of layout options.
- Owning capability artifacts: required only if the external layout contract,
  option scope semantics, or supported renderer features change; otherwise
  record no impact with the verification evidence.
- Delivery truth: required — keep this issue, the registry, the owning
  capability `issues:` list, and `.okf/log.md` synchronized.
- Reference boundary: `external/` remains read-only and untouched.

## Blocked by

—

## Scheduling note

Deferred by explicit user direction on 2026-08-26. The current implementation
continues to support the approved parent-level option tranche; node- and
edge-targeted ELK options remain catalog-only until this issue is deliberately
resumed.

## User stories addressed

- US-EX-003

## Contract and scenario trace

- Contract: `docs/architecture/explore-architecture/canonical-api-cli-contract.md`
- Scenarios: SC-EX-012, SC-EX-013, SC-EX-016

## Scenario traceability and verification plan

| Source rule / use case | Scenario | Issue criterion | Planned verification | State |
|---|---|---|---|---|
| ELK option targets determine where presentation settings are applied | SC-EX-012 | Root/node/edge options are placed according to catalog metadata | Adapter request-shape and catalog tests | planned |
| Layout application changes geometry only | SC-EX-013 | Targeted options produce valid positions/routes without model mutation | Scene/layout integration and immutability tests | planned |
| Unsupported target-specific behavior remains safe and visible | SC-EX-016 | Options needing unsupported scene features remain catalog-only | Validation, fallback, and browser diagnostics review | planned |

## Verification surfaces

- Backend boundary: target-aware ELK graph construction, validation, and
  deterministic option application.
- Frontend integration: support/applicability presentation and resulting
  routes/positions.
- End-to-end: a saved profile reloads with the same target-aware layout
  behavior in a later project session.
- Repository/OKF integrity: `external/` is untouched, links resolve, and the
  issue registry remains synchronized.

## Review handoff

Do not close this issue until automated verification passes and the user has
visually approved target-aware layout behavior in normal and full-canvas views.
