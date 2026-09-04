# OKF uncommitted-change review

Status: complete; ready to commit after explicit user approval on 2026-09-04.

## Target and intent

Target: `git diff HEAD` plus `git ls-files --others --exclude-standard`.
The target includes the whole uncommitted OKF implementation, shared viewer
integration, configuration persistence, and their specification/delivery records.
Owner: Configurable OKF knowledge views, issues 064–071, plus shared viewer concerns.
Intent sources: approved implementation plans in the task, pending issues 064–071,
`.okf/capabilities/okf-knowledge-views.md`, and its canonical reference set.
The later user decisions (architecture-first entry, shared renderer/settings,
selection-only semantic links, and removal of the 100% Fit cap) take precedence.

## Final review disposition

No recorded P0–P2 findings remain. The chronological entries below retain older
findings and pending-review statements as history; this disposition supersedes them.

- The recorded automated/browser gaps in SC-001–SC-020 are resolved.
- The user explicitly approved the completed result: "Everything is fine now.
  I approve. proceed to commits (one or more )". Issues 065–071's human gates
  are satisfied. Issue 064 has no visual gate.
- The final CSS/SVG follow-up and passing repository gates are recorded in
  `20260904-shared-css-svg-followup.md`. Architecture delta: the shared viewer
  and focused application boundaries are deepened, with separate source and
  configuration ownership preserved. No unresolved architecture debt is introduced
  by the reviewed delivery.

## Findings addressed in the current remediation pass

- SC-019 live supersession: port 49930's test-only gate held a real focus
  projection after double-click. While entered=true/status=0, switching to
  Neutral completed and showed the top-level two-node graph. Releasing focus
  returned HTTP 409 okf_operation_superseded with request ID
  req_54cf0caf94db4c841f45994e. DOM-backed snapshots before/after the late response
  match exactly for profile, status, breadcrumbs, Back enabled state, details,
  node identities and transforms. The fixture now scopes its existing pause
  rule to focus requests so the preceding click's detail lookup cannot consume
  the gate. The earlier port 47108 run paused detail and is not projection
  evidence. Architecture delta: neutral, opt-in test fixture only, no production
  API or behavior change. Human visual review remains open.
  Final Go tests, race tests, vet, build, all 40 browser tests and diff check
  pass. All 257 changed/new OKF/viewer code files remain below 600 lines. The
  live fixture and browser tab are closed. No commit or issue closeout occurred.

- Detail-renderer final failure/size audit: new interaction tests verify
  cancellation before invocation skips the renderer; cancellation inside the
  callback discards even successfully returned output, produces the canonical
  cancellation error and returns no partial detail. A subsequent ordinary
  detail request succeeds. Oversized custom Markdown containing multibyte
  characters is bounded by the existing display-input limit without splitting
  UTF-8, excludes the tail, preserves source/raw Markdown/frontmatter, and
  reports a bundle/concept-scoped truncation warning. Three race repetitions
  pass. This completes the recorded extension verification gap together with
  the prior registration, ownership, HTTP and browser evidence. Architecture
  delta: neutral, tests exercise the existing renderer/security pipeline; no
  production code or CSS changed. Human visual approval remains separate.
  Full Go tests, race tests, vet, build, all 40 browser tests and diff check
  pass. The new test file is 76 lines.

- P2 real worker failure: the bundled ELK promise stayed pending on worker
  load errors, leaving the browser without a failure warning. The prior mocked
  rejected-layout test did not cover this. Shared elk_runtime.js now observes
  native worker error/messageerror events, rejects the request and terminates
  the worker. Architecture and OKF both use this runtime; duplicate construction
  and cleanup were removed. Tests cover error/messageerror, success, constructor
  failure and listener cleanup. Live isolated port 50528 proves failed relayout
  retains exact root/target positions, failed subtree focus produces an
  inspectable fallback, and worker repair plus Back restores the normal graph
  and clears the warning. Architecture delta: deepens the shared renderer
  runtime; no new settings or production endpoints. The opt-in Go test fixture
  supplies failure controls only in its localhost test process.

- Live extension rendering on the same fixture: project:custom gives both nodes
  the Provider title label, ellipse geometry and rgb(17,34,51) fill. Inspection
  renders the Custom view heading, retains original raw Markdown/custom metadata
  and has no script or javascript anchor in its DOM. project:panic restores
  Original content with okf_detail_renderer_failed. Neutral restores Source root
  and Source target labels and rectangular geometry. This resolves the custom
  provider browser gap but does not replace the required human visual review.
  Full Go tests, race tests, vet and build pass. All 40 browser tests, browser
  syntax checks and diff check pass; all 256 changed/new OKF/viewer code files
  remain below 600 lines. Live fixture servers and their browser tabs are closed.

- SC-011 remaining live variants: isolated port 54594 verifies Neutral at depth
  1 with Full off and Fog at depth 1 with Full on. Each focus/Top level/Back
  sequence retains an unsaved Layered draft, active profile, depth and Full;
  visible counts are 1/2/1 and Back restores Top level/root/target breadcrumbs.
  Native ArrowDown/Tab commits the numeric control; an earlier programmatic
  fill did not commit and is not counted as depth-one evidence. Combined with
  prior project-profile/depth-2 and Neutral/depth-2 journeys and HTTP variants,
  the SC-011 execution gap is resolved. Human review remains open. Configuration
  SHA-256 remains B711477F76C7E4591FB6CB02EAF5F1CAD8B8C5719CAB5ECAD02A28E528D775BC.

- P2 application-service separation: Service now composes catalogService,
  sessionService, inspectionService, profileService and diagnosticService.
  Use cases moved to their owning receivers; catalog, inspection and synchronized
  state have dedicated files. The facade keeps only construction/dependency
  setup and the existing promoted API. Shared publication/write locks remain
  intact, preserving atomic refresh, persistence and session invalidation.
  Architecture delta: deepens. Owner: Configurable OKF knowledge views.
  Boundary: focused application use cases, excluding transport/source editing.
  Evidence: viewer/okf_composition.go still constructs application.API without
  change; existing navigation, persistence and HTTP tests pass. Direct component
  tests cover lazy initialization, profile publication, inspection and refresh,
  and reject unrelated operation interfaces on navigation/profile/inspection
  components. Three focused concurrency-test repetitions pass. This resolves
  the current-diff separation finding; it is not a whole-repository architecture
  health claim or an acceptance/visual gate closeout.
  Full Go tests, race tests, vet, build, all 39 browser tests and diff check
  pass. All 253 new/modified OKF/viewer code files remain below 600 lines.

- P2 provider result ownership: a new regression confirmed presentation
  annotations and diagnostic details retained provider-owned typed Go maps
  and slices. Mutating those containers changed an already-returned result.
  Shared captureExtensionValue now normalizes both outputs and registration
  metadata into owned JSON containers. UseNumber preserves integers beyond
  float64's exact range. Both failing cases pass after remediation; the full
  profile race suite passes three repetitions. Architecture delta: deepens the
  existing extension boundary with one shared capture policy, no new API and
  no renderer-specific cloning implementation. Extending the regression to
  direct RuleStrategy results also failed. Capture now belongs to the shared
  evaluateRule path, removing the presentation-only implementation. An extra
  check verifies visibility pointers are detached and unencodable output
  discards the entire result. Profile/application/HTTP race suites pass.
  Final full Go tests, race tests, vet and build pass. All 39 browser tests and
  diff check pass. The 248-file audit finds no OKF/viewer code file at 600 lines.

- P2 detail-renderer descriptor validation/ownership: registration accepted
  missing descriptions or schemas, and a schema containing typed Go maps
  could remain aliased to the provider. The new registration regression failed
  on incomplete metadata before remediation. All three extension registrations
  now reuse captureExtensionMetadata to normalize independently owned JSON;
  detail registration requires description and parameter schema before publish.
  Three focused race repetitions pass. Architecture delta: deepens the existing
  registration boundary, removes duplicate metadata encoding and adds no API.
  Diagnostic failure tests also prove partial output is discarded on provider
  errors, invalid severity, missing code or unencodable values; healthy later
  providers continue. Cancellation after a callback discards the report, skips
  later callbacks, and does not poison a subsequent request.
  Full Go tests, race tests, vet, build, all 39 browser tests and diff check
  pass. All 246 new/modified OKF/viewer code files remain below 600 lines.
  Remaining ownership review must include typed Go containers in provider
  result annotations/details, not just the now-captured registration metadata.

- P2 diagnostic snapshot consistency: report assembly previously read profile,
  catalog and source state separately and could mix bundle revisions during a
  concurrent Refresh. It now captures catalog, immutable index references and
  a configuration-specific registry together under the read lock. The focused
  diagnosticReporter assembles the report outside that lock; profile catalog
  resolution is shared with Profiles through profileCatalogBuilder. A paused
  provider regression refreshes the second bundle while the first is being
  reported and asserts both diagnostic revisions match the report's original
  catalog revision. Three focused race repetitions pass. Architecture delta:
  deepens the report boundary and removes assembly from Service without adding
  a transport API or duplicating profile resolution. Configuration/session
  orchestration separation remains an independent open finding.
  Full Go tests, race tests, vet and build pass; all 39 browser tests and syntax
  checks of 80 scripts pass. Diff check passes. All 244 changed/new OKF/viewer
  code files remain below 600 lines. No CSS or persisted configuration changed.

- SC-011 live navigation on isolated fixture port 50853: changed the session
  layout from saved Mr. Tree to Layered, double-clicked Local target, returned
  to Top level and pressed Back. Reopened layout settings at each step showed
  Layered. Counts changed from 2 visible concepts to 1, then 2, then 1;
  breadcrumbs restored Top level/root/target on Back. No profile Save was used.
  This verifies the project-profile, depth-2, non-Full journey only.

- PresentationPropertyProvider HTTP-to-scene coverage: Save As persists the
  registered provider invocation; selecting that profile produces its label,
  declared color token, registered ellipse geometry and annotation in the
  projection. Switching to Neutral removes custom presentation and restores the
  original title. Concept identity and source detail remain unchanged. Three
  race repetitions pass. Architecture delta: neutral test-only verification
  through the existing injected application boundary and profile endpoints.
  This does not claim live browser rendering or human approval.
  Full Go tests, race tests, vet, build, all 39 browser tests and diff check pass.
  The new integration test is 70 lines; no production code or CSS changed.

- PresentationPropertyProvider implementation: registered display-only providers
  use existing rule invocations and composition rather than a second pipeline.
  The adapter contributes label/token/shape/annotations, validates encodability,
  and exposes captured presentation_property metadata. Existing priority,
  parameter-validation, cloning and panic boundaries remain authoritative.
  Tests cover unequal/equal-priority composition, source/parameter isolation,
  candidate retention, catalog kind, duplicate rejection, invalid parameters and
  panic recovery. Profile/application/viewer race suites pass. Architecture delta:
  deepens the existing rule extension boundary with a focused display-only port;
  no visibility/state mutation surface is exposed to presentation providers.
  Projection/HTTP rendering evidence and further validation review remain open.
  Removed the newly added duplicate profile identity helper in favor of existing
  domain.ValidExtensionIdentity; no user-authored file was removed. Metadata
  copying is shared across the new provider catalogs.
  Full Go tests, race tests, vet, build, all 39 browser tests and diff check pass.
  Line audit covers all 241 new/modified OKF/viewer Go/JS/CSS files; none reaches
  600 lines. The registry is 578 lines and must not grow beyond the agreed limit.

- P2 combined diagnostic output bound: the HTTP regression produced 251 items
  from a registered provider. The query now applies the existing 200-item helper
  after filtering, preserving the first 199 plus a truncation warning. Both data
  and envelope diagnostics are bounded. A filter targeting item 251 still returns
  it with recovery guidance, proving filtering precedes truncation. The test
  fails before the one-line production change and passes afterward; three
  focused application/HTTP race repetitions pass. Architecture delta: neutral,
  reuse of the existing application output policy without new configuration.
  This does not bound provider allocations or execution time.
  Full Go tests, race tests, vet, build, all 39 browser tests and diff check pass.
  Changed code files are 77 and 63 lines. No new output-limit setting was added.

- DiagnosticProvider HTTP integration: a registered provider survives Save As,
  appears in the extension catalog, and contributes one filtered diagnostic with
  preserved recovery/source-revision details. Unfiltered reports retain built-in
  boundary warnings. Published session, indexed metadata and source bytes remain
  unchanged. Three HTTP race repetitions pass. Registration identity validation
  is now shared between detail and diagnostic providers, rejecting empty namespace
  segments, whitespace/control characters and ambiguous @ identities. Focused
  profile/application/viewer race suites pass. Architecture delta: neutral,
  existing injection/query boundaries and one shared pure validation function.
  Full Go tests, race tests, vet, build, all 39 browser tests and diff check pass.
  Changed code files remain below 600 lines, largest 124. Diagnostic query
  output bounding still needs review: the existing 200-item helper is not yet
  applied to the combined provider/query result.

- DiagnosticProvider implementation: host-registered providers contribute to
  the diagnostic query from cloned bundle indexes, before normal filtering.
  Metadata is captured during registration and appears in the extension catalog
  and registry revision; configuration candidate registries preserve providers.
  Duplicate/invalid registration rejects. Execution order is stable by ID/version;
  invalid output and panics become scoped warnings, cancellation aborts, and
  built-in diagnostics remain intact. Tests verify source isolation, copied
  metadata, registry revision, candidate retention, output validation, panic
  recovery and cancellation. Profile/application/viewer race suites pass.
  Architecture delta: deepens the diagnostic query extension boundary; transport
  remains unaware of provider execution. Public application/HTTP provider tests,
  further review and full repository verification remain open for this addition.

- P2 diagnostic filter gap: canonical GetDiagnostics requires project, profile,
  concept, relationship, operation, severity and category filters in addition to
  bundle. The endpoint previously ignored these parameters. A typed application
  query now implements their intersection; HTTP only maps query parameters.
  Diagnostics can carry an optional relationship ID. Tests cover each mismatch,
  matching intersection, unscoped bundle-warning compatibility and complete HTTP
  query mapping with retained recovery guidance. Three focused race repetitions
  pass. Architecture delta: neutral extension of the owning diagnostic query,
  no additional workflow for the transport and no CSS changes. Provider-supplied
  diagnostics and live browser verification remain separate open work.
  Full Go tests, race tests, vet, build, all 39 browser tests and diff check pass.
  All seven changed code files remain below 600 lines, largest 565.

- Diagnostic query ownership: the HTTP handler previously coordinated profile,
  catalog and per-bundle summary reads and performed bundle filtering itself.
  Diagnostics now has a focused application operation returning the same report
  payload and revision. The handler parses the query and writes the envelope.
  Application tests prove published-source semantics, no session mutation,
  bundle filtering, independent result ownership and cancellation; existing HTTP
  preservation coverage also passes three race repetitions. Architecture delta:
  deepens the OKF application boundary by hiding query orchestration from its
  transport client. DiagnosticOperations is separate from unrelated operation
  groups. This addresses transport ownership only, not the pending provider seam
  or broader configuration/session separation. Changed files are 51, 54, 56 and
  561 lines; no CSS change. Diff check passes.
  Full Go tests, race tests, vet, build and all 39 browser test files pass after
  this extraction. The wider GetDiagnostics filter vocabulary still requires
  review against canonical-use-cases.md; this pass preserves the bundle filter.

