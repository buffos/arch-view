# Quality profile reference

Read this file before selecting, copying, or changing a quality profile.

## What a profile is

A quality profile is a separate JSON document. It tells Arch View which rules
to run and how to configure them. It is not the quality report and it is not
the source code.

The current quality schema is `arch-view.quality/v1`. The repository's full
profile is normally:

```text
profile_id: profile:full
profile_version: 1.0.0
```

Always confirm the exact identity in the discovered profile or MCP catalog.
Do not silently substitute another profile.

## Top-level shape

Keep these fields when copying a profile:

```json
{
  "schema_version": "arch-view.quality/v1",
  "profile_id": "profile:full",
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
```

`baseline` is optional. It points to a saved baseline and must not be added to
a first, unbaselined review unless the user asks for that. Make a separate
temporary copy. Never edit the saved profile just to test a different limit.

When a saved profile already contains a baseline reference, the normal
analysis path loads the exact matching baseline automatically. The managed
workflow appends reviewed findings to that canonical baseline and updates the
reference; it does not require a baseline path on every later run.

The reference has only the baseline identity and revision. Do not copy finding
payloads into the profile:

```json
"baseline": {
  "baseline_id": "baseline:main",
  "revision": "1.0.0"
}
```

Each item in `enabled_rules` is a complete binding:

```json
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
  },
  "severity": "warning"
}
```

Do not replace `enabled_rules` with a short list containing only the rule you
are interested in. Missing bindings mean that those rules are not selected.
Keep every existing binding, including disabled bindings, unless the request
explicitly changes them.

## Current full-profile catalog

The current `profile:full@1.0.0` contains 15 enabled bindings:

| Rule ID | Assessment | Parameters in the full profile | Severity |
| --- | --- | --- | --- |
| `architecture:forbidden-dependency` | exact | `rule-config:architecture-constraint`, `{}` | error |
| `architecture:layer-direction` | exact | `rule-config:architecture-constraint`, `{}` | error |
| `architecture:module.max-afferent-coupling` | exact | `rule-config:module-coupling`, `external_policy=exclude`, `limit=10`, `operator=greater_than`, `unit=unit:module` | warning |
| `architecture:module.max-efferent-coupling` | exact | `rule-config:module-coupling`, `external_policy=exclude`, `limit=10`, `operator=greater_than`, `unit=unit:module` | warning |
| `architecture:no-cycles` | exact | `rule-config:no-cycles`, `{}` | error |
| `signal:solid.dip` | signal | `rule-config:solid-signal`, `{}` | info |
| `signal:solid.isp` | signal | `rule-config:solid-signal`, `{}` | info |
| `signal:solid.lsp` | signal | `rule-config:solid-signal`, `{}` | info |
| `signal:solid.ocp` | signal | `rule-config:solid-signal`, `{}` | info |
| `signal:solid.srp` | signal | `rule-config:solid-signal`, `{}` | info |
| `source:callable.max-cyclomatic-complexity` | exact | `rule-config:source-callable-complexity`, `limit=10`, `operator=greater_than`, `unit=unit:complexity` | warning |
| `source:callable.max-lines` | exact | `rule-config:source-callable-size`, `limit=50`, `operator=greater_than`, `unit=unit:line` | warning |
| `source:callable.max-nesting-depth` | exact | `rule-config:source-callable-nesting`, `limit=4`, `operator=greater_than`, `unit=unit:depth` | warning |
| `source:file.max-lines` | exact | `rule-config:source-file-size`, `limit=500`, `operator=greater_than`, `unit=unit:line` | warning |
| `source:public-symbol.documentation` | exact | `rule-config:documentation-coverage`, `{}` | warning |

The catalog is authoritative when it reports a different rule version or
parameter schema. Do not invent a rule or a parameter. If the requested rule
does not exist in the catalog and cannot be found in a profile, report that
fact instead of creating a guessed binding.

`constraints` holds explicit architecture policies. The forbidden-dependency
and layer-direction rules do not invent policies from names or graph layout.
An empty `constraints` array can therefore leave those rules with no policy to
evaluate. Preserve existing constraints. Do not add one unless the user gives
the policy.

## Temporary 600-line profile

For a review that asks for a 600-line threshold:

1. Find the complete `profile:full` document and verify its version.
2. Make a deep copy in a temporary file. Leave the saved profile unchanged.
3. Find the binding whose `rule_id` is exactly `source:file.max-lines`.
4. Set that binding's `parameters.payload.limit` to `600`.
5. Set its `enabled` field to `true` for this evaluation.
6. Keep its `rule_id`, `rule_version`, `parameters.namespace`, `parameters.schema_version`, `operator`, `unit`, and `severity` unchanged.
7. Keep every other rule binding, all five SOLID bindings, `severity_policy`, `constraints`, and `extensions` unchanged.
8. Do not add a baseline reference to the temporary profile.

The resulting nested value must be:

```json
"payload": {
  "operator": "greater_than",
  "limit": 600,
  "unit": "unit:line"
}
```

If the full profile does not contain `source:file.max-lines` or one of the
five SOLID rules, use `get_quality_rules` in MCP mode to confirm the catalog
binding before adding it. In CLI-only mode, use the catalog table above only
when the application is using rule version `1.0.0`. Add a complete binding,
not only a rule ID. If the version or parameter schema differs, stop and
report the mismatch.

In MCP, `evaluate_quality.rule_bindings` replaces the profile's complete
`enabled_rules` list. It does not merge one changed binding into the profile.
If temporary bindings are sent, send the full preserved list.
