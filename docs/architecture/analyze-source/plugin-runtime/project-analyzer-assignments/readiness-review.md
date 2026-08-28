# Project analyzer assignments and view selection architecture readiness review

## Findings

No High or Medium findings remain. The v1/v2 configuration boundary, strict
assignment schema, path safety, nearest-file behavior, precedence, scoped
diagnostics, cache key/invalidation policy, scope API, UI selection behavior,
and layout separation are defined.

## Cross-document consistency

- The PRD defines the developer/CI workflow and explicitly separates layout.
- The domain model fixes configuration, assignment, resolution, scope, and
  cache invariants.
- The use-case model distinguishes loading, validation, resolution, caching,
  and scope selection.
- The contract defines v2 JSON, errors, API projection, reanalysis, and CLI
  mappings.
- The scenarios cover compatibility, precedence, invalid paths/config,
  unavailable analyzers, caching, recovery, and separation.

## Residual risks

- Source-content fingerprinting can be expensive in very large repositories and needs benchmark tuning.
- A future persistent cache would need explicit retention and invalidation policy.
- A future assignment editor must preserve the strict schema and non-sensitive-value rule.

## Application synthesis gate

The application PRD and architecture summary are synchronized with the
repository-relative assignment workflow, combined/per-scope viewer journey,
and cache boundary. Layout remains a separate existing product capability.

## Artifact impact

- Capability truth: updated with the exact schema and scope-selection artifact set.
- Product truth: synchronized for configuration-driven analyzer selection and the dropdown journey.
- Architecture truth: synchronized for precedence, cache, runtime, and viewer boundaries.
- Delivery truth: no issues created in this pass.

## Readiness

READY FOR ARCHITECTURE IMPLEMENTATION
