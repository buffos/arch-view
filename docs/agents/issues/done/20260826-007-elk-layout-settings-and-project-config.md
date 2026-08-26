# 007 — ELK layout settings and persistent project configuration

Execution type: AFK
Review gate: visual-review
Status: done

## Parent PRD

docs/architecture/explore-architecture/prd.md

## What to build

Create a vertical viewer slice that lets a developer inspect and change the
pinned ELK layout behavior, apply it to the current scene, and persist project
presentation preferences safely.

- Add an accessible layout settings surface reachable from the local viewer.
- Expose a searchable, grouped catalog of the algorithms and options available
  from the pinned ELK/elkjs adapter. Each entry should show its type, default,
  current value, description, applicability, and renderer support.
- Validate typed values, enum values, algorithm/option compatibility, and
  unsupported options before applying or saving them. Do not treat arbitrary
  ELK keys as valid merely because they are strings.
- Apply a valid profile explicitly to the current hierarchy scene. Re-run the
  selected layout adapter, consume its node positions and edge routes, and
  clear manual positions for that hierarchy path as `Reset layout` does. Keep
  canonical model facts and relationship direction unchanged.
- Add `arch-view.config/v1` project configuration using the canonical file
  name `.archview.json`. The file stores layout presentation settings only;
  analyzer options, canonical model data, viewport state, and manual node
  positions stay outside it.
- Resolve configuration from the selected target directory upward through its
  ancestors toward the filesystem root. The nearest file wins as one complete
  profile; v1 does not merge multiple files. If no file exists, built-in
  defaults apply. An invalid nearest file is reported and is not silently
  bypassed in favor of a farther file.
- Make saving explicit with distinct `Save` and `Save As` actions. If discovery
  loaded `.archview.json` from folder X, ordinary `Save` must atomically
  overwrite that exact active file and must never create or copy a new file in
  the project root. If no file is active, ordinary `Save` is unavailable or
  returns `save_as_required`. `Save As` is the only operation that accepts a
  user-selected custom destination folder; it writes the fixed `.archview.json`
  filename there atomically after explicit confirmation and makes that file
  active for the current session. Neither action edits source files. A
  model-only session may apply settings for the current session but cannot
  persist project configuration.
- Show the effective configuration origin (`default`, `project`, `ancestor`,
  or `session`) and actionable diagnostics for invalid or unavailable settings.

## Acceptance criteria

- [x] The viewer exposes a keyboard-accessible settings surface with grouped
  search/filtering for the pinned ELK algorithms and options.
- [x] The catalog reports every option exposed by the pinned adapter and
  clearly distinguishes editable, unsupported, and non-applicable entries;
  defaults, current values, types, allowed values, and descriptions are
  visible.
- [x] Valid settings can be applied explicitly without mutating the
  canonical model, semantic relationships, cycle facts, reference policy, or
  evidence. The selected layout returns positions and edge routes to the
  existing renderer.
- [x] Applying a new profile clears manual positions for the active hierarchy
  path and the existing `Reset layout` behavior can return to built-in
  defaults. Resetting the session does not silently delete a project file.
- [x] `.archview.json` is versioned as `arch-view.config/v1` and contains only
  layout algorithm/options in the v1 schema.
- [x] Project-backed sessions discover `.archview.json` from the selected
  target directory through its ancestors toward the filesystem root; the
  nearest file wins without merging, and no file uses built-in defaults.
- [x] The settings surface identifies the effective file/origin and active
  path. When a valid configuration was discovered in an ancestor folder, the
  ordinary `Save` action overwrites that exact file and does not create a new
  project-root copy; a new `open --project` session reloads it through the same
  nearest-ancestor discovery.
- [x] When no configuration file is active, ordinary `Save` is unavailable or
  returns `save_as_required` and does not create a file.
- [x] `Save As` is the only action that accepts a custom destination folder;
  after explicit confirmation it atomically creates/replaces the fixed
  `.archview.json` filename there, reports the new active path, and rejects
  invalid profiles before writing. The ordinary `Save` request has no
  destination field and neither operation edits source files.
- [x] A custom `Save As` file is marked with `custom` origin when it is outside
  the selected target's ancestor chain. It remains active for the current
  session; automatic discovery in a later session still follows the target's
  ancestor chain, so explicit configuration selection is a later capability.
- [x] Model-only sessions can apply session settings but expose persistence as
  unavailable with a clear explanation.
- [x] Malformed, unsupported, or incompatible nearest configuration produces
  an actionable diagnostic, does not silently fall through to another file,
  and uses safe defaults for the active session.
- [x] Existing fallback behavior remains available if the ELK worker or the
  selected layout request fails; the failure is visible to the user.
- [x] Backend, frontend, end-to-end, and repository-integrity verification
  covers the catalog, validation, discovery precedence, safe writes,
  model-only behavior, apply/reset behavior, and no semantic mutation.
- [x] A visual review confirms the settings surface, origin/error states,
  apply/reset workflow, and layout changes in windowed and full-canvas views.

## Artifact sync required

- Application PRD: required: `docs/prd.md` — the settings journey and
  project-scoped presentation preferences are part of the product target.
