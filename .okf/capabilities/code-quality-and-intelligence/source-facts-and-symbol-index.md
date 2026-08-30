---
type: capability
title: Source facts and symbol index
description: Expose files, symbols, documentation, and source-level metrics in a stable searchable index.
tags: [code-intelligence, source-index, symbols, documentation]
timestamp: 2026-08-30T08:15:05Z
state: implemented
state_changed: 2026-08-30T08:15:05Z
project: /project.md
parent: /capabilities/code-quality-and-intelligence.md
artifact_root: docs/architecture/code-quality-and-intelligence/source-facts-and-symbol-index
discovery_notes: docs/architecture/code-quality-and-intelligence/source-facts-and-symbol-index/discovery-notes.md
gap_analysis: docs/architecture/code-quality-and-intelligence/source-facts-and-symbol-index/requirements-gap-analysis.md
orchestration_status: docs/architecture/code-quality-and-intelligence/source-facts-and-symbol-index/orchestration-status.md
prd: docs/architecture/code-quality-and-intelligence/source-facts-and-symbol-index/prd.md
glossary: docs/architecture/code-quality-and-intelligence/source-facts-and-symbol-index/domain-glossary.md
domain_model: docs/architecture/code-quality-and-intelligence/source-facts-and-symbol-index/canonical-domain-model.md
use_cases: docs/architecture/code-quality-and-intelligence/source-facts-and-symbol-index/canonical-use-cases.md
contract: docs/architecture/code-quality-and-intelligence/source-facts-and-symbol-index/canonical-api-cli-contract.md
scenarios: docs/architecture/code-quality-and-intelligence/source-facts-and-symbol-index/acceptance-scenarios.md
readiness_review: docs/architecture/code-quality-and-intelligence/source-facts-and-symbol-index/readiness-review.md
issues:
  - docs/agents/issues/done/20260829-048-versioned-source-index-and-file-facts.md
  - docs/agents/issues/done/20260829-049-registered-go-source-fact-extractor.md
  - docs/agents/issues/done/20260829-050-scope-safe-source-index-aggregation.md
  - docs/agents/issues/done/20260829-051-structural-source-fact-queries.md
  - docs/agents/issues/done/20260829-052-module-source-facts-viewer.md
---

# Intent

Make the source structure behind each architecture module directly inspectable and searchable.

# Scope

This capability covers first-class files, physical line counts, module
descriptions, documentation evidence, declarations such as
functions/classes/interfaces, extractor-reported callable body spans and
versioned source metrics, containment, visibility, and the foundation for
later call or implementation relationships. It does not own quality policy or
MCP transport.

# Relationships

- Parent: [Code quality and code intelligence](../code-quality-and-intelligence.md)
- Input: [Analyze source code](../analyze-source.md)
- Consumer: [Deterministic quality checks](deterministic-quality-checks.md)
- Consumer: [Live analysis and MCP](live-analysis-and-mcp.md)
- Viewer consumer: [Explore and inspect architecture](../explore-architecture.md)

# Current and target truth

Observed code facts:

- Module observations already retain source-reference IDs; supported analyzers read source files through Tree-sitter-backed providers and some retain file paths and file counts in metadata.
- The optional source-index attachment now exposes deterministic files,
  declarations, documentation, spans, provenance, coverage, and bounded query
  projections through the analysis, canonical, aggregate, and local viewer
  paths. The final rendered viewer approval is recorded for issue 052.
- The local viewer consumes the attachment progressively: graph startup uses a
  compact model response, node/group details show only human-scale summary
  fields, and a same-tab inspection route loads bounded Files, Symbols,
  documentation, and evidence sections on demand. Technical identifiers remain
  secondary, and embedded exports answer the same bounded queries locally.

User-confirmed target decisions:

- `SourceIndex` is an optional sibling of modules, relationships, source references, diagnostics, and derived data in both `AnalysisResult` and the canonical model.
- Files and symbols are first-class scoped observations. Containment is represented by extensible relations (`module -> file -> symbol`) rather than singular module fields or duplicated container arrays.
- `FileRecord` owns repository-relative path, language, namespaced roles, deterministic size facts, content hash, analysis status, provenance, and extension blocks. Full source content is not part of the index.
- `SymbolRecord` initially covers named declarations and keeps a stable coarse category plus an open language-specific kind. Calls, inheritance, implementations, and references are separate relations/occurrences, not symbol arrays.
- Callable symbols may carry an extractor-reported, hash-linked `body_span`;
  the Go extractor also advertises `source:callable.metrics` with versioned
  body-line, cyclomatic-complexity, and maximum-nesting metrics. These are
  provider facts consumed by deterministic quality rules, not quality findings.
- Documentation is a separate provenance-backed record with all candidates retained, language-owned precedence, and explicit `present`, `absent`, `unknown`, `unsupported`, and `partial` states.
- Canonical spans use UTF-8 byte offsets, one-based line/column values, end-exclusive ranges, and a file content hash. Existing `SourceReference` remains backward-compatible and can be adapted.
- Registered language extractors own syntax declarations, visibility, documentation, semantic resolution, and uncertainty. The core owns orchestration, validation, deduplication, identity assignment, and snapshot assembly.
- Opaque snapshot-local IDs, open vocabularies, capability negotiation, provenance, and typed extension blocks preserve additive evolution without forcing central schema edits.
- Coverage is explicit: unsupported or unknown facts are never fabricated as absence, and a combined multi-scope index is a deterministic projection rather than a new source of truth.
- Bounded module/file query filters resolve membership only from explicit
  containment or declaration relations; matching paths or names never infer
  membership. File-only provenance is distinct from line-level locations.

Bounded boundary:

- This capability specifies the source-facts contract and its deterministic extraction boundary: files, intrinsic file size, named declarations, documentation evidence, extractor-reported callable spans/metrics, containment, provenance, optional occurrences/relations, and snapshot metadata.
- It does not define quality policy thresholds, cyclomatic-complexity algorithms, architectural violation rules, live folder watching, MCP transport, or LLM remediation workflows. Those are downstream capabilities consuming this contract.
- The v1 contract may carry empty or omitted capability-dependent collections. It must still provide a file record for every eligible file when the producing analyzer can enumerate the source scope.

# Exact specification set

The exact-spec files are declared in the frontmatter above: discovery notes,
requirements gap analysis, domain glossary, PRD, canonical domain model,
canonical use cases, canonical API/CLI contract, acceptance scenarios,
architecture readiness review, and orchestration status.

The exact-spec set is readiness-reviewed. The approved implementation batch has
delivered the source-index contract, Go extractor, scope-safe aggregation,
bounded queries, and viewer projection; issue 052's final visual-review gate
is approved. Quality policy and live/MCP behavior remain outside this child.

# Delivery

The approved dependency-ordered delivery batch is issues 048–052. All five
issues are verified, archived, and approved, including issue 052's required
final visual inspection. The node is `implemented`; the parent roll-up remains
independently governed by its other children.