- Custom detail HTTP boundary: a registered test renderer is selected through
  Save As and session profile endpoints. The detail envelope contains sanitized
  custom output and request identity, preserving raw Markdown and frontmatter.
  A panicking provider yields original formatted Markdown plus a structured
  warning. Saving an unavailable renderer version rejects with its diagnostic
  and preserves exact configuration bytes; source bytes also remain unchanged.
  Three race repetitions pass. A stricter assertion found missing bundle scope
  on runtime renderer warnings; detail assembly now attaches the bundle ID and
  the warning provides recovery guidance. The regression fails before that fix
  and passes afterward. Architecture delta:
  neutral test-only coverage using the existing injected application boundary;
  no production dependency or CSS added. Live browser review remains open.
  Post-fix full Go tests, race tests, vet, build and diff check pass. Changed
  code files are 114, 130 and 39 lines.

- Detail presentation lifecycle/form follow-up: application regression registers
  a custom renderer and verifies Save As, rename, binding, fresh-service reload,
  custom display, unchanged source bytes and Neutral default restoration. Three
  race repetitions pass. The editor now exposes renderer ID/version and extension
  parameters, sharing the existing catalog request and CSS. Renderer replacement
  replaces parameters atomically rather than merging another renderer's settings.
  Full repository race tests also pass. Form tests cover serialization, unsafe label escaping, invalid parameters,
  required version and clearing selection; cached catalog tests cover both shapes
  and renderer suggestions. All 39 browser test files and full Go tests pass;
  vet/build and changed-JS syntax checks pass. Files changed in this pass are
  below 600 lines, largest 210. This is not live browser/human review evidence.

- DetailRenderer backend implementation: optional details.renderer carries ID,
  version and parameters through JSON, cloning, inheritance and candidate
  validation. Registered metadata appears in the extension catalog and registry
  revision. The default remains existing CommonMark. Custom output passes through
  the existing sanitizer and display limit; providers receive cloned source and
  parameters, while raw detail/metadata remain original. Provider failure falls
  back with a warning. Focused profile/interaction tests and all OKF/viewer race
  suites pass. Form selection, custom-provider application/HTTP lifecycle tests
  and full verification after this change remain open. Architecture delta:
  deepens the existing detail assembly boundary with a focused renderer port;
  transport still depends on the application API and never executes renderer code.

- SC-011 navigation HTTP follow-up: the real handler now has regression coverage
  for Neutral and Fog with full mode on/off at depth 1. Top level preserves
  bundle/profile/depth/full; repeating it adds no history; Back restores the
  focused child and then the preceding top-level view. Empty Back returns 409
  okf_navigation_history_empty. Three race repetitions pass. The canonical HTTP
  contract now documents the top endpoint and its history semantics. This does
  not claim browser layout-draft preservation has been newly verified.
  Architecture delta: neutral, test-only verification of the existing session
  navigation boundary through its HTTP adapter. No new production dependency.
  After this addition, full Go tests, race tests, vet, build, all 38 browser
  test files and diff check pass. The new HTTP test file is 62 lines. These
  results do not close the extension, architecture or human-review findings.

- Fit follow-up: removed the remaining 145% Fit-specific ceiling from architecture
  and normal-mode OKF. Both callers retain their existing general zoom safety
  limits and shared fit calculation. Regression tests exercise both canvas modes
  and assert OKF centering and available-width fit including existing padding.
  Focused Fit/math tests, changed-JS syntax checks and diff check pass. This is
  not a new full repository verification pass or human visual approval.

- SC-017 post-failure usability, isolated viewer port 46250: replacement lock
  forces Save to report Access is denied with editor still open. Cancel then
  selecting Safety fixture loads its detail under project:review; the original
  Custom field and both nodes remain. Reopening the editor shows the previously
  saved name, not the rejected draft. Configuration SHA-256 exactly matches
  pre-test baseline and zero writer temporary files remain. Lock helper released,
  tab closed, server stopped. No fixture configuration edits persisted. This
  fills the remaining post-failure browser assertion; human review stays open.

- SC-019 HTTP concurrency evidence: a registered rule pauses one focus request
  through the real viewer handler. Cancelling it yields 409
  okf_operation_cancelled; completing a newer depth request first yields 409
  okf_operation_superseded for the old request. Both envelopes include request
  IDs. A subsequent HTTP projection retains expected depth, bundle, profile,
  root node and unchanged Back history. Five race repetitions pass. The
  116-line test uses a bounded wait and a buffered release channel; no production
  callback or interface added. Architecture delta: neutral existing HTTP/rule
  boundary verification. Live browser layout-failure evidence remains open.
  Full Go tests, race tests, vet, build, all 38 browser tests and diff check pass.

- SC-007 hidden-target/Neutral live check, temporary viewer port 48961: added
  an isolated grandchild and a semantic reference from area to that grandchild.
  At depth 2, six nodes/four containment edges render; selecting area adds only
  semantic_link area-to-leaf, preserving every node transform and breadcrumb.
  At depth 1, five nodes/three containment edges exclude the grandchild; selecting
  area neither reveals it nor draws its edge, again preserving exact transforms
  and breadcrumbs. Neutral remains selected. Combined with existing Fog evidence,
  the missing profile/hidden-target checks are covered. Temporary link/concept
  removed, tab closed and viewer stopped. Human visual approval remains open.

- SC-012 remaining CommonMark assertions: expanded the interaction test to
  require heading, strong/emphasis, inline code and both unordered-list items,
  not merely nonempty sanitized output. Five race repetitions pass. Live
  temporary viewer port 51007 renders Emphasized content in em and two review
  items in a ul, alongside the existing heading/strong/code. The HTTPS anchor
  retains _blank/noopener/noreferrer; no script/img/iframe/object/embed nodes
  appear. Existing local-link and malicious-link journeys remain recorded.
  Canonical SC-012 says HTTPS is offered safely, not that external navigation
  must be performed. Fixture additions removed; original SHA-256 restored.
  Browser closed, server stopped; no production code or source edits retained.

- SC-018 evidence audit: removed an invented live same-ID replay-control gate.
  The editor exposes new commands, not replay of prior operation IDs. Added a
  frontend API assertion that explicit Save As retransmissions preserve exact
  body/header identity; five HTTP race repetitions verify original-result replay,
  changed-input conflict and unchanged saved state. Existing live two-editor
  conflict/recovery evidence covers the supported UI action. No production
  behavior changed or new retry surface introduced. Human review remains open.

- SC-020 live unreadable-source recovery, isolated viewer port 48583: selected
  Safety fixture detail, held its source open with FileShare.None, and clicked
  Refresh. The UI disabled .okf as unreadable, cleared graph/detail/breadcrumbs,
  reported zero concepts, and exposed root.md's Windows sharing error plus
  recovery guidance. second/.okf remained selectable with only Independent
  bundle. After explicit handle release, Refresh and .okf reselection restored
  both original concepts and project:review; unreadable/stale-binding warnings
  disappeared while unrelated source warnings remained. SHA-256 before/after
  matches BB5A906078799FF4BEA59DEF8E005D6C43A24157AA94CFBA12B702EB194AB370.
  Helper exited, browser closed, temporary viewer stopped. No source bytes or
  project configuration edited. Human visual-review approval remains separate.

- SC-004 live Neutral baseline, isolated viewer port 54932: selecting unbound
  second/.okf selects builtin:neutral and renders only Independent bundle with
  zero node fields. Expanded report retains invalid/.okf's missing-frontmatter
  diagnostic, affected bad.md and recovery guidance. Editor shows builtin:neutral
  and disabled Save/Rename/Delete. Cancel then node selection loads Independent
  bundle detail without changing profile or hiding the report. No persistence
  action was used. Review tab closed and temporary server stopped. This completes
  the previously missing combined browser assertion, not human visual approval.

- SC-020 unreadable-source backend evidence: a Windows-only public application
  test opens an isolated concept with zero file sharing. The real scanner marks
  that bundle unreadable/nonselectable with okf_bundle_unreadable; Session does
  not reuse its old snapshot. Selecting the independent bundle still works.
  After handle closure and Refresh, the original bundle renders again with the
  same source identity/revision and byte-for-byte content. Five race repetitions
  pass. The 85-line test uses t.TempDir and deferred handle cleanup, no mocked
  scanner, ACL changes, production edits or repository-source mutation. Existing
  frontend reset coverage was re-inspected; a live unreadable-source browser
  journey is not claimed. Architecture delta: neutral test-only boundary check.
  Full Go tests, race tests, vet, build, all 38 browser tests and diff check pass.

- Diagnostic recovery frontend follow-up: Save/Save As, Rename, Delete and Bind
  prepare both profile and bundle catalogs through loadConfigurationCatalogs.
  Promise.all returns one value only after both reads succeed; existing editor
  guards run before publication. Successful Save therefore replaces the stale
  catalog consumed by renderState before selecting/rendering the saved profile.
  Added Save assertions for repaired warnings with retained invalid-source
  diagnostics, delayed responses after editor replacement, and a failed catalog
  read preserving both old catalogs while reporting saved-but-refresh-failed.
  Existing conflict test still proves no reads/retries after rejected writes.
  Architecture delta: deepens shared metadata preparation in the existing
  profile lifecycle module; callers retain publication and command policy.
  No new endpoint, module, CSS or source rescan. All 38 browser tests pass.
  This is automated frontend evidence, not a live visual-review approval.
  Full Go tests, race tests, vet, build, changed-JS syntax and diff checks pass.
  All seven changed code files remain below 600 lines, largest 462.

- P2 partially fixed, configuration catalog publication: creating a missing
  bound profile left okf_profile_not_found in bundle/profile catalogs until a
  full source Refresh. Bind also left DefaultBundleID stale. A public application
  regression reproduced the former failure. Successful writes now recompute
  configuration-derived diagnostics and the default bundle under the publication
  lock, preserving source diagnostics separately. Removed redundant configError
  storage; Profiles reads the canonical catalog diagnostics. The test verifies
  repair, retained invalid-candidate diagnostics and a changed default bundle
  without rescanning. Five race repetitions pass. Architecture delta: neutral
  catalog publication correction, no new port or persistence responsibility.
  Browser cached-catalog synchronization remains explicitly open above.
  Modified files are 376, 252 and 57 lines; no CSS changes.
  Full Go tests, race tests, vet, build, all 38 browser tests and diff check pass.

- Architecture remediation, configuration persistence ownership:
  configurationWriter now owns candidate validation, a private configuration
  copy, expected-revision fallback, profile revision preparation and the store
  write. Its sole caller, Service.saveConfiguration, retains locking and
  publishes sessions/registry/journal only after success. The writer cannot
  reach sessions or application publication state. Owner: OKF configuration
  lifecycle. Architecture delta: deepens this boundary without a new public
  port or changing the existing ConfigurationStore contract. This is partial
  remediation, not closure of the broader session/lifecycle separation finding.
  Existing public application and HTTP race suites pass. Five repeated race
  runs cover caller ownership, failed-save pending navigation, successful-write
  supersession, real Windows replacement failure, obstruction recovery, and
  exact retry input. Added an assertion that Save, as well as Save As, leaves
  the caller profile unchanged. Changed files are 30, 250 and 87 lines; no CSS.
  Full Go tests, race tests, vet, build, all 38 browser tests and diff check pass
  after extraction. The added ownership assertion also passes five race runs.

- P2 fixed, configuration operation identity: Save/Save As, Rename, Delete,
  and Bind used millisecond timestamps for idempotency keys. Independent tabs
  issuing the same command at the same timestamp could collide in the server's
  operation journal, producing an unrelated idempotency conflict. All four
  generators now use browser crypto.randomUUID with their existing prefixes.
  A frozen-clock editor test reproduced identical Save IDs before the fix and
  now verifies distinct UUID-format IDs. All 38 browser tests pass, including
  existing API envelope, lifecycle and stale-write cases. No timestamp-based
  configuration IDs remain. No new interface, dependency, CSS or file was added.
  Architecture delta: neutral. Owner is OKF profile configuration lifecycle;
  handlers still own command identity, API adapters still forward it, and the
  server journal/revision contract is unchanged. This does not add automatic
  retries or claim live cross-tab collision testing.
  Post-fix full Go tests, race tests, vet, build, JavaScript syntax checks and
  diff check pass. Modified code files remain below 600 lines, largest 458.

- Verification refresh after the persistence-evidence audit: full
  `go test ./... -count=1`, `go test -race ./...`, `go vet ./...`, and
  `go build ./...` pass. All 38 browser test entries pass; syntax checks pass
  for 57 changed JavaScript files. All 217 new/modified OKF/viewer Go, JS and
  CSS files are below 600 lines, largest 582. Normal `git diff --check` passes.
  Race output includes valid cached package results. These gates do not resolve
  the specification, architecture, or human-review items listed above.

- SC-017 real browser persistence failure on port 49331: Windows replacement
  denial retained the open unsaved editor and previous graph/profile while
  displaying the filesystem cause. Follow-up SHA-256 equals the original
  fixture hash; no writer temporary files remain. Lock helper and browser tab
  are gone, and the remaining server was stopped. Post-failure selection and
  live retry were not observed and are not claimed. Matrix now also includes
  previously recorded SC-009/010 relationship-budget evidence from port 54789.
  Its verification convention follows canonical when-supported coverage rather
  than inventing mandatory additional operating-system permutations.

- SC-013 applied composition journey on temporary viewer port 52545: three
  registered metadata rules validate successfully; compatible annotations a/b/
  shared merge, High priority beats Low priority, and equal-priority left/right
  roles produce an empty/default role plus a root-linked field=role diagnostic.
  Browser shows the high-priority label and expanded conflict report while the
  other concept remains unchanged. HTTP projection confirms the merged values
  and default role. Repairing the second role and Refresh removes the warning
  without changing the winning label. Configuration hash unchanged by validation
  and projection; temporary rules removed and baseline hash restored exactly.
  Browser closed, server stopped. Added HTTP regression validates, creates,
  selects, repairs and reprojects the same composition; five race repetitions
  pass. Its Save As expectation was corrected to canonical 201, not product
  behavior changed. Existing registration-order test remains the evidence for
  order independence. Full profile/viewer race tests and diff check pass.

- SC-003 complete source-facts journey, temporary viewer port 52252: Neutral
  detail shows type/title/description/path, expandable unknown metadata preserving
  zero, false, null, Unicode arrays and literal script-like text; Raw Markdown
  retains the original body. Rendered CommonMark has heading/bold/code and safe
  links with no script/image/handler nodes or execution sentinels. Expanded link
  outcomes identify the resolved local target and fragment; clicking that link
  selects Local target and its source-backed detail. Source SHA-256 unchanged
  after inspection; temporary metadata additions removed afterward and baseline
  hash restored exactly. Browser closed and server stopped. Added HTTP test
  covers the same generic overview, nested unknown values, raw/safe Markdown,
  resolved local link, source identity and unchanged source bytes; five race
  repetitions pass. Human visual review remains open.

- SC-014 stale-profile browser journeys on port 54078: missing review.missing
  rule and unsupported okf.rule.visibility version 999 each preserve
  project:review, both concepts and the configured Custom field, while reporting
  the skipped rule/version. Removing the stale rule and Refresh removes the
  warning. Unsupported section schema arch-view.okf/v999 falls back to Neutral
  with a configuration diagnostic; repairing v1 restores the bound profile and
  Custom field. SHA-256 checks before/after each faulty load prove no rewrite;
  final fixture hash exactly matches its original baseline. Browser closed and
  temporary server stopped. Earlier missing-base recovery is recorded below.
  Found and fixed misleading load diagnostic: unsupported schema was described
  as invalid JSON. Load now reports invalid configuration with the precise
  underlying cause. New schema recovery test failed on that wording before the
  fix, then passed; it also verifies Save rejects the unsupported schema even
  with its exact revision, preserves bytes, and Load recovers after repair.
  Configuration/application/viewer race tests pass. Live journey preceded the
  wording correction; corrected wording is verified by the store test.

