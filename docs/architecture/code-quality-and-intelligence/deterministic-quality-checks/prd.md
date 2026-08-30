# Deterministic quality checks PRD

## Purpose

Give Arch View reproducible quality feedback from measurable source and
architecture facts. Developers and CI should be able to configure limits,
inspect evidence, and distinguish a deterministic rule result from a design
signal requiring human review.

## Actors

- **Developer:** wants oversized files/functions, complexity, missing public
  documentation, and structural architecture problems highlighted.
- **Maintainer/architect:** supplies intended layer and dependency constraints
  and reviews advisory design signals.
- **CI operator:** runs a fixed profile and needs stable machine-readable
  findings, coverage, severity, and exit-policy inputs.
- **Rule/metric author:** adds a language or rule strategy without editing the
  quality engine's central dispatch.
- **Viewer/CLI/export client:** decorates graph/source subjects and exports the
  same report.
- **Future MCP client:** requests current findings with compact evidence and
  asks for source context separately.

## Goals

1. Evaluate configurable size and complexity thresholds deterministically.
2. Measure documentation coverage without claiming comment quality.
3. Evaluate coupling, cycles, forbidden dependencies, and layer-direction
   constraints when the graph and explicit policy support them.
4. Emit stable, evidence-backed findings separate from analyzer diagnostics.
5. Preserve explicit not-evaluable/unsupported/unknown coverage.
6. Provide versioned rule/metric/config identity and reproducible reports.
7. Support baselines and suppressions without hiding changed rule meanings.
8. Add SOLID-related structural signals with honest labels and limitations.
9. Let new rules, languages, metrics, and policy extensions register
   additively.
10. Make file line-threshold findings usable in human-facing projections by
    summarizing affected files and offering an explicit filter without
    duplicating rule evaluation.

## Non-goals

- Proving subjective architecture quality or any SOLID violation.
- Replacing language parsers, source-index extractors, or canonical graph
  normalization.
- Runtime performance, production telemetry, dynamic dispatch certainty, or
  unpinned VCS-history claims.
- Automatic source edits, refactoring, or LLM-generated remediation.
- Inferring intended layer/dependency policy from directory or type names.
- Treating unsupported/unknown analysis as a clean result.

## Initial rule catalog

| Rule family | Assessment | Required input | V1 policy |
|---|---|---|---|
| `source:file.max-lines` | exact | file size | Compare `size.line_count` to configured limit. |
| `source:callable.max-lines` | exact | callable body span | Compare provider/formula-versioned body line count. |
| `source:callable.max-cyclomatic-complexity` | exact | callable control-flow metric | Use provider-declared decision vocabulary and formula version. |
| `source:callable.max-nesting-depth` | exact | callable nesting metric | Use provider-defined syntax nodes and formula version. |
| `source:public-symbol.documentation` | exact coverage check | visibility/documentation | Evaluate presence only; do not judge prose usefulness. |
| `architecture:module.max-efferent-coupling` | exact | canonical relationships | Count distinct configured target modules. |
| `architecture:module.max-afferent-coupling` | exact | canonical relationships | Count distinct configured source modules. |
| `architecture:no-cycles` | exact | canonical cycle projection | Report each canonical cycle once per policy. |
| `architecture:forbidden-dependency` | exact | relation + explicit deny policy | Report matching reported edges only. |
| `architecture:layer-direction` | exact | relation + explicit layer policy | Report edges that violate configured direction. |
| `signal:solid.srp/ocp/lsp/isp/dip` | signal | selected structural facts | Emit advisory evidence, limitations, and no proof wording. |

## Functional requirements

| ID | Requirement |
|---|---|
| DQC-FR-001 | Accept a separately versioned quality profile with open rule IDs and typed rule configuration. |
| DQC-FR-002 | Validate rule parameters, operators, units, capability requirements, profile identity, and constraint references before evaluation. |
| DQC-FR-003 | Calculate or consume formula-versioned metrics and preserve metric provenance. |
| DQC-FR-004 | Emit exact findings only when required facts are observed and the configured predicate is true. |
| DQC-FR-005 | Emit explicit coverage/not-evaluable results when required facts are unknown, unsupported, partial, or invalid. |
| DQC-FR-006 | Keep finding IDs opaque and report-local while providing stable finding keys for baseline/revision matching. |
| DQC-FR-007 | Preserve subject refs, source spans, metrics, graph refs, provider/version, rule/version, and evaluation fingerprint as evidence. |
| DQC-FR-008 | Apply explicit severity mapping and keep severity independent from exact/signal classification. |
| DQC-FR-009 | Apply baseline/suppression matching only for exact rule/profile/formula versions and retain suppressed findings in the report. |
| DQC-FR-010 | Keep SOLID outputs labeled as `signal` with deterministic indicators and limitations, never as proven violations. |
| DQC-FR-011 | Evaluate authoritative scopes independently and preserve scope provenance in combined reports. |
| DQC-FR-012 | Canonically order and digest reports so equal inputs/configuration produce equal semantic output. |
| DQC-FR-013 | Allow registered rule and metric strategies to add capabilities without modifying the core evaluator. |
| DQC-FR-014 | Expose scope-safe `source:file.max-lines` findings to human-facing consumers as a count of affected files and an explicit filter; preserve partial/unknown/unsupported coverage and do not re-evaluate raw source facts in the consumer. |

## Non-functional requirements

- Deterministic for equal source/model snapshots, provider/rule versions,
  profile, baseline, and options.
- Read-only and safe; no target application execution or source mutation.
- Explainable through compact evidence and explicit coverage.
- Backward-compatible optional report attachment.
- Bounded report size and stable pagination for future consumers.

## Success criteria

The scenarios in [acceptance-scenarios.md](acceptance-scenarios.md) pass for
threshold boundaries, metric formulas, docs coverage, graph rules, scope
isolation, baseline/revision behavior, open/closed registries, and SOLID signal
labeling. The application synthesis describes the quality report as a future
consumer of the specified source-index contract.
