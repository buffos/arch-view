# 033 — External plugin CLI and shared visible journey

## Issue Metadata

- Issue number: 033
- Owning capability node: /.okf/capabilities/analyze-source/plugin-runtime.md
- Supporting capability node: /.okf/capabilities/analyze-source/python-analysis.md
- Artifact root: docs/architecture/analyze-source/plugin-runtime/
- Issue file: docs/agents/issues/done/20260827-033-external-plugin-cli-and-visible-journey.md
- Category: feature
- Execution type: AFK
- Review gate: none
- Suggested state: done

## Parent Artifacts

- docs/prd.md
- docs/architecture/application-architecture-summary.md
- docs/architecture/analyze-source/prd.md
- docs/architecture/analyze-source/orchestration-status.md
- docs/architecture/analyze-source/plugin-runtime/prd.md
- docs/architecture/analyze-source/plugin-runtime/canonical-domain-model.md
- docs/architecture/analyze-source/plugin-runtime/canonical-use-cases.md
- docs/architecture/analyze-source/plugin-runtime/canonical-api-cli-contract.md
- docs/architecture/analyze-source/plugin-runtime/acceptance-scenarios.md
- docs/architecture/analyze-source/plugin-runtime/readiness-review.md
- docs/architecture/analyze-source/plugin-runtime/implementation-slice.md
- docs/architecture/analyze-source/python-analysis/implementation-slice.md
- docs/agents/issues/done/20260827-032-external-python-analyzer-parity.md

## What to build

Expose explicit external-plugin descriptors through the public CLI and
project-backed open flow. Add repeatable --plugin descriptor support to
analyzers, analyze, and open. Load and validate descriptors before registry
use, register process-backed analyzers deterministically, preserve the
existing explicit/automatic selection rules, and carry the loaded plugin
registry into open-session reanalysis.

Prove the unchanged visible product journey with the external Python
analyzer: descriptor → process analysis → analysis JSON → canonical model
normalization/validation/projection → local viewer/source inspection and
deterministic JSON/HTML/SVG artifacts. The external analyzer is opt-in; when
no descriptor is supplied, the five existing built-in analyzers and their
outputs remain unchanged.

## Acceptance criteria

- [x] analyzers, analyze, and open accept repeatable --plugin <descriptor>
  options. Descriptor paths are normalized and validated before any analysis
  starts; malformed descriptors, invalid manifests, duplicate IDs, missing
  commands, and hello mismatches return stable errors without partial
  registration.
- [x] arch-view analyzers --plugin <descriptor> lists built-in and external
  manifests in deterministic ID order. Without --plugin it returns the same
  five built-in manifests and bytes as before.
- [x] Explicit --analyzer org.archview.python.external selects the external
  Python process and records stable analyzer, selection, boundary, run, and
  option-fingerprint metadata. Explicit language and auto-detection preserve
  the common host contract; when built-in and external Python candidates tie,
  automatic selection fails as the existing ambiguity rule requires.
- [x] analyze forwards every selected analyzer option through the common
  precedence and validation path and produces a valid analysis-json result
  with no external protocol fields leaking into canonical output.
- [x] open --project retains the descriptor-backed analyzer for reanalysis,
  cancellation, and error reporting. A missing or invalid descriptor is
  reported clearly rather than silently falling back to the in-process
  analyzer.
- [x] A representative external Python project reaches model normalize,
  model validate, model projection, viewer model/scene/source inspection,
  and the existing JSON, self-contained HTML, and SVG outputs without a
  Python-specific branch in internal/model, internal/viewer, layout, or
  export code.
- [x] The external and in-process Python paths expose equivalent hierarchy,
  directed relationships, references, evidence, confidence, diagnostics,
  and partial-result semantics. Repeated CLI/model/export runs are
  byte-stable under unchanged inputs.
- [x] Existing Go, Python in-process, TypeScript, Rust, and Clojure CLI,
  model, viewer, export, deterministic-output, and safety tests remain
  green. No target repository code is executed and no external Python
  runtime or source is embedded in exported HTML/SVG.
- [x] After issues 030–033 pass their verification obligations, update the
  plugin-runtime and parent orchestration records, the application PRD and
  architecture summary, the implementation slice, capability issue lists,
  issue registry, and OKF log. Keep the plugin-runtime node specified until
  its next external capability frontier is explicitly chosen.

