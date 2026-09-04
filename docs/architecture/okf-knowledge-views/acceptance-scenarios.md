# Configurable OKF knowledge views — acceptance scenarios

## Verification convention

These are canonical behavior scenarios, not transport-specific tests. Each
scenario records verification intent for the root project's when-supported
verification policy:

- backend boundary: an external local-host or future CLI request can initiate
  and observe the behavior;
- frontend integration: a viewer action can initiate the behavior and consume
  its result;
- end-to-end journey: a user can reach the behavior from the Arch View
  product surface and observe the outcome.

## SC-001 — Reach and select independent bundles

**Purpose:** Prove that the feature is reachable and does not merge sources.

**Given** an Arch View project contains two valid bundles at different
repository-relative paths.

**When** the viewer user opens the OKF viewer and selects each bundle in turn.

**Then** the selector lists both bundles in deterministic path order and each
projection contains concepts from only the selected bundle. The viewer exposes
the selected bundle identity.

**Rule coverage:** bundle boundary and no-merge invariant.\
**Verification:** backend boundary when-supported; frontend integration
when-supported; end-to-end when-supported.

## SC-002 — Diagnose an invalid bundle candidate

**Purpose:** Prove that malformed source is visible but cannot silently enter
the graph view.

**Given** the project contains one valid bundle and one __BT__.okf__BT__ candidate that
fails supported OKF validation.

**When** the viewer refreshes bundle discovery.

**Then** the valid bundle is selectable, the invalid candidate is reported with
its relative path and validation diagnostics, and the invalid candidate cannot
be opened as a valid projection.

**Rule coverage:** validation-before-selection and failure isolation.\
**Verification:** backend boundary when-supported; frontend integration
when-supported; end-to-end when-supported.

## SC-003 — Preserve arbitrary source facts

**Purpose:** Prove lossless, read-only source consumption.

**Given** a valid concept document contains standard frontmatter, unknown
frontmatter, Markdown body content, and a local link.

**When** the viewer indexes the bundle and the user inspects the concept.

**Then** the detail view shows the standard overview, preserves the unknown
metadata in expandable source information, renders the Markdown safely, shows
the local link outcome, and leaves the source document byte-for-byte unchanged.

**Rule coverage:** lossless index and read-only consumption.\
**Verification:** backend boundary when-supported; frontend integration
when-supported; end-to-end when-supported.

## SC-004 — Apply the neutral built-in profile

**Purpose:** Prove that every valid bundle has a usable baseline view.

**Given** a valid bundle has no saved profile binding.

**When** the viewer user opens the bundle.

**Then** the viewer applies the immutable neutral built-in profile, projects
the available concepts and relationships, and shows the profile identity and
any source diagnostics.

**Rule coverage:** neutral fallback and built-in availability.\
**Verification:** backend boundary when-supported; frontend integration
when-supported; end-to-end when-supported.

## SC-005 — Apply two costumes to one source

**Purpose:** Prove that profiles change presentation without changing source
identity or content.

**Given** one indexed bundle has a neutral profile and a fog-of-war profile
with different state-to-token mappings.

**When** the viewer user applies each profile to the same bundle.

**Then** the scenes use the same source and concept identities while labels,
roles, colors, shapes, legend entries, or visibility differ according to the
active profile. No OKF document changes.

**Rule coverage:** profile/source separation and configurable presentation.\
**Verification:** backend boundary when-supported; frontend integration
when-supported; end-to-end when-supported.

## SC-006 — Resolve explicit hierarchy, fallback, and conflict

**Purpose:** Prove deterministic containment normalization.

**Given** some concepts declare explicit parents, some have only filesystem
nesting, and one concept has conflicting or cyclic parent claims.

**When** the selected profile projects containment.

**Then** selected explicit claims take precedence, filesystem nesting supplies
only missing parents, and the conflicting/cyclic claims are excluded from
navigable containment with concept-linked diagnostics. The affected concepts
remain available in the diagnostic/index view.

**Rule coverage:** hierarchy precedence, forest invariant, and provenance.\
**Verification:** backend boundary when-supported; frontend integration
when-supported; end-to-end when-supported.

## SC-007 — Keep semantic links separate

**Purpose:** Prove that Markdown relationships do not become navigation
containment.

**Given** a concept links to another concept that is not its containment child.

**When** the active profile enables semantic links and the user changes
containment depth.

**Then** the semantic link is shown with its own relationship kind and style,
but it does not increase the containment traversal frontier, change
breadcrumbs, or make the target a subtree child.

**Rule coverage:** relationship-kind separation.\
**Verification:** backend boundary when-supported; frontend integration
when-supported; end-to-end when-supported.

