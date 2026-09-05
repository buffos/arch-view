# Advanced ELK Stage 3 evidence

Issue: 082, Shared ELK presentation ports.

Status: implemented and automatically verified on 2026-09-05. Human visual
approval is pending. The capability remains specified and issue 083 is blocked.

## Delivered behavior

- Layered ELK receives deterministic presentation-only `in` and `out` ports,
  fixed sides, labels, and edge endpoint references.
- Port IDs are derived from scene node identity without changing canonical
  models, OKF concepts, relationship identity, or semantic reading order.
- Architecture and OKF use the same feature handler, geometry snapshot, port
  presentation, route fallback, styles, settings form, and browser SVG path.
- Manual node movement shifts its ports and reroutes affected edges between the
  new attachment points. Port graphics are noninteractive and aria-hidden.
- Malformed port geometry reports `geometry_port_invalid` and falls back only
  the affected route to deterministic orthogonal geometry.
- Go static SVG remains deterministic orthogonal and reports requested browser
  features as not applied.

## Scenario evidence

| Scenario | Evidence |
|---|---|
| SC-AER-004 | Pinned ELK fixture proves deterministic IDs, sides, labels, bounds, and endpoint references; shared architecture and OKF render tests verify presentation. |
| SC-AER-007 | Malformed-port fixture proves explicit diagnostics and per-edge fallback. |
| SC-AER-009 | Static SVG provenance test and browser SVG serializer port-retention test. |
| SC-AER-010 | Ports are noninteractive, aria-hidden presentation objects; existing list/details and OKF navigation tests remain green. |
| SC-AER-011 | Catalog/handler consistency tests and both scene renderers consume the same port modules. |

## Automated verification

- `go test ./... -count=1`: passed.
- `go test -race ./...`: passed.
- `go vet ./...`: passed.
- `go build ./...`: passed.
- All 60 browser tests: passed.
- `node --check` for all browser JavaScript: passed.
- `git diff --check`: passed.
- Modified implementation-file audit: all files are below 600 lines.
- Strict OKF validation: passed.

## Implementer visual pass

- Architecture: verified labeled input/output ports, port-bound target arrows,
  unchanged semantic selection, and normal/full-canvas rendering.
- OKF: switched the session draft to Layered, enabled the same feature, and
  verified all visible concepts use the shared labeled-port geometry.
- The feature section stays expanded when ports are enabled, and Port
  Constraints appears only after its feature prerequisite is satisfied.
- Fit, zoom, pan, selection, manual movement, focus/Back, responsive behavior,
  and downloaded SVG remain part of the required human visual review.

## Human review

Pending explicit user approval. Nested containers must not begin before this
gate is approved.
