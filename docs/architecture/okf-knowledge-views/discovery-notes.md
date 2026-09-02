# Configurable OKF knowledge views — pre-PRD discovery

## Status

Bounded discovery baseline. The capability is a first-class, root-level Arch
View capability. Its exact external contracts and implementation details are
still future specification work, but its product boundary and first coherent
slice are now clear enough for structured design.

## Confirmed direction

- Arch View remains the product home for the feature.
- OKF bundles are source data; the feature must not require converting them
  into architecture models.
- The existing ELK-backed viewer remains the rendering foundation.
- A project may contain one or more .okf directories, including maps for
  capabilities, workflows, queries, or other knowledge domains.
- The OKF experience is a separate viewer surface with its own graph selector.
  Discovery reports the available bundles, and each bundle remains an
  independent graph; graph data is never merged.
- The selected graph may have an associated profile. If no association exists,
  the viewer uses its built-in default profile.
- Profiles are reusable project-local rendering costumes: they control the
  selected graph's projection and presentation, but never alter source
  documents, graph identity, or graph contents.
- A view profile should let users choose how source structure, metadata, and
  relationships become a graph.
- A user may configure a shallow projection or show the full reachable
  hierarchy. Double-clicking a node should focus the corresponding subtree.
- A fog-of-war profile may map states such as foggy, bounded, specified, and
  implemented to visual styles, but those labels must not be hard-coded into
  the OKF reader.
- Parent roll-up nodes must be distinguishable from implementation nodes by an
  explicit profile or source policy.
- Structural hierarchy is the navigation backbone for depth limits and
  subtree drill-down. Markdown links are a separate semantic relationship
  layer that profiles can show, hide, and style independently.
- A profile may choose explicit parent/children metadata as the containment
  source. Filesystem nesting is the fallback for nodes without an explicit
  parent, and conflicts are preserved as diagnostics instead of silently
  discarding one relationship.
- Roll-up status is explicit through source metadata or profile policy.
  Having children does not by itself make a node a roll-up node. A profile
  may give declared roll-ups a distinct visual role and expose effective state
  separately from declared state.
- Profile mappings are declarative. They may select preserved OKF facts such
  as type, frontmatter fields, tags, path, and relationships, then map
  matching values to named facets, shapes, colors, labels, and other
  presentation tokens. Profiles do not execute custom code.
- State is a conventional but optional source field. Profiles may map state,
  type, tags, paths, or arbitrary frontmatter fields; missing or unknown values
  remain neutral/unknown rather than being guessed. Roll-ups may show declared
  state separately from effective state.
- Visual mappings target named, validated style tokens. A token can define
  color, shape, border, label, and emphasis; profiles use a finite shape
  vocabulary and may expose a legend.
- The declarative rule language should be rich from the first implementation
  slice. Rule types must be independently registered strategies: each owns
  its configuration schema, validation, evaluation, and explanation metadata.
  The profile evaluator composes registered rules and must not grow a central
  rule switch.
- Rule evaluation is compositional: all applicable rules run, non-conflicting
  outputs merge, explicit priority resolves scalar conflicts, and equal
  priority conflicts produce diagnostics rather than relying on incidental
  ordering.
- The initial domain-neutral rule families should cover metadata fields,
  collections and tags, strings and paths, numeric comparisons,
  relationships, hierarchy and depth, and derived facts. Planning-specific
  behavior remains an independently registered extension.
- Profiles are data-only. Executable rule implementations come from
  in-process Go implementations registered by Arch View; target OKF
  repositories cannot inject rule code. External plugin loading is outside
  this capability's current scope.
- Registered extensions expose namespaced stable IDs, versioned configuration
  schemas, declared capabilities, and human-readable metadata for discovery,
  compatibility checks, explanations, and diagnostics.
- The capability must be designed for a small first release and a large
  extension surface. Rules, predicates, relationship adapters, style
  properties, shapes, detail renderers, and other open-ended vocabularies
  should use registries and schema-driven extension contracts rather than
  fixed central enums or small-count assumptions.
- Rule extensions emit renderer-neutral annotations, visibility decisions, and
  style patches. They do not emit SVG, ELK instructions, or other renderer
  operations; scene construction and renderer adapters own that translation.
- Named profiles may extend or compose other profiles. Base ordering and
  override precedence must be explicit and deterministic so a specialized
  rendering costume can reuse a neutral base without duplicating it.
