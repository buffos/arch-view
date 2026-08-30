# Update Log

## 2026-08-30

### Deterministic quality visual review and capability closeout

* **Review:** Inspected the project-backed quality viewer at the normal desktop
  viewport and at 390×844. Verified compact summaries, exact/signal wording,
  coverage states, evidence actions, the complete rule catalog, focus styling,
  Back navigation, responsive layout, and absence of page-level horizontal
  overflow.
* **Correction:** Fixed the affected-file control after live review showed it
  broadened file queries by unioning module and report selectors. Aggregate
  quality evaluation now uses the source projection identity, and affected-only
  queries send exact report file subjects. A live rerun returned 23 of 23
  affected repository files with no unrelated entries.
* **Verification:** Added regression coverage for aggregate projection identity,
  immutable quality-query results, canonical ordering/scoped coverage, selector
  validation, exclusive profile/baseline writes, and affected-file query shape.
* **Closeout:** Archived issue 063 and advanced Deterministic quality checks from
  `specified` to `implemented`. The Code quality and code intelligence roll-up
  remains `specified` because Live analysis and MCP remains specified.

### Deterministic quality implementation-gap closure

* **SOLID producer:** Connected the Go source extractor and observation builder
  to `source:solid.structure`, so the five registered SOLID signals receive
  real syntax-observable structural facts instead of unsupported coverage on
  valid Go analysis.
* **Baseline workflow:** Added `arch-view quality baseline` to create a
  validated separate baseline document from selected finding IDs/keys or all
  active findings, with explicit overwrite protection.
* **Verification:** Added extractor, Go analyzer end-to-end, and CLI tests;
  a real `quality-profiles/full.json` repository run reports observed coverage
  for SRP, OCP, LSP, ISP, and DIP.
* **Synchronization:** Updated issue completion evidence, the deterministic
  quality gap analysis, canonical CLI contract, application PRD, and
  application architecture summary. No capability-state transition was made;
  issue 063 remains at its final visual-review gate.

### Deterministic quality checks issues 059–063 implementation batch

* **Implementation:** Delivered deterministic report lifecycle and baseline
  matching, conservative SOLID structural signals, bounded quality queries and
  evidence, headless quality-profile/exit-policy support, JSON/HTML/SVG quality
  projections, and the viewer quality summary/affected-file filter.
* **Boundaries:** Reports preserve stable finding keys, report-local IDs,
  exact-version suppression, provider failures, explicit coverage states, and
  canonical scope/provenance. Source context remains an explicit bounded
  read-only request; the viewer does not re-evaluate file thresholds.
* **Verification:** `go test ./... -count=1`, `go test -race ./...`,
  `go vet ./...`, `go build ./...`, all viewer JavaScript syntax checks and
  tests, and `git diff --check` pass.
* **OKF:** strict bundle conformance validation passes with no issues.
* **Closeout:** Issues 059–062 are verified and archived. Issue 063 is
  implemented and remains `awaiting-human-review` for the final desktop and
  responsive viewer inspection; the deterministic-quality capability remains
  `specified` and no capability-state transition was made.
* **Synchronization:** Updated the quality capability, orchestration status,
  issue registry, project/index records, and issue completion evidence. No
  commit was created.

## 2026-08-30

### Deterministic quality checks issues 053–058 delivery and closeout

* **Implementation:** Delivered the neutral `arch-view.quality/v1` profile,
  additive metric/rule catalog, strict validation, optional report attachment,
  source file/callable size rules, Go callable body/complexity/nesting metric
  providers, documentation coverage, graph coupling/cycle rules, and explicit
  forbidden-dependency/layer-direction constraints.
* **Evidence:** Preserved scope-qualified provenance, explicit unknown/
  unsupported/not-evaluable coverage, hash-linked callable body spans, exact
  reported graph relationships, canonical cycle evidence, and legacy result
  compatibility. Focused and full Go tests pass, including the race suite;
  vet, build, viewer JavaScript syntax checks, `git diff --check`, and strict
  OKF validation also pass.
* **Closeout:** Issues 053–058 are verified and archived. Issue 059 is now
  unblocked; issues 059–063 remain active. The deterministic-quality child
  remains `specified` because its remaining lifecycle, signal, query, export,
  and viewer slices are not yet delivered.
* **Synchronization:** Updated the quality/source-facts canonical artifacts,
  application synthesis, OKF capability/project/index records, registry, issue
  blockers, and completion records. No commit was created.

### Deterministic quality checks delivery slicing

* **Approval:** The user approved eleven dependency-ordered implementation
  slices for the specified Deterministic quality checks capability.
* **Delivery:** Created ready-for-agent issues 053–063 for the quality profile
  and file-size tracer bullet, callable metrics, documentation coverage, graph
  rules, explicit constraints, report lifecycle/baselines, SOLID signals,
  bounded queries, headless/export projections, and the viewer projection.
* **Artifact impact:** Product and architecture truth remain unchanged; the
  existing application synthesis already covers this quality boundary. Updated
  the owning capability references, orchestration status, issue registry, and
  planning frontier.
* **State:** The deterministic-quality child remains `specified`; no topology
  or parent roll-up state transition occurred. State totals remain 0 `foggy`,
  0 `bounded`, 4 `specified`, and 14 `implemented`.

### Source facts and symbol index issue 052 closeout

* **Closeout:** Archived issue 052 after the user approved its final
  visual-review gate. Issues 048–052 are now verified, archived, and approved.
* **State:** Advanced [Source facts and symbol index](capabilities/code-quality-and-intelligence/source-facts-and-symbol-index.md)
  from `specified` to `implemented`.
* **Roll-up:** Recomputed [Code quality and code intelligence](capabilities/code-quality-and-intelligence.md)
  as `specified`, unchanged because its deterministic-quality and
  live-analysis/MCP children remain specified. State totals are now 0 `foggy`,
  0 `bounded`, 4 `specified`, and 14 `implemented`.
* **Synchronization:** Updated the source-facts orchestration status,
  application PRD, application architecture summary, OKF project/index
  records, and active issue registry.

### File line-threshold human projection requirement

* **Specification:** Added the explicit `source:file.max-lines` human-projection
  requirement: count files over the configured limit, offer an affected-files
  filter, preserve scope/report coverage, and never re-evaluate raw file facts
  in the consumer.
* **Synchronization:** Updated the deterministic-quality capability, PRD,
  acceptance scenario, use-case, contract, gap analysis, readiness review, and
  application product/architecture summaries.
* **Delivery:** No implementation issue was created. Issue 052 remains
  `awaiting-human-review`; the threshold/filter implementation is specified but
  not scheduled in the delivery registry.

### Symbols filter and documentation lookup correction

* **Implementation:** Fixed inspection Symbols filter focus loss and empty
  result controls, added extractor-reported `language_kind` filtering, and
  changed documentation status loading to use bounded repeated subject IDs.
* **Compatibility:** The HTTP and embedded source-index adapters preserve the
  existing response envelope and support repeated `subject_id` values; the
  legacy singular query remains supported.
* **Verification:** Focused Go/JavaScript tests and a 1440×1000 local Chrome
  smoke check passed. Issue 052 remains `awaiting-human-review`.

## 2026-08-29

### Source facts and symbol index delivery implementation

* **Implementation:** Completed issues 048–051: the versioned optional
  `SourceIndex` attachment, deterministic file facts, registered Go
  declarations/documentation extraction, scope-safe canonical and aggregate
  propagation, structural queries, and bounded read-only evidence.
* **Viewer:** Implemented issue 052's module source-facts projection with
  explicit containment, scope qualification, distinct coverage states,
  provenance, and opt-in bounded source context. Automated HTTP, JavaScript,
  full-test, race, vet, and build gates pass.
* **Closeout:** Archived issues 048–051 and removed them from the active issue
  registry. Issue 052 remains active as `awaiting-human-review` because its
  required populated/partial/unsupported/combined/legacy visual inspection
  has not yet been approved.
* **State:** The source-facts capability remains `specified` until the 052
  visual gate is approved; the code-quality roll-up and its other children are
  unchanged.

## 2026-08-29

### Source facts and symbol index delivery slicing

* **Approval:** The user approved five dependency-ordered delivery slices for
  the specified Source facts and symbol index capability.
* **Delivery:** Created ready-for-agent issues 048–051 for the versioned
  source-index/file-fact contract, registered Go extraction, scope-safe
  canonical/aggregate attachment, and bounded structural queries. Created
  issue 052 for the module-inspection viewer workflow with a required
  `visual-review` gate.
* **Graph/frontier:** Linked all five pending issue paths from the source-facts
  capability. The node remains `specified`; no topology or state transition
  occurred. The code-quality roll-up remains `specified` because its other
  structural children remain specified.
