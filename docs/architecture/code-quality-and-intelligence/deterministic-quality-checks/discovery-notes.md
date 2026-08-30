# Deterministic quality checks discovery notes

## Purpose

Turn the source-index facts and canonical architecture graph into reproducible
quality information that can be shown in the viewer, exported to CI, and
returned to future tools. The result must distinguish a mechanically evaluated
rule from a design signal that requires human judgment.

## Observed in code

- The canonical model already contains modules, typed relationships, source
  evidence, diagnostics, derived cycles, hierarchy, and layers.
- The source-index child now specifies files, named symbols, documentation,
  hash-linked spans, provenance, coverage, and versioned metric slots.
- No separate quality finding, quality configuration, metric-provider, baseline,
  or suppression contract exists in the current implementation.
- Existing analyzer confidence/provenance concepts can explain evidence, but
  they must not be reused as a subjective quality score without an explicit
  scale and formula version.

## User-confirmed target behavior

- Files/functions over configurable limits can be flagged and colored.
- Cyclomatic complexity, documentation coverage, coupling, cycles,
  dependency direction, and layer constraints should be checked
  deterministically where the required facts exist.
- SOLID-related feedback is useful, but static analysis must not claim that a
  subjective architectural principle has been mathematically proven.
- Findings must be evidence-backed, configurable, reproducible, exportable,
  and consumable by the viewer and future MCP surface.

## Specified decisions

- Quality configuration is a separately owned versioned `quality` profile. It
  is independent from analyzer options, project assignments, layout settings,
  source-index data, and live watcher settings.
- Metrics are versioned facts with namespaced IDs, typed values, formula IDs and
  formula versions. A rule compares an observed metric to an explicit operator
  and threshold; unknown/unsupported metrics do not become zero.
- Metric providers and quality rules are registered strategies. The engine
  orchestrates discovery, context, evaluation, finding normalization, and
  ordering; it does not grow a central switch for each language or rule.
- Exact findings include threshold breaches and graph predicates whose input
  facts are observed. Signals include static SOLID indicators and other
  heuristics; every signal is labeled `assessment_kind: signal` and carries
  its rule basis and limitations.
- Findings are separate from analyzer diagnostics and source facts. They point
  to typed subjects, metrics, spans, and relations and have an opaque
  snapshot-local ID plus a stable matching key.
- Baselines/suppressions match an exact finding key and rule version. They do
  not delete detection, and a new rule/formula version is not silently hidden
  by an old baseline.
- Unknown, unsupported, and partial coverage is reported separately from a
  passing rule. A rule can report `not_evaluable` coverage without creating a
  violation.
- Findings are evaluated per authoritative scope. A combined report retains
  scope provenance and only evaluates cross-scope rules when an explicit,
  validated aggregate policy supplies the necessary relation facts.

## Boundary

This capability owns metric providers, quality rules, thresholds, severity,
finding identity/revision, baselines, suppressions, coverage, and report
serialization. It does not own source parsing/indexing, folder watching,
snapshot freshness, MCP transport, source edits, runtime behavior, or
subjective architectural approval.

## Initial rule families

- `source:file.max-lines`
- `source:callable.max-lines`
- `source:callable.max-cyclomatic-complexity`
- `source:callable.max-nesting-depth`
- `source:public-symbol.documentation`
- `architecture:module.max-efferent-coupling`
- `architecture:module.max-afferent-coupling`
- `architecture:no-cycles`
- `architecture:forbidden-dependency`
- `architecture:layer-direction`
- `signal:solid.srp`, `signal:solid.ocp`, `signal:solid.lsp`,
  `signal:solid.isp`, `signal:solid.dip`

The catalog is open; each rule must declare its required capabilities, formula
versions, assessment kind, and evidence policy before it can be enabled.

## Implementation and verification focus

Start with the registry, quality profile validation, file/callable size rules,
documentation coverage, graph cycle/direction rules, and deterministic finding
serialization. Add language-specific complexity providers independently. Test
unknown/unsupported handling, threshold boundaries, baseline revision behavior,
scope isolation, rule ordering, and SOLID signal labeling.
