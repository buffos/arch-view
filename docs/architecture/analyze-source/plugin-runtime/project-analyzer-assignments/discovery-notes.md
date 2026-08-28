# Project analyzer assignments and view selection discovery notes

## Purpose

Give users durable, repository-relative control over analyzer ownership and a
simple way to move between individual analyzer scopes and the combined view.

## Current implementation

- **Observed in code:** The viewer supports nearest-ancestor `.archview.json`
  discovery for presentation layout, and `open` receives one analyzer result.
- **Inferred from docs:** Analyzer options and layout settings are intentionally
  separate today; the current configuration file does not persist folder-to-
  analyzer assignments or expose an analyzer-scope selector.

## User-confirmed target behavior

- A configuration section can map repository-relative folders or project roots
  to analyzer IDs.
- Automatic detection remains available as a fallback.
- A dropdown can switch between `All` and individual analyzer/project scopes
  without re-running already completed jobs.
- Changing one assignment reanalyzes only the affected scope where possible.
- The mapping can distinguish implementations such as built-in versus
  compiled external Python analyzers.

## Specified decisions

- Assignments live in a separate top-level `analysis` section of the existing
  versioned `.archview.json`; `layout` remains presentation-only. Adding the
  section requires a compatible configuration-schema revision and preserves
  v1 layout-only files as configurations with no assignments.
- Keys are normalized repository-relative folders or project roots, and values
  reference stable logical analyzer IDs rather than deployment-specific
  `.external` IDs. The repository root is representable as `.`.
- The nearest discovered configuration file remains the one complete file, as
  in current layout resolution. Within that file, the deepest matching
  assignment wins; explicit CLI analyzer/language selection overrides the
  assignment; automatic detection is the fallback. Invalid nearest config is
  surfaced rather than bypassed, and unavailable analyzer IDs produce scoped
  diagnostics.
- The runtime stores a combined model plus cached per-job results. The
  application dropdown offers `All` and individual project/analyzer scopes;
  changing scope filters/selects existing results and does not re-run analysis.
- Editing an assignment or its options invalidates only affected jobs where
  possible. If the active scope disappears, the view returns to `All`; model
  revisions and stale evidence remain explicit.
- Assignment-scoped non-sensitive analyzer options may be stored beside the
  mapping. Analyzer defaults are overridden by assignment options, and CLI
  options override both; sensitive values remain outside persisted project
  configuration.

## Implementation and verification focus

The exact-spec set defines the v2 JSON schema, assignment validation/error
codes, precedence for combined CLI/config cases, cache invalidation keys, and
the dropdown/API contract. Implementation and verification must now cover
nearest-file errors, unavailable analyzers, affected-scope invalidation, cache
reuse, and combined/per-scope viewer projections.

## Boundary

The runtime owns assignment semantics and analyzer selection; the viewer owns
the dropdown presentation and scope navigation. Layout settings remain
presentation-only and must not alter analyzer facts or canonical model meaning.
