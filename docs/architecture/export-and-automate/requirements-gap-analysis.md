# Export and automate requirements gap analysis

## Resolved

| Area | Resolution |
|---|---|
| Canonical format | Versioned JSON, separate from the future NDJSON plugin protocol. |
| Machine contents | Neutral model, evidence references, diagnostics, cycles, layers, and analysis status. |
| Visual artifacts | Self-contained interactive HTML and scalable SVG first; raster later. |
| Determinism | Stable ordering/IDs, normalized paths, explicit versions, and recorded layout provenance. |
| Source privacy | Paths/locations by default; source embedding is explicit and future. |
| Partial results | Exported with visible diagnostics; fatal configuration/analysis errors are non-zero. |
| Automation | Reproducible local/CI artifact generation, not automatic architecture judgment. |
| Ownership | Exporter consumes model/view contracts and does not parse source or own viewer session state. |

## Specification closure and residual risks

JSON v1, CLI/HTTP syntax, atomic writes, HTML/SVG packaging, exit codes, and v1 source-embedding rejection are defined in the exact-spec artifacts. Schema migration tooling and cross-environment rendering fixtures remain implementation risks.

## Readiness

The capability has passed the architecture specification pipeline and readiness review. It may enter implementation/issue slicing after the application synthesis gate is verified; deterministic rendering and CI fixtures remain required verification work.
