# Multi-analyzer project orchestration orchestration status

## State

- Planning state: `implemented`.
- The exact-spec pipeline and approved implementation sequence are complete.
  Issues 039–043 are implemented, verified, archived, and include the approved
  mixed-language viewer review.

## Evidence

- The shared analyzer contract, canonical result format, and external process
  lifecycle are implemented and now feed the bounded multi-analyzer scheduler.
- The host plans marker-driven roots, forwards the resolved source scope, runs
  isolated jobs, and aggregates namespaced results with stable provenance and
  status/error mappings.
- The project-backed CLI and local viewer expose the cached aggregate and
  individual scopes; the project-analyzer-assignments capability now supplies
  the validated configured policy, assignment precedence, and session-cache
  metadata at that existing seam.

## Artifact sync

- **Topology:** This child is linked under `Analyzer plugin runtime`.
- **Capability:** Discovery notes separate current single-analyzer behavior from
  the bounded marker-driven job plan, invocation-root source filters,
  namespaced merge, bounded concurrency, combined model, and partial-failure
  target.
- **Product and architecture:** The application synthesis records the
  implemented multi-job orchestration boundary and the separate assignment
  ownership.
- **Delivery:** Issues 039–043 and assignment/view consumer issues 044–047 are
  archived as verified delivery records, including the approved configured
  viewer review. The owning capability node and plugin-runtime roll-ups
  reference the completed paths.

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

IMPLEMENTATION VERIFIED FOR THE ORCHESTRATION BOUNDARY. The application
synthesis gate is current, issues 039–043 passed their acceptance and
repository checks, and the user approved issue 043's declared mixed-language
`visual-review` gate. Configured assignment and source-scope integration is
implemented through issues 044–047, including the approved configured-viewer
gate owned by the assignment/view capability.

## Artifact impact

- Capability truth: updated with the complete exact-spec set, including the
  source-scope policy.
- Product truth: combined/per-scope, partial-result, and configured source
  filtering behavior is synchronized.
- Architecture truth: source-scope policy, aggregate model, cache identity,
  and canonical-normalization dependency are current.
- Delivery truth: issue files, the registry, the owning capability reference,
  and the parent roll-ups record completed issues 039–047, including the
  approved configured-viewer review. This orchestration capability and the
  assignment/view child are implemented.

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
  present in the repository, so browser obligations remain deferred under the
  root `when-supported` policy; the required issue 043 visual inspection was
  completed and approved by the user.
- `go test ./... -count=1`, `go test -race ./... -count=1`, `go vet ./...`,
  `go build ./...`, `go mod verify`, `staticcheck ./...`, JavaScript syntax and
  viewer-module tests, Python syntax parsing, strict OKF validation, and
  `git diff --check` pass on Windows amd64. The installed `golangci-lint`
  2.12.2 binary was built with Go 1.26.2 and cannot load the Go 1.27.0 standard
  library; it exits during package loading before repository lint analysis.
