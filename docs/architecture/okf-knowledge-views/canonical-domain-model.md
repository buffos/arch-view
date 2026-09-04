# Configurable OKF knowledge views — canonical domain model

## Purpose

This model defines the semantic core that every implementation of the OKF
viewer must preserve. It describes source consumption, profile interpretation,
projection, navigation, inspection, and persistence behavior without choosing a
database, package layout, transport, or UI framework.

## Modeling principles

1. Source facts are immutable input; profile interpretation is separate.
2. One bundle is one source boundary. Multiple bundles are never one aggregate
   or one projection.
3. Containment and semantic links are different relationship kinds with
   different traversal meaning.
4. Profiles own presentation policy, not source identity or source content.
5. Renderer-neutral projection precedes layout and drawing.
6. Diagnostics are domain output, not incidental log messages.
7. Determinism, provenance, safety, and visible limits are domain invariants.
8. Aggregates are used where consistency rules require a boundary, not as a
   reflection of storage tables or frontend screens.

## Ubiquitous language

The canonical terms are defined in
[domain-glossary.md](domain-glossary.md). The most important semantic
distinctions are:

- an OKF bundle is a source boundary; a projection is one profile's view of it;
- a concept document is source identity; a scene node is a projection result;
- containment drives traversal; semantic links do not expand traversal;
- declared state is source evidence; effective state is profile interpretation;
- a rule strategy is registered executable implementation; a rule invocation is
  profile data;
- Save updates an existing project-local profile; Save As creates a new one;
- planning state belongs to the Arch View planning map and is not automatically
  the state of an OKF concept.

## Subdomains and bounded-context candidates

### Core domain: knowledge-view projection

The core value is turning an arbitrary bundle into an understandable,
source-backed projection under a chosen rendering costume. The central
complexity is preserving distinctions while allowing different users to
interpret the same source differently.

### Supporting context: bundle consumption

Owns project-tree discovery, candidate validation, bundle identity, lossless
source indexing, path boundaries, and provenance. It does not decide how
concepts look or which relationships a profile displays.

### Supporting context: view configuration

Owns graph/profile bindings, built-in and project-local profile definitions,
profile composition, rule invocation configuration, profile validation, and
safe persistence in the nearest Arch View configuration.

### Supporting context: interaction and inspection

Owns focus roots, depth choices, breadcrumb/back history, selected concept
details, safe Markdown presentation, and read-side diagnostic views. It does
not alter source facts or redefine projection rules.

### Policy and extension area

The rule registry, relationship adapters, presentation-token catalog, detail
renderers, and scale/safety policies are extension areas shared by the core
contexts. They vary implementation behavior without changing the canonical
source and projection language.

These contexts are candidates for conceptual seams, not mandatory deployment
boundaries. A modular monolith, layered service, or other architecture may
implement them together while preserving their ownership.

## Aggregate design

### BundleCatalog

BundleCatalog is the project-scoped aggregate root for the current discovery
result.

It owns:

- the Arch View project identity;
- discovered bundle candidates and normalized BundleIds;
- candidate validity and diagnostic summaries;
- deterministic selector ordering;
- the selected BundleId, if one is selected.

It protects:

- bundle IDs are project-relative normalized POSIX paths;
- candidates outside the project or excluded paths are not selectable;
- only validated bundles are selectable;
- selector ordering is deterministic;
- selecting one bundle cannot implicitly select or merge another.

Discovery refresh replaces the catalog snapshot as one domain operation. The
catalog does not own the contents of a BundleIndex.

### BundleIndex

BundleIndex is an immutable aggregate root representing one validated bundle
snapshot.

It owns:

- the BundleId and source revision/fingerprint;
- ConceptDocuments and SourceFacts;
- normalized source paths and frontmatter;
- Markdown bodies and extracted link facts;
- explicit hierarchy claims, filesystem facts, and provenance;
- normalized containment and semantic-link relationships;
- source-level diagnostics.

It protects:

- all source facts remain addressable by stable concept identity;
- local resolution cannot cross the bundle boundary;
- source content is not mutated by profile or projection operations;
- relationship provenance remains available;
- normalization never silently discards a conflicting source claim.

