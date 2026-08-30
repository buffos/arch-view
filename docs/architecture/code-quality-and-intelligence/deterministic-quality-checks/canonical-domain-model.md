# Deterministic quality checks canonical domain model

## Modeling boundary

This model owns reproducible evaluation of source-index and architecture facts.
It does not own parsing, source-index identity, live freshness, MCP transport,
or source mutation. Quality findings decorate existing subjects; they do not
change module, relationship, or symbol meaning.

## QualityProfile

```text
QualityProfile {
  schema_version: "arch-view.quality/v1"
  profile_id: NamespacedId
  profile_version: string
  enabled_rules: RuleBinding[]
  severity_policy: SeverityPolicy
  constraints: ArchitectureConstraint[]
  baseline?: BaselineRef
  extensions: ExtensionBlock[]
}

RuleBinding {
  rule_id: NamespacedId
  rule_version: string
  enabled: boolean
  parameters: TypedConfigBlock
  severity?: "info" | "warning" | "error" | "blocker"
}

TypedConfigBlock {
  namespace: NamespacedId
  schema_version: string
  payload: typed JSON value
}
```

Rule IDs, parameters, and config blocks are open. A profile validator resolves
them through the registered rule catalog and rejects unknown/invalid bindings
with structured configuration diagnostics; it does not silently disable them.

## QualityEvaluation

```text
QualityEvaluation {
  schema_version: "arch-view.quality/v1"
  evaluation_id: OpaqueId
  source_snapshot_ids: OpaqueId[]
  model_revision?: string
  profile_id: NamespacedId
  profile_version: string
  provider_identities: ProviderIdentity[]
  coverage: QualityCoverage[]
  metrics: MetricFact[]
  findings: QualityFinding[]
  diagnostics: QualityDiagnostic[]
  evaluation_fingerprint: ContentDigest
  report_digest: ContentDigest
  extensions: ExtensionBlock[]
}
```

`QualityEvaluation` is an optional sibling attachment named `quality_report`
on analysis/model output. It is immutable for its input snapshot/profile. A
live service may publish later evaluations, but does not mutate this report.

## MetricFact

Quality metrics reuse the typed metric shape from the source-index contract:

```text
MetricFact {
  id: OpaqueId
  subject_ref: EntityRef
  metric_id: NamespacedId
  value: { kind: "integer" | "decimal" | "boolean" | "text", value: scalar }
  unit?: NamespacedId
  formula_id: NamespacedId
  formula_version: string
  provenance: FactProvenance
  extensions: ExtensionBlock[]
}
```

Intrinsic file size remains in `FileRecord.size`; a quality metric may reference
it or emit a derived callable/module metric, but it must not contradict the
intrinsic fact. Formula version is mandatory for complexity, coupling,
coverage, duplication, and any other derived value.

## QualityFinding

```text
QualityFinding {
  id: OpaqueId                         // report-local, opaque
  finding_key: string                  // stable matching key
  rule_id: NamespacedId
  rule_version: string
  assessment_kind: "exact" | "signal"
  status: "active" | "suppressed" | "baseline" | "resolved" | "not_evaluable"
  severity: "info" | "warning" | "error" | "blocker"
  subject_ref: EntityRef
  message_code: NamespacedId
  message: string
  observed_metric_ids: OpaqueId[]
  comparison?: Comparison
  evidence: FindingEvidence
  limitations?: string[]
  suppression?: SuppressionInfo
  provenance: FactProvenance
  extensions: ExtensionBlock[]
}

Comparison {
  operator: "greater_than" | "greater_or_equal" | "less_than" |
            "less_or_equal" | "equal" | "not_equal"
  observed_metric_id: OpaqueId
  limit: { kind: "integer" | "decimal" | "boolean" | "text", value: scalar }
  unit?: NamespacedId
}

FindingEvidence {
  source_spans: SourceSpan[]
  entity_refs: EntityRef[]
  relation_refs: EntityRef[]
  metric_refs: OpaqueId[]
  diagnostic_refs: string[]
}
```

`active` means the predicate is true in this report. `suppressed` and
`baseline` preserve a detected finding while changing its reporting projection.
`not_evaluable` is used only when a rule was requested but required facts were
not observed. `resolved` is used by an optional report-delta/history projection;
it is not invented from the absence of a finding in one isolated report.

The stable `finding_key` is derived from rule ID/version, assessment kind,
profile/policy fingerprint, scope-qualified subject stable identity, and an
optional deterministic occurrence anchor. It is not derived solely from a
line number. The report-local `id` can change between reports.

## QualityCoverage

