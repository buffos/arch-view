# Source facts and symbol index PRD

## Purpose

Give Arch View a compact, deterministic source-intelligence layer that makes
the files, named declarations, documentation evidence, and intrinsic source
metrics behind an architecture view directly inspectable.

## Actors

- **Developer:** opens a module and needs file names, declarations, documentation,
  and source size without loading every file.
- **Analyzer author:** registers language-specific extraction behavior without
  modifying the common model for every new declaration kind.
- **Quality engine:** consumes files, symbols, documentation, relations, and
  versioned metric facts to evaluate later deterministic rules.
- **Viewer/CLI client:** requests compact, evidence-backed projections and
  preserves source navigation through the existing inspection boundary.
- **Future MCP client:** searches structure and documentation with bounded
  output and asks for source context only when explicitly requested.

## Goals

1. Expose every eligible source file by normalized repository-relative name.
2. Expose deterministic raw byte count, physical line count, and content hash.
3. Expose named declarations with a stable coarse category, open language kind,
   visibility, location, and optional documentation references.
4. Expose module/file/symbol containment and declarations as evidence-backed
   relations without changing architecture relationship meaning.
5. Preserve documentation candidates and deterministic language-owned primary
   selection, including explicit absence and extraction uncertainty.
6. Provide a scope-first, multi-analyzer-safe attachment beside existing
   `AnalysisResult` and canonical model collections.
7. Allow future calls, implementations, type relations, and quality metrics
   through registered capabilities and typed extensions.
8. Make equal inputs produce equal canonical facts, IDs, ordering, and digest.
9. Support compact structural search and LLM context selection without
   embedding full source code in every model response.

## Non-goals

- Quality thresholds, severity policy, findings, baselines, or suppressions.
- Cyclomatic-complexity calculation, SOLID claims, or subjective comment-quality
  judgments.
- A complete call graph, runtime tracing, compiler-level type resolution, or
  guaranteed dynamic-language resolution.
- A general-purpose full-text or semantic search ranking engine.
- Folder watching, freshness/revision lifecycle, MCP transport, or source edits.
- Replacing existing `SourceReference`, module, relationship, diagnostic, or
  source-inspection contracts.
- Persisting complete source content in the index.

## V1 product behavior

The producer enumerates the already-resolved eligible source scope and emits a
file record for each enumerable file. It may attach declarations and
documentation only for advertised extractor capabilities. A parse failure does
not erase the file record; its status and provenance explain the limitation.

The index contains authoritative per-scope snapshots. A combined analysis may
also expose a deterministic projection, but each fact retains scope and
producer provenance and no relationship is inferred merely because paths or
names match.

The human viewer is a progressive-disclosure consumer of this attachment. The
graph card shows a compact node/group summary and links to a same-tab inspection
route. That route keeps Overview, Structure, Files, Symbols, Dependencies,
Evidence, and Technical details separate; it loads file/symbol collections in
bounded pages, preserves local filters, distinguishes coverage states, and
offers read-only source excerpts only after an explicit action. Technical IDs,
hashes, provider versions, and snapshot data remain available in the secondary
Technical details section rather than the summary card.

## Functional requirements

| ID | Requirement |
|---|---|
| SFI-FR-001 | The optional source-index attachment must be independently versioned and safely ignorable by clients that do not support it. |
| SFI-FR-002 | Every eligible enumerated file must have a normalized path, language reference, file role set, raw byte count, physical line count, SHA-256 content hash, analysis status, and provenance. |
| SFI-FR-003 | Empty, terminated, unterminated, CRLF, lone-CR, and invalid-UTF-8 files must have deterministic size behavior. |
| SFI-FR-004 | Supported extractors must emit named declarations through the common symbol shape and may advertise language-specific kinds without changing core enums. |
| SFI-FR-005 | Symbols must retain source locations and explicit visibility facts when the extractor supports them; unknown visibility is not private or public. |
| SFI-FR-006 | Documentation records must retain all candidates and expose deterministic primary selection or an explicit non-present/uncertain status. |
| SFI-FR-007 | Containment and declaration relationships must reference existing entities, carry evidence/provenance, and never be inferred from path coincidence alone. |
| SFI-FR-008 | Extractor capabilities must be registered strategies; the core assembly pipeline must not grow a language switch for each new language. |
| SFI-FR-009 | Snapshot-local IDs must be opaque, unique, deterministic for equal inputs, and independent of consumer parsing. |
| SFI-FR-010 | Unsupported, unknown, and partial coverage must be visible through provenance/coverage records and must never be rewritten as absence. |
| SFI-FR-011 | A combined result must preserve authoritative per-scope snapshots and mark any aggregate as a derived projection. |
| SFI-FR-012 | Unknown typed extension blocks must be safely ignored, while the owning capability and version remain discoverable. |
| SFI-FR-013 | Default projections must omit full source content and support bounded selection of paths, symbols, documentation, and evidence. |

## Non-functional requirements

- **Determinism:** equal source bytes, scope, extractor identity/version,
  capability request, and options produce equal canonical output.
- **Compatibility:** old analyzers and consumers continue to work when the
  optional attachment is omitted or ignored.
- **Extensibility:** adding a language extractor, metric, relation, or
  documentation format is additive and namespaced.
- **Evidence safety:** every observed/partial fact is traceable to a file/span,
  analyzer/provider, or declared project metadata where applicable.
- **Token safety:** compact projections are the default; source text is an
  explicit follow-up request with bounded limits.
- **Failure isolation:** a malformed or unsupported fact batch must not corrupt
  valid module/relationship output or unrelated scope snapshots.

## Success criteria

The scenarios in [acceptance-scenarios.md](acceptance-scenarios.md) pass for
file enumeration, line counting, declaration/documentation extraction,
provenance states, containment, deterministic IDs, scope isolation,
extension handling, legacy omission, and bounded projections. The readiness
review confirms that the application PRD and architecture summary describe the
same attachment and ownership boundaries.

## Dependencies and sequencing

This child consumes the existing analyzer source scope, syntax provider,
module/source-reference output, canonical model normalization, and
multi-analyzer scope provenance. The deterministic quality child must consume
these facts before defining thresholds. The live/MCP child must consume the
same snapshots rather than reimplementing filesystem enumeration or language
semantics.
