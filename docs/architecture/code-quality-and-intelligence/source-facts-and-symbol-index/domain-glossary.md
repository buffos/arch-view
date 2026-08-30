# Source facts and symbol index domain glossary

| Term | Category | Definition | Distinction |
|---|---|---|---|
| Source index | capability object | Optional, versioned collection of compact source facts attached beside the architecture model. | It is not the canonical architecture graph and does not contain full source code by default. |
| Source index snapshot | business object | One deterministic fact set for one analysis scope and extractor context. | A combined projection is derived from snapshots and is not authoritative. |
| Analysis scope | value object | The normalized project root, source policy, analyzer identity/version, and source set used for one snapshot. | It is narrower than the opened repository when assignments or nested roots apply. |
| File record | business object | A first-class observation of one eligible repository-relative source file. | It is not a module and does not own semantic imports. |
| File role | classification | Namespaced classification such as source, test, generated, declaration, or fixture. | Roles are extensible tags, not a closed language enum. |
| File size | value object | Raw byte count, deterministic physical line count, and content hash. | It is intrinsic source evidence, not a quality finding. |
| Content hash | value object | SHA-256 digest of raw file bytes in v1. | It supports freshness and evidence linkage; it is not a file identity guarantee across renames. |
| Symbol record | business object | A named declaration such as a callable, type, namespace, value, member, or macro. | It is not every AST node, local expression, or semantic call edge. |
| Symbol category | stable classification | Coarse language-neutral value: `callable`, `type`, `namespace`, `value`, `member`, `macro`, or `unknown`. | `language_kind` carries an extractor's more specific open vocabulary. |
| Language kind | extensibility value | Namespaced language-specific declaration kind, for example `go:function` or `rust:trait`. | It must not force a core schema enum expansion. |
| Documentation record | business object | Evidence-backed documentation candidate attached to a file, module, or symbol. | It is separate from the symbol and can represent absence or extraction uncertainty. |
| Documentation candidate | value object | One documentation source retained before or after primary selection. | All candidates remain queryable even when one is primary. |
| Primary documentation | projection fact | The candidate selected by the language extractor's deterministic precedence policy. | The core never chooses a nearest comment across languages. |
| Source span | value object | End-exclusive file location using UTF-8 byte offsets and one-based line/column values. | Ranges are evidence and navigation coordinates, not public identity. |
| Fact provenance | value object | Status, evidence, provider, version, and basis explaining how a fact was obtained. | `unknown` and `unsupported` are different from `absent`. |
| Fact status | reporting classification | `observed`, `absent`, `unknown`, `unsupported`, or `partial`. | It describes evidence coverage, not application/job lifecycle. |
| Source fact extractor | strategy/plugin | Registered provider that turns one language file and syntax tree into fact batches. | It owns language semantics; the core owns assembly and validation. |
| Capability | contract value | Namespaced extractor behavior such as declarations, documentation, visibility, calls, or implementations. | A capability can be unsupported without invalidating the file record. |
| Fact batch | transient object | Extractor output for one file before core normalization and ID assignment. | It is not the persisted/indexed snapshot. |
| Entity reference | value object | Typed reference to a file, symbol, module, documentation record, or other indexed entity. | References include snapshot/scope context where needed; consumers do not parse IDs. |
| Code relation | business object | Evidence-backed connection such as `contains` or `declares`, with open future categories. | It is not a duplicate array on a symbol or a module-level architecture relationship. |
| Symbol occurrence | business object | A source location where a symbol or unresolved target is used, called, or otherwise mentioned. | It can remain unresolved and is separate from declaration identity. |
| Metric fact | business object | Versioned typed measurement attached to an entity, such as a future complexity value. | It is a fact with formula provenance, not a threshold violation. |
| Extension block | extensibility value | Namespaced, versioned, capability-owned payload for additive facts. | Unknown blocks are ignored rather than interpreted as core fields. |
| Scope snapshot | lifecycle object | Authoritative snapshot produced for one analyzer/project scope. | Multiple scopes with equal paths are not merged solely by path. |
| Combined projection | read model | Deterministically aggregated view over authoritative scope snapshots. | It must preserve provenance and cannot invent cross-scope semantics. |
| Structural source search | query capability | Deterministic filtering of indexed paths, names, categories, documentation, and facts. | It is not semantic ranking or full-text source retrieval; those belong to a later query surface. |

Canonical status vocabularies:

- Fact coverage: `observed`, `absent`, `unknown`, `unsupported`, `partial`.
- File analysis: `complete`, `partial`, `unparsed`, `unknown`.
- Resolution: `resolved`, `unresolved`, `ambiguous`, `not_attempted`,
  `unsupported`.
- Core symbol category: `callable`, `type`, `namespace`, `value`, `member`,
  `macro`, `unknown`.
