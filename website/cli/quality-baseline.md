# quality baseline

The quality baseline subcommand creates a JSON file from a quality report.

## Baseline every active finding

~~~powershell
go run ./cmd/arch-view quality baseline --input analysis.json --output quality-baseline.json --baseline-id baseline:main --all-active --reason "Reviewed legacy findings."
~~~

## Baseline selected findings

~~~powershell
go run ./cmd/arch-view quality baseline --input analysis.json --output quality-baseline.json --baseline-id baseline:main --finding finding-one --finding finding-two --reason "Accepted for the current migration."
~~~

## Options

| Option | What it does |
| --- | --- |
| **--input** | Reads an analysis, model, or quality-report JSON file. |
| **--output** | Writes the baseline file, or uses a single dash for standard output. |
| **--baseline-id** | Names the baseline. It must be a namespaced value such as baseline:main. |
| **--finding** | Selects one finding ID or stable finding key. Repeat it for multiple findings. |
| **--all-active** | Selects every active finding. It cannot be combined with --finding. |
| **--reason** | Records why the finding was accepted. It is required. |
| **--owner** | Records the person or team responsible for the decision. |
| **--revision** | Records a human-readable baseline revision. |
| **--overwrite** | Allows an existing baseline file to be replaced. |

The command refuses to create an empty baseline. That is deliberate: an empty or unexplained baseline hides the review decision.

Read [Baselines](/quality/baselines) before using --all-active.
