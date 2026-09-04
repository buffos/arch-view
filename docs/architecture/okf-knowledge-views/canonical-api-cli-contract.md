# Configurable OKF knowledge views — canonical API/CLI contract

## Purpose

This document defines the stable external behavior of the OKF viewer. It maps
the canonical use cases to transport-neutral payloads, the existing local Arch
View HTTP host, and a reserved CLI shape. The first delivery is local and
interactive; the CLI mapping is recorded for parity and future delivery, not
as a promise to ship a public OKF CLI in the first slice.

Implementations may differ internally, but clients must receive the same
meaning, identifiers, statuses, diagnostics, retry behavior, and profile
lifecycle outcomes.

## Contract goals

- preserve one-bundle source scope and profile semantics;
- expose deterministic, source-backed projection results;
- make validation, conflict, truncation, cancellation, and persistence
  outcomes explicit;
- support the existing local web viewer without exposing package internals;
- keep HTTP and future CLI adapters semantically aligned;
- allow renderer/layout details to vary behind the renderer-neutral scene.

## Transport-neutral conventions

### JSON and timestamps

- JSON is UTF-8.
- Timestamps are RFC3339 UTC strings.
- Absent optional values are omitted or represented as null consistently by an
  adapter; an absent value is not the same as an empty collection.
- Collections are deterministically ordered as specified by each operation.

### Response envelope

Successful responses use:

    {
      "data": <result>,
      "diagnostics": [],
      "meta": {
        "request_id": "opaque-request-id",
        "operation_id": "opaque-operation-id",
        "revision": "opaque-revision"
      }
    }

Diagnostics may accompany a successful result. Truncated projections are
successful displayable results with projection status truncated.

Failed responses use:

    {
      "error": {
        "code": "stable_machine_code",
        "message": "human-readable explanation",
        "details": {},
        "diagnostics": []
      },
      "meta": {
        "request_id": "opaque-request-id"
      }
    }

Messages are explanatory; clients branch on codes and structured details.

### Identifiers

- project_id: stable identifier for the local Arch View project context.
- bundle_id: normalized repository-relative POSIX path of the .okf directory.
  This is intentionally readable and stable because it is also the persisted
  graph binding identity.
- concept_id: bundle-scoped normalized relative Markdown path. It is unique
  only together with bundle_id.
- profile_id: stable namespaced profile identity. Built-ins use a
  built-in-prefixed ID; project-local profiles use a project-prefixed ID.
- rule_id, adapter_id, property_id, shape_id: namespaced stable registry IDs.
- source_revision, profile_revision, configuration_revision, and
  projection_revision: opaque revision strings.
- operation_id and request_id: opaque caller/request identifiers.

Path IDs are serialized as strings and URL/path encoded by HTTP adapters. A
client must not construct a concept ID by guessing a filesystem path outside
the bundle boundary.

### Revisions and optimistic concurrency

Configuration-changing commands accept expected configuration/profile revision
values. A stale revision returns a revision conflict and does not overwrite
newer data. Successful writes return the new revision.

### Idempotency

Persisting or creating commands accept an operation_id or HTTP
Idempotency-Key. Repeating the same operation ID with equivalent input returns
the original result without creating a duplicate profile or applying a second
rename/delete. Reusing an operation ID with different input returns an
idempotency conflict.

Discovery, selection, navigation, and projection requests are repeatable but
may be superseded or cancelled. They do not require idempotency keys.

### Cancellation and supersession

Long-running requests may carry a cancellation token or client disconnect.
When a newer request supersedes an older session request, the older request
must not replace the newer active result. The response identifies cancelled or
superseded status where the transport can return it.

## Stable status vocabulary

### Bundle candidate status

- discovered
- validating
- valid
- invalid
- unavailable
- unreadable

Only valid candidates are selectable.

### Profile status

- valid
- invalid
- partially_applicable
- unavailable

A stored profile may be invalid or unavailable while remaining visible for
repair.

### Projection status

- requested
- evaluating
- ready
- truncated
- failed
- cancelled
- superseded

Ready and truncated are displayable. Failed, cancelled, and superseded results
must not replace a newer valid session result.

