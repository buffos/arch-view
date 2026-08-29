# External analyzer plugin implementation slice: process runtime to Python parity

## Selected frontier

The [Analyzer plugin runtime capability](../../../../.okf/capabilities/analyze-source/plugin-runtime.md) is a readiness-reviewed specified child of [Analyze source code](../../../../.okf/capabilities/analyze-source.md). The in-process host, selection rules, options, result validation, and five built-in adapters are implemented; the first external process boundary has been completed and validated by an external Python adapter.

This slice does not add a language or a capability node. It ports the existing Python analysis semantics to a separately launched Python process and proves that the same language-neutral model, viewer, and export contracts still apply. The existing in-process Python analyzer remains available as the parity baseline and fallback.

## Current and target truth

- **Observed in code:** the common analysis.Analyzer contract, deterministic host selection, option resolution, result validation, canonical model normalization, local viewer, and JSON/HTML/SVG paths exist. Issues 030–035 add the published protocol/descriptor schemas, bounded typed NDJSON codec, strict descriptor validation, stateful session validator, argv-only process adapter, external Python deployment, public descriptor loader, shared opt-in path, five compiled entrypoints, and deterministic package/release assembly.
- **Inferred from the exact specification:** an external analyzer needs version negotiation, manifest agreement, detection and analysis operations, canonical result validation, diagnostics, cancellation, timeout/size limits, protocol-only stdout, and log-only stderr.
- **User-confirmed target:** the first external deployment is an existing Python analyzer port, not a sixth language. It must be explicitly opted into and must not require model, layout, viewer, or exporter changes.
- **Verified implementation:** the process host lifecycle, external Python implementation, public opt-in path, compiled entrypoints, trusted package catalog, packaged runtime selection, explicit fallback policy, and public packaged path are complete; packaged and in-process results reach the unchanged model, viewer/source, and export consumers.

## Next future frontiers

The remaining user-confirmed target beyond this completed slice is represented
by two specified child capabilities rather than being folded into the
implemented runtime/distribution work. The multi-analyzer child now has an
approved implementation sequence; the assignment/view child remains a later
frontier:

- [Multi-analyzer project orchestration](../../../../.okf/capabilities/analyze-source/plugin-runtime/multi-analyzer-orchestration.md)
  plans and merges concurrent analyzer jobs for mixed or nested projects.
- [Project analyzer assignments and view selection](../../../../.okf/capabilities/analyze-source/plugin-runtime/project-analyzer-assignments.md)
  adds repository-relative assignments and application scope switching.

The [compiled external analyzer distribution](../../../../.okf/capabilities/analyze-source/plugin-runtime/compiled-external-analyzer-distribution.md)
child is implemented. Its Linux amd64 and Darwin arm64 execution checks remain
explicitly deferred under the root `when-supported` policy until matching
runners/toolchains are available.

## Vertical outcome

Given an explicit external-plugin descriptor and a Python repository, a developer can:

1. list and explicitly select the external Python analyzer without changing the built-in registry behavior;
2. launch one isolated process per detection or analysis request through an argv-based command, never through a shell;
3. exchange versioned NDJSON frames with a hello handshake, detection candidate, canonical analysis result, diagnostics, cancellation, and terminal status;
4. receive the same project boundary, module identities, static imports, references, evidence, diagnostics, partial-result behavior, and option precedence as the in-process Python baseline;
5. see protocol, timeout, cancellation, and process failures as stable host errors without accepting invalid or late results; and
6. pass the external result through the unchanged canonical model, local viewer, deterministic JSON/HTML/SVG export, and source-evidence paths.

## Scope and protocol decisions

- External plugins are loaded only from an explicitly supplied local descriptor. There is no implicit PATH scan, project-file execution, or target-repository plugin discovery in this slice.
- The descriptor contains a validated manifest plus an argv command and optional working directory. The process must echo the same manifest in its hello frame.
- One process handles one request and exits after one done or fatal frame. Detection and analysis are separate operations so the existing Analyzer contract remains intact.
- Every stdout line is one JSON frame. The first frame is hello. Detection uses detect and candidate; analysis uses analyze and result. Diagnostic frames are retained and merged before common result validation. Cancel is best effort before the host terminates a cancelled or timed-out process.
- Stdout is protocol-only. Stderr is bounded diagnostic log data and is never parsed as a result. Unknown frame types, malformed JSON, duplicate terminal frames, wrong request IDs, manifest mismatches, oversized frames, and invalid result envelopes are failures.
- The first external Python plugin uses the standard-library ast module, emits the existing Python observation semantics, has no third-party install step, and never imports, executes, installs, or introspects target code.
- The host owns process lifecycle, option resolution, protocol validation, result normalization, and deterministic ordering. The external analyzer owns Python project-boundary and static syntax rules.

