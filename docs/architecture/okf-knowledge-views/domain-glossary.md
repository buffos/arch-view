# Configurable OKF knowledge views — domain glossary

## Purpose

This glossary is the canonical vocabulary for the configurable OKF
knowledge-view capability. It describes product and domain concepts, not Go
types or frontend components. Terms from the OKF source format remain
preserved as source facts even when the viewer uses a more precise canonical
term.

## Actors

| Canonical term | Definition | Aliases or discouraged terms |
|---|---|---|
| Viewer user | Person who opens Arch View to select an OKF bundle, apply a profile, explore the projection, and inspect concepts. | developer, knowledge worker; use the role that is relevant in a scenario |
| Bundle maintainer | Person who creates or changes OKF source documents and expects Arch View to consume them without mutation. | OKF author, map author; these are context-specific aliases |
| Profile maintainer | Viewer user who creates, edits, renames, or deletes project-local rendering profiles. | profile author; not necessarily the bundle maintainer |

The roles may be the same person. They are separated because source ownership
and presentation ownership have different permissions and failure behavior.

## Business and domain objects

| Canonical term | Category | Definition | Aliases or discouraged terms |
|---|---|---|---|
| Arch View project | external boundary/object | Local repository context in which Arch View discovers source bundles and reads/writes the nearest project configuration. | project; qualify it when referring to the OKF source |
| OKF bundle | external object | One validated directory named `.okf` and the Markdown concept documents, reserved files, and nested source content within its parsing boundary. | OKF graph, map; a bundle may be presented as a graph but is the source boundary |
| Concept document | business object | A non-reserved Markdown document in a bundle that can be indexed as a knowledge concept. | node file, concept; “concept” may refer to the projected node when source identity is not relevant |
| Source fact | business object | Losslessly preserved information from a concept document, including normalized path, frontmatter, Markdown body, links, and unknown metadata. | raw node data; source facts are not presentation properties |
| Lossless index | business object | Read-only, source-preserving collection of source facts and normalized provenance for one bundle. | parsed graph, source model; it exists before profile interpretation |
| View profile | business object | Named, reusable rendering costume that selects projection rules, relationship layers, visual tokens, details, navigation, and ELK options for an indexed bundle. | graph profile, rendering preset; “profile” alone is acceptable after context is established |
| Built-in profile | business object | Immutable profile supplied by Arch View, such as the neutral default or fog-of-war example. | system profile; built-ins are not project-local definitions |
| Project-local profile | business object | Mutable profile definition stored in the nearest `.archview.json` for one Arch View project. | custom profile, saved profile |
| Profile composition | business object/policy | Ordered inheritance of one or more base profiles plus the named profile's overrides. | profile extension, profile stacking; composition is not source-bundle merging |
| Graph binding | business object | Association between a normalized bundle ID and an optional project-local profile ID in configuration. | graph/profile association, assignment |
| Rule invocation | business object | Versioned reference to a registered rule strategy plus parameters and priority inside a profile. | rule; use “rule” for the strategy and “invocation” for its configured use |
| Presentation token | business object | Validated, named value used by a profile to express a visual or detail outcome without targeting a concrete renderer. | style value, CSS class; tokens are renderer-neutral |
| Projection | business object | Profile-derived selection and annotation of concepts and relationships before layout. | graph view, rendered graph; projection is not yet positioned geometry |
| Renderer-neutral scene | business object | Portable nodes, relationships, annotations, styles, and diagnostics produced from a projection and consumed by renderers. | scene model, view model; it contains no ELK commands or SVG markup |
| Concept detail | business object | Read-side inspection data for one selected concept, including overview, rendered Markdown, mapped metadata, and expandable diagnostics. | node details, metadata panel |
| Diagnostic | reporting object | Structured explanation of invalid, conflicting, truncated, unresolved, unsupported, or failed processing, with affected source identity where possible. | warning, error; severity is a property of a diagnostic |

## Relationships