### Relationship kind

- containment
- semantic_link

### Diagnostic severity

- info
- warning
- error

### Diagnostic categories

- discovery
- validation
- source
- hierarchy
- relationship
- profile
- rule
- configuration
- navigation
- security
- scale
- layout
- cancellation
- infrastructure

## Canonical payload shapes

### Bundle catalog

    {
      "project_id": "project-id",
      "default_bundle_id": "knowledge/.okf",
      "bundles": [
        {
          "bundle_id": "knowledge/.okf",
          "relative_path": "knowledge/.okf",
          "status": "valid",
          "selectable": true,
          "concept_count": 42,
          "diagnostics": []
        }
      ],
      "diagnostics": []
    }

Invalid candidates use selectable false and retain their path/status and
diagnostics. Catalog ordering is normalized relative path order.

### Profile catalog

    {
      "profiles": [
        {
          "profile_id": "builtin:neutral",
          "name": "Neutral",
          "origin": "builtin",
          "status": "valid",
          "bases": [],
          "revision": "builtin-revision",
          "immutable": true
        },
        {
          "profile_id": "project:fog-of-war",
          "name": "Fog of war",
          "origin": "project_local",
          "status": "valid",
          "bases": ["builtin:neutral"],
          "revision": "profile-revision",
          "immutable": false
        }
      ],
      "bindings": [
        {
          "bundle_id": "knowledge/.okf",
          "profile_id": "project:fog-of-war"
        }
      ],
      "registry_revision": "registry-revision",
      "diagnostics": []
    }

### Projection snapshot

    {
      "status": "ready",
      "source": {
        "bundle_id": "knowledge/.okf",
        "source_revision": "source-revision"
      },
      "profile": {
        "profile_id": "project:fog-of-war",
        "profile_revision": "profile-revision"
      },
      "navigation": {
        "can_go_back": false,
        "focus_root": "",
        "depth": 2,
        "full": false,
        "breadcrumbs": []
      },
      "nodes": [
        {
          "concept_id": "capabilities/example.md",
          "source_path": "capabilities/example.md",
          "title": "Example",
          "label": "Example",
          "role": "implementation",
          "declared_state": "specified",
          "effective_state": "specified",
          "shape": "rounded_rectangle",
          "presentation_token": "state.specified",
          "annotations": {},
          "children_visible": true
        }
      ],
      "relationships": [
        {
          "relationship_id": "containment:parent:child",
          "kind": "containment",
          "from": "capabilities/index.md",
          "to": "capabilities/example.md",
          "provenance": {
            "source": "filesystem_fallback",
            "explanation": "No selected explicit parent claim was present"
          }
        }
      ],
      "counts": {
        "visible_nodes": 1,
        "hidden_nodes": 0,
        "visible_relationships": 1,
        "hidden_relationships": 0
      },
      "legend": [],
      "diagnostics": []
    }

Scene nodes and relationships contain renderer-neutral outcomes only. ELK
coordinates, SVG markup, and browser-specific values are not contract fields.
Relationship IDs are stable within a projection revision.

### Concept detail

    {
      "concept_id": "capabilities/example.md",
      "bundle_id": "knowledge/.okf",
      "source_revision": "source-revision",
      "overview": {
        "title": "Example",
        "description": "A source-backed concept",
        "type": "capability",
        "source_path": "capabilities/example.md"
      },
      "mapped_metadata": {},
      "declared_state": "specified",
      "effective_state": "specified",
      "rendered_markdown": {
        "format": "sanitized_commonmark",
        "content": "Safe rendered content",
        "links": []
      },
      "raw_markdown": "Optional source text",
      "frontmatter": {},
      "containment": {},
      "semantic_links": [],
      "provenance": [],
      "diagnostics": []
    }

Raw Markdown is secondary. Unsafe HTML, scripts, schemes, and path escapes
must not appear as executable content.

### Diagnostic

    {
      "code": "okf_hierarchy_conflict",
      "severity": "warning",
      "category": "hierarchy",
      "message": "Two explicit parents were declared",
      "bundle_id": "knowledge/.okf",
      "concept_id": "capabilities/example.md",
      "operation_id": "operation-id",
      "details": {
        "affected_claims": []
      },
      "recovery": "Choose one parent mapping or use the diagnostic/index view"
    }