## SC-008 — Map known and unknown state

**Purpose:** Prove that state interpretation is optional and explainable.

**Given** one concept has a recognized source state, one has an unrecognized
state, and one has no state; a profile maps the recognized value and declares
an explicit roll-up policy for an aggregate concept.

**When** the viewer projects and inspects the concepts.

**Then** the recognized value receives its mapped token, absent/unrecognized
values remain neutral/unknown, and the aggregate detail can distinguish
declared state from effective roll-up state. Child count alone does not assign
the roll-up role.

**Rule coverage:** state mapping and explicit roll-up policy.\
**Verification:** backend boundary when-supported; frontend integration
when-supported; end-to-end when-supported.

## SC-009 — Enforce at-most depth semantics

**Purpose:** Prove depth behavior for small and large graphs.

**Given** the selected bundle contains a containment tree deeper than two
levels.

**When** the user requests depth 1, depth 2, a value greater than the actual
tree depth, and explicit full mode.

**Then** depth 1 and depth 2 show at most the requested containment levels, the
larger value reaches the available tree, and full mode requests all available
levels subject to node/relationship safety limits. The current profile and
bundle remain unchanged.

**Rule coverage:** positive integer validation and at-most semantics.\
**Verification:** backend boundary when-supported; frontend integration
when-supported; end-to-end when-supported.

## SC-010 — Report truncation and recover through focus

**Purpose:** Prove that scale protection is visible and useful.

**Given** a bundle exceeds the active node or relationship budget.

**When** the viewer projects the bundle at full mode or the configured depth.

**Then** the viewer shows a displayable truncated result with the applied
limit, visible and hidden counts, and recovery guidance. The user can focus a
hidden subtree and obtain its projection without raising the application hard
cap.

**Rule coverage:** deterministic limits, visible omission, and recovery.\
**Verification:** backend boundary when-supported; frontend integration
when-supported; end-to-end when-supported.

## SC-011 — Focus a subtree and navigate back

**Purpose:** Prove the primary drill-down journey.

**Given** the viewer displays a bundle with a navigable containment child.

**When** the user double-clicks the child and then chooses Back.

**Then** the child becomes the focus root while the bundle, profile, and depth
policy remain active; breadcrumbs expose the previous context; Back restores
the prior focus and projection.

**Rule coverage:** ViewSession focus/history invariants.\
**Verification:** backend boundary when-supported; frontend integration
when-supported; end-to-end when-supported.

## SC-012 — Inspect safe Markdown and links

**Purpose:** Prove useful detail without code execution or path escape.

**Given** a concept contains headings, emphasis, lists, code, a local concept
link, an external HTTPS link, raw HTML/script content, an unsafe URL scheme,
and a relative path that escapes the bundle.

**When** the user opens concept detail.

**Then** supported CommonMark renders, local links stay within the selected
bundle, the HTTPS link is offered as a safe external link, unsafe HTML/scripts
and schemes do not execute, the escaping path is rejected, and relevant
diagnostics are visible.

**Rule coverage:** Markdown safety and bundle boundary.\
**Verification:** backend boundary when-supported; frontend integration
when-supported; end-to-end when-supported.

## SC-013 — Compose rules deterministically

**Purpose:** Prove extensible rule behavior and conflict semantics.

**Given** a profile invokes multiple registered rules, including compatible
annotation outputs, different-priority scalar outputs, and equal-priority
incompatible scalar outputs.

**When** the profile is validated and applied.

**Then** compatible outputs merge, the highest-priority scalar wins, the
equal-priority conflict produces a diagnostic and neutral/default field value,
and changing registry registration order does not change the result.

**Rule coverage:** rule composition, priority, and Open-Closed registry seam.\
**Verification:** backend boundary when-supported; frontend integration
when-supported; end-to-end when-supported.

## SC-014 — Repair an unavailable profile layer

**Purpose:** Prove safe fallback for stale composition.

**Given** a saved graph binding references a profile whose base or rule is
missing or uses an unsupported schema.

**When** the user opens the bundle.

**Then** the stale binding and affected profile layer are diagnosed, the
configuration is not silently rewritten, and the viewer uses the nearest valid
composed profile or neutral built-in fallback to produce a truthful result.

**Rule coverage:** stale reference preservation and profile validity.\
**Verification:** backend boundary when-supported; frontend integration
when-supported; end-to-end when-supported.

## SC-015 — Save a project-local profile and Save As a variant

**Purpose:** Prove the distinct profile lifecycle operations.

**Given** the user has a project-local profile and an immutable built-in profile
available in the profile catalog.

**When** the user edits and saves the project-local profile, then uses Save As
on the built-in profile with a new profile ID.

