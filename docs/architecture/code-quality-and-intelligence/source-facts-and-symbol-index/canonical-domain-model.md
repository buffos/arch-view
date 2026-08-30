# Source facts and symbol index canonical domain model

## Modeling boundary

This model owns compact, evidence-backed facts about source files and named
declarations. The existing canonical model remains authoritative for
architecture modules, dependency relationships, layers, cycles, and derived
graph projections. Quality policy and live snapshot lifecycle consume this
model but do not redefine it.

## SourceIndex

`SourceIndex` is the optional top-level attachment on an analysis result and
canonical model. Its v1 shape is:

```text
SourceIndex {
  schema_version: "arch-view.source-index/v1"
  snapshots: SourceIndexSnapshot[]          // authoritative, per scope
  projection?: SourceIndexSnapshot           // explicitly derived view
  extensions: ExtensionBlock[]
}
```

An ordinary single-analyzer result has one scope snapshot. A combined
multi-analyzer result retains all authoritative snapshots and may include a
`combined_projection` snapshot. The projection records its source snapshot IDs
and never becomes an authority of its own.

## SourceIndexSnapshot

```text
SourceIndexSnapshot {
  snapshot_id: OpaqueId
  snapshot_kind: "scope" | "combined_projection"
  scope_context: ScopeContext
  producer: ProducerContext
  input: InputContext
  capabilities: CapabilityDescriptor[]
  coverage: CoverageRecord[]
  files: FileRecord[]
  symbols: SymbolRecord[]
  documentation: DocumentationRecord[]
  occurrences: SymbolOccurrence[]
  relations: CodeRelation[]
  metrics: MetricFact[]
  snapshot_digest: ContentDigest
  extensions: ExtensionBlock[]
}
```

The semantic digest is calculated over canonicalized fields except
`snapshot_digest` and operational metadata. No wall-clock timestamp is part of
the semantic snapshot. An optional operational envelope may contain generated
time or freshness information, but it is not used for equality, cache identity,
or deterministic export.

### ScopeContext

```text
ScopeContext {
  scope_id: OpaqueId
  project_root: RelativePath
  source_scope_fingerprint: ContentDigest
  source_policy_fingerprint?: ContentDigest
  repository_context?: RepositoryContext
  mode: "scope" | "combined"
  source_snapshot_ids?: OpaqueId[]
}
```

Paths are relative to the normalized invocation/repository root. Absolute local
paths and machine-specific working directories do not enter the semantic
contract. `source_snapshot_ids` is required for a combined projection and
forbidden for a normal scope snapshot.

### ProducerContext

```text
ProducerContext {
  analyzer_id: NamespacedId
  analyzer_version: string
  extractors: ExtractorIdentity[]
  protocol_version?: string
}

ExtractorIdentity {
  id: NamespacedId
  version: string
}
```

### InputContext

```text
InputContext {
  eligible_file_count: integer >= 0
  source_set_digest: ContentDigest
  requested_capabilities: NamespacedId[]
}
```

The source-set digest covers normalized eligible paths and raw content hashes.
It is distinct from the semantic snapshot digest so a consumer can distinguish
input changes from extraction changes.

## FileRecord

```text
FileRecord {
  id: OpaqueId
  path: RelativePath
  language: LanguageRef
  roles: NamespacedId[]
  size: FileSize
  analysis_status: "complete" | "partial" | "unparsed" | "unknown"
  provenance: FactProvenance
  extensions: ExtensionBlock[]
}

LanguageRef {
  id: NamespacedId
  dialect?: string
}

FileSize {
  line_count: integer >= 0
  byte_count: integer >= 0
  content_hash: ContentDigest
}

ContentDigest {
  algorithm: NamespacedId       // v1 core requires "hash:sha-256"
  value: lowercase hexadecimal string
}
```

File IDs are opaque. A producer may expose `stable_key: "file-path:<path>"`
through an extension for reconciliation, but consumers must not construct or
parse the public ID. A file is emitted whenever an eligible file can be
enumerated, even if parsing fails.

V1 physical line counting operates on raw bytes and recognizes LF, CRLF, and
lone CR as one line terminator. For non-empty content:

```text
line_count = number_of_line_terminators
             + (1 when the final byte is not a line terminator, otherwise 0)
```

Thus empty bytes are `0`, `a` is `1`, `a\n` is `1`, `a\n\nb` is `3`, and
`a\r\nb` is `2`. Byte count and SHA-256 are calculated before text decoding.

## SymbolRecord

