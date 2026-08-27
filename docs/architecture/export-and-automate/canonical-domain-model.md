# Export and automate canonical domain model

## ExportJob aggregate

Fields: `job_id`, `input_model_id`, `input_revision`, `format`, `output_path`, `options`, `status` (`requested`, `writing`, `written`, `partial`, `failed`, `cancelled`), `artifact_metadata`, `diagnostics`.

### ExportRequest

`input` (`model_path` or analysis/project source), `format` (`json`, `html`, `svg`), `output_path`, `view_path?`, `embed_source=false`, `deterministic=true`.

`embed_source=true` is rejected as unsupported in v1 rather than silently ignored.

### ArtifactMetadata

`format`, `schema_version?`, `model_id`, `model_revision?`, `content_hash`, `layout_provenance?`, `diagnostic_summary`, `bytes`.

## Policies and invariants

- A written artifact is atomic: write temporary sibling, flush/close, then replace target according to platform-safe policy.
- JSON includes the complete/partial status and diagnostics.
- HTML is self-contained, embeds the pinned ELK runtime/profile data, and makes
  no network requests by default.
- Go CLI SVG contains semantic labels/metadata and no executable scripts; the
  browser's current-canvas SVG download is a separate interactive artifact
  path.
- Output ordering and rendering use stable IDs, normalized paths, explicit algorithm/version, and no wall-clock timestamps unless an explicit future option exists.
- A partial artifact is valid; a fatal job does not claim an artifact was written.

## Domain events

`ExportRequested`, `ArtifactWritten`, `PartialArtifactWritten`, `ExportFailed`, `ExportCancelled`.

## Extension points

PNG/PDF renderers, source embedding/redaction, artifact manifests, remote CI stores, and additional visual formats.