* **Artifact impact:** Delivery truth changed. The application PRD, application
  architecture summary, and exact-spec content remain unchanged because the
  approved batch implements their existing synchronized scope. Capability
  ownership remains unchanged; the source-facts node now carries the five
  delivery references. The source-facts orchestration status, issue registry,
  max issue ID, and planning frontier references were updated.
* **Scope note:** Initial delivery is Go-first as permitted by the exact
  specification. Python, TypeScript, Rust, and Clojure extractor parity remain
  later additive work; unsupported or unknown coverage must remain explicit.

## 2026-08-29

### Code quality and code intelligence roll-up state

* **State transition:** Moved [Code quality and code intelligence](capabilities/code-quality-and-intelligence.md) from `bounded` to a pure structural-child roll-up with materialized state `specified`, using `state_policy.mode: rollup`, `source: structural_children`, and `reducer: min`. Its effective state is the minimum of its three specified children.
* **Routing:** The parent has no standalone PRD, issue batch, or implementation scope. Future specification and delivery work starts at the least-mature structural child; the parent becomes effectively `implemented` only when every structural child is implemented and no parent-only scope exists.
* **Artifact sync:** Updated the parent concept, planning index, project concept, application PRD, and application architecture summary. Existing own-state parents with their own PRDs and scopes were left unchanged. Product and architecture behavior are unchanged.
* **Progress:** State totals are now 0 `foggy`, 0 `bounded`, 5 `specified`, and 13 `implemented`.

## 2026-08-29

### Source facts and symbol index bounded

* **State transition:** `foggy -> bounded` for [Source facts and symbol index](capabilities/code-quality-and-intelligence/source-facts-and-symbol-index.md).
* **Boundary:** Clarified a scope-first `SourceIndex` sibling with first-class files, symbols, documentation, spans, provenance, containment, and extensible relations.
* **Extensibility:** Confirmed opaque IDs, open vocabularies, registered extractors, explicit unsupported/unknown/partial coverage, deterministic snapshots, and typed extension blocks.
* **Separation:** Kept quality thresholds, semantic quality rules, live watching, MCP transport, search ranking, and remediation outside this capability.
* **Progress:** State totals after transition are 2 `foggy`, 2 `bounded`, 1 `specified`, and 13 `implemented`.

## 2026-08-29

### Source facts and symbol index specified

* **State transition:** `bounded -> specified` for [Source facts and symbol index](capabilities/code-quality-and-intelligence/source-facts-and-symbol-index.md).
* **Specification:** Added the complete exact-spec set: discovery/gap analysis, glossary, PRD, domain model, use cases, source-index contract, acceptance scenarios, readiness review, and orchestration status.
* **Synthesis:** Synchronized the application PRD and architecture summary with the optional `SourceIndex` attachment, scope-first authority, extractor registry, provenance/coverage, opaque IDs, deterministic digest, and compact query boundary.
* **Readiness:** No High or Medium specification findings remain. Implementation and language coverage are future delivery work; quality policy and live/MCP remain foggy children.
* **Progress:** State totals after transition are 2 `foggy`, 1 `bounded`, 2 `specified`, and 13 `implemented`.

## 2026-08-29

### Deterministic quality checks bounded

* **State transition:** `foggy -> bounded` for [Deterministic quality checks](capabilities/code-quality-and-intelligence/deterministic-quality-checks.md).
* **Boundary:** The child now owns versioned metrics, configurable deterministic rules, findings, evidence, baselines, and severity; source facts remain owned by the specified sibling.
* **Decision:** Static SOLID output is explicitly an evidence-backed signal, never a proven violation. Rule/metric providers are registered strategies so new rules do not require central branching.
* **Progress:** State totals after transition are 1 `foggy`, 2 `bounded`, 2 `specified`, and 13 `implemented`.

## 2026-08-29

### Deterministic quality checks specified

* **State transition:** `bounded -> specified` for [Deterministic quality checks](capabilities/code-quality-and-intelligence/deterministic-quality-checks.md).
* **Specification:** Added the exact metric/rule/finding profile, domain, use-case, contract, acceptance, readiness, and orchestration artifacts.
* **Design boundary:** Exact threshold/graph findings are separated from advisory SOLID signals; versioned registries, formula identity, coverage, evidence, and baseline matching keep future extensions additive and reproducible.
* **Synthesis:** Refreshed the application PRD and architecture summary to make quality a future report sibling consuming SourceIndex/model facts; live/MCP remains foggy.
* **Progress:** State totals after transition are 1 `foggy`, 1 `bounded`, 3 `specified`, and 13 `implemented`.

## 2026-08-29

### Live analysis and MCP bounded

* **State transition:** `foggy -> bounded` for [Live analysis and MCP](capabilities/code-quality-and-intelligence/live-analysis-and-mcp.md).
* **Boundary:** The child now owns configured-root watching, event coalescing, selective/full reanalysis orchestration, immutable snapshot revisions, compact structural search, quality-report retrieval, and MCP transport/permissions.
* **Separation:** Watchers do not parse; MCP does not own language semantics, quality policy, or source edits. A fix handoff is explicit and downstream.
* **Progress:** State totals after transition are 0 `foggy`, 2 `bounded`, 3 `specified`, and 13 `implemented`.

## 2026-08-29

### Live analysis and MCP specified

* **State transition:** `bounded -> specified` for [Live analysis and MCP](capabilities/code-quality-and-intelligence/live-analysis-and-mcp.md).
* **Specification:** Added the exact watcher/revision/query/MCP set: lifecycle, event normalization, overflow recovery, conservative invalidation, atomic publication, last-ready retention, structural search, quality/evidence tools, budgets/cursors, transport, root safety, and read-only permissions.
* **Synthesis:** Refreshed the application PRD and architecture summary to make live analysis the shared downstream read surface for source facts and quality reports; no autonomous remediation is included.
* **Readiness:** No High or Medium specification findings remain. Implementation and packaging verification remain future delivery work.
* **Progress:** State totals after transition are 0 `foggy`, 1 `bounded`, 4 `specified`, and 13 `implemented`.

## 2026-08-29

### Code quality and code intelligence topology

* **Topology:** Added the bounded [Code quality and code intelligence](capabilities/code-quality-and-intelligence.md) capability with three foggy children: [Source facts and symbol index](capabilities/code-quality-and-intelligence/source-facts-and-symbol-index.md), [Deterministic quality checks](capabilities/code-quality-and-intelligence/deterministic-quality-checks.md), and [Live analysis and MCP](capabilities/code-quality-and-intelligence/live-analysis-and-mcp.md).
* **Target:** Recorded the future direction for first-class files, documentation, declarations, deterministic metrics/findings, configurable quality rules, live snapshots, structural code search, and MCP reporting.
* **Boundary:** Kept subjective architectural approval, runtime tracing, target-code execution, and implicit source mutation outside the deterministic product scope; SOLID output is reserved for explicitly labeled signals.
* **Artifact sync:** Updated the project concept, planning indexes, application PRD, and application architecture summary. No delivery issues or exact-spec artifacts were created because the new children remain foggy.
* **Progress:** Current capability totals are 3 `foggy`, 1 `bounded`, 1 `specified`, and 13 `implemented`.

## 2026-08-29

### Project analyzer assignments completion and graph update

* **Review:** The user approved the declared visual review for issue 047 after
  inspecting the configured mixed-language viewer journey.
* **State transitions:** Advanced `Project analyzer assignments and view
  selection`, `Analyzer plugin runtime`, and `Analyze source code` from
  `specified` to `implemented` after issues 044–047 were completed, verified,
  and archived. Current totals are 0 `foggy`, 0 `bounded`, 1 `specified`, and
  13 `implemented`.
* **Artifact sync:** Updated affected capability nodes, parent roll-ups,
  implementation/orchestration statuses, application planning references,
  issue registry, and issue paths. The Advanced ELK renderer support node is
  the remaining specified frontier; no topology was added or split.
* **Verification:** Strict OKF validation, full Go tests, race tests, vet,
  build, module verification, JavaScript checks/tests, and `git diff --check`
  pass. No commit was created.

## 2026-08-29

### Project analyzer assignments implementation

* **Implementation:** Archived issues 044–046 after delivering strict v1/v2
  configuration loading, deterministic assignment/source-scope planning, and
  immutable session caching with selective invalidation. Issue 047's
  configured mixed-language viewer integration is automated-test complete and
  remains `awaiting-human-review` for its declared visual gate.
* **Artifact sync:** Updated the owning capability, plugin-runtime roll-ups,
  implementation slice, issue registry, issue records, and delivery status.
  The child capability remains `specified` until issue 047 receives visual
  approval; no product or architecture boundary changed.
