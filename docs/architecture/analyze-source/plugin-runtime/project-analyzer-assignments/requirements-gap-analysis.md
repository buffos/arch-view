# Project analyzer assignments and view selection requirements gap analysis

## Scope examined

This pass covers the bounded [project analyzer assignments and view selection](../../../../.okf/capabilities/analyze-source/plugin-runtime/project-analyzer-assignments.md) child, the current strict layout-only `.archview.json` reader, the single-result viewer server, the existing CLI selection flags, and the multi-analyzer aggregate contract.

## Confirmed strong areas

- **Observed in code:** the local host walks ancestors for the nearest `.archview.json`, treats that file as one complete layout profile, rejects invalid nearest configuration, and keeps layout settings separate from analyzer options. Individual analyzers accept additional exclusion globs, but the host has no persisted cross-analyzer source-scope policy.
- **Observed in code:** CLI language/analyzer selection already exists and takes precedence over automatic selection for one run.
- **User-confirmed target behavior:** a separate analysis configuration maps repository-relative folders to stable analyzer IDs, supports automatic fallback, and exposes cached `All` and individual scopes without re-running analysis.

## Blocking gaps resolved by recommendation

| Gap | Impact | Resolution |
|---|---|---|
| The new configuration shape was unspecified | High: tools could write incompatible mappings or mix layout and analysis semantics | Accept `arch-view.config/v1` layout-only files and add strict `arch-view.config/v2` with a top-level `analysis.assignments[]` section. `layout` remains unchanged and separate. |
| Path and duplicate-assignment rules were missing | High: the same folder could resolve differently across platforms | Use normalized repository-relative POSIX paths, `.` for the repository root, no absolute/parent traversal/globs, and reject duplicate normalized paths. |
| Precedence between CLI, config, and detection was incomplete | High: the same repository could produce different job sets | Apply CLI root selection first, then the deepest matching assignment, then automatic detection; combined mode includes all valid assignments and unassigned automatic roots. |
| Invalid configuration and unavailable analyzers were conflated | Medium: one missing analyzer could erase valid scopes | Syntactic/schema/path errors invalidate the nearest file; unavailable analyzer IDs become assignment-scoped diagnostics and do not bypass the nearest file or erase unrelated jobs. |
| Cache identity and invalidation were unspecified | High: a dropdown could show stale architecture facts | Cache by scope, analyzer package/version, effective options, discovery policy, and authoritative source fingerprint; assignment changes invalidate only impacted scopes. |
| Scope switching had no external contract | Medium: the feature could exist only as a UI mock | Add sorted scope summaries and a projection `scope` selector; the viewer exposes `All` and individual scopes from the same aggregate run. |
| Include/exclude source scope was unspecified | High: users could not restrict mixed-language analysis deterministically, and implementations could disagree about which files a job receives | Add strict v2 `analysis.exclude` global globs and analyzer-scoped `analysis.include[]` rules, anchored to the invocation root; apply them after root discovery, with exclusions winning and the effective source set included in cache identity. |

## Deferrable implementation details

- The assignment editor may be a later UI; this capability requires file/CLI/API consumption and viewer selection, not a new layout editor.
- In-memory session caching is the first implementation. Persistent cache storage, remote synchronization, and assignment inheritance across multiple config files are outside scope.

## Exact assumptions

- Configuration parsing is strict: unknown fields are errors in v2.
- Existing v1 layout-only files remain valid and behave as empty analysis configuration.
- Analyzer options stored in assignments are non-sensitive JSON values validated against the selected analyzer manifest.
- Source-scope `exclude` is a global additive policy; `include` is an
  analyzer-ID keyed allowlist. Empty include rules mean no additional
  allowlist, and duplicate analyzer include rules are invalid.
- Patterns are normalized POSIX globs relative to the canonical invocation
  root, with `*`, `?`, character classes, and recursive `**`; parent traversal,
  absolute paths, negation, and comments are invalid in v2.
- Root discovery and nested ownership use the fixed discovery policy before
  source filters are applied. Configured filters cannot hide project markers or
  re-include fixed, nested, or explicitly excluded paths.

## Readiness conclusion

No High or Medium specification gaps remain. The schema, precedence, source
filter semantics, validation, cache, API, and visible scope behavior are ready
to be captured in the exact artifact set.

## Artifact impact

- **Capability:** updated with exact configuration, source-scope, selection,
  cache, and viewer contracts.
- **Product:** application PRD and viewer journey are affected and must remain synchronized.
- **Architecture:** configuration ownership, analyzer-host precedence, source
  filtering, aggregate cache, and viewer projection boundaries are affected.
- **Delivery:** issues 044–047 are verified and archived implementation records,
  including the approved configured-viewer visual gate. No new capability node
  is created.
