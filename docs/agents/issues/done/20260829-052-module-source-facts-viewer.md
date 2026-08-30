# 052 — Show source facts in module inspection

## Metadata

- Issue: 052
- Type: feature
- Owning capability: `/.okf/capabilities/code-quality-and-intelligence/source-facts-and-symbol-index.md`
- Related capability: `/.okf/capabilities/explore-architecture.md`
- Artifact root: `docs/architecture/code-quality-and-intelligence/source-facts-and-symbol-index/`
- Execution: AFK
- Human review: `visual-review`
- Suggested state: done

## Parent artifacts

- `docs/prd.md`
- `docs/architecture/application-architecture-summary.md`
- `docs/architecture/code-quality-and-intelligence/source-facts-and-symbol-index/prd.md`
- `docs/architecture/code-quality-and-intelligence/source-facts-and-symbol-index/acceptance-scenarios.md`
- `docs/architecture/code-quality-and-intelligence/source-facts-and-symbol-index/canonical-api-cli-contract.md`
- `docs/architecture/code-quality-and-intelligence/source-facts-and-symbol-index/orchestration-status.md`
- `docs/architecture/explore-architecture/`
- `internal/viewer/model_http.go`
- `internal/viewer/web/app/details.js`

## What to build

Revise the module-inspection journey around a human-sized summary and a
separate, same-tab inspection view. The selected node or group card stays
compact and contains only its name, kind/language, hierarchy, active scope,
key counts, non-neutral status badges, and an `Open inspection` action. The
keyboard-accessible scene list is a separate graph card and retains its search
and keyboard behavior.

`Open inspection` uses a history-backed route such as
`/?view=inspection&kind=node&id=...&scope=...&path=...`. Direct links are
supported, invalid links fall back to the graph with an explanatory message,
and Back restores the selected node, active scope, and hierarchy. The
inspection view presents Overview, Structure, Files, Symbols, Dependencies,
Evidence, and Technical details. Overview is the default; bounded source
collections load only when opened, retain their local filters, and expose at
most 25 records per request with explicit `Load more` controls.

Human-facing rendering uses readable locations such as `Lines 13–48` and
distinguishes file provenance from line-level source locations. File-only
provenance is presented as a deduplicated module/file association with an
explicit explanation of what it does and does not establish. Ambiguous `line
unavailable` records are not presented as line-level source evidence. Source
excerpts remain opt-in, read-only, and bounded. IDs, hashes, provider
versions, snapshot data, and copy actions are secondary Technical details
rather than summary-card content.

The viewer requests the graph model with `include_source_index=false` and
uses the existing bounded `/files`, `/symbols`, `/documentation`, and
`/evidence/{entity_id}` boundaries for inspection. Repeated `module_id` and
`file_id` filters resolve membership only through explicit containment or
declaration relations. Embedded exports use the same bounded queries locally
when network endpoints are unavailable.

This issue owns presentation and the viewer adapter only. It does not add
extractors, quality rules, live watching, MCP transport, or source mutation.

## Acceptance criteria

- Selected node and group cards remain compact: no stable IDs, relationship
  IDs, hashes, full file lists, symbols, provenance, or evidence lists appear
  in the default details card.
- The graph exposes a separate keyboard-accessible scene list with its
  existing search and selection behavior, and selected nodes/groups expose an
  `Open inspection` action.
- Inspection routes parse and serialize opaque IDs and repeated hierarchy
  path segments, support direct links, preserve scope/path/selection through
  Back, and fall back to the graph with an explanatory error for invalid or
  no-longer-visible targets.
- The inspection view exposes Overview, Structure, Files, Symbols,
  Dependencies, Evidence, and Technical details. Only the active bounded
  collection is fetched on demand; Files and Symbols are paged in groups of
  25 and retain local filters.
- Module details display deterministic files and symbols with byte/line
  indicators, extraction/documentation/visibility state, explicit
  containment, and provenance in the full inspection view. Technical data
  remains available without making the human view identifier-heavy.