* **Verification:** Full Go tests, race tests, vet, build, JavaScript syntax
  checks, available viewer-module tests, and `git diff --check` pass. A
  reproducible mixed-language fixture is checked in under the owning artifact
  root. No commit was created.

## 2026-08-29

### Project analyzer assignments delivery slicing

* **Approval:** The user approved the four dependency-ordered implementation
  slices for Project analyzer assignments and view selection.
* **Delivery:** Created ready-for-agent issues 044–047 for versioned analysis
  configuration loading/validation, configured assignment and source-scope
  planning, session cache reuse with selective invalidation, and the configured
  mixed-repository viewer journey. Issue 047 carries the required
  `visual-review` gate; issue 044 is the unblocked starting point.
* **Graph/frontier:** Added all four pending issue references to the owning
  project-assignment capability. The node remains `specified`; no topology or
  state transition occurred. Current totals remain 0 `foggy`, 0 `bounded`, 4
  `specified`, and 10 `implemented`.
* **Artifact impact:** Delivery truth changed. The exact capability
  specification, product scope, architecture boundaries, and existing
  multi-analyzer ownership remain unchanged; their orchestration records now
  point to the approved delivery batch. Advanced ELK remains a separate
  specified frontier.
* **Verification:** Issue references, registry numbering, the application
  synthesis gate, and strict OKF validation were synchronized and pass. No
  commit was created.

### Multi-analyzer delivery verification and visual-review handoff

* **Implementation:** Verified and archived issues 039–042. The repository now
  plans deterministic nested roots and source scopes, executes bounded
  analyzer jobs with lifecycle/cancellation control, aggregates collision-safe
  namespaced results, and exposes the combined CLI/HTTP contracts.
* **Viewer handoff:** Issue 043 is implemented and remains active as
  `awaiting-human-review` for the required mixed-language visual inspection of
  `All`, individual scopes, failed-scope diagnostics, accessibility, and
  existing navigation/details behavior. The multi-analyzer capability remains
  `specified` until that gate is approved.
* **Graph/frontier:** Updated the owning capability, Analyze source and
  plugin-runtime roll-ups, application PRD, application architecture summary,
  implementation slice, orchestration records, registry, and issue references.
  The assignment/configuration capability remains a separate specified future
  frontier. Totals remain 0 `foggy`, 0 `bounded`, 5 `specified`, and 9
  `implemented`.
* **Verification:** `go test ./... -count=1`, targeted race tests, `go vet ./...`,
  `go build ./...`, JavaScript syntax checks, available Node viewer-module
  tests, strict OKF validation, and `git diff --check` pass on Windows amd64.
  No commit was created.

### Multi-analyzer viewer visual review and closeout

* **Human review:** The user approved issue 043's mixed-language viewer review,
  including `All` and individual scopes, failed-scope diagnostics, status and
  language presentation, keyboard accessibility, existing navigation/details
  behavior, and the collapsed scope-picker interaction.
* **Closeout:** Archived issue 043 at
  `docs/agents/issues/done/20260828-043-cached-scope-projections-and-viewer-selection.md`
  and removed its active registry row. The multi-analyzer capability is now
  `implemented`; its 039–043 delivery references and parent roll-ups point to
  completed records. The separate project-assignment/view capability remains
  `specified`.
* **Planning:** State totals are now 0 `foggy`, 0 `bounded`, 4 `specified`,
  and 10 `implemented`.
* **Verification:** Focused viewer tests, JavaScript syntax checks, `go test
  ./... -count=1`, strict OKF validation, and `git diff --check` pass. No
  commit was created.

### Planning map reconciliation after multi-analyzer closeout

* **State audit:** Reconciled the Analyze source, Analyzer plugin runtime,
  Explore, project-assignment, and multi-analyzer planning records with the
  completed 039–043 delivery and the current viewer behavior. The cached
  combined/individual scope viewer is implemented; persisted assignment and
  source-scope configuration remains specified.
* **Frontier:** The only non-implemented capability nodes are the two specified
  leaves—Project analyzer assignments and view selection, and Advanced ELK
  renderer support—plus their specified roll-up parents. No new topology or
  delivery issue was created.
* **Artifact impact:** Updated current node notes, linked orchestration records,
  and application synthesis wording. No product boundary or ownership change
  was introduced.

## 2026-08-28

### Analysis source-scope specification refresh

* **Ownership and frontier:** Extended the existing specified project-analyzer
  assignments/view-selection child in place; no new OKF capability node was
  created. Multi-analyzer orchestration consumes the resolved policy, while
  project configuration owns its persistence and validation.
* **Canonical contract:** `arch-view.config/v2` now defines optional global
  `analysis.exclude` globs and analyzer-ID keyed `analysis.include` rules with
  `globs` arrays. Patterns are normalized relative to the invocation root,
  use an explicit deterministic glob subset, apply after root discovery, and
  let exclusions win. Effective source-set identity participates in planning
  and cache identity.
* **Artifact impact:** Capability, exact-spec, application PRD, application
  architecture, and pending delivery references were synchronized. Existing
  issues 039–043 consume the policy; no new assignment issue was created.
* **State:** No node state transition occurred. Totals remain 0 `foggy`, 0
  `bounded`, 5 `specified`, and 9 `implemented`.

### Multi-analyzer orchestration issue slicing

* **Application synthesis gate:** Revalidated `docs/prd.md` and
  `docs/architecture/application-architecture-summary.md` before issue
  creation. Both reflect the implemented compiled distribution and the
  readiness-reviewed multi-analyzer boundary; no High or Medium readiness
  blockers were found.
* **Approved delivery batch:** Created issues 039–043 for deterministic root
  discovery/job planning, bounded execution/lifecycle, namespaced aggregation,
  CLI/HTTP exposure, and cached viewer scope selection. Issue 039 is unblocked;
  issues 040–043 are dependency-ordered, and issue 043 carries `visual-review`.
* **Graph/frontier:** Linked pending issue references from the multi-analyzer
  child and the plugin-runtime/analyze-source roll-ups. The multi-analyzer node
  remains `specified`; project assignment/view selection and advanced ELK remain
  specified future frontiers. Current totals are 0 `foggy`, 0 `bounded`, 5
  `specified`, and 9 `implemented`.
* **Artifact impact:** Delivery truth changed. Product and architecture
  documents were refreshed for compiled-distribution status before slicing, but
  semantic boundaries are unchanged; persisted analyzer assignments remain
  owned by the separate assignment capability.

### Planning-map reconciliation

* **Graph state:** Reconciled the completed [compiled external analyzer distribution](capabilities/analyze-source/plugin-runtime/compiled-external-analyzer-distribution.md) node as `implemented`; its 036–038 delivery references now roll up through the `plugin-runtime` and `analyze-source` ancestor concepts. Current totals remain 0 `foggy`, 0 `bounded`, 5 `specified`, and 9 `implemented`.
* **Frontier:** Preserved the three remaining specified leaf frontiers—advanced ELK renderer support, multi-analyzer project orchestration, and project analyzer assignments/view selection. No new topology or next-node selection was made.
* **Artifact impact:** Capability and topology references were synchronized. Product and application-architecture artifacts have no impact because the completed node’s product boundary and cross-capability contracts are unchanged. The active issue registry and max issue ID remain correct at 038.

### Compiled distribution closeout

* **Implementation**: Completed issues 036–038. Added trusted application-managed analyzer catalog verification with strict metadata, exact-platform selection, safe paths, symlink rejection, descriptor/executable SHA-256 checks, manifest/API agreement, stable package errors, and process-free listing.
* **Runtime**: Made packaged analyzers the normal release path for `analyzers`, `analyze`, and project-backed `open`; retained explicit in-process and opt-in local-descriptor modes, runtime provenance, no implicit fallback, and reanalysis runtime retention.
* **Parity/release**: Verified all five compiled analyzers against their in-process implementations, deterministic matrix assembly, stable tamper/missing/API/launch outcomes, target-repository discovery isolation, and a real Windows amd64 `make release` plus packaged Go analysis. Fixed the Windows `StdoutPipe`/`Cmd.Wait` lifecycle race so large terminal results drain before reaping the child.
* **Deferral**: Linux amd64 and Darwin arm64 execution remain deferred under the root `when-supported` policy because this host has no matching target runner/toolchain; re-entry requires the corresponding runner/toolchain and the same matrix, package-smoke, and parity commands.
* **Verification**: `go test ./... -count=1`, targeted race tests, `go vet ./...`, `go build ./...`, `git diff --check`, real `make analyzers`/`make release`, packaged listing, and packaged CLI analysis passed. The compiled-distribution capability moved to `implemented`; the parent plugin-runtime and analyze-source capabilities remain specified for later multi-analyzer and assignment/view work. Product and application-architecture artifacts were unchanged; no commit was created.

