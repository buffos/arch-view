# OKF scenario evidence audit

Status: complete following the user's explicit approval on 2026-09-04.
The chronological evidence and former pending-review entries below are retained;
all recorded automated/browser gaps were resolved by subsequent entries. Human
gates are now approved by "Everything is fine now. I approve. proceed to commits
(one or more )". This approval is distinct from agent-run verification.
Canonical source: `docs/architecture/okf-knowledge-views/acceptance-scenarios.md`.
Scenarios use the canonical when-supported backend, frontend, and product-journey
policy. Go HTTP tests do not substitute for supported browser interaction or the
required human review. Additional platform permutations are not implicit gates.

The entries below identify assertions inspected in this audit. Unlisted aspects
remain unverified even when a package's tests pass. Test paths are repository
relative. No issue acceptance checkbox or capability state changes follow from
this table.

| Scenario | Inspected evidence | Remaining verification |
| --- | --- | --- |
| SC-001 | `internal/viewer/okf_bundle_selection_test.go`: deterministic catalog of two valid bundles; select A/B/A; exact concept sets, foreign detail rejection, unchanged source bytes. Live temporary fixture on port 46579: A/B/A selector journey restores A's two concepts and bound profile; B shows only Independent bundle with Neutral | Human visual review remains open |
| SC-002 | `internal/viewer/okf_http_test.go`, `internal/okf/application/service_test.go`: invalid candidate status and rejected selection. Live three-candidate fixture: invalid/.okf disabled before and after Refresh; both valid bundles remain selectable. Rebuilt viewer port 50304: expanded report shows missing YAML frontmatter, bad.md, bundle_id invalid/.okf, and instruction to add valid frontmatter with non-empty type. Port 52225 repair journey: add valid frontmatter, Refresh enables candidate and removes validation error, selection renders only Repaired concept, inspection reads repaired Markdown; other bundles remain available | Human visual review |
| SC-003 | `internal/okf/interaction/markdown_test.go`: metadata visibility, bounded Markdown, source preservation. `okf_source_facts_test.go`: HTTP generic overview, nested zero/false/null/Unicode metadata, exact raw Markdown, sanitized rendering, resolved local-link outcome and unchanged bytes. Live port 52252: expanded metadata/raw/link sections, safe rendered DOM, local link opens target detail, unchanged source SHA-256 | Human visual review |
| SC-004 | SC-001 HTTP test asserts neutral on unbound selection; existing HTTP test asserts projection profile. Live port 54932: unbound second/.okf selects builtin:neutral, renders one name-only concept and keeps invalid/.okf's missing-frontmatter report; editor shows Neutral identity with Save/Rename/Delete disabled; Cancel and node inspection retain Neutral and the report | Human visual review |
| SC-005 | `internal/viewer/okf_profile_switch_test.go`: HTTP Neutral/Fog/Neutral; stable source revision/concept IDs, different styling, Fog state field, exact Neutral restoration, unchanged source bytes, no persisted configuration. Live browser: same 17 IDs; Fog adds 16 fields and computed root/state/roll-up colors; Neutral restores exact node markup/positions with zero fields | Required human visual-review gate; other acceptance scenarios remain independently open |
| SC-006 | `internal/okf/hierarchy/normalize_test.go`: explicit-parent precedence, filesystem fallback, agreeing provenance, ambiguous/cyclic exclusion, preservation of branches outside cycles. `okf_hierarchy_recovery_test.go`: exact five indexed IDs, two valid edges/provenance, scoped conflict, repaired edge and no source mutation. Live port 47048 reproduces conflict exclusion, inspectable source, repair/Refresh and independent-bundle isolation | Human visual review |
| SC-007 | Live Fog depth-2: Go selection adds 3 incident semantic links; roll-up selection replaces them with its incident links. Exact node transforms and breadcrumbs unchanged. Depth-1 graph has only root/direct children despite semantic relationships. Neutral port 48961: area links to a grandchild; depth-2 selection adds one typed semantic overlay without movement; depth-1 excludes the target and selection cannot reveal it or add its edge, with identical positions/breadcrumbs | Human visual review |
| SC-008 | Interaction state tests cover rule-derived states and roll-up exceptions. `internal/viewer/okf_detail_state_test.go`: HTTP Fog projection/detail parity for every visible concept, matching source/bundle identity; agreeing children roll up, mixed or unmapped children retain declared parent state, raw parent frontmatter remains unchanged. Live Fog detail checks cover implemented/mixed states; port 49792 unfamiliar-state fixture retains raw declared/effective value and applies state.unknown token | Human visual review |
| SC-009 | `internal/viewer/okf_depth_sequence_test.go`: HTTP depth 1/2/99/full sequence, exact nodes/hidden counts, retained bundle/Fog profile. Live keyboard depth changes 2→1→2 and Full toggle yield 17→7→17 and 20 visible concepts; Fog retained. Application test rejects zero depth. Port 49231 Full-mode journey preserves one-node budget and exposes hidden-concept recovery. Port 54789: Full and selection honor max_relationships=1, with a truncation report; repair/Refresh restores semantic overlay availability | Human visual review |
| SC-010 | `internal/okf/application/truncation_recovery_test.go`: genuinely hidden target, bounded focus recovery, Back restores prior projection. Live port 49231: Full honors one-node budget; truncation warning and Browse hidden concepts expose target; focus renders target/detail; Back restores root/truncation with profile and Full retained. Port 54789 covers relationship truncation, warning and recovery without dropping source concepts | Human visual review |
| SC-011 | Application/HTTP tests cover Neutral/Fog, depth 1, Full on/off, Top level no-op, Back restoration and empty-history rejection. Live Neutral/depth-2 drill-down restores all 17 concepts and original position. Port 50853 project-profile/depth-2 and port 54594 Neutral/depth-1/Full-off plus Fog/depth-1/Full-on verify focus, Top level and Back retain the unsaved Layered draft, bundle/profile/depth/Full context, counts and breadcrumbs. Configuration hash unchanged | Human visual review |
| SC-012 | HTTP/interaction safety tests plus live isolated fixture: raw script/onerror HTML and remote image omitted; javascript/data/escaping links rendered as inert text with scoped diagnostics; CommonMark bold/heading/code retained; external anchor has `_blank` and `noopener noreferrer`; local link selects target inside viewer. Port 51007 additionally verifies emphasis and a two-item list as real DOM elements; expanded interaction test asserts all requested CommonMark elements and rejects unsafe markup | Human review; external destination navigation is not required by the canonical safe-link-offering assertion |
| SC-013 | `internal/okf/profile/composition_acceptance_test.go`: compatible annotations, scalar priority, equal-priority conflicts, no resurrection, three registration orders. `okf_rule_composition_test.go`: HTTP validate/create/select/repair/reproject, merged annotations, high-priority label, default conflicting role and scoped diagnostic. Live port 52545 confirms winning label, expanded role-conflict report and repair/Refresh | Human visual review |
| SC-014 | `stale_binding_test.go`: missing base/rule/version fallback, diagnostics, retained declaration/binding, unchanged bytes; invalid parameters in `profile_fallback_test.go`. Live missing-base repair on port 51025 and missing-rule/version/schema repair on port 54078 preserve usable views and recover on Refresh. Per-fault hashes prove no rewrite. `schema_recovery_test.go` checks accurate cause, load/save rejection without mutation, and repaired load | Human visual review |
| SC-015 | HTTP Save/Save As/built-in rejection assertions plus live project-profile form Save and Save As; built-in Save/Rename/Delete disabled. Port 54450: built-in Fog Save As creates editable project copy preserving state mapping, rules, decorations and tree layout. Architecture layout, binding, unknown root/profile extensions preserved on disk | Human review |
| SC-016 | `internal/okf/application/configuration_lifecycle_test.go` and service lifecycle test: rename updates inheritance and two bindings; delete without fallback rejected, neutral fallback updates both. Port 54450: Cancel/Escape preserve editor/profile; rename a profile bound to A/B and both select renamed ID; confirmed delete makes both select Neutral. Original fixture profile/configuration preserved | Human visual review |
| SC-017 | Rejected-delete/injected failure tests; real read-path obstruction recovery; Windows delete-sharing denial forces atomic replacement failure, preserves bytes/session, removes temporary files, and permits same-ID retry after handle closure. Live ports 49331/46250 Save reports Access is denied and keeps the draft open. On 46250, Cancel then node inspection uses project:review with its original Custom field; reopening the editor shows the original saved name. Configuration SHA-256 unchanged; zero writer temp files | Human review; no non-Windows live replacement-failure claim |
| SC-018 | HTTP stale-write/retry/target-conflict assertions and live two-editor conflict: second draft retained, first saved name remains on disk; Cancel/Refresh/reopen/manual merge succeeds. Application tests also cover cross-command conflicts. Frontend API retransmission test preserves exact Save As body and operation ID; five HTTP race repetitions return the original creation, reject changed-input reuse and preserve configuration | Human review; the UI does not expose same-ID replay, so no live replay-control claim |
| SC-019 | `internal/okf/application/navigation_atomic_test.go`: paused evaluation, cancellation/supersession, current projection and history preserved. `configuration_projection_test.go`: successful configuration changes supersede paused evaluation; failed save preserves valid pending focus. `okf_supersession_test.go`: real HTTP handlers with paused registered rule report canonical 409 cancellation/supersession envelopes and request IDs; subsequent projection retains expected depth, source/profile/nodes and no rejected Back history; five race runs pass Live port 50528: real worker-load failure retains existing positions; changed focus uses inspectable fallback; worker repair and Back restore normal layout and clear warning. Shared runtime fixes a pending-promise defect missed by mocked rejection tests Live port 49930 held a focus projection, completed a newer Neutral selection, then released the old request. Its 409 okf_operation_superseded response retained the request ID; the browser scene, status, details and navigation remained byte-for-byte equal to the newer DOM-backed snapshot | Human visual review |
| SC-020 | `internal/okf/application/refresh_test.go`: invalidated bundle, selection of independent valid bundle. Live malformed-source fixture clears graph/detail/counts and permits independent-bundle recovery; regression in `okf_projection_reset_test.js`. `source_unreadable_windows_test.go`: real exclusive handle, unreadable status, independent bundle and exact recovery; five race runs pass. Live port 48583 reproduces unreadable-source warning, cleared graph/detail, independent selection, release/Refresh/reselection restoring both concepts and project profile, lock warning removed; source hash unchanged | Human visual review |