The index is immutable so multiple profiles can evaluate the same source
snapshot without sharing mutable interpretation state.

### ViewProfile

ViewProfile is the aggregate root for one rendering costume.

It owns:

- profile identity and origin (built-in or project-local);
- ordered base-profile references;
- rule invocations and priorities;
- hierarchy and relationship projection choices;
- presentation tokens, labels, facets, roles, legend, and detail settings;
- depth and scale settings;
- per-profile ELK layout preferences;
- profile revision and validation status.

It protects:

- built-ins cannot be overwritten;
- project-local profile IDs are unique within the configuration;
- composition order is explicit and cycle-free;
- rule references and parameters match supported schemas;
- limits are positive and cannot exceed application hard caps;
- profile validation does not depend on registry registration order;
- a profile cannot change BundleId, source facts, or source revision.

Save As creates a new ViewProfile with copied composition references and
overrides. It is not a second name for Save.

### OKFViewConfiguration

OKFViewConfiguration is the aggregate root for the optional OKF section in the
nearest Arch View configuration document.

It owns:

- the optional default BundleId;
- graph-to-profile bindings;
- project-local ViewProfiles;
- active viewer preferences that are intentionally project-local.

It protects:

- a binding references a known project-local profile or an explicitly known
  built-in profile;
- rename updates all affected bindings as one operation;
- delete requires reassignment or explicit neutral fallback;
- stale graph/profile references remain diagnosable rather than silently
  rewritten;
- a saved configuration is validated as one document while unrelated
  configuration sections remain preserved.

### ProjectionSnapshot

ProjectionSnapshot is an immutable aggregate root representing one evaluated
renderer-neutral scene.

It owns:

- source BundleId and source revision;
- profile identity and profile revision;
- FocusRoot and navigation settings;
- visible projected concepts and relationships;
- annotations, presentation tokens, legend data, and diagnostics;
- visible and hidden counts;
- projection status and deterministic ordering metadata.

It protects:

- every visible item is traceable to source facts and profile decisions;
- containment and semantic links retain distinct kinds;
- depth applies only to containment traversal;
- node and relationship limits are enforced and reported;
- a failed or superseded evaluation is not a valid replacement for a newer
  active snapshot;
- scene output contains no renderer-specific ELK commands or SVG markup.

### ViewSession

ViewSession is an interaction aggregate for one active viewer journey. It is
ephemeral and need not be persisted.

It owns:

- selected BundleId;
- active ViewProfile identity;
- current FocusRoot;
- depth mode;
- breadcrumb/back history;
- selected ConceptId;
- the active ProjectionSnapshot identity.

It protects:

- focus roots belong to the selected bundle and valid containment;
- Back returns to a prior focus state without changing source or profile;
- a selected concept belongs to the current snapshot;
- stale asynchronous results cannot replace the session's newer request.

The session references BundleIndex, ViewProfile, and ProjectionSnapshot by
identity/revision rather than containing their mutable state.

## Entities

### BundleCandidate

Represents a discovered directory before or after validation. Identity is the
normalized BundleId. It carries path, validation status, and diagnostics.

### ConceptDocument

Represents one source Markdown document within a BundleIndex. Identity is the
bundle-scoped ConceptId, normally its normalized relative document path. It
contains source facts but does not contain profile-derived style or visibility.

### SourceFactSet

Represents preserved frontmatter, Markdown body, links, unknown metadata,
filesystem facts, and provenance for one ConceptDocument. Arbitrary metadata
is retained as typed values where supported and as opaque values otherwise.

### Relationship

Represents a source- or profile-derived connection between two ConceptIds.
RelationshipKind distinguishes containment from semantic link. Provenance
explains the source claim, fallback, or adapter that produced it.

### RuleInvocation

Represents one configured use of a registered RuleStrategy. It carries stable
strategy ID, schema version, parameters, priority, and enabled status.

### PresentationDecision

Represents a renderer-neutral visibility, annotation, label, style, role, or
detail outcome with the rule/profile provenance that produced it.

### Diagnostic