## Canonical error codes

### Input and validation

- okf_invalid_request
- okf_bundle_invalid
- okf_profile_invalid
- okf_rule_invalid
- okf_unsupported_schema
- okf_depth_invalid
- okf_limit_invalid
- okf_binding_invalid

### Detail presentation selection

Profiles may select a registered detail presentation through
`details.renderer = {"id":"okf.detail.commonmark","version":"1","parameters":{}}`.
Omission inherits the selected renderer from base profiles; without an inherited
selection, the existing formatted CommonMark display remains the default. An
explicit selection replaces the inherited renderer and its parameters together.
The extension catalog exposes `detail_renderer` entries with ID, version,
description, capabilities and parameter schema. Renderer implementations are
registered by the host, never loaded as executable code from bundle content.
All custom display Markdown passes through the existing sanitizer, link-boundary
checks and display-size limit. Raw Markdown and source metadata remain original.
An unavailable renderer or rejected parameters invalidate a proposed profile
with `okf_detail_renderer_invalid`. A runtime renderer failure displays original
Markdown with `okf_detail_renderer_failed`; cancellation still aborts the request.

### Missing or unavailable resources

- okf_project_not_found
- okf_bundle_not_found
- okf_bundle_unavailable
- okf_concept_not_found
- okf_profile_not_found
- okf_rule_not_found

### Policy and security

- okf_bundle_boundary_violation
- okf_unsafe_markdown
- okf_unsafe_link
- okf_builtin_immutable
- okf_delete_binding_required
- okf_profile_cycle

### Consistency and persistence

- okf_revision_conflict
- okf_idempotency_conflict
- okf_profile_id_conflict
- okf_configuration_invalid
- okf_configuration_write_failed

### Processing

- okf_hierarchy_conflict
- okf_relationship_unresolved
- okf_projection_failed
- okf_layout_failed
- okf_operation_cancelled
- okf_operation_superseded
- okf_operation_timeout

Truncation is normally a successful projection status with diagnostics, not a
failed request. A client may request strict behavior through an adapter
option, but the underlying projection status remains truncated.

## Canonical HTTP mapping

The local Arch View host uses the versioned base path /v1/okf. Exact request
body and response envelope fields above are normative; route naming is the
canonical local mapping.

The ValidateProfile response data contains `profile` (the validated editable
declaration), `effective_profile` (the backend-composed draft including inherited
settings), `valid`, and `diagnostics`. Preview does not publish the draft to a
session or persist it. Editors display effective values but preserve omitted
declaration fields unless the user changes them.

GetExtensionCatalog response data contains an `extensions` array. A descriptor
contains `id`, `version`, `kind`, `description`, and `capabilities`. Rule
descriptors may also contain `parameter_schema`, an object schema describing
their parameters using `type`, `properties`, `required`, and applicable value
constraints. The schema belongs to the descriptor's strategy version; clients
must not reuse it for a different version. Built-in rules publish these schemas.
An injected legacy strategy without schema metadata omits the field rather
than implying that every parameter is valid. Profile validation remains the
authoritative check before applying or persisting a profile. This describes the
rule descriptor wire format, not completion of the other extension families
required by GetExtensionCatalog.

The profile catalog's `registry_revision` identifies the registered rule
ID/version set and validated shape definitions using a deterministic SHA-256
revision. It excludes project profile edits, tracked by `configuration_revision`,
and does not invoke optional extension metadata callbacks. The full extension
catalog revision below also includes human-readable metadata and schemas.

The extension-catalog envelope's `meta.revision` is a SHA-256 content revision of the returned
ordered descriptor array, including parameter schemas. Unchanged descriptors
retain their revision; provider metadata and schema changes change it.

