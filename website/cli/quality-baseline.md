# quality baseline

Arch View has two baseline commands.

- `quality baseline` creates one standalone baseline file. It is kept for
  compatibility and migration.
- `quality baseline add` appends reviewed findings to one project-managed
  baseline and updates the selected profile to point to it.

Read [Baselines](/quality/baselines) for the human workflow.

## Managed append command

~~~powershell
go run ./cmd/arch-view quality baseline add --project . --profile quality-profiles/full.json --baseline-file main.json --input quality-report.json --finding <finding-key> --reason "Accepted because this generated file is controlled by another tool."
~~~

For all active and observed findings, use `--all-active` instead of
`--finding`. Use that only when every selected finding has been reviewed:

~~~powershell
go run ./cmd/arch-view quality baseline add --project . --profile quality-profiles/full.json --baseline-file main.json --input quality-report.json --all-active --reason "Reviewed legacy findings during migration."
~~~

The baseline file is relative to `quality-baselines/`. The profile file is
relative to `quality-profiles/` when you pass a managed path. Both files are
updated with atomic document writes. If the profile or baseline changed during
the operation, the command fails instead of overwriting the other decision.

The command returns JSON with the new revision, added keys, existing keys, and
`reevaluation_required`. Run `analyze` again after an update.

## Options

| Option | What it does |
| --- | --- |
| **--project** | Project root containing `quality-profiles/` and `quality-baselines/`. Required. |
| **--profile** | Managed profile JSON file. Required. |
| **--baseline-file** | Direct JSON filename inside `quality-baselines/`. Required. |
| **--input** | Analysis or quality-report JSON file. Required. |
| **--finding** | Adds one report ID or stable finding key. Repeat it for selected findings. |
| **--all-active** | Adds every active finding with observed coverage. It cannot be combined with `--finding`. |
| **--baseline-id** | Identity for a new baseline. If omitted, `main.json` becomes `baseline:main`. |
| **--reason** | Why the finding is accepted. Required. |
| **--owner** | Person or team responsible for the decision. |
| **--revision** | Explicit next baseline revision. Normally Arch View increments it. |
| **--expected-revision** | Expected current revision. Use it to detect a concurrent change. |

Only active findings with observed coverage can be added. Unsupported,
not-evaluable, partial, stale, and failed findings are rejected. A duplicate
with the same reason is harmless. A duplicate with a different reason is an
error because it represents a conflicting decision.

## Standalone command

~~~powershell
go run ./cmd/arch-view quality baseline --input quality-report.json --output quality-baseline.json --baseline-id baseline:main --finding <finding-key> --reason "Accepted after review."
~~~

| Option | What it does |
| --- | --- |
| **--input** | Reads an analysis, model, or quality-report JSON file. |
| **--output** | Writes the baseline file, or `-` for standard output. |
| **--baseline-id** | Names the baseline. It must be a value such as `baseline:main`. |
| **--finding** | Selects one finding ID or stable finding key. Repeat it for more findings. |
| **--all-active** | Selects every active finding with observed coverage. |
| **--reason** | Records why the finding was accepted. Required. |
| **--owner** | Records the responsible person or team. |
| **--revision** | Sets the baseline revision. |
| **--overwrite** | Allows replacement of an existing standalone file. |

The standalone command does not attach the file to a profile. Use the managed
command when later analyses should discover the baseline automatically.
