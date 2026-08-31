# Baseline JSON

A baseline is a separate JSON file. Its schema version is
**arch-view.quality-baseline/v1**. A managed project normally keeps one
canonical file for each quality profile in `quality-baselines/`. The profile
stores the baseline ID and revision, so Arch View can discover that exact file
without loading unrelated JSON files.

## Fields

| Field | Meaning |
| --- | --- |
| schema_version | The baseline format version. |
| baseline_id | The name of the baseline. |
| revision | Revision used for exact matching. A new managed baseline starts at `1.0.0`; later appends increment the numeric suffix. |
| entries | Findings that were reviewed and accepted for now. |
| extensions | Extra namespaced data for compatible tools. |

Each entry records the finding key, rule and profile versions, reason, and optional owner.

## Example

~~~json
{
  "schema_version": "arch-view.quality-baseline/v1",
  "baseline_id": "baseline:main",
  "revision": "2026-08",
  "entries": [
    {
      "finding_key": "finding-key-from-report",
      "rule_id": "source:file.max-lines",
      "rule_version": "1.0.0",
      "profile_id": "profile:team",
      "profile_version": "1.0.0",
      "formula_versions": [],
      "reason": "Known generated file.",
      "owner": "platform-team"
    }
  ],
  "extensions": []
}
~~~

The baseline is exact-version aware. If the code, rule, or calculation changes,
review the entry again. Only active findings with observed coverage may be
added. Unsupported, not-evaluable, partial, stale, and failed results are not
valid baseline decisions.
