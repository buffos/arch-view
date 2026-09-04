# Advanced ELK Stage 1 evidence

Date: 2026-09-04
Issue: 080, Shared ELK settings and feature registry.
Status: automated verification complete; awaiting human visual review.
Capability remains specified. Issues 081–083 remain blocked.

## Scope and architecture

One embedded JSON feature catalog supplies Go validation and live/embedded
browser metadata. Shared JavaScript handlers use fixed phases and deterministic
dependency ordering. Unimplemented handlers are not offered as enabled features.
Both scene adapters call the same feature-aware ELK runtime. Settings share
controls, validation messages, filtering, structured values and CSS. Persistence
continues through existing architecture/OKF boundaries and atomic project writes.

No new analyzer facts, source identities, semantic relationships or endpoints.
Geometry v1 feature output is owned by issues 081–083, not claimed by Stage 1.

## Executed scenario trace

| Rule / scenario | Issue criterion | Executed evidence |
|---|---|---|
| AER-R-001 / SC-AER-011 | Shared registry and deterministic dispatch | layout_features_test.js checks ordering, permuted registration, missing/duplicate handlers, unknown dependencies, cycles, overlapping ownership, feature-owned fallback. features_test.go checks metadata availability and ownership. |
| AER-R-002 / SC-AER-001 | Persistence and omitted/empty distinction | layout/features_test.go checks Apply vs Save/reload, safe unavailable preferences without file mutation and independent copies. okf/profile/layout_features_test.go checks inheritance and explicit empty through JSON/composition/clone. okf/configuration/layout_features_test.go checks v2 upgrade, independent architecture settings, idempotency and reload. |
| AER-R-002 / SC-AER-008 | Validation and compatibility | layout_features_http_test.go checks 422 for unknown/unavailable features and v2 request acceptance. layout_form_test.js checks retained incompatible values and disabled Apply. Existing OKF revision/conflict/lifecycle tests remain green. |
| AER-R-003 / SC-AER-008 | Usable-first settings and verified values | layout_form_test.js exercises shared filter/reasons, four numeric padding inputs and encoding. layout/runtime_test.go passes the real published catalog into layout_settings_fixture.cjs and executes the bundled runtime. |
| AER-R-004 / SC-AER-001, SC-AER-010 foundation | Preserve scene behavior | Existing okf_graph, detail, viewport, route, spline, export, embedded and navigation tests pass. No advanced geometry enabled. |
| AER-R-005 / SC-AER-001, SC-AER-011 | Shared defaults and embedded fallback | layout_value clone/request tests, shared runtime tests, export/layout_features_test.go retain saved unavailable preferences and the shared diagnostic path in embedded HTML. |

Test paths above are under internal/viewer/web/app unless qualified. Backend
layout tests are under internal/viewer/layout; OKF tests are under internal/okf.

## Pinned-runtime findings

- Mr. Tree MIDDLE_TO_MIDDLE and AVOID_OVERLAP produce different route sections.
- MODEL_ORDER versus DESCENDANTS/FAN changes sibling positions.
- DFS is the supported search order. BFS works for a strict tree but overflows
  on a graph with shared descendants in the shipped runtime. It is not offered.
- NONE routing fails with an invalid vector-chain error. It is not offered.
- CONSTRAINT weighting needs per-node constraints not implemented in this
  tranche. It is not offered; the dialog explains why.
- Layered component spacing changes disconnected-component separation.
- Four-sided graph padding translates content and changes bounds as expected
  in both Mr. Tree and Layered. It is not broadcast onto leaf nodes.
- No ELK upgrade was performed.

## Actual browser checks

Used the opt-in TestOKFLiveReviewFixture, with disposable project configuration.

- Opened both modes and confirmed the same usable-first settings controls.
- OKF initially used Mr. Tree; architecture initially used Layered.
- Applied OKF padding, focused Source root, used Back and reopened settings:
  the session padding was retained. Architecture padding remained unset.
- Selected architecture SPLINES, changed to Mr. Tree: the value remained
  visible, incompatibility was explained, and Apply/Save/Save As were disabled.
- Reset the conflicting option, saved MIDDLE_TO_MIDDLE, reloaded: Mr. Tree and
  the routing preference were retained.
- Checked settings at 420 × 850 and the normal desktop width. Four padding
  inputs and Apply remained reachable. Desktop uses the existing two-column
  settings grid; no copied responsive rules.
- Entered OKF Full canvas and used Fit: 212%, not capped at 100%.
  Wheel scrolling changed zoom to 204%.
- Focus/Back, selection, pan/manual positioning and route behavior also retain
  their existing automated browser coverage. Human visual inspection is still
  required; these checks do not waive it.

Actual advanced labels/ports/container/curve Download SVG inspection belongs
to issues 081–083, once those features exist. Stage 1 keeps the current exporter.

## Verification commands

- go test ./... -count=1
- go test -race ./...
- go vet ./...
- go build ./...
- node --test over all browser *test.js files: 52 passing tests.
- node --check over browser JavaScript and the pinned runtime fixture.
- Strict OKF validator: 20 concepts, 3 indexes, 1 log, no errors or warnings.
- git diff --check.
- New/modified implementation line audit: largest is profile/registry.go at
  581 lines, below 600. No new stylesheet; existing shared layout classes reused.

## Human review still required

Inspect both settings dialogs, filtering, Mr. Tree controls, padding,
incompatible-algorithm recovery, Save/reload and OKF navigation. Check normal
and Full canvas interaction. Stage 2 must not start until explicit approval.
