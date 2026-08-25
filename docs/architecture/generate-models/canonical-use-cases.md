# Generate architecture models canonical use cases

## Application services

### ModelNormalizationService

- `NormalizeAnalyzerResult`
- `ValidateCanonicalModel`
- `DeriveGraphProjections`

### ModelProjectionService

- `BuildHierarchyProjection`
- `AggregateProjectionRelationships`
- `GetCycleReport`
- `GetLayerPlan`

## Canonical commands

### NormalizeAnalyzerResult

Input: analyzer result and normalization policy. Output: canonical model with status and diagnostics. It validates endpoints/evidence, merges duplicates, assigns/validates stable IDs, and sorts collections.

### DeriveGraphProjections

Input: canonical model and relation-type policy. Output: cycle groups, feedback relationships, layer plan, and algorithm provenance. It never mutates canonical relationships.

### BuildHierarchyProjection

Input: canonical model and selected hierarchy path. Output: visible module/group nodes, aggregated typed relations, counts, contributors, and evidence links.

## Canonical queries

- `GetCycleReport(model_id, scope?)`
- `GetLayerPlan(model_id, algorithm?)`
- `BuildHierarchyProjection(model_id, hierarchy_path)`

## Failure model

Invalid envelope/schema, duplicate conflicting IDs, missing endpoints, missing evidence, unsupported relation type, and algorithm/resource limits are distinct diagnostics. Recoverable observation conflicts yield `partial`; invalid top-level model identity yields `failed`.

## End-to-end chains

1. Analyzer result → normalize → validate → derive cycles/layers → expose model.
2. Model → select hierarchy → aggregate relations → return traceable projection.
3. Cyclic model → preserve all relations → derive feedback set → assign layers → report cycle group.
