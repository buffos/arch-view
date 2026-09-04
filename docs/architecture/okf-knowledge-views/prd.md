# Configurable OKF knowledge views — product requirements

## Purpose

Arch View already helps developers understand source code through an
architecture model and interactive diagrams. This capability adds a separate
first-class way to explore arbitrary Open Knowledge Format (OKF) bundles that
developers or knowledge workers maintain in the same project.

This document defines the product behavior for the OKF viewer. It turns the
bounded discovery baseline into a stable reference for domain modeling,
application-service modeling, external contracts, acceptance scenarios, and
implementation planning.

## Product summary

The OKF viewer discovers validated .okf bundles in an Arch View project and
lets the viewer user select one bundle at a time. It reads the selected bundle
without changing it, applies a reusable view profile, and presents a
depth-bounded or full interactive projection through the shared Arch View
viewer and ELK-backed layout path.

A profile is a rendering costume: it determines how source facts become
containment, semantic links, facets, roles, presentation tokens, details, and
navigation behavior. The source bundle remains independent of the profile and
of every other discovered bundle. The viewer never merges bundles or silently
changes source documents.

## Product goals

1. Let a user discover and switch among multiple independent OKF bundles in one
   project.
2. Preserve enough source information to explain where every displayed concept,
   relationship, metadata value, and diagnostic came from.
3. Support reusable project-local profiles that can express different ways of
   understanding the same bundle, including a neutral profile and a
   fog-of-war presentation.
4. Make structural navigation useful for both small graphs and large,
   fast-branching graphs through configurable maximum depth and subtree focus.
5. Keep hierarchy, semantic links, source state, effective state, and
   presentation roles distinct and explainable.
6. Allow the presentation vocabulary and rule families to grow substantially
   without changing the source format or adding a central conditional branch
   for every new rule.
7. Give users useful, safe concept inspection with visible explanations for
   conflicts, invalid data, unsupported profile features, and truncation.
8. Reuse the existing renderer-neutral viewer and ELK layout infrastructure
   without making OKF source data part of the architecture model.

## Non-goals for this capability

- Merging multiple OKF bundles into one graph.
- Replacing the architecture model, source analyzers, or architecture viewer.
- Requiring every OKF author to use Arch View's planning vocabulary.
- Editing, normalizing, or otherwise mutating OKF source documents.
- Executing arbitrary JavaScript, expressions, or code supplied by a bundle or
  profile.
- Loading external rule plugins from a target repository in the first slice.
- Providing a public OKF CLI, remote collaboration service, export format, or
  source synchronization workflow in the first slice.
- Inferring that a concept is a roll-up merely because it has children.
- Treating a profile as a second source of graph identity or source content.

## Users and actors

### Viewer user

Opens an Arch View project, selects a bundle and profile, explores the
projection, focuses subtrees, and inspects concepts.

### Bundle maintainer

Creates or updates OKF documents and expects Arch View to consume them
read-only. The bundle maintainer may also be the viewer user.

### Profile maintainer

Creates, edits, saves, renames, and deletes project-local rendering profiles.
The profile maintainer may be the viewer user or a different project
collaborator with access to the local configuration.

## Domain scope

The capability owns these conceptual areas:

- project-local discovery and validation of independent OKF bundles;
- lossless source indexing for concept documents;
- profile selection, composition, validation, and lifecycle;
- structural containment and semantic-link projection;
- declarative rule evaluation and renderer-neutral presentation annotations;
- depth, subtree focus, scale protection, and truncation reporting;
- concept detail inspection and safe Markdown rendering;
- project-local graph/profile bindings and OKF viewer preferences.

The architecture model, source analysis, generic ELK renderer contract, and
headless export remain owned by their existing capabilities.

## Core business capabilities

### Discover and select one bundle

The viewer scans the project tree recursively for directories named .okf,
applies configured dependency/output exclusions, validates candidates, and
offers the valid bundles in deterministic path order. Each bundle is an
independent graph source and is selected as a whole.

