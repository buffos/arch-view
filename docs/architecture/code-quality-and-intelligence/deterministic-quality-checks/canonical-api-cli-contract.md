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
    "profile_digest": { "algorithm": "sha256", "value": "opaque-profile-digest" },
    "options_digest": { "algorithm": "sha256", "value": "opaque-options-digest" },
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
generic metadata map. For a project-backed local viewer, profiles are separate
JSON documents discovered from the direct project-relative
`quality-profiles/*.json` directory. They are not stored inside
`.archview.json`, which remains layout/analyzer configuration. The file name is
presentation metadata; profile identity remains the versioned `profile_id` and
`profile_version` in the document and report.

## Managed baseline reference and document

A profile may point to one canonical baseline document:

```json
{
  "baseline": {
    "baseline_id": "baseline:main",
    "revision": "1.0.1"
  }
}
```

The baseline itself is a separate `arch-view.quality-baseline/v1` JSON file in
the direct project-relative `quality-baselines/` directory. The loader resolves
only the exact `baseline_id` and `revision` named by the selected profile. It
does not scan all JSON files and guess. A missing baseline is a warning and
leaves findings unsuppressed; an invalid or ambiguous match is an error.

Managed baseline updates keep all accepted entries in that one file. A new
file starts at revision `1.0.0`; later appends increment the numeric patch (for
example, `1.0.0` to `1.0.1`). A duplicate with the same exact finding/rule/
profile/formula identity and the same review metadata is idempotent. A duplicate
with a different reason or owner is a conflict. The profile version does not
change, but its digest changes when its baseline reference changes.

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

The local HTTP adapter and the live/MCP gateway expose equivalent catalog,
report, comparison, and explicitly
permissioned policy semantics. The adapter delegates to the quality services;
it does not re-evaluate rules or implement a second baseline matcher:

- `GET /v1/quality/profiles` — discover project-local profile documents and
  report invalid entries without making them selectable.
- `GET /v1/quality/baselines` — list direct project-local baseline documents by
  ID/revision/status and read one bounded page of entries when a single safe
  filename is selected. It never selects a baseline for a later evaluation.
- `GET /v1/quality/rules` — return the complete registered rule catalog. With
  `profile_id` and `profile_version`, each entry also reports whether the
  selected profile explicitly binds it and whether it is enabled. Catalog
  defaults are suitable for session-only viewer toggles and do not alter the
  profile document.
- `PUT /v1/quality/profiles/save` — validate and persist the selected
  profile's complete `rule_bindings` array back to its discovered JSON file.
- `PUT /v1/quality/profiles/save-as` — validate and create a new profile JSON
  file from a selected profile. The request supplies the source profile
  identity, new profile identity/version, direct `.json` filename, and the
  complete `rule_bindings` array; an existing destination is rejected.
- `POST /v1/quality/evaluate` — evaluate the already loaded model or selected
  cached analysis scope with one discovered profile; it does not invoke an
  analyzer. The request uses
  `arch-view.quality-evaluation-request/v1` and carries `profile_id`,
  `profile_version`, an optional scope, and an optional `rule_bindings` array.
  When supplied, that array is a temporary session selection; it is validated
  and evaluated without writing the project profile.
- `POST /v1/quality/validate` — validate a quality profile.
- `GET /v1/models/{model_id}/quality` — retrieve bounded report/coverage.
- `GET /v1/models/{model_id}/quality/findings` — filter and paginate findings.
- `GET /v1/models/{model_id}/quality/findings/{finding_id}/evidence` — retrieve
  bounded evidence/context.
- `GET /v1/models/{model_id}/quality/coverage` — filter and paginate coverage
  states independently from findings.
- `POST /v1/quality/baselines/create` — create a validated
  `arch-view.quality-baseline/v1` document from all active findings in the
  selected report or from explicit finding IDs/keys. The project-backed viewer
  supplies the profile identity, baseline identity, revision, reason, optional
  owner, and direct JSON filename. The request may attach the new baseline to
  the selected profile; the file is written under the separate
  `quality-baselines/` directory and an existing destination is rejected.
