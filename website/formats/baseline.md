# Baseline JSON

A baseline is a separate JSON file. Its schema version is **arch-view.quality-baseline/v1**.

## Fields

| Field | Meaning |
| --- | --- |
| schema_version | The baseline format version. |
| baseline_id | The name of the baseline. |
| revision | Optional human-readable revision. |
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

The baseline is exact-version aware. If the code, rule, or calculation changes, review the entry again.
