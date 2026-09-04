# Configurable OKF knowledge views — canonical use-case model

## Purpose

This document defines the stable application-layer intents for the OKF viewer.
It sits between the canonical domain model and any HTTP, embedded-web, CLI, or
other external interface. It describes orchestration and outcomes, not routes,
payload serialization, controllers, or package layout.

## Application-layer goals

- expose stable user intents for discovering, selecting, projecting, inspecting,
  navigating, and configuring OKF views;
- keep command/state-changing operations distinct from read queries;
- coordinate BundleCatalog, BundleIndex, ViewProfile,
  OKFViewConfiguration, ProjectionSnapshot, and ViewSession without leaking
  their internal representations;
- preserve deterministic, source-backed, safe outcomes across retries and
  implementations;
- make failures and diagnostics first-class application results;
- allow the later external contract to map the same behavior to multiple
  transports.

## Design principles

1. Use intent-based names rather than screen or CRUD names.
2. Queries do not mutate source documents or project configuration.
3. Commands that persist profile/configuration changes validate before commit.
4. Domain policies own semantic invariants; application services coordinate
   aggregates and operation lifecycles.
5. Every projection result identifies its source and profile revisions.
6. Cancellation and supersession are explicit outcomes for long-running reads.
7. Retry-sensitive commands accept a caller operation ID/idempotency key.
8. External interfaces may reshape data, but may not change command/query
   meaning or hide diagnostics.

## Application boundaries

### Bundle viewing service

Coordinates project discovery, bundle selection, indexing, profile resolution,
projection, and session state. It depends on BundleDiscoveryService,
BundleValidationService, BundleIndexingService, ProfileResolutionService, and
ProjectionService.

### Navigation service

Coordinates depth changes, subtree focus, breadcrumb/back behavior, and
superseded projection requests. It depends on the active ViewSession and
ProjectionService but does not own profile or source mutation.

### Concept inspection service

Provides source-backed concept details, safe Markdown, relationship
provenance, mapped metadata, and diagnostics for the active snapshot.

### Profile configuration service

Coordinates profile validation, graph/profile bindings, Save, Save As,
rename, delete, revision conflicts, and atomic configuration persistence.
It depends on ConfigurationPersistenceService and the RuleRegistry/catalog.

### Diagnostic and registry catalog service

Exposes structured diagnostics and metadata about available rule strategies,
relationship adapters, presentation properties, shapes, and detail renderers.
It does not evaluate a profile by itself; it supports profile editing and
explanation.

These are application-service boundaries, not required deployment processes.

## Canonical commands

### RefreshBundleCatalog

**Intent:** Re-scan the project for eligible OKF bundles and refresh the
current discovery result.

**Inputs:**

- project identity/root;
- optional refresh request ID;
- cancellation token.

**Application responsibilities:**

1. Apply project boundary/exclusion policy.
2. Validate discovered candidates.
3. Build a deterministic BundleCatalog snapshot.
4. Preserve stale configuration references for later diagnostics.
5. Publish BundleCatalogRefreshed.

**Outcomes:**

- catalog with valid and invalid candidates plus diagnostics;
- empty catalog with diagnostics when no valid bundle exists;
- cancelled refresh that does not replace a newer catalog.

This command is repeatable for the same project snapshot. It does not persist
graph/profile settings.

### SelectBundle

**Intent:** Make one valid BundleId the source for the current viewer session.

**Inputs:**

- session identity;
- BundleId;
- optional profile selection;
- optional request ID/cancellation token.

**Preconditions:**

- the BundleId is present and valid in the current BundleCatalog;
- the bundle can be indexed within its source boundary.

**Application responsibilities:**

1. Load or reuse an immutable BundleIndex.
2. Resolve the saved binding or selected profile, falling back to the neutral
   built-in profile with diagnostics when necessary.
3. Set the session's selected BundleId.
4. Request an initial projection.

**Outcomes:**

- selected bundle and ready/truncated ProjectionSnapshot;
- bundle/profile diagnostics with no misleading partial scene;
- selection rejected when the candidate is invalid or unavailable.

Selection is session state. Persisting the default graph or binding is a
separate configuration command.

### SelectProfile

**Intent:** Apply one built-in or project-local ViewProfile to the selected
bundle for the current session.

**Inputs:**

- session identity;
- ProfileId;
- optional request ID/cancellation token.

**Preconditions:**

- a BundleId is selected;
- the profile is resolvable or can produce a diagnosed fallback.

**Application responsibilities:**

1. Resolve ordered bases and registered rule strategies.
2. Validate composition, limits, tokens, and profile parameters.
3. Set the session's active profile identity.
4. Request a new projection from the same BundleIndex.

