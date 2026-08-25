# Python analysis canonical API/CLI contract

## Analyzer options

```json
{
  "source_roots": ["path"],
  "python_version": "3.x|null",
  "include_stubs": false,
  "include_tests": false,
  "exclude": ["glob"]
}
```

Source-root precedence is explicit option > project configuration > `src/` if present > project root. `pyproject.toml` is preferred when multiple configuration files exist.

## Manifest and result

`id=org.archview.python`, `language=python`, markers=`pyproject.toml`, `setup.cfg`, `setup.py`, capabilities=`detect`, `static_dependencies`, `dynamic_diagnostics`.

The adapter emits common module observations and `depends_on` observations. Dynamic imports use `reference.scope=dynamic` and confidence basis `dynamic`; no project code is imported.

## CLI example

`arch-view analyze --project ./repo --language python --format analysis-json --output analysis.json`

All parent analysis status, error, evidence, and exit-code rules apply.