## Executed evidence for this audit pass

SC-011 update: the browser Top level/Back journey with a layout draft is now
verified for project:review at depth 2, Full off, on isolated port 50853.
Applied Layered over the saved Mr. Tree layout without saving. Double-click
Local target, Top level, then Back yielded visible counts 1, 2, 1 and restored
the target breadcrumb. Reopening Layout settings after each operation retained
Layered. Configuration SHA-256 remained
B711477F76C7E4591FB6CB02EAF5F1CAD8B8C5719CAB5ECAD02A28E528D775BC.
This supersedes that journey's pending entry above; other profile/depth
variants and human visual review remain open.

SC-018 coverage clarification: same-operation-ID replay is an API behavior,
not a second Save As click. The editor creates a fresh UUID for each command
and exposes no same-ID retransmit control. The frontend API test now sends the
same explicit ID twice and compares complete requests, including both body and
Idempotency-Key. TestOKFHTTPRejectsStaleEditorAndReplaysCreation passes five race
repetitions and proves server replay/conflict/no-mutation semantics. The prior
live two-editor journey remains the supported UI evidence. This follows the
canonical when-supported convention without adding a new retry UI requirement.

Live SC-017 temporary fixture on port 49331: held its configuration open without
Windows delete sharing, then attempted form Save with name Unsaved locked-file
draft. The editor stayed open with that draft and reported project configuration
could not be updated: Access is denied. The selected project:review profile and
both previous graph labels remained visible. After the helper exited, SHA-256
matched the pre-failure baseline exactly and no .archview.json.tmp-* files were
present. Browser inventory confirms the test tab is closed; the remaining test
server was stopped. No live successful retry or post-failure selection is claimed.
The existing application test covers same-operation-ID retry after releasing
the handle. Only the isolated temporary fixture was locked, not this repository.