## Ordered delivery issues

| Issue | Outcome | Owner | Blocked by | Review gate |
|---|---|---|---|---|
| [030](../../../agents/issues/done/20260827-030-external-protocol-schema-and-conformance-fixture.md) | Completed: publish the v1 protocol and descriptor schemas, bounded wire-frame rules, and a test-only process fixture covering valid and invalid streams | Plugin runtime contract | none | none |
| [031](../../../agents/issues/done/20260827-031-process-backed-analyzer-host-runtime.md) | Completed: Go process-backed Analyzer adapter with handshake, detection/analysis lifecycle, limits, cancellation, stderr handling, and conformance tests | Plugin runtime host | 030 | none |
| [032](../../../agents/issues/done/20260827-032-external-python-analyzer-parity.md) | Completed: external stdlib-only Python plugin with parity harness and no-target-execution evidence | External Python plugin + analysis | 031 | none |
| [033](../../../agents/issues/done/20260827-033-external-plugin-cli-and-visible-journey.md) | Completed: explicit descriptor loading for analyzers, analyze, and open with shared model/viewer/export verification | CLI + existing consumers | 032 | none |

## Approved compiled-distribution delivery sequence

The approved migration step is issue 034: port and reuse each existing Go,
Python, TypeScript, Rust, and Clojure implementation behind a compiled plugin
entrypoint before distribution assembly and packaged-runtime cutover. Issue
035 now assembles the application-managed distribution and writes the
deterministic package index. Issues 036–038 then verify package trust, make
packaged execution the default, and close release/parity verification.

| Issue | Outcome | Owner | Blocked by | Review gate |
|---|---|---|---|---|
| [034](../../../agents/issues/done/20260828-034-compiled-analyzer-plugin-entrypoints.md) | Completed: shared child-side runner and five compiled analyzer entrypoints reuse the current implementations with manifest, protocol, option, and result parity | Analyzer entrypoints | none | none |
| [035](../../../agents/issues/done/20260828-035-compiled-analyzer-distribution-assembly.md) | Completed: assemble deterministic platform packages and `analyzers/index.json` through explicit `make analyzers` and atomic host-plus-analyzers `make release` builds | Distribution build | 034 | none |
| [036](../../../agents/issues/done/20260828-036-trusted-analyzer-package-verification.md) | Completed: discover, validate, and checksum-verify only trusted application-managed packages | Package trust boundary | none | none |
| [037](../../../agents/issues/done/20260828-037-packaged-runtime-selection.md) | Completed: make verified packaged execution the default and retain explicit local/in-process migration overrides | Runtime selection | 036 | none |
| [038](../../../agents/issues/done/20260828-038-compiled-analyzer-parity-and-release-verification.md) | Completed: verify cross-platform parity, deterministic release behavior, and the full acceptance matrix, with documented non-host deferrals | Release verification | 037 | none |

## Approved multi-analyzer delivery sequence

Issues 039–043 are the approved vertical sequence for multi-analyzer project
orchestration. All five issues are verified and archived, including the
approved post-implementation `visual-review` gate for cached viewer scope
selection. Persisted `.archview.json` analyzer assignments remain owned by the
separate project-analyzer-assignments child.

| Issue | Outcome | Owner | Blocked by | Review gate |
|---|---|---|---|---|
| [039](../../../agents/issues/done/20260828-039-multi-project-root-discovery-and-job-planning.md) | Completed: discover project roots and produce a deterministic bounded analyzer job plan | Discovery and planning | none | none |
| [040](../../../agents/issues/done/20260828-040-bounded-multi-analyzer-execution.md) | Completed: execute planned jobs with bounded workers, cancellation, timeouts, and isolated diagnostics | Orchestration runtime | 039 | none |
| [041](../../../agents/issues/done/20260828-041-namespaced-aggregate-model-and-status.md) | Completed: merge successful and partial scope results into namespaced aggregate models and statuses | Aggregate model | 040 | none |
| [042](../../../agents/issues/done/20260828-042-combined-analysis-cli-and-http-exposure.md) | Completed: expose combined analysis, status, scope, and cached projection contracts through CLI and HTTP | CLI and HTTP | 041 | none |
| [043](../../../agents/issues/done/20260828-043-cached-scope-projections-and-viewer-selection.md) | Completed: expose cached All/individual scope projections and failed-scope diagnostics in the local viewer, with approved visual review | Viewer integration | 042 | visual-review |

## Approved project-assignment delivery sequence

Issues 044–047 are the approved vertical sequence for the specified project
analyzer assignments and view-selection child. Issues 044–046 are verified and
archived; issue 047 has completed automated verification and remains
`awaiting-human-review` for its declared visual gate. They reuse the completed
multi-analyzer planner, aggregate, transport, and viewer projection seams;
they do not add a new capability node or duplicate issue 043's selector.

