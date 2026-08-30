# Quality profiles

A quality profile is a separate JSON file. It is a list of rules and their settings.

The profile answers:

> “Which checks should Arch View run, and what limits should they use?”

## The simple shape

~~~json
{
  "schema_version": "arch-view.quality/v1",
  "profile_id": "profile:team",
  "profile_version": "1.0.0",
  "enabled_rules": [
    {
      "rule_id": "source:file.max-lines",
      "rule_version": "1.0.0",
      "enabled": true,
      "parameters": {
        "namespace": "rule-config:source-file-size",
        "schema_version": "1.0.0",
        "payload": {
          "operator": "greater_than",
          "limit": 500,
          "unit": "unit:line"
        }
      }
    }
  ]
}
~~~

The IDs identify a stable rule for the program. You normally choose rules in the viewer or copy them from an example profile.

## Every rule should be visible

The profile stores an entry for every available rule that you want to manage. Use **enabled: true** to run it and **enabled: false** to keep it available without running it.

This makes the profile easy for a person to understand:

~~~text
File size                 enabled: true
Public documentation      enabled: true
Dependency inversion      enabled: false
~~~

The profile can still be small for automation. If an entry is missing, the current application treats it as not selected; the viewer shows the complete catalog so you can see what is available.

## Rule parameters

Most threshold rules use:

| Parameter | Meaning |
| --- | --- |
| operator | How the observed number is compared with the limit. |
| limit | The number at which the rule reacts. |
| unit | What the number counts, such as lines or modules. |

For example, **greater_than** with a limit of 10 reacts to 11, but not to 10.

## Severity

A rule can use a severity such as info, warning, error, or blocker. Severity describes how strongly the result should affect a person or an automated exit policy.

Severity does not change the measurement.

## Architecture policies

Forbidden dependencies and layer direction need explicit policy. Arch View does not invent your architecture boundaries.

Example forbidden dependency:

~~~json
{
  "kind": "forbidden_dependency",
  "parameters": {
    "namespace": "constraint:forbidden",
    "schema_version": "1.0.0",
    "payload": {
      "from_module_patterns": ["web/*"],
      "to_module_ids": ["module:database"]
    }
  }
}
~~~

This means: “Report a dependency when a module matching web/* depends on the database module.”

## Save and run

When you change a profile in the viewer, save it, then run the checks again. A saved profile is configuration. It is not a report.

If a profile cannot be saved, read the validation message. Common causes are an unknown rule, a wrong version, an invalid parameter type, or an invalid architecture constraint.