- SC-006 cross-surface hierarchy evidence: isolated hierarchy/.okf bundle on
  temporary viewer port 47048 contains five concepts with filesystem fallback,
  explicit-parent override, and an ambiguous two-parent claim. Browser shows
  all five concepts but only the two valid containment edges. Expanded report
  names area/topic/detail and recovery guidance; selecting that concept exposes
  its source body and empty containment parent. HTTP projection confirms
  filesystem_fallback and explicit_parent provenance; summary retains all five
  indexed IDs. Repairing the ambiguous claim and Refresh adds only the intended
  other-to-detail edge, removes the hierarchy warning, preserves Neutral and the
  unrelated invalid-bundle diagnostic. Selecting second/.okf still renders only
  Independent bundle. Added HTTP regression reproduces both stages, checks exact
  IDs/edges/provenance/diagnostic scope and unchanged source bytes; five race
  repetitions pass. Browser closed, server stopped; repaired hierarchy fixture
  retained only in the temporary test directory. Human review remains open.

- Architecture remediation, profile edit ownership: Save and Save As now use
  profileLifecycle for identity policy, source copying, validation and immutable
  candidate edits. The component has no Service, storage or session dependency.
  Individual profile validation moved to configurationValidator and is shared
  by the public validation path and lifecycle editor. Service retains the
  existing configuration mutex, idempotency replay, persisted revision checks
  and session publication; public method signatures and error precedence are
  unchanged. This deepens the candidate-edit boundary rather than merely moving
  Service methods between files. The broader coordinator/session ownership
  finding remains open. Public application regression test verifies Save As
  does not mutate caller input and editing its copy preserves the source and
  opaque extensions; five race repetitions pass. Application/viewer race tests,
  full uncached Go tests, full `go test -race ./...`, vet, build and diff check
  pass. Line audit: all 214
  new/modified OKF/viewer Go/JS/CSS/HTML files are below 600 lines.

- P2 layout failure erased unrelated diagnostics: renderState's layout-error
  callback replaced catalog/projection findings with a single layout warning
  and downgraded an existing error tone. It now appends the layout finding to
  the already assembled diagnostic list and retains error tone when present.
  Existing graph tests verify previous-layout retention, changed-scene fallback,
  and suppression of superseded failures; graph/state tests and bootstrap syntax
  pass. The status callback correction is code-inspected; a live injected-layout
  failure journey remains open under SC-019 and is not claimed by these tests.

- SC-015/016 lifecycle verification completed on rebuilt temporary viewer port
  54450. The previous stalled tab was absent from fresh browser inventory.
  New DOM Delete confirmation focuses Cancel; Cancel and Escape each close only
  that confirmation, restore focus to Delete, and preserve the editor/profile.
  Explicit confirmation deletes only project:browser-renamed, closes editor,
  selects Neutral, and preserves project:review and unrelated configuration.
  Built-in Fog Save As creates editable project:builtin-copy with Fog state
  mapping, rules, decorations, tokens and mrtree layout intact. Binding this
  copy to both independent bundles, renaming it to project:bound-renamed, and
  selecting A/B proves both bindings updated. Confirmed deletion moves both
  bundles to Neutral. Temporary copies removed through the viewer; original
  profile/revision, root/profile extensions, architecture layout and original
  fixture binding restored and checked. Browser closed and server stopped.
  Existing service lifecycle test now covers two bindings through rename,
  rejected deletion without fallback, and successful neutral fallback; five
  race-enabled repetitions pass. Full `go test -race ./...`, `go vet ./...`
  and `go build ./...` also passed before this test-only expansion. Human
  visual-review gates remain open; no issue or capability state changed.

- Delete confirmation remediation: added a renderer-neutral DOM confirmation
  helper using existing layout dialog/button styles, with Cancel as default,
  explicit Delete profile action, literal text for the source ID, and Escape
  cancellation. No CSS or native dialog dependency added. Lifecycle adapter
  rejects absent/builtin drafts, cancelled/failed confirmations, concurrent
  confirmation attempts, and source/draft/JSON/bundle/revision changes while
  awaiting confirmation. Confirmed current deletion retains the existing
  revision-checked server endpoint and shared publication guards. Tests cover
  these behaviors and dialog cleanup. All 38 frontend test files and syntax
  checks pass. Live verification pending; earlier stalled IAB tab 25 remains
  unresolved, with no deletion confirmed. Temporary server 48461 stopped.

- SC-016 live Rename verified on temporary viewer port 48461: selected
  project:browser-copy, edited Profile ID/Name to project:browser-renamed and
  Browser renamed profile, then clicked Rename. Editor closed and selector
  selected the renamed ID. Disk contains the renamed profile and original
  project:review; architecture layout and unknown root extension are unchanged.
  Application/viewer race tests pass. Subsequent Delete click stalled test tab
  25; getJsDialog returned undefined and DOM/close calls timed out. No confirm
  action was taken. User asked to cancel a native dialog if visible. This is
  not successful Delete evidence or completed visual review.

- Rename remediation verified in code: the lifecycle adapter now reads the
  existing Profile ID and Name fields, retaining the original selected ID as
  the server command's source. Native prompts are no longer required. Tests
  make prompt throw, check source/target identity and successful selection,
  reject builtin targets, and verify revision conflicts preserve the draft,
  selected source, revision, and open editor. All 36 frontend test files pass.
  Live Rename verification remains pending. Delete still uses native confirm;
  its compatibility and safe confirmation flow require review separately.
  `go test ./... -count=1`, focused browser syntax checks, and the repository's
  normal `git diff --check` pass. Bootstrap/lifecycle/rename-test line counts
  are 465/60/39, below the 600-line threshold.

- P2 Save As unusable in the in-app browser: live click failed with
  `prompt() is not supported.` No JavaScript dialog existed to accept, and
  the draft remained unsaved. Save As now uses the existing Profile ID field,
  asks for a new project ID when unchanged/builtin, and keeps the original
  selected profile as source identity. No duplicate form or native prompt added.
  Tests explicitly make prompt throw, verify form-based copy and stale-write
  recovery; all 35 frontend tests pass. Rebuilt viewer port 53494 successfully
  creates/selects project:browser-copy, closes editor, and retains project:review.
  Disk inspection verifies unknown profile/root extensions and architecture
  layout preserved. Copy retained only in temporary test fixture for subsequent
  lifecycle verification; browser closed and server stopped. Rename still uses
  native prompts and requires separate remediation before its browser gate.

- SC-015 built-in editor protection verified live on temporary viewer port
  49461: selected builtin:fog-of-war, opened Edit profile, and inspected actual
  button disabled states. Save/Rename/Delete disabled; Save As and Cancel enabled;
  Advanced JSON retains builtin:fog-of-war identity. Cancel closed the editor
  without a write. This verifies UI protection, not completed Save As creation.
  Browser tab closed and test server stopped.

- SC-014 missing-base live recovery, temporary viewer port 51025: bound profile
  with project:missing-review-base opens using Neutral and a named missing-profile
  diagnostic with repair guidance. Disk inspection confirms the project:review
  binding and broken base remain, not silently rewritten. Restoring the base
  and Refresh selects project:review again, restores Custom: retained, and removes
  the missing-profile diagnostic. Other bundle diagnostics persist. Temporary
  configuration restored, browser closed, server stopped. Missing-rule/version
  and schema variants remain separate from this observed missing-base journey.

- SC-009/010 relationship-budget live check, temporary viewer port 54789:
  max_relationships=1 retains both source concepts and one containment edge;
  Full and root selection do not bypass the limit to add the semantic edge.
  Grouped report exposes okf_relationships_truncated. Restoring configuration
  and Refresh removes that diagnostic; selecting root then draws containment
  plus the semantic edge, with two relationships reported. Temporary config
  restored, browser closed, server stopped. Human review remains separate.

- Projection extraction failure audit: new direct builder tests assert cancelled
  requests retain context.Canceled, foreign focus retains concept-not-found, and
  neither error returns a populated snapshot. These tests and the existing
  paused cancellation/supersession publication tests pass five race-enabled
  repetitions. No production defect was found in these paths; the evidence
  specifically covers failure propagation and session-publication isolation.

- Architecture delta: deepens projection construction. projectionBuilder owns
  effective-profile fallback, breadcrumb derivation, and renderer-neutral build
  plus diagnostics. It takes source/profile/navigation inputs and has no session
  map, persistence, or publication access. Service retains generation and context
  checks before publishing; failure/supersession behavior remains covered by the
  application/viewer race suites, which pass. A direct builder test verifies
  Neutral fallback, retained navigation and diagnostics, and input isolation
  without constructing Service. Remaining lifecycle/session orchestration work
  is not declared complete by this extraction.

- Consolidated gates after operationJournal, typed retry identity, and independent
  configuration validation/cancellation changes: full uncached Go tests, Go race
  suite, vet, build, all 35 frontend test files, every viewer JS syntax check,
  OKF validation, and diff checks pass. All 210 changed/new scoped code files
  remain below 600 lines including blanks. Scenario matrix now records completed
  unfamiliar-state and node-budget browser evidence; relationship-limit variants,
  other listed journeys, contract choices, and human gates remain open.

- SC-008 unknown-state browser evidence: temporary target with source state
  unrecognized-review-state, rebuilt viewer port 49792, selected Fog and
  inspected target. Node uses state.unknown token with profile-provided fill
  #eef2ff and stroke #7483a9; label and expanded State interpretation retain
  unrecognized-review-state as declared/effective state. The generic renderer
  does not replace the source vocabulary with a known planning state. Fixture
  source restored, browser tab closed, temporary server stopped.

- Validation cancellation follow-up: a pre-cancelled empty or binding-only
  document returned success because cancellation was checked only in the
  profile loop. Regression reproduced the nil result. Validation now checks
  cancellation before registry construction and while checking bindings.
  Both cases retain context.Canceled through the application error wrapper;
  application/viewer race suites pass. Persistence already has its own context
  checks; this finding does not claim a cancelled write reached disk.

- Architecture delta: deepens configuration validation. configurationValidator
  now owns complete proposed-document validation with an isolated profile
  registry and injected layout validator. It has no Service, session map, store,
  or persistence access. saveConfiguration captures its dependencies before
  invoking it; subsequent validation/write/publication ordering is unchanged.
  Direct tests prove draft profiles are not published, duplicate IDs and missing
  binding targets are rejected, and layout failures propagate. Application/viewer
  race suites pass. Overall lifecycle/navigation separation remains incomplete.

- Typed retry evidence strengthened: Save and Save As now have explicit direct
  application assertions that exact retries return deeply equal original
  results, changed typed input returns a conflict, and the configuration file
  remains byte-for-byte unchanged across both retry forms. The retry and journal
  tests pass five race-enabled repetitions. This is persisted-state evidence,
  not only an error-code check.

- P2 typed profile input omitted from retry identity: application callers can
  omit raw transport bytes, causing Save/Save As with a changed profile but
  the same operation ID and target to replay the old result rather than reject
  conflicting input. A new direct application test reproduced Save As returning
  success for a changed name. Both commands now encode the typed profile into
  command identity in addition to existing transport bytes and arguments;
  unencodable profiles fail before persistence. Tests cover changed-input
  conflicts for Save and Save As with nil bodies; existing exact-retry tests and
  application/viewer race suites pass. No public payload or persistence format
  changes are introduced.

- Architecture delta: deepens the OKF configuration retry boundary. Extracted
  operationJournal owns successful-operation input matching and isolated retry
  results; excludes configuration validation, writes, and session publication.
  Configuration lifecycle callers now ask for replay/record instead of owning
  the map and conflict algorithm. Service's configuration lock still serializes
  complete writes; journal locking protects only its entries and introduces no
  callback or reverse lock acquisition. Direct tests cover zero-value journal,
  trimmed/blank IDs, input/output ownership, exact replay and changed-input
  conflict. Application/viewer race suites pass. The wider configuration service
  separation finding remains open; this extraction alone does not close it.

- P2 catalog report lost on navigation: live budget journey showed the invalid
  candidate diagnostic disappearing after Full because only Refresh passed
  catalog diagnostics into renderState. Every scene render now combines the
  current catalog diagnostics with projection diagnostics; Refresh no longer
  supplies a duplicate copy. Rebuilt viewer port 54699 confirms exactly one
  invalid-bundle diagnostic group survives Full and switching to Neutral.
  All 35 frontend test files, bootstrap syntax and diff checks pass. Browser
  tab closed and temporary server stopped.

- SC-009/010 live budget journey, rebuilt temporary fixture on port 49231:
  configured a one-node profile limit, enabled Full, and observed root only
  with one hidden concept and truncation guidance. Browse hidden concepts
  loaded target; Focus subtree rendered target and its detail while preserving
  project:review and Full. Back restored root, one-hidden count, truncation
  warning, and disabled Back. The node limit was not raised. Temporary profile
  configuration restored afterward, browser tab closed, test server stopped.
  This proves the node-budget journey, not every relationship-limit variant.

- Full gates after projection reset, diagnostic aggregation, and inspector
  extraction: `go test ./... -count=1`, `go test -race ./...`, `go vet ./...`,
  `go build ./...`, all 35 frontend test files, every viewer JavaScript syntax
  check, and diff checks pass. OKF validation reports 20 concepts, three indexes,
  one log, no errors or warnings. All 206 changed/new scoped code files are
  below 600 lines including blank lines. The pending renderer-selector question
  was surfaced again; no answer or automatic continuation is treated as approval.

- Inspector fallback follow-up: direct test installs an invalid profile with a
  missing base and raw-Markdown suppression. Inspection uses Neutral instead of
  the invalid detail settings and preserves the missing-base diagnostic and
  recovery guidance. Focused race test passes. Registry ResolveProfile composes
  declarations under one read lock; this inspection did not establish a
  mixed-version defect and does not claim a concurrent-detail acceptance test.

- Application separation follow-up: extracted conceptInspector from the session
  navigation module. Its dependency exposes only profile lookup/resolution and
  rule evaluation; it cannot access sessions, catalog discovery, or configuration
  writes. Service still owns selecting the session's index and rejecting a
  missing bundle. The inspector retains existing Neutral fallback and shared
  safe Markdown rendering. Direct tests check source identity, missing-profile
  fallback, foreign-concept rejection, and safe Markdown without constructing
  Service. Application/viewer race suites pass. This narrows the inspection
  responsibility but does not resolve the remaining configuration/navigation
  orchestration finding or add the pending DetailRenderer contract.

- SC-002 repair journey verified in rebuilt viewer on port 52225: temporary
  invalid candidate initially disabled; adding valid frontmatter and pressing
  Refresh enabled it and removed the validation error. Selecting that bundle
  rendered exactly Repaired concept; clicking it loaded the repaired source
  body. Both other bundles remained enabled. Restored the temporary invalid
  fixture afterward, closed the browser tab, and stopped the test server.
  Agent verification does not close the human visual-review gate.

- Invalid-candidate report live verification: rebuilt viewer on port 50304,
  expanded the existing diagnostic report and its affected-source disclosure.
  Observed `okf_bundle_invalid`, `missing YAML frontmatter`, `bad.md`,
  `bundle_id: invalid/.okf`, and the non-empty-type repair instruction. The
  valid selected bundle's two-node graph remained visible. Browser tab closed
  and temporary server stopped. This closes the preceding live-report evidence
  gap, not the remaining repair journey or human visual-review gate.