- A profile may extend multiple bases through an ordered list. Derived
  overrides take precedence, inheritance cycles are invalid, and ambiguous
  equal-priority conflicts remain explicit diagnostics.
- The viewer should always provide a neutral built-in profile that works for
  any OKF bundle. It may also ship an optional fog-of-war example profile that
  demonstrates state and roll-up styling without making planning semantics
  mandatory.
- Built-in profiles are immutable. Editing one creates a working copy and
  requires Save As with a project-local profile name; Save updates only
  project-local profiles.
- Project-local profiles may be renamed or deleted. Rename updates all graph
  bindings atomically; deletion requires reassignment or an explicit fallback
  to the neutral profile. Built-in profiles cannot be renamed or deleted.
- Save As preserves the active profile's composition by creating a new named
  variant with the same base references and current overrides. Flattening into
  a standalone profile is a separate possible action, not an implicit Save As
  behavior.
- Large-graph protection uses a profile-configurable max-node limit plus an
  application hard cap. Reaching either limit produces an explicit truncation
  diagnostic and hidden-node count; nodes are never silently omitted. Users
  can reduce depth or drill into a subtree to continue.
- Single-click inspection should lead with an Overview containing title,
  description, type, path, mapped facets, and rendered Markdown. Hierarchy,
  relationships, selected frontmatter, and raw diagnostics remain expandable,
  with technical metadata secondary to human-readable content.
- The human-readable panel renders a sanitized CommonMark subset: headings,
  emphasis, lists, code, and safe links are allowed; raw HTML, scripts, and
  unsafe URL schemes are removed. Raw Markdown remains secondary technical
  data.
- Profiles are editable in the OKF viewer. Save updates the active profile;
  Save As creates a new named profile entry in the project configuration.
- Profile depth accepts any integer greater than or equal to 1 and means
  "at most this many hierarchy levels." A value deeper than the available
  tree reaches the entire tree. An explicit full/unbounded mode remains
  available, subject to the safety node limit.

## Brownfield observations

- The current architecture viewer has an architecture-specific scene model,
  an existing interactive inspection surface, and accessibility-oriented
  summary/details paths.
- The web viewer already invokes ELK and adapts ELK geometry into custom SVG.
- The advanced ELK renderer capability is a separate, still-specified
  renderer improvement under the architecture viewer capability.
- Project configuration already separates analysis choices from presentation
  and layout settings.
- The current viewer and PRD are architecture-oriented, so this capability
  requires an explicit product and application-architecture boundary.

## Working conceptual boundary

The likely pipeline is:

OKF bundle discovery -> lossless OKF index -> view profile -> renderer-neutral
graph scene -> ELK layout -> existing viewer interaction and inspection.

The OKF index should preserve source paths, frontmatter, Markdown body, links,
and unknown metadata. The view profile should derive display roles, facets,
effective values, and styles without mutating source documents.

Structural hierarchy and semantic Markdown links should remain separate
relationship classes. A profile may show either or both. Roll-up behavior
should be explicit, for example through a state-policy mode, rather than
inferred merely because a node has children.

## Bounded configuration baseline

Keep project-local OKF viewer settings in an optional top-level okf section of
the nearest .archview.json, with an independently versioned section schema.
The section contains:

- default_graph: a repository-relative POSIX bundle path;
- bindings: graph paths associated with optional project profile IDs;
- profiles: named project-local rendering costumes;
- profile definitions containing ordered bases, declarative rules, projection,
  style tokens, detail/navigation settings, and the OKF-specific ELK layout
  profile.

The configuration stores graph identity and preferences, not indexed graph
content. Discovery remains authoritative for the dropdown. Missing graph or
profile references produce visible diagnostics while preserving the stale
configuration for repair and falling back to the first available graph or
neutral profile without silently rewriting JSON. Built-in profiles are
immutable and project profiles are editable through Save/Save As/rename/delete
with the lifecycle rules above. A separate global profile catalog is deferred
until cross-project reuse is demonstrated.

## Candidate profile layers

1. Bundle and source selection.
2. Hierarchy selection and parent/child projection.
3. Relationship extraction and relation visibility.
4. Facets and derived values, including state and aggregate role.
5. Visual rules for color, shape, labels, and legends.
6. Node details and Markdown rendering.
7. Depth, subtree, and scale limits.
8. ELK and shared viewer presentation options.

## Resolved bounded design recommendations