### Build a lossless index

The reader indexes concept documents while preserving normalized path,
frontmatter, Markdown body, inline links, unknown metadata, and provenance.
The index is read-only and is the input to every profile.

### Apply a rendering profile

The viewer applies a built-in or project-local profile to the selected index.
The profile chooses hierarchy mappings, semantic-link visibility, metadata
facets, state and role mappings, presentation tokens, detail behavior,
navigation limits, and ELK layout preferences.

### Explore and focus

The initial view starts at the bundle root with the active depth policy. A
viewer user may change the depth, use full mode, inspect a concept, and
double-click a concept to make it the new focus root. Breadcrumbs or Back
navigation return to earlier focus roots.

### Inspect safely

Single-click inspection leads with title, description, type, source path,
mapped facets, sanitized Markdown, and effective presentation information.
Frontmatter, containment, semantic links, provenance, and technical
diagnostics are available as expandable secondary information.

### Maintain project-local profiles

The viewer can edit project-local profiles and persist them in the nearest
.archview.json. Built-ins cannot be overwritten. Save updates the active
project-local profile; Save As creates a new profile while retaining its
composition references and overrides. Rename and delete operations preserve
binding integrity.

## Required business rules and invariants

### Bundle and source rules

1. A bundle is identified by its normalized repository-relative POSIX path.
2. The bundle directory is the source and path-resolution boundary.
3. A bundle must pass supported OKF validation before it is selectable.
4. Invalid, unavailable, or unreadable candidates remain reportable through
   diagnostics and never appear as silently valid graphs.
5. The selected bundle is the only source for the current projection. The
   viewer never merges bundles.
6. Source facts are preserved even when a profile hides the corresponding
   projected node or relationship.
7. Arch View may persist its own configuration but never mutates OKF files.

### Containment and relationship rules

1. Containment is the only relationship class used for depth traversal,
   breadcrumbs, and subtree focus.
2. Semantic links are separate and may be independently hidden, styled, or
   displayed.
3. A selected profile may map explicit parent/children metadata to
   containment.
4. Filesystem nesting supplies a missing parent only when no selected explicit
   hierarchy claim supplies one.
5. Explicit claims, fallback decisions, and conflicts retain provenance.
6. A self-parent, containment cycle, or multiple-parent conflict is diagnosed
   and excluded from navigable containment rather than silently repaired.
7. A concept affected by invalid containment remains available in the
   lossless index and diagnostic view.
8. Child count alone never assigns the roll-up role.
9. Local Markdown links resolve only inside the selected bundle; unresolved
   links remain visible as diagnostics.

### Profile and rule rules

1. A profile is presentation and projection configuration, not source data.
2. Built-in profiles are immutable. Project-local profiles are mutable.
3. A profile may compose multiple ordered bases. Cycles and invalid bases are
   diagnosed.
4. Profile composition has deterministic precedence independent of registry
   registration order.
5. All matching rule invocations run. Compatible collection and annotation
   outputs merge deterministically.
6. Explicit priority resolves incompatible scalar values. Equal-priority
   conflicts produce diagnostics and use the neutral/default value for that
   field.
7. Missing rule IDs, unsupported schemas, and invalid parameters invalidate
   only the affected rule/profile layer and fall back to the nearest valid
   composed profile where possible.
8. Rules produce renderer-neutral facts, visibility decisions, annotations, and
   style patches. They cannot emit ELK commands, SVG markup, or arbitrary code.
9. Rule strategies, relationship adapters, style properties, shapes, and
   detail renderers have namespaced stable IDs, versioned schemas, declared
   capabilities, and human-readable metadata.
10. A profile may map arbitrary preserved frontmatter, but unknown fields remain
    available as source metadata and are not guessed into universal semantics.

### State and presentation rules

1. Declared state is source or explicitly mapped metadata.
2. Effective state is the result of the selected profile's mapping or explicit
   roll-up policy.
3. Missing or unrecognized state is neutral/unknown; it is not automatically
   treated as foggy.
