# Quality profile JSON

A quality profile is a separate JSON file. Its schema version is **arch-view.quality/v1**.

## Top-level fields

| Field | Meaning |
| --- | --- |
| schema_version | The profile format version. |
| profile_id | The profile's stable name, such as profile:default. |
| profile_version | The version of this profile's rules and settings. |
| enabled_rules | The rules that are available to the profile, with true or false enabled state. |
| severity_policy | Default severity behavior. |
| constraints | Explicit forbidden-dependency and layer-direction policies. |
| baseline | Optional reference to a baseline file. |
| extensions | Extra, namespaced data for compatible tools. |

## A clear profile

~~~json
{
  "schema_version": "arch-view.quality/v1",
  "profile_id": "profile:team",
  "profile_version": "1.0.0",
  "enabled_rules": [],
  "severity_policy": {
    "namespace": "severity:default",
    "schema_version": "1.0.0",
    "payload": {}
  },
  "constraints": [],
  "extensions": []
}
~~~

## Rule binding fields

Each entry in enabled_rules can contain:

| Field | Meaning |
| --- | --- |
| rule_id | The rule to run or keep available. |
| rule_version | The exact version of that rule. |
| enabled | true runs the rule; false keeps it turned off. |
| parameters | Typed values for this rule. |
| severity | Optional info, warning, error, or blocker value. |

The rule reference explains what each parameter means. Do not copy a parameter block from a different rule without checking its namespace.
