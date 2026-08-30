# Source facts and symbol index requirements gap analysis

## Scope examined

This pass covers the bounded [Source facts and symbol index](../../../../.okf/capabilities/code-quality-and-intelligence/source-facts-and-symbol-index.md) child, the common analysis/model contracts, the Tree-sitter syntax boundary, the five current language analyzers, and the existing multi-analyzer scope orchestration.

## Confirmed strong areas

- **Observed in code:** analysis already has a shared result envelope, source
  evidence, diagnostics, and language-neutral module observations.
- **Observed in code:** the syntax provider exposes parse trees and precise
  byte/point ranges, so declaration extraction can be added behind a shared
  provider rather than making the host parse languages itself.
- **Observed in code:** analyzers already discover file paths and some
  analyzer-specific file metadata, while orchestration preserves independent
  scope provenance and cache identity.
- **Inferred from architecture:** an optional sibling payload can be added to
  analysis and canonical model results without changing module-level graph
  semantics or breaking old clients.
- **User-confirmed target behavior:** files, documentation, declarations,
  visibility, and deterministic source metrics must be directly consumable by
  the viewer and future compact query/reporting surfaces.

## Blocking gaps resolved by this specification

| Gap | Impact | Resolution |
|---|---|---|
| File facts would otherwise be scattered through module metadata | High: every consumer would invent a different file schema | Add an optional `SourceIndex` sibling with first-class scoped file records and containment relations. |
| `FileRecord` could become a language-specific catch-all | High: later metrics and analyzers would require repeated refactors | Keep the core record limited to path, language, roles, intrinsic size, status, provenance, and typed extension blocks. |
| Symbol ownership and semantic relations were ambiguous | High: callers, implementations, and declarations could be conflated | Keep symbols as named declarations; represent `contains`, `declares`, and future semantic edges as extensible relations/occurrences. |
| Documentation absence could be confused with parser limitations | High: quality checks would report false missing comments | Use fact provenance states `present`, `absent`, `unknown`, `unsupported`, and `partial`; let each extractor define attachment precedence. |
| Source locations had incompatible coordinate conventions | High: evidence and future fixes could point at the wrong bytes | Fix UTF-8 byte offsets, one-based line/column, end-exclusive spans, and content-hash linkage. |
| IDs could become accidental public paths or ranges | High: renames and multi-scope analysis would break consumers | Assign opaque snapshot-local IDs; expose optional stable keys and identity basis only for reconciliation. |
| Adding languages would require central switches | High: violates the agreed open/closed direction | Register `SourceFactExtractor` strategies with declared capabilities; core orchestration never dispatches on language branches. |
| Multi-analyzer results could merge equal paths incorrectly | High: one physical path can have different scope/analyzer semantics | Preserve authoritative per-scope snapshots and make any combined view a deterministic projection. |
| Future metrics had no versioned formula boundary | Medium: values could not be compared reproducibly | Reserve typed `MetricFact` records with namespaced metric IDs and mandatory formula versions. |

## Deferrable implementation details

- The registry may use Go interfaces, a generated registry, or a process/plugin
  adapter; the contract does not prescribe the mechanism.
- The first implementation may support only Go declarations and documentation.
  Other languages advertise `unsupported` or `unknown` capability coverage
  until their extractors are registered.
- The core may calculate IDs with a digest or another deterministic allocator;
  the observable contract is opacity, uniqueness within a snapshot, and
  reproducibility for equal inputs.
- A query service may build path/name/documentation indexes for speed. Those
  are read-model projections and cannot replace the source-index records.

## Exact assumptions

- Eligible files are the files remaining after the already-resolved source
  scope policy. By default, excluded files are not emitted as `FileRecord`s;
  an explicit diagnostic/coverage option may include exclusion evidence without
  making excluded files part of the analyzed fact set.
- A file record is emitted whenever the producer can enumerate an eligible
  file, even if parsing fails. Its status and provenance explain what is
  unavailable.
- Physical line count treats CRLF and lone CR as line terminators, counts a
  final unterminated line, and does not count a trailing empty line after a
  terminator. Empty bytes therefore have zero lines; `a` and `a\n` have one.
- The v1 content hash is SHA-256 over the raw file bytes. Byte count is also
  measured before text decoding.
- Documentation text is targeted fact content, not a source-code mirror. A
  producer may omit raw text and retain normalized text plus evidence according
  to the requested projection.
- A combined projection sorts records by canonical reference/path/order and
  never infers cross-scope relationships that no extractor reported.

## Readiness conclusion

No High or Medium specification gaps remain for the bounded source-facts
contract. The remaining work is implementation of the core index seam and
registered language extractors, followed by the application-synthesis update
and delivery planning owned by the next workflow step.

## Artifact impact

- **Capability:** the linked exact-spec set defines records, relations,
  provenance, extraction, identity, coverage, attachment, and compatibility.
- **Product:** module inspection now has a precise future source-facts contract
  without claiming that it is implemented.
- **Architecture:** analysis/model attachment, per-scope ownership, and
  projection boundaries are explicit.
- **Delivery:** no implementation issues are created in this pass.
