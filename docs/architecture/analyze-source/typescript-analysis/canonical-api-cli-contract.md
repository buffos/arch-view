# TypeScript analysis canonical API/CLI contract

## Analyzer options

```json
{
  "config": "path|null",
  "include_js": false,
  "include_tests": false,
  "runtime": "auto | esm | cjs",
  "exclude": ["glob"]
}
```

`config` is required when multiple `tsconfig` candidates remain after root detection. `extends`, `baseUrl`, `paths`, rootDirs, package exports, and include/exclude are resolved statically.

## Manifest and result

`id=org.archview.typescript`, `language=typescript`, markers=`tsconfig.json`, `package.json`, capabilities=`detect`, `static_dependencies`, `aliases`, `exports`.

The adapter emits common `depends_on` observations with `metadata.kind` and alias/runtime provenance. Literal dynamic imports may be local; computed imports use `dynamic` references.

## CLI example

`arch-view analyze --project ./repo --language typescript --config ./tsconfig.app.json --format analysis-json --output analysis.json`

Parent analysis status, diagnostics, evidence, safety, and exit-code rules apply.