- Combined views clearly retain scope qualification and do not infer
  containment or relationships from matching paths or names.
- Missing, unsupported, unknown, partial, and unavailable coverage states are
  distinct, accessible, and actionable where evidence exists.
- Source context is an explicit bounded action using the existing path-safe,
  read-only source boundary; it is not embedded in the graph model payload or
  the default summary card.
- Existing graph navigation, scope switching, evidence links, export behavior,
  keyboard navigation, and legacy models without a source index remain
  functional.
- Browser-level tests and representative fixtures cover populated, empty,
  partial, unsupported, combined, embedded-export, and legacy inspection
  states, including lazy request boundaries and readable source locations.
- The required `visual-review` gate records the rendered result at the target
  viewport and resolves any material layout, contrast, density, or keyboard
  accessibility findings before closeout.

## Artifact synchronization

- Application PRD: required at batch closeout. Describe the compact summary
  card and the delivered human-oriented inspection journey, without
  expanding the approved product boundary.
- Application architecture summary: required at batch closeout. Describe the
  graph boot/query boundary, bounded inspection read model, and same-tab
  progressive-disclosure projection.
- Source-index query contract: document repeated module/file containment
  filters, explicit relation semantics, bounded pages, and the distinction
  between human inspection and technical details.
- Owning capability artifacts: update orchestration status and advance this
  node from `specified` to `implemented` only after issues 048–052 and their
  artifact obligations are complete.
- Planning graph: update `.okf/index.md`, `.okf/log.md`, and the parent roll-up
  after the full batch. The code-quality roll-up remains `specified` while its
  other structural children remain specified.
- Delivery registry and capability node: retain all five issue links and record
  the final state in the completed closeout.

## Human review

The required `visual-review` gate was approved. The review covered the compact
summary card, separate scene list, inspection navigation, populated/partial/
unsupported/legacy states, bounded source action, responsive layout, keyboard
focus order, and screen-reader labels in the existing local viewer at the
target viewport. Browser automation provided supporting evidence; the user
approval is the final review record.

## Blocked by

None — issue 051 is verified and archived; issue 052's required final visual
inspection is approved.

## Specification anchors

- Source-facts requirements: SFI-FR-007, SFI-FR-013.
- Source-facts acceptance scenarios: SFI-AC-003, SFI-AC-014.
- User-facing journeys: inspect a module and its source evidence; inspect a
  module's source facts.

## User journeys covered

- Journey 2: inspect a module and its source evidence.
- Journey 12: inspect a module's source facts.

## Verification obligations

| Layer | Required evidence |
|---|---|
| Backend | Stable viewer adapter payloads, explicit containment filters, scope qualification, evidence bounds, model opt-out/default behavior, and legacy omission tests. |
| Frontend | Route/history tests, compact-card tests, lazy section/cache tests, bounded pagination, readable locations, distinct coverage states, embedded exports, and keyboard access. |
| E2E | When-supported: analysis to local HTTP to module-inspection rendering with bounded source evidence. |

## Implementation and verification

- Added the compact graph summary card, separate accessible scene-list card,
  same-tab inspection route/history controller, semantic section navigation,
  bounded lazy Files/Symbols queries, readable locations, distinct coverage
  states, secondary Technical details, embedded-export query support, and
  responsive inspection styles in `internal/viewer/web/`.
- Added the source-index viewer adapter's repeated `module_id`/`file_id`
  filters, explicit containment/declaration membership, page coverage
  metadata, default-preserving `include_source_index` model opt-out, and
  bounded read-only source evidence behavior.
- Fixed aggregate viewer bootstrap sequencing so analysis-scope discovery starts
  independently of the canonical model download; `bootstrap_test.js` covers the
  delayed-model regression that previously left the scope selector loading
  indefinitely.
- Added local HTTP and JavaScript coverage for populated, partial,
  unsupported, unknown, missing, combined, embedded, and legacy models. The
  final automated command results are recorded below after the verification
  run.