| Canonical term | Category | Definition | Aliases or discouraged terms |
|---|---|---|---|
| Containment | relationship | Navigable structural parent/child relationship used for depth traversal, breadcrumbs, and subtree focus. | hierarchy edge, parent link; do not call every relationship a containment |
| Semantic link | relationship | Relationship derived from a Markdown link or registered relation adapter and shown independently from containment. | reference edge, dependency edge, link edge |
| Explicit hierarchy claim | source fact/policy input | Parent or children relationship declared in source metadata and selected by a profile's hierarchy mapping. | declared parent, frontmatter hierarchy |
| Filesystem fallback | policy | Containment derived from directory nesting when no selected explicit hierarchy claim supplies a parent. | path hierarchy, folder hierarchy |
| Relationship provenance | reporting object | Source location and adapter/rule explanation showing why a containment or semantic link exists. | edge evidence |

Containment and semantic links are never merged implicitly. A concept may have
both, and each layer has independent visibility and style decisions.

## Workflows and actions

| Canonical term | Category | Definition | Aliases or discouraged terms |
|---|---|---|---|
| Discover bundles | workflow/action | Recursively find eligible `.okf` directories under the project, apply safety exclusions, and produce stable bundle candidates. | scan project, find graphs |
| Validate bundle | workflow/action | Check a candidate against the supported OKF conformance rules before it becomes selectable. | parse bundle; parsing and validation are separate outcomes |
| Select bundle | workflow/action | Choose one discovered bundle as the current source graph. | choose graph, open map |
| Index bundle | workflow/action | Read a selected bundle into a lossless index without changing source files. | load graph, parse graph |
| Project view | workflow/action | Evaluate profile composition and rule invocations against the index to produce the current renderer-neutral scene. | render graph; rendering includes layout and drawing after projection |
| Inspect concept | workflow/action | Open the concept detail for a selected node, preserving safe source content and mapped presentation information. | click node, view metadata |
| Focus subtree | workflow/action | Make a selected concept the focus root and rebuild the projection from that containment subtree. | drill down, double-click node |
| Edit profile | workflow/action | Change a project-local profile's projection or presentation settings in the viewer. | customize graph, edit costume |
| Save profile | workflow/action | Validate and persist changes to the active project-local profile. | apply profile; Save does not mutate a built-in |
| Save As profile | workflow/action | Create a new project-local profile from the current composition and overrides without flattening base references. | clone profile, duplicate preset |
| Rename profile | workflow/action | Change a project-local profile ID/name and update bindings atomically. | retitle preset |
| Delete profile | workflow/action | Remove a project-local profile after reassignment or explicit neutral fallback is established. | remove preset |

## Policies and rules

| Canonical term | Category | Definition | Aliases or discouraged terms |
|---|---|---|---|
| Bundle boundary | policy | The discovered `.okf` directory is the independent source and resolution boundary; a view never combines multiple bundles. | graph boundary |
| Read-only consumption | policy | Arch View may read source bundles and persist its own configuration, but it never edits OKF documents. | non-destructive viewing |
| Hierarchy precedence policy | policy | Selected explicit hierarchy metadata supplies containment first; filesystem fallback supplies missing parents; conflicts remain diagnostics. | parent resolution |
| Rule strategy | rule | In-process Go implementation registered under a stable namespaced ID that validates and evaluates one declarative rule family. | plugin rule; external plugins are not implied |
| Rule invocation | rule/configuration | Profile data selecting a rule strategy, version, parameters, and priority. | rule declaration |
| Rule composition policy | policy | All applicable invocations run; compatible outputs merge; priorities resolve scalar conflicts; equal-priority conflicts diagnose and use neutral/default fallback. | precedence policy |
| Profile validity | policy/state | Whether a profile's bases, rule schemas, parameters, tokens, and limits can be resolved and validated. | usable preset |
| Roll-up policy | policy | Explicit mapping that marks a concept as an aggregate/presentation node and may derive effective state from declared structural children. | aggregate rule; child count alone is not a roll-up policy |
| Depth policy | policy | `depth` is an integer at least 1 with at-most semantics, or an explicit full/unbounded mode subject to safety limits. | levels, depth limit |
| Scale-safety policy | policy | Configurable node/relationship limits and application hard caps prevent unbounded projection or layout work and require visible truncation diagnostics. | pagination; limits are not pagination |
| Markdown safety policy | policy | Inspection renders a sanitized CommonMark subset and rejects unsafe HTML, scripts, schemes, and path escapes. | sanitization, safe rendering |
| Configuration atomicity policy | policy | Validated profile changes replace the configuration as one write where the host permits; failed validation or writes preserve the previous file. | safe save |

