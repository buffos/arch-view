# Deterministic quality checks architecture readiness review

## Scope reviewed

The review covers the bounded deterministic-quality child, specified source
facts, canonical model graph/layer outputs, existing diagnostics/provenance,
and future viewer/export/live consumers.

## Findings

No High or Medium specification findings remain. The rule/metric registry,
profile ownership, formula/version identity, exact-vs-signal boundary,
threshold semantics, finding identity, coverage, baseline behavior, evidence,
scope isolation, and deterministic report contract are explicit.

## Cross-document consistency

- The PRD limits this child to reproducible metrics/rules/findings and excludes
  parsing, live watching, MCP transport, source editing, and subjective
  architectural approval.
- The domain model makes quality a versioned optional report sibling and keeps
  metrics, findings, coverage, constraints, and baselines distinct.
- The use cases put language/metric/rule behavior behind registries and keep
  profile validation, evidence, and report assembly in the core service.
- The contract fixes `arch-view.quality/v1`, typed profile blocks, exact and
  signal examples, optional attachment, query mappings, and omission semantics.
- The scenarios cover threshold boundaries, complexity/documentation coverage,
  graph constraints, finding revisions, baselines, open/closed registration,
  partial failures, determinism, all SOLID labels, and the scope-safe human
  projection of file line-threshold findings.
- The application PRD and architecture summary identify the source-index child
  as the input boundary and keep live/MCP as downstream consumers.

## Residual implementation risks

- Complexity and nesting formulas need language-specific fixtures and must
  declare their decision vocabulary/version before enabling cross-language
  comparisons.
- Layer and forbidden-dependency policy needs a user-facing configuration
  design, but the constraint contract already prevents name-based inference.
- SOLID signals require conservative thresholds and clear UX copy; this is a
  calibration concern, not a reason to call them exact violations.
- CI exit-code policy and viewer color palettes are projections and must not be
  encoded as finding semantics.

## Application synthesis gate

The application PRD and architecture summary are synchronized with the quality
report as a future sibling consumer of source-index/model facts. They preserve
the distinction between deterministic findings and advisory SOLID signals and
continue to claim no implementation in this planning pass.

## Artifact impact

- **Capability truth:** this exact-spec set defines profile, metric, rule,
  finding, baseline, coverage, and report behavior.
- **Product truth:** configurable deterministic checks and honest signal output
  are now explicit future behavior.
- **Architecture truth:** quality remains independent from source extraction,
  live freshness, transport, and graph semantics.
- **Delivery truth:** no implementation issues are created in this pass.

## Readiness

READY FOR ARCHITECTURE IMPLEMENTATION