## Artifact sync required

- Application PRD: required: docs/prd.md — record the opt-in external
  descriptor workflow, the external Python pilot, and the updated
  implementation sequence without adding a new language or changing the
  static-analysis product boundary.
- Application architecture summary: required:
  docs/architecture/application-architecture-summary.md — record the
  descriptor/process/NDJSON boundary, host ownership, stderr/stdout safety,
  and unchanged model/viewer/export consumers.
- Owning/supporting capability artifacts: required:
  /.okf/capabilities/analyze-source.md;
  /.okf/capabilities/analyze-source/plugin-runtime.md;
  /.okf/capabilities/analyze-source/python-analysis.md;
  docs/architecture/analyze-source/orchestration-status.md;
  docs/architecture/analyze-source/plugin-runtime/orchestration-status.md;
  docs/architecture/analyze-source/python-analysis/orchestration-status.md;
  docs/architecture/analyze-source/plugin-runtime/implementation-slice.md.
- Issue registry: required; archive issue files only after acceptance and
  remove their active registry rows at closeout.
- Reason/no-impact decision: the public opt-in process route changes
  extensibility/deployment maturity but leaves the canonical model, viewer,
  export schemas, static-analysis scope, and built-in behavior unchanged.

## Human review gate

None. This slice reuses the existing viewer and export contracts without
changing rendered UI/UX or navigation behavior.

## Blocked by

Satisfied by archived issue 032:
docs/agents/issues/done/20260827-032-external-python-analyzer-parity.md

## Artifact anchors

- Plugin-runtime PRD requirements PR-FR-001 through PR-FR-011.
- Canonical API/CLI contract: --plugin mapping, descriptor validation,
  selection parity, external protocol, and error/exit-code behavior.
- Python implementation slice: external output must be semantically
  equivalent to the completed in-process Python path.
- Existing shared consumer contracts: canonical model, viewer scene/source
  routes, and JSON/HTML/SVG exporters.

## Acceptance scenarios addressed

- SC-PR-001 through SC-PR-011
- SC-PY-001 through SC-PY-005
- SC-AS-001, SC-AS-002, SC-AS-004, SC-AS-005, SC-AS-006, and SC-AS-008

## Verification obligations

- Policy source: /.okf/project.md

| Scenario | Backend boundary | Frontend integration | End-to-end journey |
|---|---|---|---|
| SC-PR-001/002/008/009 | passed: descriptor loading, manifest listing, duplicate/mismatch, and no-flag regression tests | not-applicable | passed: external manifest through CLI |
| SC-PR-003/004/005 | passed: explicit/automatic selection, ambiguity, options, and stable metadata tests | not-applicable | passed: external Python analysis JSON |
| SC-PR-006/007/010/011 | passed by the process adapter conformance and lifecycle suite | not-applicable | passed by external process integration |
| SC-PY-001 through SC-PY-005 | passed by the issue 032 parity and safety harness | not-applicable | passed: external Python shared path |
| SC-AS-001/002/004/005/006/008 | passed: common host/model/export assertions | passed: existing viewer model/scene/source routes | passed: external Python repository to viewer and JSON/HTML/SVG |

## Implementation and verification

- Added repeatable `--plugin` descriptor loading to `analyzers`, `analyze`,
  and project-backed `open`; descriptors are normalized and fully validated
  before atomic registration, and the same host registry is retained by open
  reanalysis callbacks.
- Kept the default five built-in analyzer listing and selection behavior
  unchanged. External Python selection remains explicit, while automatic
  built-in/external ties remain ambiguous.
- Verified the shared external journey through analysis JSON, canonical
  normalization/validation/projection, viewer model/scene/source routes, and
  deterministic JSON, HTML, and SVG exports without consumer-specific plugin
  branches.
- Verification passed: focused external Python/CLI tests, `go test ./...
  -count=1`, `go vet ./...`, strict OKF validation, and `git diff --check`.

## Closeout expectations

The issue is complete only when the external Python path is independently
usable, parity evidence is recorded, the unchanged built-in path is
regression-tested, and all application/capability/delivery artifacts agree.
If a later language or discovery mechanism is desired, create a new
fog-clearing or implementation-slice decision rather than extending this
pilot implicitly.