`go test -race ./internal/okf/application -run
'TestWindowsReplacementFailurePreservesConfigurationAndSession|TestObstructedConfigurationPathPreservesSessionAndAllowsRetry'
-count=5` passes. The Windows test opens the real saved document without delete
sharing, then invokes the application Save path. Replacement fails with the
canonical write-failure code and OS access-denied cause on this host. Original
bytes/session remain unchanged, no temporary writer files remain, and the same
operation ID succeeds after closing the handle. The initial test expected only
sharing-violation errno; it was corrected to accept access-denied as well, not
treated as a production defect. Non-Windows failure injection is not covered.

`go test -race ./internal/okf/application -run
TestObstructedConfigurationPathPreservesSessionAndAllowsRetry -count=1` passes.
The temporary fixture renames its original configuration aside and places an
empty directory at the document path. The real store rejects Save, preserves
the session and obstruction, and does not alter the preserved original bytes.
After restoring the original file, the same operation ID succeeds. This tests
filesystem read-path obstruction and retry recovery, not mid-replace failure.

Live profile persistence/conflict journey, isolated safety fixture on port 49481:
seeded a valid v1 project document with a bound project profile, separate layered
DOWN architecture layout, and unknown root/profile extension fields. Browser
form changed name and added `frontmatter.custom` with label Custom/max length 30.
Save closed the editor and rendered `Custom: retained`; disk inspection confirmed
architecture layout, binding, and both extension values preserved. Two tabs then
edited the same saved revision. First saved `First editor wins`; second Save
kept its dialog open with recovery guidance and `Unsaved second editor` retained
in Advanced JSON. Disk still contained the first name. After Cancel/Refresh,
the second tab reopened the first name, manually reapplied the retained name
edit, and saved successfully while keeping the graph field. No automatic merge
was used. Final disk inspection retained one profile and both extension values.
Tabs/server stopped. Initial fixture startup rejected a missing root schema;
the fixture was corrected before this journey, not production code.

