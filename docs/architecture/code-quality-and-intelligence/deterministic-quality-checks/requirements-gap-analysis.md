# Deterministic quality checks requirements gap analysis

## Scope examined

This pass covers the bounded deterministic-quality child, the specified source
index, canonical modules/relationships/cycles/layers, existing diagnostics and
provenance, and the future viewer/export/live consumers.

## Confirmed strong areas

- **Observed in code:** canonical normalization already derives cycles, layers,
  hierarchy, relationship evidence, and stable source references.
- **Observed in the specified contract:** files, symbols, documentation,
  source spans, explicit coverage, and formula-versioned metric slots are
  available without changing architecture graph semantics.
- **Inferred from architecture:** a quality report can be an optional sibling
  of source index/model data, allowing older clients to ignore it and the
  viewer to decorate subjects without mutating module facts.
- **User-confirmed target behavior:** configurable size/complexity/documentation
  and structural architecture findings must be reproducible and evidence-backed.

## Blocking gaps resolved by this specification

| Gap | Impact | Resolution |
|---|---|---|
| “Metric” had no formula boundary | High: equal names could mean different values across languages/versions | Every metric declares a namespaced ID, typed value, formula ID/version, unit, and provenance. |
| Rules could become a central language switch | High: each new language/rule would require core edits | Register focused metric providers and rule strategies with required capabilities. |
| Threshold edge behavior was unspecified | High: CI and viewer could disagree at the boundary | Each threshold carries an explicit operator, typed value, unit, and inclusive/exclusive semantics. |
| Findings had no stable lifecycle identity | High: every run would create noisy new violations | Use opaque report-local IDs plus stable matching keys derived from rule/version/profile/subject identity. |
| Baselines could hide changed rules | High: a new formula might be silently suppressed | Baseline entries match exact rule and formula versions; changed versions require explicit migration. |
| Missing data could look like a pass or violation | High: unsupported language features would create false claims | Use `observed`, `absent`, `unknown`, `unsupported`, `partial`, and `not_evaluable` coverage. |
| SOLID could be presented as fact | High: static indicators do not prove design intent or substitutability | Mark SOLID output as `signal`, expose evidence/limitations, and prohibit exact-violation wording. |
| Findings could be conflated with analyzer errors | Medium: consumers could not distinguish bad code from analysis failure | Keep quality findings and diagnostics as separate collections with links where relevant. |
| Aggregate scopes could invent architecture constraints | High: independent language graphs do not imply cross-language semantics | Evaluate per scope by default; require explicit validated aggregate facts for cross-scope rules. |

## Deferrable implementation details

- The registry can be in-process, generated, or backed by a compiled plugin;
  the strategy contract is stable.
- The first complexity provider may support only Go and return explicit
  unsupported coverage for other languages.
- Severity names, UI colors, and CI exit policy are configurable projections;
  the finding contract keeps severity distinct from assessment kind.
- A future history provider may emit change-proneness metrics only when its
  commit/repository input is pinned. It is not part of current static source
  determinism.

## Implementation gaps audited after issue delivery

The initial delivery audit found two gaps between the specified contract and
the shipped behavior. Both are now resolved:

| Gap | Resolution | Verification |
|---|---|---|
| The five SOLID rules were registered, but the real Go analyzer did not publish `source:solid.structure`, so a full-profile report classified them as unsupported. | The Go source extractor now derives syntax-observable structural counts and the Go observation builder requests the capability. | Extractor, Go analyzer end-to-end, and real repository report checks show all five SOLID rules with `observed` coverage for valid Go source. |
| Baseline lifecycle functions existed, but users had no command to create a baseline from a report; only `--quality-baseline` consumption was exposed. | Added `arch-view quality baseline` with finding/stable-key selection, `--all-active`, validation, and overwrite protection. | CLI test covers analysis-envelope input, baseline output, and the overwrite guard. |

## Exact assumptions

- Default threshold operator is `greater_than` for maximum limits and
  `less_than` for minimum limits only when explicitly declared; no implicit
  equality behavior is allowed.
- A callable line metric uses an extractor-supplied body span and its formula
  version, not a generic nearest-brace heuristic. If no body span exists, the
  rule is `not_evaluable`.
- Cyclomatic complexity is `1 + decision_points` for the provider's declared
  decision vocabulary. Boolean-operator counting, `case` treatment, and
  language-specific short-circuit semantics are provider/formula decisions,
  not hidden core behavior.
- Coupling counts distinct reported architecture module targets. Whether
  external/reference modules count is a profile parameter and appears in the
  evaluation fingerprint.
- Documentation coverage counts only subjects whose visibility is observed and
  whose documentation capability is supported. Unknown/unsupported subjects are
  excluded from the denominator and reported in coverage, never treated as
  documented or undocumented.
- Human-facing summaries of file line thresholds derive their count and
  affected-file filter from the scope/profile quality report. Missing, partial,
  unknown, or unsupported report coverage is not silently presented as zero.
- A cycle finding is exact when it is derived from the canonical reported graph
  and the selected graph projection; no cycle is inferred from file paths.
- Layer-direction and forbidden-dependency findings require an explicit policy
  mapping/constraint. The engine cannot infer the intended architecture from
  names alone.
- SOLID signals are advisory structural patterns. None is an exact proof of an
  SRP, OCP, LSP, ISP, or DIP violation.

## Readiness conclusion

No High or Medium specification gaps remain for deterministic quality checks.
The remaining work is implementation of providers/rules and language-specific
fixtures, not a missing product or contract decision.

## Artifact impact

- **Capability:** this exact-spec set defines metrics, rules, findings,
  configuration, evidence, coverage, and lifecycle.
- **Product:** quality feedback is repeatable and distinguishable from
  subjective architectural review.
- **Architecture:** quality is a sibling consumer of source index/model facts;
  the live/MCP child consumes reports without owning their semantics.
- **Delivery:** no implementation issues are created in this planning pass.
