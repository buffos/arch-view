# Configurable OKF knowledge views — requirements gap analysis

## Status

This analysis covers the bounded [Configurable OKF knowledge views](../../../.okf/capabilities/okf-knowledge-views.md)
capability before its exact-specification artifacts are written. The discovery
notes and application synthesis provide enough product direction to proceed
without another broad interview.

## Strong enough to build on

- The capability is a first-class Arch View surface, separate from the
  architecture model but allowed to share the existing renderer-neutral scene,
  ELK layout, SVG rendering, and viewer interaction infrastructure.
- Project-tree discovery finds one or more independent `.okf` bundles. The
  selected bundle is the complete graph in the viewer; bundles are never
  merged.
- A lossless index preserves concept paths, frontmatter, Markdown body, links,
  and unknown metadata. Profiles interpret that index without mutating source
  documents.
- Profiles are project-local rendering costumes. They control projection,
  hierarchy, semantic-link visibility, facets, roles, roll-up policy, styles,
  details, navigation, and per-profile ELK options.
- Structural containment and Markdown-link relationships are separate layers.
  Explicit parent/children metadata takes precedence when selected by a
  profile; filesystem nesting is the fallback, and conflicts are diagnostics.
- Rules are declarative profile data evaluated by namespaced, versioned,
  in-process Go registry strategies. Matching rules compose; priorities resolve
  scalar conflicts; equal-priority conflicts are diagnosed; rule output is
  renderer-neutral.
- The user journey is defined: select a graph, select or edit a profile,
  project a depth-bounded or full view, inspect a node, and double-click to
  focus a subtree. Save and Save As have distinct project-profile behavior.
- Unknown state is neutral. Roll-up state is explicit and can expose declared
  and effective values. Large views are bounded by configurable limits plus an
  application hard cap, with visible diagnostics and recovery through depth or
  subtree focus.
- Inspection uses sanitized CommonMark and expandable metadata/diagnostic
  sections; unsafe HTML, scripts, URL schemes, and out-of-bundle file access
  are rejected.

## Blocking gaps

None currently block the next exact artifact. The high-impact product
decisions—graph independence, profile ownership, hierarchy precedence,
renderer boundary, rule extensibility, editing lifecycle, depth semantics,
inspection behavior, and scale protection—are stable enough for domain and
use-case modeling.

The assumptions below must be carried into the exact artifacts. If later
modeling exposes a contradiction, route back to the smallest affected artifact
instead of silently changing the product boundary.

## Deferrable gaps and bounded assumptions

### 1. Bundle identity and malformed candidates

**Category:** missing policy or invariant\
**Impact:** medium; affects discovery determinism and configuration bindings.

**Assumption:** A bundle ID is its normalized repository-relative POSIX path.
The bundle root is the parsing boundary. A candidate is selectable only after
strict OKF validation; an invalid candidate remains in diagnostics with its
path and validation errors. If a configured graph is missing or invalid, the
viewer preserves the stale reference, reports it, and selects the first valid
bundle in stable path order when one exists.

### 2. Hierarchy normalization and cycles

**Category:** missing failure path\
**Impact:** high; affects navigation, domain invariants, and reproducible
  projections.

**Assumption:** The normalized containment graph must be a forest rooted at the
selected bundle's root concepts. Explicit parent/children claims and
filesystem fallback are retained as provenance. A cycle, self-parent, or
multiple-parent conflict is not silently repaired: the affected claim is
excluded from navigable containment, a diagnostic is attached to each
affected concept, and the concepts remain available through the flat
diagnostic/index view. Semantic Markdown links may still render separately.

### 3. Rule conflict fallback

**Category:** missing policy or invariant\
**Impact:** high; affects deterministic rendering and acceptance tests.

**Assumption:** Every scalar style, visibility, projection, or detail field has
one effective value after profile composition. Highest priority wins.
Compatible collection/annotation outputs merge deterministically. Equal
priority for incompatible scalar values yields a diagnostic and uses the
profile's neutral/default value for that field; evaluation never depends on
registration order. A missing rule ID, unsupported schema version, invalid
parameters, or inheritance cycle invalidates only the affected rule/profile
layer and falls back to the nearest valid composed profile.

### 4. Profile and configuration persistence

**Category:** missing failure path\
**Impact:** medium; affects user trust and cross-platform behavior.

**Assumption:** The optional `okf` configuration section is validated
independently from the existing layout and analysis sections. Save writes a
complete validated document through a same-directory temporary file followed
by an atomic replacement where the host permits it. A validation or write
failure leaves the prior file untouched and reports the failure in the
viewer. Unknown fields in unrelated configuration sections are preserved by
the existing configuration owner.

### 5. Deterministic graph limits and truncation

**Category:** missing read/report requirement\
**Impact:** high; affects usability, reproducibility, and testability for large
  bundles.

**Assumption:** `depth` limits containment traversal from the current focus
root; semantic links never expand the containment frontier. Within a limit,
concepts and relationships are selected in normalized path order. When
`max_nodes` or `max_relationships` is reached, the projection stops
deterministically, reports visible limits and hidden counts, and exposes
subtree focus as the recovery path. The application hard cap cannot be
raised by a project profile.

### 6. Local Markdown links

**Category:** underspecified workflow\
**Impact:** medium; affects inspection navigation and security tests.

**Assumption:** A relative Markdown link resolves only within the selected
bundle and maps to the normalized concept-document path, optionally with a
document fragment for the detail panel. A link that cannot resolve remains
visible as an unresolved-link diagnostic. External HTTP, HTTPS, and mail
links are safe outbound links; all other schemes and path escapes are
rejected.

### 7. Viewer and service boundary

**Category:** missing scope boundary\
**Impact:** medium; affects contract shape and implementation sequencing.

**Assumption:** The first slice is a local interactive viewer capability. It
exposes the discovery, selection, projection, inspection, and profile
persistence behavior through the existing local Arch View host and web
viewer. It does not promise a public OKF CLI, export format, remote service,
collaboration, or source mutation. Those may be added later without
changing the lossless index/profile boundary.

### 8. Layout failure and cancellation

**Category:** missing failure path\
**Impact:** medium; affects resilience and user-visible outcomes.

**Assumption:** Projection and layout requests are cancellable. A superseded
request cannot replace the active view. ELK failure, timeout, or cancellation
produces a diagnostic and retains the last valid scene when possible; a
first-load failure shows an empty diagnostic state rather than partial,
misleading geometry.

## Vocabulary watchlist

The following terms are reserved for the glossary and must not be used
interchangeably:

- **bundle:** one validated `.okf` directory and its source documents;
- **concept document:** one Markdown document indexed as a graph concept;
- **lossless index:** source-preserving representation before profile
  interpretation;
- **containment:** navigable structural parent/child relationship;
- **semantic link:** a Markdown-link relationship separate from containment;
- **profile:** a named rendering costume and its composition;
- **rule:** a registered declarative evaluator invoked by a profile;
- **projection:** the profile-derived graph scene before layout;
- **scene:** renderer-neutral nodes, relationships, annotations, and diagnostics;
- **focus root:** the current bundle root or subtree root for depth traversal;
- **declared state:** state read from source or explicitly mapped metadata;
- **effective state:** state derived by an explicit profile policy;
- **roll-up:** a presentation role explicitly assigned by source metadata or
  profile policy, never inferred from child count alone.

## Next step

Proceed to glossary stabilization and then capability PRD authoring. Preserve
the assumptions above in the domain model, use cases, external contract, and
acceptance scenarios. No user clarification is required before that sequence.
