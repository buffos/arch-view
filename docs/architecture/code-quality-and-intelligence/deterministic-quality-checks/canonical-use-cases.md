# Deterministic quality checks canonical use cases

## Application services

### QualityCatalog

- `RegisterMetricProvider`
- `RegisterQualityRule`
- `ListQualityCapabilities`
- `ListQualityRules`

### QualityEvaluationService

- `ValidateQualityProfile`
- `EvaluateQualityProfile`
- `EvaluateArchitectureConstraints`
- `NormalizeQualityReport`
- `CompareQualityReports`

### QualityPolicyService

- `ValidateBaseline`
- `CreateBaselineEntry`
- `ResolveSuppression`

### QualityQueryService

- `ListFindings`
- `GetFindingEvidence`
- `GetQualityCoverage`

## `RegisterMetricProvider` / `RegisterQualityRule` — commands

**Input:** provider/rule ID and version, declared capabilities, assessment kind,
parameter schema, and strategy implementation or adapter.

**Rules:** duplicate same-version registration is idempotent; conflicting
implementations are rejected; exact rules must declare observable required
inputs; signal rules must declare their limitations. Registration is additive
and does not require a central language/rule switch.

**Output:** immutable catalog entry and capability descriptor.

## `ValidateQualityProfile` — command

**Input:** `QualityProfile`, active catalog, available source/model capability
coverage, and optional baseline.

**Rules:** validate schema/version, rule existence/version, typed parameters,
threshold units/operators, severity mappings, constraint selectors, baseline
references, and profile identity. Unknown rules or invalid parameters produce
configuration diagnostics and are not silently ignored.

**Outcome:** immutable validated profile or structured diagnostics. Validation is
read-only and does not evaluate code.

## `EvaluateQualityProfile` — command

**Input:** validated profile, authoritative source-index/model snapshots,
registered metric providers/rules, and optional previous report for comparison.

**Responsibilities:**

1. Resolve required capabilities and compute versioned metrics.
2. Evaluate rules only against observed compatible facts.
3. Emit exact findings for true predicates and signal findings for declared
   advisory rules.
4. Emit explicit coverage/not-evaluable results for missing inputs.
5. Attach source/module/relationship/metric evidence.
6. Apply severity and exact baseline/suppression matching.
7. Assign opaque IDs, stable finding keys, canonical ordering, evaluation
   fingerprint, and report digest.

**Outcome:** immutable `QualityEvaluation`. A provider failure affects its
declared metric/rule coverage and diagnostics; it does not erase unrelated
findings or source/model facts.

## `EvaluateArchitectureConstraints` — command

**Input:** canonical reported graph, explicit forbidden-dependency/layer
constraints, scope selector, and profile.

**Rules:** evaluate only reported relationships and explicit layer assignments;
do not infer intended architecture from names, directories, or dependency
direction. Cross-scope constraints require an explicit aggregate graph and
provenance.

**Outcome:** exact structural findings or not-evaluable coverage.

## `CompareQualityReports` — query

**Input:** two reports with compatible profile/rule/formula context.

**Output:** added, unchanged, suppressed, and resolved finding transitions,
matched by `finding_key` and exact rule/formula versions.

**Rules:** absence from a partial/not-evaluable report is not a resolved finding.
Changed rule or formula versions are incompatible unless a migration policy is
explicitly supplied.

## `ValidateBaseline` / `CreateBaselineEntry` — commands

**Input:** baseline, finding key, exact rule/profile/formula versions, reason,
and optional owner.

**Rules:** a baseline entry must match an existing finding or an explicit
approved migration; it never changes the finding's evidence or recalculates a
metric. Old versions cannot suppress new meanings.

**Outcome:** immutable baseline revision and a visible suppression result on the
next report.

## `ListFindings` — query

**Input:** report/snapshot selector, scope, subject, rule, assessment kind,
severity, status, and bounded pagination.

**Output:** compact findings with message, status, evidence summary, observed
metrics, provenance, coverage, and a continuation cursor.

**Rules:** default order is scope/subject/rule/severity/finding key. The query
does not return full source context unless explicitly requested. A human-facing
projection may summarize active `source:file.max-lines` findings by scope and
configured limit and provide a file-only filter, but it must use the report's
findings and coverage rather than re-evaluating raw file line counts.

## `GetFindingEvidence` — query

**Input:** finding ID/key and explicit source-context byte/line budget.

**Output:** source spans, module/relationship refs, metric facts, diagnostics,
and optional bounded context through the existing source-inspection boundary.

**Rules:** read-only, scope-bound, path-safe, and no automatic fix. A signal's
limitations are included with its evidence.

## Failure model

- `QualityProfileInvalid`
- `QualityRuleUnknown`
- `QualityRuleUnsupported`
- `MetricInputUnavailable`
- `MetricFormulaInvalid`
- `QualityRuleEvaluationFailed`
- `QualityFindingInvalid`
- `QualityEvidenceInvalid`
- `QualityBaselineInvalid`
- `QualityReportDigestMismatch`
- `QualityScopeUnavailable`

Invalid configuration fails validation before evaluation. Provider/rule failures
are isolated and surfaced as coverage/diagnostics. A report can be partial but
must never present missing input as a clean pass.

## Architecture-neutral mapping

The quality engine can run in-process or behind a future plugin host. The
stable boundary is profile validation, provider/rule capabilities, immutable
metric/finding output, explicit coverage, and evidence. Viewer, export, CLI,
and MCP adapters consume the report; none owns threshold or SOLID semantics.
