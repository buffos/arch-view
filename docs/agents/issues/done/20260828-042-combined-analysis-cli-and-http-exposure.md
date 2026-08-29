# 042 — Expose combined analysis through CLI and HTTP

## Issue Metadata

- Issue number: `042`
- Owning capability node: `/.okf/capabilities/analyze-source/plugin-runtime/multi-analyzer-orchestration.md`
- Artifact root: `docs/architecture/analyze-source/plugin-runtime/multi-analyzer-orchestration/`
- Issue file: `docs/agents/issues/done/20260828-042-combined-analysis-cli-and-http-exposure.md`
- Category: `feature`
- Execution type: `AFK`
- Review gate: `none`
- Suggested state: `done`

## Parent Artifacts

- `docs/prd.md`
- `docs/architecture/application-architecture-summary.md`
- `docs/architecture/analyze-source/plugin-runtime/multi-analyzer-orchestration/prd.md`
- `docs/architecture/analyze-source/plugin-runtime/multi-analyzer-orchestration/canonical-domain-model.md`
- `docs/architecture/analyze-source/plugin-runtime/multi-analyzer-orchestration/canonical-use-cases.md`
- `docs/architecture/analyze-source/plugin-runtime/multi-analyzer-orchestration/canonical-api-cli-contract.md`
- `docs/architecture/analyze-source/plugin-runtime/multi-analyzer-orchestration/acceptance-scenarios.md`
- `docs/architecture/analyze-source/plugin-runtime/multi-analyzer-orchestration/readiness-review.md`
- `docs/architecture/analyze-source/plugin-runtime/implementation-slice.md`

## What to build

Expose the aggregate run and scope contracts to command-line and HTTP/CI
consumers. Make `arch-view analyze --project <repository>` plan the combined
run by default while preserving the existing explicit `--analyzer`/`--language`
single-scope behavior. Serialize the deterministic aggregate result, scope
summaries, diagnostics, job plan/progress, and selected scope projection using
the approved schemas and map complete/partial/failed/cancelled outcomes to the
documented HTTP and CLI status rules. Support `--scope` and the HTTP projection
query as cached selection operations that never invoke an analyzer again. This
slice owns transport/CLI exposure; the `open` viewer wiring and rendered scope
selector are completed by issue 043. Carry the validated invocation-root
source-scope policy and its effective-source-set fingerprint through the plan
and response; configuration parsing/resolution remains owned by the assignment
boundary.

## Acceptance criteria

- [x] The default project-backed `analyze` command invokes the combined
  orchestration path when no single-analyzer override is supplied, emits the
  exact `arch-view.aggregate/v1` response, and returns success for usable
  complete or partial results.
- [x] Explicit `--analyzer` and `--language` selections remain compatible with
  the existing single-scope workflow and cannot accidentally broaden into an
  aggregate run; `--scope` selects a cached scope/projection without starting
  another analyzer job.
- [x] The HTTP surface implements the approved analysis, run, scope-list, and
  projection mappings: `POST /v1/analyses`, `GET /v1/analyses/{run_id}`,
  `GET /v1/analyses/{run_id}/scopes`, and
  `GET /v1/analyses/{run_id}/projection?scope=all|<scope_id>`; lifecycle events
  may be streamed through `GET /v1/analyses/{run_id}/events`.
- [x] Complete and partial usable runs use HTTP 200 and CLI exit 0; invalid
  requests, selection conflicts, unavailable analyzers, aggregate failures,
  and caller cancellation use the documented 400/409/422/500/130 semantics
  without leaking unstable internal errors.
- [x] Scope lists and projections are sorted and deterministic, preserve
  per-scope diagnostics/provenance, return `analysis_scope_not_found` for an
  unknown scope, and never trigger re-analysis during selection.
- [x] Job plan/lifecycle information is available to CLI/HTTP consumers with
  stable run/job/scope IDs and completed/total counts; analyzer-internal
  percentage is not required.
- [x] The CLI and HTTP paths preserve the resolved source-scope policy and
  effective-source-set identity, expose `analysis_scope_filter_invalid` with
  the documented invalid-input status, and keep command-line exclusions
  additive so they cannot re-include configured exclusions.
