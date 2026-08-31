# Live analysis and MCP acceptance scenarios

These are backend/contract scenarios. They cover the shared read model,
freshness assurance, analyzer-neutral agent queries, quality-policy boundary,
and security contract; visual rendering remains a downstream consumer concern.

## LAM-AC-001 — Initial ready snapshot

**Given** a valid session with an allowed repository root and source/quality
requests

**When** the session starts

**Then** it performs an initial bounded scan and publishes one coherent ready
revision only after source index, model, and requested quality report validate
against the same input fingerprint.

## LAM-AC-002 — Invalid roots fail closed

**Given** a live config containing an absolute root, `..` escape, or symlink
that resolves outside the opened repository

**When** the session is validated

**Then** validation fails with a root/permission diagnostic and no watcher or
query access is granted.

## LAM-AC-003 — Events are normalized and debounced

**Given** multiple modify events for the same file and related files within the
configured debounce window

**When** the watcher coordinator processes them

**Then** it produces one canonical event group with deduplicated root-safe paths
and at most one candidate rebuild for that group.

## LAM-AC-004 — Overflow triggers rescan

**Given** a watcher overflow, backend error, or ambiguous rename

**When** the coordinator plans invalidation

**Then** it selects full rescan for the affected session/scope, records the
reason, and never commits guessed incremental facts.

## LAM-AC-005 — Readers never see a mixed revision

**Given** revision 12 is ready and revision 13 is being built

**When** a query arrives during the build

**Then** it returns revision 12 with `updating`/`stale` freshness. After commit,
new queries return revision 13; no response combines old source facts with new
quality findings.

## LAM-AC-006 — Failed update preserves last ready

**Given** revision 12 is ready and a changed file causes analyzer/quality
failure

**When** candidate publication fails

**Then** revision 12 remains queryable, status is stale/degraded with explicit
diagnostics and changed paths, and no empty replacement is published.

## LAM-AC-007 — Unchanged rescan does not create semantic noise

**Given** a filesystem event occurs but raw source hashes and requested policy
produce the same semantic digest

**When** the rescan completes

**Then** the coordinator may acknowledge the event without a new semantic
revision, while operational event status remains available.

## LAM-AC-008 — Selective invalidation is conservative

**Given** one changed file belongs to one analyzer scope and cache/input
identity proves the unaffected scopes independent

**When** invalidation is planned

**Then** only the proven affected scope is rebuilt. If dependency impact or
scope ownership is uncertain, the coordinator plans a broader rescan.

## LAM-AC-009 — Structural search is deterministic

**Given** a ready snapshot and a path/language/symbol/documentation query

**When** the same query, projection, ordering, and budget run twice

**Then** result order, counts, truncation, and cursor semantics are equal and
results identify one snapshot/revision.

## LAM-AC-010 — Cursor cannot cross revisions

**Given** a cursor created for revision 12

**When** it is used with revision 13 or a changed query/projection/budget

**Then** the server rejects it as invalid rather than silently restarting or
returning mixed pages.

## LAM-AC-011 — Quality query shares the report contract

**Given** a current quality report with active, suppressed, and not-evaluable
results

**When** `get_quality_findings` is called

**Then** it returns those statuses, coverage, rule/formula versions, evidence
summaries, and the exact revision; suppressed findings are not deleted.

## LAM-AC-012 — Unsupported semantic query is explicit

**Given** a language scope without caller/callee extraction capability

**When** `get_callers_callees` is requested

**Then** the response is bounded and identifies `unsupported`/`unknown` coverage;
it does not look like a successful empty caller list.

## LAM-AC-013 — Default response is token-bounded

**Given** a large repository query with no explicit budget

**When** an MCP tool responds

**Then** it enforces the operation-specific compact default and configured hard
max bytes/items, sets truncation and omitted fields, and returns a cursor. Full
source content is not included.

## LAM-AC-014 — Source context is explicit and safe

**Given** an indexed entity/span and a bounded context request

**When** `get_source_context` is called

**Then** it returns only the requested root-safe lines/bytes with matching hash;
arbitrary paths, traversal, over-limit requests, and out-of-scope spans are
rejected.

## LAM-AC-015 — MCP and viewer/CLI parity

**Given** the same snapshot, query, projection, and budget

**When** viewer, CLI, and MCP adapters ask the shared query provider

**Then** semantic records, statuses, evidence, ordering, and revision metadata
match; only transport envelopes differ.

## LAM-AC-016 — Default read-only permission boundary

**Given** a v1 MCP session with its default read-only operation policy

**When** a client requests shell execution, target application execution,
arbitrary configuration write, or source edit

**Then** the operation is rejected with permission/unsupported diagnostics and
the live snapshot remains unchanged. Explicit quality-profile or baseline
operations are allowed only through their separate validated allowlisted
permission and authorization.

## LAM-AC-017 — Transport is replaceable

**Given** the same session and query provider exposed through stdio and an
explicit authenticated HTTP adapter

**When** equivalent requests are made

**Then** both return the same versioned query semantics, while the network
adapter additionally enforces its authentication/origin policy.

