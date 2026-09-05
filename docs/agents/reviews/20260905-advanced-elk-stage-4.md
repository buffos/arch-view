# Advanced ELK Stage 4 evidence

Issue: 083, Shared ELK nested containers.

Status: implemented and automatically verified on 2026-09-05. Explicit human
visual approval is still required before issue closeout or capability promotion.

## Delivered behavior

- The registered `compound` handler converts only visible supplied hierarchy
  into presentation-only nested Layered ELK containers; hidden intermediate
  concepts are not invented.
- Semantic parent nodes remain ordinary graph nodes inside their presentation
  containers, so routes terminate on real node boundaries rather than frames.
- The top-level concept remains on the canvas instead of creating a redundant
  container around the whole graph.
- ELK-relative child, edge, label, port, and junction coordinates are validated
  and normalized into the shared absolute `arch-view.geometry/v1` snapshot.
- Architecture and OKF render the same non-semantic container frames behind
  scene content. The semantic parent remains the selectable header.
- Moving a container moves every visible descendant and reroutes affected edges
  through the existing shared geometry path. Moving a child beyond its original
  frame expands its ancestor presentation frames.
- Invalid hierarchy or bounds produce an explicit diagnostic and deterministic
  flat-layout fallback without corrupting the last renderable scene.
- Existing defaults remain unchanged: compound is opt-in, OKF uses Mr. Tree by
  default, and selected-only semantic links remain arrowless and excluded from
  ELK placement.

## Scenario evidence

| Scenario | Evidence |
|---|---|
| SC-AER-005 | Pinned ELK nested-hierarchy and hidden-segment fixtures; shared architecture/OKF frame tests; live OKF scene produced four containers. |
| SC-AER-007 | Malformed hierarchy and bounds fixtures prove diagnostic flat fallback. |
| SC-AER-009 | Static provenance test plus browser SVG serialization retaining container frames. |
| SC-AER-010 | Frames are aria-hidden presentation objects; descendant movement and existing navigation/accessibility suites pass. |
| SC-AER-011 | Compound composes with labels, ports, and splines through ordered shared handlers; all browser tests pass. |

## Automated verification

- `go test ./... -count=1`: passed.
- `go test -race ./...`: passed.
- `go vet ./...`: passed.
- `go build ./...`: passed.
- All 77 browser tests: passed.
- `node --check` for all browser JavaScript: passed.
- `git diff --check`: passed.
- Strict OKF validation: passed with 20 concepts and no issues.
- Modified/new implementation-file audit: all files are below 600 lines.

## Implementer visual pass

- In a live OKF session, switched the session draft from Mr. Tree to Layered,
  enabled Nested containers, and observed all 17 nodes with three bounded
  capability containers.
- Full canvas and Fit preserved the complete scene; the browser reported no
  runtime errors.
- The architecture scene projection remains intentionally progressive and
  usually has no simultaneously visible parent/child pairs, so enabling the
  feature there normally produces no visual containers.

## Human review

Pending explicit user approval. Review both viewers and the downloaded SVG,
including Fit, zoom, pan, container movement, selection, and focus/Back.