### Brownfield implementation verification

* **Observed in code:** The Go analyzer covers module selection, eligible
  package discovery, local import resolution, exclusions, evidence, and
  non-local diagnostics. Canonical model generation covers normalization,
  hierarchy, aggregation, cycles, layers, validation, partial results, and
  deterministic output. Export covers versioned JSON, self-contained HTML,
  static and browser SVG, reference visibility, overwrite safety, and CLI
  status behavior.
* **Delivery evidence:** Issues 001, 002, 005, 011, 013, 015, and 016 are done.
  Their traceability records cover every acceptance scenario owned by Go
  analysis, Generate architecture models, and Export and automate.
* **Verification:** `go test ./...` and
  `node --test internal/viewer/web/*_test.js` pass on 2026-08-28. Existing
  visual approvals cover the export behavior that requires human review.
* **State transitions:** Advanced `Go analysis`, `Generate architecture
  models`, and `Export and automate` from `specified` to `implemented`.
  Current totals are 0 `foggy`, 0 `bounded`, 6 `specified`, and 8
  `implemented`.
* **Artifact sync:** Updated the three capability nodes and orchestration
  records, the graph index, application PRD, and application architecture
  summary. Product behavior, architecture boundaries, topology, and delivery
  issues have no impact because this audit corrects planning status to match
  already completed work.

* **Fog clearing:** Resolved the remaining product-boundary decisions for all
  four foggy frontiers using the user-authorized recommended answers. Compiled
  analyzers are the production implementation for stable logical IDs, built by
  a reproducible make/build target into an application-managed analyzer tree;
  in-process adapters remain only for development, tests, or explicit fallback.
* **Bounded orchestration:** Confirmed marker-driven nested project discovery,
  bounded concurrent jobs, namespaced merge identity, no inferred
  cross-language/cross-root relationships, partial-success behavior, and one
  combined model backed by per-job results.
* **Bounded assignments:** Confirmed a separate `analysis` section in
  `.archview.json`, deterministic nested assignment/CLI precedence, stable
  analyzer IDs, cached `All` and per-scope views, and affected-scope
  reanalysis behavior.
* **Bounded renderer frontier:** Confirmed staged ELK extension priority:
  route/output features first, structural scene features second, and broader
  target-specific options only with concrete renderer support. Browser,
  embedded HTML, and browser SVG share the advanced geometry while Go static
  SVG retains its deterministic orthogonal contract.
* **State transitions:** Advanced `Advanced ELK renderer support`, `Compiled
  external analyzer distribution`, `Multi-analyzer project orchestration`,
  and `Project analyzer assignments and view selection` from `foggy` to
  `bounded`. Current totals are 0 `foggy`, 4 `bounded`, 5 `specified`, and 5
  `implemented`.
* **Artifact sync:** Updated all four capability concepts, their discovery and
  orchestration records, parent/source-analysis records, project/index
  frontiers, application PRD, and application architecture summary. No exact
  specification set, delivery issue, or ADR was created; those remain
  downstream of bounded-node specification.

### Exact specification closure

* **Specification pass:** Ran the exact-spec pipeline for all four bounded
  frontiers: compiled external analyzer distribution, multi-analyzer project
  orchestration, project analyzer assignments/view selection, and advanced ELK
  renderer support. Each now has a requirements gap analysis, glossary, PRD,
  canonical domain model, use-case set, API/CLI contract, acceptance scenarios,
  and readiness review with no High or Medium findings.
* **Trust decision:** Recorded the user's confirmation that production may
  execute only bundled, checksum-verified analyzer executables from the
  application-managed `analyzers/` tree. User descriptors and in-process
  adapters remain explicit developer/test or migration overrides.
* **State transitions:** Advanced the four child nodes from `bounded` to
  `specified`. Current totals are 0 `foggy`, 0 `bounded`, 9 `specified`, and 5
  `implemented`.
* **Artifact sync:** Synchronized the OKF map, parent/source-analysis records,
  application PRD, application architecture summary, and delivery/orchestration
  records. No implementation issues or ADRs were created; the specified
  frontiers are ready for issue slicing after application synthesis.

### Compiled distribution issue slicing

* **Application synthesis gate:** The root PRD and application architecture
  summary remain current, with no unresolved High or Medium findings for this
  child. Product and architecture semantics therefore have no impact from this
  delivery-only update.
* **Approved migration sequence:** Issue 034 explicitly ports and reuses the
  existing Go, Python, TypeScript, Rust, and Clojure implementations behind
  compiled plugin entrypoints. Issues 035–038 then assemble the distribution,
  verify package trust, make packaged execution the default, and close release
  and parity verification.
* **Artifact sync:** Updated the compiled-distribution node, its orchestration
  status, the plugin-runtime and Analyze source delivery records, the project
  and graph frontiers, the application PRD and architecture summary, the
  implementation slice, and the issue registry.
* **Planning state:** The compiled-distribution node remains `specified`; issue
  slicing changes delivery truth only. Current totals remain 0 `foggy`, 0
  `bounded`, 6 `specified`, and 8 `implemented`.

### Issue 034 completion

* **Implementation:** Added one shared child-side NDJSON process runner and
  compiled entrypoints for the existing Go, Python, TypeScript, Rust, and
  Clojure analyzers. Each command calls its existing `New()` constructor; no
  language semantics or registry behavior were duplicated or replaced.
* **Verification:** Runner protocol tests and the five-language compiled
  process-adapter parity test pass. `go test ./... -count=1`, `go vet ./...`,
  `git diff --check`, and strict OKF validation pass with zero errors and zero
  warnings.
* **Artifact sync:** Archived issue 034, unblocked issue 035, and updated the
  compiled-distribution, plugin-runtime, Analyze source, and five supporting
  analyzer delivery records. Product and architecture semantics remain
  unchanged; package assembly, trust, runtime selection, and release closure
  remain in issues 035–038.

### Issue 035 completion

* **Implementation:** Added the reusable analyzer distribution and release
  assembler, exact `windows-amd64`/`linux-amd64`/`darwin-arm64` target matrix,
  explicit Go/C/C++ build inputs, descriptor and SHA-256 generation, strict
  canonical index validation, safe replacement checks, atomic publication,
  and root `make analyzers`/`make release` targets.
* **Verification:** The focused distribution tests, full `go test ./...` and
  `go test -race ./...`, `go vet ./...`, `go build ./...`, `staticcheck ./...`,
  `golangci-lint run`, `go mod verify`, and `git diff --check` pass. Real
  Windows-target `make analyzers` and `make release` smoke tests
  pass with all five analyzers; repeated analyzer assembly produces a stable
  index for unchanged inputs.
* **Artifact sync:** Archived issue 035, unblocked issue 036, and updated the
  compiled-distribution node, orchestration status, plugin-runtime
  implementation slice, project/graph frontiers, issue registry, and this
  OKF log. Product and application architecture semantics remain unchanged;
  trust verification, packaged runtime selection, and final release/parity
  closure remain in issues 036–038.

## 2026-08-28

* **Topology:** Added three foggy future children under [Analyzer plugin
  runtime](capabilities/analyze-source/plugin-runtime.md): compiled external
  analyzer distribution, multi-analyzer project orchestration, and project
  analyzer assignments/view selection.
* **Target clarification:** Recorded the user-confirmed direction to replace
  the script-based external pilot with compiled analyzer executables that reuse
  the existing analyzer implementations, run multiple applicable analyzers
  concurrently, and persist folder-to-analyzer assignments with an application
  scope selector. Current implementation remains separate from this target.
* **Artifact sync:** Updated the parent/child OKF concepts, project/index
  frontiers, plugin-runtime discovery and orchestration records, application
  PRD, and application architecture summary. No delivery issues or ADRs were
  created; no planning state advanced or regressed.

## 2026-08-27

* **Issues 031–033 completed:** Archived the process-backed analyzer host,
  external Python parity deployment, and explicit plugin CLI/shared journey at
  the dated records in `docs/agents/issues/done/`. The external Python pilot
  is standard-library-only, opt-in through a local descriptor, and remains
  separate from the built-in Python analyzer.
* **Artifact sync:** Updated the plugin-runtime contract, readiness review,
  implementation slice, orchestration status, Python and Analyze source
  capability records, parent/project planning frontier, application PRD,
  application architecture summary, and issue registry. The plugin-runtime
  capability remains `specified` because no next external frontier was chosen.
* **Verification:** External/in-process parity, repeat byte stability,
  no-target-execution, canonical model/viewer/source/export integration,
  `go test ./... -count=1`, `go vet ./...`, `git diff --check`, and strict OKF
  validation pass with zero errors and zero warnings.

## 2026-08-27

