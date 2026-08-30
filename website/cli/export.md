# export

The export command turns a canonical model into a file.

Basic example:

~~~powershell
go run ./cmd/arch-view export --input model.json --format html --output architecture.html
~~~

## Required options

| Option | What it does |
| --- | --- |
| **--input** | Reads a canonical model JSON file. |
| **--format** | Chooses json, html, or svg. |
| **--output** | Writes the result to this file. |

## Display options

| Option | What it does |
| --- | --- |
| **--reference-visibility** | Uses hidden, aggregated, or expanded references. Hidden is the default. |
| **--view-path** | Selects a hierarchy segment for the exported visual. Repeat it for a deeper path. |
| **--reference-scope** | Selects which reference scopes are visible. Repeat it when needed. |
| **--deterministic** | Defaults to true so equivalent input produces repeatable output. |
| **--overwrite** | Allows an existing output file to be replaced. |
| **--embed-source** | Requests source contents inside the export. This is unsupported in model version 1. |

## Quality exit options

| Option | What it does |
| --- | --- |
| **--quality-exit-on** | Returns exit code 1 for a matching finding at or above the chosen severity. |
| **--quality-exit-status** | Restricts which finding statuses count. It is repeatable or comma-separated. |

## Which format should I choose?

| Format | Use it for |
| --- | --- |
| json | Machine-readable data. |
| html | A self-contained interactive browser file. |
| svg | A static image that can be embedded or inspected. |

The exporter validates the model before writing. If the model is invalid, it stops instead of producing a misleading file.
