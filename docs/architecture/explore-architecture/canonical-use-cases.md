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
- `InspectImports`
- `InspectSourceEvidence`

### ReanalysisService

- `ReanalyzeProject`
- `ReplaceModelRevision`

## Canonical queries/commands

`OpenViewSession(model_id)` returns an initial local-first overview scene and state. `SetHierarchyPath` returns a new aggregated scene. `SetSelection` returns evidence/details. `SetViewOptions` can hide, aggregate, or expand non-local references and filter their scopes without changing the canonical model. `InspectImports` returns the selected module/group's individual import relationships, target scope, confidence, counts, and source evidence as a list-oriented result. `InspectSourceEvidence` validates root containment and returns read-only text/locations. `ReanalyzeProject` runs analysis through the parent service and replaces the model only after a valid result is available.

## Failure model

Unknown model/path, stale selection, source outside root, unreadable source, invalid hierarchy path, renderer capability failure, and reanalysis failure are visible outcomes. A failed reanalysis does not discard the last valid model; it marks the session with a diagnostic.

## Architecture-neutral mapping

These intents may be browser state handlers, application services, or local HTTP operations. The canonical behavior is independent of frontend framework and renderer.
