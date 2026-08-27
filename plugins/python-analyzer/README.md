# External Python analyzer

This directory contains the opt-in Arch View Python analyzer deployment. It
is a standard-library-only process and is intentionally separate from the
built-in `internal/analyzers/python` implementation, which remains the parity
baseline and default fallback.

The launcher reads and writes one Arch View analyzer protocol frame per NDJSON
line. It writes protocol frames only to stdout; diagnostics and unexpected
runtime messages go to stderr. It uses `ast`, `tokenize`, and safe data readers
for `pyproject.toml`, `setup.cfg`, and `setup.py`. It never imports, executes,
installs, type-checks, or introspects target repository code.

## Invocation

From the repository root, supply the descriptor explicitly:

```text
arch-view analyzers --plugin plugins/python-analyzer/external-plugin.json
arch-view analyze --project <python-repository> --plugin plugins/python-analyzer/external-plugin.json --analyzer org.archview.python.external --format analysis-json --output analysis.json
arch-view open --project <python-repository> --plugin plugins/python-analyzer/external-plugin.json --analyzer org.archview.python.external
```

The descriptor uses `python launcher.py`; relative script and working-directory
values resolve from the descriptor directory. No plugin is discovered from a
target project or machine-wide PATH scan, and the external analyzer is never
selected implicitly.