Represents a structured condition affecting bundle, profile, relationship,
projection, navigation, inspection, layout, or persistence. It includes
severity, code/category, human explanation, affected identity, and recovery
guidance where applicable.

## Value objects

- BundleId: normalized repository-relative POSIX path for one .okf directory.
- ConceptId: BundleId plus normalized relative Markdown concept path.
- ProfileId: stable project-local or built-in profile identifier.
- ProfileRevision: monotonic or content-addressed identity for one profile
  definition used by a ProjectionSnapshot.
- SourceRevision: fingerprint or equivalent identity for one BundleIndex
  snapshot.
- FocusRoot: bundle root or ConceptId used as the containment traversal origin.
- DepthLimit: positive integer with at-most semantics, or explicit FullDepth.
- NodeBudget: profile limit plus application hard cap.
- RelationshipBudget: profile limit plus application hard cap.
- LayoutBudget: recommended five-second request budget with cancellation.
- PathBoundary: normalized root and resolution rules for a bundle.
- Provenance: source location, adapter/rule identity, and explanation.
- PresentationToken: validated named style/detail value.
- LegendEntry: mapping between a token/role/state and explanatory text.
- Breadcrumb: prior FocusRoot and depth/profile context.
- DiagnosticCode: stable category/code used for filtering and acceptance tests.
- MarkdownFragment: sanitized detail content plus safe link targets.
- ConfigurationLocation: nearest project configuration path and OKF section
  revision.

## Status vocabularies

### Bundle validity

Discovered, validating, valid, invalid, unavailable, or unreadable.

### Profile validity

Valid, invalid, partially applicable, or unavailable. A profile can remain
stored while being unavailable because a base or rule is missing; the viewer
must diagnose and fall back.

### Projection status

Requested, evaluating, ready, truncated, failed, cancelled, or superseded.

### Concept state

An open source/profile value. Known values may include foggy, bounded,
specified, and implemented, but unknown values remain neutral and profiles
must not pretend this vocabulary is universal.

### Planning state

The separate Arch View map state: foggy, bounded, specified, or implemented.
It is not part of the OKF BundleIndex lifecycle.

## Policies and rule objects

### BundleBoundaryPolicy

Validates project-relative path identity, exclusions, symlink rules, and local
link resolution. It prevents a source bundle from reading outside its boundary.

### HierarchyPrecedencePolicy

Chooses explicit parent/children claims when selected by the profile, then
filesystem fallback for missing parents. It records provenance and emits
diagnostics for self-parenting, cycles, multiple parents, and conflicts.

### RelationshipVisibilityPolicy

Controls independent visibility and presentation of containment and semantic
links. It never converts one kind into the other.

### RuleCompositionPolicy

Runs all applicable invocations, merges compatible collection/annotation
outputs, resolves scalar conflicts by explicit priority, and uses
neutral/default fallback with a diagnostic for equal-priority conflicts.

### ProfileInheritancePolicy

Resolves ordered bases, detects cycles, validates schemas, and preserves
explicit override precedence.

### StateMappingPolicy

Maps arbitrary source metadata to declared/effective state and presentation
tokens. Missing or unknown values remain neutral.

### RollupPolicy

Explicitly marks aggregate/presentation roles and may derive effective state
from declared structural children. Child count alone is never sufficient.

### ScaleSafetyPolicy

Applies depth, node, relationship, and layout budgets. It determines
deterministic truncation and recovery guidance.

### MarkdownSafetyPolicy

Sanitizes supported CommonMark, validates local path resolution, and rejects
unsafe HTML, scripts, schemes, and path escapes.

### ConfigurationWritePolicy

Validates a complete new configuration before atomic replacement and preserves
the prior valid file if validation or writing fails.

## Domain services

### BundleDiscoveryService

Traverses an Arch View project according to BundleBoundaryPolicy and returns a
deterministically ordered BundleCatalog snapshot.

### BundleValidationService

Applies the supported OKF validator to candidates and returns validity and
diagnostics without changing source files.

### BundleIndexingService

Builds an immutable BundleIndex with source facts, extracted links, hierarchy
claims, normalized relationships, and provenance.

### HierarchyNormalizationService

Combines profile-selected explicit claims with filesystem fallback, validates
the containment forest, and emits conflict diagnostics.

