# External analyzer plugin implementation slice: process runtime to Python parity

## Selected frontier

The [Analyzer plugin runtime capability](../../../../.okf/capabilities/analyze-source/plugin-runtime.md) is a readiness-reviewed specified child of [Analyze source code](../../../../.okf/capabilities/analyze-source.md). The in-process host, selection rules, options, result validation, and five built-in adapters are implemented; the next confirmed frontier is an opt-in external process boundary validated by an external Python adapter.

This slice does not add a language or a capability node. It ports the existing Python analysis semantics to a separately launched Python process and proves that the same language-neutral model, viewer, and export contracts still apply. The existing in-process Python analyzer remains available as the parity baseline and fallback.

## Current and target truth

- **Observed in code:** the common analysis.Analyzer contract, deterministic host selection, option resolution, result validation, canonical model normalization, local viewer, and JSON/HTML/SVG paths exist. Registry registration is still in-process only, while issue 030 now adds the published protocol/descriptor schemas, bounded typed NDJSON codec, strict descriptor validation, stateful session validator, and test-only subprocess fixture. The process adapter, public descriptor loader, and shared opt-in path remain pending.
- **Inferred from the exact specification:** an external analyzer needs version negotiation, manifest agreement, detection and analysis operations, canonical result validation, diagnostics, cancellation, timeout/size limits, protocol-only stdout, and log-only stderr.
- **User-confirmed target:** the first external deployment is an existing Python analyzer port, not a sixth language. It must be explicitly opted into and must not require model, layout, viewer, or exporter changes.
- **Target mismatch:** the process host lifecycle, external Python implementation, and public opt-in path are still absent; the protocol/descriptor schema and conformance-fixture portion is implemented by issue 030.

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
| [031](../../../agents/issues/pending/20260827-031-process-backed-analyzer-host-runtime.md) | Add the Go process-backed Analyzer adapter with handshake, detection/analysis lifecycle, limits, cancellation, stderr handling, and conformance tests | Plugin runtime host | 030 | none |
| [032](../../../agents/issues/pending/20260827-032-external-python-analyzer-parity.md) | Port the existing Python static-analysis semantics to an external stdlib-only Python plugin and compare it with the in-process baseline | External Python plugin + analysis | 031 | none |
| [033](../../../agents/issues/pending/20260827-033-external-plugin-cli-and-visible-journey.md) | Add explicit descriptor loading to analyzers, analyze, and open; prove shared model/viewer/export behavior and synchronize final application records | CLI + existing consumers | 032 | none |

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
- **Frontend integration:** the existing language-neutral model, hierarchy projection, viewer evidence/details/source routes, and export serializers consume external Python results without language-specific branches.
- **End-to-end:** descriptor → external Python process → analysis JSON → canonical model → local viewer and deterministic JSON/HTML/SVG artifacts.
- **Repository/OKF integrity:** full repository gates, strict OKF validation, synchronized issue references, and the untouched upstream reference boundary.

## Artifact impact

The slice refines an already confirmed plugin-runtime boundary; it does not change product topology or canonical model semantics. The capability exact-spec set, application PRD, application architecture summary, parent orchestration, capability references, issue registry, and OKF log are synchronized to make the external Python pilot the next delivery frontier. Issue 033 owns the consolidated final application-status refresh after the shared public path is verified.
