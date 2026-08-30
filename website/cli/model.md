# model commands

Model commands work with the canonical JSON format.

## normalize

Converts analysis JSON into canonical model JSON.

~~~powershell
go run ./cmd/arch-view model normalize --input analysis.json --output model.json
~~~

| Option | What it does |
| --- | --- |
| **--input** | Reads analysis JSON. |
| **--output** | Writes model JSON, or uses a single dash for standard output. |

Normalization sorts collections, derives graph information, and validates the result.

## validate

Checks a model without writing a new one.

~~~powershell
go run ./cmd/arch-view model validate --input model.json
~~~

| Option | What it does |
| --- | --- |
| **--input** | Reads the model JSON to validate. |

## projection

Creates a smaller hierarchy view.

~~~powershell
go run ./cmd/arch-view model projection --input model.json --output projection.json --path internal --path analyzers
~~~

| Option | What it does |
| --- | --- |
| **--input** | Reads the canonical model JSON. |
| **--output** | Writes the projection JSON, or uses a single dash for standard output. |
| **--path** | Selects one hierarchy segment. Repeat it to select a deeper path. |

Use projection when a full graph is too large for the question you are asking.
