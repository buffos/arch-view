# Python analysis discovery notes

## Purpose

Provide useful static package/module architecture for Python while making dynamic import behavior visible instead of pretending it is fully resolvable.

## Target boundary

The analyzer reads project configuration and Python syntax, discovers package/module files, resolves absolute and relative imports against the project boundary where possible, and returns evidence, metadata, confidence, and diagnostics. It never imports the target project or executes Python code.

## Confirmed decisions

- `pyproject.toml` is the preferred project marker, with common fallback metadata/configuration supported as an analyzer concern.
- `src/` layouts, regular packages, and namespace packages are represented through explicit hierarchy paths rather than filename string heuristics alone.
- Package/module is the default node granularity; files remain evidence. `import` and `from ... import ...` produce `depends_on` observations.
- Relative imports resolve from the importing package context. Absolute imports resolve against configured project roots where possible.
- `importlib`, `__import__`, plugin discovery, conditional imports, and other dynamic cases produce unresolved/dynamic diagnostics with confidence rather than forced edges.
- Tests, `__pycache__`, generated/build/cache/vendor directories, `.git`, and directories named `external` are excluded by default.
- Static AST/configuration parsing is the first implementation. Type-checker or environment-assisted resolution is optional, read-only, and separately reported.

## Open questions for exact specification

- Supported project markers and precedence among `pyproject.toml`, setup metadata, and configured roots.
- Namespace-package and editable-install resolution rules without importing code.
- Syntax-version selection, stub-file handling, conditional imports, and confidence levels.
- Exact treatment of re-exports and package `__init__` evidence.