```text
SymbolRecord {
  id: OpaqueId
  name: string
  qualified_name?: string
  category: "callable" | "type" | "namespace" | "value" | "member" | "macro" | "unknown"
  language_kind?: NamespacedId
  visibility: VisibilityFact
  locations: SymbolLocation[]
  documentation_ids?: OpaqueId[]       // derived lookup projection
  stable_key?: string                  // reconciliation hint, never authority
  identity_basis?: NamespacedId        // e.g. "identity:qualified-name"
  provenance: FactProvenance
  extensions: ExtensionBlock[]
}

VisibilityFact {
  classification: "public" | "protected" | "private" | "internal" |
                  "package" | "module" | "default" | "unknown"
  language_value?: string
  provenance: FactProvenance
}

SymbolLocation {
  kind: NamespacedId                  // normally "location:declaration"
  span: SourceSpan
  source_reference_ids?: string[]      // compatibility/evidence links
}
```

The category is intentionally coarse and stable. A language extractor can add
`go:function`, `python:class`, `typescript:interface`, `rust:trait`, or another
open `language_kind` without changing the core category vocabulary. V1 named
declarations include functions/methods, classes/structs/interfaces/traits/
enums, type aliases, namespaces/modules, top-level constants/variables, and
macros when supported. Local variables and arbitrary expressions are not
required symbol records.

`documentation_ids` is a materialized lookup convenience. The authoritative
attachment is `DocumentationRecord.subject_ref`; normalizers must derive the
array from that relation or omit it, never accept contradictory independent
values.

## DocumentationRecord

```text
DocumentationRecord {
  id: OpaqueId
  subject_ref: EntityRef                    // file, module, or symbol
  selection_group: NamespacedId
  format: NamespacedId                       // e.g. "go:doc", "python:docstring"
  raw_text?: string
  normalized_text?: string
  spans: SourceSpan[]
  source_reference_ids?: string[]
  attachment_basis: NamespacedId            // language-native, syntax, convention, heuristic
  precedence_rank?: integer >= 0
  is_primary: boolean
  status: "present" | "absent" | "unknown" | "unsupported" | "partial"
  completeness: "complete" | "summary_only" | "truncated" | "unknown"
  provenance: FactProvenance
  extensions: ExtensionBlock[]
}
```

When extraction is attempted for an eligible subject, the extractor emits a
primary record with `present` or an explicit non-present/uncertain status. All
candidate records are retained. The extractor, not the core, supplies
`selection_group`, `precedence_rank`, and `is_primary` according to the
language's documentation rules. The core validates at most one primary per
subject and selection group and applies deterministic tie validation; it never
chooses the nearest comment across languages.

Documentation text is targeted fact content. Raw text may be omitted from a
compact projection; normalized text and evidence are sufficient for a default
LLM/search response. No complete source file is embedded by this record.

## SourceSpan

```text
SourceSpan {
  file_id: OpaqueId
  start: SpanPosition
  end: SpanPosition
  coordinate_system: "utf8-byte"
  content_hash: ContentDigest
}

SpanPosition {
  byte_offset: integer >= 0
  line: integer >= 1
  column: integer >= 1
}
```

Offsets and columns are byte coordinates in the file's original UTF-8 byte
stream. End is exclusive. `end` must not precede `start`, and the span hash
must equal the referenced file hash. Existing zero-based Tree-sitter points are
adapted to one-based values at this boundary; legacy `SourceReference` remains
available for old consumers.

## FactProvenance and coverage

```text
FactProvenance {
  status: "observed" | "absent" | "unknown" | "unsupported" | "partial"
  basis: "syntax" | "compiler" | "project_metadata" | "convention" | "heuristic"
  evidence_ids: string[]
  provider: NamespacedId
  provider_version: string
  score?: {
    value: number
    scale_id: NamespacedId
    scale_version: string
  }
}

CoverageRecord {
  capability: NamespacedId
  subject_kind: "file" | "symbol" | "documentation" | "occurrence" | "relation" | "metric"
  status: "observed" | "absent" | "unknown" | "unsupported" | "partial"
  eligible_count?: integer >= 0
  observed_count?: integer >= 0
  reason?: string
  provenance: FactProvenance
}
```

`unknown` means the producer could not determine the fact. `unsupported` means
the producer does not implement the capability. Neither means `absent`, and
neither is a quality violation by itself.

## EntityRef

```text
EntityRef {
  kind: "module" | "file" | "symbol" | "documentation" |
        "occurrence" | "relation" | "metric"
  id: OpaqueId
  snapshot_id?: OpaqueId
  scope_id?: OpaqueId
}
```