### ProfileResolutionService

Resolves built-in/project-local profile composition and registered strategies
into a validated effective profile.

### ProjectionService

Evaluates the effective profile against a BundleIndex and creates a
ProjectionSnapshot under ScaleSafetyPolicy and RuleCompositionPolicy.

### DetailRenderingService

Builds ConceptDetail read data under MarkdownSafetyPolicy and includes source,
mapped metadata, provenance, and diagnostics.

### ConfigurationPersistenceService

Applies Save, Save As, rename, and delete operations to
OKFViewConfiguration under ConfigurationWritePolicy.

These services coordinate aggregates; they do not become a generic “manager”
with ownership of every rule.

## Invariants

### Source and identity invariants

1. Every BundleId is unique within a project and identifies exactly one bundle
   root.
2. Every ConceptId belongs to exactly one BundleId.
3. A projection references one BundleId and one SourceRevision.
4. Source facts and OKF files are never mutated by viewing or profile changes.
5. Local link resolution never crosses the PathBoundary.

### Relationship invariants

1. Containment is acyclic and has at most one navigable parent per concept.
2. Semantic links cannot expand depth traversal or change focus ancestry.
3. Invalid relationship claims remain diagnosable with provenance.
4. Relationship kind is preserved through indexing, projection, scene, and
   inspection.

### Profile invariants

1. Built-in profiles are immutable.
2. Project-local ProfileId values are unique within OKFViewConfiguration.
3. Composition has no cycles and deterministic ordering.
4. Every effective scalar decision has one value or a neutral/default fallback.
5. Rule evaluation cannot depend on registration order.
6. Profile limits cannot exceed application hard caps.
7. Profiles cannot alter BundleId, ConceptId, SourceRevision, or source facts.

### Projection invariants

1. Every visible node, relationship, annotation, and style is source- or
   profile-explainable.
2. Node and relationship budgets produce visible hidden counts and diagnostics.
3. Projection ordering is deterministic for equivalent inputs.
4. A superseded/cancelled/failed request cannot replace a newer valid session
   snapshot.
5. Renderer-neutral scene data contains no concrete renderer instructions.

### Configuration invariants

1. A valid graph binding points to a valid bundle identity and a resolvable
   profile or explicit fallback.
2. Profile rename updates all bindings atomically.
3. Profile deletion cannot leave an active binding silently dangling.
4. Failed saves preserve the previous valid configuration.
5. Unrelated layout and analysis configuration remains intact.

## Lifecycle rules

### Bundle candidate lifecycle

Discovered -> validating -> valid
                       -> invalid
                       -> unavailable

Only valid candidates may be selected. A refresh may replace a candidate's
status, but it does not rewrite stale configuration references.

### Profile lifecycle

Built-in: available -> immutable

Project-local: draft/editing -> valid
                         -> invalid-but-stored
                 valid -> renamed
                 valid -> deleted-after-reassignment

Save validates before replacing the stored definition. Save As creates a new
project-local identity. Deletion requires an explicit replacement or neutral
fallback for every binding.

### Projection lifecycle

Requested -> evaluating -> ready
                       -> truncated
                       -> failed
                       -> cancelled
                       -> superseded

Only ready or truncated snapshots may be displayed as current results;
truncated snapshots must visibly explain what is hidden.

## Domain events

- BundleCatalogRefreshed: discovery/validation produced a new candidate set.
- BundleSelected: one BundleId became the active source.
- BundleIndexBuilt: an immutable source snapshot became available.
- ProfileResolved: a profile composition became valid for evaluation.
- ProfileApplied: a profile was evaluated against a selected source.
- ProjectionBuilt: a ready or truncated renderer-neutral snapshot exists.
- ProjectionRejected: profile, source, or policy prevented projection.
- FocusRootChanged: navigation moved to a new subtree root.
- ConceptInspected: a concept detail was requested for the active snapshot.
- ProfileSaved: a project-local profile was updated successfully.
- ProfileSavedAs: a new project-local profile was created.
- ProfileRenamed: a project-local profile and its bindings were renamed.
- ProfileDeleted: a profile was removed after safe reassignment/fallback.
- ConfigurationSaveFailed: a proposed configuration remained unapplied.