* **Issue 030 completed:** Archived [external protocol schema and conformance
  fixture](../docs/agents/issues/done/20260827-030-external-protocol-schema-and-conformance-fixture.md)
  after implementing the bounded typed NDJSON codec, strict descriptor
  validation, stateful one-request protocol checks, streamed-diagnostic merge,
  and test-only subprocess fixture.
* **Artifact sync:** Refreshed the protocol contract, external implementation
  slice, plugin-runtime and analyze-source delivery records, issue registry,
  and capability references. Product/model/viewer/export architecture is
  unchanged; issue 031 is now unblocked.
* **Verification:** Focused protocol tests, `go test ./... -count=1`,
  `go vet ./...`, schema parsing, and OKF validation pass. The external
  fixture remains test-only and is not publicly registered.

## 2026-08-27

* **External plugin slice approved:** Following the confirmed direction to
  port an existing language externally, selected the specified Analyzer plugin
  runtime as the next frontier and chose an opt-in external Python parity
  pilot rather than a new language.
* **Spec slicing:** Added the external analyzer implementation slice and
  tightened the descriptor, manifest handshake, detect/analyze NDJSON
  lifecycle, stdout/stderr policy, cancellation, timeout/size limits, and
  published protocol/descriptor schemas.
* **Delivery slicing:** Created ready-for-agent issues 030–033 in dependency
  order for the protocol schema/conformance fixture, Go process-backed host,
  external Python parity analyzer, and explicit CLI/shared model-viewer-export
  integration.
* **Artifact sync:** Updated the plugin-runtime and Python capability
  references, parent/project frontiers, analyzer orchestration records,
  application PRD, application architecture summary, and issue registry.
  Topology and capability states are unchanged; the active registry now ends
  at issue 033.

## 2026-08-27

* **Rust implementation complete**: Accepted and archived issues 023–025 after
  implementing the data-only Cargo boundary, safe crate/workspace selection,
  reachable module/evidence discovery, static relationships, uncertainty, and
  the shared canonical/public output path. The Rust analysis capability moved
  from `specified` to `implemented`; the parent Analyze source code capability
  remains `specified` for the later TypeScript, Clojure, and plugin-runtime
  territories.
* **Artifact sync**: Updated the Rust and parent OKF nodes, orchestration
  records, implementation slice, application PRD, application architecture
  summary, and delivery registry. The active registry is empty and current max
  issue ID remains 029. State totals are now 1 `foggy`, 0 `bounded`, 7
  `specified`, and 3 `implemented`.
* **Verification**: Strict OKF v0.1 validation, focused and full Go tests,
  race tests, vet, build, staticcheck, golangci-lint, Node checks, and diff
  checks passed. No visual review gate applied to the backend/CLI Rust adapter.

## 2026-08-27

* **Rust delivery slicing approved**: After the application synthesis gate and
  explicit user approval, created ordered AFK issues 023–025 for Cargo
  boundary/registration, module discovery/evidence, and relationships/uncertainty
  plus the end-to-end output path.
* **Artifact sync**: Linked the Rust implementation slice and issue paths from
  the Rust capability, updated the Rust and parent orchestration records, and
  recorded that issue 025 owns the final application PRD and architecture-summary
  refresh. No planning node state changed during slicing.

* **Issue 029 and Clojure capability complete:** Verified the public
  `--platform` integration for analyze/open option collection and the shared
  Clojure path from analysis JSON through canonical normalization, validation,
  hierarchy projection, local viewer/source inspection, and deterministic
  JSON/HTML/SVG export. Archived the issue at
  `docs/agents/issues/done/20260827-029-clojure-public-integration-and-deterministic-exports.md`.
* **State transition:** Advanced `Clojure compatibility` from `specified` to
  `implemented` after issues 026–029 were exhausted and verified. Updated the
  application PRD, application architecture summary, owning/parent
  orchestration records, implementation slice, issue registry, and OKF index;
  state totals are now 1 foggy, 0 bounded, 7 specified, and 3 implemented.
* **Verification:** `go test ./... -count=1`, focused Clojure visible-journey
  tests, canonical validation, deterministic repeated exports, and no-
  evaluation safety checks passed.

## 2026-08-27

* **Issue 028 complete:** Added platform-aware `.cljc` reader-conditional
  selection, spliced/default branch metadata, polymorphic defprotocol/defmulti
  evidence, dynamic loading references/diagnostics, malformed-reader recovery,
  and no-evaluation tests. Archived the issue at
  `docs/agents/issues/done/20260827-028-clojure-platform-polymorphism-and-safety.md`;
  issue 029 is now the next unblocked Clojure frontier.
* **Verification:** Focused Clojure safety tests and common result-path tests
  passed; no application PRD or architecture-summary update was required by
  issue 028 because the behavior remains inside the confirmed adapter and
  language-neutral contract.

## 2026-08-27

* **Issue 027 complete:** Added conservative static Clojure namespace
  dependency extraction for require/use/macro clauses, local module
  resolution, standard/external/unresolved references, merged aliases and
  referred-symbol evidence, confidence, source locations, partial diagnostics,
  deterministic ordering, and cancellation-safe execution. Archived the issue
  at `docs/agents/issues/done/20260827-027-clojure-static-namespace-dependencies.md`;
  issue 028 is now the next unblocked Clojure frontier.
* **Verification:** Focused dependency tests and common result validation
  passed; no application PRD or architecture-summary update was required by
  issue 027 because the existing static workflow and language-neutral boundary
  remained unchanged.

## 2026-08-27

* **Issue 026 complete:** Implemented and verified the registered Clojure
  analyzer foundation: marker/configuration precedence, safe source roots,
  flavor and test filtering, namespace modules, deterministic source evidence,
  recoverable malformed/missing-namespace diagnostics, cancellation, and the
  no-evaluation boundary. Archived the issue at
  `docs/agents/issues/done/20260827-026-clojure-project-and-namespace-discovery.md`;
  issue 027 is now the next unblocked Clojure frontier.
* **Verification:** Focused Clojure and CLI tests passed; no application PRD
  or architecture-summary update was required by issue 026 because the common
  analyzer boundary and confirmed product scope were unchanged.

## 2026-08-27

* **Planning:** Following the specified-node routing in the fog-of-war
  workflow, created the approved Clojure delivery slice under the existing
  `Clojure compatibility` capability. Issues 026–029 cover project and
  namespace discovery, static dependency extraction, platform/polymorphic and
  safety metadata, and public shared-path integration in dependency order.
* **Artifact sync:** Linked the four pending issues from the Clojure child and
  parent analyzer concepts, added the Clojure implementation slice, refreshed
  analyzer orchestration status, and updated the active issue registry through
  max ID 029. The application-synthesis gate is current; product scope,
  canonical model semantics, and cross-capability architecture remain
  unchanged until implementation closeout.
* **Issue 022 complete**: The user explicitly approved the TypeScript result
  after reviewing the running local viewer in windowed and full-canvas modes,
  including graph readability, hierarchy, labels, directed arrows,
  references/diagnostics, evidence/source locations, navigation, Fit/reset,
  and export controls. The issue is archived and the TypeScript capability
  advances from `specified` to `implemented`.
* **Artifact sync**: Updated the README with TypeScript/JavaScript project
  prerequisites, moved issue 022 to `done/`, removed its active registry row,
  updated capability/project/index state and implementation-slice references,
  and synchronized the application PRD, architecture summary, and analyzer
  orchestration statuses. The parent Analyze source capability remains
  `specified` for its later Rust, Clojure, and plugin-runtime territories.
* **Strict review hardening**: Completed the code-review loop over the full
  uncommitted TypeScript slice. The final pass covers safe package/config
  boundaries, runtime/package-export provenance, Windows path semantics,
  lexical false-positive guards, deterministic output, and the shared CLI/
  viewer boundary; no actionable P0–P2 findings remain.
* **Final verification and closeout**: Full tests, race, vet, build,
  Staticcheck, golangci-lint, viewer JavaScript syntax validation, strict OKF
  validation, and `git diff --check` pass. Issues 020–022 remain archived,
  the active registry is empty, the maximum issue ID remains 022, and no
  commit was created.

## 2026-08-27

* **Issue 021 complete**: Extended the registered TypeScript analyzer with
  conservative lexical import/export/require/dynamic discovery, relative,
  alias, root-directory, self-package, and package-export resolution, typed
  dependency metadata, source evidence, uncertainty references, and
  recoverable diagnostics/partial results. The issue is archived and issue
  022 is unblocked for the public visible-journey slice; the TypeScript node
  remains `specified` because the final visual-review gate is still open.
* **Verification**: Focused and full Go tests, race, vet, build,
  staticcheck, golangci-lint, strict OKF validation, and `git diff --check`
  passed for the completed static-dependency slice.
