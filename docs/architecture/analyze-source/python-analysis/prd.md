# Python analysis PRD

## Purpose

Provide useful static package/module architecture for Python while exposing dynamic-import uncertainty honestly.

## Scope and boundary

Read Python source/configuration, discover packages/modules, resolve deterministic absolute and relative imports, and emit common observations/evidence/diagnostics. Never import, execute, install, or introspect the target project.

Project marker precedence: explicit analyzer roots/options; `pyproject.toml`; `setup.cfg`; `setup.py` metadata. Source roots: explicit configuration, then `src/` when present, then project root.

Defaults exclude tests, `__pycache__`, generated/build/cache/vendor directories, `.git`, and directories named `external`. `.py` files are included; `.pyi` stubs require `include_stubs=true`.

## Functional requirements

| ID | Requirement |
|---|---|
| PY-FR-001 | Detect configured/common Python project boundaries without importing code. |
| PY-FR-002 | Discover regular and namespace packages under effective source roots. |
| PY-FR-003 | Resolve absolute and relative static imports where filesystem/configuration proves the target. |
| PY-FR-004 | Emit dynamic/conditional import diagnostics with confidence. |
| PY-FR-005 | Preserve module/package/file evidence and deterministic output. |

## Non-goals

Executing Python, importing installed environments, complete runtime plugin discovery, type-checking, and call graphs.
