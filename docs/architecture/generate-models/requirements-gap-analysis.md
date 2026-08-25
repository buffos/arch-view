# Generate architecture models requirements gap analysis

## Summary

The capability is specified. The shared model schema, normalization rules, projection semantics, and external contract are documented and readiness-reviewed; residual risks are implementation/benchmark work.

## Resolved gaps

| Area | Resolution |
|---|---|
| Structural hierarchy | Explicit `hierarchy []string`; never inferred from punctuation in an ID. |
| Graph granularity | Stable module/package-level nodes with many source files. |
| Relationship semantics | Typed relationships, beginning with `depends_on` and explicit `from`/`to` direction. |
| Containment | Separate from semantic graph edges and dependency layering. |
| Non-local targets | Retained as references or diagnostics, not default project module nodes. |
| Evidence | Source references survive normalization and aggregation. |
| Aggregation | Merge duplicate module/relationship observations deterministically and retain traceability. |
| Cycles | Canonical edges remain intact; cycles and feedback edges are derived projections. |
| Layers | Derived, algorithm-provenanced data rather than stable identity. |
| Language-specific semantics | Tags and metadata, not hard-coded core types. |
| Determinism | Validate, deduplicate, and sort all stable collections in the host. |
| Failure behavior | Preserve usable partial results and attach diagnostics/confidence for recoverable conflicts. |

## Specification closure and residual risks

### Canonical schema

Defined in [the canonical contract](canonical-api-cli-contract.md), including the `arch-view.model/v1` envelope, required fields, endpoint invariants, and compatibility behavior.

### Identity and references

Defined as stable project-scoped module IDs, structured hierarchy, explicit module/reference targets, and scopes for external, standard-library, unresolved, and dynamic references.

### Graph derivation

The first implementation must benchmark cycle detection, feedback-edge selection, and layer assignment on large graphs; algorithm/version provenance and deterministic output are specified.

### Aggregation and provenance

Contributor/evidence IDs, deterministic merging, and partial diagnostics are specified; conflict behavior remains a fixture/testing risk.

### Projection compatibility

The renderer-neutral hierarchy projection and versioned model contract are specified for viewer/export consumers without coupling to one renderer.

## Readiness

The capability has passed the architecture specification pipeline and readiness review. It may enter implementation/issue slicing after the application synthesis gate is verified.