- Application architecture summary: required:
  `docs/architecture/application-architecture-summary.md` — add the local
  configuration resolver, typed ELK settings boundary, and narrowly scoped
  atomic write capability; keep analyzer and canonical-model boundaries
  separate.
- Owning capability node/artifacts: required:
  `.okf/capabilities/explore-architecture.md`;
  `docs/architecture/explore-architecture/discovery-notes.md`;
  `docs/architecture/explore-architecture/requirements-gap-analysis.md`;
  `docs/architecture/explore-architecture/orchestration-status.md`;
  `docs/architecture/explore-architecture/prd.md`;
  `docs/architecture/explore-architecture/domain-glossary.md`;
  `docs/architecture/explore-architecture/canonical-domain-model.md`;
  `docs/architecture/explore-architecture/canonical-use-cases.md`;
  `docs/architecture/explore-architecture/canonical-api-cli-contract.md`;
  `docs/architecture/explore-architecture/acceptance-scenarios.md`;
  `docs/architecture/explore-architecture/readiness-review.md`.
- Contributor guidance: required: `README.md` — document the implemented
  configuration name, precedence, scope, and model-only limitation alongside
  the existing ELK/layout explanation.
- Delivery truth: required: this issue, `docs/agents/issues/issues.md`,
  `.okf/index.md`, `.okf/log.md`, and the first implementation slice at
  `docs/architecture/analyze-source/go-analysis/implementation-slice.md`.
- No impact: analyzer/plugin contracts, language-specific analyzer artifacts,
  canonical model semantics, and export behavior remain unchanged because the
  v1 file stores presentation settings only. The export capability is not an
  automatic consumer of `.archview.json` in this issue.
- Reference boundary: the upstream [reference repository](https://github.com/unclebob/arch-view) remains read-only and untouched.

## Blocked by

—

## User stories addressed

- US-EX-003

## Contract and scenario trace

- Contract: `docs/architecture/explore-architecture/canonical-api-cli-contract.md`
- Scenarios: SC-EX-012, SC-EX-013, SC-EX-014, SC-EX-015, SC-EX-016

## Scenario traceability and verification plan

| Source rule / use case | Scenario | Issue criterion | Planned verification | State |
|---|---|---|---|---|
| Pinned layout adapter is inspectable and typed | SC-EX-012 | Catalog shows algorithms/options, metadata, applicability, and support | Catalog contract/unit tests plus browser settings review | automated-passed; visual-approved |
| Applying layout is explicit and presentation-only | SC-EX-013 | Valid profile recalculates positions/routes and preserves model semantics | Viewer layout integration tests and scene/model immutability assertions | automated-passed; visual-approved |
| Nearest-ancestor configuration discovery | SC-EX-014 | Target-to-root search selects nearest complete file or defaults | Resolver fixtures for target, parent, no-file, and invalid-nearest cases | automated-passed |
| Safe project persistence | SC-EX-015 | `Save` overwrites the active discovered file; `Save As` is the only custom-destination write and both are atomic | Config write/replace/destination tests plus local-session smoke test | automated-passed |
| Invalid/model-only behavior | SC-EX-016 | Diagnostics, safe defaults, and disabled persistence are visible | Invalid-schema, unsupported-option, worker-failure, and model-only tests | automated-passed; visual-approved |

## Verification surfaces

- Backend boundary: config schema/validation, ancestor resolution, origin and
  active-path reporting, active-file `Save`, explicit custom-folder `Save As`,
  atomic writes, and model-only restrictions.
- Frontend integration: settings catalog, typed controls, apply/reset states,
  error/origin presentation, and reuse of the current ELK/fallback renderer.
- End-to-end: `open --project` loads the nearest `.archview.json`, ordinary
  `Save` writes back to that same path without creating another file, and
  `Save As` writes only to its explicitly selected destination; a subsequent
  session reproduces a saved profile when it is discoverable from the target.
- Repository/OKF integrity: the upstream [reference repository](https://github.com/unclebob/arch-view) remains unchanged, links resolve, the
  issue registry max ID is 009, and the synchronized planning artifacts agree.

## Review handoff

The user explicitly approved the visual review of the settings surface,
configuration origin/error states, apply/reset workflow, and resulting graph in
normal and full-canvas views. The strict code-review loop found no remaining
actionable P0–P2 findings.

## Implementation result

The Go viewer serves the complete catalog from the pinned ELK bundle, validates
typed and applicable profile options, resolves the nearest `.archview.json`
without merging or fallback, and exposes explicit session `Apply`/`Reset`,
active-file `Save`, and confirmed custom-folder `Save As` operations. The
browser settings dialog presents the catalog, origin, diagnostics, and
persistence state. The final review also rejects malformed requests that omit
the required algorithm, enforces the layout-request size limit, and serializes
configuration writes with session updates. Automated repository, backend
contract, catalog, discovery, persistence, and JavaScript-syntax checks pass.

## Closeout result

Closed on 2026-08-26 after explicit user approval. The issue was moved to the
dated delivery archive; its registry row was removed, issue 008 was unblocked,
and the OKF capability/index/log and implementation-slice references were
synchronized. The upstream reference repository was not modified.