1. Discovery scans the project tree recursively for directories named .okf.
   It skips .git, node_modules, vendor, dist, build, cache, generated, and
   other configured dependency/output directories; it does not follow
   symlinks outside the project root. Candidate bundles are validated before
   appearing as selectable graphs, while invalid candidates remain visible
   through diagnostics. Bundle paths are stable sorted graph IDs, and a bundle
   root is an independent parsing boundary.
2. The OKF section uses a versioned, strict schema with separate graph
   bindings and reusable profile definitions. Profiles include ordered base
   references, rule invocations with explicit priority, projection,
   style/detail/navigation settings, and a per-profile ELK layout profile.
   Save, Save As, rename, delete, built-in immutability, stale references, and
   inheritance cycles follow the confirmed lifecycle rules.
3. Containment uses explicit parent/children metadata when the selected
   profile declares it, and filesystem nesting for nodes without an explicit
   parent. Conflicting declarations remain diagnostics. Markdown links are
   semantic relationships, not containment.
4. Frontmatter fields are preserved without a fixed universal schema.
   Default adapters understand parent, children, state, type, title,
   description, and tags; profiles and registered relation adapters can map
   additional fields. Inline Markdown links become semantic edges with
   source/target provenance.
5. The in-process registry exposes namespaced IDs, versions, schemas,
   capabilities, and descriptions. Rule providers cover metadata,
   collections/tags, strings/paths, numeric comparisons, relationships,
   hierarchy/depth, and derived facts. Providers receive immutable
   renderer-neutral view facts and emit annotations, visibility decisions, and
   style patches. All matches compose; non-conflicting values merge, explicit
   priority resolves scalar conflicts, and equal-priority conflicts diagnose.
6. The initial navigation defaults to depth 2, accepts any integer at least 1
   as an at-most depth, and provides explicit full/unbounded mode. A
   profile-configurable max_nodes defaults to 1000; an application hard cap
   defaults to 3000 for the first implementation baseline. A recommended
   max_relationships default is 10000 with a 30000 application hard cap, and
   each projection request has a recommended five-second layout budget.
   These are tunable safety defaults rather than closed vocabulary limits.
   A future benchmark may tune them without changing the model. Layout and
   relationship work are bounded as well, with cancellation and visible
   diagnostics.
7. Inspection renders sanitized CommonMark with headings, emphasis, lists,
   code, and safe HTTP/HTTPS/mail links. Local concept links navigate within
   the selected graph; external links open safely in a new context. Unknown
   frontmatter is collapsed under metadata, raw Markdown is secondary, and
   unsafe HTML, scripts, URL schemes, and out-of-bundle file access are
   rejected.
8. Initial style providers expose validated fill/stroke/text colors, opacity,
   border width/dash, label emphasis, and a built-in shape set of rectangle,
   rounded rectangle, pill, diamond, and hexagon. The registry can add
   properties and shapes without modifying the scene or evaluator.
9. The first bounded slice includes separate OKF viewer navigation, recursive
   bundle discovery/dropdown, one-graph-at-a-time loading, lossless indexing,
   hierarchy and semantic-link layers, neutral and fog-of-war profiles,
   project-local profile persistence/editing, declarative registered rules,
   depth/subtree navigation, max-node diagnostics, sanitized inspection, and
   the shared renderer-neutral scene plus current ELK/SVG path. It reuses the
   existing supported ELK catalog; broader advanced-ELK renderer features,
   external rule plugins, graph merging, source mutation, and export-specific
   OKF projections are outside this first slice.

No unresolved product question blocks bounded status. The next planning phase
is exact specification: domain glossary, capability PRD, canonical domain and
use-case models, external contracts, acceptance scenarios, and readiness
review. Those artifacts may refine implementation-level limits while keeping
these boundaries stable.

## Bounded non-goals

- Replacing the architecture model or language analyzers.
- Requiring OKF authors to adopt Arch View's planning vocabulary.
- Mutating OKF documents from the viewer.
- Allowing arbitrary executable JavaScript as a profile expression language.
- Slicing implementation issues before the capability reaches a bounded,
  coherent scope and the root-level application synthesis is reviewed.

## Decision log

### 2026-09-02

- User confirmed this is a first-class capability under the Arch View product,
  rather than a child of the architecture viewer.
- User confirmed the capability should be added at the project root level.
- Work continues on branch codex/configurable-okf-viewer; the pre-existing
  .gitignore change was committed before planning began.