- Added Symbols filter focus preservation, an empty-result-safe filter
  surface, extractor-reported `language_kind` filtering, and repeated subject
  lookup for documentation status. Documentation is now loaded for the
  current bounded file/symbol page instead of assuming the first documentation
  page contains its records.
- Clarified the Evidence view's source-reference semantics: file-only
  references are shown as deduplicated `Module/file association` provenance
  with an explanation that they provide source context but do not prove a
  line-level fact or separate relationship; located references retain their
  bounded source actions. The Symbols kind selector now reuses the existing
  `select-control` styling.

## Final visual inspection — approved

The required final inspection was completed against the local Go session with
the installed Chrome executable at 1440×1000 and a 390×844 responsive width.
The user approved the rendered viewer on 2026-08-30. The inspection confirmed:

- selecting the `internal` group keeps the right card compact and free of
  visible raw IDs, hashes, file lists, symbols, provenance, and evidence lists;
- the separate accessible scene list remains available, and `Open inspection`
  enters a same-tab route with the selected node, scope, and hierarchy;
- Overview, Structure, Files, Symbols, Dependencies, Evidence, and Technical
  details render without page errors; Files/Symbols request bounded data only;
- graph startup makes no eager source-index request, while inspection fetches
  bounded `/documentation`, `/files`, and `/symbols` data on demand;
- Back restores the graph selection, direct valid inspection URLs render, and
  an invalid target falls back to the graph with an explanatory error;
- the inspection layout remains usable at the responsive width and the
  section navigation can scroll horizontally.

The installed Chrome executable was used because the bundled Playwright
browser binary is not installed. The user also approved the keyboard focus
order, screen-reader labels, and populated/partial/unsupported/legacy visual
variants. No layout, contrast, density, or keyboard accessibility finding
remains open.

### Review corrections — approved

The follow-up visual review identified and corrected three presentation issues:

- Aggregate inspection scope status now uses the aggregate result, so a
  partial combined projection is shown as `Partial` rather than `Unknown`.
- Switching an inspection to a valid scope where the selected node is absent
  now clears the stale selection, keeps the loaded scope hierarchy, and shows
  an informational notice. Malformed or stale direct inspection links retain
  the error treatment.
- Symbols now render the bounded read-only source panel after `View source`.
  The action is offered only when source text is available; embedded exports
  explain that the excerpt is unavailable instead of presenting a dead action.

The installed Chrome smoke check was rerun at 1440×1000 and confirmed the
three corrections without page errors. File line-threshold semantics and the
human count/filter projection are specified under deterministic quality checks;
they remain outside this correction pass. The visual-review gate was
subsequently approved by the user.

The follow-up Symbols/documentation review was then exercised against the
local Go viewer. A no-match Symbols filter kept its value, focus, and kind
selector; the kind selector narrowed requests with `language_kind`; and
`DecodeAt` rendered `docs present` with its readable
`internal/analysis/config/config.go · Lines 149–187` location. The graph
startup still made no source-index request before inspection was opened. No
page errors were reported. The visual-review gate was subsequently approved
by the user.

The subsequent Evidence and control-style review was exercised at 1440×1000.
The Symbol kind selector now has the same computed control styles as the
existing graph selects. File-only source references are grouped by path under
`File provenance`, labeled as module/file association, and explain that they
provide context rather than line-level proof; the internal group rendered 25
bounded rows rather than the full 385-reference list. No page errors were
reported. The visual-review gate was subsequently approved by the user.

## Closeout

The user approved issue 052 on 2026-08-30. Normal closeout is complete: its
delivery file is archived, the active registry entry is removed, and the
source-facts capability and application artifacts are synchronized. The
source-facts capability advances to `implemented`; the code-quality and
code-intelligence parent remains `specified` because its deterministic-quality
and live-analysis/MCP children remain specified.

## Handoff

The approved batch is verified and synchronized. The source-facts capability
is implemented through issues 048–052. Its parent code-quality roll-up remains
`specified` until the deterministic-quality and live-analysis/MCP children are
also implemented.
