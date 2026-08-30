# Source facts and symbol index canonical use cases

## Application services

### SourceFactExtractorRegistry

- `RegisterSourceFactExtractor`
- `ListSourceFactCapabilities`
- `ResolveSourceFactExtractor`

### SourceIndexService

- `EnumerateEligibleSourceFiles`
- `BuildSourceIndexSnapshot`
- `ValidateSourceIndex`
- `AttachSourceIndex`
- `BuildCombinedSourceIndexProjection`

### SourceFactQueryService

- `InspectModuleSourceFacts`
- `FindFiles`
- `FindSymbols`
- `FindDocumentation`
- `GetSourceFactEvidence`

The query service is a deterministic structural read model. It does not own
semantic ranking, live watching, MCP transport, quality policy, or source
editing.

## `RegisterSourceFactExtractor` — command

**Input:** extractor ID/version, supported language/dialect, capability
descriptors, and the registered strategy implementation or process adapter.

**Rules:** IDs and versions are explicit; capabilities are namespaced and
sorted; duplicate registrations for the same ID/version are idempotent; a
conflicting implementation is rejected. Registration does not require a core
language switch.

**Output:** immutable registry entry and capability list.

## `EnumerateEligibleSourceFiles` — query

**Input:** an existing resolved analysis scope.

**Output:** normalized POSIX-relative paths, raw bytes/content hashes when
available, language assignment, roles, and source-scope fingerprint.

**Rules:** use the existing source-scope policy and fixed exclusions; do not
re-discover project roots or silently broaden the scope. An enumerated file
remains eligible even when later parsing fails.

## `BuildSourceIndexSnapshot` — command

**Input:** eligible files, scope context, analyzer/provider identity, requested
extractor capabilities, and optional syntax trees.

**Responsibilities:**

1. Create mandatory file records and intrinsic size facts.
2. Resolve registered extractors by declared capability.
3. Give each extractor language-owned inputs and collect provisional batches.
4. Validate spans, records, relation endpoints, documentation groups, and
   extension ownership.
5. Assign opaque IDs, derive stable lookup projections, calculate coverage,
   canonicalize ordering, and calculate the snapshot digest.

**Outcome:** one authoritative `SourceIndexSnapshot`, including explicit
partial/unknown/unsupported coverage. A parser or extractor failure does not
discard valid file facts or unrelated extractor output.

## `ValidateSourceIndex` — command

**Input:** a source-index attachment or snapshot.

**Rules:** validate schema version, required fields, path safety, hash format,
span coordinates, record uniqueness, reference resolution, relation endpoint
rules, documentation primary uniqueness, typed metric values, capability
coverage, and digest consistency. Unknown extension blocks are ignored after
namespace/version validation.

**Outcome:** valid snapshot or structured diagnostics. Validation must not
reinterpret unknown facts as absent.

## `AttachSourceIndex` — command

**Input:** an `AnalysisResult` or canonical model and a validated source-index
attachment.

**Rules:** attach only as the optional `source_index` sibling. Preserve all
existing modules, relationships, references, diagnostics, and derived data.
An analyzer or host that has no source-index capability may omit the field.

**Outcome:** a backward-compatible result. Existing consumers that ignore the
field receive the same architecture semantics.

## `BuildCombinedSourceIndexProjection` — command

**Input:** authoritative source snapshots from the completed/partial
multi-analyzer run.

**Responsibilities:** retain each source snapshot, namespace entity references
by scope/snapshot, sort records deterministically, and optionally construct a
read-only combined projection.

**Rules:** equal paths from different scopes are not merged solely by path;
module/file containment is retained only when reported; no cross-language call,
type, implementation, or dependency relation is inferred. A failed scope is
represented in aggregate diagnostics/status and does not erase successful
snapshots.

## `InspectModuleSourceFacts` — query

**Input:** a module reference, scope/projection selector, and a field
projection/limit.

**Output:** module-linked file names, file sizes/status, declarations,
documentation summaries, visibility, relation evidence, and content hashes as
requested. Full source content is not returned by default.

**Rules:** follow explicit containment relations and preserve scope/provenance;
do not equate path/name matches with membership. Results are bounded and
canonically ordered for stable UI/MCP consumption.

## `FindFiles` / `FindSymbols` / `FindDocumentation` — queries

**Input:** structural filters such as normalized path prefix/glob, language,
role, symbol name/qualified name/category/language kind, documentation text,
one or more explicit subjects, status, and scope. Pagination/maximum result
count is explicit.

**Output:** compact entity projections with IDs, names, locations, status,
provenance, and optional bounded documentation text.

**Rules:** matching is deterministic and case policy is explicit per query;
default ordering is path/span/name/ID. These queries do not claim semantic
relevance or replace a future full-text/embedding search capability.

## `GetSourceFactEvidence` — query

**Input:** entity/relation/documentation ID and an explicit detail budget.

**Output:** source spans, legacy source-reference links, normalized/raw
documentation when requested, and bounded source context through the existing
source-inspection boundary.

**Rules:** the index itself remains source-code-free by default. Requests must
be read-only, path-safe, scope-bound, and bounded by bytes/lines.

## Failure model

- `SourceScopeUnavailable`
- `SourceFileEnumerationFailed`
- `SourceFileHashFailed`
- `SourceExtractorUnavailable`
- `SourceExtractorFailed`
- `SourceFactInvalid`
- `SourceSpanInvalid`
- `SourceReferenceInvalid`
- `SourceRelationInvalid`
- `DocumentationSelectionInvalid`
- `MetricFactInvalid`
- `SourceIndexDigestMismatch`
- `SourceIndexCapabilityUnsupported`

File-level failures produce a file status or capability coverage diagnostic
where possible. Attachment fails closed for an invalid index payload, while
the existing architecture result remains usable without the optional field.

## Architecture-neutral mapping

The extractor can run in-process, in an external analyzer, or behind a future
plugin host. The stable boundary is the fact batch plus the normalized
source-index contract. Viewer, quality, CLI, and MCP adapters consume the same
facts; none owns language parsing or source-index identity.