| Issue | Outcome | Owner | Blocked by | Review gate |
|---|---|---|---|---|
| [044](../../../agents/issues/done/20260829-044-load-and-validate-analysis-configuration.md) | Completed: load v1/v2 `.archview.json` profiles and validate assignments, filters, analyzer options, and layout preservation | Configuration boundary | none | none |
| [045](../../../agents/issues/done/20260829-045-resolve-configured-assignments-and-source-scopes.md) | Completed: resolve configured assignments/source scopes through CLI, HTTP, `open`, and reanalysis planning | Assignment and planning integration | 044 | none |
| [046](../../../agents/issues/done/20260829-046-session-cache-and-selective-invalidation.md) | Completed: reuse immutable session scope results and invalidate only affected jobs | Cache and reanalysis | 045 | none |
| [047](../../../agents/issues/pending/20260829-047-configured-scope-viewer-journey.md) | Automated implementation complete: expose configured scopes, diagnostics, and cache/reanalysis states through the existing viewer journey | Viewer integration | 046 | visual-review |

## Slice acceptance

- A valid descriptor and hello manifest use the same API version and manifest fields; invalid descriptors or mismatches fail before an analyzer can run.
- The protocol schema and fixture cover both detection and analysis, complete/partial outcomes, streamed diagnostics, cancellation, fatal errors, unknown frames, stdout contamination, malformed JSON, oversized lines, wrong request IDs, and late output.
- The process adapter cannot invoke a shell, accept an external result after cancellation, leak a child process, treat stderr as protocol, or bypass common host result validation.
- The external Python analyzer and in-process Python analyzer agree on the representative fixture's modules, relationships, references, source evidence, diagnostics, project boundary, option effects, and partial status, apart from explicitly allowed analyzer/run provenance.
- With an explicit plugin descriptor, the CLI and project-backed open path preserve selection metadata, option fingerprints, deterministic results, and the existing model normalization, validation, projection, viewer, source inspection, JSON, HTML, and SVG behavior.
- Without an external-plugin flag, all existing built-in analyzer listings, selection behavior, tests, and output remain unchanged.

## Explicit non-goals

- A new language, a replacement of the in-process Python analyzer, or implicit machine-wide plugin discovery.
- A plugin marketplace, package manager, remote process, sandbox runtime, arbitrary project configuration execution, or cross-platform installer.
- Changes to the canonical model JSON shape, relationship vocabulary, layout configuration, renderer routing, viewer UX, or export semantics.
- Runtime tracing, type graphs, call graphs, dynamic-import execution, or environment-assisted Python resolution.
- General protocol migration beyond the published v1 compatibility rules.

## Verification surfaces

- **Backend boundary:** descriptor and manifest validation, frame schema/conformance, argv-only process launch, handshake, detection, analysis, option forwarding, result validation, diagnostics, cancellation, timeout/size limits, stderr isolation, and child-process cleanup.
- **External plugin boundary:** Python syntax-only analysis, project-marker/source-root/options parity, deterministic serialization, no target-code execution, and usable partial results.
- **Compiled distribution boundary:** application-managed index validation, exact platform selection, trusted path/digest/manifest checks, packaged-by-default runtime selection, explicit fallback, five-analyzer parity, and atomic release assembly.
- **Frontend integration:** the existing language-neutral model, hierarchy projection, viewer evidence/details/source routes, and export serializers consume external Python results without language-specific branches.
- **End-to-end:** verified package or explicit descriptor → NDJSON process → analysis JSON → canonical model → local viewer and deterministic JSON/HTML/SVG artifacts.
- **Repository/OKF integrity:** full repository gates, strict OKF validation, synchronized issue references, and the untouched upstream reference boundary.

## Artifact impact

Issues 034–038 are implemented and verified: all five compiled commands reuse
the existing analyzer constructors and reach the existing process adapter and
host validation path; trusted package verification, packaged runtime selection,
explicit fallback, parity, deterministic assembly, and Windows release
behavior are complete. Linux amd64 and Darwin arm64 execution remain
documented `when-supported` deferrals. The slice refines an already confirmed
plugin-runtime boundary; it does not change product topology or canonical model
semantics. Issues 030–033 remain the completed delivery record for the v1
external Python pilot. Issues 039–043 now represent verified delivery for the
multi-analyzer planning, execution, aggregation, transport, and cached viewer
scope-selection path. Issues 044–046 are verified delivery records for the
assignment/configuration frontier; issue 047 remains active for its required
visual review. The application PRD and application architecture summary
require no semantic change for this delivery slicing; their product
actors, workflows, and architectural boundaries remain unchanged.
