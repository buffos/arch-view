# Project analyzer assignments and view selection architecture readiness review

## Findings

No High or Medium findings remain. The v1/v2 configuration boundary, strict
assignment and source-scope schema, glob/path safety, invocation-root
anchoring, nearest-file behavior, precedence, scoped diagnostics, cache
key/invalidation policy, scope API, UI selection behavior, and layout separation
are defined.

## Cross-document consistency

- The PRD defines the developer/CI workflow and explicitly separates layout.
- The domain model fixes configuration, assignment, resolution, source-scope,
  scope, and cache invariants.
- The use-case model distinguishes loading, validation, resolution, caching,
  and scope selection.
- The contract defines v2 JSON, glob matching, errors, API projection,
  reanalysis, and CLI mappings.
- The scenarios cover compatibility, precedence, invalid paths/config and
  filters, unavailable analyzers, include/exclude behavior, caching, recovery,
  and separation.

## Residual risks

- Source-content fingerprinting can be expensive in very large repositories and needs benchmark tuning.
- A future persistent cache would need explicit retention and invalidation policy.
- A future assignment editor must preserve the strict schema and non-sensitive-value rule.
- A future glob matcher must preserve the explicit v2 subset and avoid silently
  importing host-specific `.gitignore` behavior.

## Application synthesis gate

The application PRD and architecture summary are synchronized with the
repository-relative assignment workflow, invocation-root source filters,
combined/per-scope viewer journey, and cache boundary. Layout remains a
separate existing product capability.

## Artifact impact

- Capability truth: updated with the exact assignment and source-scope schema
  and scope-selection artifact set.
- Product truth: synchronized for configuration-driven analyzer selection and the dropdown journey.
- Architecture truth: synchronized for precedence, source filtering, cache,
  runtime, and viewer boundaries.
- Delivery truth: existing approved issues 039–043 are synchronized as
  consumers; no separate assignment implementation issue was created.

## Readiness

READY FOR ARCHITECTURE IMPLEMENTATION