- P2 invalid-candidate diagnostics omitted from the page: discovery stores
  frontmatter/validation diagnostics on each BundleCandidate, separately from
  catalog-wide diagnostics. Refresh previously displayed only the latter, so
  the disabled bundle lacked its source-specific repair information. The load
  path now includes non-selectable candidates in the existing grouped report,
  retaining path/recovery and supplying bundle identity without mutating the
  catalog. Valid candidate diagnostics remain with their projections to avoid
  duplicate reports. `okf_state_test.js` covers aggregation and source ownership;
  all 35 frontend test files and syntax/diff checks pass. Live rendering of the
  newly included validation record has not yet been inspected.

- P2 refresh/detail race follow-up: a click on the old graph while catalog
  discovery was pending could start a newer detail request, then repopulate
  source content after invalidation cleared the projection. A deferred-response
  regression reproduced this before the fix. Projection reset now advances the
  detail request generation as well as clearing content. Both late success and
  late failure leave the cleared view and recovery message intact. All 35
  frontend test files pass; no browser timing simulation is claimed.

- P2 stale invalid-bundle projection: live Refresh after corrupting the selected
  temporary bundle left its previous graph, detail, and counts visible beside
  an error. The load path now clears projection-derived state and UI when the
  selected catalog candidate is no longer selectable. It preserves selection
  and layout draft for recovery, displays catalog diagnostics and recovery
  guidance, and leaves other bundles selectable. The same reset also clears
  hidden-concept controls when no valid bundle remains. Regression test:
  `okf_projection_reset_test.js`. Rebuilt viewer on port 48593 verified zero
  graph nodes and cleared detail/counts, then independent-bundle selection.
  The malformed temporary source was restored and the browser tab closed.

- Live bundle-isolation follow-up: reused the verified running temporary-fixture
  server on port 46579. Its invalid candidate remained disabled before and after
  Refresh. Selecting A/B/A showed only each bundle's concepts and restored A's
  bound profile; B selected Neutral. Browser tab closed after inspection. This
  does not cover invalidating a previously valid bundle or unreadable sources.
- Current verification after catalog ownership and profile lifecycle extraction:
  `go test ./... -count=1`, `go test -race ./...`, `go vet ./...`,
  `go build ./...`, all 34 browser test files, every viewer JavaScript syntax
  check, and `git diff --check` passed. OKF validation passed with 20 concepts,
  three indexes, one log, and no diagnostics. The 202 changed/new OKF, viewer,
  and shared-document code files remain below 600 lines. These gates do not
  resolve the outstanding architecture, contract, or human-review findings.

- P2: shape registration accepted identifiers containing the `@` reference
  delimiter, creating ambiguous ID/version keys and definitions that profile
  references could not resolve. Registration now rejects delimiter, whitespace,
  control-character, and empty namespace-segment identities. Regression tests
  verify rejection leaves the catalog untouched and valid version strings still
  round trip. Presentation, profile, and projection race suites pass. Numeric
  content-dimension overflow was subsequently reproduced and fixed below.

- Extension catalog follow-up: verified injected relationship-adapter descriptors
  include actual ID/version and optional description/schema without fabricating
  metadata for legacy adapters. Application/viewer race suites pass.
- P1: relationship adapter ID/version callbacks executed outside the panic guard,
  allowing a metadata failure to abort bundle processing. Metadata and execution
  now share the recovery boundary. `TestRelationshipAdapterMetadataPanicIsIsolated`
  covers both callbacks and proves a subsequent valid adapter still contributes
  its semantic relationship. `go test -race ./internal/okf/application
  ./internal/viewer` and `git diff --check` pass. Other extension provider families
  and the final acceptance/visual gates remain unverified; this is not closeout.

- P0: architecture Save reused a cached document and could erase subsequent OKF
  edits. Save As could replace unrelated destination sections. Both layout and
  OKF persistence now use `internal/projectdocument` to read the current document
  under a common writer lock and replace it atomically. Superseded duplicate
  atomic writers were removed; existing file permissions are retained.
  Evidence: `TestLayoutSavePreservesLaterOKFEditsAndAnalysisCanReload` and
  `TestLayoutSaveAsPreservesDestinationSections`.
- P1: the analysis configuration decoder rejected the new OKF section, breaking
  reload/reanalysis after profile persistence. It now permits unrelated root
  sections while retaining its strict owned-section validation. Covered by the
  configuration interoperability test above.
- P0: JSON `null` could reach a nil-map write in OKF configuration persistence.
  Shared document validation now rejects non-object documents. Evidence:
  `TestStoreRejectsNullDocumentWithoutMutatingIt`.
- P1: source refresh and profile Save retained stale session projections.
  Both now invalidate cached projections and increment request generations.
  Configuration mutations and refresh publication are serialized. Evidence:
  `TestSessionRebuildsAfterSourceRefreshAndProfileSave`.
- P1: caller-supplied/static profile revisions did not change with presentation
  edits or inherited changes. Revisions now hash profile content; effective
  revisions include composition. The session and inheritance tests cover this.
- P1: eager defaults and clone/JSON empty-slice conversions erased inherited
  settings and explicit empty overrides. Declarations stay sparse; defaults
  apply after composition. Boolean presence survives JSON persistence and
  explicit false detail/state/relationship overrides are honored. Evidence:
  `TestSparseProfileInheritanceSurvivesValidationAndPersistence`.
- P1: partial root hierarchy/relationship settings defaulted omitted booleans
  to false, and explicit zero navigation limits were replaced with defaults.
  Omitted booleans now receive defaults without overwriting explicit false.
  Navigation JSON preserves field presence through cloning/persistence, so
  zero overrides are rejected rather than inherited/defaulted. Evidence:
  `TestPartialRootProfileDefaultsOnlyOmittedBooleans` and
  `TestExplicitZeroNavigationLimitsAreNotDefaulted`.
- P1: profile command retries identified only raw bodies, ignoring targets and
  command identity. Application retry fingerprints now include command, target
  arguments, expected revision, and body. Further explicit cross-target retry
  regression coverage remains to be added.
- P1: profile rename left inherited base references pointing at the old ID.
  References are now renamed with the profile and its bindings. Concurrent
  session changes are guarded by the captured request generation.
- P1: custom profile decoding treated a wrapped profile as an opaque extension;
  the transport now distinguishes wrapped and direct forms before decoding.
  Concept query values are no longer URL-decoded twice.
- P1: browser detail selection shared the navigation/layout request generation;
  stale detail responses mutated state before checking freshness. Detail uses
  its own counter, invalidated by projection changes. Layout application gets a
  new generation and reapplies the current selection/semantic overlay after
  layout. Browser behavioral verification of this change remains pending.
- P1: cancelled profile drafts survived and could leak into later layout edits;
  editor close now discards that draft. Invalid form JSON now blocks Save.
- P1: unrelated form edits materialized all displayed defaults as overrides,
  breaking sparse profile inheritance. Form rendering now captures the original
  declaration and displayed values, and serialization applies only changes.
  Reverting an edit restores omission; shared layout edits remain independent.
  Browser tests cover sparse edits, reversion, row/map removal, and layout
  preservation. All 20 browser test files pass after this remediation.
- P1: Markdown advertised CommonMark but used regex parsing, missing reference
  links and treating code examples as relationships. Indexing and detail now
  share Goldmark parsing and a bundle-scoped link policy. Images remain inert,
  raw HTML is disabled, and safe local link IDs preserve spaces. Evidence:
  `TestCommonMarkLinksAndRenderingSharePolicy` and
  `TestUnsafeLinksAndRawHTMLRemainInert`; all OKF package tests pass.
- Full proposed configuration validation now checks dependent profiles and
  bindings before persistence. Delete accounts for inherited base references.
  `TestProfileInheritanceRenameAndDeleteAreAtomic` verifies rename propagation,
  rejection without a replacement, cycle rejection without disk mutation,
  and explicit neutral fallback. `TestProfileRetryIdentityIncludesCommandAndTarget`
  verifies same-command replay and cross-command/cross-target conflicts.
- P1: a shared configuration store keyed retries only by operation ID, allowing
  one project's result to be replayed in another project. Retry keys now include
  the absolute project root. `TestStoreScopesOperationIdentityByProject` verifies
  both projects persist independently with identical operation IDs and inputs.
- P1: source size limits were checked after an unbounded file read. Source
  reads now require regular files and stop at the limit plus one sentinel byte.
  Evidence: `TestReadConceptFileBoundsAllocation`.
- P1: concept detail ignored explicit raw/unknown-metadata visibility settings
  and sliced Markdown in the middle of UTF-8 characters. Explicit disabled
  sections are now omitted from the display, and bounded Markdown preserves
  character boundaries with a truncation diagnostic. Evidence:
  `TestDetailVisibilityAndUnicodeTruncation` and the detail markup test.
- P1: cycle exclusion removed the whole traversal path, including valid branches
  outside the cycle. Iterative traversal now removes only cyclic edges.
  `TestCycleExclusionPreservesBranchesOutsideTheCycle` covers this regression.
- P2: agreeing explicit parent/children declarations collapsed to one edge but
  lost one source of provenance. Deduplication now retains distinct proofs;
  `TestNormalizePrefersExplicitHierarchyAndSeparatesSemanticLinks` verifies both.
  All OKF package tests passed after these hierarchy fixes.
- P1: hand-written YAML conversion discarded aliases and silently overwrote
  duplicate nested keys. Frontmatter now uses the YAML decoder for aliases,
  merge keys, duplicate detection, and expansion limits, with string-key and
  JSON-value validation. `TestFrontmatterPreservesAliasesAndMergeValues` and
  `TestFrontmatterRejectsLossyOrRecursiveValues` cover preservation and rejection.
- P2: detail reads did not normalize the empty default-session alias, unlike
  navigation writes. Reads now share the same alias semantics. Back with no
  history now obtains the current initialized session instead of returning an
  empty/stale view. Evidence: `TestDefaultSessionAliasWorksForDetail` and
  `TestBackWithoutHistoryInitializesCurrentSession`. Navigation cancellation
  and rollback review remains open.
- P1: ready application services bypassed cancellation checks, allowing already
  cancelled requests to mutate navigation before projection rejected them.
  `ensureReady` now checks cancellation even on the ready path.
  `TestCancelledNavigationDoesNotMutateReadySession` verifies unchanged session
  state for cancelled depth and focus commands. Mid-flight rollback remains open.
- P1: navigation mutated active focus/profile/depth/history before projection
  completed. Commands now build detached candidate sessions and publish them
  only after successful projection, a final cancellation check, and request
  generation validation. Failed work leaves the previously published session
  intact. `TestNavigationPublishesOnlySuccessfulCurrentProjection` pauses a
  rule during projection and verifies cancellation, supersession, read isolation,
  and unchanged history. Earlier mid-flight rollback findings are addressed by
  this candidate-publication path; end-to-end browser behavior remains to review.
- P1: catalog refresh assigned profile/layout responses before checking request
  freshness. Refresh now collects results locally and publishes catalog,
  profiles, and layout options together only for the current request.
  Deferred-response browser tests verify stale refreshes cannot overwrite newer
  state, and a failed profile fetch does not publish a partial catalog.
- P1: HTTP routing decoded profile IDs twice and split encoded slashes as route
  separators. Routes now split the escaped path and decode each segment once.
  `TestOKFProfilePathDecodesEachSegmentExactlyOnce` exercises real HTTP Save As,
  Save, and Delete with percent sequences, encoded slashes, plus signs, and spaces.
- Evidence correction: issue 004's historical Fit acceptance and smoke-test
  observations were rewritten to describe the new implementation. Restored the
  original delivered evidence and added a clearly separated Fit follow-up citing
  the current automated tests, without claiming a new human visual review.
  Pending issues 065–071 still retain their explicit visual-review gates.
- P2: projection hidden-relationship counts omitted links hidden by profile
  visibility. Counts now include policy-hidden links without falsely reporting
  budget truncation. `TestRelationshipCountsIncludePolicyHiddenLinks` covers
  both policy and budget cases.
- P1: negative node-field lengths could panic during projection if a malformed
  profile reached rendering. Truncation now applies a safe default for nonpositive
  lengths and clamps oversized lengths to the hard cap. Covered by
  `TestNodeFieldTruncationDefendsAgainstInvalidProfileLimits`.
- P1: empty style-token and state-token maps were omitted from profile JSON,
  restoring inherited mappings after reload. Style serialization now preserves
  explicit empty maps. `TestEmptyStyleMappingsSurviveProfileRoundTrip` verifies
  the empty overrides through JSON decoding and profile composition.
- P2: the OKF node renderer ignored the effective rule-selected node shape,
  reading only the base style shape. It now honors the effective shape first.
  `okf_graph_test.js` renders the actual adapter and verifies shape precedence,
  worker URL forwarding/cleanup, and containment-only ELK input with semantic
  links absent from the initial rendering.
- Rendering boundary inspected: architecture and OKF call shared `edgeGeometry`;
  OKF uses shared `buildELKGraph` and `adaptELKLayout`. The explicit straight
  semantic-overlay route remains in the shared geometry module. This verifies
  those reuse paths, not the still-pending full architecture-delta assessment.
- P1: registered rules received shared source/profile maps and could unwind
  the request with a panic. Rule execution now clones document and parameter
  inputs and converts panics into rule diagnostics. Projection checks cancellation
  immediately after rule evaluation, including the last node. Evidence:
  `TestRuleFailureIsIsolatedFromSourceAndProfile` plus profile/projection/application
  race tests. Validation of individual strategy parameters remains to review.

## Architecture delta assessment

- Owner: shared project configuration, used by architecture layout and OKF
  knowledge views. Boundary: bounded atomic document updates, excluding the
  semantics of each configuration section. Delta: both writers use the same
  fresh-document transaction and no longer own duplicate atomic writers.
  Verdict: **deepens**. Evidence: `internal/projectdocument/document.go`,
  `internal/viewer/layout/persistence.go`, `internal/okf/configuration/store.go`,
  and the cross-section preservation tests recorded above.
- Owner: viewer graph rendering. Boundary: shared edge geometry, ELK input/output
  adaptation, and viewport gestures, with scene-specific adapters. Verdict:
  **deepens** for the extracted shared operations. Evidence: architecture
  `graph.js`, OKF `okf_graph.js`, shared `graph_route.js`, `layout_request.js`,
  and renderer/route tests. This does not establish all interaction parity.
- Owner: Configurable OKF knowledge views, issues 064–071. Boundary: application
  operations consumed by HTTP transport, excluding filesystem/configuration
  construction. Initial verdict: **widens**, P2, because the viewer depended on
  concrete `*application.Service` and reconfigured injected instances. Remediated
  verdict: **deepens**. `application.API` composes catalog, session, profile, and
  inspection operation interfaces, excluding construction/setter methods.
  `ServerOptions.OKFApplication` and `Server.okf` now use that boundary.
  `okf_composition.go` installs the default validator before returning the default
  application. Injected applications retain their dependencies, verified by
  `TestViewerPreservesInjectedApplicationDependencies`. Application/viewer tests
  pass with the race detector. The application implementation's own responsibility
  split remains a separate review item; this resolves the transport P2 only.
- Pre-existing debt: the viewer server already combines architecture transport
  and construction. The current OKF concrete dependency worsens that boundary;
  the concrete transport P2 above was remediated in this diff. No whole-repository architecture-health
  claim is made.

Commit readiness remains **not ready**. Remaining architecture and
acceptance/evidence checks must be resolved before a clean-review verdict.

## Remaining review and verification

