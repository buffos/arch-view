# MCP tools

Every tool returns a bounded `arch-view.query/v1` envelope. The envelope tells
the assistant the session, revision, freshness, coverage, result count,
omitted fields, and budget. A small result is not the same as “nothing was
found” when coverage says `unsupported`, `unknown`, `partial`, or
`not_evaluable`.

## Snapshot and navigation

| Tool | Plain meaning |
| --- | --- |
| `get_snapshot_status` | Read the current live state and last ready revision. |
| `ensure_current_snapshot` | Reconcile files and wait for a verified current revision. |
| `list_scopes` | List analyzer scopes, languages, roots, and capabilities. |
| `find_files` | Find indexed files with paths, sizes, and locations. |
| `find_symbols` | Find declarations with names, kinds, locations, and documentation status. |
| `get_documentation` | Find documentation records and their source locations. |
| `find_text` | Search exact literal or bounded regex text without returning whole files. |
| `get_module_facts` | Read files, symbols, and documentation explicitly contained by a module. |
| `get_callers_callees` | Read declared caller/callee relations, or an explicit unsupported state. |
| `get_source_context` | Read a bounded line/byte excerpt for an indexed source location. |

Use `find_text` when you know a phrase. Use `find_symbols` when you know a
declaration name. Use `get_source_context` only after you have an indexed
entity or span.

## Quality tools

| Tool | Plain meaning |
| --- | --- |
| `get_quality_profiles` | List available profile identities and validation status. |
| `get_quality_rules` | Read the complete rule catalog, parameters, and capability requirements. |
| `get_quality_baselines` | List saved baselines or read bounded entries from explicitly selected files. |
| `get_quality_findings` | Read bounded findings, statuses, coverage, and evidence summaries. |
| `get_finding_evidence` | Read the evidence for one finding; source context is opt-in and bounded. |
| `evaluate_quality` | Evaluate a profile or temporary rule bindings without persisting them. It can use the profile baseline, no baseline, or selected temporary baseline files. |
| `compare_quality_reports` | Compare two compatible reports and classify changes. |
| `validate_quality_profile` | Validate a profile before saving or using it. |
| `preview_baseline` | Show the exact findings a baseline would record without writing a file. |
| `save_quality_profile` | Save an existing profile only when the session grants the operation and authorization. |
| `save_quality_profile_as` | Save a validated profile as a new document. It cannot overwrite an existing profile. |
| `create_baseline` | Record selected active findings with their exact rule/profile/formula versions. |
| `append_baseline` | Merge reviewed findings into the canonical baseline, update the profile reference, and request a new evaluation. It is an authorized write. |

The quality tools do not decide whether a finding is a real design problem.
Exact checks calculate a defined value. Advisory signals suggest a review.
Incomplete coverage is always reported so an assistant does not mistake
missing analyzer data for a clean result.

### Baseline modes

`evaluate_quality` uses `baseline_mode: "profile"` by default. It resolves the
profile's `baseline_id` and `revision` to one unique file in
`quality-baselines/`. Use `baseline_mode: "none"` for a clean report. Use
`baseline_mode: "selected"` with `baseline_files: ["main.json"]` for a
temporary explicit selection. A selected baseline changes only that request;
it does not change the profile or session.

`get_quality_baselines` lists safe project-relative filenames. Add
`include_entries: true` and select exactly one `file_names` value to read its
entries. Entries are bounded by `max_items`, `max_bytes`, and `next_cursor`.

The old `preview_baseline` and `create_baseline` pair creates a standalone
baseline. The managed workflow uses `append_baseline` after review. It merges
into the canonical file, updates the profile reference, and returns
`reevaluation_required: true` when anything changed.

## Copy-paste prompt for an MCP quality review

Use this prompt when the configured MCP server is named `arch_view_local` and
you want an agent to inspect the current repository, fix real problems, and
record only reviewed exceptions:

~~~text
Use only the arch_view_local MCP server for analysis.

Goal:

- Check the current repository for quality findings.
- Use a file line threshold of 600 for this evaluation.
- Inspect all SOLID signals.
- Decide which findings represent real problems.
- Fix real problems using normal editing tools.
- Re-scan after every fix.
- For reviewed findings that are intentionally accepted or false positives, append them to the profile's canonical baseline with a clear reason.
- Never baseline unsupported, not-evaluable, incomplete, or stale findings.

Workflow:

1. Call ensure_current_snapshot with consistency=require_current.
2. Report freshness, revision, coverage, and any partial scopes.
3. Read the available profiles and rules.
4. Use profile:full version 1.0.0, or the exact full profile discovered.
5. Discover saved baselines with get_quality_baselines. Do not assume that every JSON file is active; the profile's baseline reference is authoritative.
6. Run a temporary clean quality evaluation with baseline_mode: "none" where source:file.max-lines uses:
   operator: greater_than
   limit: 600
   unit: unit:line
7. Keep all other profile rule bindings unchanged. Include all five SOLID rules:
   signal:solid.srp
   signal:solid.ocp
   signal:solid.lsp
   signal:solid.isp
   signal:solid.dip
8. Retrieve all file-size findings and all SOLID findings, following pagination.
9. For every SOLID finding, retrieve its evidence and a small bounded source context. Treat SOLID results as review signals, not automatic violations.
10. Fix only findings that are real problems.
11. After every source fix, ensure the snapshot is current again and re-run the same clean evaluation.
12. For intentionally accepted findings, confirm they are still active in the newest observed report. Call append_baseline with that report's current report ID and revision, the exact finding_keys, and a specific reason. If the profile has no baseline reference, also provide a safe direct file_name such as main.json and optionally baseline_id such as baseline:main. If it already has a canonical baseline, use its current baseline revision as expected_baseline_revision when available. This requires an MCP server started with policy writes authorized.
13. Treat append_baseline as a policy write, not a source edit. It merges into the canonical baseline, updates the profile reference, and returns reevaluation_required.
14. Re-run evaluate_quality with baseline_mode: "profile". Confirm accepted findings are suppressed, while changed or new findings remain active and unsupported or not-evaluable coverage remains visible.
15. Finish with changed files, remaining findings, and baselined findings.
~~~

The 600-line threshold is temporary. It does not change the saved profile.
`baseline_mode=none` is also temporary: use it for the complete review so an
existing baseline does not hide findings. The final `baseline_mode=profile`
run verifies the normal saved-baseline behavior. If the server has a different
name, replace `arch_view_local` in the first line. Managed baseline appending
requires a server started with the explicitly authorized policy-write options
described in [MCP installation](/mcp/installation#required-options).

## Small workflow

~~~text
get_snapshot_status
  -> ensure_current_snapshot
  -> find_symbols or find_text
  -> get_source_context or get_finding_evidence
  -> edit source with normal tools
  -> ensure_current_snapshot
  -> compare_quality_reports
~~~

Requests can include `consistency: "latest_ready"` for a fast, possibly stale
answer or `consistency: "require_current"` when freshness matters. Results are
bounded by `max_items`, `max_bytes`, and an opaque cursor. Use the cursor to
load another page; do not ask for an unbounded whole repository response.
