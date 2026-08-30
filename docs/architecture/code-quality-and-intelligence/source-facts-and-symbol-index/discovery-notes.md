# Source facts and symbol index discovery notes

## Purpose

Make the source structure behind every architecture module inspectable without
requiring consumers to load full source text. The result is the stable fact
boundary that deterministic quality rules, the viewer, structural search, and
future MCP consumers can share.

## Observed in code

- `internal/analysis/types.go` exposes `AnalysisResult` with modules,
  relationships, references, source references, diagnostics, and a summary.
  `ModuleObservation` can retain source-reference IDs, but there is no common
  file, symbol, documentation, or metric index.
- `internal/model/types.go` is the canonical language-neutral model. It has no
  source-index sibling yet, so adding source facts to module metadata would
  couple a new concern to the existing architecture graph.
- The shared syntax boundary is `internal/analysis/syntax.Provider`, whose
  parsed nodes expose types, named/error state, text, and byte/point ranges.
  Tree-sitter point columns are byte offsets and must not be exposed as an
  undocumented coordinate convention.
- The Go, Python, TypeScript, Rust, and Clojure analyzers already enumerate
  source files or preserve selected source positions, but their file and
  declaration facts are language-specific and are not emitted through one
  source-index contract. Rust recognizes documentation attributes without
  currently turning them into documentation records.
- Multi-analyzer orchestration already retains per-scope results and source
  scope provenance. It does not merge independent analyzer observations solely
  by path and therefore provides the correct ownership boundary for a
  scope-first index.

## User-confirmed target behavior

- A module can show the names of its contributing files without displaying
  their code by default.
- Modules and named declarations can expose language-supported documentation,
  functions, methods, classes, structs, interfaces, traits, enums, aliases,
  namespaces, constants, variables, and macros.
- Files expose deterministic line and byte counts and a content hash.
- The same facts are useful to the graph viewer, deterministic quality checks,
  compact structural search, and an eventual read-only MCP surface.
- The schema must be open for new languages, metrics, calls, implementations,
  and other relations without repeated central enum edits or `FileRecord`
  refactors.

## Specified decisions

- `SourceIndex` is an optional top-level sibling of architecture modules and
  relationships in both analysis and canonical-model payloads. Existing
  consumers may ignore it; an analyzer or host may omit it when the requested
  extractor capability is unavailable.
- A scope owns first-class `FileRecord`, `SymbolRecord`,
  `DocumentationRecord`, `SymbolOccurrence`, `CodeRelation`, and `MetricFact`
  records. `module -> file -> symbol` containment is a relation, not a
  singular module field or duplicated source of truth.
- File records contain repository-relative POSIX paths, namespaced roles,
  intrinsic size facts, content hashes, analysis status, provenance, and typed
  extension blocks. Full source content is not stored in the index.
- Symbol records use a stable coarse category and an open language-specific
  kind. They represent named declarations, not every local variable or AST
  expression. Calls, references, inheritance, and implementations are
  separate occurrences/relations.
- Documentation records retain every candidate and let the language extractor
  choose a deterministic primary candidate. The core does not apply a
  language-agnostic nearest-comment heuristic.
- Canonical spans use raw UTF-8 byte offsets, one-based line and column values,
  end-exclusive ranges, and the file content hash. Existing
  `SourceReference` values remain compatible through an adapter.
- Registered extractors own language semantics, declaration recognition,
  visibility, documentation attachment, and uncertainty. The core owns
  orchestration, validation, deduplication, opaque ID assignment, coverage,
  and snapshot assembly.
- Provenance distinguishes observed, absent, unknown, unsupported, and
  partial. Unsupported or unknown facts are never fabricated as absence.
- IDs are opaque and snapshot-local. A deterministic producer may expose an
  optional stable key and identity basis for reconciliation, but consumers do
  not parse IDs and source ranges are evidence, not identity.
- Extension blocks are namespaced and typed by their owner. Unknown extension
  blocks are ignored by consumers that do not advertise the capability.
- A combined multi-analyzer index retains authoritative per-scope snapshots;
  any combined projection is explicitly derived and never becomes a new
  source of truth.

## Boundary

This capability owns source facts and the deterministic index contract. It does
not own quality thresholds or findings, cyclomatic-complexity policy, SOLID or
other architectural signals, folder watching, snapshot freshness, MCP
transport, search ranking, or source mutation. Those capabilities consume this
contract.

## Implementation and verification focus

Implementation should first establish the core record types, validation,
snapshot identity, extractor registry, and a Go extraction slice. Subsequent
language extractors can register capabilities independently. Verification must
cover line-count edge cases, declaration/documentation evidence, explicit
coverage states, multi-scope isolation, deterministic serialization, malformed
extractor batches, backward-compatible omission, and token-safe projections.