Events carry stable identities and revisions, not mutable aggregate instances.

## Cross-aggregate references and consistency

- BundleCatalog references BundleId; it does not contain BundleIndex content.
- OKFViewConfiguration references BundleId and ProfileId; it does not own source
  facts or projection results.
- ProjectionSnapshot references BundleId/SourceRevision and
  ProfileId/ProfileRevision; it stores derived results immutably.
- ViewSession references the current aggregate identities and owns only
  navigation consistency.
- A profile save, rename, or delete is consistent within
  OKFViewConfiguration. Reprojection is a subsequent operation and may fail
  independently with diagnostics.
- Bundle discovery/indexing and profile resolution are separate operations.
  Missing source or profile references are diagnosable fallback conditions, not
  reasons to corrupt the other aggregate.

## Read-model expectations

The domain must support read projections for:

1. Bundle selector: valid bundles, invalid candidates, stable IDs, diagnostics.
2. Profile selector: built-ins, project-local profiles, validity, bindings.
3. Profile editor: effective composition, overrides, registry metadata,
   validation conflicts, Save/Save As affordances.
4. Graph scene: renderer-neutral nodes, relationships, tokens, legend,
   diagnostics, visible/hidden counts, and layout input.
5. Concept detail: overview, sanitized Markdown, mapped metadata, source path,
   containment, semantic links, provenance, and diagnostics.
6. Navigation history: current focus, breadcrumbs, depth, full mode, recovery
   paths.
7. Diagnostic view: structured filtering by severity, category, source,
   profile, operation, and recovery guidance.

These read models may be materialized, streamed, or computed on demand.

## Extension points

The canonical extension seams are:

- RuleStrategy: metadata, tags/collections, strings/paths, numeric,
  relationships, hierarchy/depth, and derived-fact evaluators.
- RelationshipAdapter: maps additional source conventions into typed
  relationship facts without changing the core index.
- PresentationPropertyProvider: contributes validated renderer-neutral style or
  detail properties.
- ShapeProvider: contributes validated shape tokens understood by the scene
  contract and renderer adapters.
- DetailRenderer: contributes safe detail formats under the Markdown/security
  boundary.
- DiagnosticProvider: contributes structured explanations and recovery
  guidance.

Each extension has a namespaced stable ID, versioned schema, declared
capabilities, human-readable metadata, validation, and deterministic behavior.
The profile contains data; registered in-process Go implementations contain
executable behavior. External repository code is not an extension source.

## Minimum canonical scenarios

1. Discover several valid bundles and select exactly one.
2. Diagnose an invalid bundle without making it selectable.
3. Index arbitrary frontmatter and preserve unknown fields.
4. Apply neutral and fog-of-war profiles to the same index.
5. Resolve explicit hierarchy, filesystem fallback, and conflicts.
6. Show containment and semantic links independently.
7. Apply composed rules with priority and equal-priority diagnostics.
8. Render unknown state neutrally and distinguish declared/effective state.
9. Enforce depth and scale budgets with deterministic hidden counts.
10. Focus a subtree and navigate back.
11. Inspect sanitized Markdown and reject unsafe links/content.
12. Save a project profile, Save As a variant, rename it, and delete it after
    safe reassignment.
13. Retain the last valid scene after a later layout failure or cancellation.

## Mapping guidance for architecture variants

Any architecture may implement the model if it preserves:

- immutable source snapshots and explicit source/profile revisions;
- one-bundle projection boundaries;
- deterministic discovery, normalization, rule composition, and truncation;
- explicit aggregate ownership of configuration and projection consistency;
- renderer-neutral scene output before ELK or another layout engine;
- structured diagnostics and read models;
- safe Markdown and path handling;
- registered extension strategies without profile-supplied executable code.

A layered implementation may map aggregates to domain modules and services to
application orchestration. A transaction-script implementation may enforce the
same invariants in command handlers. An event-driven implementation may use
the domain events for cache/read-model updates. None may collapse source facts
into presentation state or make ELK-specific geometry part of the canonical
domain model.