- User confirmed that discovered OKF bundles must never be merged. The
  separate OKF viewer should present available bundles in a graph dropdown and
  treat the selected bundle as the complete graph being viewed.
- User confirmed project-local persistence: store the selected/default graph,
  graph-to-profile bindings, and reusable profile definitions in an optional
  okf section of .archview.json.
- User confirmed recursive project-tree discovery for valid .okf bundles, with
  standard safety exclusions such as .git, node_modules, dist, and generated
  output, and deterministic ordering in the graph selector.
- User confirmed that profiles are project-local reusable rendering costumes.
  They configure how a selected graph is projected and presented, not the
  source bundle or its graph identity.
- User confirmed that structural hierarchy and Markdown-link relationships are
  separate profile-controlled layers. Hierarchy drives navigation; links are
  semantic edges.
- User confirmed hierarchy precedence: profiles may select explicit
  parent/children metadata, filesystem nesting is the fallback, and conflicts
  remain visible as diagnostics.
- User confirmed that roll-up status must be explicit through source metadata
  or profile policy; child count alone must not imply a roll-up node.
- User confirmed declarative profile mappings: preserved and derived facts map
  to named presentation tokens without arbitrary executable code.
- User confirmed that state is an optional conventional field. Unknown or
  missing values remain neutral/unknown, and roll-ups may expose declared and
  effective state separately.
- User confirmed named, validated style tokens with finite shapes and colors;
  mappings select tokens, and the viewer may show a legend.
- User rejected a minimal rule vocabulary. The rule system should be rich from
  the start and follow the Open-Closed Principle: add/register new rule
  strategies rather than modifying a central switch or conditional chain.
- User confirmed compositional rule resolution: merge non-conflicting outputs,
  use explicit priority for scalar conflicts, and diagnose equal-priority
  conflicts.
- User confirmed broad domain-neutral rule families for metadata,
  collections/tags, strings/paths, numeric comparisons, relationships,
  hierarchy/depth, and derived facts; planning-specific behavior remains an
  extension.
- User confirmed that profiles are data-only and executable rule
  implementations must come from in-process Go implementations registered by
  Arch View; target OKF repositories cannot inject rule code. External plugin
  loading is out of scope for now.
- User confirmed registered extensions should expose namespaced stable IDs,
  versioned schemas, declared capabilities, and human-readable metadata.
- User confirmed the capability should be designed for 3 to 3000 extensions
  across rules, properties, styles, and related vocabularies; temporal
  simplicity must not create fixed-count or closed-enum constraints.
- User confirmed that rule extensions produce renderer-neutral annotations and
  style patches; scene construction and renderers translate those outputs.
- User confirmed that named profiles may extend/compose other profiles with
  explicit deterministic base and override precedence.
- User confirmed that profiles may extend multiple ordered bases, with cycle
  detection and explicit diagnostics for ambiguous conflicts.
- User confirmed that Save As preserves profile composition, creating a new
  named variant with the same base references and current overrides.
- User agreed that the viewer should ship a neutral built-in profile and an
  optional fog-of-war example profile; the latter remains editable and does
  not make planning semantics mandatory.
- User agreed that built-in profiles are immutable: editing one requires Save
  As to create a project-local profile, while Save updates project-local
  profiles only.
- User agreed that project-local profiles may be renamed or deleted safely:
  rename updates bindings atomically, deletion requires reassignment or
  explicit neutral fallback, and built-ins remain immutable.
- User confirmed large-graph handling: profile-configurable node limits plus
  an application hard cap, explicit truncation diagnostics and hidden counts,
  and depth reduction/subtree drill-down as recovery paths.
- User confirmed the single-click inspection direction: human-readable
  overview, rendered Markdown, and mapped metadata first; hierarchy,
  relationships, frontmatter, and technical diagnostics expandable.
- User confirmed sanitized CommonMark for rendered Markdown, with raw HTML,
  scripts, and unsafe URL schemes removed; raw Markdown remains secondary.
- User confirmed profile editing and persistence behavior: Save updates the
  active profile, while Save As creates a new profile entry in the JSON.
- User confirmed that depth is any integer at least 1 with "at most" semantics;
  deeper-than-available values show the whole tree, and full/unbounded mode
  remains explicit.
- At the user's direction, the remaining bounded design questions were
  resolved with recommended defaults rather than left as product ambiguity.
- The capability advanced from foggy to bounded after the bounded design
  recommendations and first-slice boundary were recorded. This does not claim
  that implementation or exact specification is complete.