Registered shape descriptors include `definition_schema`, a JSON Schema 2020-12
object with versioned `$id` `urn:arch-view:okf:shape-definition:1`. It describes
the shared declarative definition format, including geometry-specific fields.
Cross-coordinate containment and convexity require registration validation;
schema validation alone does not establish usable geometry. Content width and
height must be at least `2^-52`, the floating-point precision step at unit scale,
so subnormal fractions cannot overflow node sizing. This field describes
definitions, not executable provider parameters, and participates in the catalog
content revision.

Registered shapes appear with `kind: shape` and `shape_definition`, also carried
by projected nodes. Definitions contain `id`, `version`, `description`,
`geometry` (rectangle, ellipse, or convex polygon), optional normalized `points`
and `corner_radius`, and a normalized `content` box. Geometry contains no SVG or
executable code. Polygon outlines must be nondegenerate, strictly convex, contain
the unit-box center, and enclose the content box; either winding is accepted.
Shape references use `namespaced.id@version`, with version 1 when omitted.
Built-in short names resolve under `okf.shape`. An unavailable shape reference
produces `okf_shape_unsupported` and the default rounded rectangle in projection.
Explicit style-token shape references are also checked during effective-profile
validation. Save and Save As reject unavailable references with the existing
`okf_profile_invalid` status-400 envelope and an `okf_shape_unsupported`
diagnostic, without changing configuration. Rule-generated shapes are checked
when evaluated during projection, since their output can depend on source facts.

Metadata rule `field` values support dotted nested frontmatter paths. An exact
literal key wins when both a dotted key and a nested path exist. State-field
selection uses the same lookup. Roll-up classification is supplied by profile
rules, not inferred from source type or role words. Fog declares its role/type
and `state_policy.mode` conventions through metadata-equality rules at priority
-1; Neutral declares none. Custom profiles that replace Fog's rules must include
their desired roll-up mapping explicitly. Setting `state.roll_up` alone enables
state reduction but does not classify any concept as a roll-up.

Injected relationship adapters are listed with `kind: relationship_adapter`,
their registered ID/version, and description or parameter schema when supplied
by the provider. Rule catalog metadata failures return
`okf_extension_catalog_failed` with status 500; relationship-adapter metadata
failures return `okf_relationship_adapter_failed` with status 500. Neither failure
returns a partial success catalog.

The local filesystem scanner applies per-bundle source-input limits separately
from projection limits: by default, 10,000 Markdown files and 64 MiB of source
bytes, including index/log files. Hosts can configure these through the scanner's
`IndexLimits`; they are not profile or layout options. A bundle exceeding either
limit remains visible but unselectable, with an `okf_bundle_source_limit` error
diagnostic naming the limit and maximum. Indexing does not publish a partial
source snapshot. Other independent bundles remain available. These input limits
do not specify an exact heap-memory ceiling for parsed metadata.

| Use case | Method and path | Result |
|---|---|---|
| RefreshBundleCatalog | POST /v1/okf/catalog/refresh | Bundle catalog |
| GetBundleCatalog | GET /v1/okf/catalog | Bundle catalog |
| GetBundleIndexSummary | GET /v1/okf/bundles/summary?bundle_id=... | Index summary |
| GetProfileCatalog | GET /v1/okf/profiles | Profiles and bindings |
| GetExtensionCatalog | GET /v1/okf/extensions | Registry metadata |
| ValidateProfile | POST /v1/okf/profiles/validate | Effective validation result |
| SelectBundle | PUT /v1/okf/sessions/{session_id}/bundle | Session/projection |
| SelectProfile | PUT /v1/okf/sessions/{session_id}/profile | Session/projection |
| SetNavigationDepth | PUT /v1/okf/sessions/{session_id}/navigation/depth | Session/projection |
| FocusSubtree | POST /v1/okf/sessions/{session_id}/navigation/focus | Session/projection |
| NavigateBack | POST /v1/okf/sessions/{session_id}/navigation/back | Session/projection |
| NavigateTopLevel | POST /v1/okf/sessions/{session_id}/navigation/top | Session/projection |
| GetNavigationState | GET /v1/okf/sessions/{session_id}/navigation | Navigation state |
| GetCurrentProjection | GET /v1/okf/sessions/{session_id}/projection | Projection snapshot |
| GetConceptDetail | GET /v1/okf/sessions/{session_id}/concept-detail?concept_id=... | Concept detail |
| GetDiagnostics | GET /v1/okf/diagnostics?... | Diagnostics |
| BindProfileToBundle | PUT /v1/okf/bindings?bundle_id=... | Configuration revision |
| SaveProjectProfile | PUT /v1/okf/profiles/{profile_id} | Profile/configuration revision |
| SaveProjectProfileAs | POST /v1/okf/profiles/save-as | New profile/configuration revision |
| RenameProjectProfile | POST /v1/okf/profiles/{profile_id}/rename | Profile/configuration revision |
| DeleteProjectProfile | DELETE /v1/okf/profiles/{profile_id} | Configuration revision |

