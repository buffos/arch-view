# Source facts and symbol index canonical API/CLI contract

## Version and attachment

The optional result field is `source_index`. Its independent contract is
`arch-view.source-index/v1`; it is not a revision of the existing analyzer
protocol or canonical model schema.

```json
{
  "modules": [],
  "relationships": [],
  "source_references": [],
  "diagnostics": [],
  "source_index": {
    "schema_version": "arch-view.source-index/v1",
    "snapshots": [],
    "projection": null,
    "extensions": []
  }
}
```

Omitted `source_index` means that no source-index attachment was produced. A
consumer must not treat omission as evidence that the repository has no files,
symbols, or documentation.

## Source-index attachment shape

The following is a compact normative example. Fields marked `?` in the model
are optional in JSON; arrays that are part of the contract are emitted as empty
arrays when the producer supports the collection but found no items.

```json
{
  "schema_version": "arch-view.source-index/v1",
  "snapshots": [
    {
      "snapshot_id": "opaque-snapshot-id",
      "snapshot_kind": "scope",
      "scope_context": {
        "scope_id": "opaque-scope-id",
        "project_root": "services/catalog",
        "source_scope_fingerprint": {
          "algorithm": "hash:sha-256",
          "value": "..."
        },
        "mode": "scope"
      },
      "producer": {
        "analyzer_id": "analyzer:go",
        "analyzer_version": "1.0.0",
        "extractors": [
          { "id": "extractor:go-source-facts", "version": "1.0.0" }
        ]
      },
      "input": {
        "eligible_file_count": 1,
        "source_set_digest": {
          "algorithm": "hash:sha-256",
          "value": "..."
        },
        "requested_capabilities": [
          "source:files",
          "source:declarations",
          "source:documentation"
        ]
      },
      "capabilities": [],
      "coverage": [],
      "files": [
        {
          "id": "opaque-file-id",
          "path": "catalog/service.go",
          "language": { "id": "language:go" },
          "roles": ["role:source"],
          "size": {
            "line_count": 42,
            "byte_count": 1200,
            "content_hash": {
              "algorithm": "hash:sha-256",
              "value": "..."
            }
          },
          "analysis_status": "complete",
          "provenance": {
            "status": "observed",
            "basis": "syntax",
            "evidence_ids": [],
            "provider": "extractor:go-source-facts",
            "provider_version": "1.0.0"
          },
          "extensions": []
        }
      ],
      "symbols": [],
      "documentation": [],
      "occurrences": [],
      "relations": [],
      "metrics": [],
      "snapshot_digest": {
        "algorithm": "hash:sha-256",
        "value": "..."
      },
      "extensions": []
    }
  ],
  "projection": null,
  "extensions": []
}
```

## Entity references and compatibility

Entity IDs are opaque and meaningful only with their typed snapshot/scope
context. Consumers must not parse prefixes, paths, declaration names, or byte
ranges out of IDs. A producer may expose stable reconciliation keys and
identity basis as optional fields/extensions; these are not guaranteed to
survive renames or moves.

Existing `SourceReference` records remain supported. An adapter may link a new
record's `source_reference_ids` and convert existing ranges into `SourceSpan`
values when the file hash and coordinate convention are known. If they are not
known, the new fact is `unknown`/`partial`, not silently reinterpreted.

## Extractor boundary

The transport-neutral strategy contract is:

```text
SourceFactExtractor.Capabilities() -> CapabilityDescriptor[]
SourceFactExtractor.Extract(SourceFactInput) -> FactBatch
```

Capabilities are namespaced, for example:

- `source:files`
- `source:size`
- `source:declarations`
- `source:documentation`
- `source:visibility`
- `source:occurrences`
- `source:calls`
- `source:implementations`
- `source:metrics`

The first four are the initial source-facts target; later capabilities are
additive. The core validates and assembles output but never dispatches on a
closed language-kind enum.

## Request and projection policy