**Outcomes:**

- effective profile metadata and ready/truncated ProjectionSnapshot;
- invalid-layer diagnostics with nearest valid fallback;
- projection failure/cancellation while retaining the prior snapshot when safe.

SelectProfile does not persist edits or change the graph/profile binding.

### SetNavigationDepth

**Intent:** Change the active session's containment depth policy.

**Inputs:**

- session identity;
- positive integer depth or explicit full mode;
- optional request ID/cancellation token.

**Application responsibilities:**

1. Validate at-least-1 integer or explicit full mode.
2. Keep the current BundleId, ProfileId, and FocusRoot.
3. Request a replacement projection under profile and application budgets.
4. Update the session only when the request is the current request.

**Outcomes:**

- updated ready/truncated snapshot;
- invalid depth rejection without session mutation;
- cancellation/supersession with the previous valid session snapshot retained.

### FocusSubtree

**Intent:** Make a selected ConceptId the current containment focus root.

**Inputs:**

- session identity;
- ConceptId;
- optional request ID/cancellation token.

**Preconditions:**

- the concept belongs to the selected BundleId;
- the concept is a valid containment descendant or current focus target.

**Application responsibilities:**

1. Push the previous navigation context onto breadcrumb history.
2. Set the new FocusRoot.
3. Preserve active BundleId, ProfileId, and depth policy.
4. Request a replacement projection from the subtree.

**Outcomes:**

- focused ready/truncated snapshot with updated navigation history;
- rejected focus with a diagnostic when the concept is not in the current
  bundle or is not navigably contained;
- cancellation/supersession without committing an incomplete focus.

### NavigateBack

**Intent:** Return to the previous focus context.

**Inputs:**

- session identity;
- optional request ID/cancellation token.

**Preconditions:**

- breadcrumb history is non-empty.

**Application responsibilities:**

1. Pop the previous FocusRoot and navigation settings.
2. Request a replacement projection.
3. Preserve history if the request is cancelled or fails.

**Outcomes:**

- prior focus ready/truncated snapshot;
- empty-history rejection with no mutation;
- diagnostic failure with the last valid focus retained.

### BindProfileToBundle

**Intent:** Persist the selected project-local profile as the preferred
profile for one bundle, optionally setting the project default bundle.

**Inputs:**

- configuration location/revision;
- BundleId;
- optional ProfileId;
- optional make-default flag;
- caller operation ID/idempotency key.

**Preconditions:**

- the BundleId is a valid or intentionally stale reference being repaired;
- ProfileId resolves to a built-in or project-local profile, or the explicit
  neutral fallback is requested.

**Application responsibilities:**

1. Validate the proposed binding.
2. Update OKFViewConfiguration as one aggregate operation.
3. Atomically persist the complete validated configuration.

**Outcomes:**

- new configuration revision and binding;
- stale-reference diagnostic with configuration preserved;
- validation, revision-conflict, or persistence failure with prior file intact.

Repeating the same operation ID and content returns the original result without
duplicating or reordering configuration.

### SaveProjectProfile

**Intent:** Persist edits to an existing project-local profile.

**Inputs:**

- configuration location/revision;
- project-local ProfileId;
- complete proposed profile definition;
- caller operation ID/idempotency key.

**Preconditions:**

- the profile is project-local and editable;
- the proposed definition validates against registered schemas, composition
  rules, presentation tokens, and safety limits.

**Application responsibilities:**

1. Validate the profile and its bases.
2. Enforce optimistic configuration/profile revision checks.
3. Replace the profile within OKFViewConfiguration.
4. Atomically persist configuration.
5. Return the updated profile revision and diagnostics.

**Outcomes:**

- saved project-local profile;
- validation failure with field/rule diagnostics;
- revision conflict requiring reload/merge;
- persistence failure with prior configuration intact.

### SaveProjectProfileAs

**Intent:** Create a new project-local profile from the current composition and
overrides.

**Inputs:**

- configuration location/revision;
- new ProfileId/display name;
- source profile composition and current overrides;
- caller operation ID/idempotency key.

**Preconditions:**

- the new ProfileId is unique;
- the copied composition and overrides validate;
- the source profile may be built-in or project-local, but the new profile is
  project-local.

**Application responsibilities:**

1. Preserve ordered base references rather than flattening them.
2. Create the new profile as one configuration change.
3. Make the new profile the active profile for the current session.
4. Persist the current bundle binding to the new profile only when the caller
   requests binding persistence; otherwise leave the stored binding unchanged.

**Outcomes:**

- new project-local ProfileId and revision;
- duplicate-ID, validation, revision-conflict, or persistence failure;
- repeated operation ID returns the original created profile.