Index entities require their snapshot context when referenced from a combined
projection. Existing architecture module references may use the established
scope-qualified model ID. Consumers compare typed references and do not infer
meaning from ID text.

## CodeRelation

```text
CodeRelation {
  id: OpaqueId
  category: "contains" | "declares" | "unknown"
  language_kind?: NamespacedId
  from_ref: EntityRef
  to_ref?: EntityRef
  unresolved_target?: UnresolvedTarget
  evidence_spans: SourceSpan[]
  provenance: FactProvenance
  extensions: ExtensionBlock[]
}

UnresolvedTarget {
  display_name: string
  qualified_name?: string
  language_kind?: NamespacedId
}
```

V1 uses `contains` for known module-to-file and file-to-symbol hierarchy and
`declares` for declaration evidence where the extractor can state it. Future
`calls`, `inherits`, `implements`, and `references` are additive relation
categories or extension-owned kinds; they do not become arrays on symbols.
Exactly one of `to_ref` and `unresolved_target` is present.

## SymbolOccurrence

```text
SymbolOccurrence {
  id: OpaqueId
  symbol_ref?: EntityRef
  target_name: string
  source_span: SourceSpan
  occurrence_kind: NamespacedId
  resolution_status: "resolved" | "unresolved" | "ambiguous" | "not_attempted" | "unsupported"
  provenance: FactProvenance
  extensions: ExtensionBlock[]
}
```

Occurrences preserve a useful source location even when no target can be
resolved. V1 does not require call extraction; an extractor advertises it
through capabilities before emitting call occurrences.

## MetricFact

```text
MetricFact {
  id: OpaqueId
  subject_ref: EntityRef
  metric_id: NamespacedId
  value: {
    kind: "integer" | "decimal" | "boolean" | "text"
    value: scalar
  }
  unit?: NamespacedId
  formula_id: NamespacedId
  formula_version: string
  provenance: FactProvenance
  extensions: ExtensionBlock[]
}
```

Intrinsic file size remains in `FileSize` so it is always available. The
versioned metric slot is reserved for derived facts such as future complexity,
fan-in, or documentation coverage; thresholds and findings belong elsewhere.

## Extensibility contract

```text
ExtensionBlock {
  namespace: NamespacedId
  schema_version: string
  capability: NamespacedId
  payload: typed JSON value
}
```

Extension payloads are owned by their namespace and capability. Unknown blocks
are preserved when round-tripping where possible and ignored by unsupported
consumers. A new capability adds a block or record category; it does not
repurpose a core field.

## Extractor strategy

The conceptual registered strategy is:

```text
SourceFactExtractor {
  ID() -> NamespacedId
  Version() -> string
  Capabilities() -> CapabilityDescriptor[]
  Extract(input: SourceFactInput) -> FactBatch
}
```

`SourceFactInput` includes the eligible file path/bytes, syntax tree when
available, scope context, and requested capabilities. `FactBatch` contains
language-owned provisional declarations, documentation candidates, spans,
relations, occurrences, and metric facts. The core normalizer validates the
batch, assigns opaque IDs, deduplicates, derives lookup projections, computes
coverage, sorts records, and assembles the snapshot. It does not switch on
language names to decide semantics.

## Invariants and canonicalization

- Relative paths use `/`, contain no empty or `..` segment, and are unique within
  a scope snapshot.
- File IDs, symbol IDs, record IDs, and relation IDs are unique within their
  snapshot and are not parsed by consumers.
- Every indexed span resolves to an emitted file and matching content hash.
- Every relation endpoint resolves, unless `unresolved_target` is supplied.
- A containment relation never changes an architecture relationship and cannot
  be inferred from a matching path alone.
- Documentation primary selection is unique per subject/selection group when a
  present candidate exists. All candidates remain retained.
- Capabilities, coverage, and extension blocks are sorted by canonical IDs.
  Files sort by path; other collections sort by subject/path, primary span
  byte offset, category/kind, name, and opaque ID.
- The semantic digest excludes operational timestamps and is computed after
  canonical sorting. Equal inputs, producer versions, requested capabilities,
  and options therefore produce equal snapshot contents and digest.

## Lifecycle and events

`scope_discovered -> files_enumerated -> parsed -> extracted -> normalized ->
validated -> attached`.

Possible events are `SourceScopeEnumerated`, `SourceFactsExtracted`,
`SourceFactsCoverageChanged`, `SourceIndexValidated`, and
`SourceIndexAttached`. A file parse/extraction failure moves only that file or
capability to `partial`, `unparsed`, `unknown`, or `unsupported`; it does not
invalidate unrelated scope facts or the architecture model.