4. A profile may present declared and effective state separately.
5. Visual outcomes use validated named presentation tokens and a finite
   built-in shape/color vocabulary. The active profile may expose a legend.
6. A fog-of-war profile is an optional presentation convention, not an OKF
   requirement.

### Navigation and scale rules

1. The default depth is 2.
2. A numeric depth must be an integer of at least 1 and means “at most” that
   many containment levels.
3. Full/unbounded mode is explicit and remains subject to safety limits.
4. Profile max_nodes defaults to 1000 and cannot exceed the application hard
   cap of 3000 in the first implementation baseline.
5. Profile max_relationships defaults to 10000 and cannot exceed the
   application hard cap of 30000 in the first implementation baseline.
6. Semantic links never expand the containment traversal frontier.
7. Node and relationship selection is deterministic by normalized path order.
8. Reaching a limit produces visible truncation diagnostics and hidden counts;
   omission is never silent.
9. A user can reduce depth or focus a hidden subtree to recover detail.
10. Projection and layout work is cancellable. A superseded request cannot
    replace the current active view.

### Inspection and security rules

1. Markdown detail uses a sanitized CommonMark subset: headings, emphasis,
   lists, code, and safe HTTP/HTTPS/mail links.
2. Raw HTML, scripts, unsafe URL schemes, and path escapes are removed or
   rejected.
3. Local concept links stay within the selected bundle and may navigate to the
   target concept or a document fragment.
4. External safe links open in a new context with appropriate browser safety
   behavior.
5. Raw Markdown is available as secondary source information.
6. Unknown frontmatter is collapsed by default and can be expanded.
7. Diagnostics identify invalid, unresolved, unsupported, conflicting,
   truncated, timed-out, or failed processing.

## Required workflows

### WF-01: Discover and open a bundle

1. The viewer user opens an Arch View project.
2. The local host recursively finds eligible .okf directories, applies
   exclusions, and validates candidates.
3. The viewer shows valid bundles in stable path order and reports invalid
   candidates separately.
4. The viewer user selects one bundle.
5. The host indexes only that bundle and selects the saved profile binding or
   the neutral built-in profile.
6. The viewer shows the initial projection at the active depth.

If no valid bundle exists, the viewer shows an empty state with discovery and
validation diagnostics. If a saved graph or profile is missing, the viewer
preserves the stale configuration, reports the problem, and falls back without
silently rewriting the file.

### WF-02: Change profile and project the graph

1. The viewer user selects a profile from built-in and project-local profiles.
2. The viewer validates profile composition and rule invocations.
3. The host projects the selected bundle into separate containment and
   semantic-link layers.
4. The viewer updates styles, labels, facets, details, legend, and layout
   according to the profile.
5. Any invalid layer, rule conflict, unsupported feature, or truncation is
   shown as a diagnostic without changing source data.

### WF-03: Navigate depth and focus a subtree

1. The viewer user chooses a numeric depth or explicit full mode.
2. The viewer rebuilds the projection from the current focus root.
3. If a concept is double-clicked, it becomes the focus root and breadcrumbs or
   Back expose the previous focus.
4. The viewer preserves the selected bundle and active profile while changing
   focus.
5. Hidden counts and recovery diagnostics remain available when limits prevent
   the full requested projection.

### WF-04: Inspect a concept

1. The viewer user single-clicks a projected node.
2. The viewer opens concept detail with title, description, type, source path,
   mapped facets, effective presentation, and sanitized Markdown.
3. The user may expand source frontmatter, containment, semantic links,
   provenance, and diagnostics.
4. A local concept link navigates within the selected bundle; an unresolved
   link remains a diagnostic; an external safe link opens separately.

### WF-05: Edit and persist a profile

1. The profile maintainer selects a project-local profile and changes profile
   settings in the viewer.
2. The viewer validates the composed profile and shows conflicts or invalid
   values before saving.
