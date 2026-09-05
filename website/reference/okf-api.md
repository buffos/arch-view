# OKF HTTP API

The local viewer exposes the canonical OKF contract under `/v1/okf`. It is intended for the browser and local integrations, not for editing source documents or providing a remote OKF service.

## Response envelope

Successful responses place the result in `data` and may include `diagnostics` and a revision. Failed responses provide a structured error code, message, details, and diagnostics. Clients should display diagnostics as a report and use the code for recovery behavior.

## Catalog and source

| Method | Endpoint | Purpose |
| --- | --- | --- |
| GET | `/v1/okf/catalog` | List discovered bundles and selectable status. |
| POST | `/v1/okf/catalog/refresh` | Rediscover and validate bundles. |
| GET | `/v1/okf/bundles/summary?bundle_id=...` | Read counts, files, source revision, and bundle diagnostics. |
| GET | `/v1/okf/diagnostics` | Query diagnostics by project, bundle, profile, concept, relationship, operation, severity, or category. |
| GET | `/v1/okf/extensions` | List registered presentation and detail extensions. |

## Profiles and bindings

| Method | Endpoint | Purpose |
| --- | --- | --- |
| GET | `/v1/okf/profiles` | List built-in and project-local profiles and bindings. |
| POST | `/v1/okf/profiles/validate` | Validate and preview a profile without saving it. |
| PUT | `/v1/okf/profiles/{profile_id}` | Save a project-local profile. |
| POST | `/v1/okf/profiles/save-as` | Create a project-local profile copy. |
| POST | `/v1/okf/profiles/{profile_id}/rename` | Rename a project-local profile. |
| DELETE | `/v1/okf/profiles/{profile_id}` | Delete or replace a project-local profile. |
| PUT | `/v1/okf/bindings?bundle_id=...` | Bind a bundle to a profile. |

Built-in profiles cannot be overwritten. Profile writes require valid content and may use `If-Match` or `expected_revision` together with an idempotency key.

## Sessions and navigation

Replace `{session_id}` with the project session identifier, normally `default` in the local viewer.

| Method | Endpoint | Purpose |
| --- | --- | --- |
| GET | `/v1/okf/sessions/{session_id}` | Read session state, navigation, profile, and diagnostics. |
| PUT | `/v1/okf/sessions/{session_id}/bundle` | Select a bundle. |
| PUT | `/v1/okf/sessions/{session_id}/profile` | Select a profile. |
| GET | `/v1/okf/sessions/{session_id}/projection` | Read the current renderer-neutral projection. |
| GET | `/v1/okf/sessions/{session_id}/navigation` | Read navigation state. |
| PUT | `/v1/okf/sessions/{session_id}/navigation/depth` | Change depth or Full mode. |
| POST | `/v1/okf/sessions/{session_id}/navigation/focus` | Focus a concept and its subtree. |
| POST | `/v1/okf/sessions/{session_id}/navigation/back` | Return to the previous focus. |
| POST | `/v1/okf/sessions/{session_id}/navigation/top` | Return to the top level. |
| GET | `/v1/okf/sessions/{session_id}/concept-detail?concept_id=...` | Read one concept's profile-driven details. |

## Safe writes and failures

The server validates bundle boundaries, profile composition, layout settings, revisions, and request sizes. It uses atomic project-document replacement for configuration writes. A stale revision, conflicting idempotency key, invalid profile, or unavailable bundle returns an explicit failure and does not partially update the project.

The API does not expose a source-editing endpoint. Markdown and frontmatter remain the authority for OKF content.