- P2 extension-catalog contract gap confirmed: canonical-use-cases.md
  GetExtensionCatalog requires rules, relationship adapters, presentation
  properties, shapes, detail renderers, and versioned schemas. Rule schemas and
  injected relationship-adapter descriptors are now implemented and verified.
  Shape providers now have registered geometry definitions and editor choices;
  their final contract/schema audit remains open. Presentation-property and
  detail-renderer providers remain missing.
  Passing rule evaluation/validation tests does not satisfy this discovery
  requirement. Complete the supported extension descriptors and their schema
  contract, or obtain an explicit scope decision; do not advertise unregistered
  provider families merely to fill the catalog.

- Navigation contract decision required: FocusSubtree requires a real ConceptId;
  there is no top-level reset command. Reusing SelectBundle would reset the
  profile, full flag, depth, and history, violating the requested navigation
  preservation. Recommend an explicit ReturnToTopLevel operation that preserves
  bundle/profile/depth/full and pushes prior context for Back. Also reconcile
  NavigateBack's documented empty-history rejection with the implementation and
  existing test that initialize/return the current session instead. Do not
  silently redefine either behavior or treat an empty ConceptId as a command.

- Inherited profile display P1 has been remediated through backend preview and
  sparse form serialization (evidence below). Live form interaction remains at
  its visual-review gate, and numeric/control validation still needs review.

- Complete review of source discovery/indexing, hierarchy, projection budgets,
  rule validation, Markdown safety, contracts, styles, and changed callers.
- Continue configuration lifecycle review beyond the covered inheritance,
  atomic rejection, and cross-project/cross-command identity cases.
- Review inherited-value display in the form and remaining control validation;
  sparse serialization now has regression coverage.
- Inspect remaining profile empty-map override semantics; partial root settings
  and explicit zero navigation limits now have regression coverage.
- Check source refresh/failure behavior, initial selection/default profile,
  navigation rollback, processing deadlines, and supersession end to end.
- Complete architecture-delta assessment: shared document boundary deepens the
  persistence boundary and transport now uses the application API; application
  responsibility/injection boundaries require further review.
- Audit acceptance checkboxes and evidence against actual coverage. Preserve
  human visual-review gates; do not rewrite old smoke-test evidence as a newly
  executed result. No clean review/commit until this audit is complete.
- Re-run the strict review after remediation and run final relevant gates.

## Verification to date

- P2 scanner ownership/recovery: reproduced a scanner returning retained
  candidates. First indexing failure mutated that shared candidate to invalid
  and unselectable, so the second refresh skipped indexing even after recovery.
  Catalog construction now clones candidates and scan diagnostics through the
  existing helpers before editing/publication. Regression fails before the fix
  and passes afterward, proving a second indexing attempt succeeds and consumer
  edits to diagnostic details do not mutate scanner data. Application/viewer
  race suites and diff checks pass. No public interface change; ownership is
  enforced at the existing catalog boundary.

- Configuration separation follow-up: deletion/reference reassignment now uses
  `deleteConfigurationProfile`, producing an isolated candidate through a narrow
  profile lookup. Direct tests cover multiple bindings, inheritance, explicit
  replacement/neutral fallback, missing/self replacement rejection, and no input
  mutation or aliasing. The command still owns full validation, persistence,
  retry identity, and existing session behavior. Canonical DeleteProjectProfile
  does not mandate changing active sessions to the replacement, so this refactor
  does not introduce that policy. Application/viewer race suites and diff checks
  pass. Architecture delta: deepens configuration edits without new public API;
  the broader service-composition finding remains open.

- Configuration separation follow-up: rename reference rewriting now lives in
  pure `renameConfigurationProfile`, which validates identity collisions and
  rewrites inherited bases/bindings in an isolated configuration clone. The
  application command retains normalization, revision/idempotency checks,
  persistence, and generation-guarded session transitions. Direct tests require
  no store/server/session and cover multiple bindings, inherited references,
  unrelated bindings, collision/missing-ID rejection, and no input aliasing.
  Application/viewer race suites and diff checks pass. Architecture delta:
  deepens the configuration-edit boundary; the overall Service separation
  remains incomplete, not waived by this incremental extraction.

- Consolidated current-worktree checks after persistence coverage: full
  `go test ./... -count=1`, `go test -race ./...`, `go vet ./...`, and
  `go build ./...` pass. Every viewer JavaScript file passes `node --check`;
  all 34 browser test files pass. Diff check passes. The line audit covers 198
  current changed/new Go/JS/CSS/HTML files across OKF/viewer/projectdocument;
  all are below 600 lines. Open findings are summarized near the top of this
  report. These gates do not establish a clean review or authorize a commit.

- Persistence-envelope audit: canonical failure meta requires request_id; it
  does not require operation_id, so that omission is not a finding. Added
  contract tests proving write failures, revision conflicts, and idempotency
  conflicts keep their status/code/message plus supplied structured details and
  diagnostic profile/operation/recovery context. Contract and viewer race suites
  pass. This verifies envelope preservation, not that every persistence producer
  supplies complete diagnostics. No production contract change was made.

- SC-017 atomic-replacement follow-up: Windows-specific test opens the saved
  document without FILE_SHARE_DELETE. The real Save path returns
  okf_configuration_write_failed with access-denied cause; original file bytes
  and active session are unchanged, temporary files removed, and retry with the
  same operation ID succeeds after handle closure. Five race-enabled repetitions
  pass alongside read-path obstruction coverage. No production fix was needed.
  The first assertion allowed only sharing violation; the observed access-denied
  result was added as an expected OS denial. Other platforms and browser feedback
  remain separate verification work.

- SC-017 filesystem follow-up: `configuration_filesystem_failure_test.go` uses
  the actual store with a temporary directory obstructing the configuration file
  path. Save fails without changing the selected session or replacing the
  obstruction. Restoring the original bytes/file permits the same operation ID
  to save, demonstrating failed attempts do not poison retry identity. Focused
  race test passes. This is a read-path filesystem failure, not proof of an
  interrupted atomic replacement or browser infrastructure-error handling.

- Current extension completeness audit: `ports.go` exposes RuleStrategy,
  RelationshipAdapter, and ShapeProvider, and `application/extensions.go`
  catalogs those three kinds. Canonical-domain-model.md, Extension points,
  additionally requires PresentationPropertyProvider, DetailRenderer, and
  DiagnosticProvider. Those named seams remain absent; rule annotations and
  static diagnostics are not evidence that their provider contracts exist.
  This remains a P2 specification gap, not an approved deferral. The proposed
  DetailRenderer profile selector (`details.renderer` with ID/version/parameters,
  default CommonMark, shared safety boundary) is awaiting user approval. Other
  family contracts must also be made explicit before claiming extensibility is
  complete. Existing safety and graph tests cover built-in behavior only.

- Live SC-015/018 persistence follow-up: isolated project profile Save updates
  name and adds a custom presentation field, preserving the independent
  architecture layout, binding, and unknown root/profile extensions on disk.
  Two editor tabs reproduce stale revision rejection with retained draft and
  first writer's name preserved. Explicit Cancel/Refresh/reopen/manual merge
  then saves successfully. No production edit was needed. Scenario audit records
  the fixture and controls used; tabs/server stopped. Save As browser retry and
  other lifecycle variants still need verification, and human gates remain open.

- SC-012 live safety follow-up: an isolated temporary project exercised raw
  script/onerror HTML, remote image, javascript/data links, a bundle-escaping
  path, safe local/external links, and CommonMark formatting. Browser DOM had
  no active HTML/image elements or execution sentinels. Unsafe targets became
  inert text with scoped diagnostics; external anchor attributes were safe.
  Clicking the local link selected its target and loaded detail without leaving
  the viewer. External new-tab interaction and other attack variants remain
  unverified. Fixture path and details are in the scenario audit. Tab/server
  stopped; no repository source changes or persistence commands were used.

- OKF conformance revalidation: the skill's `okf-validate.mjs .okf --json`
  reports conformant/passed, 20 concepts, 3 indexes, 1 log, no errors, and no
  warnings. Issue 071's historical SC-008 note now distinguishes the executed
  current-browser implemented/mixed-state paths from the unverified unmapped
  variant. No acceptance checkbox or capability state was changed. Diff check
  passes. Verification-command discovery found no `.github` workflow directory;
  README names `go test ./...`, while Makefile exposes analyzer/release packaging
  rather than an additional test/lint target. The issue's explicit Go/Node and
  cross-surface verification obligations therefore remain the review gate set.

- Live SC-007/008/009 follow-up on port 53757: Go then code-quality selection
  replaces incident semantic overlays while all node transforms and breadcrumbs
  remain unchanged. Detail states match their graph fields. Full displays all
  20 concepts; keyboard depth 1/2 changes produce exact 7/17-node frontiers and
  retain Fog. See scenario evidence for counts and the automation-fill caveat.
  No persistence writes; temporary tab/server closed. Unknown-state, budget,
  other-profile variants, and required human approval remain open.

- SC-008 HTTP inspection follow-up: new `okf_detail_state_test.go` verifies Fog
  projection/detail declared/effective state parity for every visible concept
  with matching source revision and bundle identity. Agreeing children roll up
  the parent's effective state; mixed or unmapped children retain its declared
  state. Raw parent frontmatter remains bounded in all cases. Focused race test
  and diff check pass; file is 71 lines. No production fix was needed. Live
  detail inspection and the pending DetailRenderer contract decision remain open.

- P2 profile catalog consistency: reproduced a concurrent deletion while an
  earlier catalog request was paused in rule parameter validation. The response
  retained the deleted profile's old declaration/configuration revision but
  resolved it against the new registry, producing a missing-profile diagnostic
  and empty status. `Profiles` now captures an isolated registry with the
  configuration using existing `WithProjectProfiles`; all status resolution and
  rule registry revision reads use that snapshot outside the service lock.
  `TestProfileCatalogResolvesOneConfigurationSnapshot` fails before the fix and
  passes afterward, including 20 race-enabled repetitions. It proves the paused
  response equals the original catalog and a later request sees the deletion.
  Application/viewer race suites and diff checks pass. Test file is 98 lines.
  Architecture delta: neutral; the existing registry snapshot boundary is reused
  and no provider callbacks are moved under the application lock.

- SC-005 live browser follow-up: current build on port 53298 completed
  Neutral/Fog/Neutral through the profile selector. All 17 concept IDs remained;
  Fog added 16 state fields, and computed styles confirmed violet root, green
  implemented nodes, and specified-state fill with a magenta 3px roll-up border.
  Neutral restored exact node markup and transforms with zero fields. The
  architecture link returned to the layered architecture graph and its controls.
  No persistence command was used. Temporary tab/server stopped. Human visual
  approval and the remaining scenario journeys are still open.

- Post-catalog-extraction gates: `go test ./... -count=1`, `go test -race ./...`,
  `go vet ./...`, and `go build ./...` pass. `git diff --check` passes. Every
  current changed/new Go/JS/CSS/HTML file under internal/okf, internal/viewer,
  and internal/projectdocument is below 600 lines. These passing checks do not
  close the outstanding architecture, extension-contract, or visual gates.

- Application boundary follow-up: source catalog construction now belongs to
  `catalogBuilder`, depending only on scanner, indexer, and relationship adapter
  ports. `Service.Refresh` coordinates configuration loading and atomic
  publication/session invalidation, but no longer implements source iteration
  and per-bundle failure handling. The direct builder test proves independent
  failure isolation with scanner/indexer doubles and no store/session setup.
  Application and viewer race suites pass. Architecture delta: deepens this
  catalog boundary. Owner: OKF bundle catalog. The broader Service still owns
  navigation and configuration commands; that separation is not declared
  complete by this extraction. No new public interface or endpoint was added.

- P2 conflict guidance: the dialog exposed only the backend revision-conflict
  message. Its Refresh control is outside the modal, and Cancel clears the
  draft, so users lacked instructions for preserving work before reload/merge.
  Shared `configurationFailureMessage` now tells users to copy Advanced JSON
  before Cancel, Refresh, reopen, and merge into the latest profile. Save and
  other configuration commands reuse this formatter without changing revisions
  or retrying. The API/editor test asserts the instructions and retained draft.
  All 34 browser test files pass; changed JavaScript passes syntax checking.
  Live interaction remains unverified. Architecture delta: neutral. Owner is
  OKF profile configuration; its UI error presentation remains in the existing
  lifecycle module and callers gain no new persistence or recovery workflow.

- SC-018 frontend follow-up: `okf_profile_conflict_test.js` connects the actual
  API error decoder and editor save handler. A simulated 409 preserves draft,
  Advanced JSON, layout override, selected profile, and stale revision; it shows
  the error without closing, refreshing, or retrying. A subsequent Save As with
  an explicitly supplied fresh revision retains extension data and selects the
  copy. Both this test and existing save tests pass, as does syntax checking.
  The test does not exercise a browser revision-refresh control. That recovery
  path still needs live verification; no clean-review claim follows.

- SC-018 boundary follow-up: added `TestOKFHTTPRejectsStaleEditorAndReplaysCreation`.
  Two editors read the same catalog revision. The first Save publishes its name
  and advances revision; the stale Save returns 409/okf_revision_conflict. A
  retried creation returns its original response without reverting that Save,
  and a changed target returns 409/okf_idempotency_conflict. Whole-file bytes and
  current catalog remain unchanged after rejected/replayed requests. Focused
  viewer race tests pass, including the re-inspected SC-015 lifecycle test.
  No production change was needed. Browser conflict recovery and the broader
  review remain open.

- SC-005 HTTP round trip now verifies Neutral/Fog/Neutral on one source: source
  revision and concept IDs stay fixed, styling changes, Fog shows the mapped
  state field, and Neutral restores its original nodes and projection revision.
  Source bytes remain unchanged and no `.archview.json` is created by session
  selection. Focused race verification passes; no production change was needed.
  The corresponding live profile-switch journey remains open in the matrix.

- SC-009 now has HTTP depth 1/2/99/full sequence coverage, exact node and hidden
  counts, and retained bundle/Fog profile. A semantic link to the deepest concept
  cannot enlarge the shallow containment frontier. Focused race test passes.
  Consolidated current-worktree gates also pass: `go test ./... -count=1`,
  `go test -race ./...`, `go vet ./...`, `go build ./...`, all 33 browser test
  files, syntax checks for 69 non-ELK JavaScript files, and `git diff --check`.
  The line audit covers 189 changed/new OKF, viewer, and project-document files;
  none reaches 600 lines. These automated gates do not close the incomplete
  scenario surfaces, missing extension families, pending contract decisions, or
  human visual-review gates.

- P2: GetProfileCatalog still returned the hard-coded `builtin:rules:v1` registry
  revision after custom registrations. It now hashes registered rule identities
  and validated shape definitions, independent of registration order and project
  profile edits. Optional metadata callbacks are not invoked, preserving profile
  availability when a description/schema callback is broken. The full extension
  catalog retains its separate metadata/schema content revision. Application and
  viewer race suites pass, including stable/change/order and metadata-failure
  regressions. The wire semantics are documented; other review gates stay open.

- SC-006/009/014 assertion audit inspected existing hierarchy and projection
  tests and updated the scenario matrix without duplicating their coverage.
  Added the missing application journey for opening a saved binding with an
  unavailable base, rule, or rule version. Each case produces usable fallback,
  retains scoped-layer diagnostics and the repairable binding/declaration, and
  leaves configuration bytes unchanged. Focused race verification passes. The
  complete depth sequence and HTTP/browser repair remain open.