NavigateTopLevel clears subtree focus while preserving the selected bundle,
profile, depth and full-mode setting. When leaving a focused view, it records
that view in navigation history so NavigateBack restores it. At top level it
is a no-op and does not append duplicate history. Neither operation writes
profile or architecture layout settings. Browser session layout drafts remain
in effect across both operations. NavigateBack with empty history returns HTTP
409 with code `okf_navigation_history_empty` and does not publish a projection.

GetDiagnostics accepts optional `project_id`, `bundle_id`, `profile_id`,
`concept_id`, `relationship_id`, `operation_id`, `severity`, and `category`
query parameters. Nonempty filters combine by intersection using exact matches.
The bundle filter retains unscoped project warnings, preserving existing behavior;
other entity filters require the matching entity ID. A different project ID
returns an empty report. Optional `relationship_id` on a diagnostic identifies
its relationship when available. Filtering preserves recovery guidance and
structured details and does not refresh source or navigate a session.
The combined report is limited to 200 diagnostics after filtering. When more
match, the first 199 are followed by `okf_diagnostics_truncated`. This is a
response-count limit, not a memory or execution-time limit on in-process code.

Host-registered `diagnostic_provider` extensions may add explanations from
published bundle snapshots. Their catalog metadata declares ID, version,
description, capabilities and output schema. Providers receive independent
snapshot copies and execute in stable ID/version order. The application assigns
the inspected bundle ID to their results and applies the normal query filters.
Provider errors, panics or malformed output produce
`okf_diagnostic_provider_failed` warnings without removing built-in diagnostics.
Cancellation aborts the query. Provider registration is host configuration, not
executable bundle content or a public HTTP mutation operation.

Host-registered `presentation_property` providers use the existing profile
`rules` invocation format: provider ID in `rule_id`, version, parameters,
enabled flag and priority. They contribute label, token, shape or annotation
properties through the same composition/conflict policy as rules. They cannot
alter source, visibility or effective state. Their catalog entries expose
capabilities and parameter schemas. Existing rule failure diagnostics apply;
rendering still uses the normal safe scene adapters and token/shape handling.

Session mutation endpoints return the projection when evaluation completes
within the adapter's request policy; otherwise they return operation metadata
that the client can use with the projection query. A session is ephemeral and
does not change persisted bindings unless a binding command is invoked.

### HTTP method and request guidance

- GET queries are read-only.
- PUT is used for idempotent selection, navigation-setting, binding, and
  profile replacement.
- POST is used for refresh, focus, validation, Save As, and rename intents.
- DELETE is used only for project-local profile deletion with explicit
  replacement/fallback data.
- Configuration writes require an expected revision and Idempotency-Key where
  the operation can create or remove a profile.

## HTTP status mapping

| Outcome | HTTP status |
|---|---:|
| Success, including displayable truncation | 200 |
| Successful creation by Save As | 201 |
| Invalid request/schema/parameters | 400 |
| Unsafe or disallowed profile/source operation | 403 |
| Bundle, concept, or profile not found | 404 |
| Revision/idempotency/profile identity conflict | 409 |
| Valid request rejected by domain policy | 422 |
| Operation timeout | 504 |
| Host/filesystem/processing infrastructure failure | 500 or 503 |

Cancelled or superseded session requests use 409 when the transport must
return an HTTP failure; the response code identifies the canonical operation
status and does not imply source or configuration corruption.

## Canonical CLI mapping

