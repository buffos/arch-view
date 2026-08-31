# CLI workflow

Read this file when no Arch View MCP server is available.

## Choose the executable

Use the installed command when it is on `PATH`:

```text
arch-view analyze --project . --quality-profile <profile.json> --format analysis-json --output quality-report.json
```

When working from the Arch View source checkout, the equivalent is:

```text
go run ./cmd/arch-view analyze --project <target-repository> --quality-profile <profile.json> --format analysis-json --output quality-report.json
```

Use `go run ./cmd/arch-view` only when the current directory is the Arch View
checkout. From another repository, use `arch-view` on `PATH` or a built binary.

With no `--language` or `--analyzer`, `analyze` runs the configured built-in
analyzers and produces the combined model. Add those options only when the
user asks for one analyzer or a narrow scope.

`--quality-profile` is a file path. If the profile has a `baseline` reference,
`analyze` automatically resolves the unique matching file in the target
repository's `quality-baselines/` directory. `--output` is required. The
`analysis-json` command writes a complete JSON result with `quality_report`.
Do not add `--overwrite` to this format. A later analysis can write the same
JSON output path again.

## Run a current report

1. Locate the selected profile JSON. Do not assume that a profile is built into
   the executable.
2. Read and validate its identity and rule bindings using
   [the profile reference](quality-profile.md).
3. Make the temporary 600-line copy if requested.
4. Run `analyze` with that temporary profile.
5. Read the JSON structurally at `quality_report`. Use
   [the report reference](quality-report.md) before interpreting it.

The CLI has no separate live snapshot call. Each `analyze` invocation reads
the repository at that moment. After any edit, run the command again. A failed
command or invalid JSON is a failed scan, not a clean report.

## Inspect findings without wasting context

The CLI report contains the complete findings array. It does not use MCP's
cursor pagination. Filter the JSON structurally by `rule_id`, `status`, or
`assessment_kind` with a JSON-aware tool, a short script, or an editor. Do not
paste the entire report into the conversation.

For each finding under review:

1. Read `message`, `comparison`, `limitations`, and `evidence`.
2. Map `evidence.source_spans[].file_id` through
   `source_index.snapshots[].files[]` to get the real path.
3. Use `rg` for an exact symbol or phrase.
4. Read only the reported lines and a small surrounding range.
5. Decide whether the source needs a change. Do not fix a signal from its
   message alone.

## Baseline and verify

For a complete review, first produce a clean report so an existing baseline
does not hide findings:

```text
arch-view analyze --project . --quality-profile quality-profiles/full.json --no-quality-baseline --format analysis-json --output quality-report.json
```

Use this newest clean report as the input to the managed append command:

```text
arch-view quality baseline add --project . --profile quality-profiles/full.json --baseline-file main.json --input quality-report.json --finding <finding-id-or-key> --reason "Accepted because <specific reason>."
```

Use the exact `id` or `finding_key` from the report. Repeat `--finding` for
each separately reviewed finding. The command updates the profile reference
and automatically increments the baseline revision. It is idempotent when the
same entry and reason already exist. Do not baseline unsupported,
not-evaluable, partial, stale, or failed results. Do not use `--all-active` for
a selective review.

Run the same analysis again without a baseline flag:

```text
arch-view analyze --project . --quality-profile quality-profiles/full.json --format analysis-json --output quality-report.json
```

Use `--no-quality-baseline` when you need a clean report, or
`--quality-baseline <path>` for an explicit one-run override. Confirm that only
the intended exact findings become suppressed and that new findings remain
visible.
