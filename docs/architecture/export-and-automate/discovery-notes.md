# Export and automate discovery notes

## Purpose

Make analysis usable from scripts, reports, documentation, and repeatable local/CI workflows without coupling outputs to a source language or interactive session.

## Reference observations

The read-only reference supports headless analysis and writes an EDN architecture representation. Arch View will retain the useful headless workflow while using versioned JSON as the language-neutral interchange format and adding deterministic visual artifacts.

## Target boundary

The capability owns CLI export options, versioned machine-readable output, deterministic visual artifacts, headless execution status, and repeatable automation behavior. It consumes canonical model and renderer-neutral view inputs. It does not parse source syntax, implement analyzer plugins, or own interactive viewer session state.

## Confirmed export and automation decisions

1. Versioned JSON is the canonical interchange/export format. It contains project/analyzer metadata, modules, hierarchy, typed relationships, evidence references, diagnostics, cycles, derived layers, and analysis status.
2. JSON output is deterministic: stable IDs and ordering, normalized paths, explicit schema/model versions, and no implicit wall-clock timestamps. Layout metadata records the algorithm/version or seed when relevant.
3. The first visual artifacts are self-contained interactive HTML and scalable SVG. Both are generated from the same neutral model/view contract; they do not re-parse source code. PNG/raster output remains a later adapter.
4. Source contents are not embedded by default. Exports retain repository-relative paths and locations; an explicit future option may embed source with clear size/privacy implications.
5. Headless runs expose complete, partial-with-diagnostics, and fatal states. Warnings and recoverable unresolved dependencies remain visible in artifacts; configuration/readability/fatal errors produce non-zero exit codes.
6. CLI options select project, language, configuration, output format, output path, and deterministic rendering settings. The same run can produce machine and visual artifacts from one model.
7. CI integration begins with stable artifact generation and documented exit behavior. Automatic architectural approval, cloud collaboration, and policy judgment remain outside this capability.
8. The export format is distinct from the future external analyzer process protocol: exported JSON is a durable artifact, while NDJSON is a later streaming plugin boundary.

## Actors and inputs

- A developer invokes headless analysis locally or a CI job invokes it reproducibly.
- The exporter consumes the validated model, derived graph projections, and optional view configuration.
- Downstream tools consume JSON, HTML, or SVG without knowing the analyzed language.

## Open questions for exact specification

- Exact JSON schema, version compatibility, migration policy, and schema publication location.
- CLI command/flag names, output naming, overwrite behavior, and atomic-write policy.
- HTML asset embedding/CSP policy and the exact subset of interactive features available offline.
- SVG accessibility metadata, layout determinism, font handling, and large-graph limits.
- Exit-code matrix, diagnostics serialization, and CI logging conventions.
- Optional source embedding, redaction, and artifact-size controls.
