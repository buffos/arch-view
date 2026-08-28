# Project analyzer assignments and view selection canonical use cases

## Application services

### AnalysisConfigurationService

- `LoadAnalysisConfiguration`
- `ValidateAssignments`
- `ResolveAssignmentForProjectRoot`
- `CalculateEffectiveAnalyzerOptions`

### AnalysisCacheService

- `BuildAnalysisCacheKey`
- `ReadCachedScope`
- `InvalidateAffectedScopes`
- `StoreScopeResult`

### AnalysisScopeService

- `ListAnalysisScopes`
- `SelectAnalysisScope`
- `BuildCombinedProjection`

## Canonical commands and queries

### `LoadAnalysisConfiguration` — query

**Input:** selected repository root and ancestor filesystem chain.

**Output:** nearest complete profile, origin/path, v1/v2 schema, assignments,
and diagnostics. It never merges or bypasses a nearer invalid file.

### `ValidateAssignments` — command

**Input:** v2 analysis section, repository root, analyzer registry.

**Responsibilities:** validate shape, path safety, duplicate paths, analyzer
IDs, and non-sensitive option values. Structural errors reject the profile;
unavailable analyzers remain scoped diagnostics.

### `ResolveAssignmentForProjectRoot` — query

**Input:** project root, CLI selection, validated configuration, registry.

**Outcome:** CLI selection, deepest assignment, or automatic detection in that
order. A resolution never silently falls through from an unavailable explicit
assignment.

### `BuildAnalysisCacheKey` — command

**Input:** resolved scope and authoritative input fingerprint.

**Outcome:** stable key. Any change to analyzer package/version, root ownership,
effective options, discovery policy, or included source content creates a new
key.

### `SelectAnalysisScope` — query

**Input:** aggregate run and `all` or one scope ID.

**Outcome:** projection from existing cached results. Unknown scope returns a
stable not-found outcome; no analyzer invocation occurs.

## Failure model

- `AnalysisConfigInvalid`
- `AssignmentPathInvalid`
- `AssignmentDuplicatePath`
- `AssignmentAnalyzerUnavailable`
- `AssignmentOptionInvalid`
- `CLISelectionConflict`
- `AnalysisScopeNotFound`
- `AnalysisScopeStale`
- `AnalysisCacheUnavailable`

## Architecture-neutral mapping

Configuration can be resolved by a library, CLI host, or service. Cache storage
can be memory or a later durable implementation. The stable meaning is
precedence, validation, invalidation, and no-reanalysis scope selection.
