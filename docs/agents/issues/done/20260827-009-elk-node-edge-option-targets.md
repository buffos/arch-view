# 009 — Route node- and edge-targeted ELK options through the scene adapter

Execution type: AFK
Review gate: visual-review
Status: done

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

- [x] The adapter partitions each validated option using its catalog target
  metadata and emits it at the correct ELK graph element level.
- [x] The initial node/edge tranche is editable only when its value type,
  algorithm applicability, target, and renderer-visible effect are covered by
  tests; all other target-specific entries remain catalog-only.
- [x] A configured value is applied uniformly and deterministically to the
  eligible visible nodes or edges, with no hidden per-element state or change
  to the v1 configuration schema.
- [x] Options are never silently misplaced on the root graph, and output-only
  ELK metadata such as junction points is never exposed as an editable input.
- [x] Apply, Reset, Save, Save As, nearest-ancestor discovery, fallback
  behavior, and manual-position clearing retain their existing semantics.
- [x] Focused tests assert root/node/edge request placement, typed validation,
  algorithm compatibility, unsupported target rejection, deterministic output,
  and canonical model immutability.
- [x] A visual review confirms that supported node/edge-targeted settings do
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
- Reference boundary: the upstream [reference repository](https://github.com/unclebob/arch-view) remains read-only and untouched.

## Blocked by

—

## Scheduling note

Resumed by explicit user direction on 2026-08-27. The current implementation
continues to support the approved parent-level option tranche; this issue now
implements the bounded node- and edge-targeted ELK option tranche.

## User stories addressed

- US-EX-003

## Contract and scenario trace

- Contract: `docs/architecture/explore-architecture/canonical-api-cli-contract.md`
- Scenarios: SC-EX-012, SC-EX-013, SC-EX-016

## Scenario traceability and verification plan

| Source rule / use case | Scenario | Issue criterion | Verification evidence | State |
|---|---|---|---|---|
| ELK option targets determine where presentation settings are applied | SC-EX-012 | Root/node/edge options are placed according to catalog metadata | `go test ./internal/viewer/layout -count=1`; `node internal/viewer/web/layout_request_test.js` | complete |
| Layout application changes geometry only | SC-EX-013 | Targeted options produce valid positions/routes without model mutation | `go test ./... -count=1`, race suite, profile/request immutability assertions, and user visual review | complete |
| Unsupported target-specific behavior remains safe and visible | SC-EX-016 | Options needing unsupported scene features remain catalog-only | Targeted catalog/validation tests and user visual support/applicability review | complete |

## Verification surfaces

- Backend boundary: target-aware ELK graph construction, validation, and
  deterministic option application.
- Frontend integration: support/applicability presentation and resulting
  routes/positions.
- End-to-end: a saved profile reloads with the same target-aware layout
  behavior in a later project session.
- Repository/OKF integrity: the upstream [reference repository](https://github.com/unclebob/arch-view) is untouched, links resolve, and the
  issue registry remains synchronized.

## Review handoff

Automated verification passed. The user confirmed on 2026-08-27 that the
target-aware settings review passes in normal and full-canvas views, including
the resulting routes and supported/unsupported option treatment.

## Implementation result

The layout registry now supports the bounded simple-scene target tranche:
`org.eclipse.elk.priority` is applied to every eligible visible node and edge,
and the layered direction, shortness, and straightness priority options are
applied to every eligible visible edge. Parent options remain on the root
graph. Unsupported node/edge/port/label/junction options remain catalog-only;
the v1 flat configuration schema and canonical model are unchanged.

Verification passed on 2026-08-27:

- `go test ./... -count=1`
- `go test -race ./...`
- `go vet ./...`
- `go build ./...`
- `staticcheck ./...`
- `golangci-lint run`
- `node --check internal/viewer/web/layout_request.js`
- `node internal/viewer/web/layout_request_test.js`
- `git diff --check`

The user-approved visual review covered the editable targeted priority options
in the settings surface, application in windowed and full-canvas views,
readable routes, and correct target placement.

## Artifact sync result

- Application PRD: no impact; the existing layout-settings journey and product
  scope are unchanged.
- Application architecture summary: no external contract impact; target
  mapping remains inside the layout adapter and the renderer-neutral scene,
  configuration schema, and canonical model remain unchanged.
- Owning capability artifacts: synchronized with the target-level application
  rule, supported tranche, and current verification status in the Explore exact
  specification set.
- Delivery truth: the issue is complete and archived; the owning capability
  and implementation-slice references are synchronized.
- Reference boundary: the upstream [reference repository](https://github.com/unclebob/arch-view) remains read-only and untouched.