- [x] Existing single-scope analysis JSON, export formats, explicit runtime
  selection, and stable package/process errors remain compatible.
- [x] Command and transport tests cover mixed repositories, partial and
  no-usable-result outcomes, explicit selection compatibility, scope
  projection without re-analysis, deterministic serialization, status/exit
  mappings, and stable error payloads.

## Artifact sync required

- Application PRD: `required` — CLI/HTTP behavior now exposes the configured
  source-scope contract; the synchronized workflow is in `docs/prd.md`.
- Application architecture summary: `required` — transport must preserve the
  plugin-manager source-scope policy and effective-source-set identity.
- Owning capability node/artifacts: `required: /.okf/capabilities/analyze-source/plugin-runtime/multi-analyzer-orchestration.md` and `docs/architecture/analyze-source/plugin-runtime/multi-analyzer-orchestration/orchestration-status.md`; update the parent plugin-runtime delivery record with the public contract evidence.
- Issue registry: `required`; node `issues:` reference: `required when .okf exists`.
- Reason/impact decision: delivery truth changes and the public transport must
  preserve source-scope configuration/error semantics; product topology and
  canonical model ownership remain unchanged.

## Human review gate

None. This slice changes CLI, JSON, and HTTP behavior but no rendered UI/UX;
the local viewer exposure is intentionally owned by issue 043.

## Blocked by

Blocked by `docs/agents/issues/done/20260828-041-namespaced-aggregate-model-and-status.md`.

## Artifact anchors

- `MAO-FR-008` through `MAO-FR-010`.
- `SourceScopePolicy`, `analysis_scope_filter_invalid`, and effective-source-set
  identity in the CLI/HTTP mapping.
- `SelectAnalysisScope`, `AnalysisRun`, scope summaries, and cached projection
  rules.
- The aggregate response, job-plan/progress stream, HTTP status mappings, and
  CLI mappings in the canonical contract.

## Acceptance scenarios addressed

- `SC-MAO-001` — Discover mixed-language roots (CLI/transport exposure)
- `SC-MAO-005` — Preserve independent success after failure (aggregate output)
- `SC-MAO-009` — Expose combined and individual scopes (transport contract)
- `SC-MAO-010` — Report no-usable-result failure (CLI/HTTP status mapping)

## User stories addressed

The capability PRD has no numbered user-story section. This slice supports
application journeys 9 and 4 through repeatable CLI/HTTP/CI consumption; the
rendered viewer journey remains in issue 043.

## Verification obligations

- Policy source: `/.okf/project.md`

| Scenario | Backend boundary | Frontend integration | End-to-end journey |
| --- | --- | --- | --- |
| `SC-MAO-001` | `implemented` | `not-applicable` | `implemented` |
| `SC-MAO-005` | `implemented` | `not-applicable` | `implemented` |
| `SC-MAO-009` | `implemented` | `not-applicable` | `implemented` |
| `SC-MAO-010` | `implemented` | `not-applicable` | `implemented` |

## Scenario traceability

| Source rule | Scenario | Issue criterion | Verification evidence |
| --- | --- | --- | --- |
| MAO-FR-001/008/009 | SC-MAO-001/005/009 | combined default, aggregate transport, and cached scope contract | `cmd/arch-view` contract tests and `internal/viewer/aggregate_http_test.go` |
| MAO-FR-010 | SC-MAO-010 | stable failure/status mappings and no-usable-result response | aggregate status tests, CLI error mapping tests, and HTTP status assertions |

## Implementation and verification

Implemented combined CLI analysis, explicit single-scope compatibility,
aggregate HTTP run/scope/projection/events endpoints, stable status/error
mapping, and source-scope identity propagation. Product and architecture truth
required no further change because the exposed contracts match the refreshed
application synthesis. Verified with `go test ./cmd/arch-view`, focused viewer
and orchestration tests, `go test ./... -count=1`,
`go test -race ./... -count=1`, `go vet ./...`, `go build ./...`,
`staticcheck ./...`, JavaScript syntax checks, strict OKF validation, and
`git diff --check`.

## Handoff

After this issue passes, issue 043 may connect the aggregate run to
project-backed `open`, expose cached scope projections, and add the required
viewer selector and visual-review gate.