* **Artifact sync**: Updated the issue registry, dated capability issue
  references, implementation slice, project/index frontier, application PRD,
  application architecture summary, and analyzer orchestration records. No
  topology, canonical schema, viewer, exporter, or upstream-reference change
  was introduced.

* **Issue 022 automated implementation complete**: Connected the TypeScript
  analyzer through the public CLI and shared analysis → canonical model →
  projection/viewer/export path. Added regression coverage for explicit and
  automatic selection, analyzer options, source evidence, diagnostics,
  directed relationships, deterministic analysis, and JSON/HTML/SVG output.
* **Review gate**: Full repository and strict OKF verification passed. Issue
  022 remains in `pending/` with registry state `awaiting-human-review` for
  the required windowed/full-canvas visual review; no TypeScript capability
  state transition or issue archival has been recorded.

## 2026-08-27

* **Issue 020 complete**: Implemented and registered the TypeScript analyzer's
  safe project/config boundary, JSONC `extends` resolution, effective source
  scope, package/runtime context, deterministic module discovery, source
  evidence, and recoverable configuration/scope diagnostics.
* **Verification**: Focused and full Go tests, race, vet, build,
  staticcheck, golangci-lint, strict OKF validation, and `git diff --check`
  passed. Issue 020 is archived; issue 021 is unblocked and issue 022 remains
  dependent on it. The TypeScript node remains `specified` because its static
  dependency, visible-journey, and final visual-review work is not exhausted.
* **Artifact sync**: Updated the active registry, dated capability issue
  references, implementation slice, project/index frontier, application PRD,
  application architecture summary, and analyzer orchestration records. No
  topology, canonical schema, viewer, exporter, or upstream-reference change
  was introduced.

## 2026-08-27

* **TypeScript delivery slice approved**: Created issues 020–022 for the
  TypeScript project boundary/module discovery, static dependencies and
  uncertainty, and public/visible analysis path in dependency order, with
  acceptance criteria, verification obligations, and artifact-sync rules.
* **Artifact sync**: Linked the implementation slice and active issue paths
  from the TypeScript and parent Analyze source capability nodes and updated
  the project index, application PRD, application architecture summary, and
  orchestration statuses. The TypeScript node remains `specified` until its
  scoped implementation and final review gates complete.

## 2026-08-27

* **Issue 019 complete**: The user explicitly approved the corrected Python
  top-level and drilled hierarchy in windowed and full-canvas modes, including
  directed edges, labels, references, evidence, diagnostics, viewport controls,
  and the existing export surface. The issue is archived, its active registry
  row is removed, and the Python analysis capability advances from `specified`
  to `implemented`; the parent Analyze source code capability remains
  `specified` while later language and plugin territories remain.
* **Issue 019 automated implementation complete**: Connected the public Python
  CLI options and deterministic analyzer selection to the existing
  analysis → canonical model → local viewer/export journey. The representative
  integration fixture verifies package/module hierarchy, directed imports,
  source evidence, references, confidence, dynamic/unresolved diagnostics,
  model normalization/validation/projection, self-contained HTML, static SVG,
  current-canvas download markers, and repeated byte-stable output without
  Python-specific branches in the shared model, scene, layout, or browser
  paths.
* **Verification**: `go test ./... -count=1`, `go test -race ./...`, `go vet
  ./...`, `go build ./...`, `staticcheck ./...`, `golangci-lint run`, the
  viewer JavaScript syntax/pure-module tests, and the issue-focused Python
  journey tests pass. At this automated checkpoint, issue 019 remained active
  until the declared visual-review gate could be completed.
* **Visual-review correction**: The live Python journey exposed windowed
  **Fit** measuring an aspect-ratio-expanded SVG instead of its clipped graph
  viewport, which could leave right-side modules outside the visible canvas.
  Fit now derives its available area from the visible graph container, with a
  pure browser-module regression test covering the mismatch.

* **Issue 017 complete**: Added the registered in-process Python analyzer
  foundation. Marker precedence is `pyproject.toml`, `setup.cfg`, then
  `setup.py`; configuration and roots are read safely as data, with `src/`
  and repository-root fallback rules. Regular and namespace packages, modules,
  stubs, tests, exclusions, source evidence, stable IDs, and recoverable
  diagnostics are emitted through the common analyzer contract. The CLI now
  supports Python selection/options without adding language-specific host
  orchestration. Python import relationships remain assigned to issue 018 and
  the visible journey to issue 019.
* **Verification**: Focused Python fixtures and CLI/model normalization pass;
  `go test ./... -count=1`, race tests, vet, build, Staticcheck,
  golangci-lint, and `git diff --check` pass. No canonical schema, topology,
  renderer, or upstream reference repository changed.
* **Issue 018 complete**: Extended the registered Python analyzer with a
  conservative static import pass for absolute, relative, package-init
  re-export, standard-library, external, unresolved, conditional, and dynamic
  behavior. Proven local targets become deterministic `depends_on`
  observations with merged source evidence; uncertainty remains references,
  confidence, and recoverable diagnostics. The analyzer never imports or
  executes target code, and issue 019 is now the next unblocked Python frontier.
* **Verification**: Focused Python import/resolution fixtures, common result
  validation, byte-stable repeated output, `go test ./... -count=1`,
  `go test -race ./...`, `go vet ./...`, `go build ./...`, `staticcheck ./...`,
  `golangci-lint run`, JavaScript syntax/pure-module tests, `git diff --check`,
  and strict OKF validation pass. No canonical schema, topology, renderer, or
  upstream reference repository changed.

## 2026-08-27

* **Implementation verification**: Audited the Explore and inspect architecture
  capability against its 17 acceptance scenarios, issues 003–016, the exact
  PRD, and the repository verification gates. The current specified scope is
  complete, so the parent node moves from `specified` to `implemented`.
* **Topology**: Added the foggy child capability [Advanced ELK renderer
  support](capabilities/explore-architecture/advanced-elk-renderer-support.md)
  so ports, labels, junctions, compound geometry, broader target-specific
  options, and spline-specific refinements are visible future work rather than
  residual notes hidden under implementation.
* **Planning**: Recorded the candidate renderer workstreams and entry criteria
  in the [future-work register](../docs/architecture/explore-architecture/advanced-elk-renderer-support/future-work.md).
  No delivery issues were created for the child because its supported subset,
  contract impact, and acceptance behavior still need fog clearing.
* **Verification**: `go test ./... -count=1`, `go test -race ./...`, `go vet
  ./...`, `go build ./...`, `staticcheck ./...`, `golangci-lint run`, the
  JavaScript syntax/pure-module tests, strict OKF validation, and `git diff
  --check` pass. State totals are now 1 `foggy`, 0 `bounded`, 9 `specified`,
  and 1 `implemented`.

## 2026-08-26

* **Architecture refactor**: Implemented issues 010–015 while preserving the
  canonical model, scene, HTTP, CLI, configuration, and export contracts.
  Routing is now renderer-neutral with deterministic orthogonal/manual and
  existing self-loop behavior; spline segments are reserved but not rendered.
* **Composition**: Split the live viewer into native ES modules, kept
  `app.js` as a small entrypoint, added embedded esbuild bundling with external
  import rejection for self-contained HTML, and decomposed the Go viewer host
  and CLI into focused files.
* **Capabilities**: Isolated scene projection, the Go scanner/import/
  observation pipeline, the ELK option-handler registry, and canonical model
  normalization/validation into explicit packages. Issue 009's node/edge ELK
  target mapping is explicitly deferred by the user.
