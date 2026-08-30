# Live analysis and MCP acceptance scenarios

These are backend/contract scenarios. They cover the shared read model and
security boundary; visual rendering remains a downstream consumer concern.

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

**Then** it enforces the configured default max bytes/items, sets truncation and
omitted fields, and returns a cursor. Full source content is not included.

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

## LAM-AC-016 — Read-only permission boundary

**Given** a v1 MCP session

**When** a client requests shell execution, target application execution,
configuration write, or source edit

**Then** the operation is rejected with permission/unsupported diagnostics and
the live snapshot remains unchanged.

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
