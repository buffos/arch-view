# Source facts and symbol index architecture readiness review

## Scope reviewed

The review covers the bounded source-facts child, its discovery notes and gap
analysis, the PRD, glossary, canonical domain model, use cases, API/CLI
contract, acceptance scenarios, existing analysis/model/syntax contracts, and
multi-analyzer scope orchestration.

## Findings

No High or Medium specification findings remain. The review confirms that the
source-index boundary, file/symbol/documentation contracts, span coordinates,
provenance states, extractor strategy, identity policy, multi-scope ownership,
coverage rules, deterministic serialization, and compatibility attachment are
explicit.

## Cross-document consistency

- The PRD limits this child to compact source facts and explicitly excludes
  quality policy, live watching, MCP transport, and source mutation.
- The domain model makes `SourceIndex` an optional sibling and separates
  authoritative scope snapshots from a derived combined projection.
- The use cases place extraction semantics in registered strategies and
  validation/assembly in the core service.
- The external contract fixes `arch-view.source-index/v1`, the optional
  `source_index` attachment, opaque IDs, hashes/spans, coverage/provenance,
  compact projections, and backward-compatible omission.
- The scenarios cover the user-visible file/module behavior, language
  declarations, documentation, deterministic metrics, extensibility, failure
  isolation, and token safety.
- The application PRD and architecture summary are synchronized with the
  source-index attachment and its dependency order before this review is
  closed.

## Residual implementation risks

- Initial language coverage will be uneven. Unsupported/unknown capability
  coverage must remain visible until each extractor is implemented.
- The byte/line adapter must be tested against Tree-sitter positions and
  invalid-UTF-8 inputs so evidence never drifts from the hashed file.
- Documentation precedence is language-owned and needs focused fixtures for
  Go package docs and each subsequent language; this does not change the core
  contract.
- Query indexes and compact text budgets require benchmark tuning, but they
  are read-model concerns and do not change the source-facts schema.

## Application synthesis gate

The application PRD and application architecture summary now describe the
source-index contract as the first implemented-by-future sequence step,
including its optional result attachment, per-scope/combined projection
boundary, extractor registry, and dependency on the existing analysis scope.
They continue to state that no implementation is claimed in this planning
pass.

## Artifact impact

- **Capability truth:** the source-facts child links this complete exact-spec
  set and advances to `specified`.
- **Product truth:** module inspection and future structural search have a
  stable compact source-facts target.
- **Architecture truth:** source-index ownership and attachment are separated
  from graph semantics, quality policy, and live/MCP lifecycle.
- **Delivery truth:** no implementation issues are created; issue slicing is
  deferred until the required application synthesis and future delivery
  workflow are requested.

## Readiness

READY FOR ARCHITECTURE IMPLEMENTATION
