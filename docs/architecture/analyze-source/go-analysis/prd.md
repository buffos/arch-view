# Go analysis PRD

## Purpose

Provide the first complete static analyzer for Go package architecture.

## Scope

Analyze one selected Go module, discover packages and source files, resolve static imports, classify non-local targets, and return evidence/diagnostics through the common analyzer contract. Do not execute Go code, build scripts, tests, or applications.

## Boundary rules

- `go.mod` is the primary boundary.
- A `go.work` root with multiple modules requires explicit `--module`; one run never silently composes multiple modules.
- Package is the graph node; files are evidence.
- Default exclusions: `_test.go`, `vendor`, generated marker files, build/cache output, `.git`, and directories named `external`.
- Default build view uses normal source files and no optional build tags; selected tags are explicit options.

## User stories

- US-GO-001 — As a developer, I can analyze a Go module and see its project-local packages and static imports.
- US-GO-002 — As a CI job, I can produce deterministic Go analysis and architecture-model output without executing the target repository.
- US-GO-003 — As a maintainer, I can see unresolved, external, and build-conditional dependencies as evidence or diagnostics rather than fabricated local modules.

## Functional requirements

| ID | Requirement |
|---|---|
| GO-FR-001 | Detect a Go module from `go.mod` and report ambiguous workspace selection. |
| GO-FR-002 | Discover packages from eligible `.go` files. |
| GO-FR-003 | Extract static imports with file/line/column evidence. |
| GO-FR-004 | Resolve imports inside the selected module to package IDs. |
| GO-FR-005 | Classify standard-library, third-party, missing, cgo, and conditional imports as references/diagnostics. |
| GO-FR-006 | Honor explicit tags, include-tests, and exclusion options. |
| GO-FR-007 | Produce deterministic output without invoking the target program. |

## Non-goals

Call graphs, runtime dispatch, full type graphs, proc execution, build-script evaluation, and automatic multi-module workspace composition.
