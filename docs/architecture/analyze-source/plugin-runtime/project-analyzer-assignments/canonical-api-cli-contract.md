# Project analyzer assignments and view selection canonical API/CLI contract

## Configuration schema

Existing layout-only files remain valid:

```json
{
  "schema_version": "arch-view.config/v1",
  "layout": { "algorithm": "layered", "options": {} }
}
```

Analysis-enabled files use `arch-view.config/v2`:

```json
{
  "schema_version": "arch-view.config/v2",
  "layout": { "algorithm": "layered", "options": {} },
  "analysis": {
    "assignments": [
      {
        "path": ".",
        "analyzer_id": "org.archview.go",
        "options": { "include_tests": false }
      },
      {
        "path": "frontend",
        "analyzer_id": "org.archview.typescript",
        "options": {}
      }
    ]
  }
}
```

`assignments` is an array so duplicate paths can be rejected explicitly. Paths
are repository-relative POSIX paths; `.` means the repository root. Absolute
paths, `..`, empty segments, and glob characters are invalid. Assignment order
is ignored and canonical serialization sorts by path. Unknown v2 fields are
errors. `options` is optional and may contain only non-sensitive manifest
options.

## Precedence

For a selected root:

1. CLI `--analyzer` or `--language` selection.
2. Deepest assignment matching the root.
3. Automatic detection.

`--analyzer` and `--language` must be mutually compatible. An explicitly
unavailable assignment produces a scoped error and is not silently replaced by
automatic detection. Combined mode plans all valid assignments plus unassigned
automatic roots; nested assignments override ancestor assignments.

## Scope response

`GET /v1/analyses/{run_id}/scopes` returns:

```json
{
  "schema_version": "arch-view.scopes/v1",
  "active_scope": "all",
  "scopes": [
    {
      "scope_id": "scope-<sha256>",
      "label": "frontend · TypeScript",
      "project_root": "frontend",
      "analyzer_id": "org.archview.typescript",
      "status": "complete",
      "summary": { "module_count": 4, "relationship_count": 3, "diagnostic_count": 0 }
    }
  ]
}
```

The list is sorted by scope ID. Failed/unavailable scopes remain listed with
diagnostics. `GET /v1/analyses/{run_id}/projection?scope=all|<scope_id>` returns
the renderer-neutral projection; omitting `scope` means `all`. An unknown scope
returns `404` with `analysis_scope_not_found`.

## Reanalysis and cache behavior

`POST /v1/reanalysis` reloads the nearest configuration and invalidates only
affected job cache keys when the repository and discovery policy permit it.
Changing only the active scope never calls reanalysis. The response reports
the new aggregate run, scope statuses, cache hits, and invalidation reasons.

## Errors

| Code | Meaning |
|---|---|
| `analysis_config_invalid` | Nearest configuration is malformed, unsafe, or schema-invalid. |
| `analysis_assignment_invalid` | One assignment has an invalid path or option shape. |
| `analysis_analyzer_unavailable` | An assigned logical analyzer is not available. |
| `analysis_selection_conflict` | CLI analyzer/language selections conflict. |
| `analysis_scope_not_found` | Requested scope is not in the aggregate run. |
| `analysis_scope_stale` | Requested cached scope no longer matches current inputs. |

## CLI mapping

```text
arch-view analyze --project <repository> [--language <id>] [--analyzer <id>]
arch-view open --project <repository>
```

The default project run is combined when multiple scopes are applicable.
`--analyzer`/`--language` constrain the selected root according to the
precedence rules. The CLI can request a scope for export using
`--scope <scope-id>`; scope selection itself never triggers a new analyzer.

## Layout separation and parity

`layout` and `analysis` are independent schemas. Layout settings cannot alter
assignment resolution, analyzer options, identities, or canonical model facts.
All clients must preserve scope IDs, per-scope status, diagnostics, and
combined/individual semantic parity.
