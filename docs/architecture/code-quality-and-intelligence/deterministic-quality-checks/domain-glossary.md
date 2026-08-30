# Deterministic quality checks domain glossary

| Term | Category | Definition | Distinction |
|---|---|---|---|
| Quality profile | policy object | Versioned set of enabled rules, parameters, severity mappings, constraints, and baseline reference. | It is separate from analyzer, layout, assignment, and live watcher configuration. |
| Metric provider | strategy/plugin | Registered component that calculates a named versioned metric from source-index/model facts. | It is not a quality rule and does not decide severity. |
| Metric fact | fact object | Typed value with metric ID, formula ID/version, unit, subject, and provenance. | A metric is not a finding until a rule evaluates it. |
| Quality rule | strategy/plugin | Registered evaluator that consumes declared capabilities and emits findings or coverage. | New rules are additive; the engine does not switch on rule names. |
| Rule capability | contract value | A source/model fact requirement such as `source:size`, `source:visibility`, `graph:cycles`, or `metric:cyclomatic`. | Missing capability makes a rule not evaluable, not passing. |
| Threshold | policy value | Typed comparison operator and limit applied to one metric. | Its boundary semantics are explicit; it is not a UI color. |
| Assessment kind | classification | `exact` for mechanically evaluated facts or `signal` for advisory structural heuristics. | It prevents SOLID indicators from being presented as proofs. |
| Quality finding | business object | Evidence-backed result associated with a subject and rule evaluation. | It is separate from analyzer diagnostics and source facts. |
| Finding key | reconciliation value | Stable matching key used to carry a finding across reports. | It is not a report-local finding ID and is not derived solely from a line range. |
| Quality report | business object | Immutable result of evaluating one source/model snapshot with one profile. | A live service may publish revisions of reports but does not mutate an old report. |
| Coverage result | reporting object | Explicit observed/not-evaluable/unsupported state for a rule or metric. | It distinguishes missing analysis from a clean result. |
| Severity | policy projection | Configured impact class such as `info`, `warning`, `error`, or `blocker`. | Severity is independent of exact-vs-signal assessment kind. |
| Baseline | policy object | Set of acknowledged finding keys tied to rule/profile/formula versions. | It suppresses reporting state, not detection or evidence. |
| Suppression | policy result | A finding status indicating an explicit matching baseline/policy entry. | It is visible in reports and does not mean resolved. |
| Constraint | architecture policy | Explicit allowed/forbidden dependency or layer relation rule. | It expresses intended architecture; the graph alone cannot infer it. |
| Coupling | derived metric | Count/formula over distinct reported module dependencies in a selected projection. | It is not a universal measure of design quality. |
| Cyclomatic complexity | derived metric | Provider-defined `1 + decision_points` for a callable. | Decision vocabulary and formula version are language-specific. |
| Documentation coverage | derived metric | Ratio/count over eligible visible subjects and their documentation status. | It measures presence/coverage, not prose usefulness. |
| SOLID signal | advisory result | Evidence-backed structural indicator associated with a SOLID principle. | It is never an exact proof of violation. |
| Rule revision | version value | Version of rule implementation and parameters that determines finding meaning. | A changed revision may require a new baseline match. |
| Evaluation fingerprint | identity value | Digest of input snapshot IDs, profile, rule versions, formula versions, and relevant options. | It provides reproducibility and cache identity. |
| Finding evidence | evidence object | Source spans, module/relationship refs, metric facts, and provenance supporting a result. | Evidence explains a finding; it does not expand its scope. |
| Not evaluable | status | The required fact/capability is absent, unknown, unsupported, or invalid. | It is not pass, fail, or suppression. |

Canonical vocabularies:

- Assessment: `exact`, `signal`.
- Finding status: `active`, `suppressed`, `baseline`, `resolved`,
  `not_evaluable`.
- Coverage: `observed`, `absent`, `unknown`, `unsupported`, `partial`.
- Severity: `info`, `warning`, `error`, `blocker`.