The following mapping is reserved for a future local CLI adapter. It must
remain semantically identical to the HTTP surface if delivered.

    arch-view okf catalog [--project-root PATH] [--refresh] [--json]
    arch-view okf bundle-summary --bundle-id ID [--json]
    arch-view okf profiles list [--json]
    arch-view okf profiles validate --file PROFILE.json [--json]
    arch-view okf profiles save --profile-id ID --file PROFILE.json
      --expected-revision REV --operation-id ID [--json]
    arch-view okf profiles save-as --source-profile ID --new-profile-id ID
      --file PROFILE.json --operation-id ID [--json]
    arch-view okf profiles rename --profile-id ID --new-profile-id ID
      --expected-revision REV --operation-id ID [--json]
    arch-view okf profiles delete --profile-id ID
      (--replacement-profile-id ID | --neutral-fallback)
      --expected-revision REV --operation-id ID [--json]
    arch-view okf view --session-id ID --bundle-id ID
      [--profile-id ID] [--depth N | --full] [--json]
    arch-view okf focus --session-id ID --concept-id ID [--json]
    arch-view okf back --session-id ID [--json]
    arch-view okf detail --session-id ID --concept-id ID [--json]
    arch-view okf diagnostics [--bundle-id ID] [--profile-id ID] [--json]

CLI machine-readable output uses the same data, diagnostics, and error shapes
as HTTP. Human-readable output may format labels and Markdown differently but
must not omit error codes, hidden counts, or recovery guidance.

## CLI exit-code guidance

- 0: successful query or command, including a valid truncated projection;
- 2: invalid request, schema, or profile input;
- 3: missing/unavailable project, bundle, concept, profile, or rule;
- 4: revision, profile identity, or idempotency conflict;
- 5: policy or security rejection;
- 6: projection, layout, or other processing failure;
- 7: configuration persistence failure;
- 8: cancellation or supersession.

## Parity rules

All implementations and adapters must preserve:

1. one-bundle selection and no implicit merging;
2. bundle/concept/profile identity and revisions;
3. containment versus semantic_link relationship kinds;
4. depth, full mode, node/relationship budgets, and visible hidden counts;
5. profile composition, rule priority, and equal-priority conflict behavior;
6. built-in immutability, Save, Save As, rename, and delete semantics;
7. source read-only behavior and Markdown/path safety;
8. diagnostic codes, categories, severity, affected identity, and recovery;
9. idempotency, revision conflict, cancellation, and supersession behavior;
10. renderer-neutral scene meaning.

Adapters may vary layout coordinates, SVG/CSS, labels' typography, transport
syntax, and human-readable formatting.

## Minimum first-slice surface

The first implementation must expose the local host behavior needed for:

- catalog discovery and validation;
- one-bundle selection;
- profile catalog and neutral/fog-of-war profiles;
- projection with containment/semantic-link layers;
- depth, full mode, subtree focus, and Back;
- concept detail and diagnostics;
- project-local profile Save and Save As;
- existing ELK-backed viewer integration.

The future CLI mapping, export-specific OKF projections, remote access, and
collaboration surface are not first-slice delivery requirements.

The user-approved current-canvas Download SVG action reuses the existing
browser exporter. It serializes the rendered scene locally and adds no OKF
HTTP endpoint, headless export format, or CLI command.

## Suggested contract tests

Contract tests should verify:

- identifiers remain stable for equivalent project paths and concept files;
- catalog ordering and invalid-candidate diagnostics are deterministic;
- selecting one bundle never returns concepts from another;
- profile composition and rule conflict outcomes do not depend on registration
  order;
- projection depth and limits produce exact visible/hidden counts;
- containment and semantic links remain separate;
- local links cannot escape the bundle and unsafe Markdown cannot execute;
- Save and Save As produce their distinct profile/configuration results;
- built-in Save is rejected with the immutable error code;
- rename/delete preserve or reject bindings atomically;
- stale revisions and repeated operation IDs have the defined outcomes;
- cancelled/superseded requests cannot replace the active session result;
- HTTP and CLI JSON data/error shapes are equivalent when the CLI is delivered.