```text
QualityCoverage {
  rule_id: NamespacedId
  rule_version: string
  status: "observed" | "absent" | "unknown" | "unsupported" | "partial" | "not_evaluable"
  required_capabilities: NamespacedId[]
  available_capabilities: NamespacedId[]
  subject_count?: integer >= 0
  evaluated_count?: integer >= 0
  reason?: string
  provenance: FactProvenance
}
```

Coverage is not a pass/fail finding. A rule with unsupported complexity on a
language has `unsupported`/`not_evaluable` coverage rather than a clean result.

## ArchitectureConstraint

```text
ArchitectureConstraint {
  id: NamespacedId
  kind: "forbidden_dependency" | "layer_direction" | "unknown"
  parameters: TypedConfigBlock
  provenance: FactProvenance
  extensions: ExtensionBlock[]
}
```

V1 typed parameters resolve module selectors from explicit module IDs, stable
path globs, tags, or layer assignments. A selector cannot infer intent from a
directory name without being present in the profile. The graph contributes
only reported relationships.

## Baseline and suppression

```text
Baseline {
  schema_version: "arch-view.quality-baseline/v1"
  baseline_id: NamespacedId
  entries: BaselineEntry[]
  extensions: ExtensionBlock[]
}

BaselineEntry {
  finding_key: string
  rule_id: NamespacedId
  rule_version: string
  profile_id: NamespacedId
  formula_versions: { metric_id: NamespacedId, version: string }[]
  reason: string
  owner?: string
}

SuppressionInfo {
  baseline_id?: NamespacedId
  reason: string
  exact_version_match: boolean
}
```

An entry matches only the specified rule/profile/formula versions. Changed
versions remain visible until explicitly baselined; an old entry cannot hide a
new rule meaning.

## Strategies and registries

```text
MetricProvider {
  ID() -> NamespacedId
  Version() -> string
  Capabilities() -> CapabilityDescriptor[]
  Compute(EvaluationContext) -> MetricBatch
}

QualityRule {
  ID() -> NamespacedId
  Version() -> string
  AssessmentKind() -> "exact" | "signal"
  RequiredCapabilities() -> NamespacedId[]
  Evaluate(EvaluationContext, MetricBatch) -> RuleResult
}
```

The engine composes registered providers/rules through focused interfaces. A
new language metric or rule is an added strategy; the evaluator is closed for
modification. Rule implementations must be substitutable: they may return
findings, coverage, and diagnostics through the same result contract and may
not strengthen input preconditions beyond their declared capabilities.

## Initial deterministic formulas

- `source:file.line_count`: the intrinsic `FileSize.line_count`.
- `source:callable.body_line_count`: physical lines in the extractor-supplied
  body span, with formula version and source hash.
- `source:callable.cyclomatic_complexity`: `1 + decision_points`; the provider
  declares the language decision vocabulary and handling of boolean operators,
  cases, and short-circuit constructs.
- `source:callable.max_nesting_depth`: maximum provider-defined control-flow
  nesting depth.
- `source:documentation.coverage`: documented eligible visible subjects / all
  eligible visible subjects with supported documentation and observed
  visibility. Unknown/unsupported subjects are reported in coverage, not
  counted as documented or undocumented.
- `architecture:module.efferent_coupling`: distinct reported target modules
  after the profile's scope/external policy.
- `architecture:module.afferent_coupling`: distinct reported source modules.
- `architecture:module.cycle_participation`: canonical cycle membership/count.

Provider-specific formula details are part of the metric version. The core
never silently substitutes a generic brace/comment heuristic.

## SOLID signal boundary

Static facts can support structural indicators such as method/dependency counts,
interface size, concrete dependency edges, type switches, and hierarchy shape.
They cannot prove responsibility boundaries, extension intent, behavioral
substitutability, or appropriate abstraction. Therefore `signal:solid.*` rules
must emit `assessment_kind: signal`, evidence, and limitations. No SOLID rule
may emit an exact violation or use proof language. A compiler/type-checker
diagnostic remains a language/contract fact, not proof of an LSP violation.

## Canonicalization and invariants

- Profile rules, providers, capabilities, constraints, baselines, metrics, and
  findings are sorted by stable IDs and declared ordering keys.
- Every exact finding references observed metrics/relationships or source
  facts; every signal exposes its structural evidence and limitation.
- Every finding subject and evidence reference resolves in the input scope or
  carries explicit unresolved/coverage status.
- Threshold units and value kinds match the observed metric; no numeric coercion
  or missing-value default is implicit.
- The evaluation fingerprint includes source/model snapshot IDs, profile and
  baseline digests, rule/provider versions, formula versions, and options.
- The report digest excludes operational timestamps and is calculated after
  canonical sorting. Equal inputs produce equal semantic reports.