Live SC-012 isolated project at
`C:/Users/buffo/AppData/Local/Temp/archview-okf-safety-b998a47f358d409d95336c9a36227b7d`,
server port 47536. Root Markdown included raw script and image/onerror HTML,
javascript/data links, an escaping `../outside.md` link to an existing file,
a remote image, safe local/external links, and CommonMark heading/bold/code.
Inspection produced zero script/img/iframe/object/embed elements and neither
execution sentinel appeared on body. Only safe local/external anchors remained;
external had `_blank` plus `noopener noreferrer`. Unsafe/escaping links were
inert text and diagnosed. Local target click selected `target`, loaded its detail,
kept the two-node graph, and did not navigate out of the viewer. Unknown state
displayed explicitly in detail for these stateless concepts. No external link
was opened, and this does not prove every malformed-input variant. Temporary
server/tab stopped; fixture retained for further isolated review. Repository
OKF source was not edited by fixture setup.

Live graph/detail journey, current build on port 53757: Fog at depth 2 starts
with 17 nodes and no semantic overlays. Keyboard selection of Go analysis draws
3 incident semantic edges; selecting code quality replaces those with 13 edges
incident to that concept. Both selections preserve every captured node transform
and Top level breadcrumbs. Go detail displays implemented/implemented; code
quality displays specified/specified and its three containment children. Full
shows 20 nodes/0 hidden. Returning to depth 1 gives 7/13, root plus six direct
children; keyboard ArrowUp restores depth 2 and 17/3, and ArrowDown returns 7/13.
Fog and bundle remain selected. An initial automation fill did not commit a
depth change, so keyboard steps were used to verify the actual control events.
No persistence commands were used; temporary tab and server were closed.
Unknown-state detail and safety-budget browser variants remain unverified.

