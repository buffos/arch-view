# 032 — External Python analyzer parity

## Issue Metadata

- Issue number: 032
- Owning capability node: /.okf/capabilities/analyze-source/plugin-runtime.md
- Supporting capability node: /.okf/capabilities/analyze-source/python-analysis.md
- Artifact root: docs/architecture/analyze-source/plugin-runtime/
- Supporting artifact root: docs/architecture/analyze-source/python-analysis/
- Issue file: docs/agents/issues/pending/20260827-032-external-python-analyzer-parity.md
- Category: feature
- Execution type: AFK
- Review gate: none
- Suggested state: ready-for-agent

## Parent Artifacts

- docs/architecture/analyze-source/plugin-runtime/prd.md
- docs/architecture/analyze-source/plugin-runtime/canonical-domain-model.md
- docs/architecture/analyze-source/plugin-runtime/canonical-use-cases.md
- docs/architecture/analyze-source/plugin-runtime/canonical-api-cli-contract.md
- docs/architecture/analyze-source/plugin-runtime/acceptance-scenarios.md
- docs/architecture/analyze-source/plugin-runtime/implementation-slice.md
- docs/architecture/analyze-source/python-analysis/prd.md
- docs/architecture/analyze-source/python-analysis/canonical-domain-model.md
- docs/architecture/analyze-source/python-analysis/canonical-api-cli-contract.md
- docs/architecture/analyze-source/python-analysis/acceptance-scenarios.md
- docs/architecture/analyze-source/python-analysis/implementation-slice.md
- docs/agents/issues/done/20260827-030-external-protocol-schema-and-conformance-fixture.md
- docs/agents/issues/pending/20260827-031-process-backed-analyzer-host-runtime.md

## What to build

Create the first real external analyzer by porting the implemented Python
static-analysis semantics to a separately launched, standard-library-only
Python plugin. Place the external deployment under plugins/python-analyzer/
with its descriptor, launcher, source, and invocation documentation; keep
the existing in-process adapter under internal/analyzers/python as the
semantic parity baseline and fallback.

The external plugin must use Python ast and safe data readers for project
configuration. It must not import, execute, install, type-check, or
introspect target code. Its only output is the issue 030 protocol on stdout;
human logs go to stderr.

## Acceptance criteria

- [ ] The external descriptor has a stable external manifest ID,
  org.archview.python.external, language python, API arch-view.analyzer/v1,
  the required detection markers, and the Python option descriptors needed
  by the current adapter: source_roots, python_version, include_stubs,
  include_tests, and exclude.
- [ ] The plugin responds to detect and analyze with the protocol handshake,
  returns a valid detection candidate, forwards effective options without
  lossy renaming, emits a canonical complete or partial result, and exits
  with one terminal done or fatal frame.
- [ ] Safe project-boundary behavior matches the in-process baseline:
  pyproject.toml/setup.cfg/setup.py marker precedence, configured or src/
  source roots, repository-root fallback, regular and namespace packages,
  stubs/tests/exclusions, stable module IDs, and repository-relative source
  evidence.
- [ ] Static imports match the baseline fixture semantics for absolute,
  relative, package-init re-export, standard-library, external,
  unresolved, conditional, and dynamic imports. Uncertain behavior remains
  references, confidence, or recoverable diagnostics rather than invented
  local relationships.
- [ ] A parity harness compares the external and in-process analyzers after
  removing only allowed run/analyzer provenance. Modules, relationships,
  references, source evidence, project boundary, option effects, diagnostic
  codes/severity/recoverability, and complete/partial status must agree for
  the representative fixture.
- [ ] Repeated external runs with unchanged source, descriptor, interpreter,
  options, and protocol version produce byte-stable analysis JSON after
  common host normalization.
- [ ] A target fixture containing import-time and file-write side effects
  proves that the external analyzer never executes target code; the
  sentinel file is not created and the result remains static-analysis data.
- [ ] The external plugin contains no viewer, model-normalization, layout,
  or export-specific code. Existing in-process Python behavior and all
  repository tests remain green.

## Artifact sync required

- Application PRD: none — this is the confirmed external deployment of the
  implemented Python capability, not a new product language or workflow.
- Application architecture summary: required only if the implementation
  exposes a process-boundary mismatch; issue 033 owns the consolidated
  external deployment status.
- Owning/supporting capability artifacts: required:
  /.okf/capabilities/analyze-source/plugin-runtime.md;
  /.okf/capabilities/analyze-source/python-analysis.md;
  docs/architecture/analyze-source/orchestration-status.md;
  docs/architecture/analyze-source/plugin-runtime/orchestration-status.md;
  docs/architecture/analyze-source/plugin-runtime/implementation-slice.md;
  docs/architecture/analyze-source/python-analysis/orchestration-status.md.
- Issue registry: required; both owning capability issue lists must reference
  the pending issue until closeout.
- Reason/no-impact decision: the external deployment adds a second runtime
  packaging for already implemented Python semantics while preserving the
  language-neutral result, model, viewer, and export contracts.

## Human review gate

None. The external plugin changes analysis/runtime behavior but does not
change rendered UI/UX or navigation.

## Blocked by

Blocked by issue 031:
docs/agents/issues/pending/20260827-031-process-backed-analyzer-host-runtime.md

## Artifact anchors

- Python child contract: project markers, source-root precedence, static AST
  imports, uncertainty, evidence, diagnostics, and no-evaluation safety.
- Plugin-runtime contract: manifest, options, detection, process frames,
  canonical result, and terminal status.
- The existing Python implementation slice: issues 017–019 define the
  semantic baseline that this external runtime must reproduce.

## Acceptance scenarios addressed

- SC-PY-001 through SC-PY-005
- SC-AS-001 — Analyze a resolvable project
- SC-AS-002 — Preserve a partial result
- SC-AS-004 — Honor explicit selection
- SC-AS-005 — Retain multi-file evidence
- SC-AS-006 — Enforce exclusions and safety
- SC-AS-008 — Deterministic repeat
- SC-PR-003 — Resolve unambiguous detection
- SC-PR-005 — Apply option precedence
- SC-PR-006 — Cancel safely

## Verification obligations

- Policy source: /.okf/project.md

| Scenario | Backend boundary | Frontend integration | End-to-end journey |
|---|---|---|---|
| SC-PY-001/002 | planned: external marker/configuration and module-discovery tests | not-applicable | not-applicable |
| SC-PY-003/004 | planned: external static import/uncertainty/evidence tests and parity harness | not-applicable | not-applicable |
| SC-PY-005 | planned: AST-only side-effect sentinel and malformed-input recovery tests | not-applicable | not-applicable |
| SC-AS-001/002/004/005/006/008 | planned: process adapter integration with common host normalization and deterministic repeated output | not-applicable | deferred to issue 033 |

## Handoff

Issue 033 may begin after the external plugin passes parity and direct
process-adapter tests. The CLI integration should load the descriptor
explicitly and must not make the external Python implementation the default
for ordinary Python projects.
