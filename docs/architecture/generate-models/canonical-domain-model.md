# Generate architecture models canonical domain model

## Modeling principles

The model is a durable semantic interchange object, not a database schema or renderer scene. Structural containment, semantic relationships, evidence, and derived graph calculations remain separate.

## CanonicalModel aggregate

Fields:

- `schema_version`
- `model_id`
- `status`
- `project` (`root_label`, `boundary`, `language`)
- `analyzer` (`id`, `version`, `api_version`)
- `modules[]`
- `references[]`
- `source_references[]`
- `relationships[]`
- `diagnostics[]`
- `derived` (`cycles`, `feedback_relationship_ids`, `layers`, `algorithm_provenance`)

## Entities and value objects

### Module

`id`, `kind`, `language`, `name`, `display_name`, `hierarchy[]`, `source_reference_ids[]`, `tags[]`, `metadata`.

Invariant: `id` is unique; hierarchy segments are ordered non-empty strings; all source IDs resolve.

### Reference

`id`, `name`, `scope`, `language?`, `metadata`. Scope is one of `external`, `standard_library`, `unresolved`, `dynamic`.

### Relationship

`id`, `type`, `from_module_id`, exactly one of `to_module_id` or `to_reference_id`, `source_reference_ids[]`, `confidence?`, `metadata`.

The first type is `depends_on`. Future types are open strings governed by capability/version negotiation.

### SourceReference

`id`, `path`, `start?`, `end?`, `symbol?`, `kind`. Paths are repository-relative with normalized separators; line/column values are one-based.

### Diagnostic

`id`, `code`, `severity`, `message`, `recoverable`, `subject?`, `source_reference_ids[]`, `metadata`.

### Derived projections

- `CycleGroup`: `id`, sorted `module_ids`, sorted `relationship_ids`, optional canonical paths.
- `LayerPlan`: `algorithm`, `algorithm_version`, sorted `{layer, module_ids}`, `feedback_relationship_ids`.
- `HierarchyProjection`: selected hierarchy path, visible modules/groups, aggregated relationships, contributor IDs.

## Invariants and policies

- No hierarchy edge is included in dependency cycles or layer calculations.
- Canonical relationship records are never removed to derive a layer plan.
- Exact duplicates are merged; evidence IDs are unioned and sorted.
- Aggregated relations retain count and contributor relationship IDs.
- Invalid endpoint/evidence references create diagnostics; only irreparably malformed models fail.
- Canonical collections are sorted by stable IDs/type and never depend on map iteration order.

## Domain events and extension points

Conceptual events: `ModelNormalized`, `ModelPartiallyNormalized`, `CycleProjectionDerived`, `LayerPlanDerived`, `HierarchyProjectionCreated`.

Extension points: relation types, metadata/tags, alternate cycle/layer algorithms, additional projections, and schema versions.
