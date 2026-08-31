# Agent skill

Arch View includes a small project-local agent skill at
`.codex/skills/arch-view/SKILL.md`. It tells a coding agent how to use Arch
View without first reading Arch View's implementation. The skill supports both
MCP and the command line. It uses MCP when the configured server is available
and falls back to the Arch View CLI when it is not.

The local skill is split into focused reference files. It includes separate
instructions for the quality-profile JSON, quality-report fields and evidence,
MCP requests, and CLI commands. The agent reads the references needed for the
current task instead of loading every format at once.

The skill is useful when an agent needs to:

- find architecture and source-quality problems;
- find missing public documentation and the reported source location;
- inspect file size, callable size, complexity, and nesting;
- inspect cycles, forbidden dependencies, layer direction, and coupling;
- review SOLID indicators without treating them as proof;
- create a quality report for one analyzer or all registered analyzers; or
- review and create a baseline.

## Short request

The skill contains the detailed MCP and CLI workflow, so a short request is
enough:

~~~text
Use the Arch View skill to perform a complete quality review of this repository. Temporarily use a 600-line file threshold, inspect all SOLID signals, fix real findings, append only reviewed accepted findings to the canonical baseline, re-scan after each fix, and report changes and remaining findings.
~~~

The skill chooses MCP when it is configured. Otherwise it runs the equivalent
review with `arch-view analyze` and local terminal tools. For the full
MCP-specific prompt, see [MCP tools](/mcp/tools#copy-paste-prompt-for-an-mcp-quality-review).

## Copy-paste skill workflow

Use this prompt when you want the project-local skill to choose the available
transport and handle the complete quality-review and baseline workflow:

~~~text
Use the Arch View skill for this quality review. Use the configured Arch View MCP server when available; otherwise use Arch View CLI and normal terminal tools.

Goal:

- Check the current repository for quality findings.
- Use a temporary file line threshold of 600 for this evaluation.
- Inspect all five SOLID signals.
- Decide which findings represent real problems.
- Fix real problems using normal editing tools.
- Re-scan after every fix.
- Append only reviewed, intentionally accepted findings to the profile's canonical baseline.
- Never baseline unsupported, not-evaluable, incomplete, stale, failed, or already suppressed findings.

Workflow:

1. Establish a current analysis. With MCP, call ensure_current_snapshot with consistency=require_current and report freshness, revision, coverage, and partial scopes. With the CLI, run a new analyze command against the repository.
2. Discover the exact full profile and rule catalog. With MCP, use get_quality_profiles and get_quality_rules. With the CLI, read the selected profile JSON and preserve its complete rule-binding shape.
3. Inspect saved baselines. With MCP, use get_quality_baselines. With the CLI, inspect only the profile-referenced file under quality-baselines/. Do not guess that every JSON file is active.
4. Run a clean temporary evaluation with no baseline suppression. With MCP, use baseline_mode=none. With the CLI, use --no-quality-baseline. Set source:file.max-lines to operator greater_than, limit 600, unit unit:line.
5. Keep every other profile binding unchanged and include signal:solid.srp, signal:solid.ocp, signal:solid.lsp, signal:solid.isp, and signal:solid.dip.
6. Retrieve all file-size and SOLID findings. Follow MCP cursors; filter the complete CLI JSON locally. Read evidence and small bounded source context for each SOLID finding. Treat SOLID results as review signals, not proof of violations.
7. Fix only findings that are real problems. Use normal editing tools for source changes.
8. After every fix, establish a current analysis again and re-run the same clean evaluation.
9. For an accepted finding, confirm it is active in the newest observed report and record a specific reason. With MCP, call append_baseline with the exact current finding_keys, report identity, and authorized write access. If no profile baseline exists, provide a safe file_name such as main.json. With the CLI, run quality baseline add with the newest report, profile, baseline file, exact finding key, and reason.
10. Never append unsupported, not-evaluable, partial, stale, failed, resolved, or suppressed findings.
11. Re-run the normal profile evaluation after appending. With MCP, use baseline_mode=profile. With the CLI, omit --no-quality-baseline and --quality-baseline so the profile reference is loaded automatically.
12. Confirm accepted findings are suppressed, changed or new findings remain visible, and incomplete coverage is still reported. Finish with changed files, remaining findings, and baselined findings.
~~~

This is a workflow request, not a requirement to use MCP. The local skill
references define the exact profile fields, report fields, MCP request fields,
CLI command, and baseline validation rules. The skill should read only the
reference needed for the selected transport and current task.

## The same checks through MCP or the CLI

MCP and the CLI use the same quality evaluator. The difference is how the
agent reaches it:

| Need | MCP | CLI fallback |
| --- | --- | --- |
| Verify fresh input | `ensure_current_snapshot` | Run `analyze` again |
| List analyzers/scopes | `list_scopes` | Use `--language` or `--analyzer` when needed |
| List profiles and rules | `get_quality_profiles`, `get_quality_rules` | Read the selected profile JSON |
| Run checks | `evaluate_quality` | `analyze --quality-profile ... --format analysis-json` |
| Filter findings | `get_quality_findings` with `rule_ids` | Filter `quality_report.findings` in the JSON |
| Get lines and evidence | `get_finding_evidence` | Read each finding's evidence in the JSON |
| Read saved baselines | `get_quality_baselines` | Read `quality-baselines/*.json` locally |
| Evaluate with a baseline | `evaluate_quality` with `baseline_mode` | `analyze` automatically loads the profile reference |
| Add reviewed findings | `append_baseline` | `quality baseline add` |

For a one-shot report, run this from the target repository when `arch-view` is
on `PATH`:

~~~text
arch-view analyze --project . --quality-profile quality-profiles/full.json --format analysis-json --output quality-report.json
~~~

Replace `quality-profiles/full.json` with a profile that exists in the target
repository.
The file contains `quality_report.findings`, `quality_report.coverage`, and
`quality_report.diagnostics`. A finding for
`source:public-symbol.documentation` reports public symbols whose documentation
status was observed as absent. Its evidence contains the file and, when the
analyzer supplied one, a line range.

## Recommended agent loop

1. Ensure the snapshot is current.
2. Select the scope and profile.
3. Read the rule catalog before changing thresholds or interpreting a result.
4. Request only the rule findings needed for the current question.
5. Request evidence and a small source excerpt only for findings under review.
6. Fix findings that are real design or source problems.
7. Read the saved baseline when you need to understand existing suppressions.
8. Append only current, reviewed findings with an explicit reason and
   authorization.
9. Re-scan and compare the new report. The append result tells you that a new
   evaluation is required.

Coverage matters. `unsupported` is not a pass: the analyzer did not provide a
required fact. `not_evaluable` is not a pass: the available facts were not
enough to calculate the check. SOLID entries are advisory signals. They point
to a review; they do not prove an SRP, OCP, LSP, ISP, or DIP violation.

The agent should use normal editing tools for source changes. Arch View reads
the repository, returns bounded evidence, and writes profiles/baselines only
when the session explicitly permits those policy operations. A missing
baseline is a warning with no suppression. An invalid or ambiguous baseline is
an error.

See [MCP tools](/mcp/tools), [quality findings](/quality/findings),
[quality rules](/quality/rules), and [baselines](/quality/baselines) for the
human explanation of the underlying workflow.