The session activation is immediate; binding persistence is explicit so Save As
does not unexpectedly change another future viewer session.

### RenameProjectProfile

**Intent:** Rename a project-local profile without breaking graph bindings.

**Inputs:**

- configuration location/revision;
- existing ProfileId;
- new ProfileId/display name;
- caller operation ID/idempotency key.

**Application responsibilities:**

1. Validate new identity uniqueness.
2. Update profile identity and every affected binding atomically.
3. Persist the new configuration revision.

**Outcomes:**

- renamed profile and updated bindings;
- duplicate/missing profile, revision-conflict, or persistence failure with no
  partial rename.

### DeleteProjectProfile

**Intent:** Remove a project-local profile after its bindings are made safe.

**Inputs:**

- configuration location/revision;
- ProfileId;
- replacement ProfileId or explicit neutral fallback;
- caller operation ID/idempotency key.

**Preconditions:**

- the profile is project-local;
- every binding that points to it has an explicit replacement or fallback.

**Application responsibilities:**

1. Validate replacement/fallback.
2. Update affected bindings.
3. Remove the profile.
4. Persist the complete configuration atomically.

**Outcomes:**

- deleted profile and valid reassigned bindings;
- deletion rejected because a binding has no replacement;
- revision-conflict or persistence failure with prior configuration intact.

## Canonical queries

### GetBundleCatalog

Returns the current BundleCatalog for a project, including valid candidates,
invalid candidates, stable IDs, selector metadata, and diagnostics. It never
merges bundle contents.

### GetBundleIndexSummary

Returns source-backed summary metadata for one selected BundleId and
SourceRevision: concept count, available roots, source diagnostics, and
revision identity. It does not expose profile-derived visibility as source
truth.

### GetProfileCatalog

Returns built-in and project-local profiles, composition summaries, validity,
registry capabilities, graph bindings, and profile diagnostics for a project.

### ValidateProfile

Evaluates a proposed profile definition without persisting it. Returns
validated effective settings, field/rule conflicts, unsupported features,
inheritance diagnostics, and whether Save or Save As is permitted.

### GetCurrentProjection

Returns the current session's ProjectionSnapshot metadata and
renderer-neutral scene. It includes source/profile revisions, visible and
hidden counts, navigation state, legend, diagnostics, and relationships
separated by kind.

### GetConceptDetail

Returns ConceptDetail for a ConceptId in the current snapshot or BundleIndex.
It includes overview, mapped metadata, sanitized Markdown, safe links,
containment, semantic links, provenance, and relevant diagnostics.

### GetNavigationState

Returns selected bundle/profile, focus root, depth mode, breadcrumb/back
history, selected concept, projection status, and recovery guidance.

### GetDiagnostics

Returns structured diagnostics filtered by project, bundle, profile, concept,
relationship, operation, severity, or category. Diagnostics must preserve
recovery guidance and provenance where available.

### GetExtensionCatalog

Returns namespaced rule strategies, relationship adapters, presentation
properties, shapes, detail renderers, schemas, versions, capabilities, and
human-readable descriptions available in the current Arch View process.

## Conceptual command and query shapes

### Command shape

Every command carries:

- operation ID/idempotency key when it can persist or create state;
- project/session identity as appropriate;
- aggregate identity and expected revision when consistency matters;
- intent-specific input;
- cancellation context for projection/discovery work.

Every command returns:

- success or rejected outcome;
- resulting identity/revision/session state where applicable;
- structured diagnostics;
- events or event metadata when the caller needs to update read models.

### Query shape

Every query carries the smallest stable scope needed to answer it and may
include a source/profile revision or session identity. A query returns a
read-model result with diagnostics, freshness/revision metadata, and explicit
empty/truncated/failed status. Queries do not silently turn a failed
projection into a successful empty graph.

## Transaction and consistency expectations

| Use-case group | Consistency boundary | Retry/cancellation expectation |
|---|---|---|
| RefreshBundleCatalog | one catalog snapshot | repeatable; cancellable; stale refresh cannot replace newer |
| SelectBundle/SelectProfile | one ViewSession transition plus immutable read snapshots | request supersession; no persistence |
| SetNavigationDepth/FocusSubtree/NavigateBack | one ViewSession transition and one projection request | cancellable; failed request leaves prior session result |
| BindProfileToBundle | one OKFViewConfiguration aggregate | atomic write; revision checked; idempotent |
| SaveProjectProfile | one OKFViewConfiguration aggregate | atomic write; revision checked; idempotent |
| SaveProjectProfileAs | one configuration change, optional session activation/binding | idempotent creation; no duplicate profile on retry |
| RenameProjectProfile | one configuration aggregate including all bindings | atomic; no partial rename; idempotent |
| DeleteProjectProfile | one configuration aggregate including reassignment | atomic; explicit replacement/fallback; idempotent |
| Get* queries | read snapshot appropriate to query | repeatable for a supplied revision; cancellable where expensive |

