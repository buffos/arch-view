# Quality report reference

Read this file before interpreting findings or creating a baseline.

## Where the report lives

The CLI `analyze --format analysis-json` writes a model JSON document. Its
quality report is at the top-level property:

```text
quality_report
```

The model may also contain `source_index`. Use it to map evidence file IDs to
real repository paths.

MCP wraps results in an `arch-view.query/v1` envelope. Inspect the envelope
before the nested result. Important fields are:

| Field | Meaning |
| --- | --- |
| `revision` | Immutable live-analysis revision used by the response. |
| `freshness` | Whether the response is current, stale, updating, or failed. |
| `scope_ids` | Analyzer scopes covered by the response. |
| `coverage` or `capabilities` | What the analyzers actually supplied. |
| `omitted_fields` | Data deliberately left out because of projection or budget. |
| `budget` | Item and byte limits, including truncation. |
| `diagnostics` | Query or analysis problems. |

Do not call a small result clean when coverage is partial, unsupported,
unknown, not evaluable, omitted, or truncated.

## Quality report fields

The report contains these useful fields:

| Field | Use |
| --- | --- |
| `profile_id`, `profile_version` | Confirm which profile produced the report. |
| `source_snapshot_ids` | Identify the source snapshots used. |
| `coverage` | Decide whether each rule had enough input. |
| `findings` | Read the actual reported observations. |
| `metrics` | Inspect the measured facts behind findings. |
| `diagnostics` | Explain evaluation problems. |
| `report_digest` and `evaluation_fingerprint` | Technical identity for comparisons and baselines. |

For each finding, use:

| Field | Meaning |
| --- | --- |
| `id` | Report-local finding ID. The CLI baseline command accepts it. |
| `finding_key` | Stable exact identity. MCP baselines use this. |
| `rule_id`, `rule_version` | The check and its version. |
| `assessment_kind` | `exact` or `signal`. |
| `status` | Usually `active`, `suppressed`, `baseline`, `resolved`, or `not_evaluable`. |
| `severity` | `info`, `warning`, `error`, or `blocker`. It is not proof of a defect. |
| `subject_ref` | The module, file, symbol, or relation being reported. |
| `message` | Human-readable description of the observation. |
| `comparison` | The measured value, operator, limit, and unit for exact checks. |
| `evidence` | Source spans, entities, relations, metrics, and diagnostics supporting the finding. |
| `limitations` | Reasons the observation cannot prove more than it says. |
| `provenance` | How the fact was produced. |

## Coverage states

Treat these as different states:

- `observed`: the analyzer supplied the required fact and the rule could use it.
- `absent`: the fact was checked and not found. For documentation, this can mean no documentation record was attached.
- `unknown`: the analyzer cannot establish whether the fact exists.
- `unsupported`: the analyzer does not provide the required capability. This is not a pass and is not a finding to baseline.
- `partial`: only part of the eligible input was analyzed.
- `not_evaluable`: facts exist, but they cannot produce a valid check result. This is not a pass and is not a finding to baseline.

An analyzer or scope can also be `partial`, `failed`, or `degraded`. Report the
scope instead of presenting only the findings that happened to be returned.

## Exact checks and SOLID signals

An `exact` finding says the reported metric satisfied the configured rule. For
example, `14 greater_than 10` is a real result under that profile. Review the
subject and evidence before deciding whether the result needs a code change.

A `signal` finding is a structural review prompt. The five SOLID rules are
signals. They do not prove SRP, OCP, LSP, ISP, or DIP violations. A concrete
dependency count, a large interface, or a deep hierarchy is an observed shape,
not design intent. Inspect the source and decide whether the design is wrong.

Do not baseline a signal merely because the wording is inconvenient. Baseline
it only after the agent understands the code and records why the signal is
accepted for now.

## Reading evidence

An evidence source span has a `file_id` and start/end positions. In a CLI model,
find the matching file in:

```text
source_index.snapshots[].files[].id
```

Then use that file's `path`. Read only the reported line range plus a small
context. A span with valid line numbers is line-level evidence. If no span is
reported, say that the result has file/entity/metric provenance only. Do not
turn a file name into a claim about a precise line.

For MCP, request `get_finding_evidence` for one finding at a time. Set
`include_source_context: true` only for a finding under review. Keep
`max_lines` and `max_context_bytes` bounded. Source context is read-only and
must belong to the same revision as the finding.

## Baselines

A baseline records an exact reviewed finding. It does not fix source code and
it does not prove that the finding is wrong. Baseline entries include the
finding key, rule and profile versions, formula versions, and a reason.

Before baseline creation:

1. Confirm the finding is still `active` in the newest current report.
2. Read its evidence and source context.
3. Decide that it is intentionally accepted, out of scope, or a false positive.
4. Do not baseline `unsupported`, `not_evaluable`, `partial`, stale, or failed results.
5. Use a specific reason that names the code or decision.

CLI managed workflow:

```text
arch-view quality baseline add --project . --profile quality-profiles/full.json --baseline-file main.json --input quality-report.json --finding <finding-id-or-key> --reason "Accepted because <specific reason>."
```

The managed command creates `quality-baselines/main.json` when needed,
appends without duplicates, updates the profile's baseline reference, and
bumps the baseline revision. If `--baseline-id` is omitted, the filename gives
the first identity (`main.json` becomes `baseline:main`). Repeat `--finding`
for separately reviewed findings. Do not use `--all-active` when the review
must distinguish real problems from accepted exceptions.

Run the same analysis again without `--quality-baseline`; the profile reference
is loaded automatically. Use `--no-quality-baseline` for a clean comparison.
Use `--quality-baseline <path>` only for an explicit per-run override.

The old standalone CLI command and MCP `preview_baseline`/`create_baseline`
pair remain available for migration. MCP's managed command is
`append_baseline`. It uses the current report revision, merges into the
canonical baseline, updates the profile reference, and returns that a new
evaluation is required. Policy writes are off by default and need explicit
server authorization.
