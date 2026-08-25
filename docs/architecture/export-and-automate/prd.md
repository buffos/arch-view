# Export and automate PRD

## Purpose

Produce durable, deterministic architecture artifacts for local scripts, documentation, and CI without coupling to source language or interactive session state.

## User stories

- US-EXP-001 — As a CI job, I can write versioned deterministic JSON from a validated architecture model.
- US-EXP-002 — As a maintainer, I can create self-contained HTML and accessible SVG reports from the same model/view contract.

## Scope

Export canonical versioned JSON, self-contained interactive HTML, and scalable SVG from one validated model/view input. Provide CLI options, deterministic behavior, status/exit semantics, and repeatable CI use. Do not parse source, run analyzers directly, or own browser session state.

## Rules

- JSON is the canonical durable interchange format and is distinct from the future analyzer NDJSON protocol.
- HTML/SVG are projections of the same model/view contract.
- Visual exports use the viewer's local-first reference visibility policy by default; canonical JSON retains every reference and explicit view options can aggregate or expand reference scopes.
- Stable IDs/order, normalized paths, explicit versions, and layout provenance are required.
- Source contents are not embedded by default; paths/locations are retained.
- Complete and partial artifacts may be written; fatal input/render/write errors are non-zero.
- No automatic architectural approval or policy judgment is performed.

## Functional requirements

| ID | Requirement |
|---|---|
| EX-FR-001 | Export a validated model as versioned JSON. |
| EX-FR-002 | Produce self-contained interactive HTML without network dependencies. |
| EX-FR-003 | Produce scalable deterministic SVG with accessible metadata. |
| EX-FR-004 | Preserve status, diagnostics, evidence references, cycles, layers, and provenance. |
| EX-FR-005 | Make repeated exports byte-stable for identical inputs/options. |
| EX-FR-006 | Expose clear CLI options, output behavior, and exit codes. |
| EX-FR-007 | Support reproducible local/CI artifact generation. |
| EX-FR-008 | Preserve reference scope, confidence, and import/evidence traceability while matching the viewer's selected visibility policy. |

## Non-goals

PNG/raster in v1, source embedding, cloud collaboration, automatic architecture gates, and exporter-specific source parsing.