`go test -race ./internal/viewer -run
TestOKFHTTPDetailMatchesProjectionRollupState -count=1` passes for agreeing,
mixed, and unmapped child states. Assertions verify root presence, roll-up
outcomes, raw declared source state, and projection/detail parity for each
visible node. This is HTTP evidence, not live browser detail verification.

Live current-build profile round trip, local server port 53298: Neutral initially
had 17 concepts and zero node fields. Selecting Fog through the profile combobox
kept the same ordered concept IDs and displayed 16 state fields. Computed SVG
styles were root fill rgb(67,56,202), implemented fill rgb(217,244,223), and the
specified roll-up fill rgb(255,240,194) with rgb(244,114,182) 3px stroke. Selecting
Neutral restored exact captured node inner markup and transforms, with no state
fields. The shared Back to architecture viewer link loaded the architecture
summary, layered graph, quality controls, and reciprocal OKF link. Only session
selection/navigation was exercised, with no Save. Review tab and server closed.
This is browser execution evidence, not human visual approval.

Conflict-guidance follow-up: source inspection confirms Cancel clears the draft
and Refresh is outside the modal. Save/configuration errors now share explicit
copy-before-Cancel, Refresh, reopen, and merge instructions. The API/editor test
asserts these instructions as well as preservation. All 34 browser test files
pass. No live dialog interaction has been claimed for this follow-up.

`node --test internal/viewer/web/app/okf_profile_conflict_test.js
internal/viewer/web/app/okf_profile_save_test.js` passes. The new test joins the
real API envelope decoder to the editor Save handler with a simulated HTTP 409.
It verifies draft/Advanced JSON/layout/revision preservation, displayed failure,
and no automatic retry or refresh. Save As succeeds after an explicitly supplied
fresh revision, retaining extension fields. That revision update is supplied by
the test, not a browser control: live refresh-and-recovery remains unverified.

`go test -race ./internal/viewer -run
'TestOKFHTTPRejectsStaleEditorAndReplaysCreation|TestOKFHTTPProfileSaveAsSaveAndBuiltinProtection'
-count=1` passes. SC-018 now has HTTP boundary evidence; SC-015's existing
HTTP assertions were re-inspected and executed. Neither substitutes for the
browser lifecycle and conflict-recovery journey.

`go test -race ./internal/viewer -run
TestOKFHTTPProfileSwitchPreservesSourceAndRestoresNeutral -count=1` passes for
the SC-005 HTTP profile round trip.

`go test -race ./internal/viewer -run
TestOKFHTTPDepthSequencePreservesBundleAndProfile -count=1` passes for SC-009
and the depth-frontier subset of SC-007.

`go test -race ./internal/viewer -run
'TestOKFHTTPSelectsIndependentValidBundlesWithoutSourceMutation' -count=1`
passes. This executes SC-001 backend assertions and the unbound-neutral subset
of SC-004. It does not execute or certify the whole matrix.

`go test -race ./internal/okf/profile -run
TestSC013RegisteredRuleCompositionIsOrderIndependent -count=1` passes. This proves
the SC-013 registry behavior listed above, not its HTTP or browser surfaces.

`go test -race ./internal/okf/application -run
TestConfigurationMutationSupersedesInFlightProjection -count=1` passes for Save
and Delete. It checks the public application response and session after releasing
the stale work, including profile, rendered label, focus, Back, and revision.

The stateful interaction table remains open across HTTP/browser surfaces and
failure variants. Human visual-review gates remain open.

`go test -race ./internal/okf/application -run
TestSC014StaleBoundProfilePreservesDeclarationAndProducesFallback -count=1`
passes for missing base, missing rule, and missing version.

`go test -race ./internal/okf/application -run
'TestFailedProfileSavePreservesInFlightProjectionAndConfiguration|TestConfigurationMutationSupersedesInFlightProjection'
-count=1` passes. This adds injected failed-persistence coverage at the application
boundary; it does not claim a real disk-failure or browser journey.
