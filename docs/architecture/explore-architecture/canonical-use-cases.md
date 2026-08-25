# Explore and inspect architecture canonical use cases

## Application services

### ViewSessionService

- `OpenViewSession`
- `SetHierarchyPath`
- `SetSelection`
- `SetViewOptions`
- `CloseViewSession`

### EvidenceInspectionService

- `InspectModule`
- `InspectRelationship`
- `InspectDiagnostic`
- `InspectSourceEvidence`

### ReanalysisService

- `ReanalyzeProject`
- `ReplaceModelRevision`

## Canonical queries/commands

`OpenViewSession(model_id)` returns initial overview scene and state. `SetHierarchyPath` returns a new aggregated scene. `SetSelection` returns evidence/details. `InspectSourceEvidence` validates root containment and returns read-only text/locations. `ReanalyzeProject` runs analysis through the parent service and replaces the model only after a valid result is available.

## Failure model

Unknown model/path, stale selection, source outside root, unreadable source, invalid hierarchy path, renderer capability failure, and reanalysis failure are visible outcomes. A failed reanalysis does not discard the last valid model; it marks the session with a diagnostic.

## Architecture-neutral mapping

These intents may be browser state handlers, application services, or local HTTP operations. The canonical behavior is independent of frontend framework and renderer.