- Live accessibility follow-up compared the automation AX mapping with the
  DOM-backed browser snapshot. The latter exposes all 17 named node buttons in
  normal and Full canvas mode. Keyboard Enter on Go analysis selected it,
  populated its source detail, and preserved `translate(581 143)`. Thus the
  earlier unnamed checkbox report is not sufficient evidence to change SVG
  markup; actual assistive-technology review remains open. Double-click focused
  Go analysis to one concept, enabled Back, and retained Neutral/depth 2. Back
  restored 17 concepts, disabled Back, and restored the original node position.
  These are current-build live observations, not proof of every navigation or
  screen-reader variant. Temporary loopback server and browser tab were closed.

- Current-build live browser check: launched `go run ./cmd/arch-view open
  --project . --language go --analyzer-runtime in-process --port 0` on loopback.
  The root URL displayed the loaded architecture scene and an OKF navigation
  link. Following that link opened Neutral OKF with 17 visible concepts. In
  Full canvas, wheel-up changed zoom from 144% to 152%; Fit selected 159%.
  Screenshot inspection showed the tree and controls within the canvas. No
  project configuration was saved. Temporary tab and server were stopped.
  Accessibility follow-up remains: the automation accessibility tree initially
  exposed unnamed checkbox-like node entries and then only the graph container
  in Full canvas, despite source markup containing button roles and labels.
  Determine whether this is SVG `role=img` subtree suppression or a tool mapping
  issue before changing markup. This agent check is not human visual sign-off.

- SC-017 x SC-019 failure counterpart verified: an injected configuration-store
  failure during paused projection leaves the entire configuration file and
  active session unchanged. Releasing the pending focus then succeeds, proving
  failed persistence does not spuriously supersede valid work. Focused race tests
  cover successful Save/Delete and failed Save together. No production change
  was needed. Real filesystem-failure and HTTP/browser feedback remain separate
  evidence obligations in the scenario audit.

- Stateful SC-019 audit now includes active-profile Save and Delete while rule
  evaluation is paused. Both operations invalidate old work; releasing it returns
  `okf_operation_superseded`, leaves the current rendered label/profile intact,
  and cannot publish the abandoned focus or Back history. The public application
  regression passes under race detection. This is application evidence, not
  HTTP/browser journey coverage or failed-persistence coverage. No production
  change was needed for these two cases.

- SC-013 composition audit added a public-registry regression rather than relying
  only on private merge-helper tests. Validation/evaluation correctly preserve
  compatible annotations, highest-priority scalars, neutral equal-priority
  conflicts, and identical results/diagnostics across three registration orders.
  Third and lower-priority rules do not resurrect conflicted fields. Focused
  race verification passes; no implementation change was needed. HTTP/browser
  acceptance evidence remains open in the scenario audit.

- Started assertion-level SC-001–SC-020 evidence audit in
  `docs/agents/reviews/20260904-okf-scenario-evidence.md`, distinguishing inspected
  coverage from unverified scenarios and surfaces. Added and executed an HTTP
  SC-001 regression selecting two valid bundles A/B/A, verifying deterministic
  catalog order, exact node isolation, foreign-detail rejection, neutral default,
  and unchanged source bytes. Its focused race run passes. This does not close
  the complete matrix or human review gates.

- P2: rule registration accepted ambiguous ID/version delimiters and let ID or
  Version callback panics escape. Both failures were reproduced in registration
  tests. Rules and shapes now reuse one domain identity validator. Rule metadata
  is read once inside recovery before taking the registry lock, and rejected
  metadata is never published. Regression tests cover invalid identities,
  callback failures, and successful lookup after failed registrations. All OKF
  and viewer race suites pass; pending contract decisions are unchanged.

- P2: catalog schema copying retained provider-owned `[]string` slices, including
  `required` and nested `enum` lists. A registered shared-schema regression failed
  before the fix: mutating a catalog response changed the provider's schema.
  The shared domain clone helper now copies string slices while preserving nil
  versus empty representation. All OKF and viewer race suites pass. The pending
  detail-renderer profile contract decision remains unanswered and unchanged.

- Detail-renderer contract audit: `interaction/markdown.go:Detail` directly calls
  `SanitizeMarkdown`, which directly calls the fixed CommonMark implementation.
  `domain.DetailSettings` contains only raw/unknown visibility flags and
  `application.Extensions` exposes no detail-renderer descriptors. Thus the
  canonical DetailRenderer seam and GetExtensionCatalog requirement remain
  unimplemented; existing safe-Markdown tests do not prove extension support.
  The canonical references describe safe detail formats but do not define
  renderer selection or parameter wire fields. Proposed contract decision:
  optional `details.renderer: {id, version, parameters}`, defaulting to the
  existing CommonMark renderer. Registered implementations would produce
  declarative content that still passes through the existing size/link/security
  boundary, never trusted HTML. Obtain a decision on that profile contract
  before adding selection/persistence/editor behavior; no such change made here.

- P2: a subnormal positive shape content extent passed Go registration and
  browser validation, allowing node-size division to produce Infinity. Both
  failures were reproduced before remediation. Registration, browser validation,
  and the definition schema now share the `2^-52` minimum unit-box extent.
  Tests cover rejected subnormal width/height, the exact precision boundary,
  finite sizing, and the existing rectangle fallback for invalid definitions.
  All 33 browser test files and presentation/application/viewer race suites pass;
  `git diff --check` passes. This closes the tracked numeric-overflow finding,
  not the remaining extension and acceptance review.

- Shape descriptors now publish a versioned `definition_schema` for their
  declarative geometry, including geometry-specific field constraints. The HTTP
  regression verifies publication and content revision; a nested-mutation test
  verifies catalog schema ownership. Cross-coordinate containment and convexity
  remain registration checks, explicitly documented rather than claimed as JSON
  Schema checks. All OKF/viewer tests and presentation/application/viewer race
  suites pass, as does `git diff --check`. New and modified code remains below
  600 lines. Other extension families and the overall acceptance audit remain
  open; this is not a clean-review verdict.

- Shared route review found custom-shape self-loops and orthogonal fallback
  routes still attached to rectangular bounds. Both now use the existing shared
  shape-boundary functions. A custom trapezoid regression verifies self-loop and
  fallback attachment plus unchanged legacy rectangle attachment. All 33 browser
  test files, `go test ./... -count=1`, `go vet ./...`, `go build ./...`, syntax
  checks on 69 non-ELK JavaScript files, and `git diff --check` pass. Changed route
  and regression files are 355 and 21 lines. `go test -race ./...` also passes;
  acceptance, remaining extension contracts, and visual gates are still open.

- Profile style-token shape inputs now offer registered ID/version suggestions
  from GetExtensionCatalog. Inputs remain editable and server validation remains
  authoritative. Suggestions survive form rerender and do not replace draft
  controls when the asynchronous catalog arrives; catalog failures preserve
  editing. Tests cover kind filtering, HTML escaping, per-form cache isolation,
  draft preservation, and failure behavior. All 32 browser test files,
  viewer/export Go tests, touched-script syntax checks, and diff check pass.
  Human interaction/accessibility review of the datalist remains pending.

- Persisted style-token shapes now validate against the registered shape ID and
  version after inheritance. Unknown references invalidate profiles before
  persistence; runtime-generated rule shapes still receive projection fallback
  diagnostics. Tests cover built-in aliases, registered custom versions,
  inherited missing versions, and HTTP Save/Save As rejection. Invalid Save As
  creates no configuration file; invalid Save leaves existing bytes unchanged.
  Profile/application/viewer race suites and the focused HTTP race regression
  pass. Shape schema/editor discovery and visual verification remain open.

- Shape catalog exposure connected: GetExtensionCatalog now includes six actual
  built-in providers plus injected shape definitions, IDs, versions, descriptions,
  and capabilities. Content revisions include geometry. Tests verify wire
  definitions, custom versioned registration, catalog revision changes, and
  protection against catalog-client mutation. Application/viewer race suites
  and diff checks pass. Saved-profile reference validation and complete shape
  schema/catalog/editor integration review remain pending.

- Browser shape-definition consumption connected: shared SVG drawing, ELK and
  fallback sizing, text placement, and endpoint attachment now consume scene
  shape_definition data. A custom versioned triangle test exercises geometry,
  content placement, and boundary intersection without a browser name entry.
  Invalid geometry types/non-numeric coordinates fall back without injection.
  All 31 browser test files pass. Registration catalog exposure and validation
  of persisted shape references remain pending, along with visual verification.

- Shape providers are now wired into the profile registry and projection.
  Six validated built-ins register through the same provider boundary as custom
  shapes. References resolve namespaced IDs and versions, with legacy built-in
  short names retained. Scene nodes carry defensive shape definitions; unknown
  references fall back with a concept-scoped diagnostic. Tests cover a custom
  versioned triangle, provider/snapshot isolation, and explicit unknown-shape
  fallback. OKF/viewer tests and focused projection/profile/presentation/viewer
  race suites pass. Browser consumption, extension-catalog exposure, and
  validation of saved profile shape references remain pending. Do not claim
  custom provider rendering works until those paths are connected and verified.

- Shape-definition geometry validation now rejects degenerate, duplicate,
  self-crossing, and non-convex polygons; either winding order is accepted.
  Content-box corners must lie inside the declared outline, and polygon outlines
  must contain the unit-box center used for boundary attachment. Ellipse and
  rounded-rectangle checks reject escaping text regions and irrelevant geometry
  fields. Registration tests verify rejected geometry is never published.
  Presentation race tests, vet, and diff check pass. Provider integration is
  still pending; this is validation of the newly introduced definition contract,
  not a claim of an active extension catalog or browser path.

- Shape-provider foundation added: renderer-neutral unit-box definitions and a
  versioned registry that snapshots provider output, rejects invalid numeric
  bounds/unsupported geometry/duplicate IDs and versions, and isolates provider
  panics. Tests verify provider, caller, and catalog mutation isolation plus
  malformed registration rejection. Presentation/domain race suites and diff
  check pass. This registry is NOT yet wired into application catalogs,
  projection, profile validation, or browser rendering; polygon validity and
  content containment checks also remain. It does not yet satisfy ShapeProvider.
  Ownership is the OKF presentation boundary, excluding source interpretation
  and executable browser markup.

- Shared shape-boundary geometry: straight routes now intersect diamond,
  hexagon, ellipse, and pill outlines. Layout-provided endpoint positions are
  projected onto those outlines without changing internal bends/control points
  or mutating cached routes. OKF's adapter adds shape facts to position records;
  the shared edge renderer owns attachment. Architecture rectangle geometry is
  unchanged. Tests check outline equations across quadrants/axes, route
  immutability, rectangular no-op behavior, and malformed-route handling. All
  30 browser test files and viewer/export Go tests pass; focused boundary/route
  tests and syntax/diff checks pass after the malformed-input guard. Visual
  verification and extension-provider registration remain open.

- Shape vocabulary rendering follow-up: shared markup now includes diamond and
  hexagon. Shared shape sizing reserves an inscribed text rectangle; OKF text
  translates into that area, and shared ELK node dimensions enlarge the outline
  instead of reducing font size. Fallback column widths now respect enlarged
  nodes. Tests cover polygon coordinates, content-dimension round trips, rendered
  label translation, and non-overlap. All 29 browser test files passed, followed
  by focused graph/shape/layout tests and viewer/export Go tests after the added
  integration assertion. Edge attachment to nonrectangular outlines, provider
  registration, unknown-shape diagnostics, and visual review remain open.

- Shape rendering review: both viewers now call the shared node_shape module
  for SVG node bodies. Architecture retains byte-equivalent rounded rectangles
  and hit zones. OKF rectangle and pill tokens now render distinct corner radii;
  ellipse behavior is preserved. Geometry tests and all 29 browser test files
  pass, as do viewer/export Go tests. Architecture delta: deepens shared drawing
  while leaving scene selection and styling in adapters. No CSS was added.
- P1 shape-provider gap remains: PRD requires diamond and hexagon as well as
  rectangle/rounded rectangle/pill. Diamond/hexagon, provider registration,
  unknown-token diagnostics, shape-aware text bounds and edge attachment still
  require review/implementation. The shared markup extraction is a prerequisite,
  not completion of ShapeProvider or of the full shape vocabulary.

- Consolidated verification after generic roll-up remediation: `go test ./...
  -count=1`, `go test -race ./...`, `go vet ./...`, `go build ./...`, all 28
  browser test files, syntax checks for 61 non-ELK JavaScript files, and
  `git diff --check` pass. Line audit covers 156 changed/new existing files under
  internal/okf, internal/viewer, and internal/projectdocument; none reach 600
  lines. OKF validator reports conformant/passed, 20 concepts, three indexes,
  one log, zero errors and warnings. All verification processes completed.
  These automated results do not satisfy the pending human visual review or
  prove the full SC-001–SC-020 cross-surface matrix. Reinspection of canonical
  extension seams confirms presentation-property, shape, detail-renderer, and
  diagnostic-provider behavior still needs implementation/contract review.

- Generic roll-up mapping remediation: removed source role/type/state_policy
  inference from ResolveConceptState. It now interprets rule-selected roles only.
  Fog declares its planning vocabulary through existing metadata_equals rules
  at fallback priority -1. Metadata rules and declared-state lookup support dotted
  nested paths, preserving literal-key precedence. Tests prove Fog vs Neutral
  behavior for identical source data and a custom unrelated nested vocabulary.
  The detail/graph fixture now supplies its aggregate mapping explicitly rather
  than depending on implicit type interpretation. All OKF and viewer race suites
  pass. Architecture delta: deepens profile interpretation by moving planning
  conventions out of the shared resolver into profile data. No new strategy,
  CSS, source mutation, or registry branch was added.

- Role precedence correction: an explicit rule-selected non-roll-up role no
  longer loses to source `state_policy.mode: rollup`. Regression tests cover an
  independent rule role, an explicit roll-up rule role, legacy fallback without
  a rule role, declared/effective state, and source immutability. Profile,
  interaction, and projection race suites pass.
- P1 generic-profile gap confirmed in state_resolution.go: implicit role lookup
  still recognizes source `role`, type names aggregate/rollup/roll-up, and
  `state_policy.mode` outside a profile-declared mapping. Current rule strategies
  can supply renderer-neutral roles, but the fallback vocabulary is hard-coded.
  Move those conventions into Fog's declarative policy while preserving explicit
  user role mappings; do not claim the precedence fix completes genericity.

- Per-bundle source-input limits implemented at FilesystemScanner: default
  10,000 Markdown files and 64 MiB, configurable by the host before scanner use.
  Index/log files consume the same budget. Exceeding a limit stops indexing,
  discards partial documents, and produces a scale/error diagnostic with recovery
  guidance. Tests prove exact-limit acceptance, rejection without a revision or
  partial documents, and independent smaller-bundle availability. Source,
  application, and viewer race suites pass. This bounds accepted raw source
  input, not exact parsed-heap usage, all project bundles in aggregate, or directory
  enumeration memory. Architecture delta: neutral in source ownership; no new
  application/client workflow or profile configuration responsibility.

- Source revision memory remediation: buildIndex now collects sorted paths and
  hashes each successfully read file before parsing it. It no longer retains a
  second bundle-wide map of raw bytes. The regression test compares the original
  sorted-path/raw-byte hash protocol with the result, including punctuation vs
  directory order, CRLF content, and index/log files. Index-file edits still
  change the revision and indexed Markdown is unchanged. Source/application/viewer
  race suites pass. This reduces duplicate retention; it does not impose a total
  source budget or bound parsed document memory. The aggregate-scale finding
  remains open. Architecture delta: neutral within the source-indexing boundary.

- Refresh interruption follow-up: the checks immediately after discovery and
  between candidates discarded context causes and mislabeled deadlines as
  cancellation. Both now use the shared context error mapping. The regression
  test asserts errors.Is, error code, HTTP status, and unchanged published
  catalog for cancellation and deadline expiry. Application/viewer race suites
  pass. The application boundary remains unchanged.
