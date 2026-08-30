# Deterministic quality checks canonical API/CLI contract

## Version and optional attachment

The quality contract is independently versioned as
`arch-view.quality/v1`. It is an optional top-level `quality_report` sibling of
the source index, architecture modules, relationships, diagnostics, and derived
data. It does not revise the analyzer protocol.

```json
{
  "source_index": { "schema_version": "arch-view.source-index/v1" },
  "quality_report": {
    "schema_version": "arch-view.quality/v1",
    "evaluation_id": "opaque-evaluation-id",
    "source_snapshot_ids": ["opaque-source-snapshot-id"],
    "profile_id": "profile:default",
    "profile_version": "1.0.0",
    "coverage": [],
    "metrics": [],
    "findings": [],
    "diagnostics": [],
    "extensions": []
  }
}
```

Omission means no quality report was produced; it does not mean the repository
passed all rules. Older clients may ignore the field.

## Quality profile

```json
{
  "schema_version": "arch-view.quality/v1",
  "profile_id": "profile:default",
  "profile_version": "1.0.0",
  "enabled_rules": [
    {
      "rule_id": "source:file.max-lines",
      "rule_version": "1.0.0",
      "enabled": true,
      "parameters": {
        "namespace": "rule-config:source-file-size",
        "schema_version": "1.0.0",
        "payload": { "operator": "greater_than", "limit": 500, "unit": "unit:line" }
      },
      "severity": "warning"
    }
  ],
  "severity_policy": { "namespace": "severity:default", "schema_version": "1.0.0", "payload": {} },
  "constraints": [],
  "extensions": []
}
```

Rule configuration is typed by the rule namespace/schema, not an unversioned
generic metadata map. The host may store this profile in a separate quality
configuration section, but it must remain independent from layout and analyzer
assignment options.

## Exact finding example

```json
{
  "id": "opaque-finding-id",
  "finding_key": "opaque-stable-finding-key",
  "rule_id": "source:file.max-lines",
  "rule_version": "1.0.0",
  "assessment_kind": "exact",
  "status": "active",
  "severity": "warning",
  "subject_ref": { "kind": "file", "id": "opaque-file-id", "snapshot_id": "opaque-source-snapshot-id" },
  "message_code": "quality:file-lines-exceeded",
  "message": "File has 642 lines; configured maximum is 500",
  "observed_metric_ids": ["opaque-metric-id"],
  "comparison": {
    "operator": "greater_than",
    "observed_metric_id": "opaque-metric-id",
    "limit": { "kind": "integer", "value": 500 },
    "unit": "unit:line"
  },
  "evidence": {
    "source_spans": [],
    "entity_refs": [],
    "relation_refs": [],
    "metric_refs": ["opaque-metric-id"],
    "diagnostic_refs": []
  },
  "provenance": {
    "status": "observed",
    "basis": "syntax",
    "evidence_ids": [],
    "provider": "rule:source-file-size",
    "provider_version": "1.0.0"
  },
  "extensions": []
}
```

## SOLID signal example

```json
{
  "id": "opaque-signal-id",
  "finding_key": "opaque-solid-signal-key",
  "rule_id": "signal:solid.srp",
  "rule_version": "1.0.0",
  "assessment_kind": "signal",
  "status": "active",
  "severity": "info",
  "subject_ref": { "kind": "symbol", "id": "opaque-type-id", "snapshot_id": "opaque-source-snapshot-id" },
  "message_code": "quality:solid-structural-signal",
  "message": "Structural signal: type combines a high member count with multiple dependency clusters",
  "limitations": [
    "Static structure cannot prove responsibility boundaries or an SRP violation."
  ],
  "evidence": {
    "source_spans": [],
    "entity_refs": [],
    "relation_refs": [],
    "metric_refs": [],
    "diagnostic_refs": []
  },
  "provenance": {
    "status": "observed",
    "basis": "heuristic",
    "evidence_ids": [],
    "provider": "rule:solid-srp-signal",
    "provider_version": "1.0.0"
  },
  "extensions": []
}
```

No signal may use exact-violation wording or be promoted to an exact finding
without a separately specified, observable rule.

## Catalog and report queries

Future transport adapters expose equivalent read semantics:

- `GET /v1/quality/rules` — registered rules, versions, capabilities, and
  assessment kind.
- `POST /v1/quality/validate` — validate a quality profile.
- `POST /v1/quality/evaluate` — evaluate a selected immutable source/model
  snapshot and profile.
- `GET /v1/models/{model_id}/quality` — retrieve bounded report/coverage.
- `GET /v1/models/{model_id}/quality/findings` — filter and paginate findings.
- `GET /v1/models/{model_id}/quality/findings/{finding_id}/evidence` — retrieve
  bounded evidence/context.

These routes are a transport mapping, not an implementation requirement of this
planning pass. Unknown rules/configuration return structured `422` diagnostics;
partial provider coverage returns a valid report with explicit status.

Consumers may filter the bounded findings query by
`rule_id=source:file.max-lines` and file subject, then present a summary such as
the number of files over the configured limit and an explicit affected-files
filter. That is a projection of the quality report; it must not recompute a
threshold from raw `FileRecord.size.line_count`, and partial/unknown/unsupported
coverage must remain visible.

## CLI/export behavior

Headless analysis may accept a versioned quality profile and emits the same
`quality_report` JSON used by viewer and future MCP adapters. Exit policy is a
caller-owned projection over finding severity/status; the report itself remains
complete and does not hide baselined findings. HTML/SVG may color or annotate
subjects from the report, but quality findings do not mutate architecture
semantics. No source edits are performed.

## Determinism and safety

- Profile, rule/provider versions, source/model snapshot IDs, baseline, formula
  versions, and options are part of the evaluation fingerprint.
- Findings and metrics are canonically ordered and digested; operational times
  are excluded from semantic equality.
- Unknown/unsupported/not-evaluable coverage is never converted to pass or
  failure.
- Subject and evidence refs are scope-safe; source context remains an explicit
  bounded read request.
- The common error shape remains `{ "error": { "code", "message", "details" } }`.