3. Save updates the active project-local definition.
4. Save As creates a new project-local definition with a new ID while
   preserving ordered base references and current overrides.
5. Renaming a project-local profile updates graph bindings atomically.
6. Deleting a project-local profile requires reassignment or explicit neutral
   fallback before bindings can be left valid.
7. Built-in profiles offer Save As but never direct overwrite.
8. A failed validation or write leaves the previous configuration intact and
   reports the failure.

### WF-06: Handle invalid or interrupted processing

1. The host reports invalid bundle data, hierarchy conflicts, missing profiles,
   unsupported rules, truncation, ELK failure, timeout, or cancellation as
   structured diagnostics.
2. A failed or superseded projection cannot replace a newer valid projection.
3. If a previous scene exists, layout failure retains it when safe to do so.
4. On first load with no valid scene, the viewer shows a diagnostic empty state
   rather than misleading partial geometry.

## Functional requirements

### Discovery and indexing

- FR-01: Discover eligible .okf directories recursively within the project
  root.
- FR-02: Skip .git, dependency directories, generated/output directories,
  configured exclusions, and symlink targets outside the project root.
- FR-03: Validate candidates before selectable status and expose invalid
  candidates through diagnostics.
- FR-04: Assign stable bundle IDs from normalized repository-relative POSIX
  paths and order selector entries deterministically.
- FR-05: Index one selected bundle losslessly, preserving unknown metadata and
  provenance.
- FR-06: Do not read across bundle boundaries when resolving local links.

### Profiles and projection

- FR-07: Offer a neutral built-in profile for every valid bundle and support an
  optional fog-of-war example profile.
- FR-08: Load the graph/profile binding from the nearest .archview.json when
  valid and provide a neutral fallback when it is not.
- FR-09: Support project-local profiles with ordered bases, declarative rule
  invocations, projection settings, presentation tokens, detail settings,
  navigation settings, and per-profile ELK layout options.
- FR-10: Project containment and semantic links as independently configurable
  relationship layers.
- FR-11: Preserve and display hierarchy/relationship provenance and
  normalization diagnostics.
- FR-12: Apply compositional registered rules with deterministic priority and
  conflict behavior.
- FR-13: Produce renderer-neutral scene data that the existing ELK-backed
  viewer can lay out and render.

### Navigation and inspection

- FR-14: Support default depth 2, integer depth values at least 1, and
  explicit full mode.
- FR-15: Support subtree focus through double-click and breadcrumb/Back
  navigation.
- FR-16: Enforce profile and application node/relationship limits with visible
  hidden counts and recovery guidance.
- FR-17: Show concept detail on single click with overview, mapped metadata,
  sanitized Markdown, and expandable source/diagnostic information.
- FR-18: Navigate safe local concept links inside the selected bundle and
  handle unresolved/external links according to the security rules.

### Profile lifecycle

- FR-19: Edit and validate project-local profiles in the viewer.
- FR-20: Save changes to the active project-local profile.
- FR-21: Save As to a new project-local profile without flattening composition.
- FR-22: Keep built-in profiles immutable.
- FR-23: Rename and delete project-local profiles without leaving invalid graph
  bindings.
- FR-24: Persist the optional okf section without corrupting unrelated
  configuration sections or losing the previous valid file on failure.

### Diagnostics and resilience

- FR-25: Represent bundle, profile, hierarchy, rule, link, truncation, layout,
  timeout, cancellation, and persistence failures as structured diagnostics.
- FR-26: Keep diagnostics associated with the relevant bundle, concept,
  relationship, profile, or operation where possible.
- FR-27: Prevent stale asynchronous requests from replacing the active view.

## Non-functional requirements

### Determinism

Given the same bundle contents, profile definitions, registry versions, and
navigation settings, discovery order, projection ordering, diagnostics, and
layout inputs must be repeatable. Registration order must not alter rule
outcomes.

### Safety

The viewer must not execute source-bundle or profile-provided code, cross the
bundle/path boundary, render unsafe HTML or URL schemes, or mutate OKF source
files. Configuration writes must validate before replacement.

