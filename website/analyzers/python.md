# Python analyzer

The Python analyzer reads project layout and imports without importing or running the target project.

## Project markers

- pyproject.toml
- setup.cfg
- setup.py

## Options

| Option | What it does | Default |
| --- | --- | --- |
| source roots | Selects repository-relative Python source roots. Repeatable. | automatic |
| Python version | Tells static interpretation which major/minor version to use. | automatic |
| include stubs | Includes .pyi stub files. | false |
| include tests | Includes test files and test directories. | false |
| exclude | Adds repository-relative exclusion globs. Repeatable. | none |

Example:

~~~powershell
go run ./cmd/arch-view analyze --project . --language python --python-version 3.12 --include-stubs --output analysis.json
~~~

## Dynamic imports

Some Python imports are chosen at runtime. The analyzer may report an unresolved or ambiguous reference. That is a limit of static information, not proof that the import fails.