**Then** Save updates the existing project-local definition; Save As creates a
new project-local definition, preserves ordered base references and overrides,
and makes the new profile available for the current session. Direct overwrite
of the built-in is rejected.

**Rule coverage:** mutable versus immutable profile lifecycle.\
**Verification:** backend boundary when-supported; frontend integration
when-supported; end-to-end when-supported.

## SC-016 — Preserve binding integrity during rename and delete

**Purpose:** Prove that profile maintenance cannot leave silent dangling
bindings.

**Given** multiple bundle bindings point to a project-local profile.

**When** the profile maintainer renames it, then attempts deletion with and
without a replacement/fallback.

**Then** rename updates all affected bindings atomically. Deletion without an
explicit replacement or neutral fallback is rejected. Deletion with a valid
replacement/fallback removes the profile and leaves every binding valid.

**Rule coverage:** configuration aggregate invariants.\
**Verification:** backend boundary when-supported; frontend integration
when-supported; end-to-end when-supported.

## SC-017 — Keep failed persistence non-destructive

**Purpose:** Prove that invalid or failed configuration writes preserve the
last valid state.

**Given** the nearest configuration contains a valid layout, analysis, and OKF
section.

**When** the user attempts to save an invalid profile or the configuration
write fails.

**Then** the viewer reports field/profile or filesystem diagnostics, the prior
configuration remains intact, unrelated layout and analysis settings remain
unchanged, and the active valid profile remains usable.

**Rule coverage:** validation-before-write and configuration atomicity.\
**Verification:** backend boundary when-supported; frontend integration
when-supported; end-to-end when-supported.

## SC-018 — Reject stale writes and duplicate creation

**Purpose:** Prove concurrency and retry safety for configuration commands.

**Given** two clients edit the same configuration revision and one client has
already saved; a second Save As request is retried with the same operation ID.

**When** the stale client saves and the retried Save As is processed.

**Then** the stale save returns a revision conflict without overwriting the
newer profile, and the repeated Save As returns the original created profile
without creating a duplicate. Reusing the operation ID with different input
returns an idempotency conflict.

**Rule coverage:** optimistic concurrency and idempotency.\
**Verification:** backend boundary when-supported; frontend integration
when-supported; end-to-end when-supported.

## SC-019 — Prevent stale projection results

**Purpose:** Prove cancellation and request supersession behavior.

**Given** a projection request is evaluating for a selected session and the
user changes profile, depth, or focus before it finishes.

**When** the newer request completes or the older request is cancelled.

**Then** the newer valid projection is the active scene, the older result
cannot replace it, and cancellation/supersession is visible through operation
status or diagnostics. If an existing valid scene exists during layout failure,
it remains displayed when safe.

**Rule coverage:** ViewSession request ordering and resilience.\
**Verification:** backend boundary when-supported; frontend integration
when-supported; end-to-end when-supported.

## SC-020 — Isolate bundle failures

**Purpose:** Prove that independent graph sources fail independently.

**Given** a project contains several bundles and one becomes unavailable or
unreadable during refresh.

**When** the viewer refreshes and selects another valid bundle.

**Then** the unavailable bundle is diagnosed, the other valid bundles remain
selectable, and the selected valid bundle can be indexed and viewed without
including data from the failed bundle.

**Rule coverage:** independent bundle boundary and failure isolation.\
**Verification:** backend boundary when-supported; frontend integration
when-supported; end-to-end when-supported.

## Stateful interaction coverage

| Interaction | Coverage decision |
|---|---|
| Bundle validity × selection | SC-001, SC-002, SC-014, SC-020 |
| Profile validity/origin × save/apply | SC-004, SC-005, SC-014, SC-015 |
| Projection status × navigation action | SC-009, SC-010, SC-011, SC-019 |
| Configuration mutation × in-flight projection | SC-017, SC-018, SC-019 |
| Independent bundle failure × other bundle selection | SC-002, SC-020 |
| Retry/concurrency × profile persistence | SC-016, SC-017, SC-018 |
| Time × lifecycle | not-applicable: no time-based business lifecycle exists in this capability; request budgets and cancellation are covered by SC-019 |
| Approval/return/shortage branches | not-applicable: this is a knowledge-view capability, not a transactional commerce workflow |

## Scenario quality notes

The set covers reachability, normal discovery, alternate presentation,
hierarchy and relationship policy, state mapping, depth/scale behavior,
inspection/security, profile lifecycle, persistence failure, concurrency,
cancellation, and independent work failure. It intentionally does not create
separate scenarios for every rule family or every built-in style token; those
are covered by parameterized contract/domain tests once the registry contract
is implemented.