### Explainability

Every derived role, state, relationship, style, hidden count, and diagnostic
should be explainable through source provenance, profile mapping, rule
invocation, and/or normalization policy.

### Extensibility

The rule and presentation registry must support growth from the initial
built-ins to hundreds or thousands of independently registered rule families,
properties, shapes, relationship adapters, and detail renderers without
changing the profile data model's central evaluator branch.

### Responsiveness and bounded work

Projection and layout must honor cancellation and the configured limits. The
first baseline recommends a five-second layout budget per projection request;
exceeding it produces a diagnostic and preserves the last valid scene when
possible.

### Compatibility

The optional OKF configuration section must coexist with existing layout and
analysis settings in .archview.json. Existing architecture-view behavior
must remain unchanged when no OKF viewer is selected.

### Accessibility

The OKF viewer must preserve the existing viewer's keyboard navigation, focus
visibility, readable labels, accessible graph/list alternatives, and usable
Back/breadcrumb interaction.

## Acceptance scenarios

Detailed reusable scenarios are maintained in acceptance-scenarios.md. The
core product outcomes are:

- a project with several valid bundles shows each independently and never as a
  merged graph;
- an invalid bundle is diagnosed and cannot silently become selectable;
- the same bundle renders differently under two profiles without source
  changes;
- depth 1, depth 2, a larger depth, and full mode have the defined at-most
  behavior and remain safety-limited;
- double-click focus and Back navigation form a recoverable subtree journey;
- explicit hierarchy, filesystem fallback, semantic links, and conflicts are
  visibly distinguishable;
- missing/unknown state remains neutral while declared/effective state mapping
  is explainable;
- a built-in profile cannot be overwritten, while Save and Save As have their
  distinct project-local outcomes;
- equal-priority rule conflicts, unsupported rules, stale bindings, failed
  writes, invalid Markdown links, and layout cancellation produce diagnostics;
- unsafe Markdown content cannot execute or escape the selected bundle;
- repeated equivalent inputs produce deterministic projection inputs and
  outcomes.

## Success measures

The first slice is successful when a viewer user can:

1. open a project containing multiple valid OKF bundles and select each one;
2. apply at least the neutral profile and a fog-of-war profile;
3. see hierarchy and semantic links as separate configurable layers;
4. inspect source-backed concept details and diagnostics;
5. limit depth, focus a subtree, and navigate back;
6. create or edit a project-local profile, Save it, and Save As a variant;
7. encounter no silent source mutation, graph merging, unsafe Markdown
   execution, or unexplained truncation;
8. reach all of the above through the existing ELK-backed viewer path.

## Explicit assumptions

Approved delivery refinements, 2026-09-04: architecture is the default page;
OKF is an explicit secondary mode. The default OKF layout is ELK Mr. Tree.
Containment alone determines placement, with arrows toward children. Semantic
links are arrowless overlays shown only for the selected concept, without
relayout. Both viewers share Fit, interactions, layout controls, and
current-canvas Download SVG. This browser download reuses the existing SVG
exporter and is not a public OKF CLI or headless export contract. Profile-local
layout and configurable node/detail presentation remain separate from the
architecture model and its saved layout.

- The first delivery is local and interactive; public CLI, headless export, remote
  service, and collaboration contracts are future capabilities.
- The existing nearest-ancestor configuration owner can preserve unrelated
  configuration fields while adding the independently versioned okf section.
- The initial limits are tunable safety defaults: max_nodes 1000/3000,
  max_relationships 10000/30000, and a five-second layout budget.
- The supported OKF validator remains the authority for bundle conformance.
- The first built-in shape vocabulary is rectangle, rounded rectangle, pill,
  diamond, and hexagon; registry expansion does not change the scene contract.
- No unresolved product decision blocks downstream modeling. Exact serialized
  field names, endpoint shapes, and implementation package boundaries belong in
  the canonical contract and architecture artifacts.
