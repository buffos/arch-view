# Generate architecture models discovery notes

## Purpose

Convert analyzer observations into one deterministic, language-neutral architecture model that can be consumed by exploration, layout preparation, and headless outputs.

## Reference observations

The read-only reference implementation separates source extraction from graph projection. It derives namespace segments from Clojure names, aggregates edges when a view collapses several modules, classifies edges using abstract-module metadata, and computes feedback edges, cycles, and layers for a view. Those ideas are useful, but the Clojure namespace string must not remain the model's structural representation.

## Target boundary

The capability receives validated analyzer observations and produces:

- project and analyzer metadata;
- canonical module records;
- explicit hierarchy paths;
- typed semantic relationships;
- source references, evidence, tags, and confidence;
- external or unresolved dependency references without inventing module nodes;
- deterministic aggregation and validation diagnostics;
- derived cycle and layer information for downstream projections.

It stops before interaction state, scene coordinates, renderer-specific graphics, and export packaging.

## Confirmed model decisions

1. Hierarchy is structural data on a module (`hierarchy []string`), not a dependency edge. Containment must not create false dependency cycles or affect dependency layering.
2. A module is the canonical node. A module may have many source files and many source references.
3. Module identity is a stable opaque project-scoped identifier. Display names and language-specific identities are separate fields; hierarchy is never recovered by splitting punctuation in an ID.
4. Relationships are typed records. `depends_on` is the first supported type; `calls`, `implements`, and `references` remain extensible future types.
5. Relationship direction is explicit: `from` depends on `to`. Layering and diagnostics use this semantic direction rather than guessing from an arrow drawn by a renderer.
6. External, standard-library, and unresolved targets are retained as dependency references or diagnostics, but are not silently promoted to project module nodes under the default local scope.
7. Analyzer-provided evidence is preserved through normalization. When several observations describe one module or relationship, their source references are merged and deduplicated rather than discarded.
8. Conflicting identity or metadata observations do not make the whole result disappear. The host retains the usable canonical value, records a diagnostic, and marks confidence or provenance as needed.
9. The canonical graph keeps every accepted semantic relationship, including cycles. Cycle groups, feedback edges, and acyclic layering inputs are derived projections and never delete or rewrite canonical edges.
10. Layers are derived analysis data, not part of stable module identity. A layer result records its algorithm/provenance so a future layout strategy can coexist with the canonical model.
11. Collapsed or hierarchical views aggregate edges by typed endpoint pair while retaining counts and contributing relationship/evidence IDs for traceability.
12. Host normalization validates references, deduplicates exact records, and sorts modules, relations, evidence, tags, and diagnostics deterministically.
13. Language concepts such as abstract classes, protocols, traits, interfaces, and Clojure polymorphism are metadata or tags. The core model does not hard-code one language's classification rules.

## Actors and inputs

- The analyzer host supplies project metadata and analyzer observations.
- The model validator checks identity, hierarchy, endpoint, evidence, and diagnostic invariants.
- The graph engine derives cycles, feedback edges, layers, and aggregation projections.
- Exploration and export capabilities consume the canonical model and derived projections without parsing source syntax.

## Child territories

- Canonical identity and normalization
- Relationship and evidence aggregation
- Cycle and layer derivation
- Hierarchical projection and traceability
- Model validation and versioning

## Open questions for exact specification

- Exact serialized field names, required/optional fields, and schema version policy.
- The deterministic algorithm for generating or validating project-scoped module IDs.
- The endpoint shape and retention rules for external, standard-library, and unresolved references.
- The first cycle and feedback-edge algorithm, including large-graph performance limits.
- Layer direction, numbering, labels, and behavior when the graph is cyclic or disconnected.
- Conflict precedence when analyzers provide incompatible names, tags, hierarchy paths, or confidence values.
- Whether aggregation exposes contributor IDs directly or through a separate evidence index.

## References

- [Reference architecture projection](https://github.com/unclebob/arch-view)
- [Reference layer assignment](https://github.com/unclebob/arch-view)
- [Reference edge classification](https://github.com/unclebob/arch-view)
- [Analyze source code discovery notes](../analyze-source/discovery-notes.md)
- [Parent planning concept](../../../.okf/capabilities/generate-models.md)
