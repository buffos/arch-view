# Export and automate domain glossary

| Term | Definition | Distinction |
|---|---|---|
| Export request | Value object describing input model, format, output, and deterministic options. | Not an analyzer request. |
| Artifact | Produced JSON, HTML, or SVG file with status/metadata. | Not an interactive session. |
| Canonical JSON | Versioned durable model interchange representation. | Separate from plugin NDJSON frames. |
| Visual artifact | HTML or SVG projection of model/view semantics. | Raster is deferred. |
| Deterministic export | Same bytes for same model/options/environment contract. | No implicit clock or random seed. |
| Partial artifact | Valid artifact containing partial-analysis diagnostics. | Not a successful complete analysis. |
| Fatal export error | Input, render, or write failure preventing a valid artifact. | Produces non-zero exit. |
| Source reference | Path/location retained without source contents by default. | Embedding is a future explicit policy. |
| Layout provenance | Algorithm/version/seed metadata explaining visual placement. | Not semantic model data. |
