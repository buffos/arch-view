# 044 — Load and validate versioned analysis configuration

## Issue Metadata

- Issue number: `044`
- Owning capability node: `/.okf/capabilities/analyze-source/plugin-runtime/project-analyzer-assignments.md`
- Related consumer node: `/.okf/capabilities/explore-architecture.md`
- Artifact root: `docs/architecture/analyze-source/plugin-runtime/project-analyzer-assignments/`
- Issue file: `docs/agents/issues/done/20260829-044-load-and-validate-analysis-configuration.md`
- Category: `feature`
- Execution type: `AFK`
- Review gate: `none`
- Suggested state: `done`

## Parent Artifacts

- `docs/prd.md`
- `docs/architecture/application-architecture-summary.md`
- `docs/architecture/analyze-source/plugin-runtime/project-analyzer-assignments/prd.md`
- `docs/architecture/analyze-source/plugin-runtime/project-analyzer-assignments/domain-glossary.md`
- `docs/architecture/analyze-source/plugin-runtime/project-analyzer-assignments/canonical-domain-model.md`
- `docs/architecture/analyze-source/plugin-runtime/project-analyzer-assignments/canonical-use-cases.md`
- `docs/architecture/analyze-source/plugin-runtime/project-analyzer-assignments/canonical-api-cli-contract.md`
- `docs/architecture/analyze-source/plugin-runtime/project-analyzer-assignments/acceptance-scenarios.md`
- `docs/architecture/analyze-source/plugin-runtime/project-analyzer-assignments/readiness-review.md`
- `docs/architecture/explore-architecture/canonical-api-cli-contract.md`
- `docs/architecture/analyze-source/plugin-runtime/multi-analyzer-orchestration/canonical-api-cli-contract.md`

## What to build

Implement the shared configuration boundary for the nearest complete
`.archview.json`. It must decode the existing `arch-view.config/v1` layout-only
profile and the strict `arch-view.config/v2` profile without changing layout
semantics, returning a validated analysis configuration for downstream
planning. Preserve the nearest-file-wins rule: an invalid nearest file is an
actionable error and must not fall through to a valid ancestor.

Validate repository-relative assignment paths, duplicate normalized paths,
analyzer-scoped include rules, global exclusion globs, option shapes, and
manifest compatibility before any target analyzer executes. Keep an assigned
but unavailable logical analyzer as an explicit diagnostic-ready assignment so
the planner can retain that failed scope rather than silently replacing it.
Reject unsafe or ambiguous configuration, avoid exposing sensitive option
values in diagnostics, and ensure layout configuration writes or round-trips
cannot silently erase a v2 `analysis` section.

## Acceptance criteria

- [x] The loader accepts existing v1 layout-only files with unchanged layout
  behavior and treats analysis as absent; it accepts the exact v2 `analysis`
  shape without merging configuration files or changing layout/model facts.
- [x] Configuration discovery selects the nearest `.archview.json` from the
  invocation/repository root toward the filesystem root; an invalid nearest
  file returns a stable configuration error and does not use a farther file.
- [x] Assignment paths are normalized POSIX repository-relative paths, with
  `.` representing the root; absolute paths, parent traversal, empty segments,
  glob characters, and duplicate normalized paths are rejected with field/path
  details.
- [x] Global excludes and analyzer-scoped includes accept only the documented
  deterministic glob subset, reject unsafe or malformed patterns and duplicate
  analyzer rules, and preserve invocation-root anchoring independently of the
  configuration-file and process working directories.
- [x] Analyzer IDs and option shapes are checked against available manifests
  without executing target code. Unavailable assigned IDs remain represented by
  a scoped `analysis_analyzer_unavailable` diagnostic input rather than being
  dropped or automatically reassigned.
- [x] Configuration diagnostics identify the failing field or pattern without
  leaking sensitive option values; canonicalized assignments and filter rules
  have deterministic ordering and fingerprints.
- [x] Existing layout save/round-trip paths preserve a valid v2 `analysis`
  section or refuse the mutation explicitly; they never downgrade the file to
  v1 by silently discarding assignment or filter data.
