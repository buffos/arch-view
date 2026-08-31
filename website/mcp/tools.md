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
| `get_quality_findings` | Read bounded findings, statuses, coverage, and evidence summaries. |
| `get_finding_evidence` | Read the evidence for one finding; source context is opt-in and bounded. |
| `evaluate_quality` | Evaluate a profile or temporary rule bindings without persisting them. |
| `compare_quality_reports` | Compare two compatible reports and classify changes. |
| `validate_quality_profile` | Validate a profile before saving or using it. |
| `preview_baseline` | Show the exact findings a baseline would record without writing a file. |
| `save_quality_profile` | Save an existing profile only when the session grants the operation and authorization. |
| `save_quality_profile_as` | Save a validated profile as a new document. It cannot overwrite an existing profile. |
| `create_baseline` | Record selected active findings with their exact rule/profile/formula versions. |

The quality tools do not decide whether a finding is a real design problem.
Exact checks calculate a defined value. Advisory signals suggest a review.
Incomplete coverage is always reported so an assistant does not mistake
missing analyzer data for a clean result.

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
