# Source facts and symbol index acceptance scenarios

These scenarios are deterministic backend/contract scenarios. They do not
claim a browser visual gate; a future viewer issue may add one when the source
facts are rendered directly.

## SFI-AC-001 — Enumerated files are first-class facts

**Given** an eligible scope containing `pkg/a.go` and `pkg/b.go`

**When** a source-index snapshot is built

**Then** it contains one file record for each path, normalized with POSIX
separators, the correct language/roles, raw byte count, SHA-256 hash, analysis
status, and file provenance; no module field is required to duplicate the path.

## SFI-AC-002 — Physical line counting is stable

**Given** files with empty bytes, `a`, `a\n`, `a\r\n`, `a\rb`, and invalid
UTF-8 bytes

**When** file size facts are calculated

**Then** line count follows the v1 LF/CRLF/CR rule, byte count/hash use raw
bytes, and the result does not depend on text decoder success.

## SFI-AC-003 — Module inspection follows containment

**Given** a module observation and reported module-to-file containment
relations

**When** a client inspects the module's source facts

**Then** the response lists the related file names and their compact size/status
facts in deterministic order, and does not infer membership from a matching
path that lacks a relation.

## SFI-AC-004 — Named declarations use the common symbol contract

**Given** a supported file containing a function, a type, an interface/trait,
and a top-level value

**When** its registered extractor advertises declaration capability

**Then** the snapshot contains named symbol records with coarse categories,
open language kinds, declaration spans, identity basis, and visibility facts
where supported; local expressions are not required to become symbols.

## SFI-AC-005 — Documentation candidates and primary selection are preserved

**Given** a module with documentation candidates in multiple files and a
symbol with a language-native doc comment

**When** the extractor builds documentation facts

**Then** every candidate is retained, the extractor marks the deterministic
primary candidate per selection group, normalized/raw text follows the
projection policy, and no core nearest-comment heuristic changes the choice.

## SFI-AC-006 — Documentation absence is not parser uncertainty

**Given** one supported declaration with no documentation, one file for which
documentation extraction is unsupported, and one file whose parse failed

**When** documentation coverage is assembled

**Then** the first subject is `absent`, the second is `unsupported`, and the
third is `unknown` or `partial` with evidence/diagnostics; none is reported as
an observed empty comment.

## SFI-AC-007 — Spans are hash-linked and unambiguous

**Given** a declaration span produced from a Tree-sitter point range

**When** it is normalized into the source-index contract

**Then** its byte offsets are raw UTF-8 coordinates, line/column values are
one-based, end is exclusive, and its content hash matches the referenced file.

## SFI-AC-008 — Extractors are open/closed strategies

**Given** a newly registered language extractor with a new language kind and a
new documentation format

**When** the source-index service resolves the extractor by capability

**Then** the new facts are accepted through open language-kind/format values
without a core language switch or changes to existing record fields, subject to
normal validation.

## SFI-AC-009 — IDs are opaque and deterministic

**Given** equal source bytes, scope, analyzer/extractor versions, capabilities,
and options

**When** two snapshots are built

**Then** their canonical facts, ordering, opaque IDs, and semantic digest are
equal. A consumer can use IDs but cannot rely on their text encoding. A rename
or move may change IDs/stable keys and is not treated as an in-place identity
guarantee.

## SFI-AC-010 — Scope isolation is preserved

**Given** two analyzer scopes that contain equal relative paths but have
different scope IDs or extractor contexts

**When** a combined source index is assembled

**Then** both authoritative snapshots remain available, references are
scope/snapshot-qualified, and no records or relations are merged solely by
path/name coincidence.

## SFI-AC-011 — Partial extraction does not erase valid facts

**Given** one file whose extractor returns a malformed declaration batch and
another file with valid file facts

**When** the snapshot is normalized

**Then** the malformed batch yields a structured diagnostic and partial/unknown
coverage, valid file facts remain, and unrelated architecture modules and
relationships are unchanged.

## SFI-AC-012 — Extensions are safely additive

**Given** a valid snapshot containing an extension block from an unknown
namespace/capability

**When** an older consumer validates and reads the snapshot

**Then** it ignores the unknown typed payload, preserves core facts, and does
not reinterpret the extension as a core field or fail the architecture result.

## SFI-AC-013 — Legacy results remain compatible

**Given** an analyzer result without a `source_index` field

**When** an existing client consumes it

**Then** modules, relationships, source references, diagnostics, and existing
viewer/export behavior remain valid. Omission is not interpreted as an empty
source scope, and a future client may request a registered host-side index
independently.

## SFI-AC-014 — Compact projections protect tokens

**Given** a module containing many files and symbols

**When** a viewer, CLI, or future MCP adapter requests default source facts

**Then** the result is bounded, deterministically ordered, includes names,
locations, status, provenance, and selected documentation, omits full source
content, and requires an explicit bounded request for source context.

## SFI-AC-015 — Metric facts are formula-versioned

**Given** an extractor or downstream quality engine emits a numeric metric

**When** it is stored in the source-index metric collection

**Then** it has a namespaced metric ID, typed value, formula ID/version, subject
reference, and provenance. A later formula revision creates a new version or
metric identity rather than silently changing the meaning of an old value.

## SFI-AC-016 — Callable metrics require explicit body evidence

**Given** a callable whose extractor can identify its implementation body and
one callable without a usable body span

**When** the extractor advertises `source:callable.metrics`

**Then** the first symbol may carry a hash-linked `body_span` and provider-
owned formula-versioned body-line, complexity, or nesting metrics. The second
symbol receives explicit unknown/unsupported/partial coverage, and the core
does not estimate a body range from braces, names, or neighboring symbols.
