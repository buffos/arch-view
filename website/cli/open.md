# open

The open command starts the local browser viewer.

## Open and analyze a project

~~~powershell
go run ./cmd/arch-view open --project . --port 0
~~~

The server reads the project, builds the model, and prints a local address.

## Open an existing model

~~~powershell
go run ./cmd/arch-view open --model architecture.json --port 0
~~~

The viewer uses the model as it is. It does not run a new analysis.

## Input options

Exactly one of these options is required:

| Option | What it does |
| --- | --- |
| **--project** | Analyzes the project before opening the viewer. |
| **--model** | Opens a canonical model JSON file without analyzing a project. |

## Common options

The project form accepts the analyzer and source-selection options described on the [analyze](/cli/analyze) page:

- analyzer runtime, plugin, trust, language, and analyzer selection;
- Go module and build-tag options;
- Rust crate, feature, and target options;
- TypeScript config, JavaScript, runtime, and source-root options;
- Python version and stub options;
- Clojure platform option;
- tests, examples, generated files, external references, exclusions, and safe mode.

## Server option

| Option | What it does |
| --- | --- |
| **--port** | Selects the local TCP port. Use 0 to choose an available port. Valid values are 0 through 65535. |

## Important difference

Open is for a person at a browser. Analyze is for creating a report that another command or CI job can consume.