Discovery/indexing and configuration persistence are separate consistency
boundaries. A source change may invalidate a cached index or projection, but it
does not partially rewrite profile configuration.

## Cross-cutting application concerns

### Revision and freshness

Source, profile, configuration, catalog, session, and projection results carry
revision/freshness metadata. A caller can distinguish an older valid snapshot
from the current requested operation.

### Idempotency

Persisting commands use caller operation IDs. A repeated operation with the
same ID and equivalent content returns the original outcome. Reuse of an ID
with different content is rejected as an idempotency conflict.

### Cancellation and supersession

Projection and discovery requests carry cancellation context. When a newer
request supersedes an older request, the older request may finish internally
but cannot publish a result into the active session.

### Diagnostics

Application outcomes preserve diagnostic category, severity, affected
identity, source/profile provenance, and recovery guidance. An error is not
reduced to a transport status or log line.

### Authorization and safety

The local host enforces project/bundle path boundaries, read-only source
consumption, safe Markdown handling, profile schema validation, application
hard caps, and no execution of target-provided rule code.

## Failure model

### Validation failure

Input bundle, profile, rule, token, depth, limit, binding, or link is invalid.
Return structured diagnostics and leave the previous valid aggregate/session
state intact.

### Missing or unavailable resource

Bundle, profile, source concept, or base reference is missing/unavailable.
Preserve stale configuration where required, select neutral fallback where
defined, and expose the missing identity.

### Consistency conflict

Configuration revision, profile revision, or idempotency key conflicts with
the caller's proposed operation. Do not overwrite; return current revision and
recovery guidance.

### Projection policy outcome

The request is valid but truncated by depth/node/relationship budgets. Return a
displayable truncated snapshot with hidden counts and recovery guidance.

### Processing failure

Indexing, rule evaluation, relationship normalization, ELK layout, timeout, or
Markdown processing fails. Preserve the last valid scene where safe; on first
load return a diagnostic empty state rather than misleading partial geometry.

### Cancellation or supersession

The caller or a newer request stops work. Return a non-current outcome and do
not replace the active valid session result.

### Persistence failure

Configuration validation or atomic write fails. Preserve the prior file and
return a diagnostic that distinguishes validation from filesystem failure.

## Domain events at the application boundary

Application services may publish or hand off:

- BundleCatalogRefreshed;
- BundleSelected;
- BundleIndexBuilt;
- ProfileResolved;
- ProjectionBuilt or ProjectionRejected;
- FocusRootChanged;
- ConceptInspected;
- ProfileSaved, ProfileSavedAs, ProfileRenamed, or ProfileDeleted;
- ConfigurationSaveFailed.

Events identify project, bundle, profile, session, operation, and revisions.
They do not expose mutable aggregate instances or renderer-specific geometry.

## End-to-end use-case chains

### Chain A: Open and inspect

RefreshBundleCatalog -> GetBundleCatalog -> SelectBundle ->
GetCurrentProjection -> GetConceptDetail.

### Chain B: Compare rendering costumes

GetProfileCatalog -> SelectProfile(neutral) -> GetCurrentProjection ->
SelectProfile(fog-of-war) -> GetCurrentProjection -> GetConceptDetail.

### Chain C: Explore a large graph

SelectBundle -> SetNavigationDepth(2) -> FocusSubtree ->
GetNavigationState -> NavigateBack.

### Chain D: Create and use a project profile

GetProfileCatalog -> ValidateProfile -> SaveProjectProfileAs ->
SelectProfile -> BindProfileToBundle (when persistence is requested) ->
GetCurrentProjection.

### Chain E: Recover from bad data or processing

GetBundleCatalog reports invalid candidate -> select a valid bundle ->
GetCurrentProjection reports hierarchy/rule/truncation diagnostics ->
reduce depth or FocusSubtree -> inspect diagnostics and source detail.

## Mapping guidance for external interfaces

An HTTP, embedded-web, CLI, or other adapter may expose these use cases using
its own naming and serialization, but it must preserve:

- command/query distinction;
- operation IDs, revisions, and cancellation behavior;
- one-bundle source scope;
- explicit profile and navigation identities;
- structured diagnostics and truncated/failed outcomes;
- Save versus Save As semantics;
- safe local-link and Markdown outcomes.

Transport adapters must not expose profile-supplied executable code, convert
diagnostics into silent omission, or make renderer-specific ELK/SVG details
part of the canonical application surface.