- Source scale review: the 8 MiB per-document read cap does not bound aggregate
  bundle memory. buildIndex retains raw bytes for every Markdown file as well
  as parsed documents, and discovery validates before indexing again. There is
  no total byte/file budget in FilesystemScanner. This remains a scale finding,
  not a claim that the projection's node/relationship limits bound indexing.
  A fix must preserve lossless accepted indexes and report a rejected oversized
  bundle explicitly rather than silently dropping documents.

- Extension metadata HTTP follow-up: real injected rule and relationship-adapter
  metadata panics now have end-to-end handler tests. Both return status 500,
  the expected error code and request ID, and no partial success data. A later
  profiles request succeeds on the same server. The test file is 97 lines.
  Executed successfully: focused HTTP race tests, `go test ./... -count=1`,
  `go vet ./...`, `go build ./...`, all 28 browser test files through
  `node --test`, and `git diff --check`.
- P2 extension revision finding resolved: the fixed `builtin:rules:v1` revision
  was unrelated to injected providers. The contract now computes a SHA-256
  revision from the returned, sorted catalog, including schemas. Invalid JSON
  metadata fails before success is written. Contract tests cover stable object
  key order, schema/version changes, and encoding rejection; HTTP tests cover
  identical catalogs, changed injected versions/schemas, and correspondence
  between the returned descriptors and revision. Contract/viewer race suites
  and `git diff --check` pass. Architecture delta: neutral; serialization identity
  belongs to the existing contract boundary and does not expand application API.

- Rule catalog callback failure isolation: description/schema panics now return
  a structured `okf_extension_catalog_failed` error with HTTP status 500 rather
  than escaping the application query. The regression test covers both metadata
  callbacks and verifies that registered rules remain resolvable after failure.
  Architecture delta: neutral, owned by OKF extension discovery. The application
  query still owns transport-neutral errors and delegates catalog creation to
  the existing RuleRegistry port; no new client workflow or provider execution
  responsibility was added.
  Verification: application and viewer race suites and `git diff --check` pass.

- Relationship adapter execution review: the port is actively injected and
  invoked during Refresh, not merely declared. Invocation panics previously
  escaped and cancellation after an empty result was missed. Adapter invocation
  now converts panics to adapter-scoped diagnostics, checks cancellation directly
  after return, and uses canonical timeout/cancellation errors. Returned
  diagnostic/provenance slices are copied before enrichment to avoid mutating
  provider-owned output. Regression tests prove a failing adapter does not
  prevent a later valid adapter and empty results do not swallow cancellation.
  Application/viewer race tests and diff checks passed. Adapter metadata
  discovery and other provider-family requirements remain open.

- Rule-schema HTTP evidence: an actual httptest server now verifies the
  `/v1/okf/extensions` envelope includes all five built-in versioned rule
  descriptors with object schemas, properties, and required parameters.
  The focused HTTP race test passed. Documented the additive parameter_schema
  wire field, strategy-version association, legacy omission, and authoritative
  profile validation in the canonical transport document. This does not reduce
  the still-open requirement for the other extension provider families.

- Rule schema catalog tranche: built-in strategies now implement the optional
  ParameterSchemaProvider interface and expose object parameter schemas through
  `parameter_schema` on their versioned extension descriptor. The registry clones
  provider data without switching on strategy IDs. Legacy injected strategies
  remain compatible; absent schemas are omitted rather than fabricated. Tests
  cover all five built-ins, serializable metadata, and caller-mutation isolation.
  Profile/application/viewer race tests and diff checks passed. The OCP skill
  guided ownership of schema metadata by each strategy. Other provider-family
  catalogs and full schema/validator conformance evidence remain open.

- OKF artifact gate: ran the installed okf-validate skill's bundled validator
  against the current `.okf` with `--json`. It reported conformant/passed,
  20 concepts, 3 indexes, 1 log, zero errors, and zero warnings (exit 0).
  This verifies bundle conformance and planning-profile roll-up consistency;
  it does not authorize capability-state promotion, closeout, or human visual
  approval. No OKF files were changed by this validation pass.

- Extension catalog reentrancy: Catalog invoked strategy ID/version/description
  under the registry read lock, allowing an injected metadata callback that
  needs the write lock to deadlock. It now snapshots strategy references under
  lock and invokes callbacks after releasing it, retaining deterministic sorting.
  A reentrant metadata regression and profile/application/viewer race suites
  passed, as did diff checks. Parameter-schema catalog completeness remains
  under review. The pending navigation semantics question is still unanswered;
  automatic goal continuations do not authorize that contract change.

- Hidden-concept live journey: at 375×812, expanded the hidden selector, focused
  compiled-external-analyzer-distribution (absent from the initial graph), and
  used Back to restore Top level. Long IDs initially overflowed the page; reused
  existing `okf-editor-field` styling, rebuilt the server, and verified selector
  bounds 17.59–342.41px inside the 360px content viewport with scrollWidth equal
  to clientWidth. No CSS was added. Screenshot inspection confirms this control
  fits. Focused browser test and diff check passed. Tabs closed, viewport reset,
  both temporary servers stopped. This covers depth-hidden recovery; the backend
  budget-truncation journey is separately tested, not a live budgeted UI journey.

- SC-010 viewer recovery implementation: added a collapsed hidden-concept
  selector using the existing bundle summary's stable concept IDs and existing
  Focus operation. It loads only when opened, excludes currently visible nodes,
  checks source revision, and ignores stale responses after projection changes.
  No new endpoint, source write, or graph expansion is introduced. Tests cover
  lazy loading, filtering, escaping, focus ID forwarding, revision mismatch,
  and supersession. All 28 browser test files, bootstrap syntax, and diff checks
  passed. The application truncation/focus/Back regression already passes;
  live-browser recovery and responsive inspection of this new control remain
  required before claiming complete SC-010 journey evidence.

- SC-010 backend recovery evidence: a real filesystem bundle plus saved limited
  profile now exercises full-mode truncation, focus of a concept proven absent
  from the initial projection, retained bundle/profile/full policy, and Back to
  the identical prior projection revision/nodes. The focused test and full
  application race tests passed; diff checks passed. This proves application
  recovery, not viewer reachability: the accessible concept list is still built
  only from snapshot.nodes, so arbitrary hidden subtree discovery is not yet
  demonstrated in the UI. Keep the end-to-end recovery requirement open.

- Diagnostic report completeness: grouped warnings discarded recovery guidance
  and all affected contexts after the first four. The compact summary remains,
  with each distinct recovery message shown once and full scoped facts available
  in an expandable section. Bundle/concept/profile/operation IDs and structured
  diagnostic details are retained without repeating the warning sentence.
  Tests cover 17 grouped occurrences, final-path availability, recovery
  deduplication, context retention, and HTML escaping. All 27 browser test
  files, module syntax, and diff checks passed. No CSS was added. This improves
  actionable failure reporting but does not by itself complete SC-010 recovery
  or the full acceptance matrix.

- Detail settings follow-up from responsive inspection: `state.show_declared`
  was ignored by Detail, and effective profiles omitted true detail defaults,
  making the form show unchecked controls for enabled behavior. Explicit false
  now omits declared state without changing effective state or source metadata.
  Profile normalization exposes the existing enabled defaults for declared
  state, raw Markdown, and unknown metadata; explicit false remains false.
  Normalization was extracted from the near-limit registry into defaults.go.
  Tests cover effective defaults, explicit false normalization, and detail
  visibility independent of other fields. All OKF/viewer race tests passed;
  additional focused tests, vet, and build passed. Full review remains open.

- Responsive editor browser check: current-worktree server, 375×812 viewport.
  Screenshot inspection showed the profile dialog contained within the viewport
  with vertical scrolling. Invalid Advanced JSON set the form's inert state;
  valid sparse JSON rehydrated Name and cleared inert after preview. Cancel
  discarded the draft without Save/Bind or any configuration write. The live
  accessibility tree exposed an unnamed Advanced JSON textarea; added its
  `Advanced profile JSON` accessible label. Bootstrap syntax and diff checks
  passed. Browser override reset, temporary tab closed, server interrupted.
  This is agent inspection, not the required human visual approval. The form's
  declared-state visibility option still needs its backend behavior audited.

- Current-worktree browser smoke verification: launched the local Go project
  viewer with the in-process analyzer on an ephemeral loopback port. The default
  page loaded the architecture graph and the optional OKF navigation link.
  Following it loaded Neutral with 17 visible concepts, grouped diagnostics,
  and containment-only edges. Selecting Code quality and code intelligence
  loaded Markdown detail and the new expandable state/containment/link/provenance
  sections. Double-click focus exposed project/concept breadcrumbs; Back
  restored Top level. No captured console errors appeared during page startup.
  These observations came from live DOM interaction, not screenshot approval;
  they do not prove responsive layout, wheel/drag behavior, or the full matrix.
  The temporary browser tab was closed and the local server was interrupted.

- Consolidated verification after profile lifecycle, cancellation, detail, and
  layout recovery remediation: `go test ./... -count=1`, `go test -race ./...`,
  `go vet ./...`, and `go build ./...` passed. All 27 browser test files passed.
  All 39 changed/new JavaScript files passed syntax checks. A fresh audit of
  135 changed/new OKF/viewer/shared-document files found none at or above 600
  lines. Diff checks passed. Both Go test jobs completed; no verification
  process remains live. These gates validate the current code but do not prove
  the outstanding full SC-001–SC-020 journey matrix, current visual review,
  hidden-subtree recovery, top-level navigation, or final architecture audit.
  The verdict remains not ready to commit; no issue or capability was closed.

- SC-019 layout recovery follow-up: an ELK failure always replaced an existing
  calculated layout with fallback placement. The OKF scene adapter now retains
  its last rendered layout when node and containment-relationship data match
  exactly. Changed scenes still use safe fallback geometry rather than stale
  positions. The status distinguishes retained versus fallback layout, and
  superseded failures cannot overwrite current status. A container-scoped weak
  cache owns this renderer-local state without persisting it or changing ELK
  behavior. Tests cover successful layout followed by failure, changed nodes,
  and stale failure publication; all 27 browser test files, bootstrap syntax,
  and diff checks passed. Full cross-surface/visual acceptance remains open.

- Detail provenance follow-up: hierarchy normalization diagnostics were all
  relabelled with the inspected concept ID, falsely attributing unrelated
  conflicts to that concept. Detail now retains the diagnostic's original
  concept ownership. Its provenance includes the selected concept's incoming
  and outgoing normalized containment proofs, preserving explicit parent/child
  agreement and filesystem fallback alongside original document provenance.
  Duplicate normalized proofs are omitted without changing source data.
  Tests cover unrelated conflict ownership, both agreeing explicit claims,
  fallback, and source immutability. Interaction/application/viewer race tests,
  vet, build, and diff checks passed. This fills an SC-006/SC-012 detail gap;
  it is not full journey or visual acceptance evidence.

- Acceptance audit, SC-008/SC-012: ConceptDetail carried declared/effective
  state, containment, resolved semantic links, and provenance, but renderDetail
  omitted all of them. Added compact expandable source-backed sections, with
  escaped data and no new CSS or planning-specific state/type interpretation.
  Frontmatter is not reintroduced through this path, preserving the profile's
  hidden-unknown-metadata behavior. Markup tests cover differing states, facts,
  escaped unsafe link targets, absent sections, and hidden unknown fields.
  Focused markup tests, syntax, and diff checks passed. The issue 071 broad
  cross-surface diagnostic/status checkbox is reopened rather than treating
  API coverage as browser proof. SC-010 hidden-subtree recovery and SC-011
  previous-context breadcrumb evidence still need resolution. Human visual
  gates and the full acceptance matrix remain open.

- Projection timeout follow-up: Build and detail had collapsed deadline errors
  into cancellation, while source indexing used a 409 status for timeouts.
  A shared domain error mapper now preserves the context cause and maps timeouts
  to 504 and cancellations to 409 for application, projection, detail, and source
  indexing. Build also checks cancellation before returning its snapshot.
  Tests verify exact error code/status/cause and no publishable snapshot for
  pre-cancelled and expired contexts. All OKF/viewer race tests passed before
  the final source mapper reuse; all OKF tests and diff checks passed afterward.
  This consolidates error semantics without changing response fields. Final
  acceptance and architecture review remain incomplete.

- Source cancellation follow-up: indexing no longer continues revision/link
  processing after a cancelled walk, and link resolution checks cancellation
  between concepts and individual links. Discovery distinguishes deadline expiry
  from cancellation. Index cancellation takes precedence over incidental source
  errors in an incomplete read, rather than misreporting an invalid bundle.
  Focused source/application race tests, vet, build, and diff checks passed.
  The three changed Go files are 139, 243, and 74 lines. Full
  `go test ./... -count=1` also completed successfully; no job remains live.
  Top-level breadcrumb behavior remains unchanged: the existing
  focus contract requires a real concept ID and does not define a root-reset
  command. Broader review and human visual gates remain open.

- Configuration editor lifecycle follow-up: P2, Bind/Rename/Delete could publish
  late responses into a reopened editor or newer bundle/profile selection.
  Their shared operation guard captures editor content, selection, and revision;
  suppresses overlapping configuration writes; and checks again after catalog
  loading. Known newer revisions are retained. Rename now honors cancellation
  of its display-name prompt. Save also checks bundle selection. Both paths
  distinguish successful persistence followed by refresh failure from write
  failure. Tests cover supersession, duplicate operations, revision retention,
  refresh failure, Save As conflict and success. All 27 browser test files
  passed; the final Save As cases also passed separately. Five changed/new JS
  files passed syntax checks and remain below 600 lines. Diff checks passed.
  The focused shared guard removes duplicated response-publication behavior
  from bootstrap. This does not establish live-browser acceptance or complete
  the overall architecture/acceptance review.

- Profile save supersession follow-up: P2, late validation/write/catalog responses
  could save a closed draft or close a newly opened/edited draft. Extracted Save
  and Save As orchestration into `okf_profile_save.js`, capturing the declaration,
  JSON input, selected profile, and revision before awaits. Superseded validation
  does not write; completed writes preserve newer edits and do not overwrite a
  newer known configuration revision. Duplicate Save submissions are suppressed.
  Layout Save also checks that its editor remains open after inherited preview.
  Tests execute the actual save flow for close during validation, edits during
  write, reopening during catalog reload, duplicate submission, and success.
  All 26 browser test files passed, bootstrap syntax and diff checks passed.
  The extraction narrows bootstrap responsibility and adds no transport contract.
  Remaining lifecycle operations and full acceptance review are still open.

- Advanced JSON edit safety follow-up: P2, invalid JSON left the previous form
  editable, allowing a form change to silently discard unfinished JSON. The
  shared editor-preview module now owns JSON input transitions and locks the
  stale form until a valid declaration resolves. Tests cover malformed and
  non-object JSON, retained input/declaration, valid recovery, and a superseded
  pending response that must not unlock the stale form. All 25 browser test
  files passed; the added supersession case also passed separately. Changed
  runtime modules passed syntax checks and diff checks passed. This extraction
  narrows bootstrap orchestration without adding a new boundary. Broader
  acceptance and architecture review remain incomplete.

- Consolidated post-remediation verification: full `go test ./... -count=1`,
  full `go test -race ./...`, `go vet ./...`, `go build ./...`, all 25 browser
  test files, and diff checks passed. All 35 changed/new JS files passed syntax
  checks. Line-count audit covered 127 changed/new files under OKF, viewer, and
  shared-document packages; none reached 600 lines. All jobs completed.
  Issue 071's all-surfaces verification checkbox was reopened: its wording
  includes declared `when-supported` surfaces, but the complete end-to-end
  matrix remains open. Its completion language was qualified without changing
  capability state or converting historical visual inspection into current
  evidence. Final acceptance and architecture audits remain incomplete.