- [x] Focused tests cover v1 compatibility, v2 decoding, nearest-file errors,
  unsafe/duplicate paths, glob validation, manifest/options validation,
  unavailable analyzers, deterministic canonicalization, sensitive-value
  redaction, and layout/analysis separation.

## Artifact sync required

- Application PRD: `none` — this issue realizes the already synchronized
  configuration workflow and changes no product scope, actor, or business rule.
- Application architecture summary: `none` — the resolver boundary,
  layout/analysis separation, and plugin-manager ownership are already defined;
  implementation must record any discovered contract change before closure.
- Owning capability node/artifacts: `required: /.okf/capabilities/analyze-source/plugin-runtime/project-analyzer-assignments.md` and `docs/architecture/analyze-source/plugin-runtime/project-analyzer-assignments/orchestration-status.md`; exact-spec documents require no semantic edits.
- Issue registry: `required`; node `issues:` reference: `required when .okf exists`.
- Reason/no-impact decision: delivery truth is created by this issue and its
  node reference. Product, architecture, and topology remain unchanged because
  the implementation is bounded by the readiness-reviewed v2 contract.

## Human review gate

None. This slice has no rendered UI/UX change and is independently verifiable
through configuration, manifest, and layout-preservation tests.

## Blocked by

None — the exact configuration contract, analyzer manifests, and layout
configuration boundary are already available.

## Artifact anchors

- `PAA-FR-001`, `PAA-FR-002`, `PAA-FR-004`, `PAA-FR-005`, `PAA-FR-007`, and `PAA-FR-012`.
- The v1/v2 schema, nearest-file, path-safety, glob, manifest-validation, and
  layout-separation rules in `canonical-api-cli-contract.md`.
- `SC-PAA-001`, `SC-PAA-004`, `SC-PAA-005`, `SC-PAA-010`, and `SC-PAA-013`.
- The existing layout configuration discovery/session boundary, extended
  without moving analysis semantics into the viewer.

## User stories addressed

The capability PRD has no numbered user-story section. This slice supports
application journeys 10 and 11 by making the durable configuration contract
loadable and safely diagnosable before planning.

## Verification obligations

- Policy source: `/.okf/project.md`

| Scenario | Backend boundary | Frontend integration | End-to-end journey |
| --- | --- | --- | --- |
| `SC-PAA-001` | `when-supported` | `not-applicable` | `when-supported` |
| `SC-PAA-004` | `when-supported` | `not-applicable` | `when-supported` |
| `SC-PAA-005` | `when-supported` | `not-applicable` | `when-supported` |
| `SC-PAA-010` | `when-supported` | `not-applicable` | `when-supported` |
| `SC-PAA-013` | `when-supported` | `not-applicable` | `when-supported` |

The browser surface is not changed in this slice; the configured visible
journey is verified by issue 047.

## Implementation and verification

Implemented the shared `internal/analysis/config` boundary with strict v1/v2
decoding, nearest-file discovery, path/glob safety, manifest-aware option
validation, unavailable-analyzer diagnostics, deterministic fingerprints, and
layout-preserving v2 round trips. The focused configuration and layout tests
cover the acceptance rules, and the configured CLI fixture exercises the
loader through the public analysis path.

Verification passed:

- `go test ./internal/analysis/config ./internal/viewer/layout ./cmd/arch-view -count=1`
- `go test ./... -count=1`
- `go vet ./...`
- `go build ./...`
- `git diff --check`

## Scenario traceability

| Source rule | Scenario | Issue criterion | Verification evidence |
| --- | --- | --- | --- |
| PAA-FR-001/004 | `SC-PAA-001`/`SC-PAA-004` | v1 compatibility, nearest-file selection, and invalid-nearest failure | configuration loader and layout-preservation tests |
| PAA-FR-002/005 | `SC-PAA-005` | safe normalized assignments and manifest-aware validation | path, duplicate, option, and unavailable-analyzer tests |
| PAA-FR-007/012 | `SC-PAA-010`/`SC-PAA-013` | independent layout/analysis ownership and safe filter diagnostics | schema, glob, redaction, and round-trip tests |

## Handoff

Issue 045 consumed the validated configuration through the combined CLI,
HTTP, and project-backed planning paths.