Rules evaluate data; policies constrain what outcomes are valid. A policy is
not automatically a new rule family, and a rule must not embed renderer-specific
behavior.

## States and statuses

| Canonical term | Category | Definition | Aliases or discouraged terms |
|---|---|---|---|
| Declared state | source/presentation status | State value read from source metadata or explicitly mapped frontmatter. It may be absent or unknown. | raw state, source state |
| Effective state | presentation status | State shown after profile mapping or explicit roll-up policy has interpreted available source facts. | computed state, displayed state |
| Unknown state | presentation status | Neutral outcome when a state is absent, unrecognized, or not mapped by the active profile. | missing state; do not treat as foggy automatically |
| Bundle validity | processing status | Valid, invalid, or unavailable result of bundle validation and loading. | parse status |
| Projection validity | processing status | Valid, truncated, diagnostic, or failed result of profile evaluation and scene construction. | render status |
| Planning state | planning status | Arch View map maturity value: `foggy`, `bounded`, `specified`, or `implemented`. It describes planning readiness, not an OKF concept's source state. | lifecycle state; never use it as a graph-node state by implication |

The planning states and OKF concept states are separate vocabularies. A
fog-of-war profile may map source metadata to presentation tokens named
“foggy”, “bounded”, “specified”, or “implemented”, but that mapping is a
profile choice rather than a universal OKF rule.

## Metrics and reports

| Canonical term | Definition | Aliases or discouraged terms |
|---|---|---|
| Focus root | Bundle root or selected subtree concept from which containment traversal begins. | current node, viewport root |
| Visible concept count | Number of concepts included in the current projection. | rendered nodes; layout failures may occur after projection |
| Hidden concept count | Deterministic count of concepts excluded by depth or node limits and recoverable through a narrower focus. | omitted nodes; never imply silent loss |
| Hidden relationship count | Count of relationships excluded by relationship limits or visibility policy, with reason. | dropped edges |
| Truncation diagnostic | Diagnostic explaining which limit stopped projection and how many items remain hidden. | overflow warning |
| Unresolved link diagnostic | Diagnostic for a local Markdown link that cannot resolve inside the selected bundle. | broken link |

## External systems and boundaries

| Canonical term | Definition | Aliases or discouraged terms |
|---|---|---|
| OKF format | Source directory-and-Markdown convention consumed by the capability. | OKF graph format |
| `.archview.json` | Existing nearest-ancestor Arch View configuration document that owns layout, analysis, and optional OKF viewer settings. | Arch View config, project config |
| ELK layout | Existing layout service/integration that calculates graph geometry from the renderer-neutral scene. | ELK renderer; ELK performs layout, while the viewer renderer draws |
| Arch View local host | Existing local application boundary serving discovery, projection, profile persistence, and viewer data. | backend, server |
| Rule registry | In-process catalog of available rule strategies, style properties, shapes, relationship adapters, and metadata. | plugin registry; external plugin loading is out of scope for this slice |

## Critical distinctions

- A bundle is a source boundary; a projection is one profile's view of that
  bundle. Selecting another profile does not create another source graph.
- A concept document is source identity; a projected scene node is a
  renderer-neutral view result. The latter may be hidden by profile rules without
  deleting the former.
- Containment drives navigation and depth. Semantic links express meaning and
  do not expand the containment frontier.
- A view profile is the complete rendering costume. An ELK layout profile is
  only one part of it.
- Declared state is evidence; effective state is interpretation. Unknown is
  neutral, not an inferred planning status.
- A rule strategy is executable in-process implementation. A rule invocation is
  data stored in a profile. Profiles cannot contain arbitrary executable code.
- A profile composition conflict is not a source-data conflict. Both must be
  diagnosed distinctly.
- Save changes a project-local profile. Save As creates a new profile. Neither
  mutates the source bundle.

## Open terminology issues

No terminology issue blocks exact specification. The source adapter will need
to document how arbitrary frontmatter fields such as `parent`,
`children`, `state`, `type`, `title`,
`description`, and `tags` map into canonical source
facts; those are adapter mappings, not a universal OKF schema. The exact
serialized names for profile rule families and configuration fields belong in
the canonical contract.