- Profile map-key follow-up: P2, direct assignment to plain objects lost the
  valid `__proto__` mapping key or changed a token dictionary's prototype;
  duplicate rows silently overwrote earlier rows. One form helper now defines
  ordinary own properties and rejects duplicate keys for state mappings,
  state-token mappings, and tokens. Regression tests exercise actual form
  serialization, JSON round trips, unchanged prototypes, and duplicate rejection
  for all three dictionaries. All 25 browser test files, form syntax check, and
  diff checks passed. Final repo-wide gates and visual review remain pending.

- Numeric control validation follow-up: P2, parseInt/parseFloat silently changed
  invalid or fractional entries before backend validation. A shared numeric-input
  reader now checks complete finite values, safe integers where required, browser
  validity, and range bounds. Profile fields, repeat rows, decorations, and the
  depth toolbar use it. Errors are shown directly instead of being mislabeled as
  invalid rule JSON. Form tests now import the real module and dependencies rather
  than rewriting its imports. Numeric and form regression tests cover fractions,
  malformed values, bounds, browser bad input, and valid scientific/decimal input.
  All 24 browser test files, bootstrap syntax check, and diff checks passed;
  focused numeric/form/preview tests passed again after decoration validation.
  Final repo-wide gates and human visual review remain pending.

- Inherited form display remediation: existing ValidateProfile HTTP route now
  returns separate declaration and effective draft through focused application
  PreviewProfile. The preview serializes with configuration changes and does not
  publish or persist the draft. A separate profile-validation HTTP file keeps the
  existing handler below 600 lines. Form opening, committed base-list changes,
  and valid Advanced JSON changes request server composition. Generation and
  declaration checks reject stale responses; unresolved previews block form
  interaction and saves. Effective controls retain declaration-based change
  tracking. Edited replacement maps retain other inherited entries.
  Backend and HTTP tests verify effective depth versus omitted declaration and
  unchanged session/configuration. Browser tests verify effective display,
  omission preservation, explicit edits, complete replacement maps, stale preview
  suppression, and preview failure safety. Full Go tests, OKF/viewer race tests,
  all 23 browser test files, syntax checks, vet, build, and diff checks passed.
  All twelve changed/new code files are below 600 lines. Human visual review is
  still pending; broader review is not clean. No verification process is running.

- Ordered base composition follow-up: P1, normalizing each base independently
  inserted defaults before sibling composition. A later base containing only
  a node field reset an earlier base's explicit layout, limits, hierarchy, and
  token settings. `TestMultipleBasesApplyDefaultsAfterComposition` reproduced
  the regression before remediation. Recursive resolution now composes sparse
  declarations; defaults and effective validation run at the requested profile
  boundary. The regression verifies preserved earlier settings and an explicit
  later depth override. All OKF/viewer race tests and diff checks passed.
  Changed files are 595 and 154 lines. This resolves premature normalization,
  not the separate frontend inherited-value display item.
  Full Go tests, all 22 browser test files, vet, and build passed. All verification
  processes from this pass have completed; broader review remains incomplete.

- Source text preservation follow-up: P2, parsing normalized CRLF throughout the
  Markdown body and accepted invalid UTF-8 bytes outside frontmatter, allowing
  later JSON encoding to replace source content silently. Parsing now preserves
  body line endings and rejects invalid UTF-8 with a validation error. Tests
  compare LF/CRLF bodies containing Greek, Japanese and euro characters exactly
  and reject an invalid body byte. All OKF/viewer race tests and diff checks pass.
  This concerns stored source text; rendered CommonMark may still normalize
  presentation as expected. Final repo-wide gates remain pending.

- Source diagnostic limit follow-up: P2, incremental collection silently dropped
  overflow, making the existing truncation notice unreachable on that path and
  allowing a late error to be discarded after warnings. The bounded collector
  now emits the omission notice and preserves error evidence, including an error
  in the slot displaced by the notice. Bulk bounding uses the same collector.
  Unit coverage places errors before, at, and after the limit; a real source
  fixture with more than the limit of unresolved links verifies a selectable
  warning-only bundle with visible truncation. Source/application/viewer race
  tests and diff checks passed. Changed files are 239 and 56 lines. Final
  repo-wide gates remain pending; no verification process remains running.

- Diagnostic read follow-up: P1, GET diagnostics called Refresh on every read,
  invalidating session snapshots and picking up source changes as a side effect
  of opening a report. It also omitted bundle/index-local diagnostic evidence.
  The handler now initializes through the existing profile read if needed, then
  reads the published catalog and indexed summaries, including invalid candidate
  diagnostics. `TestDiagnosticReadPreservesPublishedSession` edits a source after
  projection and verifies that reading the report retains the published session
  unchanged while returning its boundary warning and invalid-bundle diagnostics.
  Viewer race tests passed. Explicit Refresh remains the source-update operation.
  OKF/viewer package tests and diff checks passed. Changed files are 597 and 36
  lines. No verification process remains running; final repo-wide gates are pending.

- Legend style follow-up: P2, state-token legend entries inherited the first
  visible node's root/roll-up decoration and rule-selected shape. A rolled-up
  implemented root could make the implemented legend violet rather than green.
  Legend fill and shape now come from the profile's base token, while node
  decorations remain unchanged. The structural-decoration test checks the
  implemented legend color and stability when focusing a child as the new root.
  Projection race tests passed. This is token-style correction, not a claim
  that the entire legend/detail acceptance surface has been reviewed.
  All OKF/viewer package tests and diff checks passed. Changed files are 467 and
  180 lines. Final repo-wide gates remain pending after the remaining review.

- Disk-profile validation follow-up: P1, resolved profiles skipped rule parameter
  validation and could be advertised as valid after manual configuration edits.
  The application also ignored `ProfileInvalid` when choosing neutral fallback.
  Resolution now includes strategy parameter diagnostics, outside registry locks;
  candidate resolution copies the strategy map instead of holding the parent
  registry lock while validating. Invalid profiles project through Neutral, with
  the session reporting the effective fallback profile and diagnostics preserving
  the requested invalid profile's identity. Details also respects invalid status.
  `TestInvalidDiskProfileRemainsRepairableWithNeutralProjection` loads malformed
  rule parameters from disk and verifies invalid catalog status, neutral fallback,
  diagnostics, and byte-for-byte preservation of the repairable configuration.
  OKF/viewer race tests passed. Unknown strategies retain the existing partial
  application behavior; this fix concerns invalid profiles, not unavailable rules.
  Full Go tests, vet, build, and diff checks passed. All five changed files remain
  below 600 lines; registry.go is 592 lines. No verification process remains running.

- Back availability follow-up: P2, browser enabled Back from nonempty ancestry
  breadcrumbs rather than session navigation history. Navigation now carries
  `can_go_back` derived from history, including both the session read model and
  its projection; browser uses that flag. `TestBackAvailabilityReflectsHistoryNotAncestry`
  covers pushed history, consumed history, and restored focus without history.
  The canonical projection example includes the additive boolean field.
  OKF/viewer race tests, all 22 browser test files, syntax check, vet, build,
  and diff checks passed. Top-level breadcrumb navigation remains a separate
  open item: FocusSubtree currently requires a concept in the selected bundle.
  Full Go tests passed. All five changed code files are below 600 lines.
  No verification process remains running from this pass.

- Detail request UI follow-up: P1, selecting a new concept retained the previous
  detail body during loading and after failure, under a stuck loading title.
  Extracted the existing selection/detail lifecycle into `okf_detail.js` and
  clear stale detail immediately. Loading and failure notices are escaped and
  displayed in the detail pane. Request-generation checks still suppress stale
  successes and failures. Regression tests exercise the real selection function,
  immediate clearing, escaping, both stale response paths, and unchanged layout
  and viewport references. No layout invocation or CSS duplication was added.
  All 22 browser test files, changed runtime JS syntax checks, vet, build, and
  diff checks passed. The changed files have 483, 28, and 31 lines. This focused
  extraction separates detail-request lifecycle from bootstrap composition;
  broader bootstrap responsibility review remains open.
  Full Go tests completed successfully. The existing `web/app/*.js` embed
  includes the extracted module. No verification process remains running.

- Refresh publication follow-up: P1, cancellation during configuration loading
  could still replace indexes/catalog/config and invalidate all sessions because
  no context check guarded publication. Refresh now checks cancellation while
  holding the publication lock before changing shared state.
  `TestCancelledRefreshPreservesPublishedCatalogAndSession` cancels after a
  successful configuration load and verifies catalog, index, and session remain
  unchanged. `TestRefreshAllowsRecoveryFromDisappearingFocusAndInvalidBundle`
  verifies a missing focus is reported, Back recovers, and another valid bundle
  stays selectable and isolated after the first becomes invalid (SC-020 backend
  evidence). No automatic bundle/focus substitution was introduced. Application
  and viewer race tests and diff checks passed. Browser recovery presentation
  still needs its separate review; this is not human visual acceptance.
  Full Go tests, vet, and build passed. Changed files are 423 and 92 lines.
  All verification processes from this pass have completed.

- Rule matching follow-up: P1, exact/visibility rules compared `fmt.Sprint`
  values, conflating missing and null metadata, strings and booleans/numbers,
  and collections with their textual representation. Contains rules searched
  printed collections, allowing `unimplemented` to match `implemented`.
  Exact matching now requires field/value presence and typed JSON-value equality;
  equivalent YAML/JSON numeric values still compare equally. Contains uses whole
  value membership for collections and preserves case-insensitive substring
  matching for string fields. Maps/scalars are not treated as text collections.
  Regression tests run through registry evaluation for both exact and visibility
  rules and cover collection/string semantics. OKF/viewer race tests and diff
  checks passed. This resolves the matching-semantics item from the parameter
  validation pass; extension catalog schema metadata remains under review.
  Full Go tests, vet and build also passed. Files changed in this pass have
  137, 39, and 58 lines. No verification process remains running.

- Rule parameter validation follow-up: P1, profile validation checked strategy
  availability but accepted malformed parameters, contrary to the canonical
  profile invariant requiring supported parameter schemas. Built-in strategies
  now implement focused `ports.RuleParameterValidator`; profile validation asks
  the selected strategy to validate cloned parameters and reports
  `okf_rule_invalid` on failure. Validator panics are contained. This keeps rule
  schema ownership in strategies rather than adding a registry switch over IDs.
  Valid/invalid cases cover all five built-ins, validator mutation/panic isolation,
  and rejected Save As preserving existing configuration bytes. OKF/viewer race
  tests, vet, build, and diff checks passed. Changed files are below 600 lines;
  registry.go is 587 lines. Rule evaluation matching semantics and extension
  catalog schema metadata remain separate review items.
  Full `go test ./... -count=1` also completed successfully. No verification
  process remains running from this pass.

- Rule/roll-up detail follow-up: P1, details ignored rule-derived effective state
  and structural roll-ups, while graph roll-ups ignored rule overrides on children
  outside the visible depth/budget. This could change the meaning of a parent
  merely by navigating. Shared profile state interpretation now handles declared
  state, rule overrides, structural role, and direct-child roll-up. Both graph
  and details call it. Hidden direct-child rules are evaluated only for explicit
  roll-ups and reused when already evaluated; their nodes remain hidden.
  `TestDetailAndGraphShareRuleAndRollupStates` checks depth-independent state,
  graph/detail agreement and source immutability. Cancellation during a hidden
  dependency is covered by `TestCancellationDuringHiddenStateDependency`.
  All OKF/viewer race tests, all 21 browser test files, vet, build, and diff checks
  passed. This resolves the earlier source-state pass's remaining rule/roll-up
  consistency item. Architecture delta for this extraction deepens the profile
  interpretation boundary: two consumers share one calculation instead of
  independently implementing state policy. Broader review remains incomplete.
  Details receives the focused `ports.RuleEvaluator` interface, excluding
  registry configuration operations. Additional fallback coverage checks disabled
  roll-up, ordinary roles, unknown children, and conflicting child states.
  Full Go tests and the repository-wide race run completed successfully. After
  narrowing the evaluator interface, focused OKF/viewer race tests, vet, build,
  and diff checks passed again. Files changed in this pass remain below 600 lines.
  No verification process remains running.

- Detail source-state follow-up: P2, detail used literal frontmatter keys while
  projection accepted prefixed profile sources, yielding `unknown` for the same
  concept. Shared `profile.DeclaredState` and `profile.MappedState` now own source
  lookup and vocabulary mapping for both paths. Blank/non-string source values
  consistently produce `unknown`; string values are trimmed before mapping.
  `TestDetailAndProjectionUseSameSourceState` compares actual detail and projection
  output for plain, dot-prefixed, colon-prefixed, and case-insensitive prefixes,
  including whitespace, missing and non-string values. Focused interaction,
  projection, profile, and application race tests and diff checks passed.
  This pass established source-state consistency only; the rule/roll-up
  follow-up above subsequently addressed derived-state consistency.
  Follow-up full `go test ./... -count=1`, `go vet ./...`, and `go build ./...`
  passed. All four files changed in this source-state pass remain below 600 lines
  (36, 526, 90, and 101). No verification process remains running from this pass.

- Rule annotation follow-up: verified priority handling per annotation key,
  preservation of independent annotations, and neutral fallback for conflicting
  equal-priority values without resurrection by subsequent equal-priority rules.
  Fixed nondeterministic conflict diagnostic ordering by sorting annotation keys
  before merging. `TestAnnotationPriorityAndConflicts` and
  `TestAnnotationConflictsHaveStableOrder` cover these behaviors.
  `go test -race ./internal/okf/profile ./internal/okf/projection` and
  `git -c core.safecrlf=false diff --check` passed. The changed registry is
  585 lines. These focused checks do not replace the final repo-wide gates.

- Latest consolidated gate run after renderer/profile/HTTP fixes: full
  `go test ./... -count=1`, `go test -race ./...`, `go vet ./...`, and
  `go build ./...` passed. All 21 browser test files passed. All 28 changed/new
  JavaScript files passed `node --check`. The 99 changed/new files under OKF,
  viewer, and shared-document packages were audited: none reached 600 lines.
  Diff checks passed. Both test jobs completed; no live verification job is
  assumed. Architecture/acceptance review remains incomplete, so this is not
  a clean-review or commit-readiness verdict.

- After frontmatter remediation: `go test ./... -count=1`, `go vet ./...`,
  `go build ./...`, all 20 browser test files, OKF-package race tests, and
  `git diff --check` passed. This is automated evidence, not visual acceptance.

- Follow-up CommonMark pass: full `go test ./... -count=1`, `go vet ./...`,
  `go build ./...`, and diff checks passed. The subsequent bounded source-read
  change has focused source tests; final repo-wide gates remain pending.

- `go test ./... -count=1`: passed after backend remediation.
- `go test -race ./...`: passed after backend remediation.
- `go vet ./...` and `go build ./...`: passed after backend remediation.
- Browser test suite: 20 passed after the browser race corrections; these tests
  do not establish the pending live interaction behavior.
- `node --check internal/viewer/web/app/okf_bootstrap.js`: passed after those corrections.
- `git diff --check`: passed.
- Changed/new OKF/viewer/shared-document file line audit: no file at or above 600
  lines at the last audit. Application navigation was split out of service.go.
- No live server, terminal job, or browser verification is assumed from earlier
  conversation. All jobs started during this pass have completed.
