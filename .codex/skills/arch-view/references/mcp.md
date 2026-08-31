# MCP workflow

Read this file when an `arch_view` MCP server is available. MCP is a live,
revision-aware query boundary. It does not edit source files or run shell
commands. Use normal agent editing tools for source changes.

## Required sequence

Use one session and keep the returned revision consistent:

1. `ensure_current_snapshot` with `consistency: "require_current"`.
2. Report `freshness`, `revision`, `coverage`, and partial or failed scopes.
3. `list_scopes` and select the intended analyzer scope IDs. Use all scopes when
   the user asked for the repository-wide report.
4. `get_quality_profiles` to discover profile IDs and versions.
5. `get_quality_rules` with the selected profile ID and version. This returns
   the rule catalog, parameter schemas, effective bindings, capabilities, and
   limitations.
6. `evaluate_quality` with the exact profile ID and version, strict freshness,
   and `persist: false`.
7. `get_quality_findings` using `query.rule_ids`, `query.statuses`, and bounded
   `max_items`. Follow `next_cursor` until every requested finding is read.
8. `get_finding_evidence` for each finding under review. Request bounded source
   context only for that finding.

The tool name and server name come from the MCP client. Do not assume that a
server called `arch_view_local` exists unless the client exposes it.

## Key request fields

| Tool | Important fields |
| --- | --- |
| `ensure_current_snapshot` | `consistency: "require_current"` |
| `list_scopes` | `consistency`, `revision`, `max_items`, `max_bytes` |
| `get_quality_profiles` | `profile_id`, `profile_version` when selecting one, plus revision fields |
| `get_quality_rules` | selected `profile_id` and `profile_version`; use the returned parameter schema |
| `evaluate_quality` | `profile_id`, `profile_version`, `consistency`, `revision`, `scope_ids`, `rule_bindings`, `baseline_mode`, `baseline_files`, `persist: false` |
| `get_quality_baselines` | `file_names`, `baseline_id`, `baseline_revision`, `include_entries`, `cursor`, bounded budgets |
| `get_quality_findings` | `report_id`, `query.rule_ids`, `query.statuses`, `cursor`, bounded budgets |
| `get_finding_evidence` | `report_id`, `finding_id`, `include_source_context`, `max_lines`, `max_context_bytes` |

Every response is an `arch-view.query/v1` envelope. Read its freshness,
coverage, omitted fields, budget, diagnostics, and cursor before trusting the
nested result. `latest_ready` can be stale. Use `require_current` for quality
evaluation and after edits.

## Temporary profiles

Follow [the profile reference](quality-profile.md). If `evaluate_quality` gets
`rule_bindings`, that array replaces the selected profile's complete
`enabled_rules` array. It does not merge with it. Send every original binding,
changing only the requested threshold and explicitly enabling the requested
SOLID rules. Do not send only `source:file.max-lines` and the five SOLID rules,
because that would silently drop the other full-profile checks.

The result of `evaluate_quality` is temporary. It does not save a profile and
`persist: true` is rejected for this operation.

## Evidence and changes

Use the finding's `finding_id` with `get_finding_evidence`. For source context,
keep the request bounded and use the same report revision. Treat SOLID results
as signals. Make source changes with normal editing tools, not MCP.

After every source change:

1. call `ensure_current_snapshot` with `require_current`;
2. evaluate the same profile and temporary bindings again;
3. retrieve the affected findings again; and
4. compare reports when the previous and current revisions are compatible.

## Baselines and writes

Only baseline active findings that were reviewed and intentionally accepted.
Never baseline unsupported, not-evaluable, partial, stale, or failed results.

`evaluate_quality` defaults to `baseline_mode: "profile"`. It resolves the
profile's exact baseline reference. Use `baseline_mode: "none"` for a clean
evaluation. Use `baseline_mode: "selected"` with safe project-relative
`baseline_files` for a temporary explicit list. Selected files affect only
that evaluation and do not change session state.

For a complete review, start with `baseline_mode: "none"`. This prevents an
existing baseline from hiding findings that still need to be reviewed. After
source fixes and baseline appends are complete, run the normal evaluation with
`baseline_mode: "profile"` and confirm that only the intended accepted
findings are suppressed.

Use `get_quality_baselines` to list available files. To read entries, pass
exactly one `file_names` value and `include_entries: true`; follow
`next_cursor` when the result is bounded.

For an accepted finding, first confirm that it is still active with observed
coverage in the newest clean report. Then use `append_baseline` with the
current report ID and revision, exact `finding_keys`, a specific `reason`,
and (when the profile has no existing baseline) a safe `file_name` and
optional `baseline_id`. If the profile already references a unique baseline,
its file can be resolved without repeating the filename. Pass the current
baseline revision as `expected_baseline_revision` when available so a
concurrent update is detected. The result lists added versus existing keys,
updates the profile reference when needed, and says
`reevaluation_required: true` when policy files changed. Run
`evaluate_quality` again with `baseline_mode: "profile"`.

The old `preview_baseline`/`create_baseline` pair remains a standalone
baseline workflow. It does not update the profile reference.

Profile and baseline writes are disabled by default. They require the server's
explicit policy-write authorization. If a write is denied, report that fact;
do not retry by weakening the request or inventing an authorization token.