## LAM-AC-018 — External edit handoff is explicit

**Given** a quality finding and a future remediation proposal

**When** no remediation capability/permission is enabled

**Then** MCP can return evidence for the finding but cannot write a patch. If a
future authorized workflow edits the file, the watcher observes a new event and
re-evaluates it as a new candidate revision.

## LAM-AC-019 — Strict query reconciles missed watcher events

**Given** the watcher reports no pending event but a file changed after the
last ready revision

**When** a client requests findings with `consistency=require_current`

**Then** the coordinator's reconciliation detects the changed input, starts or
joins a bounded reanalysis, and returns only the newly verified revision as
`current`; it never trusts the clean watcher state alone.

## LAM-AC-020 — Edit storms produce one coherent build

**Given** an editor emits many modify/rename events for several files during a
short edit burst

**When** the debounce window closes and multiple clients request current data

**Then** events are deduplicated into one event group, clients join one
single-flight reconciliation/build, and the published revision includes the
verified final input or reports `input_unstable`.

## LAM-AC-021 — Source changes during analysis are not current

**Given** a candidate build starts and one included file changes before the
candidate is published

**When** the coordinator verifies the input fingerprint

**Then** it discards and retries the candidate within policy; after the retry
limit it retains the last-ready revision and reports `input_unstable` rather
than publishing a stale candidate as current.

## LAM-AC-022 — Analyzer-neutral capability coverage

**Given** a session contains scopes produced by different registered analyzers

**When** an agent searches symbols, documentation, text, or callers/callees

**Then** the query uses the same language-neutral contract, identifies the
contributing analyzer/capability per scope, and reports unsupported/unknown
coverage explicitly without a language-specific MCP branch.

## LAM-AC-023 — Temporary quality settings do not persist

**Given** a verified source revision and a valid profile

**When** an agent requests `evaluate_quality` with temporary rule bindings and
`persist=false`

**Then** the returned report records the effective profile/options identity and
uses the deterministic-quality service, while the project profile document and
future default evaluation remain unchanged.

## LAM-AC-024 — Quality policy writes require explicit permission

**Given** a session with the default read-only operation policy

**When** a client requests profile save or baseline creation

**Then** the request is rejected without changing project files. When the
separate policy-write permission and authorization are granted, the request is
validated, restricted to the profile/baseline destination, and audited.

## LAM-AC-025 — Baseline requires a current compatible report

**Given** a baseline request refers to an old, partial, or incompatible report

**When** `create_baseline` is requested

**Then** the policy service rejects it with an explicit stale/incompatible
diagnostic. A baseline can be created only from the exact selected finding keys
and compatible rule/profile/formula versions of a verified report.

## LAM-AC-026 — Agent fix loop compares revisions

**Given** a quality finding is returned at revision 12 and the agent edits the
affected source outside MCP

**When** the agent waits for current revision 13 and compares the two quality
reports

**Then** the result identifies added, unchanged, suppressed, and resolved
findings by stable finding key, and partial/unsupported coverage is not treated
as resolution.

## LAM-AC-027 — Baseline listing is bounded and read-only

**Given** a project with several direct `quality-baselines/*.json` documents

**When** an MCP client calls `get_quality_baselines`

**Then** it receives deterministic IDs, revisions, statuses, entry counts, and
pagination metadata. Entries are returned only when one safe filename is
selected with an explicit bounded request. The call does not change the
session's effective baseline or project files, and traversal/nested filenames
are rejected.

## LAM-AC-028 — Evaluation baseline mode is request-scoped

**Given** a profile that references `baseline:main@1.0.0`

**When** one `evaluate_quality` request uses `baseline_mode=none` and another
uses `baseline_mode=selected` with explicit baseline files

**Then** the first report has no suppression, the second uses only the
selected compatible documents, and the saved profile and later default
evaluation still use the original profile reference.

## LAM-AC-029 — Authorized append updates the canonical profile baseline

**Given** a current compatible report with reviewed active findings and a live
session that grants baseline-write permission

**When** the client calls `append_baseline` with authorization and a reason

**Then** the operation creates or merges the profile's canonical baseline,
increments its revision, updates the profile reference, returns added and
already-existing finding keys, records an audit result, and sets
`reevaluation_required` when policy changed. It never edits source files.

## LAM-AC-030 — Append is protected by eligibility and conflicts

**Given** a report containing unsupported, not-evaluable, partial, stale, or
non-active findings, or a baseline whose same exact identity has another reason

**When** `append_baseline` is requested

**Then** the request is rejected without changing either policy document. A
caller with an outdated expected baseline revision receives a conflict instead
of overwriting a newer append.

## LAM-AC-031 — Startup evaluation uses the saved baseline loader

**Given** a live session whose selected profile points to a valid canonical
baseline

**When** the session publishes its startup quality report and an MCP client
later performs a default `evaluate_quality`

**Then** both reports apply the same exact baseline reference and suppression
semantics. A missing reference produces an explicit warning and leaves findings
unsuppressed; an invalid or ambiguous reference prevents an incoherent report.
