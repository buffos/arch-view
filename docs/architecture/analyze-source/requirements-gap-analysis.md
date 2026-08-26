# Analyze source code requirements gap analysis

## Summary

The capability is specified. The exact observation contract, runtime contract, and language-specific boundaries are documented and readiness-reviewed; remaining items are implementation risks rather than unresolved product decisions.

## Resolved gaps

| Area | Resolution |
|---|---|
| Run scope | One language and project per run. |
| Graph granularity | Package or module nodes, with files as evidence. |
| First relationship | Static dependency relationships. |
| Default graph scope | Project-local modules. |
| Default exclusions | Tests, generated code, vendor, caches, build output, and directories named `external`. |
| Failure behavior | Partial model plus diagnostics. |
| Evidence | Source file and parser-provided line and column when available. |
| Analyzer selection | Auto-detection with an explicit language override. |
| Project boundaries | Owned by each analyzer. |
| First plugin deployment | In-process Go interface. |
| Future external deployment | Versioned NDJSON with JSON Schema. |
| Result stability | Host-side validation, normalization, sorting, and deduplication. |
| Safety | No target application or arbitrary project-code execution. |

## Specification closure and residual risks

### Canonical analysis result

Defined in [the canonical domain model](canonical-domain-model.md) and [contract](canonical-api-cli-contract.md), including fields, invariants, statuses, evidence, confidence, and diagnostics.

### Go analysis behavior

Defined in [the Go PRD and contract](go-analysis/prd.md), including module/workspace selection, build view, local import resolution, exclusions, and diagnostics.

### Plugin manifest and host contract

Defined in [the plugin runtime contract](plugin-runtime/canonical-api-cli-contract.md), including manifest fields, capabilities, options, versioning, detection, and failures.

### Scope configuration

Defined as CLI > project configuration > analyzer defaults, with adapter-specific options and exclusions specified in the parent contract and child contracts.

### Evidence and diagnostics

Defined as repository-relative paths, one-based locations, `info|warning|error` severities, explicit recoverability/confidence, and evidence IDs retained through aggregation.

### External protocol

The in-process contract is specified first; the future process protocol now has v1 frame semantics in the plugin contract and remains an interoperability/schema publication risk, not a blocker for the built-in runtime.

## Readiness

The capability has passed the architecture specification pipeline and readiness review. It may enter implementation/issue slicing after the application synthesis gate is verified.
