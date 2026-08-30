# Baselines

A baseline is a list of findings that you reviewed and accepted for now.

## What it does

Suppose a project has an old large file. Every run reports it, even though the team already knows about it.

You can baseline that finding:

~~~text
The finding is known.
The reason is recorded.
Future runs suppress the same, exact finding.
~~~

A baseline does not change the source file. It does not make a real problem disappear. It reduces repeated noise while new work remains visible.

## When to use it

Use a baseline when:

- you understand the finding;
- it is outside the current change;
- the reason is written down;
- someone owns the decision.

Do not baseline a finding just because it is inconvenient or confusing.

## What happens after a code change?

Baselines use the finding's exact identity and version. If the code changes enough to produce a different finding, it can appear again.

If a rule version or profile changes, review the baseline instead of assuming it still applies.

## CLI example

Baseline every active finding:

~~~powershell
go run ./cmd/arch-view quality baseline --input analysis.json --output quality-baseline.json --baseline-id baseline:main --all-active --reason "Known legacy findings; review during migration."
~~~

Baseline one finding:

~~~powershell
go run ./cmd/arch-view quality baseline --input analysis.json --output quality-baseline.json --baseline-id baseline:main --finding finding-id --reason "Accepted because this generated file is controlled by another tool."
~~~

Use **--owner** to record a responsible person or team. Use **--revision** to label the revision of the baseline.

## A safe review loop

1. Run without a baseline once.
2. Read each active finding.
3. Fix findings that are real.
4. Baseline only findings with a written reason.
5. Run again.
6. Review newly appearing findings.

## The important limitation

Baseline is a review decision, not a repair mechanism. If the goal is “no unreviewed problems,” do not baseline everything automatically.
