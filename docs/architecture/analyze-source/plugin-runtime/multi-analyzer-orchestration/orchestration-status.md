# Multi-analyzer project orchestration orchestration status

## State

- Planning state: `specified`.
- The exact-spec pipeline is complete. Issues 039–042 are implemented and
  verified; issue 043 is implemented and awaiting its required visual review.

## Evidence

- The shared analyzer contract, canonical result format, and external process
  lifecycle are implemented and now feed the bounded multi-analyzer scheduler.
- The host plans marker-driven roots, forwards the resolved source scope, runs
  isolated jobs, and aggregates namespaced results with stable provenance and
  status/error mappings.
- The project-backed CLI and local viewer expose the cached aggregate and
  individual scopes; assignment persistence remains owned by the separate
  project-analyzer-assignments capability.

## Artifact sync

- **Topology:** This child is linked under `Analyzer plugin runtime`.
- **Capability:** Discovery notes separate current single-analyzer behavior from
  the bounded marker-driven job plan, invocation-root source filters,
  namespaced merge, bounded concurrency, combined model, and partial-failure
  target.
- **Product and architecture:** The application synthesis records the
  implemented multi-job orchestration boundary and the separate assignment
  ownership.
- **Delivery:** Issues 039–042 are archived as verified delivery records. Issue
  043 remains the active `awaiting-human-review` record for the local viewer's
  mixed-language visual gate. The active registry, owning capability node, and
  plugin-runtime roll-ups reference the completed paths plus the pending 043
  path.

## Exact-spec inventory

- Requirements gap analysis
- Domain glossary
- Capability PRD
- Canonical domain model
- Canonical use-case model
- Canonical API/CLI contract
- Acceptance scenarios
- Architecture readiness review

All artifacts are linked from the capability node and agree on marker traversal,
nested ownership, invocation-root source filtering, job identity, worker limits,
cancellation, aggregate status, provenance, progress, cache identity, and
no-inferred-relationship behavior.

## Readiness decision

IMPLEMENTATION VERIFIED PENDING VISUAL REVIEW. The application synthesis gate
is current, issues 039–042 passed their acceptance and repository checks, and
issue 043 passed its automated checks. The user must complete issue 043's
declared mixed-language `visual-review` gate before the capability can move to
`implemented`.

## Artifact impact

- Capability truth: updated with the complete exact-spec set, including the
  source-scope policy.
- Product truth: combined/per-scope, partial-result, and configured source
  filtering behavior is synchronized.
- Architecture truth: source-scope policy, aggregate model, cache identity,
  and canonical-normalization dependency are current.
- Delivery truth: issue files, the active registry (max ID 043), the owning
  capability reference, and the parent roll-ups record verified issues 039–042
  plus pending issue 043. The capability remains `specified` until the visual
  gate and final issue synchronization are approved.

## Implementation evidence

- Deterministic discovery, nested ownership, selection precedence, source
  filtering, stable fingerprints, and job limits are covered by
  `internal/analysis/orchestration/orchestration_test.go`.
- Bounded execution, failure isolation, cancellation, lifecycle events, and
  host-side source filtering are covered by the orchestration tests and the
  existing process-adapter cancellation/timeout suite.
- Namespaced aggregation, status rules, source identity, CLI contracts, HTTP
  scope/projection routing, and no-reanalysis behavior are covered by focused
  Go tests and the repository-wide suite.
- The strict review pass added regressions for nested invocation roots and
  content-sensitive source identity, process source-scope forwarding,
  immutable plan/cache snapshots, unique retry run IDs, planner-error
  short-circuiting, aggregate/scope parity, opened-root HTTP containment, and
  transactional failed reanalysis.
- Viewer assets pass JavaScript syntax checks. No browser-test harness is
  present in the repository, so browser obligations are deferred under the
  root `when-supported` policy; issue 043's visual inspection remains open.
- `go test ./... -count=1`, `go test -race ./... -count=1`, `go vet ./...`,
  `go build ./...`, `go mod verify`, `staticcheck ./...`, JavaScript syntax and
  viewer-module tests, Python syntax parsing, strict OKF validation, and
  `git diff --check` pass on Windows amd64. The installed `golangci-lint`
  2.12.2 binary was built with Go 1.26.2 and cannot load the Go 1.27.0 standard
  library; it exits during package loading before repository lint analysis.
