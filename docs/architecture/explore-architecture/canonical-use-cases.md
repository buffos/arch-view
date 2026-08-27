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

### LayoutConfigurationService

- `DiscoverProjectLayoutConfig`
- `ListLayoutOptions`
- `ApplyLayoutProfile`
- `SaveActiveProjectLayoutConfig`
- `SaveAsProjectLayoutConfig`
- `ResetLayoutProfile`

## Canonical queries/commands

`OpenViewSession(model_id)` returns an initial local-first overview scene and state. `SetHierarchyPath` returns a new aggregated scene. `SetSelection` returns evidence/details. `SetViewOptions` can hide, aggregate, or expand non-local references and filter their scopes without changing the canonical model. `InspectImports` returns the selected module/group's individual import relationships, target scope, confidence, counts, and source evidence as a list-oriented result. `InspectSourceEvidence` validates root containment and returns read-only text/locations. `ReanalyzeProject` runs analysis through the parent service and replaces the model only after a valid result is available.

`DiscoverProjectLayoutConfig(target_directory)` checks the target directory and its ancestors for the nearest `.archview.json`, returning the effective profile, origin, active file path, and any validation diagnostic. `ListLayoutOptions()` returns the pinned ELK adapter's grouped, typed option catalog. `ApplyLayoutProfile(profile)` validates the selected algorithm/options, applies each option at the graph-element level declared by its catalog target (`PARENTS` root, `NODES` eligible visible nodes, and `EDGES` eligible visible edges), and recalculates the current scene's node positions and edge routes without changing canonical model facts; it clears manual positions for the affected hierarchy path. `SaveActiveProjectLayoutConfig(profile)` validates and atomically overwrites the exact active discovered `.archview.json`; it never creates a file when no active file exists and returns `save_as_required` in that case. `SaveAsProjectLayoutConfig(profile, destination_folder)` is the only operation that accepts a custom destination; it writes the fixed `.archview.json` filename there atomically after explicit confirmation and makes it active for the current session. A custom file outside the target's ancestor chain is not automatically rediscovered by a later project session until an explicit configuration-selection capability exists. A model-only session may apply a profile but cannot persist a project file. `ResetLayoutProfile` returns the active session to built-in defaults; it does not silently delete a discovered project configuration.

## Failure model

Unknown model/path, stale selection, source outside root, unreadable source, invalid hierarchy path, invalid/unsupported layout configuration, renderer capability failure, configuration write failure, and reanalysis failure are visible outcomes. A malformed or unsupported nearest configuration does not fall through silently to a farther file; the active session uses safe defaults with a clear diagnostic. A failed reanalysis does not discard the last valid model; it marks the session with a diagnostic.

## Architecture-neutral mapping

These intents may be browser state handlers, application services, or local HTTP operations. The canonical behavior is independent of frontend framework and renderer.

## Current delivery status

The local HTTP viewer now maps the layout configuration use cases to catalog,
config, apply, reset, Save, and Save As endpoints. Automated contract,
resolver, and browser visual checks are complete for issue 007. Issue 008
extends the validated root-level parent-option tranche and its visual review
is complete. Issue 009 adds target-aware mapping for the bounded simple
node/edge priority tranche; automated verification and visual review are
complete.

Issue 016 implements the bounded presentation extension: a layered profile may
request ELK spline routing, self-contained HTML embeds the profile/catalog and
pinned runtime, and the viewer/export path consumes the same renderer-neutral
cubic geometry without changing model facts. Browser Download SVG captures the
current canvas; Go static SVG remains deterministic orthogonal. Automated
verification is complete; visual review remains the final gate.