* **Verification**: `go test ./... -count=1`, race tests, vet, build,
  staticcheck, golangci-lint, JavaScript syntax/pure-module tests, export
  self-containment/determinism tests, and `git diff --check` pass. Issues
  010–015 are recorded as awaiting human/repository review handoffs;
  the [upstream reference repository](https://github.com/unclebob/arch-view)
  remains outside the product and untouched.

## 2026-08-27

* **Implementation**: Resumed issue 009 after the user's explicit direction. The
  layout registry now supports `org.eclipse.elk.priority` on visible nodes and
  edges plus the layered direction, shortness, and straightness edge-priority
  options. Catalog targets control whether values are emitted on the root,
  node, or edge ELK element; unsupported target-specific options remain
  catalog-only. Automated verification passes and visual review is pending.

* **Closeout**: The user approved issue 009's target-aware settings behavior in
  normal and full-canvas views. The issue was archived, removed from the active
  registry, and synchronized across the owning capability and implementation
  slice; spline rendering remains a separate future issue.

* **Planning**: Added issue 016 under the existing Explore and inspect
  architecture capability for bounded ELK spline-route activation. It will map
  valid ELK spline control data to the reserved cubic route representation,
  preserve orthogonal fallback/manual routing, and verify browser/SVG/HTML
  parity. No new OKF capability node was created and implementation has not
  started.

* **Implementation**: Issue 016 now accepts layered `SPLINES` profiles,
  normalizes ELK's `3n−1` control-point streams into finite cubic routes, and
  renders them through the shared browser/SVG route serializers. Malformed
  spline data falls back to deterministic orthogonal geometry; manual
  movement remains orthogonal and no Libavoid routing is used. Automated
  verification passes and visual review is pending.

* **Review and closeout**: The user explicitly approved the integrated visual
  review for issues 010, 011, 012, and 014 and the repository review for
  issues 013 and 015. Issues 010–015 were moved to the dated delivery archive,
  removed from the active registry, and synchronized across the owning OKF
  capabilities and implementation slice. At that point issue 009 remained
  deferred; it was subsequently resumed and closed after visual approval.

* **Export extension**: Issue 016 now embeds the effective layout profile and
  catalog plus the pinned ELK runtime in Go-generated HTML, allowing the file
  to recalculate its scene without a server. The browser adds Download SVG for
  the current canvas; Go static SVG remains the deterministic orthogonal
  headless artifact. Automated checks pass and visual review remains pending.

* **Closeout**: The user approved issue 016's visual review of windowed and
  full-canvas spline routes, self-contained HTML, browser Download SVG,
  labels, arrowheads, navigation, self-loops, and deterministic fallback.
  The issue was archived at the dated delivery path, removed from the active
  registry, and synchronized across the Explore capability, implementation
  slice, application records, and OKF references. The upstream reference
  repository remains outside the product and untouched.

* **Frontier selection**: After issue 016 closeout, selected the specified
  Python analysis child as the next implementation frontier. The map has no
  foggy or bounded nodes; Python is the first non-Go language in the agreed
  sequence and the common analyzer, model, viewer, and export boundaries are
  ready for a visible adapter result.

* **Slice preparation**: Created the Python repository-to-visible-architecture
  implementation slice and ready-for-agent issues 017–019 in dependency order:
  project/module discovery, static import resolution and uncertainty, and
  public CLI/model/viewer/export integration with the final visual review.
  No new capability node or shared concern was needed.

* **Artifact sync**: Linked the Python slice and issues from the parent and
  child analysis concepts, refreshed the analysis orchestration status,
  application PRD, and application architecture summary with the next
  implementation order, and updated the issue registry through max ID 019.
  State totals remain 0 `foggy`, 0 `bounded`, 10 `specified`, and 0
  `implemented`; the Python node will advance only after its complete slice is
  implemented and reviewed. The upstream [reference repository](https://github.com/unclebob/arch-view) remains read-only and untouched.

## 2026-08-26

* **Closeout**: Archived issue 008 after the user's explicit approval of the parent-level ELK option tranche in windowed and full-canvas views, including representative settings, diagnostics, fallbacks, and resulting layouts. Removed its active registry row, updated the owning capability and implementation-slice references, and unblocked issue 009 without processing it; the [upstream reference repository](https://github.com/unclebob/arch-view) remains untouched.

* **Implementation**: Issue 008 expanded the editable ELK parent-level tranche with typed metadata and validation for aspect ratio, layered spacing, layering, cycle breaking, crossing minimization, node placement, and connected-component compaction. The browser request builder now forwards only catalogued editable `PARENTS` options to the root graph; node/edge-targeted options remain reserved for issue 009.
* **Correction**: Pinned catalog inspection confirmed `org.eclipse.elk.alignment` is node-targeted and that the canonical parent spacing key is `org.eclipse.elk.layered.spacing.baseValue`; neither is silently treated as a root option.
* **Verification**: Focused Go/JavaScript tests, syntax checks, repository gates, and OKF validation pass. Issue 008 is awaiting visual review of representative settings in windowed and full-canvas views; the [upstream reference repository](https://github.com/unclebob/arch-view) remains untouched.

* **Closeout**: Archived issue 005 at [the dated delivery record](../docs/agents/issues/done/20260826-005-deterministic-json-html-svg-export.md) after the user's explicit approval of HTML/SVG parity with the local viewer; required capability, orchestration, and implementation-slice references now point to the completed issue, and issue 007 remains the next ready-for-agent frontier.

* **Planning**: Added the user-confirmed ELK layout-settings and project-configuration extension as issue 007 under the existing Explore and inspect architecture capability; no new capability or shared-concern node was needed because the behavior remains viewer-owned presentation policy.
* **Design**: Chose versioned `.archview.json` as the v1 project configuration file. Discovery walks from the selected target directory through its ancestors toward the filesystem root, the nearest file wins without merging, and an invalid nearest file is surfaced instead of bypassed; the configuration changes layout presentation only.
* **Artifact sync**: Refreshed the Explore exact-spec set, application PRD, application architecture summary, capability issue references, registry, and orchestration records. Analyzer options, canonical model semantics, and export behavior are explicitly unchanged by issue 007.
* **Delivery**: Created issue 007 as `ready-for-agent`, with settings-catalog, validation, apply/reset, active-file `Save`, explicit custom-folder `Save As`, configuration discovery, and model-only-session acceptance coverage.
* **Clarification**: Ordinary `Save` overwrites the exact discovered configuration path and never creates a new project-root copy. When no file is active, it requires `Save As`; `Save As` is the only operation that accepts a custom destination folder and makes the written file active for the current session.
* **Implementation**: Issue 007 now serves the complete pinned ELK catalog (11 algorithms, 8 categories, 235 options), validates typed/applicable profiles, runs the selected ELK adapter with returned routes, and keeps deterministic fallback behavior visible.
* **Persistence**: Added nearest-ancestor `.archview.json` resolution with invalid-nearest diagnostics, session Apply/Reset, exact active-file Save, and explicitly confirmed custom-folder Save As. Model-only sessions remain non-persistent.
* **Review gate**: Automated issue 007 checks pass; the issue is `awaiting-human-review` for the settings origin/error states, apply/reset workflow, and graph behavior in windowed and full-canvas views.

## 2026-08-26

* **Implementation**: Issue 005 now renders validated models as canonical deterministic JSON, self-contained interactive HTML, and script-free accessible SVG. The CLI supports direct export and analyze-to-export with reference visibility/scope controls, source-embedding rejection, partial status, atomic writes, and overwrite protection.
* **Verification**: Issue 005 passed `go test ./... -count=1`, `go test -race ./...`, `go vet ./...`, `go build ./...`, `staticcheck ./...`, `golangci-lint run`, `node --check internal/viewer/web/app.js`, and `git diff --check`. HTTP export parity is not applicable because the current product has no HTTP export endpoint.
* **Delivery**: Issue 005 is awaiting explicit visual review of HTML/SVG parity with the approved local viewer; no issue closeout or commit has been performed.

* **Review**: Completed the strict code-review loop for issue 004. The final pass found no actionable P0–P2 findings after resolving stale reanalysis scene loading, invalid failed-revision replacement, invalid explicit source line ranges, and source-file cleanup.
* **Closeout**: Archived issue 004 after the user's explicit visual approval of navigation, evidence, source inspection, full-canvas spacing, viewport controls, and stable manual routing. Removed its active registry row and updated the dated issue references; issue 005 remains the active unblocked frontier.
* **Verification**: Final delivery gates passed: `go test ./... -count=1`, `go test -race ./...`, `go vet ./...`, `go build ./...`, `staticcheck ./...`, `golangci-lint run`, `node --check internal/viewer/web/app.js`, and `git diff --check`.
* **Documentation**: Added the temporary root README as a contributor-facing viewer guide covering layers, aggregation, diagnostics, tags, identity/confidence, and layout ownership.
* **Implementation**: Added explicit internal-relationship summaries for non-cycle group self-loops, preserved canonical contributor/evidence IDs, and separated stable node identity from relationship confidence in the scene and inspection UI.
* **Implementation**: Issue 006 now serves pinned elkjs 0.12.0 and its worker locally, runs the ELK layered layout, consumes returned edge sections/bend points in the SVG renderer, and retains the deterministic layer layout as fallback.
* **Planning**: Follow-up issue 006 remains under Explore and inspect architecture and is awaiting visual review of the new routing and summaries; the canonical model boundary is unchanged.
* **Bug fix**: Made the SVG edge hit-area stroke-only and transparent. The previous default SVG fill painted black closed shapes beneath routed edges; the fix was verified in aggregated and expanded browser views.
* **Closeout**: Archived issue 003 after the user's explicit visual approval, removed its active registry row, updated the dated OKF issue reference, and unblocked issues 004 and 005. Issue 006 remains active for its separate visual review.
* **Closeout**: Archived issue 006 after the user's explicit visual approval of the semantic summary, ELK routing, transparent edge hit areas, and temporary canonical-cycle visual fixture. Removed its active registry row and updated the dated OKF issue reference; issues 004 and 005 remain the active unblocked frontiers.
* **Verification**: Viewer scene tests, full Go tests, race tests, vet, build, JavaScript syntax check, and diff checks pass. The baseline and issue 006 visual reviews are approved.

## 2026-08-26

* **Planning**: Added issue 008 for a bounded parent-level ELK option-support tranche and issue 009 for target-aware node/edge option mapping under [Explore and inspect architecture](/capabilities/explore-architecture.md). Issue 008 is blocked by 007 and issue 009 is blocked by 008.
* **No impact**: The follow-up issues remain inside the existing viewer capability and preserve the current `arch-view.config/v1` boundary. Advanced ports, labels, junctions, and compound-graph geometry remain a later specification frontier because they would change the scene/renderer contract.
* **Review**: Completed the strict code-review loop for issue 007. The final pass found no actionable P0–P2 findings after adding required-algorithm validation, enforcing layout-request size limits, and serializing configuration writes with session updates. Repository checks and JavaScript syntax checks pass.
* **Closeout**: Archived issue 007 after the user's explicit visual approval of the settings surface, origin/error states, apply/reset workflow, and windowed/full-canvas graph behavior. Removed its active registry row, unblocked issue 008, and synchronized the capability, implementation-slice, index, and orchestration references. The [upstream reference repository](https://github.com/unclebob/arch-view) remains untouched.

## 2026-08-25

* **Implementation**: Completed issue 003's local-first viewer refinement. The scene contract now supports hidden, aggregated, and expanded reference visibility, scope-aware reference nodes/relations, confidence-aware import details, and reference summaries without mutating the canonical model.
* **Browser surface**: Added the reference visibility control, boundary summary, import/details list, readable bounded graph surface, and accessible scope/confidence presentation.
* **Verification**: `go test ./... -count=1`, `go test -race ./...`, `go vet ./...`, `go build ./...`, `golangci-lint run ./...`, `staticcheck ./...`, and `node --check internal/viewer/web/app.js` passed. The issue is now `awaiting-human-review` for visual approval at the running local session.

## 2026-08-25

* **Brownfield refinement**: Visual review of issue 003 found that expanding every non-local reference overwhelms the local architecture overview and that the baseline layout is not yet navigable.
* **Routing**: Kept the work inside the specified Explore and inspect architecture capability; issue 003 owns the local-first overview and reference visibility baseline, issue 004 owns imports/evidence inspection plus navigation and session layout, and issue 005 owns export parity. No new capability node or shared-concern promotion was needed.
* **Specification sync**: Refreshed the viewer and export PRDs, domain/use-case models, contracts, scenarios, glossary, gap/readiness/orchestration artifacts, and application synthesis with hidden/aggregated/expanded reference visibility, scope/confidence distinction, imports list behavior, and session-owned layout rules.
* **Delivery**: Returned issue 003 from `awaiting-human-review` to `ready-for-agent`; issues 004 and 005 remain blocked by it.

## 2026-08-25

* **Completion**: Archived issue 001 as docs/agents/issues/done/20260825-001-analyzer-host-and-go-project-selection.md.
* **Implementation**: Added the in-process analyzer registry, manifest/API compatibility checks, deterministic listing, explicit/automatic selection, option precedence, cancellation/error containment, result validation, Go module/workspace selection, and the analyzers/analyze CLI boundary.
* **Verification**: go test ./..., go test -race ./..., go vet ./..., and CLI smoke checks passed.
* **Progress**: Issue 002 is now unblocked; the plugin runtime and Go analysis nodes remain specified because their scoped implementation work is not exhausted.

## 2026-08-25

* **Frontier selection**: Selected the specified Go analysis child as the first implementation frontier because it is the first supported language and reaches the intended visible product journey.
* **Slice creation**: Created the vertical Go repository to visible architecture view slice, spanning analyzer host, Go package/import analysis, canonical model, local viewer, evidence, and deterministic export.
* **Delivery**: Created ready-for-agent issues 001 through 005 with dependency order, ownership, acceptance criteria, contract/scenario traceability, and visual-review gates for viewer/export work.
* **Artifact sync**: Linked the implementation slice and issue paths from the affected .okf capability nodes and updated orchestration statuses. Product and application-architecture behavior was unchanged.

## 2026-08-25

* **State change**: Moved all 10 bounded capability nodes to `specified` after the architecture specification pipeline produced PRDs, glossaries, canonical domain/use-case models, contracts, acceptance scenarios, and readiness reviews.
* **Update**: Specified the shared analysis/model contracts, Go/Python/TypeScript/Rust/Clojure adapters, plugin runtime, local web viewer, and JSON/HTML/SVG export behavior.
* **Progress**: State totals are now 0 `foggy`, 0 `bounded`, 10 `specified`, and 0 `implemented`.
* **No impact**: No delivery issues or ADRs were created. Issue slicing remains downstream of the synchronized application synthesis gate.

## 2026-08-25

* **State change**: Moved [Explore and inspect architecture](/capabilities/explore-architecture.md) and [Export and automate](/capabilities/export-and-automate.md) from `foggy` to `bounded` after reviewing their visual, interaction, export, and automation questions together.
* **Update**: Confirmed a local web surface, renderer-neutral view contract, progressive disclosure, accessible evidence inspection, versioned JSON, deterministic HTML/SVG artifacts, and CI-safe partial/fatal status behavior.
* **Progress**: State totals are now 0 `foggy`, 10 `bounded`, 0 `specified`, and 0 `implemented`.
* **No impact**: No delivery issues or ADRs were created; exact capability specifications remain gated.

## 2026-08-25

* **State change**: Moved the analyzer/plugin child territories under [Analyze source code](/capabilities/analyze-source.md) from `foggy` to `bounded`.
* **Update**: Recorded staged plugin runtime rules and bounded decisions for Go, Python, TypeScript, Rust, and Clojure analysis.
* **Progress**: State totals are now 2 `foggy`, 8 `bounded`, 0 `specified`, and 0 `implemented`.
* **Pause point**: Exploration, graphics/user visualization, and export behavior remain foggy by design until those questions are reviewed together.

## 2026-08-25

* **State change**: Moved [Generate architecture models](/capabilities/generate-models.md) from `foggy` to `bounded` after confirming separate structural hierarchy, stable module identity, typed relationships, evidence/provenance, non-local references, cycle preservation, derived layers, and deterministic normalization.
* **Update**: Added discovery notes and a requirements gap analysis for the bounded model capability.
* **No impact**: No delivery issues or ADRs were created; exact schema and algorithm specification remain gated.

## 2026-08-25

* **State change**: Moved [Analyze source code](/capabilities/analyze-source.md) from `foggy` to `bounded` after clarifying its purpose, boundary, inputs, outputs, analyzer contract, and safety rules.
* **Update**: Added discovery notes and a requirements gap analysis for the bounded capability.
* **Update**: Refreshed the application PRD and architecture summary with the confirmed analyzer decisions.
* **Progress**: State totals are 9 `foggy`, 1 `bounded`, 0 `specified`, and 0 `implemented`.
* **No impact**: No delivery issues or ADRs were created. Exact specification and issue slicing remain gated.

## 2026-08-25

* **Update**: Added Rust analysis as a foggy child capability under [Analyze source code](/capabilities/analyze-source.md).
* **Update**: Synchronized the application PRD, architecture summary, orchestration status, and project scope with the Rust roadmap addition.
* **No impact**: No delivery issues or ADRs were created; the new language territory remains unimplemented.

## 2026-08-25

* **Creation**: Created the confirmed Arch View planning graph with one project concept, four top-level capabilities, and five analyzer-support child capabilities.
* **Update**: Set the root verification policy to `when-supported` with justified deferrals required.
* **Update**: Created the initial application PRD and application architecture summary.
* **No impact**: No implementation issues, ADRs, or delivery references were created because all capability nodes remain `foggy`.

## 2026-08-25

* **Completion**: Archived issue 002, [Go package/import analysis and canonical model pipeline](../docs/agents/issues/done/20260825-002-go-package-import-model-pipeline.md), after implementing deterministic Go observations, canonical model normalization/validation, graph derivations, hierarchy projection, and headless CLI access.
* **Artifact sync**: Updated the analyze-source and generate-models capability issue references, implementation/orchestration status, active issue blockers, and acceptance traceability. Product and application-architecture truth remain unchanged because the analyzer-to-model boundary is unchanged.
* **Verification**: Backend and end-to-end obligations passed where supported; frontend integration is not applicable to this backend/model issue. The external reference folder remains untouched.
