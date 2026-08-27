# Rust analysis canonical API/CLI contract

## Analyzer options

```json
{
  "crate": "path-or-name|null",
  "features": ["string"],
  "target": "string|null",
  "include_tests": false,
  "include_examples": false,
  "exclude": ["glob"]
}
```

`crate` is required when the selected workspace has multiple candidate crates. Features/target are metadata inputs; unresolved cfg conditions are diagnosed rather than executed.

## Manifest and result

`id=org.archview.rust`, `language=rust`, markers=`Cargo.toml`, capabilities=`detect`, `static_dependencies`, `cfg_metadata`.

The adapter emits common `depends_on` observations with `metadata.kind`=`use|pub_use|dependency`. `mod` declarations remain structural observations in the module hierarchy and module metadata with source evidence; they are not semantic `depends_on` edges.

## CLI example

`arch-view analyze --project ./workspace --language rust --crate crates/api --format analysis-json --output analysis.json`

Parent analysis status, diagnostics, evidence, safety, and exit-code rules apply.