- `POST /v1/quality/baselines/append` — merge selected active findings from a
  current compatible report into the profile's canonical baseline, or create
  its first document. It rejects unsupported, not-evaluable, stale, partial,
  and non-active findings; accepts an expected revision for conflict
  detection; increments the baseline revision; updates the profile reference;
  and publishes the policy pair with atomic writes/rollback behavior. This is
  an explicitly permissioned policy operation.

Unknown rules/configuration return structured `422` diagnostics; a missing
report is represented explicitly, and partial provider coverage returns a valid
report with explicit status. Evidence never includes source text unless the
caller makes an explicit bounded source-context request; line and byte budgets
are validated at the boundary.

Consumers may filter the bounded findings query by
`rule_id=source:file.max-lines` and file subject, then present a summary such as
the number of files over the configured limit and an explicit affected-files
filter. That is a projection of the quality report; it must not recompute a
threshold from raw `FileRecord.size.line_count`, and partial/unknown/unsupported
coverage must remain visible.

## CLI/export behavior

Headless analysis may accept `--quality-profile`, `--quality-baseline`,
`--no-quality-baseline`,
`--quality-exit-on`, and repeatable `--quality-exit-status` flags and emits the
same `quality_report` JSON used by the viewer and live/MCP adapters. Exit
policy is a caller-owned projection over finding severity/status; the report
itself remains complete and does not hide baselined findings. Omitting the exit
policy is a no-op. HTML/SVG may color or annotate subjects from the report, but
quality findings do not mutate architecture semantics. No source edits are
performed.

With `--quality-profile`, the CLI automatically resolves the exact baseline
referenced by that profile from `quality-baselines/`. `--quality-baseline`
overrides that selection for one run. `--no-quality-baseline` disables
suppression for one run. A missing automatic baseline prints a warning and
evaluates without suppression; invalid or ambiguous baseline documents stop
the command.

The local CLI provides a managed baseline append workflow for automation:

```text
arch-view quality baseline add --project . \
  --profile quality-profiles/full.json \
  --baseline-file main.json \
  --input quality-report.json \
  --finding <finding-key> --reason "Accepted because ..."
```

The command creates `quality-baselines/main.json` when needed, derives
`baseline:main` when no ID is supplied, merges without duplicates, updates the
profile's baseline reference, and reports that a fresh evaluation is needed.
It does not edit source code. `--expected-revision` detects a concurrent
policy update, and `--revision` is an explicit override for a controlled
revision.

The legacy standalone baseline creation workflow remains available:

```text
arch-view quality baseline --input <analysis|model|quality-report.json> \
  --output <baseline.json|-> --baseline-id baseline:<name> \
  (--finding <finding-id-or-key> ... | --all-active) --reason <text>
```

The standalone command resolves report-local finding IDs or stable finding keys, copies
the exact rule/profile/formula identity required by the baseline contract,
validates the result, and writes a separate
`arch-view.quality-baseline/v1` document. `--all-active` selects every active
finding in the input report. Existing files are protected unless
`--overwrite` is supplied. The resulting file is consumed by a later analysis
run with `--quality-baseline`; baseline creation never changes the source,
profile, or original report. The managed `quality baseline add` command is the
workflow for appending to, and automatically attaching, the canonical profile
baseline.

## Determinism and safety

- Profile/options/baseline digests, rule/provider versions, source/model
snapshot IDs, and formula versions are part of the evaluation fingerprint.
- Findings and metrics are canonically ordered and digested; operational times
  are excluded from semantic equality.
- Unknown/unsupported/not-evaluable coverage is never converted to pass or
  failure.
- Subject and evidence refs are scope-safe; source context remains an explicit
  bounded read request.
- The common error shape remains `{ "error": { "code", "message", "details" } }`.