Existing analysis invocations may request the optional attachment through the
host's versioned analysis options. The option is independent of layout,
project-analyzer assignments, quality rules, and viewer state:

```json
{
  "source_index": {
    "enabled": true,
    "capabilities": ["source:files", "source:declarations"],
    "documentation": "normalized"
  }
}
```

An implementation may initially enable this by default for analyzer-produced
results or expose it as a host option; the payload contract is unchanged. An
unsupported requested capability returns explicit coverage and diagnostics,
not a fabricated empty result.

## Structural query contract

The local HTTP adapter and future MCP adapters expose the same read-model
semantics. The HTTP projection is:

- `GET /v1/models/{model_id}/source-index`
- `GET /v1/models/{model_id}/source-index/files`
- `GET /v1/models/{model_id}/source-index/symbols`
- `GET /v1/models/{model_id}/source-index/documentation`
- `GET /v1/models/{model_id}/source-index/evidence/{entity_id}`

These routes are a transport mapping, not an implementation requirement of
this child. Filters, page size, detail level, scope selector, and source
context byte/line budget are explicit. Default results include IDs, names,
paths, categories, status, locations, and short normalized documentation; full
source text is opt-in and bounded.

The structural collection routes accept repeated containment filters:

- `module_id` may occur more than once. Each value resolves to files only
  through an explicit `module -> file` `contains` relation.
- `file_id` may occur more than once. Each value selects that file directly;
  file and module selections are unioned.
- Symbol and documentation results are narrowed from those explicit files
  through `file -> symbol` `contains` or `declares` relations and explicit
  subject references.
- `language_kind` narrows symbol results to the extractor-reported kind, such
  as `go:function`, `go:constant`, or `go:struct`.
- `subject_id` may occur more than once on documentation queries. Each value
  selects documentation whose explicit subject reference matches that ID;
  this supports bounded lookup of documentation linked from a symbol or file
  without scanning the first arbitrary documentation page.

Unknown explicit IDs produce an empty bounded result rather than broadening the
query. Matching paths, names, or source-span file paths never substitute for a
containment relation. Collection responses retain the existing
`snapshot_id`, `scope_id`, `items`, `total`, and `next_cursor` envelope; they
may include optional coverage metadata so clients can distinguish observed,
partial, unsupported, and unknown results. The viewer requests 25 items at a
time and uses the returned cursor for `Load more`.

The local graph viewer may request a model with
`include_source_index=false` to keep startup compact. The historical default
model response continues to include the optional attachment for other clients;
bounded inspection queries remain the preferred human-viewer boundary.

## Determinism rules

- Normalize paths to POSIX-relative form before hashing or identity assignment.
- Hash raw file bytes with SHA-256 in v1.
- Count physical lines with the model's CRLF/CR/LF rule.
- Canonically sort capabilities, coverage, files, symbols, documentation,
  occurrences, relations, metrics, and extensions before serialization.
- Calculate `snapshot_digest` over the canonical semantic payload while
  excluding operational timestamps and the digest field itself.
- Equal scope, source bytes, analyzer/extractor versions, requested
  capabilities, and options produce equal fact values, IDs, ordering, and
  digest. A changed producer version produces a new snapshot context even if
  facts happen to be equal.

## Errors and status

The common error envelope remains:

```json
{
  "error": {
    "code": "source_fact_invalid",
    "message": "A source fact did not satisfy the v1 contract",
    "details": {}
  }
}
```

Per-file/per-capability limitations are represented in the index's status,
provenance, coverage, and diagnostics. A structurally invalid attachment is
rejected without invalidating the otherwise valid architecture result.

## CLI/export behavior

Headless model export includes `source_index` only when the caller requests or
the active analysis profile enables it. Default compact export omits raw
documentation text unless requested and never embeds complete source files.
HTML/SVG presentation may link to the existing source-inspection boundary but
does not serialize the full source index into visible architecture semantics.
