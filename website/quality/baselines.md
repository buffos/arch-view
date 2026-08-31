# Baselines

A baseline is a list of findings that a person reviewed and accepted for now.
It is not a repair. It does not say that the code is perfect.

## The normal workflow

Use one canonical baseline for each quality profile:

~~~text
run checks
  -> fix real findings
  -> review the remaining findings
  -> append accepted findings to the profile's baseline
  -> run checks again
~~~

The profile points to the baseline by `baseline_id` and `revision`. Arch View
then loads that exact baseline automatically. It does not load every JSON file
in `quality-baselines/`.

## CLI: add to the managed baseline

First run the profile without a baseline, or use `--no-quality-baseline` for a
clean report:

~~~powershell
go run ./cmd/arch-view analyze --project . --quality-profile quality-profiles/full.json --no-quality-baseline --format analysis-json --output quality-report.json
~~~

After reviewing a finding, append it to the canonical file:

~~~powershell
go run ./cmd/arch-view quality baseline add --project . --profile quality-profiles/full.json --baseline-file main.json --input quality-report.json --finding <finding-key> --reason "Accepted because this generated file is controlled by another tool."
~~~

The command looks in `quality-baselines/main.json`. It creates the file when
needed, adds the entry without duplicates, updates the profile reference, and
bumps the baseline revision. A new baseline starts at `1.0.0`; the next entry
update normally changes it to `1.0.1`.

If you do not provide `--baseline-id`, the first managed baseline gets an ID
from its filename. For example, `main.json` becomes `baseline:main`. Use
`--baseline-id baseline:team-main` when you need a different identity.

Run the same analysis command again. Because the profile now points to the
baseline, Arch View loads it automatically:

~~~powershell
go run ./cmd/arch-view analyze --project . --quality-profile quality-profiles/full.json --format analysis-json --output quality-report.json
~~~

Accepted exact findings become suppressed. New findings and changed exact
findings remain visible.

## Useful CLI choices

| Option | What it does |
| --- | --- |
| `--quality-baseline <path>` | Uses one explicit baseline for this run. It does not edit the profile. |
| `--no-quality-baseline` | Ignores the profile's baseline for this run. It cannot be combined with the explicit path. |
| `--expected-revision <version>` | Stops the append if another process changed the baseline first. |
| `--revision <version>` | Supplies an explicit next revision. Otherwise Arch View increments it. |
| `--owner <name>` | Records the person or team that accepted the finding. |

The explicit baseline must be valid and must match the temporary profile
reference used for that run. An invalid or ambiguous automatic baseline is an
error. A missing referenced baseline produces a warning and does not suppress
anything.

## What may be added?

Only a finding that is all of the following may enter a managed baseline:

- active in the current report;
- covered by an observed rule result;
- reviewed by a person or agent;
- accompanied by a specific reason.

Unsupported, not-evaluable, partial, stale, and failed results are rejected.
Do not baseline a result just because it is confusing. Ask what the result
measured first.

## Exact checks and signals

An exact check records a defined measurement. For example, a file with 620
lines is over a configured limit of 500. Decide whether that is a real problem
before accepting it.

A SOLID result is an advisory signal. It points to a design review. It is not
proof of a violation. Read the source and record the reason if you accept it.

## Baseline entries and revisions

Each entry stores the stable finding key, exact rule and profile versions,
formula versions, reason, and optional owner. The finding payload is not copied
into the baseline.

The baseline schema is `arch-view.quality-baseline/v1`. Revisions are part of
the exact match. When a rule, profile, formula, or finding identity changes,
the old entry does not silently suppress the new result.

## Legacy standalone command

The older command remains available for creating a separate baseline file:

~~~powershell
go run ./cmd/arch-view quality baseline --input quality-report.json --output quality-baseline.json --baseline-id baseline:main --finding <finding-key> --reason "Accepted after review."
~~~

It is useful for export or migration. It does not update a profile reference.
Use `quality baseline add` for the normal repeatable workflow.
