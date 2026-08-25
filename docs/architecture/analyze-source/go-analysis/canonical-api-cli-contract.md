# Go analysis canonical API/CLI contract

## Analyzer options

```json
{
  "module": "string|null",
  "build_tags": ["string"],
  "include_tests": false,
  "include_generated": false,
  "exclude": ["glob"]
}
```

`module` is required only when a `go.work` context contains multiple candidate modules. `include_external` does not create external module nodes; it controls whether non-local references are retained in detail.

## Manifest

`id=org.archview.go`, `language=go`, markers=`go.mod`, capabilities=`detect`, `static_dependencies`, `build_constraints`, option schema above.

## Result mapping

The adapter emits common modules, `depends_on` observations, references, source locations, tags (`test`, `generated`, `entrypoint?`), and diagnostics. Local module identity uses `go:<module-path>/<relative-import-path>`.

## CLI examples

`arch-view analyze --project ./repo --language go --format analysis-json --output analysis.json`

`arch-view analyze --project ./workspace --language go --module ./service --build-tag integration --format analysis-json --output analysis.json`

All common parent analysis HTTP/CLI status and exit-code rules apply.
